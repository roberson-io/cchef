package ops

import "testing"

func TestDitherImage(t *testing.T) {
	assertImageGolden(t, "Dither Image", "dither.png")
}

func TestDitherImageInvalid(t *testing.T) {
	if _, err := runOp(t, "Dither Image", "not an image"); err == nil {
		t.Error("expected error for non-image input")
	}
}
