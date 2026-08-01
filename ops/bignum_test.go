package ops

import (
	"math/big"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// TestBigNumEdgeCases exercises the bigNum arithmetic/formatting branches that
// the fixture tests don't reach: exponential notation, ±Infinity and NaN
// propagation, signed zero, and non-decimal fractional parsing. Every expected
// value was taken from the CyberChef-server oracle.
func TestBigNumEdgeCases(t *testing.T) {
	sp := func(op string, args ...any) core.Recipe {
		return core.Recipe{{Op: op, Args: append([]any{"Space"}, args...)}}
	}
	runCases(t, []opCase{
		// Exponential notation: exponent >= 21 or <= -7.
		{"exp: 1e21", "1000000000000000000000 1", "1e+21", sp("Multiply")},
		{"exp: 3e21", "100000000000000000000 30", "3e+21", sp("Multiply")},
		{"exp: 1.23e22 mantissa", "12300000000000000000000 1", "1.23e+22", sp("Multiply")},
		{"exp: long mantissa", "10000000000000000000000 1", "1.0000000000000000000001e+22", sp("Sum")},
		{"exp: 1e-7", "1 10000000", "1e-7", sp("Divide")},
		{"exp: 1e-9", "1 1000000000", "1e-9", sp("Divide")},
		{"fixed boundary 1e20", "100000000000000000000 1", "100000000000000000000", sp("Multiply")},
		{"fixed boundary 1e-6", "1 1000000", "0.000001", sp("Divide")},

		// ±Infinity and NaN propagation.
		{"sum inf+finite", "Infinity 1", "Infinity", sp("Sum")},
		{"sum inf+inf", "Infinity Infinity", "Infinity", sp("Sum")},
		{"sum -inf+-inf", "-Infinity -Infinity", "-Infinity", sp("Sum")},
		{"sum inf+-inf = NaN", "Infinity -Infinity", "NaN", sp("Sum")},
		{"mul inf*2", "Infinity 2", "Infinity", sp("Multiply")},
		{"mul inf*-2", "Infinity -2", "-Infinity", sp("Multiply")},
		{"mul inf*0 = NaN", "Infinity 0", "NaN", sp("Multiply")},
		{"div 1/0 = inf", "1 0", "Infinity", sp("Divide")},
		{"div -1/0 = -inf", "-1 0", "-Infinity", sp("Divide")},
		{"div 0/0 = NaN", "0 0", "NaN", sp("Divide")},
		{"div inf/inf = NaN", "Infinity Infinity", "NaN", sp("Divide")},
		{"div inf/finite", "Infinity 2", "Infinity", sp("Divide")},
		{"div -inf/finite", "-Infinity 2", "-Infinity", sp("Divide")},
		{"div finite/inf = 0", "6 Infinity", "0", sp("Divide")},
		{"div -finite/inf = -0", "-6 Infinity", "-0", sp("Divide")},
		{"median with infinities", "Infinity 1 -Infinity", "1", sp("Median")},
		{"median with equal infinities", "Infinity Infinity 5", "Infinity", sp("Median")},
		{"subtract finite-inf = -inf", "5 Infinity", "-Infinity", sp("Subtract")},
		{"subtract inf-finite = inf", "Infinity 5", "Infinity", sp("Subtract")},
		{"subtract inf--inf = inf", "Infinity -Infinity", "Infinity", sp("Subtract")},

		// Signed zero.
		{"mul 5*-0 = -0", "5 -0", "-0", sp("Multiply")},
		{"mul -5*0 = -0", "-5 0", "-0", sp("Multiply")},
		{"mul -5*-5*0 = 0", "-5 -5 0", "0", sp("Multiply")},
		{"sum -0 = -0", "-0", "-0", sp("Sum")},
		{"sum 0+-0 = 0", "0 -0", "0", sp("Sum")},
		{"subtract 5-5 = 0", "5 5", "0", sp("Subtract")},

		// Non-decimal parsing, including a fractional hex value.
		{"parse hex fraction 0x1.8", "0x1.8", "1.5", sp("Sum")},
		{"parse octal 0o17", "0o17", "15", sp("Sum")},
		{"parse binary 0b101", "0b101", "5", sp("Sum")},

		// Standard deviation of a single value is exactly zero.
		{"stddev single value", "5", "0", sp("Standard Deviation")},
	})
}

func TestBignumMoreBranches(t *testing.T) {
	if sign3(5) != 1 {
		t.Fatalf("sign3(5) = %d, want 1", sign3(5))
	}
	if bnNaN.sign() != 0 {
		t.Fatalf("NaN.sign() = %d, want 0", bnNaN.sign())
	}
	if !bnNaN.negate().nan {
		t.Fatal("NaN.negate() is not NaN")
	}
	if got := formatBigRat(big.NewRat(0, 1)); got != "0" {
		t.Fatalf("formatBigRat(0) = %q, want 0", got)
	}
	if got := formatFixed("100", 2); got != "1" {
		t.Fatalf("formatFixed(100,2) = %q, want 1", got)
	}
	if got := formatBigRat(round20(big.NewRat(-1, 3))); got != "-0.33333333333333333333" {
		t.Fatalf("round20(-1/3) = %q", got)
	}
	if got := formatBigRat(sqrtRound20(big.NewRat(2, 1))); got != "1.4142135623730950488" {
		t.Fatalf("sqrtRound20(2) = %q", got)
	}
	// All-zero digits normalise to "0" (the coeff=="" branch).
	if a, b := formatExponential("000", 21), formatExponential("0", 21); a != b {
		t.Fatalf("formatExponential(000) = %q, formatExponential(0) = %q", a, b)
	}
}

// TestBignumRounding covers the negative-halfway arm of round20 and the round-up
// arm of sqrtRound20 (both direct, since no fixture drives these exact cases).
func TestBignumRounding(t *testing.T) {
	// -1/(2e20) is exactly halfway at the 20th decimal and rounds away from zero.
	twiceScale := new(big.Int).Mul(big.NewInt(2), new(big.Int).Exp(big.NewInt(10), big.NewInt(20), nil))
	negHalfway := new(big.Rat).SetFrac(big.NewInt(-1), twiceScale)
	if got := formatBigRat(round20(negHalfway)); got != "-1e-20" {
		t.Fatalf("round20(neg halfway) = %q", got)
	}
	// sqrt(3) rounds up at the 21st decimal.
	if got := formatBigRat(sqrtRound20(big.NewRat(3, 1))); got != "1.73205080756887729353" {
		t.Fatalf("sqrtRound20(3) = %q", got)
	}
	// A perfect square is exact (no rounding).
	if got := formatBigRat(sqrtRound20(big.NewRat(4, 1))); got != "2" {
		t.Fatalf("sqrtRound20(4) = %q", got)
	}
}

// decNumReOld and parseDecimalOld are an exact copy of the pre-refactor
// regex-gated parseDecimal, kept here as the behavioural reference for
// TestParseDecimalEquivalence.
var decNumReOld = regexp.MustCompile(`^(?:\d+\.?\d*|\.\d+)(?:[eE][+-]?\d+)?$`)

func parseDecimalOld(sign, body string) (bigNum, bool) {
	if !decNumReOld.MatchString(body) {
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
	if digits == "" {
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

// TestParseDecimalEquivalence asserts the refactored parseDecimal accepts/rejects
// exactly the same strings as the old regex-gated version, and produces the same
// value, over a large structured + randomised corpus.
func TestParseDecimalEquivalence(t *testing.T) {
	structured := []string{
		"", "0", "5", "05", "5.", ".5", "1.5", "1.", ".", "00.00", "0.0e0",
		"1e5", "1E5", "1e+5", "1e-5", "1e0", "1e00", "1.e5", ".e5",
		"1e", "e5", "1..5", "1.2.3", "1e2e3", "1e5.5", "1e+", "1e-",
		"+5", "-5", "+.5", "5 ", " 5", "1 5", "1_000", "0x1", "abc", "5x",
		"1e99999999999999999999", // exponent overflows Atoi
	}
	agree := func(body string) {
		for _, sign := range []string{"", "-"} {
			gotV, gotOK := parseDecimal(sign, body)
			wantV, wantOK := parseDecimalOld(sign, body)
			if gotOK != wantOK {
				t.Fatalf("parseDecimal(%q,%q) ok=%v, old ok=%v", sign, body, gotOK, wantOK)
			}
			if gotOK && (gotV.neg != wantV.neg || gotV.val.Cmp(wantV.val) != 0) {
				t.Fatalf("parseDecimal(%q,%q) value=%v/%v, old=%v/%v", sign, body, gotV.val, gotV.neg, wantV.val, wantV.neg)
			}
		}
	}
	for _, s := range structured {
		agree(s)
	}
	// Randomised strings over the number-relevant alphabet, length 0-5 (bounded so
	// any surviving exponent stays small enough to expand quickly).
	rng := rand.New(rand.NewSource(1))
	alphabet := []byte("0123456789.eE+-")
	for range 20000 {
		n := rng.Intn(6)
		b := make([]byte, n)
		for j := range b {
			b[j] = alphabet[rng.Intn(len(alphabet))]
		}
		agree(string(b))
	}
}

// TestBigNumMod covers the modulo, whose result takes the sign of the dividend
// rather than the divisor, and which works on decimals as well as whole
// numbers. The values follow bignumber.js's own mod.
func TestBigNumMod(t *testing.T) {
	num := func(s string) bigNum {
		n, ok := parseBigNum(s)
		if !ok {
			t.Fatalf("cannot read %q", s)
		}
		return n
	}
	cases := []struct{ a, b, want string }{
		{"15", "3", "0"},
		{"4", "3", "1"},
		{"7", "3", "1"},
		{"10", "9", "1"},
		// The sign follows the dividend.
		{"-15", "3", "0"},
		{"-8", "3", "-2"},
		{"25", "3", "1"},
		{"-10", "3", "-1"},
		{"8", "-3", "2"},
		// Decimals keep their fractional part.
		{"10.5", "3", "1.5"},
		{"15.7", "3", "0.7"},
		{"8.2", "3", "2.2"},
		// A dividend smaller than the divisor is returned unchanged.
		{"123456789012345", "987654321098765432", "123456789012345"},
	}
	for _, c := range cases {
		if got := num(c.a).mod(num(c.b)).String(); got != c.want {
			t.Errorf("%s mod %s = %s, want %s", c.a, c.b, got, c.want)
		}
	}
}

// TestBigNumModSpecials covers the values that are not ordinary numbers.
func TestBigNumModSpecials(t *testing.T) {
	one, _ := parseBigNum("1")
	zero, _ := parseBigNum("0")
	if got := one.mod(zero); !got.nan {
		t.Errorf("anything mod zero should not be a number, got %s", got)
	}
	if got := bnNaN.mod(one); !got.nan {
		t.Errorf("not-a-number mod one should not be a number, got %s", got)
	}
	if got := one.mod(bnNaN); !got.nan {
		t.Errorf("one mod not-a-number should not be a number, got %s", got)
	}
	if got := (bigNum{inf: 1}).mod(one); !got.nan {
		t.Errorf("infinity mod one should not be a number, got %s", got)
	}
	// A finite number modulo infinity is the number itself.
	if got := one.mod(bigNum{inf: 1}).String(); got != "1" {
		t.Errorf("one mod infinity = %s, want 1", got)
	}
}
