package ops

import (
	"bytes"
	"image"
	"image/png"
	"testing"
)

func convertArgs(format string) []any {
	return []any{format, float64(80), "Auto", float64(9)}
}

// PNG and TIFF preserve the input pixels exactly (lossless, alpha kept).
func TestConvertImageFormatLossless(t *testing.T) {
	input := loadPNGBytes(t, "resize_input.png")
	src := decodePNGOut(t, input)
	for _, format := range []string{"PNG", "TIFF"} {
		t.Run(format, func(t *testing.T) {
			out, err := runOp(t, "Convert Image Format", input, convertArgs(format)...)
			if err != nil {
				t.Fatal(err)
			}
			assertSamePixels(t, format, decodePNGOut(t, out), src)
		})
	}
}

// BMP is 24-bit: RGB is preserved but alpha becomes opaque (matches Jimp).
func TestConvertImageFormatBMP(t *testing.T) {
	input := loadPNGBytes(t, "resize_input.png")
	src := decodePNGOut(t, input)
	out, err := runOp(t, "Convert Image Format", input, convertArgs("BMP")...)
	if err != nil {
		t.Fatal(err)
	}
	got := decodePNGOut(t, out)
	for i := 0; i < len(src.Pix); i += 4 {
		if got.Pix[i] != src.Pix[i] || got.Pix[i+1] != src.Pix[i+1] || got.Pix[i+2] != src.Pix[i+2] {
			t.Fatalf("BMP RGB changed at pixel %d", i/4)
		}
		if got.Pix[i+3] != 255 {
			t.Fatalf("BMP pixel %d alpha = %d, want 255 (24-bit)", i/4, got.Pix[i+3])
		}
	}
}

// JPEG output is lossy, but must be a valid decodable JPEG of the right size.
func TestConvertImageFormatJPEG(t *testing.T) {
	input := loadPNGBytes(t, "resize_input.png")
	out, err := runOp(t, "Convert Image Format", input, convertArgs("JPEG")...)
	if err != nil {
		t.Fatal(err)
	}
	img, format, err := image.Decode(bytes.NewReader([]byte(out)))
	if err != nil {
		t.Fatalf("decode jpeg output: %v", err)
	}
	if format != "jpeg" {
		t.Errorf("output format = %q, want jpeg", format)
	}
	if img.Bounds().Dx() != 10 || img.Bounds().Dy() != 8 {
		t.Errorf("jpeg dims = %v, want 10x8", img.Bounds())
	}
}

func TestConvertImageFormatInvalid(t *testing.T) {
	if _, err := runOp(t, "Convert Image Format", "not an image", convertArgs("PNG")...); err == nil {
		t.Error("expected error for non-image input")
	}
}

func TestPngDeflateLevel(t *testing.T) {
	if pngDeflateLevel(0) != png.NoCompression ||
		pngDeflateLevel(2) != png.BestSpeed ||
		pngDeflateLevel(5) != png.DefaultCompression ||
		pngDeflateLevel(9) != png.BestCompression {
		t.Error("pngDeflateLevel mapping")
	}
}

func TestEncodeConvertedError(t *testing.T) {
	// A zero-size image cannot be PNG-encoded.
	if _, err := encodeConverted(image.NewNRGBA(image.Rect(0, 0, 0, 0)), "PNG", 80, 9); err == nil {
		t.Error("expected encode error for 0x0 image")
	}
}
