package ops

import (
	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/charts"
	"github.com/roberson-io/cchef/internal/jsnum"
	"github.com/roberson-io/cchef/internal/opsutil"
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
		{Name: "Record delimiter", Type: core.ArgOption, Value: charts.RecordDelimiterOptions},
		{Name: "Field delimiter", Type: core.ArgOption, Value: charts.FieldDelimiterOptions},
		{Name: "Use column headers as labels", Type: core.ArgBoolean, Value: true},
		{Name: "X label", Type: core.ArgString, Value: ""},
		{Name: "Y label", Type: core.ArgString, Value: ""},
		{Name: "Colour", Type: core.ArgString, Value: charts.ColourMax},
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
	fillColour := opsutil.EscapeHTML(args[5].(string))
	radius := args[6].(float64)
	colourInInput := args[7].(bool)

	read := charts.GetScatterValues
	if colourInInput {
		read = charts.GetScatterValuesWithColour
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
	return core.NewDish([]byte(svg.Render()), core.TypeString), nil
}

// scatterOptions are the presentation choices for the scatter chart.
type scatterOptions struct {
	xLabel, yLabel string
	fill           string
	radius         float64
	colourInInput  bool
}

// scatterSVG builds the chart.
func scatterSVG(values []charts.ScatterPoint, opt scatterOptions) *charts.SVGEl {
	width := chartDimension - scatterMargin.left - scatterMargin.right
	height := chartDimension - scatterMargin.top - scatterMargin.bottom

	xs := make([]float64, len(values))
	ys := make([]float64, len(values))
	for i, p := range values {
		xs[i], ys[i] = p.X, p.Y
	}
	xScale := charts.ScaleLinear(paddedExtent(charts.Extent(xs)), [2]float64{0, width})
	yScale := charts.ScaleLinear(paddedExtent(charts.Extent(ys)), [2]float64{height, 0})

	svg := charts.NewSVGEl("svg").
		Attr("width", "100%").
		Attr("height", "100%").
		Attr("viewBox", "0 0 "+jsnum.Format(chartDimension)+" "+jsnum.Format(chartDimension)).
		Attr("xmlns", charts.SVGNamespace)

	marginedSpace := svg.Append("g").Attr("transform",
		"translate("+jsnum.Format(scatterMargin.left)+","+jsnum.Format(scatterMargin.top)+")")

	marginedSpace.Append("clipPath").Attr("id", "clip").
		Append("rect").Attr("width", jsnum.Format(width)).Attr("height", jsnum.Format(height))

	points := marginedSpace.Append("g").Class("points").Attr("clip-path", "url(#clip)")
	for _, p := range values {
		fill := opt.fill
		if opt.colourInInput {
			fill = p.Colour
		}
		circle := points.Append("circle").
			Attr("cx", jsnum.Format(xScale.Scale(p.X))).
			Attr("cy", jsnum.Format(yScale.Scale(p.Y))).
			Attr("r", jsnum.Format(opt.radius)).
			Attr("fill", fill).
			Attr("stroke", "rgba(0, 0, 0, 0.5)").
			Attr("stroke-width", "0.5")
		circle.Append("title").Text("X: " + jsnum.Format(p.X) + "\nY: " + jsnum.Format(p.Y) + "\n")
	}

	yAxisGroup := marginedSpace.Append("g").Class("axis axis--y")
	charts.RenderAxis(yAxisGroup, linearAxis(yScale, charts.AxisLeft, -width, defaultTickCount))

	svg.Append("text").
		Attr("transform", "rotate(-90)").
		Attr("y", jsnum.Format(-scatterMargin.left)).
		Attr("x", jsnum.Format(-(height/2))).
		Attr("dy", "1em").
		Style("text-anchor", "middle").
		Text(opt.yLabel)

	xAxisGroup := marginedSpace.Append("g").Class("axis axis--x").
		Attr("transform", "translate(0,"+jsnum.Format(height)+")")
	charts.RenderAxis(xAxisGroup, linearAxis(xScale, charts.AxisBottom, -height, defaultTickCount))

	svg.Append("text").
		Attr("x", jsnum.Format(width/2)).
		Attr("y", jsnum.Format(chartDimension)).
		Style("text-anchor", "middle").
		Text(opt.xLabel)

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
func linearAxis(s charts.LinearScale, orient charts.AxisOrient, tickSizeOuter float64, count int) charts.AxisSpec {
	format := charts.TickFormat(s.Domain[0], s.Domain[1], count)
	values := s.Ticks(count)
	ticks := make([]charts.AxisTick, len(values))
	for i, v := range values {
		ticks[i] = charts.AxisTick{Position: s.Scale(v), Label: format(v)}
	}
	return charts.AxisSpec{Orient: orient, Rng: s.Rng, Ticks: ticks, TickSizeOuter: tickSizeOuter}
}
