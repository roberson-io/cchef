package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

func xorRecipe(blocksize int) core.Recipe {
	return core.Recipe{{Op: "XOR Checksum", Args: []any{blocksize}}}
}

// Vectors transcribed from ../CyberChef/tests/operations/tests/XORChecksum.mjs.
const (
	xorBasicString = "The ships hung in the sky in much the same way that bricks don't."
	xorUTF8Str     = "ნუ პანიკას"
)

func TestXORChecksumFixtures(t *testing.T) {
	runCases(t, []opCase{
		{"XOR Checksum (1): nothing", "", "00", xorRecipe(1)},
		{"XOR Checksum (1): basic string", xorBasicString, "08", xorRecipe(1)},
		{"XOR Checksum (1): UTF-8", xorUTF8Str, "df", xorRecipe(1)},
		{"XOR Checksum (1): all bytes", allBytes(), "00", xorRecipe(1)},

		{"XOR Checksum (4): nothing", "", "00000000", xorRecipe(4)},
		{"XOR Checksum (4): basic string", xorBasicString, "4918421b", xorRecipe(4)},
		{"XOR Checksum (4): UTF-8", xorUTF8Str, "83a424dc", xorRecipe(4)},
		{"XOR Checksum (4): all bytes", allBytes(), "00000000", xorRecipe(4)},
	})
}

func TestXORChecksumErrors(t *testing.T) {
	for _, bs := range []any{0, -1} {
		if _, err := runOp(t, "XOR Checksum", "abc", bs); err == nil || err.Error() != "Blocksize must be a positive integer." {
			t.Fatalf("blocksize %v: got %v, want positive-integer error", bs, err)
		}
	}
}
