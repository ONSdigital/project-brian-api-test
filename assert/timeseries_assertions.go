package assert

import (
	"encoding/json"
	"fmt"
	"github.com/smartystreets/assertions"
	. "github.com/smartystreets/goconvey/convey"
)

func ShouldEqualYear(timeseriesIndex int, yearIndex int) func(actual interface{}, expected ...interface{}) string {
	return TimeSeriesValueShouldEqual(timeseriesIndex, yearIndex, "years")
}

func ShouldEqualMonth(timeseriesIndex int, monthIndex int) func(actual interface{}, expected ...interface{}) string {
	return TimeSeriesValueShouldEqual(timeseriesIndex, monthIndex, "month")
}

func ShouldEqualQuarter(timeseriesIndex int, quarterIndex int) func(actual interface{}, expected ...interface{}) string {
	return TimeSeriesValueShouldEqual(timeseriesIndex, quarterIndex, "quarters")
}

func TimeSeriesValueShouldEqual(index int, subIndex int, field string) func(actual interface{}, expected ...interface{}) string {
	return func(actual interface{}, expected ...interface{}) string {
		message := ShouldResemble(actual, expected[0])
		if len(message) > 0 {
			return generateFailureView(message, index, subIndex, field, actual, expected[0])
		}
		return message
	}
}

func generateFailureView(message string, index int, subIndex int, field string, actual interface{}, expected interface{}) string {
	var view assertions.FailureView
	json.Unmarshal([]byte(message), &view)

	actualStr, _ := json.MarshalIndent(actual, "", "  ")
	expectedStr, _ := json.MarshalIndent(expected, "", "  ")

	view.Message = fmt.Sprintf("Info: timeseries[%d].%s[%d]\n\n%s", index, field, subIndex, view.Message)
	view.Actual = string(actualStr)
	view.Expected = string(expectedStr)

	b, err := json.Marshal(view)
	So(err, ShouldBeNil)
	return string(b)

}
