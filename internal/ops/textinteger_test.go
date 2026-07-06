package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// Transcribed from ../CyberChef/tests/operations/tests/TextIntegerConverter.mjs.
// (The oracle image predates this 2025 op, so the upstream fixtures are the
// authoritative source.)
func TestTextIntegerFixtures(t *testing.T) {
	dec := core.Recipe{{Op: "Text-Integer Conversion", Args: []any{"Decimal"}}}
	hexOut := core.Recipe{{Op: "Text-Integer Conversion", Args: []any{"Hexadecimal"}}}
	str := core.Recipe{{Op: "Text-Integer Conversion", Args: []any{"String"}}}

	runCases(t, []opCase{
		{"quoted string to decimal", `"ABC"`, "4276803", dec},
		{"quoted string to hexadecimal", `"ABC"`, "0x414243", hexOut},
		{"single quoted string to decimal", `'Hello'`, "310939249775", dec},
		{"decimal to string", "4276803", "ABC", str},
		{"hexadecimal to string", "0x48656C6C6F", "Hello", str},
		{
			"round-trip string.decimal.string", `"Test"`, "Test",
			core.Recipe{
				{Op: "Text-Integer Conversion", Args: []any{"Decimal"}},
				{Op: "Text-Integer Conversion", Args: []any{"String"}},
			},
		},
		{
			"round-trip string.hex.string", `"CyberChef"`, "CyberChef",
			core.Recipe{
				{Op: "Text-Integer Conversion", Args: []any{"Hexadecimal"}},
				{Op: "Text-Integer Conversion", Args: []any{"String"}},
			},
		},
		// Fixture 8 chains Unescape/Escape Unicode (not yet ported); this exercises
		// the same Latin-1 round-trip directly: U+00FF is a single 0xFF byte value.
		{"Latin-1 round-trip through String", "ÿ", "ÿ", str},
		{"unquoted text to decimal", "Hi", "18537", dec},
		{"single character", `"A"`, "65", dec},
		{"hex to decimal conversion", "0xFF", "255", dec},
		{"decimal to hex conversion", "255", "0xff", hexOut},
		{
			"large number to string",
			"113091951015816448506195587157728348242683688608116",
			"Mary had a little cat", str,
		},
		{"whitespace handling (quoted)", `"  test  "`, "2314978187545944096", dec},
	})
}

// TestTextIntegerBranches covers the empty-input (treated as zero) paths for each
// output format and the multi-byte (non-Latin-1) error, which the transcribable
// fixtures do not directly reach in cchef.
func TestTextIntegerBranches(t *testing.T) {
	runCases(t, []opCase{
		{"empty input to decimal", "", "0", core.Recipe{{Op: "Text-Integer Conversion", Args: []any{"Decimal"}}}},
		{"empty input to hexadecimal", "", "0x0", core.Recipe{{Op: "Text-Integer Conversion", Args: []any{"Hexadecimal"}}}},
		{"empty input to string", "", "", core.Recipe{{Op: "Text-Integer Conversion", Args: []any{"String"}}}},
	})

	// A multi-byte Unicode character (Γ, U+0393) exceeds the Latin-1 range and
	// must error, both as unquoted text and inside a quoted string.
	for _, in := range []string{"aΓa", `"aΓa"`} {
		if _, err := runOp(t, "Text-Integer Conversion", in, "Decimal"); err == nil {
			t.Fatalf("expected an error for a non-Latin-1 character in %q", in)
		}
	}
}
