package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
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
	})
}
