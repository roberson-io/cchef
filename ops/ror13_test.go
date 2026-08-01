package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

// TestROR13Fixtures transcribes CyberChef's ROR13.mjs fixtures. ROR13 is the
// Windows API-name hashing convention: rotate the accumulator right 13 bits and
// add each byte, emitting the 32-bit result as uppercase hex.
func TestROR13Fixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"ROR13: AddConsoleAliasW", "AddConsoleAliasW", "0x9916128C",
			core.Recipe{{Op: "ROR13", Args: []any{}}},
		},
		{
			"ROR13: LoadLibraryA", "LoadLibraryA", "0xEC0E4E8E",
			core.Recipe{{Op: "ROR13", Args: []any{}}},
		},
		{
			"ROR13: CloseHandle", "CloseHandle", "0x0FFD97FB",
			core.Recipe{{Op: "ROR13", Args: []any{}}},
		},
	})
}
