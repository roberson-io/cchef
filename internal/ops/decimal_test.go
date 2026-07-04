package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// Decimal has no dedicated CyberChef fixture file; these are hand-verified
// vectors plus round-trips.
func TestDecimalOps(t *testing.T) {
	runCases(t, []opCase{
		{
			"To Decimal: ABC", "ABC", "65 66 67",
			core.Recipe{{Op: "To Decimal", Args: []any{"Space", false}}},
		},
		{
			"To Decimal: Comma", "ABC", "65,66,67",
			core.Recipe{{Op: "To Decimal", Args: []any{"Comma", false}}},
		},
		{
			"To Decimal: signed 0xFF", "\xff", "-1",
			core.Recipe{{Op: "To Decimal", Args: []any{"Space", true}}},
		},

		{
			"From Decimal: ABC", "65 66 67", "ABC",
			core.Recipe{{Op: "From Decimal", Args: []any{"Space", false}}},
		},
		{
			"From Decimal: signed -1", "-1", "\xff",
			core.Recipe{{Op: "From Decimal", Args: []any{"Space", true}}},
		},

		{
			"Decimal round trip", "Hello, World!", "Hello, World!",
			core.Recipe{
				{Op: "To Decimal", Args: []any{"Space", false}},
				{Op: "From Decimal", Args: []any{"Space", false}},
			},
		},
	})
}
