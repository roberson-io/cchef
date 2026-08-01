package ops

import (
	"math"
	"strings"
)

// The line generator CyberChef's entropy views draw with: d3.line() carrying
// d3-shape's monotone curve, written through d3-path. Ported from
// ../CyberChef/node_modules/d3-shape/src/curve/monotone.js and
// ../CyberChef/node_modules/d3-path/src/path.js.

// d3Point is one point on a line.
type d3Point struct{ x, y float64 }

// d3PathDigits is how many decimals d3-shape keeps when it writes a path.
const d3PathDigits = 3

// d3RoundPath rounds a coordinate the way the path writer does. JavaScript
// rounds a half towards positive infinity, where Go rounds it away from zero,
// so the two disagree on negative halves unless the rounding is written out.
func d3RoundPath(v float64) float64 {
	k := math.Pow(10, d3PathDigits)
	return math.Floor(v*k+0.5) / k
}

// d3Path builds a path in the notation d3 writes.
type d3Path struct {
	out     strings.Builder
	started bool // whether a subpath has begun, and so may be closed
}

func (p *d3Path) number(v float64) string {
	return jsNumberString(d3RoundPath(v))
}

func (p *d3Path) moveTo(x, y float64) {
	p.out.WriteString("M" + p.number(x) + "," + p.number(y))
	p.started = true
}

func (p *d3Path) lineTo(x, y float64) {
	p.out.WriteString("L" + p.number(x) + "," + p.number(y))
}

func (p *d3Path) closePath() {
	if p.started {
		p.out.WriteString("Z")
	}
}

func (p *d3Path) bezierTo(x1, y1, x2, y2, x, y float64) {
	p.out.WriteString("C" + p.number(x1) + "," + p.number(y1) +
		"," + p.number(x2) + "," + p.number(y2) +
		"," + p.number(x) + "," + p.number(y))
}

// d3Sign is the sign the curve reads from a slope, which counts zero as
// positive.
func d3Sign(v float64) float64 {
	if v < 0 {
		return -1
	}
	return 1
}

// d3Falsy reports whether a value would not be believed in JavaScript, which
// is how the curve tests the spacing between its points.
func d3Falsy(v float64) bool {
	return v == 0 || math.IsNaN(v)
}

// d3MonotoneCurve tracks the three points the curve needs to settle the slope
// at each one, and the slope it last used.
type d3MonotoneCurve struct {
	path                    *d3Path
	x0, y0, x1, y1, tangent float64
	seen                    int
}

func newD3MonotoneCurve(path *d3Path) *d3MonotoneCurve {
	nan := math.NaN()
	return &d3MonotoneCurve{path: path, x0: nan, y0: nan, x1: nan, y1: nan, tangent: nan}
}

// d3SpacingOr gives the spacing to divide by, standing a signed zero in where
// there is none so the slope runs off the way the curve expects.
func d3SpacingOr(spacing, other float64) float64 {
	if !d3Falsy(spacing) {
		return spacing
	}
	if other < 0 {
		return math.Copysign(0, -1)
	}
	return 0
}

// slope3 is the tangent at the middle of three points, held to the steepness of
// the gentler side so the curve never overshoots between them.
func (c *d3MonotoneCurve) slope3(x2, y2 float64) float64 {
	h0 := c.x1 - c.x0
	h1 := x2 - c.x1

	s0 := (c.y1 - c.y0) / d3SpacingOr(h0, h1)
	s1 := (y2 - c.y1) / d3SpacingOr(h1, h0)

	// Each product is rounded before it is added. Go may otherwise fuse a
	// multiply and an add into one operation that rounds once, which lands a
	// unit away from the value JavaScript works out.
	p := (float64(s0*h1) + float64(s1*h0)) / (h0 + h1)
	slope := (d3Sign(s0) + d3Sign(s1)) *
		math.Min(math.Min(math.Abs(s0), math.Abs(s1)), 0.5*math.Abs(p))
	if d3Falsy(slope) {
		return 0
	}
	return slope
}

// slope2 is the tangent at an end of the line, worked out from the one slope
// beside it.
func (c *d3MonotoneCurve) slope2(t float64) float64 {
	h := c.x1 - c.x0
	if d3Falsy(h) {
		return t
	}
	return (3*(c.y1-c.y0)/h - t) / 2
}

// segment writes the curve between the last two points, as the cubic those two
// tangents describe.
func (c *d3MonotoneCurve) segment(t0, t1 float64) {
	dx := (c.x1 - c.x0) / 3
	c.path.bezierTo(c.x0+dx, c.y0+float64(dx*t0), c.x1-dx, c.y1-float64(dx*t1), c.x1, c.y1)
}

// point takes the next point of the line.
func (c *d3MonotoneCurve) point(x, y float64) {
	if x == c.x1 && y == c.y1 {
		return // a point in the same place as the last adds nothing
	}

	tangent := math.NaN()
	switch c.seen {
	case 0:
		c.seen = 1
		c.path.moveTo(x, y)
	case 1:
		// Nothing is drawn until a third point settles the slope at the second.
		c.seen = 2
	case 2:
		c.seen = 3
		tangent = c.slope3(x, y)
		c.segment(c.slope2(tangent), tangent)
	default:
		tangent = c.slope3(x, y)
		c.segment(c.tangent, tangent)
	}

	c.x0, c.x1 = c.x1, x
	c.y0, c.y1 = c.y1, y
	c.tangent = tangent
}

// end finishes the line.
func (c *d3MonotoneCurve) end() {
	switch c.seen {
	case 1:
		c.path.closePath() // a line of one point closes on itself
	case 2:
		c.path.lineTo(c.x1, c.y1)
	case 3:
		c.segment(c.tangent, c.slope2(c.tangent))
	}
}

// d3LineMonotoneX draws a path through the points, curving through each without
// straying beyond the values on either side of it. No points at all draw
// nothing, which d3 reports as a missing path rather than an empty one.
func d3LineMonotoneX(points []d3Point) string {
	path := &d3Path{}
	curve := newD3MonotoneCurve(path)
	for _, p := range points {
		curve.point(p.x, p.y)
	}
	curve.end()
	return path.out.String()
}
