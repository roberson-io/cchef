package ops

import (
	"math"
	"strconv"

	"github.com/roberson-io/cchef/internal/charts"
	"github.com/roberson-io/cchef/internal/jsnum"
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
func entropyChartRoot(viewBox int) *charts.SVGEl {
	return charts.NewSVGEl("svg").
		Attr("width", "100%").
		Attr("height", "100%").
		Attr("viewBox", "0 0 "+strconv.Itoa(viewBox)+" "+strconv.Itoa(viewBox)).
		Attr("xmlns", charts.SVGNamespace)
}

// entropyPlotScales are the scales a chart maps its values through.
func entropyPlotScales(yDomain [2]float64, xDomain [2]float64, xLeft float64) (yScale, xScale charts.LinearScale) {
	yScale = charts.ScaleLinear(yDomain, [2]float64{
		entropyChartSide - entropyMarginBottom, entropyMarginTop,
	})
	xScale = charts.ScaleLinear(xDomain, [2]float64{xLeft, entropyChartSide - entropyMarginRight})
	return yScale, xScale
}

// entropyAxes draws the two axes, their labels and the chart's title.
func entropyAxes(svg *charts.SVGEl, xScale, yScale charts.LinearScale, title, xTitle, yTitle string) {
	charts.RenderAxis(
		svg.Append("g").Attr("transform", "translate(0, "+
			strconv.Itoa(entropyChartSide-entropyMarginBottom)+")"),
		linearAxis(xScale, charts.AxisBottom, entropyTickSizeOuter, entropyTickCount),
	)
	charts.RenderAxis(
		svg.Append("g").Attr("transform", "translate("+strconv.Itoa(entropyMarginLeft)+",0)"),
		linearAxis(yScale, charts.AxisLeft, entropyTickSizeOuter, entropyTickCount),
	)

	svg.Append("text").
		Style("text-anchor", "middle").
		Attr("transform", "rotate(-90)").
		Attr("y", strconv.Itoa(-entropyMarginLeft)).
		Attr("x", strconv.Itoa(-entropyChartSide/2)).
		Attr("dy", "1em").
		Text(yTitle)

	svg.Append("text").
		Style("text-anchor", "middle").
		Attr("transform", "translate("+strconv.Itoa(entropyChartSide/2)+", "+
			strconv.Itoa(entropyChartSide-entropyMarginBottom+40)+")").
		Text(xTitle)

	svg.Append("text").
		Style("text-anchor", "middle").
		Attr("transform", "translate("+strconv.Itoa(entropyChartSide/2)+", "+
			strconv.Itoa(entropyMarginTop-10)+")").
		Text(title)
}

// entropyCurvePoints maps values onto the plot, ready for the line to be drawn
// through them.
func entropyCurvePoints(values []float64, xScale, yScale charts.LinearScale) []charts.Point {
	points := make([]charts.Point, len(values))
	for i, v := range values {
		points[i] = charts.Point{X: xScale.Scale(float64(i)), Y: yScale.Scale(v)}
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
	base := yScale.Scale(lowest)
	for value, share := range freq {
		top := yScale.Scale(share)
		svg.Append("rect").
			Attr("x", jsnum.String(xScale.Scale(float64(value))+entropyBinWidth)).
			Attr("y", jsnum.String(top)).
			Attr("width", strconv.Itoa(entropyBinWidth)).
			Attr("height", jsnum.String(base-top)).
			Attr("fill", "blue")
	}

	entropyAxes(svg, xScale, yScale, "", "Byte", "Byte Frequency")
	return svg.Render()
}

// entropyLineHistogram draws the same shares as a curve rather than bars.
func entropyLineHistogram(freq []float64) string {
	yScale, xScale := entropyPlotScales(
		[2]float64{0, entropyMax(freq)},
		[2]float64{0, float64(len(freq) - 1)},
		entropyMarginLeft,
	)

	svg := entropyChartRoot(entropyChartSide)
	svg.Append("path").
		Attr("fill", "none").
		Attr("stroke", "steelblue").
		Attr("d", charts.LineMonotoneX(entropyCurvePoints(freq, xScale, yScale)))

	entropyAxes(svg, xScale, yScale, "", "Byte", "Byte Frequency")
	return svg.Render()
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
		svg.Append("path").
			Attr("d", charts.LineMonotoneX(entropyCurvePoints(blocks, xScale, yScale))).
			Attr("fill", "none").
			Attr("stroke", "steelblue")
	}

	entropyAxes(svg, xScale, yScale, "Scanning Entropy", "Block", "Entropy")
	return svg.Render()
}

// entropyImage lays the blocks out in rows, each shaded from black at the
// lowest entropy to white at the highest.
func entropyImage(blocks []float64) string {
	shade := charts.InterpolateRGB("#000000", "#FFFFFF")
	scale := charts.ScaleLinear([2]float64{0, entropyMax(blocks)}, [2]float64{0, 1})

	svg := entropyChartRoot(entropyImageSide)
	for i, block := range blocks {
		svg.Append("rect").
			Style("fill", shade(scale.Scale(block))).
			Attr("x", strconv.Itoa(i%entropyImageSide*entropyCellSize)).
			Attr("y", strconv.Itoa(i/entropyImageSide*entropyCellSize)).
			Attr("width", strconv.Itoa(entropyCellSize)).
			Attr("height", strconv.Itoa(entropyCellSize))
	}
	return svg.Render()
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
