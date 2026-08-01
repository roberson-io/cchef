package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// TestMODFixtures runs CyberChef's MOD.mjs cases.
func TestMODFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"MOD: Basic modulo operation",
			"15 4 7",
			"0 1 1",
			core.Recipe{{Op: "MOD", Args: []any{3.0, "Space"}}},
		},
		{
			"MOD: Single number",
			"10",
			"1",
			core.Recipe{{Op: "MOD", Args: []any{3.0, "Space"}}},
		},
		{
			"MOD: Comma-separated numbers",
			"15,8,23,16,5",
			"1 1 2 2 5",
			core.Recipe{{Op: "MOD", Args: []any{7.0, "Comma"}}},
		},
		{
			"MOD: Line feed separated numbers",
			"25\n13\n44\n7",
			"0 3 4 2",
			core.Recipe{{Op: "MOD", Args: []any{5.0, "Line feed"}}},
		},
		{
			"MOD: Large numbers",
			"123456789012345 987654321098765",
			"123456789012345 987654321098765",
			core.Recipe{{Op: "MOD", Args: []any{1234567890123456.0, "Space"}}},
		},
		{
			"MOD: Mixed with non-numeric values",
			"15 abc 4 def 7 xyz 23",
			"0 1 1 2",
			core.Recipe{{Op: "MOD", Args: []any{3.0, "Space"}}},
		},
		{
			"MOD: Decimal numbers",
			"10.5 15.7 8.2",
			"1.5 0.7 2.2",
			core.Recipe{{Op: "MOD", Args: []any{3.0, "Space"}}},
		},
		{
			"MOD: Negative numbers",
			"-15 -8 25 -10",
			"0 -2 1 -1",
			core.Recipe{{Op: "MOD", Args: []any{3.0, "Space"}}},
		},
		{
			"MOD: Zero in input",
			"0 5 10 15 20",
			"0 2 1 0 2",
			core.Recipe{{Op: "MOD", Args: []any{3.0, "Space"}}},
		},
		{
			"MOD: Modulus of 2 (even/odd check)",
			"1 2 3 4 5 6 7 8 9 10",
			"1 0 1 0 1 0 1 0 1 0",
			core.Recipe{{Op: "MOD", Args: []any{2.0, "Space"}}},
		},
		{
			"MOD: Numbers with extra whitespace",
			"  15   4   7  ",
			"0 1 1",
			core.Recipe{{Op: "MOD", Args: []any{3.0, "Space"}}},
		},
		{
			"MOD: Empty input",
			"",
			"",
			core.Recipe{{Op: "MOD", Args: []any{3.0, "Space"}}},
		},
		{
			"MOD: Scientific notation",
			"1e3 2e2 5e1",
			"1 2 2",
			core.Recipe{{Op: "MOD", Args: []any{3.0, "Space"}}},
		},
		{
			"MOD: Floating point precision",
			"10.123456789 20.987654321",
			"1.123456789 2.987654321",
			core.Recipe{{Op: "MOD", Args: []any{3.0, "Space"}}},
		},
		{
			"MOD: Semi-colon separated numbers",
			"17;5;8;13",
			"2 0 3 3",
			core.Recipe{{Op: "MOD", Args: []any{5.0, "Semi-colon"}}},
		},
		{
			"MOD: Colon separated numbers",
			"25:9:14:7",
			"1 1 2 3",
			core.Recipe{{Op: "MOD", Args: []any{4.0, "Colon"}}},
		},
		{
			"MOD: CRLF separated numbers",
			"30\r\n18\r\n22\r\n11",
			"0 0 4 5",
			core.Recipe{{Op: "MOD", Args: []any{6.0, "CRLF"}}},
		},
	})
}

// TestMODZeroModulus covers CyberChef's "Zero modulus error" case. Upstream
// renders an operation's error as the recipe's output text; cchef returns it as
// an error so a shell sees a non-zero exit.
func TestMODZeroModulus(t *testing.T) {
	_, err := runOp(t, "MOD", "15 4 7", 0.0, "Space")
	if err == nil || !strings.Contains(err.Error(), "Modulus cannot be zero") {
		t.Errorf("got %v, want it to refuse a zero modulus", err)
	}
}
