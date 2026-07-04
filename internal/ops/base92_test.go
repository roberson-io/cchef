package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// Cases transcribed from CyberChef tests/operations/tests/Base92.mjs.
func TestBase92Fixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"To Base92: nothing", "", "",
			core.Recipe{{Op: "To Base92"}},
		},
		{
			"To Base92: AB", "AB", "8y2",
			core.Recipe{{Op: "To Base92"}},
		},
		{
			"To Base92: Hello!!", "Hello!!", ";K_$aOTo&",
			core.Recipe{{Op: "To Base92"}},
		},
		{
			"To Base92: base-92", "base-92", "DX2?V<Y(*",
			core.Recipe{{Op: "To Base92"}},
		},

		{
			"From Base92: nothing", "", "",
			core.Recipe{{Op: "From Base92"}},
		},
		{
			"From Base92: ietf", "G'_DW[B", "ietf!",
			core.Recipe{{Op: "From Base92"}},
		},
		{
			// Exercises the 'a'-'}' ordinal branch of base92Ord (values >= 62).
			"From Base92: Hello, World!", ";K_$aOVi#vqDXS-Z", "Hello, World!",
			core.Recipe{{Op: "From Base92"}},
		},
		{
			// '!' is base92 digit 0 (base92Ord's c=='!' branch); "!!" decodes to a
			// null byte, shown here as hex.
			"From Base92: '!!' is a null byte", "!!", "00",
			core.Recipe{{Op: "From Base92"}, {Op: "To Hex", Args: []any{"None"}}},
		},

		{
			"Base92 round trip", "The quick brown fox", "The quick brown fox",
			core.Recipe{
				{Op: "To Base92"},
				{Op: "From Base92"},
			},
		},
	})
}

// TestFromBase92Invalid covers base92Ord's error branch: '~' (0x7e) is above the
// valid 'a'-'}' range, so decoding rejects it (as CyberChef does).
func TestFromBase92Invalid(t *testing.T) {
	if _, err := runOp(t, "From Base92", "~"); err == nil {
		t.Error("expected error for invalid base92 character '~'")
	}
}
