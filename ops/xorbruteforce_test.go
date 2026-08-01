package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

// TestXORBruteForceOracle checks XOR Brute Force against outputs captured from
// the CyberChef-server oracle (v11.2.0); CyberChef ships no fixture file for
// this op. Each input is a known plaintext XORed with a single-byte key, fed in
// via From Hex so the bytes are exact, then brute-forced back with a crib. The
// crib match is case-insensitive, so key 0x0a (which upper-cases the text) is
// reported alongside the true key 0x2a.
func TestXORBruteForceOracle(t *testing.T) {
	fromHex := core.RecipeOp{Op: "From Hex", Args: []any{"None"}}
	// args: [keyLength, sampleLength, sampleOffset, scheme, nullPreserving, printKey, outputHex, crib]
	xbf := func(a ...any) core.RecipeOp { return core.RecipeOp{Op: "XOR Brute Force", Args: a} }

	runCases(t, []opCase{
		{
			"XORBF: hex output, print key", "424f4646450a5d4558464e",
			"Key = 0a: 48 45 4c 4c 4f 00 57 4f 52 4c 44\nKey = 2a: 68 65 6c 6c 6f 20 77 6f 72 6c 64",
			core.Recipe{fromHex, xbf(1, 100, 0, "Standard", false, true, true, "hello")},
		},

		{
			"XORBF: text output, print key (null byte)", "424f4646450a5d4558464e",
			"Key = 0a: HELLO\x00WORLD\nKey = 2a: hello world",
			core.Recipe{fromHex, xbf(1, 100, 0, "Standard", false, true, false, "hello")},
		},

		{
			"XORBF: text output, no key", "424f4646450a5d4558464e",
			"HELLO\x00WORLD\nhello world",
			core.Recipe{fromHex, xbf(1, 100, 0, "Standard", false, false, false, "hello")},
		},

		// Tab (0x09) is escaped to U+E009 by escapeWhitespace.
		{
			"XORBF: whitespace escaping", "5e4b4823424f584f",
			"Key = 0a: TAB)HERE\nKey = 2a: tabhere",
			core.Recipe{fromHex, xbf(1, 100, 0, "Standard", false, true, false, "here")},
		},

		// A result byte that is not valid UTF-8 falls back to per-byte chars
		// (0xdf -> U+00DF, 0xff -> U+00FF).
		{
			"XORBF: invalid-UTF8 fallback", "6243d5",
			"hIß\nHiÿ",
			core.Recipe{fromHex, xbf(1, 100, 0, "Standard", false, false, false, "hi")},
		},
	})
}

func TestXORBruteForceOffsetClamp(t *testing.T) {
	if _, err := runOp(t, "XOR Brute Force", "abcdef", 1.0, 100.0, -5.0, "Standard", false, true, false, ""); err != nil {
		t.Fatalf("XOR Brute Force negative offset: %v", err)
	}
	if _, err := runOp(t, "XOR Brute Force", "abc", 1.0, 100.0, 999.0, "Standard", false, true, false, ""); err != nil {
		t.Fatalf("XOR Brute Force large offset: %v", err)
	}
}
