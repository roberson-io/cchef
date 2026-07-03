package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// TestVarIntOracle checks VarInt Encode/Decode against CyberChef-server output
// (v11.2.0); no upstream fixture. Values are LEB128 base-128 varints; the big
// case (2^64-1) confirms arbitrary-precision handling.
func TestVarIntOracle(t *testing.T) {
	runCases(t, []opCase{
		{"decode 300", "ac02", "300",
			core.Recipe{{Op: "From Hex", Args: []any{"None"}}, {Op: "VarInt Decode", Args: []any{}}}},
		{"decode 2^64-1", "ffffffffffffffffff01", "18446744073709551615",
			core.Recipe{{Op: "From Hex", Args: []any{"None"}}, {Op: "VarInt Decode", Args: []any{}}}},
		{"encode 300", "300", "ac02",
			core.Recipe{{Op: "VarInt Encode", Args: []any{}}, {Op: "To Hex", Args: []any{"None"}}}},
		{"encode 2^64-1", "18446744073709551615", "ffffffffffffffffff01",
			core.Recipe{{Op: "VarInt Encode", Args: []any{}}, {Op: "To Hex", Args: []any{"None"}}}},
	})
}
