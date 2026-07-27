package ops

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// qrDecodeCase is one of CyberChef's own decode fixtures: an image, the text it
// carries, and whether the image is normalised before it is read.
type qrDecodeCase struct {
	Name      string `json:"name"`
	Input     string `json:"input"` // the image in hexadecimal
	Want      string `json:"want"`
	Normalise bool   `json:"normalise"`
}

// TestParseQRCodeFixtures covers CyberChef's four cases: a JPEG, a PNG, a PNG
// whose background is transparent, and a code photographed at an angle.
func TestParseQRCodeFixtures(t *testing.T) {
	file, err := os.Open("testdata/parse_qr_code.jsonl")
	if err != nil {
		t.Fatalf("open fixtures: %v", err)
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for scanner.Scan() {
		if len(scanner.Bytes()) == 0 {
			continue
		}
		var c qrDecodeCase
		if err := json.Unmarshal(scanner.Bytes(), &c); err != nil {
			t.Fatalf("parse fixture: %v", err)
		}

		t.Run(c.Name, func(t *testing.T) {
			image, err := hex.DecodeString(c.Input)
			if err != nil {
				t.Fatalf("decode image: %v", err)
			}
			recipe := core.Recipe{{Op: "Parse QR Code", Args: []any{c.Normalise}}}
			out, err := recipe.Execute(core.NewDish(image, core.TypeArrayBuffer))
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if out.String() != c.Want {
				t.Errorf("read %q, want %q", out.String(), c.Want)
			}
		})
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("read fixtures: %v", err)
	}
}

// TestParseQRCodeRejections covers the inputs the operation refuses: something
// that is not an image at all, and an image carrying no code.
func TestParseQRCodeRejections(t *testing.T) {
	for _, tc := range []struct{ name, input string }{
		{"not an image", "68656c6c6f20776f726c64"},
		{"an image with no code", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var image []byte
			if tc.input != "" {
				image, _ = hex.DecodeString(tc.input)
			} else {
				image = qrBlankPNG(t)
			}
			recipe := core.Recipe{{Op: "Parse QR Code", Args: []any{false}}}
			if _, err := recipe.Execute(core.NewDish(image, core.TypeArrayBuffer)); err == nil {
				t.Error("accepted input carrying no QR code")
			}
		})
	}
}

// qrBlankPNG builds a plain white image, which carries no code to find.
func qrBlankPNG(t *testing.T) []byte {
	t.Helper()
	out, err := convertImage(qrRenderPNG([][]byte{{0}}, 8, 4), "PNG", 80, 9)
	if err != nil {
		t.Fatalf("build a blank image: %v", err)
	}
	return out
}

// TestParseQRCodeRoundTrip reads back what the generator produces, over every
// encoding mode, correction level and a spread of versions. The two operations
// share no code beyond the field arithmetic, so agreement between them is
// evidence for both.
func TestParseQRCodeRoundTrip(t *testing.T) {
	texts := map[string]string{
		"numeric":              "8675309",
		"a long numeric run":   strings.Repeat("0123456789", 30),
		"alphanumeric":         "HELLO WORLD $%*+-./:",
		"a long alphanumeric":  strings.Repeat("ABC123 ", 40),
		"bytes":                "Hello world!",
		"mixed case and marks": "The quick brown fox; jumps over 13 lazy dogs!",
		"a long byte run":      strings.Repeat("cchef ", 90),
		// The remainders each mode leaves when its groups do not divide the
		// length evenly, and a version past the widest character count field.
		"numeric leaving two digits": "12345678",
		"numeric leaving one digit":  "1234567",
		"alphanumeric of odd length": "ABCDE",
		"a version past twenty-six":  strings.Repeat("0123456789", 130),
	}
	for _, level := range []string{"Low", "Medium", "Quartile", "High"} {
		for name, text := range texts {
			t.Run(name+" at "+level, func(t *testing.T) {
				generate := core.Recipe{{
					Op:   "Generate QR Code",
					Args: []any{"PNG", float64(4), float64(4), level},
				}}
				image, err := generate.Execute(core.NewDish([]byte(text), core.TypeString))
				if err != nil {
					t.Fatalf("generate: %v", err)
				}

				parse := core.Recipe{{Op: "Parse QR Code", Args: []any{false}}}
				out, err := parse.Execute(core.NewDish(image.Bytes(), core.TypeArrayBuffer))
				if err != nil {
					t.Fatalf("parse: %v", err)
				}
				if out.String() != text {
					t.Errorf("read %q, want %q", out.String(), text)
				}
			})
		}
	}
}

// TestParseQRCodeCorrectsDamage covers the error correction, which an undamaged
// code never exercises: the syndromes come out zero and the block is returned
// untouched. Blocks of modules are painted over at each correction level, up to
// roughly what that level is meant to withstand.
func TestParseQRCodeCorrectsDamage(t *testing.T) {
	const text = "Hello world! This is a somewhat longer message."
	for _, tc := range []struct {
		level  string
		blocks int
	}{
		{"Low", 2}, {"Medium", 4}, {"Quartile", 6}, {"High", 8},
	} {
		t.Run(tc.level, func(t *testing.T) {
			generate := core.Recipe{{
				Op:   "Generate QR Code",
				Args: []any{"PNG", float64(4), float64(4), tc.level},
			}}
			out, err := generate.Execute(core.NewDish([]byte(text), core.TypeString))
			if err != nil {
				t.Fatalf("generate: %v", err)
			}

			damaged, err := qrDamage(out.Bytes(), tc.blocks)
			if err != nil {
				t.Fatalf("damage the image: %v", err)
			}

			parse := core.Recipe{{Op: "Parse QR Code", Args: []any{false}}}
			read, err := parse.Execute(core.NewDish(damaged, core.TypeArrayBuffer))
			if err != nil {
				t.Fatalf("parse a code with %d damaged blocks: %v", tc.blocks, err)
			}
			if read.String() != text {
				t.Errorf("read %q, want %q", read.String(), text)
			}
		})
	}
}

// qrDamage paints blocks of modules over the middle of a code, leaving its
// finder patterns and the quiet zone alone.
func qrDamage(png []byte, blocks int) ([]byte, error) {
	img, _, err := decodeImageNRGBA(png, "Invalid file type.")
	if err != nil {
		return nil, err
	}

	const module = 4 // the size the codes above are generated at
	side := img.Bounds().Dx()
	start := side / 3 // clear of the finder patterns in every corner
	for b := range blocks {
		left := start + b*module*2
		top := start + (b%3)*module*2
		for y := top; y < top+module && y < side; y++ {
			for x := left; x < left+module && x < side; x++ {
				i := img.PixOffset(x, y)
				img.Pix[i], img.Pix[i+1], img.Pix[i+2] = 0, 0, 0
			}
		}
	}
	return encodeConverted(img, "PNG", 100, 9)
}
