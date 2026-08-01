package ops

import (
	"math"
	"testing"
)

// TestJSISOTimestamp covers the shim for JavaScript's Date#toISOString. The
// expectations come from Node itself. Operations only reach the four-figure
// years and the years past 9999, so the sign branches are exercised here
// directly rather than through a recipe.
func TestJSISOTimestamp(t *testing.T) {
	for _, tc := range []struct {
		name string
		ms   int64
		want string
	}{
		{"the epoch itself", 0, "1970-01-01T00:00:00.000Z"},
		{"a millisecond before it", -1, "1969-12-31T23:59:59.999Z"},
		{"a millisecond after it", 1, "1970-01-01T00:00:00.001Z"},
		{"a recent moment", 1774514156502, "2026-03-26T08:35:56.502Z"},
		{"the day the Gregorian calendar began", -12219292800000, "1582-10-15T00:00:00.000Z"},
		{"a leap day", 951782400000, "2000-02-29T00:00:00.000Z"},
		{"the last four-figure year", 253402300799999, "9999-12-31T23:59:59.999Z"},
		{"the first year that needs a sign", 253402300800000, "+010000-01-01T00:00:00.000Z"},
		{"the largest timestamp a v7 UUID can hold", 281474976710655, "+010889-08-02T05:31:50.655Z"},
		{"the first year, which needs no sign", -62167219200000, "0000-01-01T00:00:00.000Z"},
		{"a millisecond before the first year", -62167219200001, "-000001-12-31T23:59:59.999Z"},
		{"the whole of the year before that", -62198755200000, "-000001-01-01T00:00:00.000Z"},
		{"the furthest ahead a date may sit", 8640000000000000, "+275760-09-13T00:00:00.000Z"},
		{"the furthest back a date may sit", -8640000000000000, "-271821-04-20T00:00:00.000Z"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := jsISOTimestamp(tc.ms); got != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}

// TestJSToInt32 covers the shim for the conversion JavaScript puts a number
// through before a bitwise operator sees it: the fraction is dropped towards
// zero and what is left is read as a signed 32-bit value, so a large number
// wraps rather than saturating. The expectations come from Node.
func TestJSToInt32(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   float64
		want int64
	}{
		{"zero", 0, 0},
		{"a small whole number", 1, 1},
		{"a small negative whole number", -1, -1},
		{"a fraction dropped towards zero", 1.9, 1},
		{"a negative fraction dropped towards zero", -1.9, -1},
		{"a fraction below one", 0.5, 0},
		{"a negative fraction below one", -0.5, 0},
		{"the largest value that fits", 2147483647, 2147483647},
		{"one past it, which wraps to the lowest", 2147483648, -2147483648},
		{"the lowest value that fits", -2147483648, -2147483648},
		{"one below it, which wraps to the largest", -2147483649, 2147483647},
		{"a full 32 bits of ones", 4294967295, -1},
		{"a whole turn, which comes back to zero", 4294967296, 0},
		{"a whole turn and one", 4294967297, 1},
		{"a number far past 32 bits", 1e15, -1530494976},
		{"its negative", -1e15, 1530494976},
		{"a number past what a float counts exactly", 1e21, -559939584},
		{"its negative", -1e21, 559939584},
		{"the value a version 1 timestamp lands on", 32609465.9998, 32609465},
		{"no number at all", math.NaN(), 0},
		{"beyond counting", math.Inf(1), 0},
		{"beyond counting the other way", math.Inf(-1), 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := jsToInt32(tc.in); got != tc.want {
				t.Errorf("got %d, want %d", got, tc.want)
			}
		})
	}
}

// TestJSChr covers the shim for turning a number into the character it stands
// for. Anything above the basic plane is split into a surrogate pair and joined
// back up; anything else is taken as a sixteen-bit unit, so a negative number
// wraps round rather than being refused.
func TestJSChr(t *testing.T) {
	for _, tc := range []struct {
		name string
		code float64
		want string
	}{
		{"a letter", 65, "A"},
		{"nothing at all", 0, "\x00"},
		{"the last of the one-byte characters", 127, "\x7f"},
		{"a character that takes two bytes", 233, "é"},
		{"a character that takes three", 0x3042, "あ"},
		{"the last of the basic plane", 0xFFFF, "￿"},
		{"the first beyond it, which needs a pair", 0x10000, "\U00010000"},
		{"an emoji", 128512, "😀"},
		{"the last code point there is", 0x10FFFF, "\U0010FFFF"},
		{"one below nothing, which wraps to the top", -1, "￿"},
		{"three below", -3, "�"},
		{"a lone half of a pair, which cannot be written", 0xD800, "�"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := jsChr(tc.code); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// TestJSTrimSpace covers the shim for JavaScript's String#trim, which pares away
// a wider set of characters than Go's own does at one end and a narrower one at
// the other: a byte order mark goes, a zero width space stays.
func TestJSTrimSpace(t *testing.T) {
	for _, tc := range []struct{ name, in, want string }{
		{"nothing to take off", "c", "c"},
		{"a space at each end", " b ", "b"},
		{"every ordinary space character", " a\t\n\r\v\f ", "a"},
		{"a byte order mark, which counts as space", "\ufeff x \ufeff", "x"},
		{"an ideographic space", "\u2000\u2001\u2002\u2003\u2009\u3000e", "e"},
		{"a zero width space, which does not count", "\u200bd\u200b", "\u200bd\u200b"},
		{"space all the way through", " \t\n ", ""},
		{"nothing at all", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := jsTrimSpace(tc.in); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}
