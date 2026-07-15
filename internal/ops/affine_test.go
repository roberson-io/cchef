package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// Cases transcribed from CyberChef tests/operations/tests/Ciphers.mjs (the
// Affine Encode/Decode fixtures); the uppercase and round-trip cases are
// authored and verified against the CyberChef-server oracle.
func TestAffineFixtures(t *testing.T) {
	runCases(t, []opCase{
		// Encode.
		{
			"Affine Encode: no input", "", "",
			core.Recipe{{Op: "Affine Cipher Encode", Args: []any{1, 0}}},
		},
		{
			"Affine Encode: no effect", "some keys are shaped as locks. index[me]", "some keys are shaped as locks. index[me]",
			core.Recipe{{Op: "Affine Cipher Encode", Args: []any{1, 0}}},
		},
		{
			"Affine Encode: normal", "some keys are shaped as locks. index[me]", "vhnl tldv xyl vcxelo xv qhrtv. zkolg[nl]",
			core.Recipe{{Op: "Affine Cipher Encode", Args: []any{23, 23}}},
		},
		{
			"Affine Encode: uppercase preserved", "Hello", "Rclla",
			core.Recipe{{Op: "Affine Cipher Encode", Args: []any{5, 8}}},
		},

		// Decode.
		{
			"Affine Decode: no input", "", "",
			core.Recipe{{Op: "Affine Cipher Decode", Args: []any{1, 0}}},
		},
		{
			"Affine Decode: no effect", "vhnl tldv xyl vcxelo xv qhrtv. zkolg[nl]", "vhnl tldv xyl vcxelo xv qhrtv. zkolg[nl]",
			core.Recipe{{Op: "Affine Cipher Decode", Args: []any{1, 0}}},
		},
		{
			"Affine Decode: normal", "vhnl tldv xyl vcxelo xv qhrtv. zkolg[nl]", "some keys are shaped as locks. index[me]",
			core.Recipe{{Op: "Affine Cipher Decode", Args: []any{23, 23}}},
		},
		{
			"Affine Decode: uppercase preserved", "Rclla", "Hello",
			core.Recipe{{Op: "Affine Cipher Decode", Args: []any{5, 8}}},
		},

		// Round trip.
		{
			"Affine round trip", "The Quick Brown Fox!", "The Quick Brown Fox!",
			core.Recipe{
				{Op: "Affine Cipher Encode", Args: []any{7, 11}},
				{Op: "Affine Cipher Decode", Args: []any{7, 11}},
			},
		},
	})
}

func TestAffineErrors(t *testing.T) {
	cases := []struct {
		name string
		op   string
		a, b any
	}{
		{"encode non-integer a and b", "Affine Cipher Encode", 0.1, 0.00001},
		{"decode non-integer a and b", "Affine Cipher Decode", 0.1, 0.00001},
		{"encode non-coprime a", "Affine Cipher Encode", 8, 0},
		{"decode non-coprime a", "Affine Cipher Decode", 8, 23},
		{"encode negative a", "Affine Cipher Encode", -1, 0},
		{"decode negative b", "Affine Cipher Decode", 1, -5},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := runOp(t, c.op, "abc", c.a, c.b); err == nil {
				t.Errorf("%s(%v, %v): expected an error", c.op, c.a, c.b)
			}
		})
	}
}
