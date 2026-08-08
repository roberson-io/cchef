package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
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
		{
			// Whitespace around comma-separated codes is tolerated the way
			// JavaScript's parseInt is (" 101" -> 101), matching CyberChef.
			"From Charcode: comma-space tolerated", "104, 101, 108, 108, 111", "hello",
			core.Recipe{{Op: "From Charcode", Args: []any{"Comma", 10}}},
		},
	})
}

func TestJSParseInt(t *testing.T) {
	cases := []struct {
		s    string
		base int
		want int64
		ok   bool
	}{
		{"101", 10, 101, true},
		{"  101", 10, 101, true}, // leading whitespace skipped
		{"\t\n42", 10, 42, true}, // other whitespace forms
		{"104x", 10, 104, true},  // trailing junk ignored
		{"+65", 10, 65, true},    // leading plus
		{"-5", 10, -5, true},     // leading minus applied
		{"ff", 16, 255, true},    // base 16
		{"z", 36, 35, true},      // base 36 digit
		{"", 10, 0, false},       // empty -> NaN
		{"   ", 10, 0, false},    // whitespace only -> NaN
		{"xyz", 10, 0, false},    // no valid digit -> NaN
		{"9", 8, 0, false},       // digit out of range for base -> NaN
	}
	for _, c := range cases {
		got, ok := jsParseInt(c.s, c.base)
		if got != c.want || ok != c.ok {
			t.Errorf("jsParseInt(%q, %d) = (%d, %v); want (%d, %v)", c.s, c.base, got, ok, c.want, c.ok)
		}
	}
}

func TestCharcodeBranches(t *testing.T) {
	if _, err := runOp(t, "To Charcode", "a", "Space", 99.0); err == nil {
		t.Fatal("To Charcode: expected an error for out-of-range base")
	}
	if _, err := runOp(t, "From Charcode", "61", "Space", 99.0); err == nil {
		t.Fatal("From Charcode: expected an error for out-of-range base")
	}
	// A non-numeric token is parseInt's NaN, which Utils.chr renders as NUL
	// (matching CyberChef) rather than raising an error.
	if out, err := runOp(t, "From Charcode", "zz", "Space", 16.0); err != nil || out != "\x00" {
		t.Fatalf("From Charcode(\"zz\") = %q, %v; want a NUL byte", out, err)
	}
	// Empty input is a special case that yields empty output.
	if out, err := runOp(t, "From Charcode", "", "Space", 16.0); err != nil || out != "" {
		t.Fatalf("From Charcode(\"\") = %q, %v; want empty", out, err)
	}
	// An empty token between delimiters is a NUL, not skipped: "61  62" -> a NUL b.
	if out, err := runOp(t, "From Charcode", "61  62", "Space", 16.0); err != nil || out != "a\x00b" {
		t.Fatalf("From Charcode(empty token) = %q, %v; want \"a\\x00b\"", out, err)
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
