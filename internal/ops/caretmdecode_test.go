package ops

import (
	"bytes"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// Transcribed from ../CyberChef/tests/operations/tests/CaretMdecode.mjs.
func TestCaretMdecodeFixtures(t *testing.T) {
	fullInput := "^@^A^B^C^D^E^F^G^H^I^J^K^L^M^N^O^P^Q^R^S^T^U^V^W^X^Y^Z^[^\\^]^^^_ !\"#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_`abcdefghijklmnopqrstuvwxyz{|}~^?M-^@M-^AM-^BM-^CM-^DM-^EM-^FM-^GM-^HM-^IM-^JM-^KM-^LM-^MM-^NM-^OM-^PM-^QM-^RM-^SM-^TM-^UM-^VM-^WM-^XM-^YM-^ZM-^[M-^\\M-^]M-^^M-^_M- M-!M-\"M-#M-$M-%M-&M-'M-(M-)M-*M-+M-,M--M-.M-/M-0M-1M-2M-3M-4M-5M-6M-7M-8M-9M-:M-;M-<M-=M->M-?M-@M-AM-BM-CM-DM-EM-FM-GM-HM-IM-JM-KM-LM-MM-NM-OM-PM-QM-RM-SM-TM-UM-VM-WM-XM-YM-ZM-[M-\\M-]M-^M-_M-`M-aM-bM-cM-dM-eM-fM-gM-hM-iM-jM-kM-lM-mM-nM-oM-pM-qM-rM-sM-tM-uM-vM-wM-xM-yM-zM-{M-|M-}M-~M-^?"
	fullOutput := "\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f\x10\x11\x12\x13\x14\x15\x16\x17\x18\x19\x1a\x1b\x1c\x1d\x1e\x1f\x20\x21\x22\x23\x24\x25\x26\x27\x28\x29\x2a\x2b\x2c\x2d\x2e\x2f\x30\x31\x32\x33\x34\x35\x36\x37\x38\x39\x3a\x3b\x3c\x3d\x3e\x3f\x40\x41\x42\x43\x44\x45\x46\x47\x48\x49\x4a\x4b\x4c\x4d\x4e\x4f\x50\x51\x52\x53\x54\x55\x56\x57\x58\x59\x5a\x5b\x5c\x5d\x1f\x60\x61\x62\x63\x64\x65\x66\x67\x68\x69\x6a\x6b\x6c\x6d\x6e\x6f\x70\x71\x72\x73\x74\x75\x76\x77\x78\x79\x7a\x7b\x7c\x7d\x7e\x7f\x80\x81\x82\x83\x84\x85\x86\x87\x88\x89\x8a\x8b\x8c\x8d\x8e\x8f\x90\x91\x92\x93\x94\x95\x96\x97\x98\x99\x9a\x9b\x9c\x9d\x9e\x9f\xa0\xa1\xa2\xa3\xa4\xa5\xa6\xa7\xa8\xa9\xaa\xab\xac\xad\xae\xaf\xb0\xb1\xb2\xb3\xb4\xb5\xb6\xb7\xb8\xb9\xba\xbb\xbc\xbd\xbe\xbf\xc0\xc1\xc2\xc3\xc4\xc5\xc6\xc7\xc8\xc9\xca\xcb\xcc\xcd\xce\xcf\xd0\xd1\xd2\xd3\xd4\xd5\xd6\xd7\xd8\xd9\xda\xdb\xdc\xdd\x8d\x2d\x5f\xe0\xe1\xe2\xe3\xe4\xe5\xe6\xe7\xe8\xe9\xea\xeb\xec\xed\xee\xef\xf0\xf1\xf2\xf3\xf4\xf5\xf6\xf7\xf8\xf9\xfa\xfb\xfc\xfd\xfe\xff"

	runCases(t, []opCase{
		{
			"Caret/M-decode: nothing", "", "",
			core.Recipe{{Op: "Caret/M-decode", Args: []any{}}},
		},
		{
			"Caret/M-decode: Full set", fullInput, fullOutput,
			core.Recipe{{Op: "Caret/M-decode", Args: []any{}}},
		},
	})
}

// TestCaretMdecodeBranches covers the literal-passthrough branches (when an
// escape prefix is not completed) and the trailing-prefix drop, verified against
// the CyberChef-server oracle. Output is shown as hex for readability.
func TestCaretMdecodeBranches(t *testing.T) {
	toHex := core.Recipe{
		{Op: "Caret/M-decode", Args: []any{}},
		{Op: "To Hex", Args: []any{"None"}},
	}
	runCases(t, []opCase{
		{"M-^ below range passes through", "M-^ ", "4d2d5e20", toHex},
		{"M- with control byte passes through", "M-\n", "4d2d0a", toHex},
		{"M without dash passes through", "MX", "4d58", toHex},
		{"caret below range passes through", "^ ", "5e20", toHex},
		{"trailing incomplete prefix is dropped", "M-^", "", toHex},
	})
}

// TestCMDecoder documents the caret/M-notation state machine via known escapes:
// ^A=Ctrl-A(1), ^?=DEL(127), M-A='A'+128(193), M-^A='A'+64(129), plain 'a'.
func TestCMDecoder(t *testing.T) {
	decode := func(s string) []byte {
		d := &cmDecoder{}
		for i := 0; i < len(s); i++ {
			d.feed(s[i])
		}
		return d.out
	}
	cases := map[string][]byte{
		"^A":   {1},
		"^?":   {127},
		"M-A":  {193},
		"M-^A": {129},
		"a":    {97},
	}
	for in, want := range cases {
		if got := decode(in); !bytes.Equal(got, want) {
			t.Fatalf("decode(%q) = %v, want %v", in, got, want)
		}
	}
}
