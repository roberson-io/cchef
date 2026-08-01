package ops

import "testing"

func TestBlurImageFast(t *testing.T) {
	assertImageGolden(t, "Blur Image", "blur.png", float64(3), "Fast")
}

func TestBlurImageGaussian(t *testing.T) {
	assertImageGolden(t, "Blur Image", "gaussian.png", float64(2), "Gaussian")
}

func TestBlurImageInvalid(t *testing.T) {
	if _, err := runOp(t, "Blur Image", "not an image", float64(5), "Fast"); err == nil {
		t.Error("expected error for non-image input")
	}
	// A fast-blur amount beyond the lookup tables is rejected.
	if _, err := runOp(t, "Blur Image", loadPNGBytes(t, "resize_input.png"), float64(300), "Fast"); err == nil {
		t.Error("expected error for out-of-range blur amount")
	}
}
