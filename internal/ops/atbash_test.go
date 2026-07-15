package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// Cases transcribed from ../CyberChef/tests/operations/tests/Ciphers.mjs (the
// Atbash fixtures); the uppercase and self-inverse cases are authored and
// verified against the CyberChef-server oracle.
func TestAtbashFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"Atbash: no input", "", "",
			core.Recipe{{Op: "Atbash Cipher", Args: []any{}}},
		},
		{
			"Atbash: normal", "old slow slim horn", "low hold horn slim",
			core.Recipe{{Op: "Atbash Cipher", Args: []any{}}},
		},
		{
			"Atbash: uppercase preserved, non-alpha passthrough", "Hello, World!", "Svool, Dliow!",
			core.Recipe{{Op: "Atbash Cipher", Args: []any{}}},
		},
		{
			// Atbash is its own inverse: applying it twice restores the input.
			"Atbash: self-inverse", "The Quick Brown Fox!", "The Quick Brown Fox!",
			core.Recipe{
				{Op: "Atbash Cipher", Args: []any{}},
				{Op: "Atbash Cipher", Args: []any{}},
			},
		},
	})
}
