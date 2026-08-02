package ops

import (
	"errors"
	"math"
	"strconv"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/charts"
	"github.com/roberson-io/cchef/internal/jsnum"
)

func init() {
	core.Register(HeatmapChart{})
}

// HeatmapChart bins two-variable data into a grid and shades each cell by how
// many points fall in it. Ported from CyberChef's Heatmap chart.
type HeatmapChart struct{}

// Meta returns the operation metadata.
func (HeatmapChart) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Heatmap chart",
		Module:      "Charts",
		Description: "A heatmap is a graphical representation of data where the individual values contained in a matrix are represented as colours.",
		InfoURL:     "https://wikipedia.org/wiki/Heat_map",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (HeatmapChart) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Record delimiter", Type: core.ArgOption, Value: charts.RecordDelimiterOptions},
		{Name: "Field delimiter", Type: core.ArgOption, Value: charts.FieldDelimiterOptions},
		{Name: "Number of vertical bins", Type: core.ArgNumber, Integer: true, Value: float64(25)},
		{Name: "Number of horizontal bins", Type: core.ArgNumber, Integer: true, Value: float64(25)},
		{Name: "Use column headers as labels", Type: core.ArgBoolean, Value: true},
		{Name: "X label", Type: core.ArgString, Value: ""},
		{Name: "Y label", Type: core.ArgString, Value: ""},
		{Name: "Draw bin edges", Type: core.ArgBoolean, Value: false},
		{Name: "Min colour value", Type: core.ArgString, Value: charts.ColourMin},
		{Name: "Max colour value", Type: core.ArgString, Value: charts.ColourMax},
	}
}

// heatmapEpsilon nudges the upper bound so a point exactly at the maximum falls
// in the last bin rather than one past it.
const heatmapEpsilon = 0.000000001

// Run renders the heatmap chart.
func (HeatmapChart) Run(in *core.Dish, args []any) (*core.Dish, error) {
	recordDelimiter := charRep(args[0].(string))
	fieldDelimiter := charRep(args[1].(string))
	vBins, hBins := int(args[2].(float64)), int(args[3].(float64))
	headingsIncluded := args[4].(bool)
	xLabel, yLabel := args[5].(string), args[6].(string)
	drawEdges := args[7].(bool)
	minColour, maxColour := args[8].(string), args[9].(string)

	if vBins <= 0 {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return nil, errors.New("Number of vertical bins must be greater than 0")
	}
	if hBins <= 0 {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return nil, errors.New("Number of horizontal bins must be greater than 0")
	}

	headings, values, err := charts.GetScatterValues(in.String(), recordDelimiter, fieldDelimiter, headingsIncluded)
	if err != nil {
		return nil, err
	}
	if headings != nil {
		xLabel, yLabel = headings[0], headings[1]
	}

	bins, err := heatmapPacking(values, vBins, hBins)
	if err != nil {
		return nil, err
	}

	svg := heatmapSVG(values, bins, heatmapOptions{
		vBins: vBins, hBins: hBins,
		xLabel: xLabel, yLabel: yLabel, drawEdges: drawEdges,
		minColour: minColour, maxColour: maxColour,
	})
	return core.NewDish([]byte(svg.Render()), core.TypeString), nil
}

// heatmapOptions are the presentation choices for the heatmap chart.
type heatmapOptions struct {
	vBins, hBins         int
	xLabel, yLabel       string
	drawEdges            bool
	minColour, maxColour string
}

// heatmapBin is one grid cell: its position and how many points fall in it.
type heatmapBin struct {
	x, y  int
	count int
}

// heatmapPacking assigns each point to a grid cell.
func heatmapPacking(values []charts.ScatterPoint, vBins, hBins int) ([][]heatmapBin, error) {
	xs := make([]float64, len(values))
	ys := make([]float64, len(values))
	for i, p := range values {
		xs[i], ys[i] = p.X, p.Y
	}
	xBounds, yBounds := charts.Extent(xs), charts.Extent(ys)
	if xBounds[0] == xBounds[1] {
		//nolint:staticcheck,revive // CyberChef's verbatim error text
		return nil, errors.New("Cannot pack points. There is no difference between the minimum and maximum X coordinate.")
	}
	if yBounds[0] == yBounds[1] {
		//nolint:staticcheck,revive // CyberChef's verbatim error text
		return nil, errors.New("Cannot pack points. There is no difference between the minimum and maximum Y coordinate.")
	}

	bins := make([][]heatmapBin, vBins)
	for y := range bins {
		bins[y] = make([]heatmapBin, hBins)
		for x := range bins[y] {
			bins[y][x] = heatmapBin{x: x, y: y}
		}
	}
	for _, p := range values {
		fractionOfY := (p.Y - yBounds[0]) / ((yBounds[1] + heatmapEpsilon) - yBounds[0])
		fractionOfX := (p.X - xBounds[0]) / ((xBounds[1] + heatmapEpsilon) - xBounds[0])
		y := int(math.Floor(float64(vBins) * fractionOfY))
		x := int(math.Floor(float64(hBins) * fractionOfX))
		bins[y][x].count++
	}
	return bins, nil
}

// heatmapSVG builds the chart.
func heatmapSVG(values []charts.ScatterPoint, bins [][]heatmapBin, opt heatmapOptions) *charts.SVGEl {
	width := chartDimension - scatterMargin.left - scatterMargin.right
	height := chartDimension - scatterMargin.top - scatterMargin.bottom
	binWidth := width / float64(opt.hBins)
	binHeight := height / float64(opt.vBins)

	xs := make([]float64, len(values))
	ys := make([]float64, len(values))
	for i, p := range values {
		xs[i], ys[i] = p.X, p.Y
	}
	xScale := charts.ScaleLinear(charts.Extent(xs), [2]float64{0, width})
	yScale := charts.ScaleLinear(charts.Extent(ys), [2]float64{height, 0})

	maxCount := 0
	for _, row := range bins {
		for _, bin := range row {
			maxCount = max(maxCount, bin.count)
		}
	}
	colour := sequentialColour(opt.minColour, opt.maxColour, float64(maxCount))

	svg := charts.NewSVGEl("svg").
		Attr("width", "100%").
		Attr("height", "100%").
		Attr("viewBox", "0 0 "+jsnum.Format(chartDimension)+" "+jsnum.Format(chartDimension)).
		Attr("xmlns", charts.SVGNamespace)

	marginedSpace := svg.Append("g").Attr("transform",
		"translate("+jsnum.Format(scatterMargin.left)+","+jsnum.Format(scatterMargin.top)+")")
	marginedSpace.Append("clipPath").Attr("id", "clip").
		Append("rect").Attr("width", jsnum.Format(width)).Attr("height", jsnum.Format(height))

	binGroups := marginedSpace.Append("g").Class("bins").Attr("clip-path", "url(#clip)")
	stroke, strokeWidth := "none", "none"
	if opt.drawEdges {
		stroke, strokeWidth = "rgba(0, 0, 0, 0.5)", "0.5"
	}
	for _, row := range bins {
		rowGroup := binGroups.Append("g")
		for _, bin := range row {
			rect := rowGroup.Append("rect").
				Attr("x", jsnum.Format(binWidth*float64(bin.x))).
				Attr("y", jsnum.Format(height-binHeight*float64(bin.y+1))).
				Attr("width", jsnum.Format(binWidth)).
				Attr("height", jsnum.Format(binHeight)).
				Attr("fill", colour(float64(bin.count))).
				Attr("stroke", stroke).
				Attr("stroke-width", strokeWidth)
			percentage := 100.0 * float64(bin.count) / float64(len(values))
			rect.Append("title").Text("Count: " + strconv.Itoa(bin.count) + "\nPercentage: " +
				strconv.FormatFloat(percentage, 'f', 2, 64) + "%\n")
		}
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

// sequentialColour maps a value in [0, maxValue] onto the colour ramp, as
// d3.scaleSequential over an interpolateLab ramp does.
func sequentialColour(minColour, maxColour string, maxValue float64) func(float64) string {
	interp := charts.InterpolateLab(minColour, maxColour)
	return func(v float64) string {
		if maxValue == 0 {
			return interp(0)
		}
		return interp(v / maxValue)
	}
}
