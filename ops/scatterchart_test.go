package ops

import (
	"os"
	"strings"
	"testing"
)

// loadChartGolden reads a golden SVG. The goldens are CyberChef's own output,
// produced by running the real operations under Node, with three mechanical
// substitutions applied: D3's `__data__` bindings removed, `clippath` corrected
// to `clipPath`, and the trailing "; " dropped from style attributes.
func loadChartGolden(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile("testdata/chart_" + name + ".svg")
	if err != nil {
		t.Fatalf("read golden %s: %v", name, err)
	}
	return string(b)
}

// assertChartGolden runs a chart operation and compares its SVG to the golden,
// reporting the first difference rather than dumping kilobytes of markup.
func assertChartGolden(t *testing.T, golden, op, input string, args ...any) {
	t.Helper()
	got, err := runOp(t, op, input, args...)
	if err != nil {
		t.Fatalf("%s: %v", op, err)
	}
	want := loadChartGolden(t, golden)
	if got == want {
		return
	}
	for i := 0; i < len(got) && i < len(want); i++ {
		if got[i] != want[i] {
			t.Fatalf("%s: SVG differs at byte %d\n got: …%s\nwant: …%s",
				golden, i, excerpt(got, i), excerpt(want, i))
		}
	}
	t.Fatalf("%s: SVG length %d, want %d\n got tail: %s\nwant tail: %s",
		golden, len(got), len(want), tail(got, len(want)), tail(want, len(got)))
}

// excerpt returns a window of s around index i for a difference report.
func excerpt(s string, i int) string {
	start := max(0, i-60)
	end := min(len(s), i+60)
	return strings.ReplaceAll(s[start:end], "\n", "\\n")
}

// tail returns what one string has beyond the length of the other.
func tail(s string, from int) string {
	if from >= len(s) {
		return "(nothing)"
	}
	return strings.ReplaceAll(s[from:min(len(s), from+120)], "\n", "\\n")
}

func TestScatterChart(t *testing.T) {
	assertChartGolden(t, "scatter_basic", "Scatter chart",
		"100 100\n200 200\n300 300\n400 400\n500 500",
		"Line feed", "Space", false, "time", "stress", "black", float64(5), false)
}

// With headings enabled the first record supplies the axis labels.
func TestScatterChartHeadings(t *testing.T) {
	assertChartGolden(t, "scatter_headings", "Scatter chart",
		"xh yh\n1 2\n3 4\n5 9",
		"Line feed", "Space", true, "", "", "red", float64(10), false)
}

// A third column can colour each point individually.
func TestScatterChartColourColumn(t *testing.T) {
	assertChartGolden(t, "scatter_colour3", "Scatter chart",
		"1 2 red\n3 4 blue\n5 9 green",
		"Line feed", "Space", false, "x", "y", "black", float64(4), true)
}

func TestScatterChartErrors(t *testing.T) {
	_, err := runOp(t, "Scatter chart", "1 2 3", "Line feed", "Space", false, "", "", "black", float64(5), false)
	if err == nil || err.Error() != "Each row must have length 2." {
		t.Errorf("error = %v, want a row-length error", err)
	}
	_, err = runOp(t, "Scatter chart", "a b", "Line feed", "Space", false, "", "", "black", float64(5), false)
	if err == nil || err.Error() != "Values must be numbers in base 10." {
		t.Errorf("error = %v, want a numeric error", err)
	}
}

// The colour option is HTML-escaped, so it cannot break out of the attribute.
func TestScatterChartEscapesColour(t *testing.T) {
	out, err := runOp(t, "Scatter chart", "1 2\n3 4",
		"Line feed", "Space", false, "", "", `black" onload="alert(1)`, float64(5), false)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, `onload="alert(1)"`) {
		t.Error("colour option escaped its attribute")
	}
	if !strings.Contains(out, "&quot;") {
		t.Error("expected the quote in the colour to be escaped")
	}
}
