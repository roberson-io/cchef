package ops

import (
	"strings"
	"testing"
)

// The axis markup is compared against fragments taken from CyberChef's own
// output, produced by running the real operations under Node.

// A left axis: ticks grow leftwards, labels are end-anchored, and the domain
// path runs down the left edge.
func TestRenderAxisLeft(t *testing.T) {
	g := newSVGEl("g").class("axis axis--y").attr("transform", "translate(30, 20)")
	renderAxis(g, axisSpec{
		orient:        axisLeft,
		rng:           [2]float64{100, 0},
		ticks:         []axisTick{{position: 50, label: "1.000000"}},
		tickSizeOuter: 6,
	})
	want := `<g class="axis axis--y" transform="translate(30, 20)" fill="none" font-size="10" ` +
		`font-family="sans-serif" text-anchor="end">` +
		`<path class="domain" stroke="currentColor" d="M-6,100.5H0.5V0.5H-6"></path>` +
		`<g class="tick" opacity="1" transform="translate(0,50.5)">` +
		`<line stroke="currentColor" x2="-6"></line>` +
		`<text fill="currentColor" x="-9" dy="0.32em">1.000000</text></g></g>`
	if got := g.render(); got != want {
		t.Errorf("left axis =\n%s\nwant\n%s", got, want)
	}
}

// A top axis: ticks grow upwards and labels sit above the line.
func TestRenderAxisTop(t *testing.T) {
	g := newSVGEl("g").class("axis axis--x").attr("transform", "translate(50, 50)")
	renderAxis(g, axisSpec{
		orient:        axisTop,
		rng:           [2]float64{0, 430},
		ticks:         []axisTick{{position: 215, label: "a"}},
		tickSizeOuter: 6,
	})
	want := `<g class="axis axis--x" transform="translate(50, 50)" fill="none" font-size="10" ` +
		`font-family="sans-serif" text-anchor="middle">` +
		`<path class="domain" stroke="currentColor" d="M0.5,-6V0.5H430.5V-6"></path>` +
		`<g class="tick" opacity="1" transform="translate(215.5,0)">` +
		`<line stroke="currentColor" y2="-6"></line>` +
		`<text fill="currentColor" y="-9" dy="0em">a</text></g></g>`
	if got := g.render(); got != want {
		t.Errorf("top axis =\n%s\nwant\n%s", got, want)
	}
}

// A bottom axis with a negative outer tick size draws the domain path as full
// height gridlines, which is how the scatter chart uses it.
func TestRenderAxisBottomNegativeOuter(t *testing.T) {
	g := newSVGEl("g").class("axis axis--x").attr("transform", "translate(0,450)")
	renderAxis(g, axisSpec{
		orient:        axisBottom,
		rng:           [2]float64{0, 470},
		ticks:         []axisTick{{position: 0, label: "80"}},
		tickSizeOuter: -450,
	})
	want := `<g class="axis axis--x" transform="translate(0,450)" fill="none" font-size="10" ` +
		`font-family="sans-serif" text-anchor="middle">` +
		`<path class="domain" stroke="currentColor" d="M0.5,-450V0.5H470.5V-450"></path>` +
		`<g class="tick" opacity="1" transform="translate(0.5,0)">` +
		`<line stroke="currentColor" y2="6"></line>` +
		`<text fill="currentColor" y="9" dy="0.71em">80</text></g></g>`
	if got := g.render(); got != want {
		t.Errorf("bottom axis =\n%s\nwant\n%s", got, want)
	}
}

// A left axis with a negative outer tick size, as the scatter chart's y axis
// uses it.
func TestRenderAxisLeftNegativeOuter(t *testing.T) {
	g := newSVGEl("g").class("axis axis--y")
	renderAxis(g, axisSpec{
		orient:        axisLeft,
		rng:           [2]float64{450, 0},
		ticks:         []axisTick{{position: 450, label: "80"}},
		tickSizeOuter: -470,
	})
	want := `<path class="domain" stroke="currentColor" d="M470,450.5H0.5V0.5H470"></path>` +
		`<g class="tick" opacity="1" transform="translate(0,450.5)">` +
		`<line stroke="currentColor" x2="-6"></line>` +
		`<text fill="currentColor" x="-9" dy="0.32em">80</text></g>`
	if got := g.render(); !strings.Contains(got, want) {
		t.Errorf("left axis =\n%s\nwant it to contain\n%s", got, want)
	}
}

// A zero outer tick size collapses the domain path to a plain line.
func TestRenderAxisZeroOuter(t *testing.T) {
	g := newSVGEl("g")
	renderAxis(g, axisSpec{orient: axisLeft, rng: [2]float64{100, 0}, tickSizeOuter: 0})
	if want := `d="M0.5,100.5V0.5"`; !strings.Contains(g.render(), want) {
		t.Errorf("zero-outer left axis = %s, want it to contain %s", g.render(), want)
	}

	h := newSVGEl("g")
	renderAxis(h, axisSpec{orient: axisBottom, rng: [2]float64{0, 470}, tickSizeOuter: 0})
	if want := `d="M0.5,0.5H470.5"`; !strings.Contains(h.render(), want) {
		t.Errorf("zero-outer bottom axis = %s, want it to contain %s", h.render(), want)
	}
}

// A right-hand axis anchors its labels at the start.
func TestRenderAxisRight(t *testing.T) {
	g := newSVGEl("g")
	renderAxis(g, axisSpec{
		orient:        axisRight,
		rng:           [2]float64{100, 0},
		ticks:         []axisTick{{position: 50, label: "5"}},
		tickSizeOuter: 6,
	})
	out := g.render()
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
