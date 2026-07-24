package ops

import (
	"image"
	"testing"
)

// cropArgs builds Crop Image args (x, y, w, h, then autocrop defaults off).
func cropArgs(x, y, w, h float64) []any {
	return []any{x, y, w, h, false, float64(0.02), true, false, float64(0)}
}

func TestCropImage(t *testing.T) {
	input := loadPNGBytes(t, "resize_input.png") // 10x8
	src := decodePNGOut(t, input)
	got := runImageOpBytes(t, input, "Crop Image", cropArgs(2, 1, 3, 2)...)
	if got.Rect.Dx() != 3 || got.Rect.Dy() != 2 {
		t.Fatalf("crop dims = %dx%d, want 3x2", got.Rect.Dx(), got.Rect.Dy())
	}
	// Build the expected 3x2 subregion at (2,1).
	want := image.NewNRGBA(image.Rect(0, 0, 3, 2))
	for cy := range 2 {
		for cx := range 3 {
			si := src.PixOffset(2+cx, 1+cy)
			di := want.PixOffset(cx, cy)
			copy(want.Pix[di:di+4], src.Pix[si:si+4])
		}
	}
	assertSamePixels(t, "crop", got, want)
}

func TestCropImageAutocrop(t *testing.T) {
	input := loadPNGBytes(t, "autocrop_input.png")
	// autocrop args: x,y,w,h ignored; autocrop=true, tolerance 0.02, frames=true.
	args := []any{float64(0), float64(0), float64(1), float64(1), true, float64(0.02), true, false, float64(0)}
	got := runImageOpBytes(t, input, "Crop Image", args...)
	want := decodePNGOut(t, loadPNGBytes(t, "autocrop_golden.png"))
	assertSamePixels(t, "autocrop", got, want)
}

func TestCropImageAutocropOptions(t *testing.T) {
	bordered := loadPNGBytes(t, "autocrop_input.png") // 12x10, 2px border -> 8x6
	// x,y,w,h then autocrop=true, tolerance, cropOnlyFrames, cropSymmetric, leaveBorder
	mk := func(frames, sym bool, border float64) []any {
		return []any{float64(0), float64(0), float64(1), float64(1), true, float64(0.02), frames, sym, border}
	}
	cases := []struct {
		name       string
		args       []any
		wantW      int
		wantH      int
		wantChange bool
	}{
		{"symmetric", mk(true, true, 0), 8, 6, true},
		{"non-frames", mk(false, false, 0), 8, 6, true},
		{"leave 1px border", mk(true, false, 1), 10, 8, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := runImageOpBytes(t, bordered, "Crop Image", c.args...)
			if got.Rect.Dx() != c.wantW || got.Rect.Dy() != c.wantH {
				t.Errorf("dims = %dx%d, want %dx%d", got.Rect.Dx(), got.Rect.Dy(), c.wantW, c.wantH)
			}
		})
	}

	// An image with no uniform border is returned unchanged.
	noBorder := loadPNGBytes(t, "resize_input.png") // 10x8
	got := runImageOpBytes(t, noBorder, "Crop Image", mk(true, false, 0)...)
	if got.Rect.Dx() != 10 || got.Rect.Dy() != 8 {
		t.Errorf("no-border autocrop changed dims to %dx%d", got.Rect.Dx(), got.Rect.Dy())
	}
}

func TestCropImageInvalid(t *testing.T) {
	if _, err := runOp(t, "Crop Image", "not an image", cropArgs(0, 0, 10, 10)...); err == nil {
		t.Error("expected error for non-image input")
	}
}

func TestCropImageOutOfBounds(t *testing.T) {
	input := loadPNGBytes(t, "resize_input.png") // 10x8
	if _, err := runOp(t, "Crop Image", input, cropArgs(5, 5, 20, 20)...); err == nil {
		t.Error("expected error cropping outside the image")
	}
}
