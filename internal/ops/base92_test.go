package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// Cases transcribed from CyberChef tests/operations/tests/Base92.mjs.
func TestBase92Fixtures(t *testing.T) {
	runCases(t, []opCase{
		{"To Base92: nothing", "", "",
			core.Recipe{{Op: "To Base92"}}},
		{"To Base92: AB", "AB", "8y2",
			core.Recipe{{Op: "To Base92"}}},
		{"To Base92: Hello!!", "Hello!!", ";K_$aOTo&",
			core.Recipe{{Op: "To Base92"}}},
		{"To Base92: base-92", "base-92", "DX2?V<Y(*",
			core.Recipe{{Op: "To Base92"}}},

		{"From Base92: nothing", "", "",
			core.Recipe{{Op: "From Base92"}}},
		{"From Base92: ietf", "G'_DW[B", "ietf!",
			core.Recipe{{Op: "From Base92"}}},

		{"Base92 round trip", "The quick brown fox", "The quick brown fox",
			core.Recipe{
				{Op: "To Base92"},
				{Op: "From Base92"},
			}},
	})
}
