package ops

import (
	"image"
	"testing"
)

func TestNormaliseImage(t *testing.T) {
	assertImageGolden(t, "Normalise Image", "normalize.png")
}

// A flat channel (min == max) normalises to 0.
func TestNormaliseImageFlatChannel(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = 100, 100, 100, 255
	}
	jimpNormalize(img)
	for i := 0; i < len(img.Pix); i += 4 {
		if img.Pix[i] != 0 || img.Pix[i+1] != 0 || img.Pix[i+2] != 0 {
			t.Fatalf("flat channel should normalise to 0, got %v", img.Pix[i:i+3])
		}
	}
}

func TestNormaliseImageInvalid(t *testing.T) {
	if _, err := runOp(t, "Normalise Image", "not an image"); err == nil {
		t.Error("expected error for non-image input")
	}
}
