package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
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
