package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// Cases transcribed from CyberChef tests/operations/tests/Base32.mjs.
func TestBase32Fixtures(t *testing.T) {
	std := "A-Z2-7="
	ext := "0-9A-V="
	runCases(t, []opCase{
		{"To Base32 Standard: nothing", "", "",
			core.Recipe{{Op: "To Base32", Args: []any{std}}}},
		{"To Base32 Standard", "HELLO BASE32", "JBCUYTCPEBBECU2FGMZA====",
			core.Recipe{{Op: "To Base32", Args: []any{std}}}},
		{"To Base32 Hex Extended", "HELLO BASE32 EXTENDED", "912KOJ2F41142KQ56CP20HAOAH2KSH258G======",
			core.Recipe{{Op: "To Base32", Args: []any{ext}}}},

		{"From Base32 Standard: nothing", "", "",
			core.Recipe{{Op: "From Base32", Args: []any{std, false}}}},
		{"From Base32 Standard", "JBCUYTCPEBBECU2FGMZA====", "HELLO BASE32",
			core.Recipe{{Op: "From Base32", Args: []any{std, false}}}},
		{"From Base32 Hex Extended", "912KOJ2F41142KQ56CP20HAOAH2KSH258G======", "HELLO BASE32 EXTENDED",
			core.Recipe{{Op: "From Base32", Args: []any{ext, false}}}},

		{"Base32 round trip", "The quick brown fox", "The quick brown fox",
			core.Recipe{
				{Op: "To Base32", Args: []any{std}},
				{Op: "From Base32", Args: []any{std, false}},
			}},
	})
}
