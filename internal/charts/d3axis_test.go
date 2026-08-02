package charts

import (
	"strings"
	"testing"
)

// The axis markup is compared against fragments taken from CyberChef's own
// output, produced by running the real operations under Node.

// A left axis: ticks grow leftwards, labels are end-anchored, and the domain
// path runs down the left edge.
func TestRenderAxisLeft(t *testing.T) {
	g := NewSVGEl("g").Class("axis axis--y").Attr("transform", "translate(30, 20)")
	RenderAxis(g, AxisSpec{
		Orient:        AxisLeft,
		Rng:           [2]float64{100, 0},
		Ticks:         []AxisTick{{Position: 50, Label: "1.000000"}},
		TickSizeOuter: 6,
	})
	want := `<g class="axis axis--y" transform="translate(30, 20)" fill="none" font-size="10" ` +
		`font-family="sans-serif" text-anchor="end">` +
		`<path class="domain" stroke="currentColor" d="M-6,100.5H0.5V0.5H-6"></path>` +
		`<g class="tick" opacity="1" transform="translate(0,50.5)">` +
		`<line stroke="currentColor" x2="-6"></line>` +
		`<text fill="currentColor" x="-9" dy="0.32em">1.000000</text></g></g>`
	if got := g.Render(); got != want {
		t.Errorf("left axis =\n%s\nwant\n%s", got, want)
	}
}

// A top axis: ticks grow upwards and labels sit above the line.
func TestRenderAxisTop(t *testing.T) {
	g := NewSVGEl("g").Class("axis axis--x").Attr("transform", "translate(50, 50)")
	RenderAxis(g, AxisSpec{
		Orient:        AxisTop,
		Rng:           [2]float64{0, 430},
		Ticks:         []AxisTick{{Position: 215, Label: "a"}},
		TickSizeOuter: 6,
	})
	want := `<g class="axis axis--x" transform="translate(50, 50)" fill="none" font-size="10" ` +
		`font-family="sans-serif" text-anchor="middle">` +
		`<path class="domain" stroke="currentColor" d="M0.5,-6V0.5H430.5V-6"></path>` +
		`<g class="tick" opacity="1" transform="translate(215.5,0)">` +
		`<line stroke="currentColor" y2="-6"></line>` +
		`<text fill="currentColor" y="-9" dy="0em">a</text></g></g>`
	if got := g.Render(); got != want {
		t.Errorf("top axis =\n%s\nwant\n%s", got, want)
	}
}

// A bottom axis with a negative outer tick size draws the domain path as full
// height gridlines, which is how the scatter chart uses it.
func TestRenderAxisBottomNegativeOuter(t *testing.T) {
	g := NewSVGEl("g").Class("axis axis--x").Attr("transform", "translate(0,450)")
	RenderAxis(g, AxisSpec{
		Orient:        AxisBottom,
		Rng:           [2]float64{0, 470},
		Ticks:         []AxisTick{{Position: 0, Label: "80"}},
		TickSizeOuter: -450,
	})
	want := `<g class="axis axis--x" transform="translate(0,450)" fill="none" font-size="10" ` +
		`font-family="sans-serif" text-anchor="middle">` +
		`<path class="domain" stroke="currentColor" d="M0.5,-450V0.5H470.5V-450"></path>` +
		`<g class="tick" opacity="1" transform="translate(0.5,0)">` +
		`<line stroke="currentColor" y2="6"></line>` +
		`<text fill="currentColor" y="9" dy="0.71em">80</text></g></g>`
	if got := g.Render(); got != want {
		t.Errorf("bottom axis =\n%s\nwant\n%s", got, want)
	}
}

// A left axis with a negative outer tick size, as the scatter chart's y axis
// uses it.
func TestRenderAxisLeftNegativeOuter(t *testing.T) {
	g := NewSVGEl("g").Class("axis axis--y")
	RenderAxis(g, AxisSpec{
		Orient:        AxisLeft,
		Rng:           [2]float64{450, 0},
		Ticks:         []AxisTick{{Position: 450, Label: "80"}},
		TickSizeOuter: -470,
	})
	want := `<path class="domain" stroke="currentColor" d="M470,450.5H0.5V0.5H470"></path>` +
		`<g class="tick" opacity="1" transform="translate(0,450.5)">` +
		`<line stroke="currentColor" x2="-6"></line>` +
		`<text fill="currentColor" x="-9" dy="0.32em">80</text></g>`
	if got := g.Render(); !strings.Contains(got, want) {
		t.Errorf("left axis =\n%s\nwant it to contain\n%s", got, want)
	}
}

// A zero outer tick size collapses the domain path to a plain line.
func TestRenderAxisZeroOuter(t *testing.T) {
	g := NewSVGEl("g")
	RenderAxis(g, AxisSpec{Orient: AxisLeft, Rng: [2]float64{100, 0}, TickSizeOuter: 0})
	if want := `d="M0.5,100.5V0.5"`; !strings.Contains(g.Render(), want) {
		t.Errorf("zero-outer left axis = %s, want it to contain %s", g.Render(), want)
	}

	h := NewSVGEl("g")
	RenderAxis(h, AxisSpec{Orient: AxisBottom, Rng: [2]float64{0, 470}, TickSizeOuter: 0})
	if want := `d="M0.5,0.5H470.5"`; !strings.Contains(h.Render(), want) {
		t.Errorf("zero-outer bottom axis = %s, want it to contain %s", h.Render(), want)
	}
}

// A right-hand axis anchors its labels at the start.
func TestRenderAxisRight(t *testing.T) {
	g := NewSVGEl("g")
	RenderAxis(g, AxisSpec{
		Orient:        axisRight,
		Rng:           [2]float64{100, 0},
		Ticks:         []AxisTick{{Position: 50, Label: "5"}},
		TickSizeOuter: 6,
	})
	out := g.Render()
	for _, want := range []string{
		`text-anchor="start"`,
		`<line stroke="currentColor" x2="6"></line>`,
		`<text fill="currentColor" x="9" dy="0.32em">5</text>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("right axis = %s\nwant it to contain %s", out, want)
		}
	}
}
