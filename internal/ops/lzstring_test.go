package ops

import (
	"encoding/binary"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// lzsGolden is one case from testdata/lzstring.jsonl: an input, and what
// lz-string compresses it into under each of the three formats.
//
// The compressed form is a run of 16-bit code units, so it is recorded as those
// units rather than as text: under the plain format about a quarter of these
// cases come out holding a surrogate with no partner, which has no character to
// be written as.
//
// Most cases were recorded from lz-string itself. The two whose input holds a
// lone surrogate were recorded from CyberChef instead, because an operation is
// handed bytes rather than a JavaScript string: a lone surrogate cannot reach
// it as one character, so those bytes are read one per character and compress
// to something the bare library never sees.
type lzsGolden struct {
	Name     string `json:"name"`
	InputHex string `json:"inputHex"`

	DefaultHex              string `json:"defaultHex"`
	DefaultUnits            int    `json:"defaultUnits"`
	DefaultSHA256           string `json:"defaultSHA256"`
	DefaultHasLoneSurrogate bool   `json:"defaultHasLoneSurrogate"`

	UTF16Hex    string `json:"utf16Hex"`
	UTF16Units  int    `json:"utf16Units"`
	UTF16SHA256 string `json:"utf16SHA256"`

	Base64Hex    string `json:"base64Hex"`
	Base64Units  int    `json:"base64Units"`
	Base64SHA256 string `json:"base64SHA256"`
}

// want returns what a format should compress this case into.
func (g lzsGolden) want(format string) (hexOf string, units int, sum string) {
	switch format {
	case "UTF16":
		return g.UTF16Hex, g.UTF16Units, g.UTF16SHA256
	case "Base64":
		return g.Base64Hex, g.Base64Units, g.Base64SHA256
	}
	return g.DefaultHex, g.DefaultUnits, g.DefaultSHA256
}

// lzsFormats are the three the operations offer.
var lzsFormats = []string{"default", "UTF16", "Base64"}

// unitsHex renders code units the way the corpus records them.
func unitsHex(units []uint16) string {
	b := make([]byte, 2*len(units))
	for i, u := range units {
		binary.BigEndian.PutUint16(b[2*i:], u)
	}
	return hex.EncodeToString(b)
}

// hexUnits reads back what unitsHex wrote.
func hexUnits(t *testing.T, s string) []uint16 {
	t.Helper()
	b := unhex(t, s)
	units := make([]uint16, len(b)/2)
	for i := range units {
		units[i] = binary.BigEndian.Uint16(b[2*i:])
	}
	return units
}

// TestLZStringFixtures covers CyberChef's own cases
// (../CyberChef/tests/operations/tests/LZString.mjs).
func TestLZStringFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"LZString Compress To Base64", "hello world", "BYUwNmD2AEDukCcwBMg=",
			core.Recipe{{Op: "LZString Compress", Args: []any{"Base64"}}},
		},
		{
			"LZString Decompress From Base64", "BYUwNmD2AEDukCcwBMg=", "hello world",
			core.Recipe{{Op: "LZString Decompress", Args: []any{"Base64"}}},
		},
	})
}

// TestLZStringCompressGoldens compresses each case under each format and
// compares the code units with lz-string's own.
func TestLZStringCompressGoldens(t *testing.T) {
	for _, g := range readJSONL[lzsGolden](t, "testdata/lzstring.jsonl") {
		for _, format := range lzsFormats {
			t.Run(g.Name+" ("+format+")", func(t *testing.T) {
				out, err := runOp(t, "LZString Compress", string(unhex(t, g.InputHex)), format)
				if err != nil {
					t.Fatalf("LZString Compress: %v", err)
				}
				got := unitsHex(lzsStringToUnits(out))
				wantHex, wantUnits, wantSum := g.want(format)
				if wantHex != "" {
					if got != wantHex {
						t.Fatalf("got  %s\nwant %s", got, wantHex)
					}
					return
				}
				if n := len(got) / 4; n != wantUnits {
					t.Errorf("gave %d code units, want %d", n, wantUnits)
				}
				if sum := digest([]byte(got)); sum != wantSum {
					t.Errorf("digest %s, want %s", sum, wantSum)
				}
			})
		}
	}
}

// TestLZStringDecompressGoldens reads back the compressed forms lz-string
// produced, for the cases the corpus records whole.
func TestLZStringDecompressGoldens(t *testing.T) {
	for _, g := range readJSONL[lzsGolden](t, "testdata/lzstring.jsonl") {
		if g.InputHex == "" {
			continue // nothing compresses to a stream that reads back as itself
		}
		for _, format := range lzsFormats {
			hexOf, _, _ := g.want(format)
			if hexOf == "" {
				continue
			}
			t.Run(g.Name+" ("+format+")", func(t *testing.T) {
				stream := lzsUnitsToString(hexUnits(t, hexOf))
				out, err := runOp(t, "LZString Decompress", stream, format)
				if err != nil {
					t.Fatalf("LZString Decompress: %v", err)
				}
				// The fixture records the input as the UTF-8 text lz-string was
				// given. What comes back out of an operation is bytes, and
				// CyberChef writes a string of characters that all fit in a
				// byte as one byte each, so the expectation is converted the
				// same way before comparing.
				if want := string(textAsBytes(string(unhex(t, g.InputHex)))); out != want {
					t.Errorf("got %q, want %q", out, want)
				}
			})
		}
	}
}

// TestLZStringRoundTrips puts every case through both operations under every
// format, which covers the cases the corpus records only as a digest.
func TestLZStringRoundTrips(t *testing.T) {
	for _, g := range readJSONL[lzsGolden](t, "testdata/lzstring.jsonl") {
		if g.InputHex == "" {
			continue
		}
		for _, format := range lzsFormats {
			t.Run(g.Name+" ("+format+")", func(t *testing.T) {
				in := string(unhex(t, g.InputHex))
				stream, err := runOp(t, "LZString Compress", in, format)
				if err != nil {
					t.Fatalf("LZString Compress: %v", err)
				}
				back, err := runOp(t, "LZString Decompress", stream, format)
				if err != nil {
					t.Fatalf("LZString Decompress: %v", err)
				}
				// Text whose characters all fit in a byte comes back one byte
				// each, as CyberChef writes it, so that is what to compare to.
				want := string(textAsBytes(in))
				if back != want {
					t.Errorf("round trip changed the text (%d bytes in, %d out)",
						len(want), len(back))
				}
			})
		}
	}
}

// TestLZStringDefaultKeepsLoneSurrogates covers the cases whose plain-format
// output holds a surrogate with no partner — a quarter of the corpus. There is
// no character for one, so it is written as three bytes holding its own number
// and read back the same way. Writing strict UTF-8 would put a replacement
// character there instead and the text would not come back.
func TestLZStringDefaultKeepsLoneSurrogates(t *testing.T) {
	seen := 0
	for _, g := range readJSONL[lzsGolden](t, "testdata/lzstring.jsonl") {
		if !g.DefaultHasLoneSurrogate {
			continue
		}
		seen++
		t.Run(g.Name, func(t *testing.T) {
			in := string(unhex(t, g.InputHex))
			stream, err := runOp(t, "LZString Compress", in, "default")
			if err != nil {
				t.Fatalf("LZString Compress: %v", err)
			}
			// The units themselves are held to lz-string's own by
			// TestLZStringCompressGoldens; what matters here is that one of
			// them has no character to be written as, and that the text still
			// comes back.
			if !hasLoneSurrogate(lzsStringToUnits(stream)) {
				t.Fatal("no lone surrogate in the output, so this case proves nothing")
			}
			back, err := runOp(t, "LZString Decompress", stream, "default")
			if err != nil {
				t.Fatalf("LZString Decompress: %v", err)
			}
			if want := string(textAsBytes(in)); back != want {
				t.Errorf("round trip changed the text")
			}
		})
	}
	if seen == 0 {
		t.Fatal("no case in the corpus holds a lone surrogate")
	}
}

// hasLoneSurrogate reports whether any surrogate here is without its partner.
func hasLoneSurrogate(units []uint16) bool {
	for i := 0; i < len(units); i++ {
		u := units[i]
		if u < 0xD800 || u > 0xDFFF {
			continue
		}
		if u < 0xDC00 && i+1 < len(units) && units[i+1] >= 0xDC00 && units[i+1] <= 0xDFFF {
			i++
			continue
		}
		return true
	}
	return false
}

// TestLZStringCompressOfNothing covers empty input, which still carries the
// mark that ends a stream.
func TestLZStringCompressOfNothing(t *testing.T) {
	want := map[string]string{"default": "4000", "UTF16": "20200020", "Base64": "0051003d003d003d"}
	for _, format := range lzsFormats {
		t.Run(format, func(t *testing.T) {
			out, err := runOp(t, "LZString Compress", "", format)
			if err != nil {
				t.Fatalf("LZString Compress: %v", err)
			}
			if got := unitsHex(lzsStringToUnits(out)); got != want[format] {
				t.Errorf("got %s, want %s", got, want[format])
			}
		})
	}
}

// TestLZStringDecompressRejectsBadInput covers input that is not a stream, and
// one that stops before the mark that would end it. CyberChef refuses the first
// of these too; the second it hands back as far as it got, or as nothing.
func TestLZStringDecompressRejectsBadInput(t *testing.T) {
	cases := []struct {
		name   string
		format string
		in     string
	}{
		{"nothing at all", "default", ""},
		{"nothing at all", "UTF16", ""},
		{"nothing at all", "Base64", ""},
		{"not a stream", "Base64", "not a stream at all"},
		{"cut short", "Base64", "BYUwNmD2AEDukCcw"},
		{"a letter outside the alphabet", "Base64", "BYUw*mD2AEDukCcwBMg="},
		{"a character the UTF16 form never writes", "UTF16", "\x01\x02"},
		{"a mark that means nothing", "default", string(rune(0xC000))},
		{"an entry that is not there", "default", string(rune(0x21B8))},
	}
	for _, c := range cases {
		t.Run(c.name+" ("+c.format+")", func(t *testing.T) {
			if _, err := runOp(t, "LZString Decompress", c.in, c.format); err == nil {
				t.Fatal("read something that should have been refused")
			}
		})
	}
}

// TestLZStringDecompressOfAnEmptyStream covers a stream carrying nothing but
// the mark that ends it, which is what compressing nothing gives.
func TestLZStringDecompressOfAnEmptyStream(t *testing.T) {
	for _, format := range lzsFormats {
		t.Run(format, func(t *testing.T) {
			stream, err := runOp(t, "LZString Compress", "", format)
			if err != nil {
				t.Fatalf("LZString Compress: %v", err)
			}
			out, err := runOp(t, "LZString Decompress", stream, format)
			if err != nil {
				t.Fatalf("LZString Decompress: %v", err)
			}
			if out != "" {
				t.Errorf("got %q, want nothing", out)
			}
		})
	}
}

// TestLZStringBase64IsPadded covers the padding that makes the output valid
// Base64, which is added to whatever length the bits come to.
func TestLZStringBase64IsPadded(t *testing.T) {
	for _, in := range []string{"", "a", "ab", "abc", "hello world", strings.Repeat("x", 50)} {
		out, err := runOp(t, "LZString Compress", in, "Base64")
		if err != nil {
			t.Fatalf("LZString Compress: %v", err)
		}
		if len(out)%4 != 0 {
			t.Errorf("%q gave %d characters, which is not a multiple of four", in, len(out))
		}
		if strings.TrimRight(out, "=") == "" && in != "" {
			t.Errorf("%q gave nothing but padding", in)
		}
	}
}

// TestLZStringUTF16StaysInRange covers the format built to survive being stored
// as text: every character it writes sits above the control range and below the
// surrogates.
func TestLZStringUTF16StaysInRange(t *testing.T) {
	for _, g := range readJSONL[lzsGolden](t, "testdata/lzstring.jsonl") {
		t.Run(g.Name, func(t *testing.T) {
			out, err := runOp(t, "LZString Compress", string(unhex(t, g.InputHex)), "UTF16")
			if err != nil {
				t.Fatalf("LZString Compress: %v", err)
			}
			for _, u := range lzsStringToUnits(out) {
				if u < 32 || u >= 0xD800 {
					t.Fatalf("wrote code unit %#04x, outside the range the format keeps to", u)
				}
			}
		})
	}
}
