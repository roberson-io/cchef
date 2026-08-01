package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

// genInput builds the RGB test input (bytes 0..47).
func genInputRGB() string {
	b := make([]byte, 48)
	for i := range b {
		b[i] = byte(i)
	}
	return string(b)
}

func TestGenerateImageRGB(t *testing.T) {
	// mode RGB, scale 2, width 4 -> 8x8 golden.
	got := runImageOpBytes(t, genInputRGB(), "Generate Image", "RGB", float64(2), float64(4))
	assertSamePixels(t, "generate-rgb", got, decodePNGOut(t, loadPNGBytes(t, "generate_rgb.png")))
}

func TestGenerateImageBits(t *testing.T) {
	// mode Bits, scale 1, width 8 -> 8x4 golden.
	got := runImageOpBytes(t, string([]byte{0xA5, 0x0F, 0xF0, 0x3C}), "Generate Image", "Bits", float64(1), float64(8))
	assertSamePixels(t, "generate-bits", got, decodePNGOut(t, loadPNGBytes(t, "generate_bits.png")))
}

// The remaining modes, each against a golden produced by the real Jimp.
func TestGenerateImageModes(t *testing.T) {
	for _, tc := range []struct {
		mode   string
		width  float64
		golden string
	}{
		{"Greyscale", 8, "generate_greyscale.png"},
		{"RG", 8, "generate_rg.png"},
		{"RGBA", 4, "generate_rgba.png"},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			got := runImageOpBytes(t, genInputRGB(), "Generate Image", tc.mode, float64(1), tc.width)
			assertSamePixels(t, tc.mode, got, decodePNGOut(t, loadPNGBytes(t, tc.golden)))
		})
	}
}

// Empty input yields a zero-height image. With scaling on (the default) Jimp
// clamps that to a single transparent row, which we match. Unscaled, Jimp emits
// a height-0 PNG; Go's encoder rejects that size, so we surface its error.
func TestGenerateImageEmptyInput(t *testing.T) {
	got := runImageOpBytes(t, "", "Generate Image", "Greyscale", float64(8), float64(64))
	if got.Rect.Dx() != 512 || got.Rect.Dy() != 1 {
		t.Errorf("bounds = %v, want 512x1", got.Rect)
	}
	for i, v := range got.Pix {
		if v != 0 {
			t.Fatalf("pixel byte %d = %d, want 0 (transparent)", i, v)
		}
	}
	if _, err := runOp(t, "Generate Image", "", "Greyscale", float64(1), float64(64)); err == nil {
		t.Error("expected an encode error for a height-0 image")
	}
}

func TestGenerateImageErrors(t *testing.T) {
	// RGB requires a byte count divisible by 3.
	if _, err := runOp(t, "Generate Image", "abcd", "RGB", float64(1), float64(2)); err == nil {
		t.Error("expected error for non-divisible byte count")
	}
	// An unrecognised mode is rejected. The engine already constrains the
	// option, so this guard is only reachable by calling Run directly.
	_, err := GenerateImage{}.Run(
		core.NewDish([]byte("abcd"), core.TypeByteArray),
		[]any{"CMYK", float64(1), float64(2)},
	)
	if err == nil || err.Error() != "Unsupported Mode: (CMYK)" {
		t.Errorf("unsupported mode error = %v, want Unsupported Mode: (CMYK)", err)
	}
}

// TestGenerateImageScaleInteger pins the integer check CyberChef declares on
// the pixel scale factor.
func TestGenerateImageScaleInteger(t *testing.T) {
	op, _ := core.Default.Get("Generate Image")
	args := core.DefaultArgs(op.Args())
	args[1] = 1.5
	_, err := core.CoerceArgs(op.Args(), args)
	if err == nil {
		t.Fatal("a fractional pixel scale factor was accepted")
	}
	if want := "Pixel Scale Factor must be an integer."; err.Error() != want {
		t.Errorf("got %q, want %q", err.Error(), want)
	}
}
