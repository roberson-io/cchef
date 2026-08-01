package ops

import (
	"math"
	"testing"
)

func approx(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestRGBToHSL(t *testing.T) {
	cases := []struct {
		r, g, b int
		h, s, l float64
	}{
		{255, 0, 0, 0, 1, 0.5},             // red max
		{0, 255, 0, 1.0 / 3, 1, 0.5},       // green max
		{0, 0, 255, 2.0 / 3, 1, 0.5},       // blue max
		{128, 128, 128, 0, 0, 128.0 / 255}, // achromatic
	}
	for _, c := range cases {
		h, s, l := rgbToHsl(c.r, c.g, c.b)
		if !approx(h, c.h) || !approx(s, c.s) || !approx(l, c.l) {
			t.Errorf("rgbToHsl(%d,%d,%d) = (%v,%v,%v), want (%v,%v,%v)", c.r, c.g, c.b, h, s, l, c.h, c.s, c.l)
		}
	}
	// gf < bf branch (red is max, green below blue) pushes hue past 6.
	if h, _, _ := rgbToHsl(255, 0, 128); h <= 0.9 {
		t.Errorf("rgbToHsl red-max/g<b hue = %v, want > 0.9", h)
	}
}

func TestHSLToRGB(t *testing.T) {
	// Achromatic (s == 0) -> grey.
	if r, g, b := hslToRgb(0, 0, 0.5); tcRound(r) != 128 || tcRound(g) != 128 || tcRound(b) != 128 {
		t.Errorf("hslToRgb grey = %v,%v,%v", tcRound(r), tcRound(g), tcRound(b))
	}
	// Round-trips of the primaries recover them (exercises tcHue2rgb branches).
	for _, c := range [][3]int{{255, 0, 0}, {0, 255, 0}, {0, 0, 255}, {90, 160, 210}} {
		h, s, l := rgbToHsl(c[0], c[1], c[2])
		r, g, b := hslToRgb(h*360, s, l)
		if int(tcRound(r)) != c[0] || int(tcRound(g)) != c[1] || int(tcRound(b)) != c[2] {
			t.Errorf("round trip %v = %d,%d,%d", c, tcRound(r), tcRound(g), tcRound(b))
		}
	}
}
