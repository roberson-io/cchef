package charts

import (
	"math"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/internal/jsnum"
)

// A port of the parts of d3-color and d3-interpolate the heatmap and hex density
// charts use: parsing a CSS colour, and blending two colours through CIELAB.

// d3Colour is a parsed sRGB colour with an alpha channel.
type d3Colour struct {
	r, g, b, alpha float64
}

// parseD3Colour parses a CSS colour string the way d3-color does: a colour
// keyword, a 3/4/6/8-digit hex form, or an rgb()/rgba()/hsl()/hsla() function.
func parseD3Colour(s string) (d3Colour, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	switch {
	case s == "":
		return d3Colour{}, false
	case s == "transparent":
		// d3 models the keyword as fully transparent with unknown channels.
		return d3Colour{r: math.NaN(), g: math.NaN(), b: math.NaN(), alpha: 0}, true
	case strings.HasPrefix(s, "#"):
		return parseHexColour(s[1:])
	case strings.HasSuffix(s, ")"):
		return parseFunctionalColour(s)
	}
	if v, ok := namedColours[s]; ok {
		return d3Colour{
			r:     float64((v >> 16) & 0xff),
			g:     float64((v >> 8) & 0xff),
			b:     float64(v & 0xff),
			alpha: 1,
		}, true
	}
	return unknownColour(), false
}

// unknownColour is what an unparseable colour behaves as: every channel NaN,
// which d3 propagates through conversion and interpolation.
func unknownColour() d3Colour {
	nan := math.NaN()
	return d3Colour{r: nan, g: nan, b: nan, alpha: nan}
}

// parseHexColour parses the digits of a #rgb, #rgba, #rrggbb or #rrggbbaa form.
func parseHexColour(digits string) (d3Colour, bool) {
	// Indexed so each byte is checked as itself; narrowing a multi-byte rune
	// could truncate it onto an unrelated ASCII hex digit.
	for i := range len(digits) {
		if !jsnum.IsHexDigit(digits[i]) {
			return d3Colour{}, false
		}
	}
	nibble := func(i int) float64 {
		v, _ := strconv.ParseUint(digits[i:i+1], 16, 8)
		return float64(v*16 + v)
	}
	pair := func(i int) float64 {
		v, _ := strconv.ParseUint(digits[i:i+2], 16, 16)
		return float64(v)
	}
	switch len(digits) {
	case 3:
		return d3Colour{r: nibble(0), g: nibble(1), b: nibble(2), alpha: 1}, true
	case 4:
		return d3Colour{r: nibble(0), g: nibble(1), b: nibble(2), alpha: nibble(3) / 255}, true
	case 6:
		return d3Colour{r: pair(0), g: pair(2), b: pair(4), alpha: 1}, true
	case 8:
		return d3Colour{r: pair(0), g: pair(2), b: pair(4), alpha: pair(6) / 255}, true
	}
	return d3Colour{}, false
}

// parseFunctionalColour parses rgb()/rgba()/hsl()/hsla() notation.
func parseFunctionalColour(s string) (d3Colour, bool) {
	name, rest, ok := strings.Cut(s, "(")
	if !ok {
		return d3Colour{}, false
	}
	parts := strings.Split(strings.TrimSuffix(rest, ")"), ",")
	if len(parts) < 3 || len(parts) > 4 {
		return d3Colour{}, false
	}

	alpha := 1.0
	if len(parts) == 4 {
		alpha = jsnum.ParseFloat(strings.TrimSpace(parts[3]))
	}
	switch name {
	case "rgb", "rgba":
		return d3Colour{
			r:     colourComponent(parts[0], 255),
			g:     colourComponent(parts[1], 255),
			b:     colourComponent(parts[2], 255),
			alpha: alpha,
		}, true
	case "hsl", "hsla":
		h := jsnum.ParseFloat(strings.TrimSpace(parts[0]))
		sat := colourComponent(parts[1], 1)
		light := colourComponent(parts[2], 1)
		r, g, b := cssHSLToRGB(h, sat, light)
		return d3Colour{r: r, g: g, b: b, alpha: alpha}, true
	}
	return d3Colour{}, false
}

// colourComponent reads one channel, scaling a percentage by full.
func colourComponent(s string, full float64) float64 {
	s = strings.TrimSpace(s)
	if pct, ok := strings.CutSuffix(s, "%"); ok {
		return jsnum.ParseFloat(pct) / 100 * full
	}
	return jsnum.ParseFloat(s)
}

// cssHSLToRGB converts CSS HSL (hue in degrees, saturation and lightness 0-1)
// to sRGB channels in 0-255, as d3-color's Hsl.rgb does. This differs from
// hslToRGB in parsecolour.go, which takes a hue in 0-1.
func cssHSLToRGB(h, s, l float64) (float64, float64, float64) {
	h = math.Mod(h, 360)
	if h < 0 {
		h += 360
	}
	// d3's Hsl.rgb: m2 = l + (l < 0.5 ? l : 1 - l) * s.
	edge := l
	if l >= 0.5 {
		edge = 1 - l
	}
	m2 := l + edge*s
	m1 := 2*l - m2
	// Channels are left unrounded; formatRGB rounds once on the way out, as d3
	// does.
	return hslChannel(h+120, m1, m2), hslChannel(h, m1, m2), hslChannel(h-120, m1, m2)
}

// hslChannel resolves one HSL channel at hue offset h.
func hslChannel(h, m1, m2 float64) float64 {
	if h < 0 {
		h += 360
	}
	if h >= 360 {
		h -= 360
	}
	switch {
	case h < 60:
		return (m1 + (m2-m1)*h/60) * 255
	case h < 180:
		return m2 * 255
	case h < 240:
		return (m1 + (m2-m1)*(240-h)/60) * 255
	default:
		return m1 * 255
	}
}

// CIELAB conversion constants, from d3-color's lab.js (D50-adapted white point).
const (
	labXn = 0.96422
	labYn = 1.0
	labZn = 0.82521
	labT0 = 4.0 / 29.0
	labT1 = 6.0 / 29.0
)

var (
	labT2 = 3 * labT1 * labT1
	labT3 = labT1 * labT1 * labT1
)

// labColour is a colour in CIELAB. A NaN channel means "unknown", which is how
// d3 represents an unparseable colour and propagates it through interpolation.
type labColour struct {
	l, a, b, alpha float64
}

// toLab converts an sRGB colour to CIELAB. NaN channels stay NaN.
func toLab(c d3Colour) labColour {
	r, g, b := srgbToLinear(c.r), srgbToLinear(c.g), srgbToLinear(c.b)
	y := xyzToLab((0.2225045*r + 0.7168786*g + 0.0606169*b) / labYn)
	x, z := y, y
	if !(r == g && g == b) {
		x = xyzToLab((0.4360747*r + 0.3850649*g + 0.1430804*b) / labXn)
		z = xyzToLab((0.0139322*r + 0.0971045*g + 0.7141733*b) / labZn)
	}
	return labColour{l: 116*y - 16, a: 500 * (x - y), b: 200 * (y - z), alpha: c.alpha}
}

// toRGB converts a CIELAB colour back to sRGB.
func (c labColour) toRGB() d3Colour {
	y := (c.l + 16) / 116
	x, z := y, y
	if !math.IsNaN(c.a) {
		x = y + c.a/500
	}
	if !math.IsNaN(c.b) {
		z = y - c.b/200
	}
	x, y, z = labXn*labToXYZ(x), labYn*labToXYZ(y), labZn*labToXYZ(z)
	return d3Colour{
		r:     linearToSRGB(3.1338561*x - 1.6168667*y - 0.4906146*z),
		g:     linearToSRGB(-0.9787684*x + 1.9161415*y + 0.0334540*z),
		b:     linearToSRGB(0.0719453*x - 0.2289914*y + 1.4052427*z),
		alpha: c.alpha,
	}
}

func xyzToLab(t float64) float64 {
	if t > labT3 {
		return math.Cbrt(t)
	}
	return t/labT2 + labT0
}

func labToXYZ(t float64) float64 {
	if t > labT1 {
		return t * t * t
	}
	return labT2 * (t - labT0)
}

func srgbToLinear(x float64) float64 {
	x /= 255
	if x <= 0.04045 {
		return x / 12.92
	}
	return math.Pow((x+0.055)/1.055, 2.4)
}

func linearToSRGB(x float64) float64 {
	if x <= 0.0031308 {
		return 255 * 12.92 * x
	}
	return 255 * (1.055*math.Pow(x, 1/2.4) - 0.055)
}

// InterpolateLab returns a function blending from one colour to another through
// CIELAB, rendering the result as a CSS rgb()/rgba() string.
func InterpolateLab(from, to string) func(float64) string {
	fromColour, _ := parseD3Colour(from)
	toColour, _ := parseD3Colour(to)
	start, end := toLab(fromColour), toLab(toColour)

	l := colourChannel(start.l, end.l)
	a := colourChannel(start.a, end.a)
	b := colourChannel(start.b, end.b)
	alpha := colourChannel(start.alpha, end.alpha)

	return func(t float64) string {
		return formatRGB(labColour{l: l(t), a: a(t), b: b(t), alpha: alpha(t)}.toRGB())
	}
}

// colourChannel interpolates one channel, following d3-interpolate's colour
// helper: blend when the endpoints differ, otherwise hold whichever endpoint is
// known. A NaN difference counts as no difference, since JavaScript treats NaN
// as falsy — that is what makes an unparseable endpoint fall back to the other.
// Note this blends as a + t*(b-a); the numeric scale interpolator does not.
func colourChannel(from, to float64) func(float64) float64 {
	d := to - from
	if d != 0 && !math.IsNaN(d) {
		return func(t float64) float64 { return from + float64(t*d) }
	}
	held := from
	if math.IsNaN(from) {
		held = to
	}
	return func(float64) float64 { return held }
}

// InterpolateRGB blends two colours through their red, green and blue channels
// directly, which is what the entropy image shades its cells with.
func InterpolateRGB(from, to string) func(float64) string {
	start, _ := parseD3Colour(from)
	end, _ := parseD3Colour(to)

	r := colourChannel(start.r, end.r)
	g := colourChannel(start.g, end.g)
	b := colourChannel(start.b, end.b)
	alpha := colourChannel(start.alpha, end.alpha)

	return func(t float64) string {
		return formatRGB(d3Colour{r: r(t), g: g(t), b: b(t), alpha: alpha(t)})
	}
}

// formatRGB renders a colour as d3 does: rgb() when opaque, rgba() otherwise,
// with channels rounded and clamped to 0-255. An unknown alpha counts as
// opaque, which is what keeps NaN out of the output.
func formatRGB(c d3Colour) string {
	channels := jsnum.Format(clampChannel(c.r)) + ", " + jsnum.Format(clampChannel(c.g)) + ", " + jsnum.Format(clampChannel(c.b))
	alpha := 1.0
	if !math.IsNaN(c.alpha) {
		alpha = math.Max(0, math.Min(1, c.alpha))
	}
	if alpha == 1 {
		return "rgb(" + channels + ")"
	}
	return "rgba(" + channels + ", " + jsnum.Format(alpha) + ")"
}

// clampChannel rounds a channel and clamps it to the 0-255 range.
func clampChannel(v float64) float64 {
	if math.IsNaN(v) {
		return 0
	}
	return math.Max(0, math.Min(255, math.Round(v)))
}
