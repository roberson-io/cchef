package ops

import (
	"crypto/md5" // #nosec G501 -- the operation's palette is defined by MD5, matching CyberChef
	"fmt"
	"strings"
	"testing"
)

// The palette scheme — each pixel becomes the first three bytes of
// md5(seed + "R.G.B"), fully opaque — was verified against CyberChef's Node
// API pixel for pixel over five image/seed pairs; the hard-coded pixels here
// are from those recordings.

// TestRandomizeColourPaletteKnownPixels checks recorded oracle pixels.
func TestRandomizeColourPaletteKnownPixels(t *testing.T) {
	got := runImageOp(t, "Randomize Colour Palette", lsbSolid3x3(), "abc")
	for i := 0; i < len(got.Pix); i += 4 {
		if fmt.Sprintf("%x", got.Pix[i:i+4]) != "5c5a2aff" {
			t.Fatalf("pixel %d: got %x, want 5c5a2aff", i/4, got.Pix[i:i+4])
		}
	}
	grad := runImageOp(t, "Randomize Colour Palette", testImage(), "abc")
	if fmt.Sprintf("%x", grad.Pix[:4]) != "54f12fff" {
		t.Errorf("first pixel: got %x, want 54f12fff", grad.Pix[:4])
	}
	seeded := runImageOp(t, "Randomize Colour Palette", testImage(), "s33d")
	if fmt.Sprintf("%x", seeded.Pix[:4]) != "a85253ff" {
		t.Errorf("seed s33d first pixel: got %x, want a85253ff", seeded.Pix[:4])
	}
}

// TestRandomizeColourPaletteWholeImage recomputes the whole palette
// independently of the operation and compares every pixel.
func TestRandomizeColourPaletteWholeImage(t *testing.T) {
	src := testImage()
	got := runImageOp(t, "Randomize Colour Palette", src, "check")
	for i := 0; i < len(src.Pix); i += 4 {
		digest := md5.Sum(fmt.Appendf(nil, "check%d.%d.%d", // #nosec G401 -- defined by the operation
			src.Pix[i], src.Pix[i+1], src.Pix[i+2]))
		want := fmt.Sprintf("%x", digest[:3]) + "ff"
		if fmt.Sprintf("%x", got.Pix[i:i+4]) != want {
			t.Fatalf("pixel %d: got %x, want %s", i/4, got.Pix[i:i+4], want)
		}
	}
}

// TestRandomizeColourPaletteEmptySeed leaves the seed blank: the palette is
// random, but the mapping must still be a palette — same source colour, same
// output — and fully opaque, at the source dimensions.
func TestRandomizeColourPaletteEmptySeed(t *testing.T) {
	src := lsbStripes8x2()
	got := runImageOp(t, "Randomize Colour Palette", src, "")
	if got.Rect != src.Rect {
		t.Fatalf("dimensions changed: %v", got.Rect)
	}
	byColour := map[string]string{}
	for i := 0; i < len(src.Pix); i += 4 {
		if got.Pix[i+3] != 255 {
			t.Fatalf("pixel %d not opaque", i/4)
		}
		key := fmt.Sprintf("%x", src.Pix[i:i+3])
		val := fmt.Sprintf("%x", got.Pix[i:i+4])
		if prev, there := byColour[key]; there && prev != val {
			t.Fatalf("colour %s mapped to both %s and %s", key, prev, val)
		}
		byColour[key] = val
	}
}

// TestRandomizeColourPaletteInvalid covers input that is not an image.
func TestRandomizeColourPaletteInvalid(t *testing.T) {
	if _, err := runOp(t, "Randomize Colour Palette", "not an image", "x"); err == nil ||
		!strings.Contains(err.Error(), "Please enter a valid image file.") {
		t.Errorf("got %v", err)
	}
}
