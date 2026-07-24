package ops

import "testing"

func TestSharpenImage(t *testing.T) {
	assertImageGolden(t, "Sharpen Image", "sharpen.png", float64(2), float64(1), float64(10))
}

func TestSharpenImageInvalid(t *testing.T) {
	if _, err := runOp(t, "Sharpen Image", "not an image", float64(2), float64(1), float64(10)); err == nil {
		t.Error("expected error for non-image input")
	}
}
