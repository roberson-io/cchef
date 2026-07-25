package ops

import (
	"strings"
	"testing"
)

func TestSeriesChart(t *testing.T) {
	assertChartGolden(t, "series_basic", "Series chart",
		"a 1 10\na 2 20\na 3 15\nb 1 5\nb 2 25\nb 3 12",
		"Line feed", "Space", "", float64(1), "mediumseagreen, dodgerblue, tomato")
}

// Upstream's regression case: x-axis values must not be able to inject markup
// into the serialised SVG.
func TestSeriesChartEscapesXValues(t *testing.T) {
	assertChartGolden(t, "series_escape", "Series chart",
		`s,x"><script>globalThis.z=1</script><g a=",1`,
		"Line feed", "Comma", "", float64(1), "red")
}

// The same case stated as the upstream fixture states it.
func TestSeriesChartNoInjection(t *testing.T) {
	out, err := runOp(t, "Series chart", `s,x"><script>globalThis.z=1</script><g a=",1`,
		"Line feed", "Comma", "", float64(1), "red")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "<script>") {
		t.Error("x-axis value injected a script element")
	}
	if strings.Contains(out, "__data__=") {
		t.Error("D3 bound data leaked into an attribute")
	}
	if !strings.HasPrefix(out, "<svg width") {
		t.Errorf("output does not start with <svg width: %.40s", out)
	}
}

// Series colours are HTML-escaped before reaching the stroke attribute.
func TestSeriesChartEscapesColours(t *testing.T) {
	out, err := runOp(t, "Series chart", "a 1 10\na 2 20",
		"Line feed", "Space", "", float64(1), `red" onload="alert(1)`)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, `onload="alert(1)"`) {
		t.Error("series colour escaped its attribute")
	}
}

func TestSeriesChartErrors(t *testing.T) {
	_, err := runOp(t, "Series chart", "a 1", "Line feed", "Space", "", float64(1), "red")
	if err == nil || err.Error() != "Each row must have length 3." {
		t.Errorf("error = %v, want a row-length error", err)
	}
	_, err = runOp(t, "Series chart", "a 1 z", "Line feed", "Space", "", float64(1), "red")
	if err == nil || err.Error() != "Values must be numbers in base 10." {
		t.Errorf("error = %v, want a numeric error", err)
	}
}

// A series missing a value at some x skips both the line segment and the point,
// rather than drawing to a gap.
func TestSeriesChartSparseSeries(t *testing.T) {
	out, err := runOp(t, "Series chart", "a 1 10\na 3 30\nb 1 5\nb 2 25\nb 3 15",
		"Line feed", "Space", "", float64(1), "red, blue")
	if err != nil {
		t.Fatal(err)
	}
	// Series a has no value at x=2, so it draws two points to series b's three.
	if got := strings.Count(out, "<circle"); got != 5 {
		t.Errorf("drew %d points, want 5 (2 for the sparse series, 3 for the full one)", got)
	}
	// x values keep their first-seen order — 1, 3, 2 — so the pairs joined are
	// (1,3) and (3,2). Series a has no value at 2, so it draws only the first
	// segment where series b draws both.
	if got := strings.Count(out, "z "); got != 3 {
		t.Errorf("drew %d line segments, want 3 (1 for the sparse series, 2 for the full one)", got)
	}
}
