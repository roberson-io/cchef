package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

// TestDishTextLatin1Fallback pins how a dish is read as text: valid UTF-8 as
// itself, anything else one character per byte. Reading 0xFF as U+FFFD would
// lose its value, which is what these operations then report.
func TestDishTextLatin1Fallback(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"hello", "hello"},
		{"\xff", "ÿ"},
		{"\xff\xfe", "ÿþ"},
		{"H\xffI", "HÿI"},
		{"\xc3\xbf", "ÿ"}, // valid UTF-8 for U+00FF, read as itself
		{"€", "€"},
		{"", ""},
	} {
		if got := dishText(core.NewDish([]byte(tc.in), core.TypeString)); got != tc.want {
			t.Errorf("dishText(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestOperationsReadTextLikeCyberChef covers the operations that walk their
// input as text. Each value is what CyberChef returns for the same bytes.
func TestOperationsReadTextLikeCyberChef(t *testing.T) {
	for _, tc := range []struct {
		op, in, want string
		args         []any
	}{
		{"To Charcode", "\xff", "ff", []any{"Space", float64(16)}},
		{"To Charcode", "\xff\xfe", "ff fe", []any{"Space", float64(16)}},
		{"To HTML Entity", "\xff", "&yuml;", nil},
		// A code point that stays within a byte is written as that single byte
		// (Latin-1), matching CyberChef's Utils.strToArrayBuffer, not its 2-byte
		// UTF-8 encoding.
		{"To Lower case", "\xff", "\xff", nil},
		{"Escape Smart Characters", "\xff", "\xff", nil},
		{"Convert to NATO alphabet", "H\xffI", "Hotel \xffIndia ", nil},
		{"Substitute", "\xe9", "\xe9", []any{"A", "B", false}},
		{"Swap case", "\xe9", "\xc9", nil},              // é -> É, one byte
		{"To Upper case", "\xe9", "\xc9", []any{"All"}}, // é -> É, one byte
		// Operations that walk the input at code-point level must read 0xe9 as
		// Latin-1 é (not U+FFFD) and write it back as the single byte 0xe9.
		{"Convert Leet Speak", "\xe9", "\xe9", []any{"To Leet Speak"}},
		{"Remove Diacritics", "\xe9", "e", nil}, // é decomposes; the accent is stripped
		{"To Camel case", "\xe9", "e", []any{false}},
		{"To Kebab case", "\xe9", "e", []any{false}},
		{"To Snake case", "\xe9", "e", []any{false}},
		{"To Braille", "\xe9", "\xe9", nil},
		{"From Braille", "\xe9", "\xe9", nil},
		{"Get All Casings", "\xe9", "\xe9\n\xc9", nil},
		{"To Case Insensitive Regex", "\xe9", "\xe9", nil},
		{"Expand alphabet range", "\xe9", "\xe9", []any{"-"}},
		{"Alternating Caps", "\xe9", "\xe9", nil},
		{"Wrap", "\xe9", "\xe9", []any{float64(16)}},
		{"Escape Unicode Characters", "\xe9", `\u00E9`, []any{`\u`, false, float64(4), true}},
		{
			"Format MAC addresses", "\xe9", "\xe9\n\xc9\n\xe9\n\xc9\n\xe9\n\xc9\n",
			[]any{"Both", true, true, true, false, false},
		},
		{"To Punycode", "\xe9", "9ca", []any{false}},
		{"Citrix CTX1 Encode", "\xe9", "EMOJ", nil},
		{
			"RAKE", "\xe9", "Scores: , Keywords: \n1, \xe9",
			[]any{"Space", "Line feed", "English (English)"},
		},
		// Upper-casing ÿ (U+00FF) yields Ÿ (U+0178), which is above a byte and so
		// is written as its UTF-8 encoding.
		{"To Upper case", "\xff", "Ÿ", []any{"All"}},
		{"To Upper case", "H\xffI", "HŸI", []any{"All"}},
	} {
		got, err := runOp(t, tc.op, tc.in, tc.args...)
		if err != nil {
			t.Errorf("%s(%q): %v", tc.op, tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("%s(%q) = %q, want %q", tc.op, tc.in, got, tc.want)
		}
	}
}

// TestFromCharcodeEmitsBytes covers charcodes above 127: each is one byte, as
// CyberChef writes them, not its UTF-8 encoding.
func TestFromCharcodeEmitsBytes(t *testing.T) {
	for _, tc := range []struct{ in, wantHex string }{
		{"ff", "ff"},
		{"ff fe", "fffe"},
		{"48 ff 49", "48ff49"},
		{"20ac", "e282ac"}, // above a byte, so UTF-8
	} {
		got, err := runOp(t, "From Charcode", tc.in, "Space", float64(16))
		if err != nil {
			t.Errorf("%q: %v", tc.in, err)
			continue
		}
		hex, err := runOp(t, "To Hex", got, "None")
		if err != nil {
			t.Fatal(err)
		}
		if hex != tc.wantHex {
			t.Errorf("From Charcode(%q) = %s, want %s", tc.in, hex, tc.wantHex)
		}
	}
}
