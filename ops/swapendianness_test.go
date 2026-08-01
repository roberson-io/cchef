package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

// Swap endianness has no dedicated CyberChef fixture file; these are
// hand-verified vectors. Hex output is space-delimited (CyberChef's toHex default).
func TestSwapEndianness(t *testing.T) {
	runCases(t, []opCase{
		{
			"Hex word 4", "0a0b0c0d", "0d 0c 0b 0a",
			core.Recipe{{Op: "Swap endianness", Args: []any{"Hex", 4, true}}},
		},
		{
			"Hex word 4, pad incomplete", "deadbeef0102", "ef be ad de 00 00 02 01",
			core.Recipe{{Op: "Swap endianness", Args: []any{"Hex", 4, true}}},
		},
		{
			"Hex word 4, no pad", "deadbeef0102", "ef be ad de 02 01",
			core.Recipe{{Op: "Swap endianness", Args: []any{"Hex", 4, false}}},
		},
		{
			"Hex word 2", "0a0b0c0d", "0b 0a 0d 0c",
			core.Recipe{{Op: "Swap endianness", Args: []any{"Hex", 2, true}}},
		},

		{
			"Raw word 2", "ABCD", "BADC",
			core.Recipe{{Op: "Swap endianness", Args: []any{"Raw", 2, true}}},
		},

		// Swapping with the same word length twice restores the original.
		{
			"Raw round trip", "ABCDEFGH", "ABCDEFGH",
			core.Recipe{
				{Op: "Swap endianness", Args: []any{"Raw", 4, true}},
				{Op: "Swap endianness", Args: []any{"Raw", 4, true}},
			},
		},
	})
}

// TestSwapEndiannessWordLength covers the non-positive word-length guard.
func TestSwapEndiannessWordLength(t *testing.T) {
	if _, err := runOp(t, "Swap endianness", "0011", "Hex", 0, true); err == nil {
		t.Fatal("expected an error for a zero word length")
	}
}
