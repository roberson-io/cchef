package jsnum

import (
	"math"
	"strings"
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

// TestString covers the String(x) spellings that differ from JSON: the
// non-finite values have names, where JSON.stringify writes null.
func TestString(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{math.NaN(), "NaN"},
		{math.Inf(1), "Infinity"},
		{math.Inf(-1), "-Infinity"},
		{0, "0"},
		{math.Copysign(0, -1), "0"},
		{1.5, "1.5"},
		{-2, "-2"},
		{0.1, "0.1"},
	}
	for _, c := range cases {
		if got := String(c.in); got != c.want {
			t.Errorf("String(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestParseInt covers parseInt(s, base): leading JS whitespace, an optional
// sign, then the longest run of digits valid in the base; no digits is NaN,
// reported as ok=false.
func TestParseInt(t *testing.T) {
	if v, ok := ParseInt("  -1f", 16); !ok || v != -31 {
		t.Errorf("signed hex: %d %v", v, ok)
	}
	if v, ok := ParseInt("FF", 16); !ok || v != 255 {
		t.Errorf("upper-case hex: %d %v", v, ok)
	}
	if v, ok := ParseInt("42abc", 10); !ok || v != 42 {
		t.Errorf("stops at the first invalid digit: %d %v", v, ok)
	}
	if v, ok := ParseInt("\u00a0+7", 10); !ok || v != 7 {
		t.Errorf("JS whitespace includes NBSP: %d %v", v, ok)
	}
	if v, ok := ParseInt("101", 2); !ok || v != 5 {
		t.Errorf("binary: %d %v", v, ok)
	}
	if _, ok := ParseInt("xyz", 16); ok {
		t.Error("no digits should be NaN")
	}
	if _, ok := ParseInt("-", 10); ok {
		t.Error("a bare sign should be NaN")
	}
}

// TestParseFloat covers parseFloat: leading whitespace, the longest numeric
// prefix including exponents and Infinity, and NaN when nothing numeric leads.
func TestParseFloat(t *testing.T) {
	cases := []struct {
		in   string
		want float64
	}{
		{"3.14", 3.14},
		{"  -2.5e2xyz", -250},
		{".5", 0.5},
		{"+Infinity", math.Inf(1)},
		{"7 8", 7},
	}
	for _, c := range cases {
		if got := ParseFloat(c.in); got != c.want {
			t.Errorf("ParseFloat(%q) = %v, want %v", c.in, got, c.want)
		}
	}
	if got := ParseFloat("abc"); !math.IsNaN(got) {
		t.Errorf("ParseFloat(\"abc\") = %v, want NaN", got)
	}
}

// TestParseHex covers parseInt(s, 16) as a float: the 0x prefix is allowed for
// that radix, parsing stops at the first non-hex digit, and values wider than
// 64 bits keep accumulating as floats rather than overflowing.
func TestParseHex(t *testing.T) {
	cases := []struct {
		in   string
		want float64
	}{
		{"ff", 255},
		{"  1a", 26},
		{"-10", -16},
		{"+10", 16},
		{"0x1F", 31},
		{"12zz", 18},
		{strings.Repeat("f", 20), 1.2089258196146292e+24},
	}
	for _, c := range cases {
		if got := ParseHex(c.in); got != c.want {
			t.Errorf("ParseHex(%q) = %v, want %v", c.in, got, c.want)
		}
	}
	for _, in := range []string{"", "zz", "-"} {
		if got := ParseHex(in); !math.IsNaN(got) {
			t.Errorf("ParseHex(%q) = %v, want NaN", in, got)
		}
	}
	if got := ParseHex("-" + strings.Repeat("f", 20)); got >= 0 {
		t.Errorf("ParseHex of a wide negative = %v, want a negative value", got)
	}
}

// TestIsSpace covers the \s class: ASCII whitespace, the Unicode space
// separators, and the BOM, but not an ordinary letter or the figure space's
// neighbours outside the range.
func TestIsSpace(t *testing.T) {
	for _, r := range []rune{' ', '\t', '\n', '\v', '\f', '\r', 0x00a0, 0x1680, 0x2000, 0x200a, 0x2028, 0x2029, 0x202f, 0x205f, 0x3000, 0xfeff} {
		if !IsSpace(r) {
			t.Errorf("IsSpace(%U) = false, want true", r)
		}
	}
	for _, r := range []rune{'a', '0', 0x200b, 0x1fff} {
		if IsSpace(r) {
			t.Errorf("IsSpace(%U) = true, want false", r)
		}
	}
}

// TestIsHexDigit covers the ASCII hex digit test both formatters and parsers
// share.
func TestIsHexDigit(t *testing.T) {
	for _, c := range []byte("0123456789abcdefABCDEF") {
		if !IsHexDigit(c) {
			t.Errorf("IsHexDigit(%q) = false, want true", c)
		}
	}
	for _, c := range []byte("gG zx-.") {
		if IsHexDigit(c) {
			t.Errorf("IsHexDigit(%q) = true, want false", c)
		}
	}
}

// TestRound covers Math.round's half-towards-positive-infinity rule, which
// differs from Go's math.Round for negative halves.
func TestRound(t *testing.T) {
	for _, tc := range []struct {
		in   float64
		want int
	}{{-0.5, 0}, {-1.5, -1}, {0.5, 1}, {1.4, 1}, {2.5, 3}} {
		if got := Round(tc.in); got != tc.want {
			t.Errorf("Round(%v) = %d, want %d", tc.in, got, tc.want)
		}
	}
	if got := RoundFloat(-2.5); got != -2 {
		t.Errorf("RoundFloat(-2.5) = %v, want -2", got)
	}
}
