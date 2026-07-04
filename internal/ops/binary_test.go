package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// Binary has no dedicated CyberChef fixture file; these are hand-verified
// vectors (each byte as 8-bit binary) plus round-trips.
func TestBinaryOps(t *testing.T) {
	runCases(t, []opCase{
		{
			"To Binary: nothing", "", "",
			core.Recipe{{Op: "To Binary", Args: []any{"Space", 8}}},
		},
		{
			"To Binary: AB", "AB", "01000001 01000010",
			core.Recipe{{Op: "To Binary", Args: []any{"Space", 8}}},
		},
		{
			"To Binary: Hi!", "Hi!", "01001000 01101001 00100001",
			core.Recipe{{Op: "To Binary", Args: []any{"Space", 8}}},
		},
		{
			"To Binary: None delim", "AB", "0100000101000010",
			core.Recipe{{Op: "To Binary", Args: []any{"None", 8}}},
		},

		{
			"From Binary: AB", "01000001 01000010", "AB",
			core.Recipe{{Op: "From Binary", Args: []any{"Space", 8}}},
		},
		{
			"From Binary: no delim", "0100000101000010", "AB",
			core.Recipe{{Op: "From Binary", Args: []any{"Space", 8}}},
		},

		{
			"Binary round trip", "Hello, World!", "Hello, World!",
			core.Recipe{
				{Op: "To Binary", Args: []any{"Comma", 8}},
				{Op: "From Binary", Args: []any{"Comma", 8}},
			},
		},
	})
}
