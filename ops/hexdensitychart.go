package ops

import (
	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/charts"
	"github.com/roberson-io/cchef/internal/jsnum"
)

func init() {
	core.Register(HexDensityChart{})
}

// HexDensityChart groups two-variable data into hexagonal cells and shades each
// by how many points it holds.
type HexDensityChart struct{}

// Meta returns the operation metadata.
func (HexDensityChart) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Hex Density chart",
		Module:      "Charts",
		Description: "Hex density charts are used in a similar way to scatter charts, however rather than plotting individual points, they group points into hexagons to show the density of the distribution.",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (HexDensityChart) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Record delimiter", Type: core.ArgOption, Value: charts.RecordDelimiterOptions},
		{Name: "Field delimiter", Type: core.ArgOption, Value: charts.FieldDelimiterOptions},
		{Name: "Pack radius", Type: core.ArgNumber, Value: float64(25)},
		{Name: "Draw radius", Type: core.ArgNumber, Value: float64(15)},
		{Name: "Use column headers as labels", Type: core.ArgBoolean, Value: true},
		{Name: "X label", Type: core.ArgString, Value: ""},
		{Name: "Y label", Type: core.ArgString, Value: ""},
		{Name: "Draw hexagon edges", Type: core.ArgBoolean, Value: false},
		{Name: "Min colour value", Type: core.ArgString, Value: charts.ColourMin},
		{Name: "Max colour value", Type: core.ArgString, Value: charts.ColourMax},
		{Name: "Draw empty hexagons within data boundaries", Flag: "draw-empty-hexagons", Type: core.ArgBoolean, Value: false},
	}
}

// Run renders the hex density chart.
func (HexDensityChart) Run(in *core.Dish, args []any) (*core.Dish, error) {
	recordDelimiter := charRep(args[0].(string))
	fieldDelimiter := charRep(args[1].(string))
	packRadius, drawRadius := args[2].(float64), args[3].(float64)
	headingsIncluded := args[4].(bool)
	xLabel, yLabel := args[5].(string), args[6].(string)
	drawEdges := args[7].(bool)
	minColour, maxColour := args[8].(string), args[9].(string)
	drawEmpty := args[10].(bool)

	headings, values, err := charts.GetScatterValues(in.String(), recordDelimiter, fieldDelimiter, headingsIncluded)
	if err != nil {
		return nil, err
	}
	if headings != nil {
		xLabel, yLabel = headings[0], headings[1]
	}

	svg := hexDensitySVG(values, hexDensityOptions{
		packRadius: packRadius, drawRadius: drawRadius,
		xLabel: xLabel, yLabel: yLabel, drawEdges: drawEdges,
		minColour: minColour, maxColour: maxColour, drawEmpty: drawEmpty,
	})
	return core.NewDish([]byte(svg.Render()), core.TypeString), nil
}

// hexDensityOptions are the presentation choices for the hex density chart.
type hexDensityOptions struct {
	packRadius, drawRadius float64
	xLabel, yLabel         string
	drawEdges              bool
	minColour, maxColour   string
	drawEmpty              bool
}

// hexDensitySVG builds the chart.
func hexDensitySVG(values []charts.ScatterPoint, opt hexDensityOptions) *charts.SVGEl {
	width := chartDimension - scatterMargin.left - scatterMargin.right
	height := chartDimension - scatterMargin.top - scatterMargin.bottom

	layout := charts.NewHexbin(opt.packRadius)
	bins := layout.Bin(values)

	maxCount := 0
	centresX := make([]float64, len(bins))
	centresY := make([]float64, len(bins))
	for i, bin := range bins {
		maxCount = max(maxCount, len(bin.Points))
		centresX[i], centresY[i] = bin.X, bin.Y
	}

	// The axes are widened past the hexagon centres so whole hexagons fit.
	xExtent := charts.Extent(centresX)
	yExtent := charts.Extent(centresY)
	xExtent[0] -= 2 * opt.packRadius
	xExtent[1] += 3 * opt.packRadius
	yExtent[0] -= 2 * opt.packRadius
	yExtent[1] += 2 * opt.packRadius

	xScale := charts.ScaleLinear(xExtent, [2]float64{0, width})
	yScale := charts.ScaleLinear(yExtent, [2]float64{height, 0})
	colour := sequentialColour(opt.minColour, opt.maxColour, float64(maxCount))
	hexagon := layout.HexagonPath(opt.drawRadius)

	stroke, strokeWidth := "none", "none"
	if opt.drawEdges {
		stroke, strokeWidth = "black", "0.5"
	}

	svg := charts.NewSVGEl("svg").
		Attr("width", "100%").
		Attr("height", "100%").
		Attr("viewBox", "0 0 "+jsnum.Format(chartDimension)+" "+jsnum.Format(chartDimension)).
		Attr("xmlns", charts.SVGNamespace)

	marginedSpace := svg.Append("g").Attr("transform",
		"translate("+jsnum.Format(scatterMargin.left)+","+jsnum.Format(scatterMargin.top)+")")
	marginedSpace.Append("clipPath").Attr("id", "clip").
		Append("rect").Attr("width", jsnum.Format(width)).Attr("height", jsnum.Format(height))

	if opt.drawEmpty {
		empties := marginedSpace.Append("g").Class("empty-hexagon")
		for _, centre := range emptyHexagons(centresX, centresY, opt.packRadius) {
			path := empties.Append("path").
				Attr("d", "M"+jsnum.Format(xScale.Scale(centre.x))+","+jsnum.Format(yScale.Scale(centre.y))+" "+hexagon).
				Attr("fill", colour(0)).
				Attr("stroke", stroke).
				Attr("stroke-width", strokeWidth)
			path.Append("title").Text("Count: 0\nPercentage: 0.00%\nCenter: " +
				charts.FormatFixed2(centre.x) + ", " + charts.FormatFixed2(centre.y) + "\n")
		}
	}

	hexagons := marginedSpace.Append("g").Class("hexagon").Attr("clip-path", "url(#clip)")
	for _, bin := range bins {
		path := hexagons.Append("path").
			Attr("d", "M"+jsnum.Format(xScale.Scale(bin.X))+","+jsnum.Format(yScale.Scale(bin.Y))+" "+hexagon).
			Attr("fill", colour(float64(len(bin.Points)))).
			Attr("stroke", stroke).
			Attr("stroke-width", strokeWidth)
		path.Append("title").Text(hexBinTooltip(bin, len(values)))
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

// hexBinTooltip is the hover text for one hexagon: its count and the bounds of
// the points it holds.
func hexBinTooltip(bin charts.HexBin, total int) string {
	xs := make([]float64, len(bin.Points))
	ys := make([]float64, len(bin.Points))
	for i, p := range bin.Points {
		xs[i], ys[i] = p.X, p.Y
	}
	xBounds, yBounds := charts.Extent(xs), charts.Extent(ys)
	percentage := 100.0 * float64(len(bin.Points)) / float64(total)

	return "Count: " + jsnum.Format(float64(len(bin.Points))) + "\n" +
		"Percentage: " + charts.FormatFixed2(percentage) + "%\n" +
		"Center: " + charts.FormatFixed2(bin.X) + ", " + charts.FormatFixed2(bin.Y) + "\n" +
		"Min X: " + charts.FormatFixed2(xBounds[0]) + "\n" +
		"Max X: " + charts.FormatFixed2(xBounds[1]) + "\n" +
		"Min Y: " + charts.FormatFixed2(yBounds[0]) + "\n" +
		"Max Y: " + charts.FormatFixed2(yBounds[1]) + "\n"
}

// hexCentre is the middle of a hexagon with no points in it.
type hexCentre struct{ x, y float64 }

// Cosine and sine of pi/6, the half-angle between hexagon vertices. Tabulated
// for the same reason as hexAngleSin: Go's libm and V8's differ in the last ulp
// here, and the values reach the SVG.
const (
	hexHalfAngleCos = 0.8660254037844387
	hexHalfAngleSin = 0.49999999999999994
)

// emptyHexagons tiles the area the data spans with hexagon centres, so cells
// holding no points can still be outlined.
func emptyHexagons(centresX, centresY []float64, radius float64) []hexCentre {
	xBounds, yBounds := charts.Extent(centresX), charts.Extent(centresY)
	centreToEdge := hexHalfAngleCos * radius
	edgeLength := hexHalfAngleSin * radius

	var centres []hexCentre
	indent := false
	for y := yBounds[0]; y <= yBounds[1]+radius; y += edgeLength + radius {
		for x := xBounds[0]; x <= xBounds[1]+radius; x += 2 * centreToEdge {
			if indent && x >= xBounds[1] {
				break
			}
			cx := x
			if indent {
				cx += centreToEdge
			}
			centres = append(centres, hexCentre{x: cx, y: y})
		}
		indent = !indent
	}
	return centres
}
