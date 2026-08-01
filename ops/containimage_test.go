package ops

import "testing"

func TestContainImageGolden(t *testing.T) {
	input := loadPNGBytes(t, "resize_input.png")
	// Contain 10x8 into 17x13, Center/Middle, Bilinear.
	t.Run("opaque background", func(t *testing.T) {
		got := runImageOpBytes(t, input, "Contain Image", float64(17), float64(13), "Center", "Middle", "Bilinear", true)
		assertSamePixels(t, "contain-opaque", got, decodePNGOut(t, loadPNGBytes(t, "contain_optrue.png")))
	})
	t.Run("transparent background", func(t *testing.T) {
		got := runImageOpBytes(t, input, "Contain Image", float64(17), float64(13), "Center", "Middle", "Bilinear", false)
		assertSamePixels(t, "contain-transparent", got, decodePNGOut(t, loadPNGBytes(t, "contain_opfalse.png")))
	})
}

// Contain into a tall box exercises the width-bound fit branch.
func TestContainImageWidthBound(t *testing.T) {
	input := loadPNGBytes(t, "resize_input.png") // 10x8
	got := runImageOpBytes(t, input, "Contain Image", float64(5), float64(20), "Center", "Middle", "Bilinear", false)
	if got.Rect.Dx() != 5 || got.Rect.Dy() != 20 {
		t.Errorf("contain dims = %dx%d, want 5x20", got.Rect.Dx(), got.Rect.Dy())
	}
}

func TestContainImageInvalid(t *testing.T) {
	if _, err := runOp(t, "Contain Image", "not an image", float64(17), float64(13), "Center", "Middle", "Bilinear", true); err == nil {
		t.Error("expected error for non-image input")
	}
}
