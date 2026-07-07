package ops

import (
	"math/big"
	"strconv"
	"strings"
)

// bigNum is a faithful subset of CyberChef's bignumber.js: an arbitrary-precision
// value that may also be NaN or ±Infinity. Finite values are held exactly as a
// big.Rat. Division, mean and square-root round to bignumber's default
// DECIMAL_PLACES = 20 using ROUND_HALF_UP (ties away from zero); addition,
// subtraction and multiplication are exact. Its String method reproduces
// bignumber's toString, including the exponential-notation thresholds.
type bigNum struct {
	nan bool
	inf int      // 0 = finite, +1 = +Infinity, -1 = -Infinity
	val *big.Rat // finite value (nil when nan/inf)
	neg bool     // sign flag; only meaningful for a zero val, to model bignumber's signed zero (-0)
}

// bnDecimalPlaces is bignumber.js's default DECIMAL_PLACES.
const bnDecimalPlaces = 20

var (
	bnNaN     = bigNum{nan: true}
	bnTen     = big.NewInt(10)
	bnScale20 = new(big.Int).Exp(bnTen, big.NewInt(bnDecimalPlaces), nil)
)

// finite builds a finite bigNum from a big.Rat, taking its sign from the value
// (a zero is therefore positive; use an explicit neg field for -0).
func finite(r *big.Rat) bigNum { return bigNum{val: r, neg: r.Sign() < 0} }

// fromInt builds a finite bigNum from an int.
func fromInt(n int) bigNum { return finite(new(big.Rat).SetInt64(int64(n))) }

// parseBigNum parses a single token the way bignumber.js's constructor does. The
// bool result is false when the token is NaN (not a valid number), so callers can
// exclude it — mirroring createNumArray's `if (!num.isNaN())` filter. "Infinity"
// / "-Infinity" parse successfully to ±Infinity.
func parseBigNum(s string) (bigNum, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return bnNaN, false
	}
	sign := ""
	body := s
	if body[0] == '+' || body[0] == '-' {
		sign = string(body[0])
		body = body[1:]
	}
	if body == "Infinity" {
		if sign == "-" {
			return bigNum{inf: -1}, true
		}
		return bigNum{inf: 1}, true
	}
	if len(body) >= 2 && body[0] == '0' {
		switch body[1] {
		case 'x', 'X':
			return parseRadix(sign, body[2:], 16)
		case 'o', 'O':
			return parseRadix(sign, body[2:], 8)
		case 'b', 'B':
			return parseRadix(sign, body[2:], 2)
		}
	}
	return parseDecimal(sign, body)
}

// parseRadix parses an integer (with optional fractional part) in the given base,
// matching bignumber.js's support for 0x / 0o / 0b prefixed values.
func parseRadix(sign, body string, base int) (bigNum, bool) {
	intStr, fracStr := body, ""
	if before, after, ok := strings.Cut(body, "."); ok {
		intStr, fracStr = before, after
	}
	if intStr == "" && fracStr == "" {
		return bnNaN, false
	}
	r := new(big.Rat)
	if intStr != "" {
		iv, ok := new(big.Int).SetString(intStr, base)
		if !ok {
			return bnNaN, false
		}
		r.SetInt(iv)
	}
	if fracStr != "" {
		fv, ok := new(big.Int).SetString(fracStr, base)
		if !ok {
			return bnNaN, false
		}
		den := new(big.Int).Exp(big.NewInt(int64(base)), big.NewInt(int64(len(fracStr))), nil)
		r.Add(r, new(big.Rat).SetFrac(fv, den))
	}
	if sign == "-" {
		r.Neg(r)
	}
	return bigNum{val: r, neg: sign == "-"}, true
}

// parseDecimal parses a base-10 number with optional fraction and exponent. It
// validates by parsing rather than a pre-check: a leading sign is rejected (it
// would be a doubled sign, since parseBigNum already stripped one), and the
// exponent Atoi, empty-digit and SetString steps below reject any remaining
// malformed input — matching bignumber.js's NaN result.
func parseDecimal(sign, body string) (bigNum, bool) {
	if body == "" || body[0] == '+' || body[0] == '-' {
		return bnNaN, false
	}
	mant := body
	exp := 0
	if i := strings.IndexAny(body, "eE"); i >= 0 {
		mant = body[:i]
		e, err := strconv.Atoi(body[i+1:])
		if err != nil {
			return bnNaN, false
		}
		exp = e
	}
	intPart, fracPart := mant, ""
	if before, after, ok := strings.Cut(mant, "."); ok {
		intPart, fracPart = before, after
	}
	digits := intPart + fracPart
	// A leading sign here (e.g. ".-99", where the integer part is empty) would be
	// accepted by SetString; reject it to match bignumber.js.
	if digits == "" || digits[0] == '+' || digits[0] == '-' {
		return bnNaN, false
	}
	num, ok := new(big.Int).SetString(digits, 10)
	if !ok {
		return bnNaN, false
	}
	r := new(big.Rat).SetInt(num)
	scale := exp - len(fracPart)
	pow := new(big.Int).Exp(bnTen, big.NewInt(int64(abs(scale))), nil)
	if scale >= 0 {
		r.Mul(r, new(big.Rat).SetInt(pow))
	} else {
		r.Quo(r, new(big.Rat).SetInt(pow))
	}
	if sign == "-" {
		r.Neg(r)
	}
	return bigNum{val: r, neg: sign == "-"}, true
}

// plus returns a + b with bignumber's ±Infinity/NaN propagation.
func (a bigNum) plus(b bigNum) bigNum {
	if a.nan || b.nan {
		return bnNaN
	}
	if a.inf != 0 || b.inf != 0 {
		switch {
		case a.inf != 0 && b.inf != 0:
			if a.inf != b.inf {
				return bnNaN // +Inf + -Inf
			}
			return a
		case a.inf != 0:
			return a
		default:
			return b
		}
	}
	s := new(big.Rat).Add(a.val, b.val)
	if s.Sign() == 0 {
		// bignumber: a zero sum is -0 only when both addends are negative zero.
		return bigNum{val: s, neg: a.neg && b.neg}
	}
	return finite(s)
}

// negate returns -a (flipping the sign, including for signed zero).
func (a bigNum) negate() bigNum {
	switch {
	case a.nan:
		return bnNaN
	case a.inf != 0:
		return bigNum{inf: -a.inf}
	default:
		return bigNum{val: new(big.Rat).Neg(a.val), neg: !a.neg}
	}
}

// minus returns a - b.
func (a bigNum) minus(b bigNum) bigNum { return a.plus(b.negate()) }

// times returns a * b.
func (a bigNum) times(b bigNum) bigNum {
	if a.nan || b.nan {
		return bnNaN
	}
	if a.inf != 0 || b.inf != 0 {
		if a.isZero() || b.isZero() {
			return bnNaN // Infinity * 0
		}
		if a.sign()*b.sign() < 0 {
			return bigNum{inf: -1}
		}
		return bigNum{inf: 1}
	}
	// Sign is the XOR of the operand signs, giving -0 when exactly one is negative.
	return bigNum{val: new(big.Rat).Mul(a.val, b.val), neg: a.neg != b.neg}
}

// div returns a / b, rounded to 20 decimal places (ROUND_HALF_UP) for finite
// operands, matching bignumber.js. The result sign is always the XOR of the
// operand signs, which is what makes bignumber's signed zero (and ±Infinity)
// come out right for divisions by/into zero and Infinity.
func (a bigNum) div(b bigNum) bigNum {
	if a.nan || b.nan {
		return bnNaN
	}
	resNeg := a.negSign() != b.negSign()
	switch {
	case a.inf != 0 && b.inf != 0:
		return bnNaN // Infinity / Infinity
	case a.inf != 0: // Infinity / finite
		return infWithSign(resNeg)
	case b.inf != 0: // finite / Infinity -> ±0
		return bigNum{val: new(big.Rat), neg: resNeg}
	case b.isZero():
		if a.isZero() {
			return bnNaN // 0 / 0
		}
		return infWithSign(resNeg)
	}
	return bigNum{val: round20(new(big.Rat).Quo(a.val, b.val)), neg: resNeg}
}

// infWithSign returns -Infinity when neg is true, otherwise +Infinity.
func infWithSign(neg bool) bigNum {
	if neg {
		return bigNum{inf: -1}
	}
	return bigNum{inf: 1}
}

func (a bigNum) isZero() bool { return a.inf == 0 && !a.nan && a.val.Sign() == 0 }

// negSign reports whether the value carries a negative sign, treating -0 and
// -Infinity as negative.
func (a bigNum) negSign() bool {
	if a.inf != 0 {
		return a.inf < 0
	}
	return a.neg
}

// cmp orders two values: -1 if a < b, 0 if equal, +1 if a > b. Any ±Infinity
// ranks by its inf field (-Infinity < every finite value < +Infinity); two finite
// values compare exactly.
func (a bigNum) cmp(b bigNum) int {
	if a.inf != 0 || b.inf != 0 {
		return sign3(a.inf - b.inf)
	}
	return a.val.Cmp(b.val)
}

func sign3(n int) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	default:
		return 0
	}
}

func (a bigNum) sign() int {
	switch {
	case a.inf != 0:
		return a.inf
	case a.nan:
		return 0
	default:
		return a.val.Sign()
	}
}

// round20 rounds an exact rational to 20 decimal places, ties away from zero.
func round20(r *big.Rat) *big.Rat {
	scaled := new(big.Rat).Mul(r, new(big.Rat).SetInt(bnScale20))
	num, den := scaled.Num(), scaled.Denom() // den > 0
	q := new(big.Int).Quo(num, den)          // truncate toward zero
	rem := new(big.Int).Rem(num, den)
	twiceRem := new(big.Int).Abs(rem)
	twiceRem.Lsh(twiceRem, 1)
	if twiceRem.Cmp(den) >= 0 { // halfway or beyond -> away from zero
		if num.Sign() < 0 {
			q.Sub(q, big.NewInt(1))
		} else {
			q.Add(q, big.NewInt(1))
		}
	}
	return new(big.Rat).SetFrac(q, bnScale20)
}

// sqrtRound20 returns sqrt(x) for a non-negative rational, rounded to 20 decimal
// places (ROUND_HALF_UP), matching bignumber.js's sqrt.
func sqrtRound20(x *big.Rat) *big.Rat {
	p, q := x.Num(), x.Denom() // p >= 0, q > 0
	// A = p * (10^20)^2; N ≈ sqrt(A / q) = sqrt(x) * 10^20.
	a := new(big.Int).Mul(p, new(big.Int).Mul(bnScale20, bnScale20))
	fl := new(big.Int).Quo(a, q)
	n0 := new(big.Int).Sqrt(fl) // floor(sqrt(A/q))
	// Round up when (n0 + 0.5)^2 <= A/q, i.e. (2n0+1)^2 * q <= 4A.
	t := new(big.Int).Lsh(n0, 1)
	t.Add(t, big.NewInt(1))
	lhs := new(big.Int).Mul(t, t)
	lhs.Mul(lhs, q)
	rhs := new(big.Int).Lsh(a, 2)
	n := n0
	if lhs.Cmp(rhs) <= 0 {
		n = new(big.Int).Add(n0, big.NewInt(1))
	}
	return new(big.Rat).SetFrac(n, bnScale20)
}

// String reproduces bignumber.js's toString for the value, including its default
// exponential-notation thresholds (exponent ≤ -7 or ≥ 21).
func (a bigNum) String() string {
	switch {
	case a.nan:
		return "NaN"
	case a.inf > 0:
		return "Infinity"
	case a.inf < 0:
		return "-Infinity"
	}
	if a.val.Sign() == 0 {
		if a.neg {
			return "-0"
		}
		return "0"
	}
	return formatBigRat(a.val)
}

// formatBigRat formats a terminating rational the way bignumber.js prints it. The
// arithmetic operations only ever produce terminating decimals (exact results, or
// values rounded to 20 decimal places), so the denominator is always 2^x·5^y.
func formatBigRat(r *big.Rat) string {
	if r.Sign() == 0 {
		return "0"
	}
	neg := r.Sign() < 0
	num := new(big.Int).Abs(r.Num())
	den := new(big.Int).Set(r.Denom())

	// Factor the denominator into 2^a · 5^b (it has no other prime factors for a
	// terminating decimal), then rescale the numerator so value = M / 10^k.
	a := stripFactor(den, 2)
	b := stripFactor(den, 5)
	k := max(b, a)
	mult := new(big.Int).Mul(
		new(big.Int).Exp(big.NewInt(2), big.NewInt(int64(k-a)), nil),
		new(big.Int).Exp(big.NewInt(5), big.NewInt(int64(k-b)), nil),
	)
	m := new(big.Int).Mul(num, mult)

	digits := m.String()
	nd := len(digits)
	exp := (nd - 1) - k // decimal exponent of the leading digit

	var out string
	if exp <= -7 || exp >= 21 {
		out = formatExponential(digits, exp)
	} else {
		out = formatFixed(digits, k)
	}
	if neg {
		return "-" + out
	}
	return out
}

// stripFactor divides out every factor p from n in place and returns the count.
func stripFactor(n *big.Int, p int64) int {
	pb := big.NewInt(p)
	mod := new(big.Int)
	count := 0
	for {
		mod.Mod(n, pb)
		if mod.Sign() != 0 {
			return count
		}
		n.Quo(n, pb)
		count++
	}
}

// formatFixed renders digits/10^k in fixed-point notation, trimming trailing
// fractional zeros.
func formatFixed(digits string, k int) string {
	if k == 0 {
		return digits
	}
	nd := len(digits)
	var intPart, fracPart string
	if nd > k {
		intPart, fracPart = digits[:nd-k], digits[nd-k:]
	} else {
		intPart = "0"
		fracPart = strings.Repeat("0", k-nd) + digits
	}
	fracPart = strings.TrimRight(fracPart, "0")
	if fracPart == "" {
		return intPart
	}
	return intPart + "." + fracPart
}

// formatExponential renders the value in bignumber's exponential notation, e.g.
// "1e+21", "1.23e+22", "1e-7".
func formatExponential(digits string, exp int) string {
	coeff := strings.TrimRight(digits, "0")
	if coeff == "" {
		coeff = "0"
	}
	var mant string
	if len(coeff) == 1 {
		mant = coeff
	} else {
		mant = coeff[:1] + "." + coeff[1:]
	}
	sign := "+"
	if exp < 0 {
		sign = "-"
	}
	return mant + "e" + sign + strconv.Itoa(abs(exp))
}
