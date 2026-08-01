package ops

import "testing"

// Golden outputs were generated with the real Jimp library; cchef reproduces its
// pixels exactly, so the decoded pixels must match byte-for-byte.
func TestResizeImageGolden(t *testing.T) {
	input := loadPNGBytes(t, "resize_input.png")
	cases := []struct {
		alg    string
		w, h   int
		golden string
	}{
		{"Nearest Neighbour", 17, 13, "resize_nn_17x13.png"},
		{"Bilinear", 4, 3, "resize_bl_4x3.png"},
		{"Bicubic", 4, 3, "resize_bc_4x3.png"},
		{"Hermite", 17, 13, "resize_hm_17x13.png"},
		{"Bezier", 17, 13, "resize_bz_17x13.png"},
	}
	for _, c := range cases {
		t.Run(c.alg, func(t *testing.T) {
			out, err := runOp(t, "Resize Image", input, float64(c.w), float64(c.h), "Pixels", false, c.alg)
			if err != nil {
				t.Fatal(err)
			}
			assertSamePixels(t, c.alg, decodePNGOut(t, out), decodePNGOut(t, loadPNGBytes(t, c.golden)))
		})
	}
}

func TestResizeImagePercent(t *testing.T) {
	input := loadPNGBytes(t, "resize_input.png") // 10x8
	got := runImageOpBytes(t, input, "Resize Image", float64(50), float64(50), "Percent", false, "Bilinear")
	if got.Rect.Dx() != 5 || got.Rect.Dy() != 4 {
		t.Errorf("50%% of 10x8 = %dx%d, want 5x4", got.Rect.Dx(), got.Rect.Dy())
	}
}

func TestResizeImageAspect(t *testing.T) {
	input := loadPNGBytes(t, "resize_input.png") // 10x8
	// scaleToFit(17,13): f = max fit = 13/8 = 1.625 -> round(10*1.625)=16, round(8*1.625)=13.
	got := runImageOpBytes(t, input, "Resize Image", float64(17), float64(13), "Pixels", true, "Bilinear")
	if got.Rect.Dx() != 16 || got.Rect.Dy() != 13 {
		t.Errorf("aspect-fit 10x8 into 17x13 = %dx%d, want 16x13", got.Rect.Dx(), got.Rect.Dy())
	}
}

// A tiny percentage rounds the target size to 0, which Jimp clamps to 1.
func TestResizeImageClampToOne(t *testing.T) {
	input := loadPNGBytes(t, "resize_input.png") // 10x8
	got := runImageOpBytes(t, input, "Resize Image", float64(1), float64(1), "Percent", false, "Bilinear")
	if got.Rect.Dx() != 1 || got.Rect.Dy() != 1 {
		t.Errorf("1%% resize = %dx%d, want 1x1", got.Rect.Dx(), got.Rect.Dy())
	}
}

// scaleToFit where the width is the limiting dimension (f = w/W branch).
func TestResizeImageAspectWidthBound(t *testing.T) {
	input := loadPNGBytes(t, "resize_input.png") // 10x8
	// fit into 5x20: 5/20 <= 10/8 -> f = 5/10 = 0.5 -> 5x4.
	got := runImageOpBytes(t, input, "Resize Image", float64(5), float64(20), "Pixels", true, "Bilinear")
	if got.Rect.Dx() != 5 || got.Rect.Dy() != 4 {
		t.Errorf("aspect-fit 10x8 into 5x20 = %dx%d, want 5x4", got.Rect.Dx(), got.Rect.Dy())
	}
}

func TestResizeImageInvalid(t *testing.T) {
	if _, err := runOp(t, "Resize Image", "not an image", float64(10), float64(10), "Pixels", false, "Bilinear"); err == nil {
		t.Error("expected error for non-image input")
	}
}
