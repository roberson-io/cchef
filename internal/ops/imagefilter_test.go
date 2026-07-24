package ops

import "testing"

func TestImageFilterGreyscale(t *testing.T) {
	src := testImage()
	got := runImageOp(t, "Image Filter", src, "Greyscale")
	for i := 0; i < len(src.Pix); i += 4 {
		grey := byte(0.2126*float64(src.Pix[i]) + 0.7152*float64(src.Pix[i+1]) + 0.0722*float64(src.Pix[i+2]))
		if got.Pix[i] != grey || got.Pix[i+1] != grey || got.Pix[i+2] != grey {
			t.Fatalf("pixel %d greyscale = (%d,%d,%d), want %d", i/4, got.Pix[i], got.Pix[i+1], got.Pix[i+2], grey)
		}
		if got.Pix[i+3] != src.Pix[i+3] {
			t.Fatalf("pixel %d alpha changed", i/4)
		}
	}
}

func TestImageFilterSepia(t *testing.T) {
	src := testImage()
	got := runImageOp(t, "Image Filter", src, "Sepia")
	clamp := func(v float64) byte {
		if v < 255 {
			return byte(v)
		}
		return 255
	}
	for i := 0; i < len(src.Pix); i += 4 {
		r, g, b := float64(src.Pix[i]), float64(src.Pix[i+1]), float64(src.Pix[i+2])
		red := r*0.393 + g*0.769 + b*0.189
		green := red*0.349 + g*0.686 + b*0.168 // Jimp reuses the new red
		blue := red*0.272 + green*0.534 + b*0.131
		if got.Pix[i] != clamp(red) || got.Pix[i+1] != clamp(green) || got.Pix[i+2] != clamp(blue) {
			t.Fatalf("pixel %d sepia = (%d,%d,%d), want (%d,%d,%d)", i/4,
				got.Pix[i], got.Pix[i+1], got.Pix[i+2], clamp(red), clamp(green), clamp(blue))
		}
	}
}

func TestImageFilterInvalid(t *testing.T) {
	if _, err := runOp(t, "Image Filter", "not an image", "Greyscale"); err == nil {
		t.Error("expected error for non-image input")
	}
}
