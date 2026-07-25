package ops

import (
	"strings"
	"testing"
)

func TestHeatmapChart(t *testing.T) {
	assertChartGolden(t, "heatmap_basic", "Heatmap chart",
		"100 100\n200 200\n300 300\n400 400\n500 500",
		"Line feed", "Space", float64(25), float64(25), true, "", "", false, "white", "black")
}

// Headings supply the labels, and edges can be drawn around each bin.
func TestHeatmapChartHeadingsAndEdges(t *testing.T) {
	assertChartGolden(t, "heatmap_headings", "Heatmap chart",
		"xh yh\n1 2\n3 4\n5 9\n2 3",
		"Line feed", "Space", float64(5), float64(5), true, "", "", true, "white", "red")
}

func TestHeatmapChartErrors(t *testing.T) {
	args := func(v, h float64, input string) (string, []any) {
		return input, []any{"Line feed", "Space", v, h, false, "", "", false, "white", "black"}
	}
	for _, tc := range []struct {
		name, want string
		v, h       float64
		input      string
	}{
		{"zero vertical bins", "Number of vertical bins must be greater than 0", 0, 5, "1 2\n3 4"},
		{"zero horizontal bins", "Number of horizontal bins must be greater than 0", 5, 0, "1 2\n3 4"},
		{"flat x", "Cannot pack points. There is no difference between the minimum and maximum X coordinate.", 5, 5, "1 2\n1 4"},
		{"flat y", "Cannot pack points. There is no difference between the minimum and maximum Y coordinate.", 5, 5, "1 2\n3 2"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input, a := args(tc.v, tc.h, tc.input)
			_, err := runOp(t, "Heatmap chart", input, a...)
			if err == nil || err.Error() != tc.want {
				t.Errorf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

// An unparseable colour falls back to the other end of the ramp rather than
// producing broken markup.
func TestHeatmapChartBadColour(t *testing.T) {
	out, err := runOp(t, "Heatmap chart", "1 2\n3 4\n5 9",
		"Line feed", "Space", float64(3), float64(3), false, "", "", false, "white", "notacolour")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out, "<svg width") {
		t.Errorf("output does not start with <svg width: %.40s", out)
	}
	if strings.Contains(out, "NaN") {
		t.Error("an unparseable colour leaked NaN into the SVG")
	}
}

// Malformed input is reported before any binning happens.
func TestHeatmapChartBadInput(t *testing.T) {
	_, err := runOp(t, "Heatmap chart", "1 2 3",
		"Line feed", "Space", float64(5), float64(5), false, "", "", false, "white", "black")
	if err == nil || err.Error() != "Each row must have length 2." {
		t.Errorf("error = %v, want a row-length error", err)
	}
}

// With nothing to scale against, every value takes the ramp's starting colour.
func TestSequentialColourZeroMax(t *testing.T) {
	colour := sequentialColour("white", "black", 0)
	if got := colour(0); got != "rgb(255, 255, 255)" {
		t.Errorf("colour(0) = %q, want the ramp start", got)
	}
	if got := colour(5); got != "rgb(255, 255, 255)" {
		t.Errorf("colour(5) = %q, want the ramp start", got)
	}
}
