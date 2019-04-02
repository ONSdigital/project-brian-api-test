package main

import (
	"bytes"
	"encoding/json"
	"github.com/pkg/errors"
	. "github.com/smartystreets/goconvey/convey"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"reflect"
	"testing"
)

type TimeseriesValue struct {
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
	Years          []TimeseriesValue `json:"years"`
	Quarters       []TimeseriesValue `json:"quarters"`
	Months         []TimeseriesValue `json:"months"`
	SourceDatasets []string          `json:"sourceDatasets"`
	Section        interface{}       `json:"section"`
	Type           string            `json:"type"`
	Description    Description       `json:"description"`
}

func getCSDBRequestBody(filename string) (io.Reader, string, error) {
	body := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(body)

	fileWriter, err := bodyWriter.CreateFormFile("file", filename)
	if err != nil {
		return nil, "", err
	}

	f, err := os.Open("resources/" + filename)
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
	resp, err := http.Post("http://localhost:8083/Services/ConvertCSDB", contentType, body)
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

func getExpectedDataJSON(filename string) ([]TimeSeries, error) {
	b, err := ioutil.ReadFile("resources/" + filename)
	if err != nil {
		return nil, err
	}

	var expected []TimeSeries
	if err = json.Unmarshal(b, &expected); err != nil {
		return nil, err
	}
	return expected, nil
}

func TestBOB(t *testing.T) {
	Convey("given a valid .csdb file upload", t, func() {
		body, contentType, err := getCSDBRequestBody("ott.csdb")
		So(err, ShouldBeNil)

		Convey("when a post request is made", func() {
			response, err := postCSDBFile(body, contentType)
			So(err, ShouldBeNil)

			defer response.Body.Close()

			Convey("then a 200 response status is returned", func() {
				So(response.StatusCode, ShouldEqual, 200)
			})

			Convey("and the expected timeseries json is returned", func() {
				dataJSON, err := readCSDBResponse(response)
				So(err, ShouldBeNil)

				expectedDataJSON, err := getExpectedDataJSON("ott-csdb.json")
				So(err, ShouldBeNil)

				So(dataJSON, ShouldHaveLength, len(expectedDataJSON))
				//So(reflect.DeepEqual(dataJSON, expectedDataJSON), ShouldBeTrue)

				for timeseriesIndex, actualTimeSeries := range dataJSON {
					expectedTimeSeries := expectedDataJSON[timeseriesIndex]

					for yearIndex, actualYear := range actualTimeSeries.Years {
						expectedYear := expectedTimeSeries.Years[yearIndex]
						if !reflect.DeepEqual(actualYear, expectedYear) {
							a, _ := json.MarshalIndent(actualYear, "", "  ")
							b, _ := json.MarshalIndent(expectedYear, "", "  ")
							t.Logf("values do not match timesseries[%d].years[%d]\n", timeseriesIndex, yearIndex)
							So(string(a), ShouldEqual, string(b))
						}
					}
				}
			})
		})
	})
}

func TODO(t *testing.T) {
	Convey("given TODO", t, func() {

		body := &bytes.Buffer{}
		bodyWriter := multipart.NewWriter(body)

		fileWriter, err := bodyWriter.CreateFormFile("file", "ott.csdb")
		So(err, ShouldBeNil)

		f, err := os.Open("resources/ott.csdb")
		So(err, ShouldBeNil)
		defer f.Close()

		copied, err := io.Copy(fileWriter, f)
		So(err, ShouldBeNil)

		fi, err := f.Stat()
		So(err, ShouldBeNil)

		So(fi.Size(), ShouldEqual, copied)

		contentType := bodyWriter.FormDataContentType()
		bodyWriter.Close()

		resp, err := http.Post("http://localhost:8083/Services/ConvertCSDB", contentType, body)
		So(err, ShouldBeNil)

		defer resp.Body.Close()

		respData, err := ioutil.ReadAll(resp.Body)
		So(err, ShouldBeNil)

		b, err := ioutil.ReadFile("resources/ott-csdb.json")
		So(err, ShouldBeNil)

		var expected []interface{}
		err = json.Unmarshal(b, &expected)
		So(err, ShouldBeNil)

		var actual []interface{}
		err = json.Unmarshal(respData, &actual)
		So(err, ShouldBeNil)

		So(actual, ShouldHaveLength, len(expected))

		for i, a := range actual {
			e := expected[i]
			if !reflect.DeepEqual(a, e) {

				actaulB, _ := json.MarshalIndent(a, "", "  ")
				expectedB, _ := json.MarshalIndent(e, "", "  ")

				t.Fatalf("comparison failure for entry %d\nActual: %s\nExpected: %s\n", i, string(actaulB), string(expectedB))
			}
		}
	})
}
