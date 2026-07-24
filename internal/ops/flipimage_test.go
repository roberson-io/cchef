package ops

import "testing"

func TestFlipImageHorizontal(t *testing.T) {
	src := testImage()
	got := runImageOp(t, "Flip Image", src, "Horizontal")
	w, h := src.Rect.Dx(), src.Rect.Dy()
	for y := range h {
		for x := range w {
			si := src.PixOffset(w-1-x, y)
			gi := got.PixOffset(x, y)
			if string(got.Pix[gi:gi+4]) != string(src.Pix[si:si+4]) {
				t.Fatalf("horizontal flip mismatch at (%d,%d)", x, y)
			}
		}
	}
}

func TestFlipImageVertical(t *testing.T) {
	src := testImage()
	got := runImageOp(t, "Flip Image", src, "Vertical")
	w, h := src.Rect.Dx(), src.Rect.Dy()
	for y := range h {
		for x := range w {
			si := src.PixOffset(x, h-1-y)
			gi := got.PixOffset(x, y)
			if string(got.Pix[gi:gi+4]) != string(src.Pix[si:si+4]) {
				t.Fatalf("vertical flip mismatch at (%d,%d)", x, y)
			}
		}
	}
}

func TestFlipImageInvalid(t *testing.T) {
	if _, err := runOp(t, "Flip Image", "not an image", "Horizontal"); err == nil {
		t.Error("expected error for non-image input")
	}
}
