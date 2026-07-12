package ops

// Tests for the Decode text / Encode text operations.
//
// CyberChef's ops wrap the `codepage` (cptable) npm library over a 152-charset
// table; cchef backs the supported subset with golang.org/x/text. There are no
// upstream fixtures, so the vectors below were generated from CyberChef via the
// CyberChef-server oracle and cover the charset families plus the round-trip and
// unmappable-character behaviour. Ordinary tests — edit as needed.

import (
	"bytes"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// TestDecodeTextVectors decodes representative graphic-character byte sequences
// for each charset family, oracle-verified against CyberChef.
func TestDecodeTextVectors(t *testing.T) {
	cases := []struct{ enc, hexin, want string }{
		{"IBM EBCDIC US-Canada (37)", "c8 85 93 93 96", "Hello"},
		{"OEM United States (437)", "48 65 9a 6c 6c 6f", "HeÜllo"},
		{"Windows-1252 Latin (1252)", "80 e9 ae", "€é®"},
		{"Windows-1251 Cyrillic (1251)", "cf f0 e8 e2 e5 f2", "Привет"},
		{"ISO-8859-1 Latin 1 Western European (28591)", "a9 e9 ff", "©éÿ"},
		{"ISO-8859-5 Latin/Cyrillic (28595)", "bf e0 d8 d2 d5 e2", "Привет"},
		{"ISO-8859-7 Latin/Greek (28597)", "c3 e5 e9 e1", "Γεια"},
		{"ISO-8859-15 Latin 9 (28605)", "a4 e9", "€é"},
		{"KOI8-R Russian Cyrillic (20866)", "f0 d2 c9 d7 c5 d4", "Привет"},
		{"Windows-874 Thai (874)", "ca c7 d1 ca b4 d5", "สวัสดี"},
		{"Japanese Shift-JIS (932)", "82 a0 41", "あA"},
		{"Simplified Chinese GBK (936)", "d6 d0 ce c4", "中文"},
		{"Traditional Chinese Big5 (950)", "a4 a4 a4 e5", "中文"},
		{"EUC Japanese (51932)", "c6 fc cb dc b8 ec", "日本語"},
		{"EUC Korean (51949)", "c7 d1 b1 b9 be ee", "한국어"},
		{"UTF-8 (65001)", "63 61 66 c3 a9 f0 9f 98 80", "café😀"},
		{"UTF-16LE (1200)", "48 00 69 00", "Hi"},
		{"UTF-16BE (1201)", "00 48 00 69", "Hi"},
		{"UTF-32LE (12000)", "41 00 00 00 00 f6 01 00", "A😀"},
		{"UTF-32BE (12001)", "00 00 00 41 00 01 f6 00", "A😀"},
	}
	for _, c := range cases {
		t.Run(c.enc, func(t *testing.T) {
			out, err := DecodeText{}.Run(abytes(string(cborBytes(t, c.hexin))), []any{c.enc})
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			if out.String() != c.want {
				t.Fatalf("got %q want %q", out.String(), c.want)
			}
		})
	}
}

// TestEncodeTextVectors encodes representative strings, oracle-verified.
func TestEncodeTextVectors(t *testing.T) {
	cases := []struct{ in, enc, hexout string }{
		{"Héllo €", "Windows-1252 Latin (1252)", "48 e9 6c 6c 6f 20 80"},
		{"Привет", "Windows-1251 Cyrillic (1251)", "cf f0 e8 e2 e5 f2"},
		{"café", "ISO-8859-1 Latin 1 Western European (28591)", "63 61 66 e9"},
		{"日本語", "Japanese Shift-JIS (932)", "93 fa 96 7b 8c ea"},
		{"中文", "Simplified Chinese GBK (936)", "d6 d0 ce c4"},
		{"中文", "Traditional Chinese Big5 (950)", "a4 a4 a4 e5"},
		{"한국어", "EUC Korean (51949)", "c7 d1 b1 b9 be ee"},
		{"café😀", "UTF-8 (65001)", "63 61 66 c3 a9 f0 9f 98 80"},
		{"Hi", "UTF-16LE (1200)", "48 00 69 00"},
		{"Hello", "IBM EBCDIC US-Canada (37)", "c8 85 93 93 96"},
		// Unmappable characters become 0x00 per UTF-16 code unit (cptable behaviour).
		{"€", "ISO-8859-1 Latin 1 Western European (28591)", "00"},
		{"😀", "ISO-8859-1 Latin 1 Western European (28591)", "00 00"},
		{"a中b", "ISO-8859-1 Latin 1 Western European (28591)", "61 00 62"},
	}
	for _, c := range cases {
		t.Run(c.enc+"/"+c.in, func(t *testing.T) {
			out, err := EncodeText{}.Run(sdish(c.in), []any{c.enc})
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			if want := cborBytes(t, c.hexout); !bytes.Equal(out.Bytes(), want) {
				t.Fatalf("got %x want %x", out.Bytes(), want)
			}
		})
	}
}

// TestTextRoundTrip encodes then decodes an ASCII string through every supported
// charset (ASCII is representable in all of them) and expects the original back.
func TestTextRoundTrip(t *testing.T) {
	const sample = "Hello, World! 123"
	for _, e := range textEncodings {
		t.Run(e.name, func(t *testing.T) {
			enc, err := EncodeText{}.Run(sdish(sample), []any{e.name})
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			dec, err := DecodeText{}.Run(abytes(string(enc.Bytes())), []any{e.name})
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			if dec.String() != sample {
				t.Fatalf("round-trip = %q", dec.String())
			}
		})
	}
}

// TestTextViaRecipe runs both operations through the engine, covering Args and
// the option-list argument coercion.
func TestTextViaRecipe(t *testing.T) {
	enc, err := core.Recipe{
		{Op: "Encode text", Args: []any{"UTF-16LE (1200)"}},
		{Op: "To Hex", Args: []any{"None"}},
	}.Execute(sdish("Hi"))
	if err != nil {
		t.Fatal(err)
	}
	if enc.String() != "48006900" {
		t.Fatalf("encode via recipe = %q", enc.String())
	}
	dec, err := core.Recipe{
		{Op: "From Hex"},
		{Op: "Decode text", Args: []any{"UTF-16LE (1200)"}},
	}.Execute(sdish("48 00 69 00"))
	if err != nil {
		t.Fatal(err)
	}
	if dec.String() != "Hi" {
		t.Fatalf("decode via recipe = %q", dec.String())
	}
}

// TestTextInvalidEncoding covers the guard for an unknown charset name (reachable
// only by bypassing the option-list validation, as CoerceArgs normally rejects
// it first).
func TestTextInvalidEncoding(t *testing.T) {
	if _, err := (DecodeText{}).Run(abytes("x"), []any{"Bogus (99999)"}); err == nil {
		t.Fatal("decode: expected error for unknown encoding")
	}
	if _, err := (EncodeText{}).Run(sdish("x"), []any{"Bogus (99999)"}); err == nil {
		t.Fatal("encode: expected error for unknown encoding")
	}
}
