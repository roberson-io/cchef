package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// Decode cases transcribed from CyberChef
// tests/operations/tests/A1Z26CipherDecode.mjs; the remaining cases are
// authored and verified against the CyberChef-server oracle.
func TestA1Z26Fixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"A1Z26 Cipher Decode: basic decode", "8 5 12 12 15", "hello",
			core.Recipe{{Op: "A1Z26 Cipher Decode", Args: []any{"Space"}}},
		},
		{
			"A1Z26 Cipher Decode: empty input returns empty string", "", "",
			core.Recipe{{Op: "A1Z26 Cipher Decode", Args: []any{"Space"}}},
		},

		// Decode with each delimiter (oracle-verified).
		{
			"A1Z26 Cipher Decode: Comma", "8,9", "hi",
			core.Recipe{{Op: "A1Z26 Cipher Decode", Args: []any{"Comma"}}},
		},
		{
			"A1Z26 Cipher Decode: Semi-colon", "8;9", "hi",
			core.Recipe{{Op: "A1Z26 Cipher Decode", Args: []any{"Semi-colon"}}},
		},
		{
			"A1Z26 Cipher Decode: Colon", "8:9", "hi",
			core.Recipe{{Op: "A1Z26 Cipher Decode", Args: []any{"Colon"}}},
		},
		{
			"A1Z26 Cipher Decode: Line feed", "8\n9", "hi",
			core.Recipe{{Op: "A1Z26 Cipher Decode", Args: []any{"Line feed"}}},
		},
		{
			"A1Z26 Cipher Decode: CRLF", "8\r\n9", "hi",
			core.Recipe{{Op: "A1Z26 Cipher Decode", Args: []any{"CRLF"}}},
		},
		{
			"A1Z26 Cipher Decode: boundary 26", "26", "z",
			core.Recipe{{Op: "A1Z26 Cipher Decode", Args: []any{"Space"}}},
		},

		// JS coercion quirks (oracle-verified): the range check uses Number()
		// (a decimal like 8.5 passes, then parseInt truncates to 8), and a
		// non-numeric token yields NaN which skips the range check.
		{
			"A1Z26 Cipher Decode: decimal token truncates", "8.5", "h",
			core.Recipe{{Op: "A1Z26 Cipher Decode", Args: []any{"Space"}}},
		},
		{
			"A1Z26 Cipher Decode: trailing non-digit ignored", "8a", "h",
			core.Recipe{{Op: "A1Z26 Cipher Decode", Args: []any{"Space"}}},
		},
		{
			"A1Z26 Cipher Decode: leading sign", "+5", "e",
			core.Recipe{{Op: "A1Z26 Cipher Decode", Args: []any{"Space"}}},
		},
		{
			// Fully non-numeric token: Number() is NaN (no range error) and
			// parseInt is NaN, so chr(NaN) yields a NUL byte.
			"A1Z26 Cipher Decode: non-numeric token", "abc", "\x00",
			core.Recipe{{Op: "A1Z26 Cipher Decode", Args: []any{"Space"}}},
		},

		// Encode (oracle-verified; A1Z26CipherEncode.mjs has no fixture file).
		{
			"A1Z26 Cipher Encode: Hello, World!", "Hello, World!", "8 5 12 12 15 23 15 18 12 4",
			core.Recipe{{Op: "A1Z26 Cipher Encode", Args: []any{"Space"}}},
		},
		{
			"A1Z26 Cipher Encode: boundary letters", "abz", "1 2 26",
			core.Recipe{{Op: "A1Z26 Cipher Encode", Args: []any{"Space"}}},
		},
		{
			"A1Z26 Cipher Encode: Comma", "abz", "1,2,26",
			core.Recipe{{Op: "A1Z26 Cipher Encode", Args: []any{"Comma"}}},
		},
		{
			"A1Z26 Cipher Encode: Semi-colon", "abz", "1;2;26",
			core.Recipe{{Op: "A1Z26 Cipher Encode", Args: []any{"Semi-colon"}}},
		},
		{
			"A1Z26 Cipher Encode: Colon", "abz", "1:2:26",
			core.Recipe{{Op: "A1Z26 Cipher Encode", Args: []any{"Colon"}}},
		},
		{
			"A1Z26 Cipher Encode: Line feed", "abz", "1\n2\n26",
			core.Recipe{{Op: "A1Z26 Cipher Encode", Args: []any{"Line feed"}}},
		},
		{
			"A1Z26 Cipher Encode: CRLF", "abz", "1\r\n2\r\n26",
			core.Recipe{{Op: "A1Z26 Cipher Encode", Args: []any{"CRLF"}}},
		},
		{
			// Non-alphabet characters are dropped; punctuation and spaces vanish.
			"A1Z26 Cipher Encode: drops non-alpha", "the quick brown fox", "20 8 5 17 21 9 3 11 2 18 15 23 14 6 15 24",
			core.Recipe{{Op: "A1Z26 Cipher Encode", Args: []any{"Space"}}},
		},
		{
			"A1Z26 Cipher Encode: all non-alpha returns empty", "123 456", "",
			core.Recipe{{Op: "A1Z26 Cipher Encode", Args: []any{"Space"}}},
		},

		// Round trip.
		{
			"A1Z26 round trip", "hello", "hello",
			core.Recipe{
				{Op: "A1Z26 Cipher Encode", Args: []any{"Comma"}},
				{Op: "A1Z26 Cipher Decode", Args: []any{"Comma"}},
			},
		},
	})
}

func TestA1Z26DecodeErrors(t *testing.T) {
	for _, in := range []string{"27", "0", "8  5", "8 5 ", " 8 5"} {
		if _, err := runOp(t, "A1Z26 Cipher Decode", in, "Space"); err == nil {
			t.Errorf("A1Z26 Cipher Decode(%q): expected an error", in)
		}
	}
}
