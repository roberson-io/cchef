package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

func TestReverseOp(t *testing.T) {
	runCases(t, []opCase{
		{
			"Reverse Character (default)", "Hello, World!", "!dlroW ,olleH",
			core.Recipe{{Op: "Reverse"}},
		},
		{
			"Reverse Byte", "abc", "cba",
			core.Recipe{{Op: "Reverse", Args: []any{"Byte"}}},
		},
		{
			"Reverse Line", "one\ntwo\nthree", "three\ntwo\none",
			core.Recipe{{Op: "Reverse", Args: []any{"Line"}}},
		},
		// By Character keeps multi-byte UTF-8 sequences intact.
		{
			"Reverse Character UTF-8", "abé", "éba",
			core.Recipe{{Op: "Reverse", Args: []any{"Character"}}},
		},
		// By Character must not destroy bytes that are not valid UTF-8: CyberChef
		// keeps them (as Latin-1 code points) rather than replacing with U+FFFD.
		// Bytes f0 aa bb reverse to code points bb aa f0, re-encoded as UTF-8.
		{
			"Reverse Character preserves non-UTF-8 bytes", "f0aabb", "c2bbc2aac3b0",
			core.Recipe{
				{Op: "From Hex", Args: []any{"Auto"}},
				{Op: "Reverse", Args: []any{"Character"}},
				{Op: "To Hex", Args: []any{"None"}},
			},
		},
	})
}
