package ops

import (
	"strings"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/charts"
	"github.com/roberson-io/cchef/internal/jsnum"
	"github.com/roberson-io/cchef/internal/opsutil"
)

func init() {
	core.Register(SeriesChart{})
}

// SeriesChart draws one line graph per named series over a shared x axis.
// Ported from CyberChef's Series chart.
type SeriesChart struct{}

// Meta returns the operation metadata.
func (SeriesChart) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Series chart",
		Module:      "Charts",
		Description: "A time series graph is a line graph of repeated measurements taken over regular time intervals.",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (SeriesChart) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Record delimiter", Type: core.ArgOption, Value: charts.RecordDelimiterOptions},
		{Name: "Field delimiter", Type: core.ArgOption, Value: charts.FieldDelimiterOptions},
		{Name: "X label", Type: core.ArgString, Value: ""},
		{Name: "Point radius", Type: core.ArgNumber, Value: float64(1)},
		{Name: "Series colours", Type: core.ArgString, Value: "mediumseagreen, dodgerblue, tomato"},
	}
}

// Series chart layout, in SVG units.
const (
	seriesSVGWidth       = 500
	interSeriesPadding   = 20
	seriesXAxisHeight    = 50
	seriesLabelWidth     = 50
	seriesHeight         = 100
	seriesWidth          = seriesSVGWidth - seriesLabelWidth - interSeriesPadding
	seriesYAxisTickCount = 5
)

// Run renders the series chart.
func (SeriesChart) Run(in *core.Dish, args []any) (*core.Dish, error) {
	recordDelimiter := charRep(args[0].(string))
	fieldDelimiter := charRep(args[1].(string))
	xLabel := args[2].(string)
	pipRadius := args[3].(float64)

	colours := strings.Split(args[4].(string), ",")
	for i, c := range colours {
		colours[i] = opsutil.EscapeHTML(c)
	}

	xValues, series, err := charts.GetSeriesValues(in.String(), recordDelimiter, fieldDelimiter)
	if err != nil {
		return nil, err
	}

	svg := seriesSVG(xValues, series, xLabel, pipRadius, colours)
	return core.NewDish([]byte(svg.Render()), core.TypeString), nil
}

// seriesSVG builds the chart.
func seriesSVG(xValues []string, series []charts.Series, xLabel string, pipRadius float64, colours []string) *charts.SVGEl {
	allSeriesHeight := float64(len(series)) * (interSeriesPadding + seriesHeight)
	svgHeight := allSeriesHeight + seriesXAxisHeight + interSeriesPadding
	xScale := charts.ScalePoint(xValues, [2]float64{0, seriesWidth})

	svg := charts.NewSVGEl("svg").
		Attr("width", "100%").
		Attr("height", "100%").
		Attr("viewBox", "0 0 "+jsnum.Format(seriesSVGWidth)+" "+jsnum.Format(svgHeight)).
		Attr("xmlns", charts.SVGNamespace)

	xAxisGroup := svg.Append("g").Class("axis axis--x").
		Attr("transform", "translate("+jsnum.Format(seriesLabelWidth)+", "+jsnum.Format(seriesXAxisHeight)+")")
	charts.RenderAxis(xAxisGroup, charts.AxisSpec{
		Orient:        charts.AxisTop,
		Rng:           [2]float64{0, seriesWidth},
		Ticks:         seriesXTicks(xValues, xScale),
		TickSizeOuter: charts.AxisTickSizeInner,
	})

	svg.Append("text").
		Attr("x", jsnum.Format(seriesSVGWidth/2)).
		Attr("y", jsnum.Format(seriesXAxisHeight/2)).
		Style("text-anchor", "middle").
		Text(xLabel)

	tooltips := seriesTooltips(xValues, series)
	chartArea := svg.Append("g").
		Attr("transform", "translate("+jsnum.Format(seriesLabelWidth)+", "+jsnum.Format(seriesXAxisHeight)+")")

	tooltipAreaWidth := seriesWidth / float64(len(xValues))
	hoverAreas := chartArea.Append("g")
	for _, x := range xValues {
		rect := hoverAreas.Append("rect").
			Attr("x", jsnum.Format(xScale.Scale(x)-tooltipAreaWidth/2)).
			Attr("y", "0").
			Attr("width", jsnum.Format(tooltipAreaWidth)).
			Attr("height", jsnum.Format(allSeriesHeight)).
			Attr("stroke", "none").
			Attr("fill", "transparent")
		rect.Append("title").Text(tooltips[x])
	}

	yAxesArea := svg.Append("g").Attr("transform", "translate(0, "+jsnum.Format(seriesXAxisHeight)+")")
	for i, serie := range series {
		renderSeries(chartArea, yAxesArea, seriesRender{
			serie: serie, index: i, xValues: xValues, xScale: xScale,
			colour: colours[i%len(colours)], pipRadius: pipRadius, tooltips: tooltips,
		})
	}
	return svg
}

// seriesRender carries what drawing one series needs.
type seriesRender struct {
	serie     charts.Series
	index     int
	xValues   []string
	xScale    charts.PointScale
	colour    string
	pipRadius float64
	tooltips  map[string]string
}

// renderSeries draws one series' line, points, y axis and name label.
func renderSeries(chartArea, yAxesArea *charts.SVGEl, r seriesRender) {
	offset := seriesHeight*float64(r.index) + interSeriesPadding*float64(r.index+1)
	yScale := charts.ScaleLinear(charts.Extent(seriesData(r.serie, r.xValues)), [2]float64{seriesHeight, 0})

	group := chartArea.Append("g").Attr("transform", "translate(0, "+jsnum.Format(offset)+")")

	var path strings.Builder
	for i, x := range r.xValues {
		if i+1 >= len(r.xValues) {
			break
		}
		next := r.xValues[i+1]
		y, ok := r.serie.Data[x]
		nextY, okNext := r.serie.Data[next]
		if !ok || !okNext {
			continue
		}
		path.WriteString("M " + jsnum.Format(r.xScale.Scale(x)) + " " + jsnum.Format(yScale.Scale(y)) +
			" L " + jsnum.Format(r.xScale.Scale(next)) + " " + jsnum.Format(yScale.Scale(nextY)) + " z ")
	}

	group.Append("path").
		Attr("d", path.String()).
		Attr("fill", "none").
		Attr("stroke", r.colour).
		Attr("stroke-width", "1")

	for _, x := range r.xValues {
		y, ok := r.serie.Data[x]
		if !ok {
			continue
		}
		circle := group.Append("circle").
			Attr("cx", jsnum.Format(r.xScale.Scale(x))).
			Attr("cy", jsnum.Format(yScale.Scale(y))).
			Attr("r", jsnum.Format(r.pipRadius)).
			Attr("fill", r.colour)
		circle.Append("title").Text(r.tooltips[x])
	}

	yAxisGroup := yAxesArea.Append("g").
		Attr("transform", "translate("+jsnum.Format(seriesLabelWidth-interSeriesPadding)+", "+jsnum.Format(offset)+")").
		Class("axis axis--y")
	charts.RenderAxis(yAxisGroup, linearAxis(yScale, charts.AxisLeft, charts.AxisTickSizeInner, seriesYAxisTickCount))

	yAxesArea.Append("g").
		Attr("transform", "translate(0, "+jsnum.Format(seriesHeight/2+offset)+")").
		Append("text").
		Style("text-anchor", "middle").
		Attr("transform", "rotate(-90)").
		Text(r.serie.Name)
}

// seriesData returns a series' values in x order, skipping missing points.
func seriesData(serie charts.Series, xValues []string) []float64 {
	values := make([]float64, 0, len(xValues))
	for _, x := range xValues {
		if v, ok := serie.Data[x]; ok {
			values = append(values, v)
		}
	}
	return values
}

// seriesXTicks labels the first, middle and last x values only, as CyberChef
// does to keep a dense axis readable.
func seriesXTicks(xValues []string, xScale charts.PointScale) []charts.AxisTick {
	wanted := map[int]bool{
		0:                                      true,
		jsnum.Round(float64(len(xValues)) / 2): true,
		len(xValues) - 1:                       true,
	}
	var ticks []charts.AxisTick
	for i, x := range xValues {
		if wanted[i] {
			ticks = append(ticks, charts.AxisTick{Position: xScale.Scale(x), Label: x})
		}
	}
	return ticks
}

// seriesTooltips builds the hover text for each x value: the value of every
// series that has one there.
func seriesTooltips(xValues []string, series []charts.Series) map[string]string {
	tooltips := make(map[string]string, len(xValues))
	for _, x := range xValues {
		var lines []string
		for _, serie := range series {
			if v, ok := serie.Data[x]; ok {
				lines = append(lines, serie.Name+": "+jsnum.Format(v))
			}
		}
		tooltips[x] = x + "\n--\n" + strings.Join(lines, "\n") + "\n"
	}
	return tooltips
}
