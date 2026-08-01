package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

const (
	bifidPlain  = "We recreate conditions similar to the Van-Allen radiation belt in our secure facilities."
	bifidNoKey  = "Vq daqcliho rmltofvlnc qbdhlcr nt qdq Fbm-Rdkkm vuoottnoi aitp al axf tdtmvt owppkaodtx."
	bifidSchrod = "Wc snpsigdd cpfrrcxnfi hikdnnp dm crc Fcb-Pdeug vueageacc vtyl sa zxm crebzp lyoeuaiwpv."
)

// Cases transcribed from ../CyberChef/tests/operations/tests/Ciphers.mjs.
func TestBifidFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"Bifid Cipher Encode: no input", "", "",
			core.Recipe{{Op: "Bifid Cipher Encode", Args: []any{"nothing"}}},
		},
		{
			"Bifid Cipher Encode: no key", bifidPlain, bifidNoKey,
			core.Recipe{{Op: "Bifid Cipher Encode", Args: []any{""}}},
		},
		{
			"Bifid Cipher Encode: normal", bifidPlain, bifidSchrod,
			core.Recipe{{Op: "Bifid Cipher Encode", Args: []any{"Schrodinger"}}},
		},
		{
			"Bifid Cipher Decode: no input", "", "",
			core.Recipe{{Op: "Bifid Cipher Decode", Args: []any{"nothing"}}},
		},
		{
			"Bifid Cipher Decode: no key", bifidNoKey, bifidPlain,
			core.Recipe{{Op: "Bifid Cipher Decode", Args: []any{""}}},
		},
		{
			"Bifid Cipher Decode: normal", bifidSchrod, bifidPlain,
			core.Recipe{{Op: "Bifid Cipher Decode", Args: []any{"Schrodinger"}}},
		},

		// Round trip with a keyword (oracle-verified via the fixtures above).
		{
			"Bifid round trip", bifidPlain, bifidPlain,
			core.Recipe{
				{Op: "Bifid Cipher Encode", Args: []any{"Schrodinger"}},
				{Op: "Bifid Cipher Decode", Args: []any{"Schrodinger"}},
			},
		},
	})
}

func TestBifidErrors(t *testing.T) {
	const wantErr = "The key must consist only of letters in the English alphabet"
	for _, op := range []string{"Bifid Cipher Encode", "Bifid Cipher Decode"} {
		t.Run(op, func(t *testing.T) {
			_, err := runOp(t, op, bifidPlain, "abc123")
			if err == nil {
				t.Fatalf("%s: expected error, got nil", op)
			}
			if err.Error() != wantErr {
				t.Fatalf("%s: got %q\nwant %q", op, err.Error(), wantErr)
			}
		})
	}
}
