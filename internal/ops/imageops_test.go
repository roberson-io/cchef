package ops

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

// testImage builds a small NRGBA image with varied colours and alpha so the
// pixel operations are exercised across the channel range.
func testImage() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, 6, 5))
	for y := range 5 {
		for x := range 6 {
			img.SetNRGBA(x, y, color.NRGBA{
				R: uint8((x*40 + 5) % 256),
				G: uint8((y*50 + 10) % 256),
				B: uint8((x*y*9 + 3) % 256),
				A: uint8((x + y) * 25 % 256),
			})
		}
	}
	return img
}

// pngBytes encodes an image to PNG.
func pngBytes(t *testing.T, img image.Image) string {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.String()
}

// decodePNGOut decodes an operation's output bytes into a dense NRGBA.
func decodePNGOut(t *testing.T, out string) *image.NRGBA {
	t.Helper()
	img, _, err := image.Decode(bytes.NewReader([]byte(out)))
	if err != nil {
		t.Fatalf("decode output: %v", err)
	}
	return toNRGBA(img)
}

// runImageOp runs an image op on a PNG-encoded image and decodes the result.
func runImageOp(t *testing.T, op string, img image.Image, args ...any) *image.NRGBA {
	t.Helper()
	out, err := runOp(t, op, pngBytes(t, img), args...)
	if err != nil {
		t.Fatalf("%s: %v", op, err)
	}
	return decodePNGOut(t, out)
}

func TestToNRGBARoundTrip(t *testing.T) {
	src := testImage()
	got := decodePNGOut(t, pngBytes(t, src))
	if got.Bounds() != src.Bounds() {
		t.Fatalf("bounds %v != %v", got.Bounds(), src.Bounds())
	}
	if !bytes.Equal(got.Pix, src.Pix) {
		t.Error("PNG round-trip through toNRGBA changed pixels")
	}
}

func TestDecodeImageNRGBAInvalid(t *testing.T) {
	// Not an image at all.
	if _, _, err := decodeImageNRGBA([]byte("hello world"), "Invalid file type."); err == nil {
		t.Error("expected error for non-image input")
	}
	// Valid PNG magic but corrupt body -> decode error.
	corrupt := append([]byte("\x89PNG\r\n\x1a\n"), []byte("garbage data here")...)
	if _, _, err := decodeImageNRGBA(corrupt, "Invalid file type."); err == nil {
		t.Error("expected decode error for corrupt PNG")
	}
}

func TestEncodeImageError(t *testing.T) {
	// A zero-size image cannot be PNG-encoded, exercising the error path.
	if _, err := encodeImageNRGBA(image.NewNRGBA(image.Rect(0, 0, 0, 0)), "png"); err == nil {
		t.Error("expected encode error for a 0x0 image")
	}
}

func TestEncodeImageFormats(t *testing.T) {
	img := testImage()
	for _, format := range []string{"png", "gif", "jpeg", "bmp", "tiff", "other"} {
		out, err := encodeImageNRGBA(img, format)
		if err != nil {
			t.Errorf("encode %s: %v", format, err)
			continue
		}
		if _, _, err := image.Decode(bytes.NewReader(out)); err != nil {
			t.Errorf("re-decode %s: %v", format, err)
		}
	}
}
