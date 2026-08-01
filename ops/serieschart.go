package ops

import (
	"strings"

	"github.com/roberson-io/cchef/core"
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
		{Name: "Record delimiter", Type: core.ArgOption, Value: recordDelimiterOptions},
		{Name: "Field delimiter", Type: core.ArgOption, Value: fieldDelimiterOptions},
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
		colours[i] = escapeHTML(c)
	}

	xValues, series, err := getSeriesValues(in.String(), recordDelimiter, fieldDelimiter)
	if err != nil {
		return nil, err
	}

	svg := seriesSVG(xValues, series, xLabel, pipRadius, colours)
	return core.NewDish([]byte(svg.render()), core.TypeString), nil
}

// seriesSVG builds the chart.
func seriesSVG(xValues []string, series []chartSeries, xLabel string, pipRadius float64, colours []string) *svgEl {
	allSeriesHeight := float64(len(series)) * (interSeriesPadding + seriesHeight)
	svgHeight := allSeriesHeight + seriesXAxisHeight + interSeriesPadding
	xScale := scalePoint(xValues, [2]float64{0, seriesWidth})

	svg := newSVGEl("svg").
		attr("width", "100%").
		attr("height", "100%").
		attr("viewBox", "0 0 "+jsNum(seriesSVGWidth)+" "+jsNum(svgHeight)).
		attr("xmlns", svgNamespace)

	xAxisGroup := svg.append("g").class("axis axis--x").
		attr("transform", "translate("+jsNum(seriesLabelWidth)+", "+jsNum(seriesXAxisHeight)+")")
	renderAxis(xAxisGroup, axisSpec{
		orient:        axisTop,
		rng:           [2]float64{0, seriesWidth},
		ticks:         seriesXTicks(xValues, xScale),
		tickSizeOuter: axisTickSizeInner,
	})

	svg.append("text").
		attr("x", jsNum(seriesSVGWidth/2)).
		attr("y", jsNum(seriesXAxisHeight/2)).
		style("text-anchor", "middle").
		text(xLabel)

	tooltips := seriesTooltips(xValues, series)
	chartArea := svg.append("g").
		attr("transform", "translate("+jsNum(seriesLabelWidth)+", "+jsNum(seriesXAxisHeight)+")")

	tooltipAreaWidth := seriesWidth / float64(len(xValues))
	hoverAreas := chartArea.append("g")
	for _, x := range xValues {
		rect := hoverAreas.append("rect").
			attr("x", jsNum(xScale.scale(x)-tooltipAreaWidth/2)).
			attr("y", "0").
			attr("width", jsNum(tooltipAreaWidth)).
			attr("height", jsNum(allSeriesHeight)).
			attr("stroke", "none").
			attr("fill", "transparent")
		rect.append("title").text(tooltips[x])
	}

	yAxesArea := svg.append("g").attr("transform", "translate(0, "+jsNum(seriesXAxisHeight)+")")
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
	serie     chartSeries
	index     int
	xValues   []string
	xScale    pointScale
	colour    string
	pipRadius float64
	tooltips  map[string]string
}

// renderSeries draws one series' line, points, y axis and name label.
func renderSeries(chartArea, yAxesArea *svgEl, r seriesRender) {
	offset := seriesHeight*float64(r.index) + interSeriesPadding*float64(r.index+1)
	yScale := scaleLinear(chartExtent(seriesData(r.serie, r.xValues)), [2]float64{seriesHeight, 0})

	group := chartArea.append("g").attr("transform", "translate(0, "+jsNum(offset)+")")

	var path strings.Builder
	for i, x := range r.xValues {
		if i+1 >= len(r.xValues) {
			break
		}
		next := r.xValues[i+1]
		y, ok := r.serie.data[x]
		nextY, okNext := r.serie.data[next]
		if !ok || !okNext {
			continue
		}
		path.WriteString("M " + jsNum(r.xScale.scale(x)) + " " + jsNum(yScale.scale(y)) +
			" L " + jsNum(r.xScale.scale(next)) + " " + jsNum(yScale.scale(nextY)) + " z ")
	}

	group.append("path").
		attr("d", path.String()).
		attr("fill", "none").
		attr("stroke", r.colour).
		attr("stroke-width", "1")

	for _, x := range r.xValues {
		y, ok := r.serie.data[x]
		if !ok {
			continue
		}
		circle := group.append("circle").
			attr("cx", jsNum(r.xScale.scale(x))).
			attr("cy", jsNum(yScale.scale(y))).
			attr("r", jsNum(r.pipRadius)).
			attr("fill", r.colour)
		circle.append("title").text(r.tooltips[x])
	}

	yAxisGroup := yAxesArea.append("g").
		attr("transform", "translate("+jsNum(seriesLabelWidth-interSeriesPadding)+", "+jsNum(offset)+")").
		class("axis axis--y")
	renderAxis(yAxisGroup, linearAxis(yScale, axisLeft, axisTickSizeInner, seriesYAxisTickCount))

	yAxesArea.append("g").
		attr("transform", "translate(0, "+jsNum(seriesHeight/2+offset)+")").
		append("text").
		style("text-anchor", "middle").
		attr("transform", "rotate(-90)").
		text(r.serie.name)
}

// seriesData returns a series' values in x order, skipping missing points.
func seriesData(serie chartSeries, xValues []string) []float64 {
	values := make([]float64, 0, len(xValues))
	for _, x := range xValues {
		if v, ok := serie.data[x]; ok {
			values = append(values, v)
		}
	}
	return values
}

// seriesXTicks labels the first, middle and last x values only, as CyberChef
// does to keep a dense axis readable.
func seriesXTicks(xValues []string, xScale pointScale) []axisTick {
	wanted := map[int]bool{
		0:                                  true,
		jsRound(float64(len(xValues)) / 2): true,
		len(xValues) - 1:                   true,
	}
	var ticks []axisTick
	for i, x := range xValues {
		if wanted[i] {
			ticks = append(ticks, axisTick{position: xScale.scale(x), label: x})
		}
	}
	return ticks
}

// seriesTooltips builds the hover text for each x value: the value of every
// series that has one there.
func seriesTooltips(xValues []string, series []chartSeries) map[string]string {
	tooltips := make(map[string]string, len(xValues))
	for _, x := range xValues {
		var lines []string
		for _, serie := range series {
			if v, ok := serie.data[x]; ok {
				lines = append(lines, serie.name+": "+jsNum(v))
			}
		}
		tooltips[x] = x + "\n--\n" + strings.Join(lines, "\n") + "\n"
	}
	return tooltips
}
