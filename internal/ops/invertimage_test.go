package ops

import "testing"

func TestInvertImage(t *testing.T) {
	src := testImage()
	got := runImageOp(t, "Invert Image", src)
	if got.Bounds() != src.Bounds() {
		t.Fatalf("bounds changed: %v", got.Bounds())
	}
	for i := 0; i < len(src.Pix); i += 4 {
		want := [4]byte{255 - src.Pix[i], 255 - src.Pix[i+1], 255 - src.Pix[i+2], src.Pix[i+3]}
		gotPx := [4]byte{got.Pix[i], got.Pix[i+1], got.Pix[i+2], got.Pix[i+3]}
		if gotPx != want {
			t.Fatalf("pixel %d: got %v, want %v (alpha unchanged)", i/4, gotPx, want)
		}
	}
}

func TestInvertImageInvalid(t *testing.T) {
	if _, err := runOp(t, "Invert Image", "not an image"); err == nil {
		t.Error("expected error for non-image input")
	}
}
