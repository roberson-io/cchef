package ops

import (
	"encoding/hex"
	"image"
	"image/color"
	"strings"
	"testing"
)

// The images here mirror the probe set sent through CyberChef's Node API, and
// every Row-order expectation below is the byte-exact answer it gave. The
// Column-order expectations are the corrected walk — upstream's column index
// arithmetic reads the wrong bytes (see the bug log) — verified bit for bit
// against the decoded bitmap.

// lsbSolid3x3 is a single colour throughout, alpha fully opaque.
func lsbSolid3x3() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, 3, 3))
	for i := range 9 {
		img.SetNRGBA(i%3, i/3, color.NRGBA{R: 200, G: 100, B: 50, A: 255})
	}
	return img
}

// lsbStripes8x2 alternates full-on and full-off red down the row.
func lsbStripes8x2() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, 8, 2))
	for y := range 2 {
		for x := range 8 {
			img.SetNRGBA(x, y, color.NRGBA{
				R: uint8(255 * (x % 2)), G: uint8(128 + 15*x), B: uint8(7 * (x + 1) % 256), A: 255,
			})
		}
	}
	return img
}

// lsbOnePixel is a 1x1 image with distinct channel values.
func lsbOnePixel() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 1, G: 2, B: 3, A: 4})
	return img
}

// runLSB runs Extract LSB and returns the extracted bytes as hex.
func runLSB(t *testing.T, img image.Image, args ...any) string {
	t.Helper()
	out, err := runOp(t, "Extract LSB", pngBytes(t, img), args...)
	if err != nil {
		t.Fatalf("Extract LSB: %v", err)
	}
	return hex.EncodeToString([]byte(out))
}

// TestExtractLSBRow covers Row order: single and several channels, channel
// order and repeats, every-bit selection, and a bit stream that is not a whole
// number of bytes (the final short group keeps its value, as CyberChef's
// fromBinary parses it).
func TestExtractLSBRow(t *testing.T) {
	cases := []struct {
		name string
		img  image.Image
		args []any
		want string
	}{
		{"single channel, bit 0", testImage(), []any{"R", "", "", "", "Row", 0.0}, "ffffff3f"},
		{"two channels", testImage(), []any{"R", "G", "", "", "Row", 0.0}, "aaaaaaaaaaaaaa0a"},
		{"all four channels", testImage(), []any{"R", "G", "B", "A", "Row", 0.0}, "abababb8b8b8abababb8b8b8ababab"},
		{"most significant bit", testImage(), []any{"R", "", "", "", "Row", 7.0}, "0c30c303"},
		{"middle bit", lsbStripes8x2(), []any{"R", "", "", "", "Row", 3.0}, "5555"},
		{"a channel named twice", lsbSolid3x3(), []any{"G", "G", "R", "", "Row", 0.0}, "00000000"},
		{"channels out of declared order", lsbSolid3x3(), []any{"B", "R", "", "", "Row", 1.0}, "aaaa02"},
		{"one pixel", lsbOnePixel(), []any{"R", "G", "B", "A", "Row", 7.0}, "00"},
		{"three channels leave a short final group", testImage(), []any{"R", "G", "B", "", "Row", 0.0}, "b6db6cb2cb6db6cb2cb6db01"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := runLSB(t, c.img, c.args...); got != c.want {
				t.Errorf("got %s, want %s", got, c.want)
			}
		})
	}
}

// TestExtractLSBColumn covers Column order, walking the pixels top to bottom
// one column at a time.
func TestExtractLSBColumn(t *testing.T) {
	cases := []struct {
		name string
		img  image.Image
		args []any
		want string
	}{
		{"single channel, bit 0", testImage(), []any{"R", "", "", "", "Column", 0.0}, "ffffff3f"},
		{"two channels", testImage(), []any{"R", "G", "", "", "Column", 0.0}, "aaaaaaaaaaaaaa0a"},
		{"all four channels", lsbSolid3x3(), []any{"R", "G", "B", "A", "Column", 0.0}, "1111111101"},
		{"alpha channel", testImage(), []any{"A", "", "", "", "Column", 0.0}, "55555515"},
		{"channels out of order, middle bit", lsbStripes8x2(), []any{"B", "R", "", "", "Column", 5.0}, "0505afaf"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := runLSB(t, c.img, c.args...); got != c.want {
				t.Errorf("got %s, want %s", got, c.want)
			}
		})
	}
}

// TestExtractLSBErrors covers the refusals: not an image, and a bit outside 0-7.
func TestExtractLSBErrors(t *testing.T) {
	if _, err := runOp(t, "Extract LSB", "not an image", "R", "", "", "", "Row", 0.0); err == nil ||
		!strings.Contains(err.Error(), "Please enter a valid image file.") {
		t.Errorf("non-image: got %v", err)
	}
	png := pngBytes(t, testImage())
	for _, bit := range []float64{8, -1} {
		if _, err := runOp(t, "Extract LSB", png, "R", "", "", "", "Row", bit); err == nil ||
			!strings.Contains(err.Error(), "Bit argument must be between 0 and 7") {
			t.Errorf("bit %v: got %v", bit, err)
		}
	}
}
