package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// TestCetaceanCipherFixtures transcribes the CyberChef CetaceanCipherEncode.mjs
// and CetaceanCipherDecode.mjs fixtures.
func TestCetaceanCipherFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"Cetacean Cipher Encode", "a b c で",
			"EEEEEEEEEeeEEEEe EEEEEEEEEeeEEEeE EEEEEEEEEeeEEEee EEeeEEEEEeeEEeee",
			core.Recipe{{Op: "Cetacean Cipher Encode", Args: []any{}}},
		},
		{
			"Cetacean Cipher Decode",
			"EEEEEEEEEeeEEEEe EEEEEEEEEeeEEEeE EEEEEEEEEeeEEEee EEeeEEEEEeeEEeee",
			"a b c で",
			core.Recipe{{Op: "Cetacean Cipher Decode", Args: []any{}}},
		},
	})
}

// TestCetaceanCipherEdge covers empty input, round-tripping, and the decode
// treatment of non-'e' characters (anything but 'e' or space is a 0 bit).
func TestCetaceanCipherEdge(t *testing.T) {
	runCases(t, []opCase{
		{
			"Cetacean Cipher Encode: empty", "", "",
			core.Recipe{{Op: "Cetacean Cipher Encode", Args: []any{}}},
		},
		{
			"Cetacean Cipher Decode: empty", "", "",
			core.Recipe{{Op: "Cetacean Cipher Decode", Args: []any{}}},
		},
		{
			"Cetacean Cipher Encode: hi", "hi",
			"EEEEEEEEEeeEeEEEEEEEEEEEEeeEeEEe",
			core.Recipe{{Op: "Cetacean Cipher Encode", Args: []any{}}},
		},
		{
			"Cetacean Cipher Decode: hi", "EEEEEEEEEeeEeEEEEEEEEEEEEeeEeEEe", "hi",
			core.Recipe{{Op: "Cetacean Cipher Decode", Args: []any{}}},
		},
		// Round-trip through both ops leaves the input unchanged.
		{
			"Cetacean Cipher: round trip", "Hello, World!", "Hello, World!",
			core.Recipe{
				{Op: "Cetacean Cipher Encode", Args: []any{}},
				{Op: "Cetacean Cipher Decode", Args: []any{}},
			},
		},
		// In decode, characters other than 'e' (here 'E' and stray letters) are 0
		// bits; a full 16-char group of 'E' decodes to a NUL.
		{
			"Cetacean Cipher Decode: all E is NUL", "EEEEEEEEEEEEEEEE", "\x00",
			core.Recipe{{Op: "Cetacean Cipher Decode", Args: []any{}}},
		},
		// A trailing group shorter than 16 bits is parsed from just those bits:
		// "eeee" is 0b1111 = 0x0f.
		{
			"Cetacean Cipher Decode: partial final group", "eeee", "\x0f",
			core.Recipe{{Op: "Cetacean Cipher Decode", Args: []any{}}},
		},
	})
}
