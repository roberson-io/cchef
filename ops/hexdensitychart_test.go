package ops

import (
	"strings"
	"testing"
)

func TestHexDensityChart(t *testing.T) {
	assertChartGolden(t, "hexdensity_basic", "Hex Density chart",
		"100 100\n200 200\n300 300\n400 400\n500 500",
		"Line feed", "Space", float64(25), float64(15), true, "", "", true, "white", "black", true)
}

// Without headings or empty hexagons, and with edges off.
func TestHexDensityChartNoLabels(t *testing.T) {
	assertChartGolden(t, "hexdensity_nolabels", "Hex Density chart",
		"1 2\n3 4\n5 9\n2 3\n4 4",
		"Line feed", "Space", float64(10), float64(8), false, "xl", "yl", false, "white", "blue", false)
}

func TestHexDensityChartErrors(t *testing.T) {
	_, err := runOp(t, "Hex Density chart", "1 2 3",
		"Line feed", "Space", float64(25), float64(15), false, "", "", false, "white", "black", false)
	if err == nil || err.Error() != "Each row must have length 2." {
		t.Errorf("error = %v, want a row-length error", err)
	}
}

// Points landing in the same hexagon are counted together.
func TestHexDensityChartGroupsPoints(t *testing.T) {
	out, err := runOp(t, "Hex Density chart", "1 1\n1 1\n1 1\n100 100",
		"Line feed", "Space", float64(25), float64(15), false, "", "", false, "white", "black", false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Count: 3") {
		t.Error("expected three coincident points to share a hexagon")
	}
	if !strings.Contains(out, "Count: 1") {
		t.Error("expected the distant point in its own hexagon")
	}
}
