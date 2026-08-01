package ops

import "math"

// tinycolor2 HSL colour maths, ported for CyberChef's Image
// Hue/Saturation/Lightness (Jimp's color([...]) uses tinycolor's
// spin/saturate/lighten). The odd percentage round-trip in hslToRgb
// (convertToPercentage -> bound01) is reproduced exactly, including its
// truncation to 4 decimal places, so results match byte-for-byte.

func tcClamp01(v float64) float64 { return math.Min(1, math.Max(0, v)) }

// tcBound01Num maps a plain number in [0,max] to [0,1] (tinycolor bound01).
func tcBound01Num(n, mx float64) float64 {
	n = math.Min(mx, math.Max(0, n))
	if math.Abs(n-mx) < 0.000001 {
		return 1
	}
	return math.Mod(n, mx) / mx
}

// tcBound01Pct maps a percentage value (s*100) to [0,1] the way tinycolor does
// when a value is stringified as "X%" and re-parsed, truncating via parseInt.
func tcBound01Pct(pct float64) float64 {
	n := math.Min(100, math.Max(0, pct))
	n = math.Trunc(n*100) / 100
	if math.Abs(n-100) < 0.000001 {
		return 1
	}
	return math.Mod(n, 100) / 100
}

// rgbToHsl converts 0-255 RGB to HSL, with h in [0,1].
func rgbToHsl(r, g, b int) (h, s, l float64) {
	rf := tcBound01Num(float64(r), 255)
	gf := tcBound01Num(float64(g), 255)
	bf := tcBound01Num(float64(b), 255)
	mx := math.Max(rf, math.Max(gf, bf))
	mn := math.Min(rf, math.Min(gf, bf))
	l = (mx + mn) / 2
	if mx == mn {
		return 0, 0, l
	}
	d := mx - mn
	if l > 0.5 {
		s = d / (2 - mx - mn)
	} else {
		s = d / (mx + mn)
	}
	switch mx {
	case rf:
		h = (gf - bf) / d
		if gf < bf {
			h += 6
		}
	case gf:
		h = (bf-rf)/d + 2
	default: // bf
		h = (rf-gf)/d + 4
	}
	return h / 6, s, l
}

func tcHue2rgb(p, q, t float64) float64 {
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

// hslToRgb converts HSL (hue in degrees; s,l in [0,1]) back to RGB in [0,255]
// (unrounded floats).
func hslToRgb(hDeg, s01, l01 float64) (r, g, b float64) {
	h := tcBound01Num(hDeg, 360)
	s := tcBound01Pct(s01 * 100)
	l := tcBound01Pct(l01 * 100)
	if s == 0 {
		return l * 255, l * 255, l * 255
	}
	var q float64
	if l < 0.5 {
		q = l * (1 + s)
	} else {
		q = l + s - l*s
	}
	p := 2*l - q
	r = tcHue2rgb(p, q, h+1.0/3)
	g = tcHue2rgb(p, q, h)
	b = tcHue2rgb(p, q, h-1.0/3)
	return r * 255, g * 255, b * 255
}

// tcRound clamps to 0-255 and rounds, matching tinycolor's toRgb.
func tcRound(v float64) byte {
	return byte(math.Round(math.Min(255, math.Max(0, v))))
}
