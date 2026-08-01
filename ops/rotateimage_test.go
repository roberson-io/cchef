package ops

import (
	"bytes"
	"math"
	"testing"
)

// rotate runs Rotate Image on PNG bytes and returns the output bytes.
func rotate(t *testing.T, pngIn string, deg float64) string {
	t.Helper()
	out, err := runOp(t, "Rotate Image", pngIn, deg)
	if err != nil {
		t.Fatalf("rotate %v: %v", deg, err)
	}
	return out
}

// Rotating by 90 degrees four times returns the original image exactly.
func TestRotateImage90Identity(t *testing.T) {
	src := testImage()
	b := pngBytes(t, src)
	for range 4 {
		b = rotate(t, b, 90)
	}
	if !bytes.Equal(decodePNGOut(t, b).Pix, src.Pix) {
		t.Error("four 90-degree rotations should restore the original")
	}
}

func TestRotateImage90Dimensions(t *testing.T) {
	src := testImage() // 6x5
	got := decodePNGOut(t, rotate(t, pngBytes(t, src), 90))
	if got.Rect.Dx() != 5 || got.Rect.Dy() != 6 {
		t.Errorf("90-degree rotation dims = %dx%d, want 5x6", got.Rect.Dx(), got.Rect.Dy())
	}
	// Jimp maps src(x,y) -> dst(y, w-1-x); invert: dst(a,b) = src(w-1-b, a).
	w := src.Rect.Dx()
	for a := 0; a < got.Rect.Dx(); a++ {
		for b := 0; b < got.Rect.Dy(); b++ {
			si := src.PixOffset(w-1-b, a)
			gi := got.PixOffset(a, b)
			if string(got.Pix[gi:gi+4]) != string(src.Pix[si:si+4]) {
				t.Fatalf("90-degree pixel (%d,%d) mismatch", a, b)
			}
		}
	}
}

func TestRotateImage180(t *testing.T) {
	src := testImage()
	got := decodePNGOut(t, rotate(t, pngBytes(t, src), 180))
	if got.Bounds() != src.Bounds() {
		t.Fatalf("180 dims changed: %v", got.Bounds())
	}
	w, h := src.Rect.Dx(), src.Rect.Dy()
	for y := range h {
		for x := range w {
			si := src.PixOffset(w-1-x, h-1-y)
			gi := got.PixOffset(x, y)
			if string(got.Pix[gi:gi+4]) != string(src.Pix[si:si+4]) {
				t.Fatalf("180 pixel (%d,%d) mismatch", x, y)
			}
		}
	}
}

func TestRotateImage270IsInverseOf90(t *testing.T) {
	src := testImage()
	b := rotate(t, pngBytes(t, src), 90)
	b = rotate(t, b, 270)
	if !bytes.Equal(decodePNGOut(t, b).Pix, src.Pix) {
		t.Error("rotate 90 then 270 should restore the original")
	}
}

func TestRotateImageNoop(t *testing.T) {
	src := testImage()
	for _, deg := range []float64{0, 360} {
		got := decodePNGOut(t, rotate(t, pngBytes(t, src), deg))
		if !bytes.Equal(got.Pix, src.Pix) {
			t.Errorf("rotate %v should be a no-op", deg)
		}
	}
}

// Arbitrary (non-90) rotations are approximate, but the output canvas size
// matches Jimp's expansion formula.
func TestRotateImageArbitraryDimensions(t *testing.T) {
	src := testImage() // 6x5
	got := decodePNGOut(t, rotate(t, pngBytes(t, src), 45))
	rad := 45 * math.Pi / 180
	wf, hf := float64(6), float64(5)
	w := int(math.Ceil(math.Abs(wf*math.Cos(rad))+math.Abs(hf*math.Sin(rad)))) + 1
	h := int(math.Ceil(math.Abs(wf*math.Sin(rad))+math.Abs(hf*math.Cos(rad)))) + 1
	if w%2 != 0 {
		w++
	}
	if h%2 != 0 {
		h++
	}
	if got.Rect.Dx() != w || got.Rect.Dy() != h {
		t.Errorf("45-degree dims = %dx%d, want %dx%d", got.Rect.Dx(), got.Rect.Dy(), w, h)
	}
}

func TestRotateImageInvalid(t *testing.T) {
	if _, err := runOp(t, "Rotate Image", "not an image", float64(90)); err == nil {
		t.Error("expected error for non-image input")
	}
}
