package ops

import "testing"

func TestCoverImageGolden(t *testing.T) {
	input := loadPNGBytes(t, "resize_input.png")
	// Cover 10x8 into 6x6, Center/Middle, Bilinear.
	got := runImageOpBytes(t, input, "Cover Image", float64(6), float64(6), "Center", "Middle", "Bilinear")
	assertSamePixels(t, "cover", got, decodePNGOut(t, loadPNGBytes(t, "cover_66.png")))
}

// Cover into a wide box exercises the width-bound fill branch.
func TestCoverImageWidthBound(t *testing.T) {
	input := loadPNGBytes(t, "resize_input.png") // 10x8
	got := runImageOpBytes(t, input, "Cover Image", float64(20), float64(6), "Center", "Middle", "Bilinear")
	if got.Rect.Dx() != 20 || got.Rect.Dy() != 6 {
		t.Errorf("cover dims = %dx%d, want 20x6", got.Rect.Dx(), got.Rect.Dy())
	}
}

func TestCoverImageInvalid(t *testing.T) {
	if _, err := runOp(t, "Cover Image", "not an image", float64(6), float64(6), "Center", "Middle", "Bilinear"); err == nil {
		t.Error("expected error for non-image input")
	}
}
