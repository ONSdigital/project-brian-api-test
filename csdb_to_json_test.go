package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	. "github.com/logrusorgru/aurora"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	diff "github.com/yudai/gojsondiff"
	"github.com/yudai/gojsondiff/formatter"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"testing"
	"time"
)

const (
	descErrFmt = "timeseries[%d].description did not match the expected value"
	typeErrFmt = "timmeseries[%d].type did not match the expected value"

	monthsLenErrFmt   = "timmeseries[%d].months length did not match the expected length"
	quartersLenErrFmt = "incorrect length timmeseries[%d].quarters"
)

var brianHost = "http://localhost:8083"

type TimeSeriesValue struct {
	Date          string `json:"date"`
	Value         string `json:"value"`
	Year          string `json:"year"`
	Month         string `json:"month"`
	Quarter       string `json:"quarter"`
	SourceDataset string `json:"sourceDataset"`
}

type Description struct {
	Title      string `json:"title"`
	CDID       string `json:"cdid"`
	Unit       string `json:"unit"`
	PreUnit    string `json:"preUnit"`
	Source     string `json:"source"`
	Date       string `json:"date"`
	Number     string `json:"number"`
	SampleSize int    `json:"sampleSize"`
}

type TimeSeries struct {
	Years          []TimeSeriesValue `json:"years"`
	Quarters       []TimeSeriesValue `json:"quarters"`
	Months         []TimeSeriesValue `json:"months"`
	SourceDatasets []string          `json:"sourceDatasets"`
	Section        interface{}       `json:"section"`
	Type           string            `json:"type"`
	Description    Description       `json:"description"`
}

func TestConvert_CSDBToJSON(t *testing.T) {
	host := os.Getenv("BRIAN_HOST")
	if len(host) > 0 {
		brianHost = host
	}

	info(t, fmt.Sprintf("\nTest config:\n\t%q:%q\n", "BRIAN_HOST", brianHost))

	if !exists("resources/outputs") {
		t.Errorf(Err(fmt.Sprintf("dir %q does not exist", "resources/outputs")))
		t.Fatalf(Err(fmt.Sprintf("make sure you have unzipped %q before running the tests", "resources/outputs.zip")))
	}

	csdbFilenames := []string{
		"ott",
		"bb",
		"berd",
		"ukea",
		"ragv",
		"sppi",
	}

	for _, filename := range csdbFilenames {
		t.Run(fmt.Sprintf("%s.csdb", filename), func(t *testing.T) {
			testCSDBJSONGeneration(t, filename)
		})
	}
}

func testCSDBJSONGeneration(t *testing.T, filename string) {
	Scenario(t, fmt.Sprintf("The correct JSON is generated for a given %s.csdb file", filename))

	Given(t, fmt.Sprintf("a valid %s.csdb file", filename))
	body, contentType, err := getCSDBRequestBody(filename)
	require.Nil(t, err, Red("error creating POST request").String())

	When(t, "a POST request is sent to /Services/ConvertCSDB")
	response, err := postCSDBFile(body, contentType)
	require.Nil(t, err, Err("error sending POST request"))

	defer func() {
		if err := response.Body.Close(); err != nil {
			panic(err)
		}
	}()

	Then(t, "a 200 response status is returned")
	require.Equal(t, response.StatusCode, 200, Err("incorrect http response status code for POST CSDB request"))

	actualTimeSeries, err := readCSDBResponse(response)
	require.Nil(t, err, Err("error reading csdb response json"))

	expectedTimeSeries, err := getExpectedResults(filename)
	require.Nil(t, err, Err("error reading expected csdb json file"))

	And(t, "the expected number of timeSeries results are returned")
	require.Equal(t, len(expectedTimeSeries), len(actualTimeSeries), Err("timeseries results length does not match expected"))

	var actual TimeSeries
	var expected TimeSeries

	And(t, "each time series value is as expected")

	for index := 0; index < len(actualTimeSeries); index++ {
		actual = actualTimeSeries[index]
		expected = expectedTimeSeries[index]

		require.Equal(t, actual.Description, expected.Description, descErrFmt, index)
		require.Equal(t, actual.Type, expected.Type, typeErrFmt, index)
		require.Len(t, actual.Years, len(expected.Years), "timeseries[%d].years length does not match expected", index)

		for yearIndex := 0; yearIndex < len(actual.Years); yearIndex++ {
			compareTimeSeriesValue(t, actual.Years[yearIndex], expected.Years[yearIndex], "actual did not match expected", "years", index, yearIndex)
		}

		require.Len(t, actual.Months, len(expected.Months), monthsLenErrFmt, index)
		for monthIndex := 0; monthIndex < len(actual.Months); monthIndex++ {
			compareTimeSeriesValue(t, actual.Months[monthIndex], expected.Months[monthIndex], "actual did not match expected", "months", index, monthIndex)
		}

		require.Len(t, actual.Quarters, len(expected.Quarters), quartersLenErrFmt, index)
		for quarterIndex := 0; quarterIndex < len(actual.Quarters); quarterIndex++ {
			compareTimeSeriesValue(t, actual.Quarters[quarterIndex], expected.Quarters[quarterIndex], "actual did not match expected", "quarters", index, quarterIndex)
		}
	}
	info(t, "Passed")
}

func compareTimeSeriesValue(t *testing.T, actual, expected TimeSeriesValue, reason string, fieldName string, tsIndex, fieldIndex int) {
	if !assert.ObjectsAreEqual(actual, expected) {
		location := fmt.Sprintf("timeseries[%d].%s[%d]", tsIndex, fieldName, fieldIndex)
		jsonDiff := getJSONDiff(actual, expected)

		errReportFmt := "\n%s: %s\n%s: %s\n%s:\n%s"
		t.Fatalf(errReportFmt,
			Bold(Red("Reason")), Red(reason),
			Bold(Red("Location")), Red(location),
			Bold(Red("JSON Diff:")), jsonDiff)
	}
}

func getJSONDiff(a, b interface{}) string {
	astr, _ := json.Marshal(a)
	bstr, _ := json.Marshal(b)

	differ := diff.New()
	d, err := differ.Compare(astr, bstr)
	if err != nil {
		panic(err)
	}

	var aJson map[string]interface{}
	json.Unmarshal(astr, &aJson)

	config := formatter.AsciiFormatterConfig{
		ShowArrayIndex: true,
		Coloring:       true,
	}

	formatter := formatter.NewAsciiFormatter(aJson, config)
	diffString, err := formatter.Format(d)
	if err != nil {
		panic(err)
	}
	return diffString
}

func getCSDBRequestBody(filename string) (io.Reader, string, error) {
	filepath := fmt.Sprintf("resources/inputs/%s.csdb", filename)
	if !exists(filepath) {
		return nil, "", errors.Errorf("input file %s.csdb does not exist", filename)
	}

	body := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(body)

	fileWriter, err := bodyWriter.CreateFormFile("file", filename+".csdb")
	if err != nil {
		return nil, "", err
	}

	f, err := os.Open(filepath)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()

	copied, err := io.Copy(fileWriter, f)
	if err != nil {
		return nil, "", err
	}

	fi, err := f.Stat()
	if err != nil {
		return nil, "", err
	}

	if fi.Size() != copied {
		return nil, "", errors.New("incorrect number of bytes copied")
	}

	contentType := bodyWriter.FormDataContentType()
	bodyWriter.Close()

	return body, contentType, nil
}

func postCSDBFile(body io.Reader, contentType string) (*http.Response, error) {
	// Requests to generate the csdb json for larger files (UKEA, RAGV) can take a loooooong time.
	// I've arbitrarily set the timeout to 20 seconds but feel free to alter this as necessary.
	timeout := time.Duration(20 * time.Second)
	httpClient := http.Client{Timeout: timeout}

	url := fmt.Sprintf("%s/Services/ConvertCSDB", brianHost)
	resp, err := httpClient.Post(url, contentType, body)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func readCSDBResponse(resp *http.Response) ([]TimeSeries, error) {
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var dataJson []TimeSeries
	err = json.Unmarshal(data, &dataJson)
	if err != nil {
		return nil, err
	}
	return dataJson, err
}

func getExpectedResults(filename string) ([]TimeSeries, error) {
	b, err := ioutil.ReadFile(fmt.Sprintf("resources/outputs/%s-csdb.json", filename))
	if err != nil {
		return nil, err
	}

	var expected []TimeSeries
	if err = json.Unmarshal(b, &expected); err != nil {
		return nil, err
	}
	return expected, nil
}

func exists(path string) bool {
	_, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}

func info(t *testing.T, message string) {
	ToColour(t, Cyan, "info", message)
}

func ToColour(t *testing.T, colour func(arg interface{}) Value, prefix string, message string) {
	t.Logf("%s: %s", Bold(colour(prefix)).String(), colour(message).String())
}

func Scenario(t *testing.T, message string) {
	ToColour(t, Green, "Scenario", message)
}

func Given(t *testing.T, message string) {
	ToColour(t, Green, "Given", message)
}

func When(t *testing.T, message string) {
	ToColour(t, Green, "When", message)
}

func Then(t *testing.T, message string) {
	ToColour(t, Green, "Then", message)
}

func And(t *testing.T, message string) {
	ToColour(t, Green, "And", message)
}

func Err(message string) string {
	return Red(message).String()
}
