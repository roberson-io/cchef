package jsnum

import (
	"math"
	"testing"
)

// TestFormat covers the spellings JavaScript gives a number, either side of the
// two points where it changes between positional and exponential notation. The
// expectations come from Node.
func TestFormat(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   float64
		want string
	}{
		{"zero", 0, "0"},
		{"one", 1, "1"},
		{"minus one", -1, "-1"},
		{"a half", 1.5, "1.5"},
		{"a negative half", -1.5, "-1.5"},
		{"a two-place fraction", 0.35, "0.35"},
		{"a round number", 10, "10"},
		{"a byte", 255, "255"},
		{"a tenth", 0.1, "0.1"},
		{"several places", 123.456, "123.456"},

		{"the smallest written positionally", 0.000001, "0.000001"},
		{"one step smaller, which needs an exponent", 1e-7, "1e-7"},
		{"smaller again", 1e-9, "1e-9"},
		{"a fraction with digits and an exponent", 1.5e-7, "1.5e-7"},

		{"the largest written positionally", 1e20, "100000000000000000000"},
		{"one step larger, which needs an exponent", 1e21, "1e+21"},
		{"with digits after the point", 1.5e21, "1.5e+21"},
		{"larger again", 1e22, "1e+22"},

		{"the largest whole number counted exactly", 9007199254740991, "9007199254740991"},
		{"its negative", -9007199254740991, "-9007199254740991"},
		{"a power of two", 4294967296, "4294967296"},
		{"past what is counted exactly", 1234567890123456789, "1234567890123456800"},

		{"the smallest number there is", 5e-324, "5e-324"},
		{"one a little larger", 1e-323, "1e-323"},
		{"the largest number there is", math.MaxFloat64, "1.7976931348623157e+308"},

		{"no number at all", math.NaN(), "NaN"},
		{"beyond counting", math.Inf(1), "Infinity"},
		{"beyond counting the other way", math.Inf(-1), "-Infinity"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Format(tc.in); got != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}

// TestFormatNegativeZero covers the sign JavaScript drops: a negative zero is
// written the same as a positive one.
func TestFormatNegativeZero(t *testing.T) {
	if got := Format(math.Copysign(0, -1)); got != "0" {
		t.Errorf("got %s, want 0", got)
	}
}
