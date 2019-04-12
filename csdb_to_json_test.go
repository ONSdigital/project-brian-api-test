package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/ONSdigital/project-brian-api-test/assert"
	"github.com/pkg/errors"
	. "github.com/smartystreets/goconvey/convey"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"testing"
	"time"
)

const (
	generateURL = "http://localhost:8083/Services/ConvertCSDB"
)

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

func Test_Generate_OTT_CSDB_TimeSeries(t *testing.T) {
	testCSDBToJson(t, "ott")
}

func Test_Generate_BB_CSDB_TimeSeries(t *testing.T) {
	testCSDBToJson(t, "bb")
}

func Test_Generate_BERD_CSDB_TimeSeries(t *testing.T) {
	t.Logf("testing generated json for BERD.csdb")
	testCSDBToJson(t, "berd")
}

func Test_Generate_UKEA_CSDB_TimeSeries(t *testing.T) {
	testCSDBToJson(t, "ukea")
}

func Test_Generate_RAGV_CSDB_TimeSeries(t *testing.T) {
	testCSDBToJson(t, "ragv")
}

func Test_Generate_SPPI_CSDB_TimeSeries(t *testing.T) {
	testCSDBToJson(t, "sppi")
}

func testCSDBToJson(t *testing.T, filename string) {
	_, err := os.Stat("resources/outputs")
	if err != nil && os.IsNotExist(err) {
		t.Fatalf("dir %q does not exist make sure you have unzipped the %q outputs.zip before running the tests", "resources/outputs", "resources/outputs.zip")
	}

	Convey(fmt.Sprintf("given a valid %s.csdb file", filename), t, func() {
		body, contentType, err := getCSDBRequestBody(filename)
		So(err, ShouldBeNil)

		Convey("when a POST request is sent to /Services/ConvertCSDB", func() {
			response, err := postCSDBFile(body, contentType)
			So(err, ShouldBeNil)

			defer response.Body.Close()

			Convey("then a 200 response status is returned", func() {
				So(response.StatusCode, ShouldEqual, 200)
			})

			actualTimeSeries, err := readCSDBResponse(response)
			So(err, ShouldBeNil)

			expectedTimeSeries, err := getExpectedResults(filename)
			So(err, ShouldBeNil)

			Convey("and the correct timeSeries json response is returned", func() {
				So(actualTimeSeries, ShouldHaveLength, len(expectedTimeSeries))

				var actual TimeSeries
				var expected TimeSeries

				for index := 0; index < len(actualTimeSeries); index++ {
					actual = actualTimeSeries[index]
					expected = expectedTimeSeries[index]

					So(actual.Description, ShouldResemble, expected.Description)

					So(actual.Type, ShouldResemble, expected.Type)

					So(actual.Years, ShouldHaveLength, len(expected.Years))
					for yearIndex := 0; yearIndex < len(actual.Years); yearIndex++ {
						So(actual.Years[yearIndex], assert.ShouldEqualYear(index, yearIndex), expected.Years[yearIndex])
					}

					So(actual.Months, ShouldHaveLength, len(expected.Months))

					for monthIndex := 0; monthIndex < len(actual.Months); monthIndex++ {
						So(actual.Months[monthIndex], assert.ShouldEqualMonth(index, monthIndex), expected.Months[monthIndex])
					}

					So(actual.Quarters, ShouldHaveLength, len(expected.Quarters))

					for quarterIndex := 0; quarterIndex < len(actual.Quarters); quarterIndex++ {
						So(actual.Quarters[quarterIndex], assert.ShouldEqualQuarter(index, quarterIndex), expected.Quarters[quarterIndex])
					}
				}
			})
		})
	})
}

func getCSDBRequestBody(filename string) (io.Reader, string, error) {
	body := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(body)

	fileWriter, err := bodyWriter.CreateFormFile("file", filename+".csdb")
	if err != nil {
		return nil, "", err
	}

	f, err := os.Open(fmt.Sprintf("resources/inputs/%s.csdb", filename))
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

	resp, err := httpClient.Post(generateURL, contentType, body)
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
