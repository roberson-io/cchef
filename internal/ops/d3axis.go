package ops

// A port of d3-axis as CyberChef's chart operations use it. Callers resolve
// their own scale into tick positions and labels; this renders the SVG.

// Axis orientations, matching d3-axis.
type axisOrient int

const (
	axisTop axisOrient = iota
	axisRight
	axisBottom
	axisLeft
)

// d3-axis's fixed geometry.
const (
	// axisOffset is d3's half-pixel alignment nudge. d3 drops it on displays
	// with a device pixel ratio above 1; off a browser it is always applied.
	axisOffset = 0.5
	// axisTickSizeInner is the length of a tick mark.
	axisTickSizeInner = 6
	// axisTickPadding is the gap between a tick mark and its label.
	axisTickPadding = 3
	// axisFontSize is the axis label size in points.
	axisFontSize = "10"
)

// axisTick is one tick: where it sits along the range, and its label.
type axisTick struct {
	position float64
	label    string
}

// axisSpec describes an axis to render.
type axisSpec struct {
	orient        axisOrient
	rng           [2]float64
	ticks         []axisTick
	tickSizeOuter float64
}

// renderAxis appends d3-axis's markup — the domain path then one group per
// tick — to g, and sets the presentation attributes d3 puts on the axis group.
func renderAxis(g *svgEl, spec axisSpec) {
	k := 1.0
	if spec.orient == axisTop || spec.orient == axisLeft {
		k = -1
	}
	vertical := spec.orient == axisLeft || spec.orient == axisRight
	range0 := spec.rng[0] + axisOffset
	range1 := spec.rng[1] + axisOffset

	g.append("path").class("domain").attr("stroke", "currentColor").
		attr("d", axisDomainPath(vertical, k, spec.tickSizeOuter, range0, range1))

	for _, tick := range spec.ticks {
		renderAxisTick(g, spec.orient, k, vertical, tick)
	}

	g.attr("fill", "none").
		attr("font-size", axisFontSize).
		attr("font-family", "sans-serif").
		attr("text-anchor", axisTextAnchor(spec.orient))
}

// axisDomainPath builds the axis's domain path. A zero outer tick size draws a
// plain line; otherwise the ends turn away from the axis by that much, which the
// scatter chart uses (with a negative size) to draw full-length gridlines.
func axisDomainPath(vertical bool, k, tickSizeOuter, range0, range1 float64) string {
	if vertical {
		if tickSizeOuter == 0 {
			return "M" + jsNum(axisOffset) + "," + jsNum(range0) + "V" + jsNum(range1)
		}
		outer := jsNum(k * tickSizeOuter)
		return "M" + outer + "," + jsNum(range0) + "H" + jsNum(axisOffset) +
			"V" + jsNum(range1) + "H" + outer
	}
	if tickSizeOuter == 0 {
		return "M" + jsNum(range0) + "," + jsNum(axisOffset) + "H" + jsNum(range1)
	}
	outer := jsNum(k * tickSizeOuter)
	return "M" + jsNum(range0) + "," + outer + "V" + jsNum(axisOffset) +
		"H" + jsNum(range1) + "V" + outer
}

// renderAxisTick appends one tick group: its mark and its label.
func renderAxisTick(g *svgEl, orient axisOrient, k float64, vertical bool, tick axisTick) {
	pos := jsNum(tick.position + axisOffset)
	transform := "translate(" + pos + ",0)"
	if vertical {
		transform = "translate(0," + pos + ")"
	}

	group := g.append("g").class("tick").attr("opacity", "1").attr("transform", transform)

	// The tick mark and label extend along x for a vertical axis, y otherwise.
	axisDir := "y"
	if vertical {
		axisDir = "x"
	}
	group.append("line").attr("stroke", "currentColor").
		attr(axisDir+"2", jsNum(k*axisTickSizeInner))
	group.append("text").attr("fill", "currentColor").
		attr(axisDir, jsNum(k*(axisTickSizeInner+axisTickPadding))).
		attr("dy", axisTickBaseline(orient)).
		text(tick.label)
}

// axisTickBaseline is the label's vertical offset for each orientation.
func axisTickBaseline(orient axisOrient) string {
	switch orient {
	case axisTop:
		return "0em"
	case axisBottom:
		return "0.71em"
	default:
		return "0.32em"
	}
}

// axisTextAnchor is how labels align for each orientation.
func axisTextAnchor(orient axisOrient) string {
	switch orient {
	case axisRight:
		return "start"
	case axisLeft:
		return "end"
	default:
		return "middle"
	}
}
