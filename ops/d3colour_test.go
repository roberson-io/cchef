package ops

import (
	"testing"
)

// Colour parsing follows d3-color, which the chart colour options are fed to.
// Expected values were taken from d3 under Node.
// Expected strings are what d3.color(…) renders, so parsing and formatting are
// checked together — channels are only rounded on the way out, as in d3.
func TestParseD3Colour(t *testing.T) {
	for _, tc := range []struct {
		in         string
		want       string
		shouldFail bool
	}{
		{in: "#f00", want: "rgb(255, 0, 0)"},
		{in: "#ff0000", want: "rgb(255, 0, 0)"},
		{in: "rgb(1,2,3)", want: "rgb(1, 2, 3)"},
		{in: "rgb(10%, 20%, 30%)", want: "rgb(26, 51, 77)"},
		{in: "hsl(120,50%,50%)", want: "rgb(64, 191, 64)"},
		// Names are case-insensitive and surrounding space is ignored.
		{in: "REBECCAPURPLE", want: "rgb(102, 51, 153)"},
		{in: " red ", want: "rgb(255, 0, 0)"},
		{in: "white", want: "rgb(255, 255, 255)"},
		// Alpha comes through both the 8-digit hex form and the keyword. The
		// keyword's colour channels are unknown, which d3 renders as zeroes.
		{in: "#ff000080", want: "rgba(255, 0, 0, 0.5019607843137255)"},
		{in: "transparent", want: "rgba(0, 0, 0, 0)"},
		{in: "bogus", shouldFail: true},
		{in: "", shouldFail: true},
	} {
		got, ok := parseD3Colour(tc.in)
		if ok == tc.shouldFail {
			t.Errorf("parse(%q) ok = %v, want %v", tc.in, ok, !tc.shouldFail)
			continue
		}
		if tc.shouldFail {
			continue
		}
		if rendered := formatRGB(got); rendered != tc.want {
			t.Errorf("parse(%q) = %s, want %s", tc.in, rendered, tc.want)
		}
	}
}

// interpolateLab blends through CIELAB, which is what gives the heatmap and hex
// density charts their colour ramps.
func TestInterpolateLab(t *testing.T) {
	for _, tc := range []struct {
		from, to string
		want     []string
	}{
		{"white", "black", []string{
			"rgb(255, 255, 255)", "rgb(185, 185, 185)", "rgb(119, 119, 119)",
			"rgb(59, 59, 59)", "rgb(0, 0, 0)",
		}},
		{"white", "red", []string{
			"rgb(255, 255, 255)", "rgb(255, 208, 190)", "rgb(255, 159, 128)",
			"rgb(255, 105, 69)", "rgb(255, 0, 0)",
		}},
		{"white", "blue", []string{
			"rgb(255, 255, 255)", "rgb(218, 195, 255)", "rgb(175, 137, 255)",
			"rgb(122, 79, 255)", "rgb(0, 0, 255)",
		}},
	} {
		interp := interpolateLab(tc.from, tc.to)
		for i, at := range []float64{0, 0.25, 0.5, 0.75, 1} {
			if got := interp(at); got != tc.want[i] {
				t.Errorf("%s->%s at %v = %q, want %q", tc.from, tc.to, at, got, tc.want[i])
			}
		}
	}
}

// An unparseable endpoint leaves the other one in place, as d3's interpolator
// does when a channel is NaN.
func TestInterpolateLabUnparseable(t *testing.T) {
	if got := interpolateLab("white", "bogus")(1); got != "rgb(255, 255, 255)" {
		t.Errorf("white->bogus at 1 = %q, want rgb(255, 255, 255)", got)
	}
	if got := interpolateLab("bogus", "black")(0); got != "rgb(0, 0, 0)" {
		t.Errorf("bogus->black at 0 = %q, want rgb(0, 0, 0)", got)
	}
}

// A translucent endpoint produces rgba() output.
func TestInterpolateLabAlpha(t *testing.T) {
	if got := interpolateLab("white", "transparent")(1); got != "rgba(255, 255, 255, 0)" {
		t.Errorf("white->transparent at 1 = %q, want rgba(255, 255, 255, 0)", got)
	}
}

// Malformed colour strings are rejected rather than producing wrong colours.
func TestParseD3ColourMalformed(t *testing.T) {
	for _, in := range []string{
		"#12",      // wrong number of hex digits
		"#12345",   // wrong number of hex digits
		"#gg0000",  // not hex
		"rgb(1,2)", // too few components
		"rgb(1,2,3,4,5)",
		"cmyk(1,2,3)", // unknown function
		"()",
	} {
		if _, ok := parseD3Colour(in); ok {
			t.Errorf("parse(%q) succeeded, want rejection", in)
		}
	}
}

// The remaining hex and functional forms.
func TestParseD3ColourMoreForms(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"#f008", "rgba(255, 0, 0, 0.5333333333333333)"},
		{"rgba(1, 2, 3, 0.5)", "rgba(1, 2, 3, 0.5)"},
		{"hsla(0, 100%, 50%, 0.25)", "rgba(255, 0, 0, 0.25)"},
		// Hue wraps, and a zero-saturation colour is grey.
		{"hsl(480, 50%, 50%)", "rgb(64, 191, 64)"},
		{"hsl(-120, 50%, 50%)", "rgb(64, 64, 191)"},
		{"hsl(0, 0%, 50%)", "rgb(128, 128, 128)"},
		// Lightness above one half takes the other branch of the HSL maths.
		{"hsl(240, 100%, 75%)", "rgb(128, 128, 255)"},
		// Every hue sextant.
		{"hsl(30, 100%, 50%)", "rgb(255, 128, 0)"},
		{"hsl(150, 100%, 50%)", "rgb(0, 255, 128)"},
		{"hsl(210, 100%, 50%)", "rgb(0, 128, 255)"},
		{"hsl(300, 100%, 50%)", "rgb(255, 0, 255)"},
	} {
		got, ok := parseD3Colour(tc.in)
		if !ok {
			t.Errorf("parse(%q) failed", tc.in)
			continue
		}
		if rendered := formatRGB(got); rendered != tc.want {
			t.Errorf("parse(%q) = %s, want %s", tc.in, rendered, tc.want)
		}
	}
}

// Both endpoints unparseable leaves every channel unknown, which renders as
// opaque black rather than NaN.
func TestInterpolateLabBothUnparseable(t *testing.T) {
	if got := interpolateLab("nope", "alsonope")(0.5); got != "rgb(0, 0, 0)" {
		t.Errorf("both unparseable = %q, want rgb(0, 0, 0)", got)
	}
}

// A ramp between identical colours holds that colour throughout.
func TestInterpolateLabSameColour(t *testing.T) {
	interp := interpolateLab("red", "red")
	for _, at := range []float64{0, 0.5, 1} {
		if got := interp(at); got != "rgb(255, 0, 0)" {
			t.Errorf("red->red at %v = %q, want rgb(255, 0, 0)", at, got)
		}
	}
}

// A string ending in ")" with no opening bracket is not a colour function.
func TestParseD3ColourNoOpenBracket(t *testing.T) {
	if _, ok := parseD3Colour("abc)"); ok {
		t.Error("parse(\"abc)\") succeeded, want rejection")
	}
}

// A multi-byte rune must not be narrowed onto an ASCII hex digit: U+0130's low
// byte is '0', so truncating would let "#İ2345" through as a colour.
func TestParseD3ColourNonASCIIHex(t *testing.T) {
	for _, in := range []string{"#İ23", "#İ23456", "#ff00İ0"} {
		if _, ok := parseD3Colour(in); ok {
			t.Errorf("parse(%q) succeeded, want rejection", in)
		}
	}
}
