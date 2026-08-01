package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// TestExtendedGCDFixtures runs CyberChef's ExtendedGCD.mjs cases.
func TestExtendedGCDFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"Extended GCD: coprime numbers (3, 11)",
			"",
			"gcd: 1\n\nBezout coefficients:\nx = 4\ny = -1\n\n",
			core.Recipe{{Op: "Extended GCD", Args: []any{"3", "11"}}},
		},
		{
			"Extended GCD: non-coprime numbers (240, 46)",
			"",
			"gcd: 2\n\nBezout coefficients:\nx = -9\ny = 47\n\n",
			core.Recipe{{Op: "Extended GCD", Args: []any{"240", "46"}}},
		},
		{
			"Extended GCD: with zero (17, 0)",
			"",
			"gcd: 17\n\nBezout coefficients:\nx = 1\ny = 0\n\n",
			core.Recipe{{Op: "Extended GCD", Args: []any{"17", "0"}}},
		},
		{
			"Extended GCD: hexadecimal input (0xFF, 0x11)",
			"",
			"gcd: 17\n\nBezout coefficients:\nx = 0\ny = 1\n\n",
			core.Recipe{{Op: "Extended GCD", Args: []any{"0xFF", "0x11"}}},
		},
		{
			"Extended GCD: using input field for value a",
			"42",
			"gcd: 7\n\nBezout coefficients:\nx = 1\ny = -1\n\n",
			core.Recipe{{Op: "Extended GCD", Args: []any{"", "35"}}},
		},
		{
			"Extended GCD: large numbers",
			"",
			"gcd: 2\n\nBezout coefficients:\nx = 12703973750415151\ny = -1577756566311408967124629843\n\n",
			core.Recipe{{Op: "Extended GCD", Args: []any{"123456789012345678901234567890", "994064509324197316"}}},
		},
	})
}

// TestExtendedGCDInputAndRefusals covers taking a value from the input, and
// each way the two values can fail to be given or read.
func TestExtendedGCDInputAndRefusals(t *testing.T) {
	// Either argument may be left blank to take that value from the input.
	got, err := runOp(t, "Extended GCD", "35", "42", "")
	if err != nil {
		t.Fatalf("value b from the input: %v", err)
	}
	if !strings.Contains(got, "gcd: 7") {
		t.Errorf("value b from the input gave %q", got)
	}

	for _, c := range []struct{ name, input, a, b, want string }{
		{"neither given", "", "", "", "Value a and Value b must be defined"},
		{"a missing and no input", "", "", "11", "Value a must be defined"},
		{"b missing and no input", "", "11", "", "Value b must be defined"},
		{"a will not read", "", "twelve", "11", "Value a must be decimal or hex"},
		{"b will not read", "", "11", "twelve", "Value b must be decimal or hex"},
		{"a blank input will not read", "twelve", "", "11", "Value a must be decimal or hex"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := runOp(t, "Extended GCD", c.input, c.a, c.b)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("got %v, want it to mention %q", err, c.want)
			}
		})
	}

	// Whitespace counts as blank, so a padded argument still falls back.
	if _, err := runOp(t, "Extended GCD", "", "   ", "   "); err == nil {
		t.Error("arguments of only spaces should count as missing")
	}
}
