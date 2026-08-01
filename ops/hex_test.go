package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

// allBytesHex is the "To Hex: All bytes" expected output from
// CyberChef tests/operations/tests/ByteRepr.mjs (Space delimiter).
const allBytesHex = "00 01 02 03 04 05 06 07 08 09 0a 0b 0c 0d 0e 0f 10 11 12 13 14 15 16 17 18 19 1a 1b 1c 1d 1e 1f 20 21 22 23 24 25 26 27 28 29 2a 2b 2c 2d 2e 2f 30 31 32 33 34 35 36 37 38 39 3a 3b 3c 3d 3e 3f 40 41 42 43 44 45 46 47 48 49 4a 4b 4c 4d 4e 4f 50 51 52 53 54 55 56 57 58 59 5a 5b 5c 5d 5e 5f 60 61 62 63 64 65 66 67 68 69 6a 6b 6c 6d 6e 6f 70 71 72 73 74 75 76 77 78 79 7a 7b 7c 7d 7e 7f 80 81 82 83 84 85 86 87 88 89 8a 8b 8c 8d 8e 8f 90 91 92 93 94 95 96 97 98 99 9a 9b 9c 9d 9e 9f a0 a1 a2 a3 a4 a5 a6 a7 a8 a9 aa ab ac ad ae af b0 b1 b2 b3 b4 b5 b6 b7 b8 b9 ba bb bc bd be bf c0 c1 c2 c3 c4 c5 c6 c7 c8 c9 ca cb cc cd ce cf d0 d1 d2 d3 d4 d5 d6 d7 d8 d9 da db dc dd de df e0 e1 e2 e3 e4 e5 e6 e7 e8 e9 ea eb ec ed ee ef f0 f1 f2 f3 f4 f5 f6 f7 f8 f9 fa fb fc fd fe ff"

// Cases transcribed from CyberChef tests/operations/tests/ByteRepr.mjs.
func TestHexFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"To Hex: nothing", "", "",
			core.Recipe{{Op: "To Hex", Args: []any{"Space"}}},
		},
		{
			"To Hex: All bytes", allBytes(), allBytesHex,
			core.Recipe{{Op: "To Hex", Args: []any{"Space"}}},
		},
		{
			"To Hex: UTF-8 None", "ნუ პანიკას", "e1839ce183a320e1839ee18390e1839ce18398e18399e18390e183a1",
			core.Recipe{{Op: "To Hex", Args: []any{"None"}}},
		},
		{
			"To Hex: CRLF delimiter", "Hi!", "48\r\n69\r\n21",
			core.Recipe{{Op: "To Hex", Args: []any{"CRLF"}}},
		},
		{
			"To Hex: 0x delimiter", "Hi!", "0x480x690x21",
			core.Recipe{{Op: "To Hex", Args: []any{"0x"}}},
		},

		{
			"From Hex: nothing", "", "",
			core.Recipe{{Op: "From Hex", Args: []any{"Space"}}},
		},
		{
			"From Hex: All bytes", allBytesHex, allBytes(),
			core.Recipe{{Op: "From Hex", Args: []any{"Space"}}},
		},
		{
			"From Hex: Auto", "e1 83,9c:e1", "\xe1\x83\x9c\xe1",
			core.Recipe{{Op: "From Hex", Args: []any{"Auto"}}},
		},

		{
			"To Hex 0x with comma", "abc", "0x61,0x62,0x63",
			core.Recipe{{Op: "To Hex", Args: []any{"0x with comma"}}},
		},
		{
			"To Hex Percent (prepended)", "abc", "%61%62%63",
			core.Recipe{{Op: "To Hex", Args: []any{"Percent"}}},
		},
		{
			"To Hex Semi-colon", "abc", "61;62;63",
			core.Recipe{{Op: "To Hex", Args: []any{"Semi-colon"}}},
		},
		{
			"To Hex Comma", "abc", "61,62,63",
			core.Recipe{{Op: "To Hex", Args: []any{"Comma"}}},
		},
		{
			"To Hex None", "abc", "616263",
			core.Recipe{{Op: "To Hex", Args: []any{"None"}}},
		},
		{
			"To Hex backslash-x (prepended)", "abc", "\\x61\\x62\\x63",
			core.Recipe{{Op: "To Hex", Args: []any{"\\x"}}},
		},

		{
			"Hex round trip", "Hello, World!", "Hello, World!",
			core.Recipe{
				{Op: "To Hex", Args: []any{"Colon"}},
				{Op: "From Hex", Args: []any{"Colon"}},
			},
		},
	})
}

func TestFromHexRejectsInvalidByte(t *testing.T) {
	if _, err := runOp(t, "From Hex", "zz", "Space"); err == nil {
		t.Fatal("expected an error for an invalid hex byte")
	}
}

// TestToHexBytesPerLine covers the "Bytes per line" argument. The expected
// values are CyberChef's, recorded through its Node API. Note that the
// delimiter before a line break is kept — only the very last one is dropped —
// which is what upstream produces.
func TestToHexBytesPerLine(t *testing.T) {
	const input = "Hello World!" // twelve bytes
	cases := []struct {
		name  string
		delim string
		per   float64
		want  string
	}{
		{"four per line", "Space", 4, "48 65 6c 6c \n6f 20 57 6f \n72 6c 64 21"},
		{"one per line", "Space", 1, "48 \n65 \n6c \n6c \n6f \n20 \n57 \n6f \n72 \n6c \n64 \n21"},
		{"zero means no breaks", "Space", 0, "48 65 6c 6c 6f 20 57 6f 72 6c 64 21"},
		{"no delimiter", "None", 4, "48656c6c\n6f20576f\n726c6421"},
		{"comma", "Comma", 3, "48,65,6c,\n6c,6f,20,\n57,6f,72,\n6c,64,21"},
		{"prepended delimiter", "0x", 4, "0x480x650x6c0x6c\n0x6f0x200x570x6f\n0x720x6c0x640x21"},
		{
			"prepended with extra", "0x with comma", 4,
			"0x48,0x65,0x6c,0x6c,\n0x6f,0x20,0x57,0x6f,\n0x72,0x6c,0x64,0x21",
		},
		{"line size divides exactly", "Space", 8, "48 65 6c 6c 6f 20 57 6f \n72 6c 64 21"},
		{"line size beyond the data", "Space", 16, "48 65 6c 6c 6f 20 57 6f 72 6c 64 21"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := runOp(t, "To Hex", input, c.delim, c.per)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if got != c.want {
				t.Errorf("got %q\nwant %q", got, c.want)
			}
		})
	}
}

// TestToHexDefaultsToNoLineBreaks checks that leaving the argument off behaves
// as it always has, so existing recipes are unaffected.
func TestToHexDefaultsToNoLineBreaks(t *testing.T) {
	got, err := runOp(t, "To Hex", "Hello World!", "Space")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got != "48 65 6c 6c 6f 20 57 6f 72 6c 64 21" {
		t.Errorf("got %q", got)
	}
}
