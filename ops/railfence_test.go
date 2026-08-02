package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// Rail Fence fixtures transcribed from
// CyberChef's tests/operations/tests/Ciphers.mjs.
func TestRailFenceFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"Rail Fence Encode: normal",
			"Cryptography is THE Art of Writing or solving codes",
			"Cytgah sTEAto rtn rsligcdsrporpyi H r fWiigo ovn oe",
			core.Recipe{{Op: "Rail Fence Cipher Encode", Args: []any{float64(2), float64(0)}}},
		},
		{
			"Rail Fence Encode: offset non-null",
			"12345678901234567890", "51746026813793592840",
			core.Recipe{{Op: "Rail Fence Cipher Encode", Args: []any{float64(4), float64(2)}}},
		},
		{
			"Rail Fence Encode: offset with spaces",
			"No one expects the spanish Inquisition.",
			"  e  n ut.ooeepcstesaihIqiiinNnxthpsnso",
			core.Recipe{{Op: "Rail Fence Cipher Encode", Args: []any{float64(3), float64(2)}}},
		},
		{
			"Rail Fence Decode: normal",
			"Cytgah sTEAto rtn rsligcdsrporpyi H r fWiigo ovn oe",
			"Cryptography is THE Art of Writing or solving codes",
			core.Recipe{{Op: "Rail Fence Cipher Decode", Args: []any{float64(2), float64(0)}}},
		},
		{
			"Rail Fence Decode: offset non-null",
			"51746026813793592840", "12345678901234567890",
			core.Recipe{{Op: "Rail Fence Cipher Decode", Args: []any{float64(4), float64(2)}}},
		},
	})
}

// TestRailFenceErrors covers the key/offset validation messages for both ops.
func TestRailFenceErrors(t *testing.T) {
	cases := []struct {
		op, input string
		key, off  float64
		sub       string
	}{
		{"Rail Fence Cipher Encode", "Cryptography is THE Art of Writing or solving codes", 1, 0, "Key has to be bigger than 2"},
		{"Rail Fence Cipher Encode", "shortinput", 22, 0, "Key should be smaller than the plain text's length"},
		{"Rail Fence Cipher Encode", "shortinput", 2, -1, "Offset has to be a positive integer"},
		{"Rail Fence Cipher Decode", "Cytgah sTEAto rtn rsligcdsrporpyi H r fWiigo ovn oe", 1, 0, "Key has to be bigger than 2"},
		{"Rail Fence Cipher Decode", "shortinput", 22, 0, "Key should be smaller than the cipher's length"},
		{"Rail Fence Cipher Decode", "shortinput", 2, -1, "Offset has to be a positive integer"},
	}
	for _, c := range cases {
		t.Run(c.op+" "+c.sub, func(t *testing.T) {
			_, err := runOp(t, c.op, c.input, c.key, c.off)
			if err == nil || !strings.Contains(err.Error(), c.sub) {
				t.Fatalf("got %v, want %q", err, c.sub)
			}
		})
	}
}
