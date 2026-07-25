package ops

import (
	"errors"
	"math"
	"strconv"

	"github.com/roberson-io/cchef/internal/core"
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
		{Name: "Record delimiter", Type: core.ArgOption, Value: recordDelimiterOptions},
		{Name: "Field delimiter", Type: core.ArgOption, Value: fieldDelimiterOptions},
		{Name: "Number of vertical bins", Type: core.ArgNumber, Value: float64(25)},
		{Name: "Number of horizontal bins", Type: core.ArgNumber, Value: float64(25)},
		{Name: "Use column headers as labels", Type: core.ArgBoolean, Value: true},
		{Name: "X label", Type: core.ArgString, Value: ""},
		{Name: "Y label", Type: core.ArgString, Value: ""},
		{Name: "Draw bin edges", Type: core.ArgBoolean, Value: false},
		{Name: "Min colour value", Type: core.ArgString, Value: colourMin},
		{Name: "Max colour value", Type: core.ArgString, Value: colourMax},
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

	headings, values, err := getScatterValues(in.String(), recordDelimiter, fieldDelimiter, headingsIncluded)
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
	return core.NewDish([]byte(svg.render()), core.TypeString), nil
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
func heatmapPacking(values []scatterPoint, vBins, hBins int) ([][]heatmapBin, error) {
	xs := make([]float64, len(values))
	ys := make([]float64, len(values))
	for i, p := range values {
		xs[i], ys[i] = p.x, p.y
	}
	xBounds, yBounds := chartExtent(xs), chartExtent(ys)
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
		fractionOfY := (p.y - yBounds[0]) / ((yBounds[1] + heatmapEpsilon) - yBounds[0])
		fractionOfX := (p.x - xBounds[0]) / ((xBounds[1] + heatmapEpsilon) - xBounds[0])
		y := int(math.Floor(float64(vBins) * fractionOfY))
		x := int(math.Floor(float64(hBins) * fractionOfX))
		bins[y][x].count++
	}
	return bins, nil
}

// heatmapSVG builds the chart.
func heatmapSVG(values []scatterPoint, bins [][]heatmapBin, opt heatmapOptions) *svgEl {
	width := chartDimension - scatterMargin.left - scatterMargin.right
	height := chartDimension - scatterMargin.top - scatterMargin.bottom
	binWidth := width / float64(opt.hBins)
	binHeight := height / float64(opt.vBins)

	xs := make([]float64, len(values))
	ys := make([]float64, len(values))
	for i, p := range values {
		xs[i], ys[i] = p.x, p.y
	}
	xScale := scaleLinear(chartExtent(xs), [2]float64{0, width})
	yScale := scaleLinear(chartExtent(ys), [2]float64{height, 0})

	maxCount := 0
	for _, row := range bins {
		for _, bin := range row {
			maxCount = max(maxCount, bin.count)
		}
	}
	colour := sequentialColour(opt.minColour, opt.maxColour, float64(maxCount))

	svg := newSVGEl("svg").
		attr("width", "100%").
		attr("height", "100%").
		attr("viewBox", "0 0 "+jsNum(chartDimension)+" "+jsNum(chartDimension)).
		attr("xmlns", svgNamespace)

	marginedSpace := svg.append("g").attr("transform",
		"translate("+jsNum(scatterMargin.left)+","+jsNum(scatterMargin.top)+")")
	marginedSpace.append("clipPath").attr("id", "clip").
		append("rect").attr("width", jsNum(width)).attr("height", jsNum(height))

	binGroups := marginedSpace.append("g").class("bins").attr("clip-path", "url(#clip)")
	stroke, strokeWidth := "none", "none"
	if opt.drawEdges {
		stroke, strokeWidth = "rgba(0, 0, 0, 0.5)", "0.5"
	}
	for _, row := range bins {
		rowGroup := binGroups.append("g")
		for _, bin := range row {
			rect := rowGroup.append("rect").
				attr("x", jsNum(binWidth*float64(bin.x))).
				attr("y", jsNum(height-binHeight*float64(bin.y+1))).
				attr("width", jsNum(binWidth)).
				attr("height", jsNum(binHeight)).
				attr("fill", colour(float64(bin.count))).
				attr("stroke", stroke).
				attr("stroke-width", strokeWidth)
			percentage := 100.0 * float64(bin.count) / float64(len(values))
			rect.append("title").text("Count: " + strconv.Itoa(bin.count) + "\nPercentage: " +
				strconv.FormatFloat(percentage, 'f', 2, 64) + "%\n")
		}
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

// sequentialColour maps a value in [0, maxValue] onto the colour ramp, as
// d3.scaleSequential over an interpolateLab ramp does.
func sequentialColour(minColour, maxColour string, maxValue float64) func(float64) string {
	interp := interpolateLab(minColour, maxColour)
	return func(v float64) string {
		if maxValue == 0 {
			return interp(0)
		}
		return interp(v / maxValue)
	}
}
