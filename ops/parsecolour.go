package ops

import (
	"fmt"
	"math"
	"regexp"
	"strconv"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(ParseColourCode{})
}

var (
	reColHex  = regexp.MustCompile(`(?i)#([a-f0-9]{2})([a-f0-9]{2})([a-f0-9]{2})`)
	reColRGB  = regexp.MustCompile(`(?i)rgba?\((\d{1,3}(?:\.\d+)?),\s?(\d{1,3}(?:\.\d+)?),\s?(\d{1,3}(?:\.\d+)?)(?:,\s?(\d(?:\.\d+)?))?\)`)
	reColHSL  = regexp.MustCompile(`(?i)hsla?\((\d{1,3}(?:\.\d+)?),\s?(\d{1,3}(?:\.\d+)?)%,\s?(\d{1,3}(?:\.\d+)?)%(?:,\s?(\d(?:\.\d+)?))?\)`)
	reColCMYK = regexp.MustCompile(`(?i)cmyk\((\d(?:\.\d+)?),\s?(\d(?:\.\d+)?),\s?(\d(?:\.\d+)?),\s?(\d(?:\.\d+)?)\)`)
)

// ParseColourCode parses a colour (hex/rgb/hsl/cmyk) and converts between formats.
type ParseColourCode struct{}

// Meta returns the operation metadata.
func (ParseColourCode) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Parse colour code",
		Module:      "Default",
		Description: "Converts a colour code in a standard format (hex, RGB, RGBA, HSL, HSLA, CMYK) into all the others.",
		InfoURL:     "https://wikipedia.org/wiki/Web_colors",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ParseColourCode) Args() []core.ArgDef { return nil }

// Run parses the colour. Ported from CyberChef ParseColourCode.mjs.
func (ParseColourCode) Run(in *core.Dish, args []any) (*core.Dish, error) {
	input := in.String()
	var r, g, b, a float64 = 0, 0, 0, 1

	switch {
	case reColHex.MatchString(input):
		m := reColHex.FindStringSubmatch(input)
		r = parseHex(m[1])
		g = parseHex(m[2])
		b = parseHex(m[3])
	case reColRGB.MatchString(input):
		m := reColRGB.FindStringSubmatch(input)
		r, g, b = atof(m[1]), atof(m[2]), atof(m[3])
		if m[4] != "" {
			a = atof(m[4])
		}
	case reColHSL.MatchString(input):
		m := reColHSL.FindStringSubmatch(input)
		r, g, b = hslToRGB(atof(m[1])/360, atof(m[2])/100, atof(m[3])/100)
		if m[4] != "" {
			a = atof(m[4])
		}
	case reColCMYK.MatchString(input):
		m := reColCMYK.FindStringSubmatch(input)
		c, mm, y, k := atof(m[1]), atof(m[2]), atof(m[3]), atof(m[4])
		r = math.Round(255 * (1 - c) * (1 - k))
		g = math.Round(255 * (1 - mm) * (1 - k))
		b = math.Round(255 * (1 - y) * (1 - k))
	}

	h0, s0, l0 := rgbToHSL(r, g, b)
	h := int(math.Round(h0 * 360))
	s := int(math.Round(s0 * 100))
	l := int(math.Round(l0 * 100))

	k := 1 - math.Max(r/255, math.Max(g/255, b/255))
	cmykStr := func(v float64) string {
		if math.IsNaN(v) {
			return "0"
		}
		return strconv.FormatFloat(v, 'f', 2, 64)
	}
	c := cmykStr((1 - r/255 - k) / (1 - k))
	mC := cmykStr((1 - g/255 - k) / (1 - k))
	y := cmykStr((1 - b/255 - k) / (1 - k))
	kStr := strconv.FormatFloat(k, 'f', 2, 64)

	hex := "#" + fmt.Sprintf("%02x", int(math.Round(r))) + fmt.Sprintf("%02x", int(math.Round(g))) + fmt.Sprintf("%02x", int(math.Round(b)))
	rgb := fmt.Sprintf("rgb(%s, %s, %s)", jsNum(r), jsNum(g), jsNum(b))
	rgba := fmt.Sprintf("rgba(%s, %s, %s, %s)", jsNum(r), jsNum(g), jsNum(b), jsNum(a))
	hsl := fmt.Sprintf("hsl(%d, %d%%, %d%%)", h, s, l)
	hsla := fmt.Sprintf("hsla(%d, %d%%, %d%%, %s)", h, s, l, jsNum(a))
	cmyk := fmt.Sprintf("cmyk(%s, %s, %s, %s)", c, mC, y, kStr)

	out := fmt.Sprintf(`<div id="colorpicker" style="white-space: normal;"></div>
Hex:  %s
RGB:  %s
RGBA: %s
HSL:  %s
HSLA: %s
CMYK: %s
<script>
    $('#colorpicker').colorpicker({
        format: 'rgba',
        color: '%s',
        container: true,
        inline: true,
        useAlpha: true
    }).on('colorpickerChange', function(e) {
        var color = e.color.string('rgba');
        window.app.manager.input.setInput(color);
        window.app.manager.input.inputChange(new Event("keyup"));
    });
</script>`, hex, rgb, rgba, hsl, hsla, cmyk, rgba)
	return core.NewDish([]byte(out), core.TypeString), nil
}

func parseHex(s string) float64 { v, _ := strconv.ParseInt(s, 16, 64); return float64(v) }
func atof(s string) float64     { v, _ := strconv.ParseFloat(s, 64); return v }

// rgbToHSL converts RGB (0-255) to HSL (each 0-1). Ported from ParseColourCode.
func rgbToHSL(r, g, b float64) (float64, float64, float64) {
	r, g, b = r/255, g/255, b/255
	cmax := math.Max(r, math.Max(g, b))
	cmin := math.Min(r, math.Min(g, b))
	l := (cmax + cmin) / 2
	if cmax == cmin {
		return 0, 0, l
	}
	d := cmax - cmin
	var s float64
	if l > 0.5 {
		s = d / (2 - cmax - cmin)
	} else {
		s = d / (cmax + cmin)
	}
	var h float64
	switch cmax {
	case r:
		h = (g - b) / d
		if g < b {
			h += 6
		}
	case g:
		h = (b-r)/d + 2
	case b:
		h = (r-g)/d + 4
	}
	return h / 6, s, l
}

// hslToRGB converts HSL (each 0-1) to RGB (0-255, rounded). Ported from ParseColourCode.
func hslToRGB(h, s, l float64) (float64, float64, float64) {
	if s == 0 {
		v := math.Round(l * 255)
		return v, v, v
	}
	hue2rgb := func(p, q, t float64) float64 {
		if t < 0 {
			t++
		}
		if t > 1 {
			t--
		}
		switch {
		case t < 1.0/6:
			return p + (q-p)*6*t
		case t < 1.0/2:
			return q
		case t < 2.0/3:
			return p + (q-p)*(2.0/3-t)*6
		default:
			return p
		}
	}
	var q float64
	if l < 0.5 {
		q = l * (1 + s)
	} else {
		q = l + s - l*s
	}
	p := 2*l - q
	return math.Round(hue2rgb(p, q, h+1.0/3) * 255),
		math.Round(hue2rgb(p, q, h) * 255),
		math.Round(hue2rgb(p, q, h-1.0/3) * 255)
}
