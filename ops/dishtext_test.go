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
		{"To Upper case", "\xff", "Ÿ", []any{"All"}},
		{"To Upper case", "H\xffI", "HŸI", []any{"All"}},
		{"To Lower case", "\xff", "ÿ", nil},
		{"Escape Smart Characters", "\xff", "ÿ", nil},
		{"Convert to NATO alphabet", "H\xffI", "Hotel ÿIndia ", nil},
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
