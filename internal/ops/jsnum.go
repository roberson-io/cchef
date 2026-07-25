package ops

import (
	"math"
	"strconv"
	"strings"
)

// jsExpMin and jsExpMax bound the range in which JavaScript prints a number
// positionally; outside it, Number#toString uses exponential notation.
const (
	jsExpMin = -6
	jsExpMax = 21
)

// jsNum formats a float the way JavaScript's Number#toString does. Go's %g
// agrees on the shortest round-trip digits but switches to exponential notation
// at different thresholds and spells the exponent differently, so the digits are
// taken from Go and the presentation is rebuilt here.
func jsNum(f float64) string {
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
	if exp < jsExpMin || exp >= jsExpMax {
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

// jsRound rounds half towards positive infinity, as JavaScript's Math.round
// does. Go's math.Round rounds halves away from zero, which differs for
// negatives such as -0.5.
func jsRound(f float64) int {
	return int(jsRoundFloat(f))
}

// jsRoundFloat is jsRound without the narrowing to int.
func jsRoundFloat(f float64) float64 {
	return math.Floor(f + 0.5)
}
