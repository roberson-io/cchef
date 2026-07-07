package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// TestChangeIPFormatFixtures transcribes CyberChef's ChangeIPFormat.mjs cases.
func TestChangeIPFormatFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"Dotted Decimal to Hex", "192.168.1.1", "c0a80101",
			core.Recipe{{Op: "Change IP format", Args: []any{"Dotted Decimal", "Hex"}}},
		},
		{
			"Decimal to Dotted Decimal", "3232235777", "192.168.1.1",
			core.Recipe{{Op: "Change IP format", Args: []any{"Decimal", "Dotted Decimal"}}},
		},
		{
			"Hex to Octal", "c0a80101", "030052000401",
			core.Recipe{{Op: "Change IP format", Args: []any{"Hex", "Octal"}}},
		},
		{
			"Octal to Decimal", "030052000401", "3232235777",
			core.Recipe{{Op: "Change IP format", Args: []any{"Octal", "Decimal"}}},
		},
	})
}

// TestChangeIPFormatBranches covers the blank-line skip and the identity
// (input format == output format) passthrough.
func TestChangeIPFormatBranches(t *testing.T) {
	out, err := runOp(t, "Change IP format", "1.2.3.4\n\n5.6.7.8", "Dotted Decimal", "Dotted Decimal")
	if err != nil {
		t.Fatal(err)
	}
	if out != "1.2.3.4\n5.6.7.8" {
		t.Fatalf("identity passthrough = %q", out)
	}
}
