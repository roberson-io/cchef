package ops

import (
	"image"
	"image/color"
	"testing"
)

func solidNRGBA(w, h int, c color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = c.R, c.G, c.B, c.A
	}
	return img
}

// jimpBlit must clip source pixels that fall outside the destination, both past
// the far edge and before the origin.
func TestJimpBlitClipping(t *testing.T) {
	dst := solidNRGBA(4, 4, color.NRGBA{0, 0, 0, 255})
	src := solidNRGBA(6, 6, color.NRGBA{255, 255, 255, 255})

	// Blit larger source at origin: pixels with xOffset/yOffset >= 4 are dropped.
	jimpBlit(dst, src, 0, 0)
	if dst.Rect.Dx() != 4 || dst.Rect.Dy() != 4 {
		t.Fatalf("blit changed dst dims to %v", dst.Rect)
	}
	// Every destination pixel was covered (opaque white over black -> white).
	for i := 0; i < len(dst.Pix); i += 4 {
		if dst.Pix[i] != 255 {
			t.Fatalf("pixel %d not composited", i/4)
		}
	}

	// Blit at a negative offset: pixels with xOffset/yOffset < 0 are dropped.
	dst2 := solidNRGBA(4, 4, color.NRGBA{0, 0, 0, 255})
	jimpBlit(dst2, solidNRGBA(2, 2, color.NRGBA{255, 255, 255, 255}), -1, -1)
	// Only dst2(0,0) receives src(1,1); the rest stay black.
	if dst2.Pix[0] != 255 {
		t.Error("expected top-left pixel composited from negative-offset blit")
	}
	if dst2.Pix[4] != 0 {
		t.Error("expected (1,0) to remain black")
	}
}
