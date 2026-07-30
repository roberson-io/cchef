package ops

import (
	"strings"
	"testing"
)

// The scheme — the chosen channel's chosen bit turns the pixel black when set
// and white when clear, always fully opaque — was verified against CyberChef's
// Node API pixel for pixel over four image/channel/bit combinations.

// TestViewBitPlane checks every channel and a spread of bits against the
// source image's own bits.
func TestViewBitPlane(t *testing.T) {
	channels := map[string]int{"Red": 0, "Green": 1, "Blue": 2, "Alpha": 3}
	for colour, c := range channels {
		for _, bit := range []float64{0, 3, 7} {
			src := testImage()
			got := runImageOp(t, "View Bit Plane", src, colour, bit)
			if got.Rect != src.Rect {
				t.Fatalf("%s/%v: dimensions changed: %v", colour, bit, got.Rect)
			}
			for i := 0; i < len(src.Pix); i += 4 {
				want := byte(255)
				if src.Pix[i+c]>>int(bit)&1 == 1 {
					want = 0
				}
				for j := range 3 {
					if got.Pix[i+j] != want {
						t.Fatalf("%s bit %v pixel %d channel %d: got %d, want %d",
							colour, bit, i/4, j, got.Pix[i+j], want)
					}
				}
				if got.Pix[i+3] != 255 {
					t.Fatalf("%s bit %v pixel %d: alpha %d, want 255", colour, bit, i/4, got.Pix[i+3])
				}
			}
		}
	}
}

// TestViewBitPlaneErrors covers the refusals: not an image, and a bit outside 0-7.
func TestViewBitPlaneErrors(t *testing.T) {
	if _, err := runOp(t, "View Bit Plane", "not an image", "Red", 0.0); err == nil ||
		!strings.Contains(err.Error(), "Please enter a valid image file.") {
		t.Errorf("non-image: got %v", err)
	}
	png := pngBytes(t, testImage())
	for _, bit := range []float64{8, -1} {
		if _, err := runOp(t, "View Bit Plane", png, "Red", bit); err == nil ||
			!strings.Contains(err.Error(), "Bit argument must be between 0 and 7") {
			t.Errorf("bit %v: got %v", bit, err)
		}
	}
}
