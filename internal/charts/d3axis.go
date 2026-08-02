package charts

import "github.com/roberson-io/cchef/internal/jsnum"

// A port of d3-axis as CyberChef's chart operations use it. Callers resolve
// their own scale into tick positions and labels; this renders the SVG.

// AxisOrient is which side of the plot an axis is drawn on, matching
// d3-axis.
type AxisOrient int

// The four orientations; only top, bottom and left are rendered here.
const (
	AxisTop AxisOrient = iota
	axisRight
	AxisBottom
	AxisLeft
)

// d3-axis's fixed geometry.
const (
	// axisOffset is d3's half-pixel alignment nudge. d3 drops it on displays
	// with a device pixel ratio above 1; off a browser it is always applied.
	axisOffset = 0.5
	// AxisTickSizeInner is the length of a tick mark.
	AxisTickSizeInner = 6
	// axisTickPadding is the gap between a tick mark and its label.
	axisTickPadding = 3
	// axisFontSize is the axis label size in points.
	axisFontSize = "10"
)

// AxisTick is one tick: where it sits along the range, and its label.
type AxisTick struct {
	Position float64
	Label    string
}

// AxisSpec describes an axis to render.
type AxisSpec struct {
	Orient        AxisOrient
	Rng           [2]float64
	Ticks         []AxisTick
	TickSizeOuter float64
}

// RenderAxis appends d3-axis's markup — the domain path then one group per
// tick — to g, and sets the presentation attributes d3 puts on the axis group.
func RenderAxis(g *SVGEl, spec AxisSpec) {
	k := 1.0
	if spec.Orient == AxisTop || spec.Orient == AxisLeft {
		k = -1
	}
	vertical := spec.Orient == AxisLeft || spec.Orient == axisRight
	range0 := spec.Rng[0] + axisOffset
	range1 := spec.Rng[1] + axisOffset

	g.Append("path").Class("domain").Attr("stroke", "currentColor").
		Attr("d", axisDomainPath(vertical, k, spec.TickSizeOuter, range0, range1))

	for _, tick := range spec.Ticks {
		renderAxisTick(g, spec.Orient, k, vertical, tick)
	}

	g.Attr("fill", "none").
		Attr("font-size", axisFontSize).
		Attr("font-family", "sans-serif").
		Attr("text-anchor", axisTextAnchor(spec.Orient))
}

// axisDomainPath builds the axis's domain path. A zero outer tick size draws a
// plain line; otherwise the ends turn away from the axis by that much, which the
// scatter chart uses (with a negative size) to draw full-length gridlines.
func axisDomainPath(vertical bool, k, tickSizeOuter, range0, range1 float64) string {
	if vertical {
		if tickSizeOuter == 0 {
			return "M" + jsnum.Format(axisOffset) + "," + jsnum.Format(range0) + "V" + jsnum.Format(range1)
		}
		outer := jsnum.Format(k * tickSizeOuter)
		return "M" + outer + "," + jsnum.Format(range0) + "H" + jsnum.Format(axisOffset) +
			"V" + jsnum.Format(range1) + "H" + outer
	}
	if tickSizeOuter == 0 {
		return "M" + jsnum.Format(range0) + "," + jsnum.Format(axisOffset) + "H" + jsnum.Format(range1)
	}
	outer := jsnum.Format(k * tickSizeOuter)
	return "M" + jsnum.Format(range0) + "," + outer + "V" + jsnum.Format(axisOffset) +
		"H" + jsnum.Format(range1) + "V" + outer
}

// renderAxisTick appends one tick group: its mark and its label.
func renderAxisTick(g *SVGEl, orient AxisOrient, k float64, vertical bool, tick AxisTick) {
	pos := jsnum.Format(tick.Position + axisOffset)
	transform := "translate(" + pos + ",0)"
	if vertical {
		transform = "translate(0," + pos + ")"
	}

	group := g.Append("g").Class("tick").Attr("opacity", "1").Attr("transform", transform)

	// The tick mark and label extend along x for a vertical axis, y otherwise.
	axisDir := "y"
	if vertical {
		axisDir = "x"
	}
	group.Append("line").Attr("stroke", "currentColor").
		Attr(axisDir+"2", jsnum.Format(k*AxisTickSizeInner))
	group.Append("text").Attr("fill", "currentColor").
		Attr(axisDir, jsnum.Format(k*(AxisTickSizeInner+axisTickPadding))).
		Attr("dy", axisTickBaseline(orient)).
		Text(tick.Label)
}

// axisTickBaseline is the label's vertical offset for each orientation.
func axisTickBaseline(orient AxisOrient) string {
	switch orient {
	case AxisTop:
		return "0em"
	case AxisBottom:
		return "0.71em"
	default:
		return "0.32em"
	}
}

// axisTextAnchor is how labels align for each orientation.
func axisTextAnchor(orient AxisOrient) string {
	switch orient {
	case axisRight:
		return "start"
	case AxisLeft:
		return "end"
	default:
		return "middle"
	}
}
