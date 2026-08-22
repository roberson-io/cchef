package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

// modExpRecipe builds a recipe with the base, modulus and exponent given, in
// the order the operation declares them.
func modExpRecipe(base, modulus, exponent string) core.Recipe {
	return core.Recipe{{Op: "Modular Exponentiation", Args: []any{base, modulus, exponent}}}
}

// TestModularExponentiationFixtures covers CyberChef's own cases
// (CyberChef's tests/operations/tests/ModularExponentiation.mjs).
func TestModularExponentiationFixtures(t *testing.T) {
	runCases(t, []opCase{
		{"basic example (2^10 mod 1000)", "", "24", modExpRecipe("2", "1000", "10")},
		{"small values (3^5 mod 7)", "", "5", modExpRecipe("3", "7", "5")},
		{"exponent zero", "", "1", modExpRecipe("999", "100", "0")},
		{"base one", "", "1", modExpRecipe("1", "1000", "999")},
		{"hexadecimal arguments", "", "256", modExpRecipe("0x10", "1000", "0x2")},
		{"base taken from the input", "5", "6", modExpRecipe("", "7", "3")},
		{"exponent taken from the input", "4", "5", modExpRecipe("2", "11", "")},
		{"RSA-like example", "", "561", modExpRecipe("123", "1000", "456")},
		{"large base and exponent", "", "560583526", modExpRecipe("123456789", "1000000007", "65537")},
		{
			"crypto-sized numbers", "", "1",
			modExpRecipe(
				"12345678901234567890123456789012345678901234567890",
				"99999999999999999999999999999999999999999999999999",
				"0",
			),
		},
		{"Fermat's little theorem", "", "1", modExpRecipe("3", "11", "10")},
	})
}

// TestModularExponentiationSigns covers the values a mathematical treatment
// would not produce. CyberChef reduces with JavaScript's remainder operator,
// whose result takes the sign of the dividend, so a negative base gives a
// negative residue rather than the one in [0, modulus). Its loop runs while the
// exponent is above zero, so a negative exponent does no work at all and the
// result is 1 rather than a modular inverse. Both are what a running CyberChef
// returns for these inputs.
func TestModularExponentiationSigns(t *testing.T) {
	runCases(t, []opCase{
		{"negative base keeps its sign", "x", "-6", modExpRecipe("-5", "7", "3")},
		{"negative modulus", "x", "6", modExpRecipe("5", "-7", "3")},
		{"negative exponent does nothing", "x", "1", modExpRecipe("5", "7", "-3")},
		{"uppercase hex prefix", "x", "25", modExpRecipe("0XFF", "1000", "2")},
	})
}

// TestModularExponentiationErrors covers the messages, which are CyberChef's
// verbatim.
func TestModularExponentiationErrors(t *testing.T) {
	for _, tc := range []struct{ name, input, base, mod, exp, want string }{
		{"modulus missing", "x", "5", "", "3", "Modulus must be defined"},
		{"modulus blank", "x", "5", "   ", "3", "Modulus must be defined"},
		{"modulus zero", "x", "5", "0", "3", "Modulus must be greater than zero"},
		{
			"base and exponent both empty with input", "9", "", "7", "",
			"Ambiguous input: specify either Base or Exponent when using Input",
		},
		{"base and exponent both empty, no input", "", "", "7", "", "Base and Exponent must be defined"},
		{"base not a number", "x", "abc", "7", "3", "Base must be decimal or hex (0x...)"},
		{"exponent not a number", "x", "5", "7", "zzz", "Exponent must be decimal or hex (0x...)"},
		{"modulus not a number", "x", "5", "nope", "3", "Modulus must be decimal or hex (0x...)"},
		{"base empty and input blank", "   ", "", "7", "3", "Base must be defined"},
		{"exponent empty and input blank", "   ", "5", "7", "", "Exponent must be defined"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := runOp(t, "Modular Exponentiation", tc.input, tc.base, tc.mod, tc.exp)
			if err == nil {
				t.Fatalf("args %q/%q/%q were accepted", tc.base, tc.mod, tc.exp)
			}
			if err.Error() != tc.want {
				t.Errorf("error = %q, want %q", err, tc.want)
			}
		})
	}
}
