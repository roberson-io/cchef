package ops

import (
	"image"
	"testing"
)

func TestImageBrightnessContrast(t *testing.T) {
	assertImageGolden(t, "Image Brightness / Contrast", "brightness_contrast.png", float64(40), float64(30))
}

// High contrast clamps channels at both ends.
func TestJimpContrastClamps(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	copy(img.Pix, []byte{0, 0, 0, 255, 255, 255, 255, 255})
	jimpContrast(img, 0.9) // factor 19
	if img.Pix[0] != 0 {
		t.Errorf("dark pixel = %d, want 0 (clamped)", img.Pix[0])
	}
	if img.Pix[4] != 255 {
		t.Errorf("bright pixel = %d, want 255 (clamped)", img.Pix[4])
	}
}

func TestImageBrightnessContrastInvalid(t *testing.T) {
	if _, err := runOp(t, "Image Brightness / Contrast", "not an image", float64(0), float64(0)); err == nil {
		t.Error("expected error for non-image input")
	}
}
