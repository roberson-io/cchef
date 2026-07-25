package ops

import (
	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(HexDensityChart{})
}

// HexDensityChart groups two-variable data into hexagonal cells and shades each
// by how many points it holds. Ported from CyberChef's Hex Density chart.
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
		{Name: "Record delimiter", Type: core.ArgOption, Value: recordDelimiterOptions},
		{Name: "Field delimiter", Type: core.ArgOption, Value: fieldDelimiterOptions},
		{Name: "Pack radius", Type: core.ArgNumber, Value: float64(25)},
		{Name: "Draw radius", Type: core.ArgNumber, Value: float64(15)},
		{Name: "Use column headers as labels", Type: core.ArgBoolean, Value: true},
		{Name: "X label", Type: core.ArgString, Value: ""},
		{Name: "Y label", Type: core.ArgString, Value: ""},
		{Name: "Draw hexagon edges", Type: core.ArgBoolean, Value: false},
		{Name: "Min colour value", Type: core.ArgString, Value: colourMin},
		{Name: "Max colour value", Type: core.ArgString, Value: colourMax},
		{Name: "Draw empty hexagons within data boundaries", Type: core.ArgBoolean, Value: false},
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

	headings, values, err := getScatterValues(in.String(), recordDelimiter, fieldDelimiter, headingsIncluded)
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
	return core.NewDish([]byte(svg.render()), core.TypeString), nil
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
func hexDensitySVG(values []scatterPoint, opt hexDensityOptions) *svgEl {
	width := chartDimension - scatterMargin.left - scatterMargin.right
	height := chartDimension - scatterMargin.top - scatterMargin.bottom

	layout := newHexbin(opt.packRadius)
	bins := layout.bin(values)

	maxCount := 0
	centresX := make([]float64, len(bins))
	centresY := make([]float64, len(bins))
	for i, bin := range bins {
		maxCount = max(maxCount, len(bin.points))
		centresX[i], centresY[i] = bin.x, bin.y
	}

	// The axes are widened past the hexagon centres so whole hexagons fit.
	xExtent := chartExtent(centresX)
	yExtent := chartExtent(centresY)
	xExtent[0] -= 2 * opt.packRadius
	xExtent[1] += 3 * opt.packRadius
	yExtent[0] -= 2 * opt.packRadius
	yExtent[1] += 2 * opt.packRadius

	xScale := scaleLinear(xExtent, [2]float64{0, width})
	yScale := scaleLinear(yExtent, [2]float64{height, 0})
	colour := sequentialColour(opt.minColour, opt.maxColour, float64(maxCount))
	hexagon := layout.hexagonPath(opt.drawRadius)

	stroke, strokeWidth := "none", "none"
	if opt.drawEdges {
		stroke, strokeWidth = "black", "0.5"
	}

	svg := newSVGEl("svg").
		attr("width", "100%").
		attr("height", "100%").
		attr("viewBox", "0 0 "+jsNum(chartDimension)+" "+jsNum(chartDimension)).
		attr("xmlns", svgNamespace)

	marginedSpace := svg.append("g").attr("transform",
		"translate("+jsNum(scatterMargin.left)+","+jsNum(scatterMargin.top)+")")
	marginedSpace.append("clipPath").attr("id", "clip").
		append("rect").attr("width", jsNum(width)).attr("height", jsNum(height))

	if opt.drawEmpty {
		empties := marginedSpace.append("g").class("empty-hexagon")
		for _, centre := range emptyHexagons(centresX, centresY, opt.packRadius) {
			path := empties.append("path").
				attr("d", "M"+jsNum(xScale.scale(centre.x))+","+jsNum(yScale.scale(centre.y))+" "+hexagon).
				attr("fill", colour(0)).
				attr("stroke", stroke).
				attr("stroke-width", strokeWidth)
			path.append("title").text("Count: 0\nPercentage: 0.00%\nCenter: " +
				formatFixed2(centre.x) + ", " + formatFixed2(centre.y) + "\n")
		}
	}

	hexagons := marginedSpace.append("g").class("hexagon").attr("clip-path", "url(#clip)")
	for _, bin := range bins {
		path := hexagons.append("path").
			attr("d", "M"+jsNum(xScale.scale(bin.x))+","+jsNum(yScale.scale(bin.y))+" "+hexagon).
			attr("fill", colour(float64(len(bin.points)))).
			attr("stroke", stroke).
			attr("stroke-width", strokeWidth)
		path.append("title").text(hexBinTooltip(bin, len(values)))
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

// hexBinTooltip is the hover text for one hexagon: its count and the bounds of
// the points it holds.
func hexBinTooltip(bin hexBin, total int) string {
	xs := make([]float64, len(bin.points))
	ys := make([]float64, len(bin.points))
	for i, p := range bin.points {
		xs[i], ys[i] = p.x, p.y
	}
	xBounds, yBounds := chartExtent(xs), chartExtent(ys)
	percentage := 100.0 * float64(len(bin.points)) / float64(total)

	return "Count: " + jsNum(float64(len(bin.points))) + "\n" +
		"Percentage: " + formatFixed2(percentage) + "%\n" +
		"Center: " + formatFixed2(bin.x) + ", " + formatFixed2(bin.y) + "\n" +
		"Min X: " + formatFixed2(xBounds[0]) + "\n" +
		"Max X: " + formatFixed2(xBounds[1]) + "\n" +
		"Min Y: " + formatFixed2(yBounds[0]) + "\n" +
		"Max Y: " + formatFixed2(yBounds[1]) + "\n"
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
	xBounds, yBounds := chartExtent(centresX), chartExtent(centresY)
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
