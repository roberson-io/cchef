// Package jsnum reads and writes numbers the way JavaScript does.
//
// The writing half covers Number#toString ([Format]), String(x) ([String])
// and Math.round ([Round]); the reading half covers parseInt ([ParseInt],
// [ParseHex]) and parseFloat ([ParseFloat]). Operations need these to report
// the same figures CyberChef does, down to where the notation switches to
// exponential and which way a negative half rounds.
package jsnum

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/roberson-io/cchef/internal/jsonval"
)

// expMin and expMax bound the range in which JavaScript prints a number
// positionally; outside it, Number#toString uses exponential notation.
const (
	expMin = -6
	expMax = 21
)

// Format writes a float the way JavaScript's Number#toString does. Go's %g
// agrees on the shortest round-trip digits but switches to exponential notation
// at different thresholds and spells the exponent differently, so the digits are
// taken from Go and the presentation is rebuilt here.
func Format(f float64) string {
	if math.IsNaN(f) {
		return "NaN"
	}
	if math.IsInf(f, 1) {
		return "Infinity"
	}
	if math.IsInf(f, -1) {
		return "-Infinity"
	}
	if f == 0 {
		return "0"
	}

	sign := ""
	if f < 0 {
		sign, f = "-", -f
	}

	// 'e' gives the shortest round-trip digits with a known exponent, from
	// which either presentation can be built.
	mantissa, exp := splitExponential(strconv.FormatFloat(f, 'e', -1, 64))
	if exp < expMin || exp >= expMax {
		return sign + jsExponential(mantissa, exp)
	}
	return sign + jsPositional(mantissa, exp)
}

// splitExponential breaks Go's "d.ddde±dd" form into its digits and exponent.
func splitExponential(s string) (digits string, exp int) {
	mantissa, expPart, _ := strings.Cut(s, "e")
	exp, _ = strconv.Atoi(expPart)
	return strings.Replace(mantissa, ".", "", 1), exp
}

// jsExponential renders digits with the exponent, as "1e-7" or "1.5e+21".
func jsExponential(digits string, exp int) string {
	var b strings.Builder
	b.WriteByte(digits[0])
	if len(digits) > 1 {
		b.WriteByte('.')
		b.WriteString(digits[1:])
	}
	b.WriteByte('e')
	if exp < 0 {
		b.WriteByte('-')
		exp = -exp
	} else {
		b.WriteByte('+')
	}
	b.WriteString(strconv.Itoa(exp))
	return b.String()
}

// jsPositional renders digits without an exponent, padding with zeros on
// whichever side the decimal point falls.
func jsPositional(digits string, exp int) string {
	switch {
	case exp < 0:
		return "0." + strings.Repeat("0", -exp-1) + digits
	case exp+1 >= len(digits):
		return digits + strings.Repeat("0", exp+1-len(digits))
	default:
		return digits[:exp+1] + "." + digits[exp+1:]
	}
}

// String formats a float the way JavaScript's String(x) does: the non-finite
// values are spelled out, and finite ones use the JSON digits.
func String(f float64) string {
	switch {
	case math.IsNaN(f):
		return "NaN"
	case math.IsInf(f, 1):
		return "Infinity"
	case math.IsInf(f, -1):
		return "-Infinity"
	default:
		return jsonval.FormatNumber(f)
	}
}

// ParseInt mimics JS parseInt(s, base) for base 10/16: skips leading
// whitespace and an optional sign, then consumes valid leading digits. ok is
// false when no digits are consumed (JS returns NaN).
func ParseInt(s string, base int) (int, bool) {
	i := 0
	for i < len(s) {
		r, width := utf8.DecodeRuneInString(s[i:])
		if !IsSpace(r) {
			break
		}
		i += width
	}
	neg := false
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		neg = s[i] == '-'
		i++
	}
	start := i
	val := 0
	for i < len(s) {
		d := digitVal(s[i])
		if d < 0 || d >= base {
			break
		}
		val = val*base + d
		i++
	}
	if i == start {
		return 0, false
	}
	if neg {
		val = -val
	}
	return val, true
}

// digitVal returns the value of an ASCII hex digit, or -1 if not one.
func digitVal(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	}
	return -1
}

// IsSpace reports whether c is matched by JavaScript's \s class.
func IsSpace(c rune) bool {
	switch c {
	case 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x20,
		0x00A0, 0x1680, 0x2028, 0x2029, 0x202F, 0x205F, 0x3000, 0xFEFF:
		return true
	}
	return c >= 0x2000 && c <= 0x200A
}

// ParseHex is parseInt(s, 16): it reads the leading hex digits and yields NaN
// when there are none. A 0x prefix is allowed and skipped, as it is for that
// radix.
func ParseHex(s string) float64 {
	s = strings.TrimLeft(s, " \t\n\r\v\f")
	neg := false
	if strings.HasPrefix(s, "-") || strings.HasPrefix(s, "+") {
		neg = s[0] == '-'
		s = s[1:]
	}
	if len(s) > 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X') {
		s = s[2:]
	}
	end := 0
	for end < len(s) && IsHexDigit(s[end]) {
		end++
	}
	if end == 0 {
		return math.NaN()
	}
	v, err := strconv.ParseUint(s[:end], 16, 64)
	if err != nil {
		// Longer than 64 bits; accumulate in float64 the way JavaScript does.
		var f float64
		for i := range end {
			d, _ := strconv.ParseUint(string(s[i]), 16, 8)
			f = f*16 + float64(d)
		}
		if neg {
			return -f
		}
		return f
	}
	if neg {
		return -float64(v)
	}
	return float64(v)
}

// IsHexDigit reports whether c is an ASCII hexadecimal digit.
func IsHexDigit(c byte) bool {
	return c >= '0' && c <= '9' || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

var floatToken = regexp.MustCompile(`^[+-]?(Infinity|\d+\.?\d*([eE][+-]?\d+)?|\.\d+([eE][+-]?\d+)?)`)

// ParseFloat mirrors JavaScript's parseFloat: skip leading whitespace, then
// parse the longest valid numeric prefix, yielding NaN when none is present.
func ParseFloat(s string) float64 {
	s = strings.TrimLeft(s, " \t\n\r\f\v")
	m := floatToken.FindString(s)
	if m == "" {
		return math.NaN()
	}
	f, _ := strconv.ParseFloat(m, 64)
	return f
}

// Round rounds half towards positive infinity, as JavaScript's Math.round
// does. Go's math.Round rounds halves away from zero, which differs for
// negatives such as -0.5.
func Round(f float64) int {
	return int(RoundFloat(f))
}

// RoundFloat is Round without the narrowing to int.
func RoundFloat(f float64) float64 {
	return math.Floor(f + 0.5)
}
