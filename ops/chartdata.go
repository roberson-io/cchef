package ops

import (
	"errors"
	"math"
	"strconv"
	"strings"
)

// Shared data parsing for the chart operations, ported from CyberChef's
// src/core/lib/Charts.mjs.

// recordDelimiterOptions and fieldDelimiterOptions are the delimiter choices the
// chart operations offer.
var (
	recordDelimiterOptions = []string{"Line feed", "CRLF"}
	fieldDelimiterOptions  = []string{"Space", "Comma", "Semi-colon", "Colon", "Tab"}
)

// Default chart colours.
const (
	colourMin = "white"
	colourMax = "black"
)

// scatterPoint is one plotted point, optionally carrying its own colour.
type scatterPoint struct {
	x, y   float64
	colour string
}

// chartSeries is one named series of the series chart, keyed by x value.
type chartSeries struct {
	name string
	data map[string]float64
}

// getValues splits input into records of exactly length fields, optionally
// taking the first record as column headings.
func getValues(input, recordDelimiter, fieldDelimiter string, headingsIncluded bool, length int) ([]string, [][]string, error) {
	var headings []string
	var values [][]string

	for i, row := range strings.Split(input, recordDelimiter) {
		split := strings.Split(row, fieldDelimiter)
		if len(split) != length {
			//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
			return nil, nil, errors.New("Each row must have length " + strconv.Itoa(length) + ".")
		}
		if headingsIncluded && i == 0 {
			headings = split
			continue
		}
		values = append(values, split)
	}
	return headings, values, nil
}

// errChartValuesNotNumbers is CyberChef's message for an unparseable value.
//
//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
var errChartValuesNotNumbers = errors.New("Values must be numbers in base 10.")

// parseChartXY parses a row's first two fields as numbers.
func parseChartXY(row []string) (float64, float64, error) {
	x, y := jsParseFloat(row[0]), jsParseFloat(row[1])
	if math.IsNaN(x) || math.IsNaN(y) {
		return 0, 0, errChartValuesNotNumbers
	}
	return x, y, nil
}

// getScatterValues parses two-column input into points.
func getScatterValues(input, recordDelimiter, fieldDelimiter string, headingsIncluded bool) ([]string, []scatterPoint, error) {
	headings, rows, err := getValues(input, recordDelimiter, fieldDelimiter, headingsIncluded, 2)
	if err != nil {
		return nil, nil, err
	}
	points := make([]scatterPoint, 0, len(rows))
	for _, row := range rows {
		x, y, err := parseChartXY(row)
		if err != nil {
			return nil, nil, err
		}
		points = append(points, scatterPoint{x: x, y: y})
	}
	return headings, points, nil
}

// getScatterValuesWithColour parses three-column input, the third column giving
// each point's colour.
func getScatterValuesWithColour(input, recordDelimiter, fieldDelimiter string, headingsIncluded bool) ([]string, []scatterPoint, error) {
	headings, rows, err := getValues(input, recordDelimiter, fieldDelimiter, headingsIncluded, 3)
	if err != nil {
		return nil, nil, err
	}
	points := make([]scatterPoint, 0, len(rows))
	for _, row := range rows {
		x, y, err := parseChartXY(row)
		if err != nil {
			return nil, nil, err
		}
		points = append(points, scatterPoint{x: x, y: y, colour: escapeHTML(row[2])})
	}
	return headings, points, nil
}

// getSeriesValues parses three-column input — series name, x value, y value —
// into the distinct x values and one entry per series. Both keep the order they
// are first seen in, as JavaScript's Set and object key order give.
func getSeriesValues(input, recordDelimiter, fieldDelimiter string) ([]string, []chartSeries, error) {
	_, rows, err := getValues(input, recordDelimiter, fieldDelimiter, false, 3)
	if err != nil {
		return nil, nil, err
	}

	var xValues []string
	seenX := make(map[string]bool)
	var series []chartSeries
	index := make(map[string]int)

	for _, row := range rows {
		name, x := row[0], row[1]
		value := jsParseFloat(row[2])
		if math.IsNaN(value) {
			return nil, nil, errChartValuesNotNumbers
		}
		if !seenX[x] {
			seenX[x] = true
			xValues = append(xValues, x)
		}
		i, ok := index[name]
		if !ok {
			i = len(series)
			index[name] = i
			series = append(series, chartSeries{name: name, data: map[string]float64{}})
		}
		series[i].data[x] = value
	}
	return xValues, series, nil
}

// chartExtent is d3.extent: the smallest and largest of the values.
func chartExtent(values []float64) [2]float64 {
	if len(values) == 0 {
		return [2]float64{}
	}
	lo, hi := values[0], values[0]
	for _, v := range values[1:] {
		lo, hi = math.Min(lo, v), math.Max(hi, v)
	}
	return [2]float64{lo, hi}
}
