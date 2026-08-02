package charts

import (
	"testing"
)

func TestGetScatterValues(t *testing.T) {
	headings, values, err := GetScatterValues("1 2\n3 4", "\n", " ", false)
	if err != nil {
		t.Fatal(err)
	}
	if headings != nil {
		t.Errorf("headings = %v, want none", headings)
	}
	want := []ScatterPoint{{X: 1, Y: 2}, {X: 3, Y: 4}}
	for i, row := range want {
		if values[i].X != row.X || values[i].Y != row.Y {
			t.Errorf("row %d = %v, want %v", i, values[i], row)
		}
	}
}

// The first record supplies the axis labels when headings are enabled.
func TestGetScatterValuesHeadings(t *testing.T) {
	headings, values, err := GetScatterValues("xh yh\n1 2", "\n", " ", true)
	if err != nil {
		t.Fatal(err)
	}
	if headings == nil || headings[0] != "xh" || headings[1] != "yh" {
		t.Errorf("headings = %v, want [xh yh]", headings)
	}
	if len(values) != 1 {
		t.Errorf("values = %v, want one row", values)
	}
}

func TestGetScatterValuesErrors(t *testing.T) {
	if _, _, err := GetScatterValues("1 2 3", "\n", " ", false); err == nil ||
		err.Error() != "Each row must have length 2." {
		t.Errorf("wrong-width error = %v", err)
	}
	if _, _, err := GetScatterValues("a b", "\n", " ", false); err == nil ||
		err.Error() != "Values must be numbers in base 10." {
		t.Errorf("non-numeric error = %v", err)
	}
	// The y column is checked too.
	if _, _, err := GetScatterValues("1 b", "\n", " ", false); err == nil {
		t.Error("expected an error for a non-numeric y value")
	}
}

// A third column supplies each point's colour, HTML-escaped as CyberChef does.
func TestGetScatterValuesWithColour(t *testing.T) {
	_, values, err := GetScatterValuesWithColour("1 2 red\n3 4 <b>", "\n", " ", false)
	if err != nil {
		t.Fatal(err)
	}
	if values[0].Colour != "red" {
		t.Errorf("colour = %q, want red", values[0].Colour)
	}
	if values[1].Colour != "&lt;b&gt;" {
		t.Errorf("colour = %q, want the escaped form", values[1].Colour)
	}
}

func TestGetScatterValuesWithColourError(t *testing.T) {
	if _, _, err := GetScatterValuesWithColour("1 2", "\n", " ", false); err == nil ||
		err.Error() != "Each row must have length 3." {
		t.Errorf("wrong-width error = %v", err)
	}
}

// Series data is grouped by name, preserving the order names and x values are
// first seen in.
func TestGetSeriesValues(t *testing.T) {
	xValues, series, err := GetSeriesValues("b 1 10\na 2 20\nb 2 30", "\n", " ")
	if err != nil {
		t.Fatal(err)
	}
	if len(xValues) != 2 || xValues[0] != "1" || xValues[1] != "2" {
		t.Errorf("xValues = %v, want [1 2] in first-seen order", xValues)
	}
	if len(series) != 2 || series[0].Name != "b" || series[1].Name != "a" {
		t.Errorf("series = %v, want b then a", series)
	}
	if series[0].Data["2"] != 30 {
		t.Errorf("series b at x=2 = %v, want 30", series[0].Data["2"])
	}
	if _, ok := series[1].Data["1"]; ok {
		t.Error("series a should have no value at x=1")
	}
}

// Duplicate x values collapse to one entry.
func TestGetSeriesValuesDuplicateX(t *testing.T) {
	xValues, series, err := GetSeriesValues("a 1 10\na 1 20", "\n", " ")
	if err != nil {
		t.Fatal(err)
	}
	if len(xValues) != 1 {
		t.Errorf("xValues = %v, want one entry", xValues)
	}
	if series[0].Data["1"] != 20 {
		t.Errorf("later value should win, got %v", series[0].Data["1"])
	}
}

func TestGetSeriesValuesErrors(t *testing.T) {
	if _, _, err := GetSeriesValues("a 1", "\n", " "); err == nil ||
		err.Error() != "Each row must have length 3." {
		t.Errorf("wrong-width error = %v", err)
	}
	if _, _, err := GetSeriesValues("a 1 x", "\n", " "); err == nil ||
		err.Error() != "Values must be numbers in base 10." {
		t.Errorf("non-numeric error = %v", err)
	}
}

// The colour column is checked for numeric x and y like the others.
func TestGetScatterValuesWithColourNonNumeric(t *testing.T) {
	if _, _, err := GetScatterValuesWithColour("a b red", "\n", " ", false); err == nil {
		t.Error("expected an error for non-numeric values")
	}
}

// The extent of no values is empty rather than a panic.
func TestChartExtentEmpty(t *testing.T) {
	if got := Extent(nil); got != [2]float64{} {
		t.Errorf("extent of nothing = %v, want zero", got)
	}
}
