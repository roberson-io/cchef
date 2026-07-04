package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// To Base / From Base have no CyberChef fixture file; these are hand-verified
// integer radix conversions (Go big.Int and JS Number.toString share the
// lower-case digit alphabet for radixes 2-36).
func TestBaseGeneric(t *testing.T) {
	runCases(t, []opCase{
		{
			"To Base 16", "255", "ff",
			core.Recipe{{Op: "To Base", Args: []any{16}}},
		},
		{
			"To Base 2", "255", "11111111",
			core.Recipe{{Op: "To Base", Args: []any{2}}},
		},
		{
			"To Base 36", "1295", "zz",
			core.Recipe{{Op: "To Base", Args: []any{36}}},
		},

		{
			"From Base 16", "ff", "255",
			core.Recipe{{Op: "From Base", Args: []any{16}}},
		},
		{
			"From Base 2", "11111111", "255",
			core.Recipe{{Op: "From Base", Args: []any{2}}},
		},
		{
			"From Base 36 (whitespace stripped)", "z z", "1295",
			core.Recipe{{Op: "From Base", Args: []any{36}}},
		},

		{
			"Base round trip", "123456789", "123456789",
			core.Recipe{
				{Op: "To Base", Args: []any{16}},
				{Op: "From Base", Args: []any{16}},
			},
		},
	})
}
