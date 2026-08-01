package ops

import (
	"math"
	"strconv"
	"strings"
)

// A port of d3-hexbin, which the hex density chart uses to group points into
// hexagonal cells.

// hexAngleSin and hexAngleCos are sine and cosine of the six hexagon vertex
// angles (i * pi/3). They are tabulated rather than computed because Go's and
// V8's libm disagree in the last ulp or two for these arguments, and the
// results are printed straight into the SVG path — these are the values
// CyberChef's own output is built from.
var (
	hexAngleSin = [6]float64{
		0, 0.8660254037844386, 0.8660254037844387,
		1.2246467991473532e-16, -0.8660254037844385, -0.866025403784439,
	}
	hexAngleCos = [6]float64{
		1, 0.5000000000000001, -0.4999999999999998,
		-1, -0.5000000000000004, 0.49999999999999933,
	}
)

// hexBin is one hexagonal cell: its centre and the points that fell in it.
type hexBin struct {
	x, y   float64
	points []scatterPoint
}

// hexbinLayout groups points into hexagons of a given radius.
type hexbinLayout struct {
	radius, dx, dy float64
}

// newHexbin builds a layout for hexagons of the given radius.
func newHexbin(radius float64) hexbinLayout {
	return hexbinLayout{
		radius: radius,
		dx:     radius * 2 * hexAngleSin[1],
		dy:     radius * 1.5,
	}
}

// bin groups points into hexagons, in the order each hexagon is first reached.
func (h hexbinLayout) bin(points []scatterPoint) []hexBin {
	var bins []hexBin
	index := make(map[string]int)

	for _, point := range points {
		if math.IsNaN(point.x) || math.IsNaN(point.y) {
			continue
		}
		pi, pj := h.cell(point.x, point.y)

		id := jsNum(pi) + "-" + jsNum(pj)
		if at, ok := index[id]; ok {
			bins[at].points = append(bins[at].points, point)
			continue
		}
		index[id] = len(bins)
		bins = append(bins, hexBin{
			x:      (pi + hexRowOffset(pj)) * h.dx,
			y:      pj * h.dy,
			points: []scatterPoint{point},
		})
	}
	return bins
}

// cell returns the hexagon column and row a point falls in. Points near a cell
// boundary are reassigned to whichever centre is actually closer, as d3 does.
func (h hexbinLayout) cell(x, y float64) (float64, float64) {
	// JavaScript's Math.round breaks halves towards positive infinity, where
	// Go's math.Round breaks them away from zero; the two disagree for
	// negative coordinates and would bin those points differently.
	py := y / h.dy
	pj := jsRoundFloat(py)
	px := x/h.dx - hexRowOffset(pj)
	pi := jsRoundFloat(px)
	py1 := py - pj

	if math.Abs(py1)*3 <= 1 {
		return pi, pj
	}
	px1 := px - pi
	pi2 := pi + hexStep(px, pi)/2
	pj2 := pj + hexStep(py, pj)
	px2, py2 := px-pi2, py-pj2
	if px1*px1+py1*py1 > px2*px2+py2*py2 {
		odd := -1.0
		if int(pj)&1 != 0 {
			odd = 1
		}
		return pi2 + odd/2, pj2
	}
	return pi, pj
}

// hexRowOffset is the half-cell indent applied to odd rows.
func hexRowOffset(pj float64) float64 {
	if int(pj)&1 != 0 {
		return 0.5
	}
	return 0
}

// hexStep is -1 when v falls below its rounded value, +1 otherwise.
func hexStep(v, rounded float64) float64 {
	if v < rounded {
		return -1
	}
	return 1
}

// hexagonPath returns the relative path d3-hexbin draws for one hexagon.
func (h hexbinLayout) hexagonPath(radius float64) string {
	var path strings.Builder
	path.WriteString("m")
	x0, y0 := 0.0, 0.0
	for i := range 6 {
		// The explicit conversions stop Go fusing the multiply and the
		// following subtraction into an FMA, which would round once where
		// JavaScript rounds twice and shift the last digits of the path.
		x1 := float64(hexAngleSin[i] * radius)
		y1 := float64(-hexAngleCos[i] * radius)
		if i > 0 {
			path.WriteString("l")
		}
		path.WriteString(jsNum(x1-x0) + "," + jsNum(y1-y0))
		x0, y0 = x1, y1
	}
	return path.String() + "z"
}

// formatFixed2 renders a number with two decimal places, as JavaScript's
// toFixed(2) does for the chart tooltips.
func formatFixed2(v float64) string {
	return strconv.FormatFloat(v, 'f', 2, 64)
}
