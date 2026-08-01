package ops

import "testing"

func TestImageOpacity(t *testing.T) {
	src := testImage()
	got := runImageOp(t, "Image Opacity", src, float64(50))
	for i := 0; i < len(src.Pix); i += 4 {
		// RGB unchanged; alpha multiplied by 0.5 and truncated (Jimp's Uint8 store).
		wantA := byte(float64(src.Pix[i+3]) * 0.5)
		if got.Pix[i] != src.Pix[i] || got.Pix[i+1] != src.Pix[i+1] || got.Pix[i+2] != src.Pix[i+2] {
			t.Fatalf("pixel %d RGB changed", i/4)
		}
		if got.Pix[i+3] != wantA {
			t.Fatalf("pixel %d alpha = %d, want %d", i/4, got.Pix[i+3], wantA)
		}
	}
}

func TestImageOpacityFull(t *testing.T) {
	src := testImage()
	got := runImageOp(t, "Image Opacity", src, float64(100))
	for i := range src.Pix {
		if got.Pix[i] != src.Pix[i] {
			t.Fatalf("opacity 100 should be a no-op at byte %d", i)
		}
	}
}

func TestImageOpacityInvalid(t *testing.T) {
	if _, err := runOp(t, "Image Opacity", "not an image", float64(100)); err == nil {
		t.Error("expected error for non-image input")
	}
}
