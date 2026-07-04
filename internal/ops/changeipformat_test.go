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
