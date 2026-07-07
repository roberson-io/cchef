package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// Cases transcribed from CyberChef tests/operations/tests/ByteRepr.mjs.
// Charcode is a text operation: input is decoded as UTF-8 to Unicode code
// points (so the ALL_BYTES string-semantics fixtures, which require raw bytes
// 0x80-0xFF that aren't valid UTF-8, don't apply to cchef's byte pipeline).
func TestCharcodeFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"To Charcode: nothing", "", "",
			core.Recipe{{Op: "To Charcode", Args: []any{"Space", 16}}},
		},
		{
			"To Charcode: UTF-8", "ნუ პანიკას", "10dc 10e3 20 10de 10d0 10dc 10d8 10d9 10d0 10e1",
			core.Recipe{{Op: "To Charcode", Args: []any{"Space", 16}}},
		},
		{
			// Mixed code-point widths exercise every reachable charcodeHexPad
			// branch: A=0x41 (pad 2), й=0x439 and €=0x20ac (pad 4), 😀=0x1f600
			// (pad 6). Verified against the CyberChef-server oracle.
			"To Charcode: mixed hex widths", "Aй€😀", "41 0439 20ac 01f600",
			core.Recipe{{Op: "To Charcode", Args: []any{"Space", 16}}},
		},

		{
			"From Charcode: UTF-8", "10dc 10e3 20 10de 10d0 10dc 10d8 10d9 10d0 10e1", "ნუ პანიკას",
			core.Recipe{{Op: "From Charcode", Args: []any{"Space", 16}}},
		},

		// Base 10 charcodes.
		{
			"To Charcode: base 10", "ABC", "65 66 67",
			core.Recipe{{Op: "To Charcode", Args: []any{"Space", 10}}},
		},

		{
			"Charcode round trip", "Hello, World!", "Hello, World!",
			core.Recipe{
				{Op: "To Charcode", Args: []any{"Comma", 16}},
				{Op: "From Charcode", Args: []any{"Comma", 16}},
			},
		},
		{
			// A long, undelimited string (>17 chars, delimiter not found) is split
			// into byte pairs, matching CyberChef. "48656c..." -> "Hello, World!".
			"From Charcode: undelimited pairs", "48656c6c6f2c20576f726c6421", "Hello, World!",
			core.Recipe{{Op: "From Charcode", Args: []any{"Space", 16}}},
		},
	})
}

func TestCharcodeBranches(t *testing.T) {
	if _, err := runOp(t, "To Charcode", "a", "Space", 99.0); err == nil {
		t.Fatal("To Charcode: expected an error for out-of-range base")
	}
	if _, err := runOp(t, "From Charcode", "61", "Space", 99.0); err == nil {
		t.Fatal("From Charcode: expected an error for out-of-range base")
	}
	if _, err := runOp(t, "From Charcode", "zz", "Space", 16.0); err == nil {
		t.Fatal("From Charcode: expected an error for an invalid code")
	}
	if out, err := runOp(t, "From Charcode", "", "Space", 16.0); err != nil || out != "" {
		t.Fatalf("From Charcode(\"\") = %q, %v; want empty", out, err)
	}
	if _, err := runOp(t, "From Charcode", "61  62", "Space", 16.0); err != nil {
		t.Fatalf("From Charcode(empty token): %v", err)
	}
	// charcodeHexPad's wide branches are unreachable via the op (code points cap at
	// 0x10FFFF) but the helper's contract holds for any caller.
	if got := charcodeHexPad(16777216); got != 8 {
		t.Fatalf("charcodeHexPad(1<<24) = %d, want 8", got)
	}
	if got := charcodeHexPad(4294967296); got != 2 {
		t.Fatalf("charcodeHexPad(1<<32) = %d, want 2", got)
	}
}
