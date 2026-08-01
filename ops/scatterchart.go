package ops

import (
	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(ScatterChart{})
}

// ScatterChart plots two-variable data as points. Ported from CyberChef's
// Scatter chart, which builds the SVG with d3 and nodom; cchef reproduces d3's
// scales and axis markup directly. See svgbuild.go for the places the
// serialisation deliberately differs.
type ScatterChart struct{}

// Meta returns the operation metadata.
func (ScatterChart) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Scatter chart",
		Module:      "Charts",
		Description: "Plots two-variable data as single points on a graph.",
		InfoURL:     "https://wikipedia.org/wiki/Scatter_plot",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ScatterChart) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Record delimiter", Type: core.ArgOption, Value: recordDelimiterOptions},
		{Name: "Field delimiter", Type: core.ArgOption, Value: fieldDelimiterOptions},
		{Name: "Use column headers as labels", Type: core.ArgBoolean, Value: true},
		{Name: "X label", Type: core.ArgString, Value: ""},
		{Name: "Y label", Type: core.ArgString, Value: ""},
		{Name: "Colour", Type: core.ArgString, Value: colourMax},
		{Name: "Point radius", Type: core.ArgNumber, Value: float64(10)},
		{Name: "Use colour from third column", Type: core.ArgBoolean, Value: false},
	}
}

// chartDimension is the square viewBox the scatter and hex density charts use.
const chartDimension = 500

// scatterMargin is the space reserved around the plotting area.
var scatterMargin = chartMargin{top: 10, right: 0, bottom: 40, left: 30}

// chartMargin is the space around a chart's plotting area.
type chartMargin struct{ top, right, bottom, left float64 }

// axisPadFraction is how far the axes extend beyond the data at each end.
const axisPadFraction = 0.1

// Run renders the scatter chart.
func (ScatterChart) Run(in *core.Dish, args []any) (*core.Dish, error) {
	recordDelimiter := charRep(args[0].(string))
	fieldDelimiter := charRep(args[1].(string))
	headingsIncluded := args[2].(bool)
	xLabel, yLabel := args[3].(string), args[4].(string)
	fillColour := escapeHTML(args[5].(string))
	radius := args[6].(float64)
	colourInInput := args[7].(bool)

	read := getScatterValues
	if colourInInput {
		read = getScatterValuesWithColour
	}
	headings, values, err := read(in.String(), recordDelimiter, fieldDelimiter, headingsIncluded)
	if err != nil {
		return nil, err
	}
	if headings != nil {
		xLabel, yLabel = headings[0], headings[1]
	}

	svg := scatterSVG(values, scatterOptions{
		xLabel: xLabel, yLabel: yLabel,
		fill: fillColour, radius: radius, colourInInput: colourInInput,
	})
	return core.NewDish([]byte(svg.render()), core.TypeString), nil
}

// scatterOptions are the presentation choices for the scatter chart.
type scatterOptions struct {
	xLabel, yLabel string
	fill           string
	radius         float64
	colourInInput  bool
}

// scatterSVG builds the chart.
func scatterSVG(values []scatterPoint, opt scatterOptions) *svgEl {
	width := chartDimension - scatterMargin.left - scatterMargin.right
	height := chartDimension - scatterMargin.top - scatterMargin.bottom

	xs := make([]float64, len(values))
	ys := make([]float64, len(values))
	for i, p := range values {
		xs[i], ys[i] = p.x, p.y
	}
	xScale := scaleLinear(paddedExtent(chartExtent(xs)), [2]float64{0, width})
	yScale := scaleLinear(paddedExtent(chartExtent(ys)), [2]float64{height, 0})

	svg := newSVGEl("svg").
		attr("width", "100%").
		attr("height", "100%").
		attr("viewBox", "0 0 "+jsNum(chartDimension)+" "+jsNum(chartDimension)).
		attr("xmlns", svgNamespace)

	marginedSpace := svg.append("g").attr("transform",
		"translate("+jsNum(scatterMargin.left)+","+jsNum(scatterMargin.top)+")")

	marginedSpace.append("clipPath").attr("id", "clip").
		append("rect").attr("width", jsNum(width)).attr("height", jsNum(height))

	points := marginedSpace.append("g").class("points").attr("clip-path", "url(#clip)")
	for _, p := range values {
		fill := opt.fill
		if opt.colourInInput {
			fill = p.colour
		}
		circle := points.append("circle").
			attr("cx", jsNum(xScale.scale(p.x))).
			attr("cy", jsNum(yScale.scale(p.y))).
			attr("r", jsNum(opt.radius)).
			attr("fill", fill).
			attr("stroke", "rgba(0, 0, 0, 0.5)").
			attr("stroke-width", "0.5")
		circle.append("title").text("X: " + jsNum(p.x) + "\nY: " + jsNum(p.y) + "\n")
	}

	yAxisGroup := marginedSpace.append("g").class("axis axis--y")
	renderAxis(yAxisGroup, linearAxis(yScale, axisLeft, -width, defaultTickCount))

	svg.append("text").
		attr("transform", "rotate(-90)").
		attr("y", jsNum(-scatterMargin.left)).
		attr("x", jsNum(-(height/2))).
		attr("dy", "1em").
		style("text-anchor", "middle").
		text(opt.yLabel)

	xAxisGroup := marginedSpace.append("g").class("axis axis--x").
		attr("transform", "translate(0,"+jsNum(height)+")")
	renderAxis(xAxisGroup, linearAxis(xScale, axisBottom, -height, defaultTickCount))

	svg.append("text").
		attr("x", jsNum(width/2)).
		attr("y", jsNum(chartDimension)).
		style("text-anchor", "middle").
		text(opt.xLabel)

	return svg
}

// defaultTickCount is d3's default number of ticks for an axis.
const defaultTickCount = 10

// paddedExtent widens an extent by a tenth of its span at each end, as the
// scatter and hex density charts do so points are not drawn on the axes.
func paddedExtent(e [2]float64) [2]float64 {
	delta := e[1] - e[0]
	return [2]float64{e[0] - axisPadFraction*delta, e[1] + axisPadFraction*delta}
}

// linearAxis resolves a linear scale into the ticks an axis renders.
func linearAxis(s linearScale, orient axisOrient, tickSizeOuter float64, count int) axisSpec {
	format := tickFormat(s.domain[0], s.domain[1], count)
	values := s.ticks(count)
	ticks := make([]axisTick, len(values))
	for i, v := range values {
		ticks[i] = axisTick{position: s.scale(v), label: format(v)}
	}
	return axisSpec{orient: orient, rng: s.rng, ticks: ticks, tickSizeOuter: tickSizeOuter}
}
