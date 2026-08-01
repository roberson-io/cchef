package ops

import (
	"math"
	"strconv"
)

// The charts CyberChef's Entropy operation draws, built with the same scales,
// axes and curve d3 gives it. See svgbuild.go for the three places cchef's SVG
// deliberately differs from what nodom writes.

// The page the charts are drawn on, and the room left round the plot for its
// axes and labels.
const (
	entropyChartSide    = 500
	entropyMarginTop    = 30
	entropyMarginRight  = 20
	entropyMarginBottom = 50
	entropyMarginLeft   = 30

	entropyTickCount     = 10
	entropyTickSizeOuter = 6
	entropyBinWidth      = 1
)

// The image view's page, which holds one cell per block of the input.
const (
	entropyImageSide = 100
	entropyCellSize  = 1
)

// entropyChartRoot starts a chart, sized to fit whatever it is drawn into.
func entropyChartRoot(viewBox int) *svgEl {
	return newSVGEl("svg").
		attr("width", "100%").
		attr("height", "100%").
		attr("viewBox", "0 0 "+strconv.Itoa(viewBox)+" "+strconv.Itoa(viewBox)).
		attr("xmlns", svgNamespace)
}

// entropyPlotScales are the scales a chart maps its values through.
func entropyPlotScales(yDomain [2]float64, xDomain [2]float64, xLeft float64) (yScale, xScale linearScale) {
	yScale = scaleLinear(yDomain, [2]float64{
		entropyChartSide - entropyMarginBottom, entropyMarginTop,
	})
	xScale = scaleLinear(xDomain, [2]float64{xLeft, entropyChartSide - entropyMarginRight})
	return yScale, xScale
}

// entropyAxes draws the two axes, their labels and the chart's title.
func entropyAxes(svg *svgEl, xScale, yScale linearScale, title, xTitle, yTitle string) {
	renderAxis(
		svg.append("g").attr("transform", "translate(0, "+
			strconv.Itoa(entropyChartSide-entropyMarginBottom)+")"),
		linearAxis(xScale, axisBottom, entropyTickSizeOuter, entropyTickCount),
	)
	renderAxis(
		svg.append("g").attr("transform", "translate("+strconv.Itoa(entropyMarginLeft)+",0)"),
		linearAxis(yScale, axisLeft, entropyTickSizeOuter, entropyTickCount),
	)

	svg.append("text").
		style("text-anchor", "middle").
		attr("transform", "rotate(-90)").
		attr("y", strconv.Itoa(-entropyMarginLeft)).
		attr("x", strconv.Itoa(-entropyChartSide/2)).
		attr("dy", "1em").
		text(yTitle)

	svg.append("text").
		style("text-anchor", "middle").
		attr("transform", "translate("+strconv.Itoa(entropyChartSide/2)+", "+
			strconv.Itoa(entropyChartSide-entropyMarginBottom+40)+")").
		text(xTitle)

	svg.append("text").
		style("text-anchor", "middle").
		attr("transform", "translate("+strconv.Itoa(entropyChartSide/2)+", "+
			strconv.Itoa(entropyMarginTop-10)+")").
		text(title)
}

// entropyCurvePoints maps values onto the plot, ready for the line to be drawn
// through them.
func entropyCurvePoints(values []float64, xScale, yScale linearScale) []d3Point {
	points := make([]d3Point, len(values))
	for i, v := range values {
		points[i] = d3Point{x: xScale.scale(float64(i)), y: yScale.scale(v)}
	}
	return points
}

// entropyBarHistogram draws one bar per byte value, its height the share of the
// input that value makes up.
func entropyBarHistogram(freq []float64) string {
	lowest, highest := entropyExtent(freq)
	yScale, xScale := entropyPlotScales(
		[2]float64{lowest, highest},
		[2]float64{0, float64(len(freq) - 1)},
		entropyMarginLeft-entropyBinWidth,
	)

	svg := entropyChartRoot(entropyChartSide)
	base := yScale.scale(lowest)
	for value, share := range freq {
		top := yScale.scale(share)
		svg.append("rect").
			attr("x", jsNumberString(xScale.scale(float64(value))+entropyBinWidth)).
			attr("y", jsNumberString(top)).
			attr("width", strconv.Itoa(entropyBinWidth)).
			attr("height", jsNumberString(base-top)).
			attr("fill", "blue")
	}

	entropyAxes(svg, xScale, yScale, "", "Byte", "Byte Frequency")
	return svg.render()
}

// entropyLineHistogram draws the same shares as a curve rather than bars.
func entropyLineHistogram(freq []float64) string {
	yScale, xScale := entropyPlotScales(
		[2]float64{0, entropyMax(freq)},
		[2]float64{0, float64(len(freq) - 1)},
		entropyMarginLeft,
	)

	svg := entropyChartRoot(entropyChartSide)
	svg.append("path").
		attr("fill", "none").
		attr("stroke", "steelblue").
		attr("d", d3LineMonotoneX(entropyCurvePoints(freq, xScale, yScale)))

	entropyAxes(svg, xScale, yScale, "", "Byte", "Byte Frequency")
	return svg.render()
}

// entropyScanningCurve draws the entropy of each block of the input in turn, so
// a stretch of encrypted or compressed data stands out from the rest.
func entropyScanningCurve(blocks []float64) string {
	yScale, xScale := entropyPlotScales(
		[2]float64{0, entropyMax(blocks)},
		[2]float64{0, float64(len(blocks))},
		entropyMarginLeft,
	)

	svg := entropyChartRoot(entropyChartSide)
	if len(blocks) > 0 {
		// The line is drawn first and coloured afterwards, which is the order
		// its attributes come out in.
		svg.append("path").
			attr("d", d3LineMonotoneX(entropyCurvePoints(blocks, xScale, yScale))).
			attr("fill", "none").
			attr("stroke", "steelblue")
	}

	entropyAxes(svg, xScale, yScale, "Scanning Entropy", "Block", "Entropy")
	return svg.render()
}

// entropyImage lays the blocks out in rows, each shaded from black at the
// lowest entropy to white at the highest.
func entropyImage(blocks []float64) string {
	shade := interpolateRGB("#000000", "#FFFFFF")
	scale := scaleLinear([2]float64{0, entropyMax(blocks)}, [2]float64{0, 1})

	svg := entropyChartRoot(entropyImageSide)
	for i, block := range blocks {
		svg.append("rect").
			style("fill", shade(scale.scale(block))).
			attr("x", strconv.Itoa(i%entropyImageSide*entropyCellSize)).
			attr("y", strconv.Itoa(i/entropyImageSide*entropyCellSize)).
			attr("width", strconv.Itoa(entropyCellSize)).
			attr("height", strconv.Itoa(entropyCellSize))
	}
	return svg.render()
}

// entropyMax is the largest of the values, which is nothing at all where there
// are none. d3 reports no maximum for an empty set, which carries through the
// scale as a domain of no width.
func entropyMax(values []float64) float64 {
	if len(values) == 0 {
		return math.NaN()
	}
	highest := values[0]
	for _, v := range values[1:] {
		highest = math.Max(highest, v)
	}
	return highest
}

// entropyExtent is the smallest and largest of the values.
func entropyExtent(values []float64) (lowest, highest float64) {
	if len(values) == 0 {
		return math.NaN(), math.NaN()
	}
	lowest, highest = values[0], values[0]
	for _, v := range values[1:] {
		lowest = math.Min(lowest, v)
		highest = math.Max(highest, v)
	}
	return lowest, highest
}
