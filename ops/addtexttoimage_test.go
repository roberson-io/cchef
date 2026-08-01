package ops

import (
	"image"
	"testing"
)

// addTextArgs builds the operation's eleven arguments.
func addTextArgs(text, hAlign, vAlign string, x, y, size float64, face string, r, g, b, a float64) []any {
	return []any{text, hAlign, vAlign, x, y, size, face, r, g, b, a}
}

// Each case is compared against a golden rendered by the real Jimp using the
// same bitmap atlases, so the output must match pixel for pixel.
func TestAddTextToImage(t *testing.T) {
	for _, tc := range []struct {
		golden string
		args   []any
	}{
		{"addtext_basic.png", addTextArgs("Hi", "None", "None", 1, 1, 16, "Roboto", 255, 255, 255, 255)},
		{"addtext_centre.png", addTextArgs("Hi", "Center", "Middle", 0, 0, 24, "Roboto Black", 255, 0, 0, 255)},
		{"addtext_mono.png", addTextArgs("A b", "Right", "Bottom", 0, 0, 8, "Roboto Mono", 0, 128, 255, 200)},
		{"addtext_slab.png", addTextArgs("Wide text here", "Center", "Top", 0, 0, 40, "Roboto Slab", 255, 255, 255, 255)},
	} {
		t.Run(tc.golden, func(t *testing.T) {
			got := runImageOpBytes(t, loadPNGBytes(t, "addtext_input.png"), "Add Text To Image", tc.args...)
			assertSamePixels(t, tc.golden, got, decodePNGOut(t, loadPNGBytes(t, tc.golden)))
		})
	}
}

// Recolouring the atlas must not leak between runs: the cached font is shared,
// so a coloured run followed by a white one has to render white again.
func TestAddTextToImageColourIsolation(t *testing.T) {
	input := loadPNGBytes(t, "addtext_input.png")
	runImageOpBytes(t, input, "Add Text To Image",
		addTextArgs("Hi", "None", "None", 1, 1, 16, "Roboto", 255, 0, 0, 255)...)
	got := runImageOpBytes(t, input, "Add Text To Image",
		addTextArgs("Hi", "None", "None", 1, 1, 16, "Roboto", 255, 255, 255, 255)...)
	assertSamePixels(t, "addtext_basic.png", got, decodePNGOut(t, loadPNGBytes(t, "addtext_basic.png")))
}

func TestAddTextToImageInvalid(t *testing.T) {
	_, err := runOp(t, "Add Text To Image", "not an image",
		addTextArgs("Hi", "None", "None", 0, 0, 16, "Roboto", 255, 255, 255, 255)...)
	if err == nil || err.Error() != "Invalid file type." {
		t.Errorf("error = %v, want Invalid file type.", err)
	}
}

// Empty text leaves the image untouched: there are no glyphs to blit.
func TestAddTextToImageEmptyText(t *testing.T) {
	input := loadPNGBytes(t, "addtext_input.png")
	got := runImageOpBytes(t, input, "Add Text To Image",
		addTextArgs("", "None", "None", 0, 0, 16, "Roboto", 255, 255, 255, 255)...)
	assertSamePixels(t, "empty text", got, decodePNGOut(t, input))
}

// Left/Top alignment pins the text to the origin regardless of the X/Y options.
func TestAddTextToImageLeftTop(t *testing.T) {
	input := loadPNGBytes(t, "addtext_input.png")
	got := runImageOpBytes(t, input, "Add Text To Image",
		addTextArgs("Hi", "Left", "Top", 40, 40, 16, "Roboto", 255, 255, 255, 255)...)
	want := runImageOpBytes(t, input, "Add Text To Image",
		addTextArgs("Hi", "None", "None", 0, 0, 16, "Roboto", 255, 255, 255, 255)...)
	assertSamePixels(t, "left/top", got, want)
}

// The font face is constrained by the engine, so an unknown one is only
// reachable by calling the renderer directly.
func TestDrawTextOnImageUnknownFace(t *testing.T) {
	_, err := drawTextOnImage(image.NewNRGBA(image.Rect(0, 0, 4, 4)),
		addTextParams{text: "Hi", size: 16, face: "Comic Sans"})
	if err == nil {
		t.Error("expected an error for an unknown font face")
	}
}

// jsRound must round a negative half towards positive infinity, as JavaScript
// does; Go's math.Round would give -1 here.
func TestJSRound(t *testing.T) {
	for _, tc := range []struct {
		in   float64
		want int
	}{{-0.5, 0}, {-1.5, -1}, {0.5, 1}, {1.4, 1}, {2.5, 3}} {
		if got := jsRound(tc.in); got != tc.want {
			t.Errorf("jsRound(%v) = %d, want %d", tc.in, got, tc.want)
		}
	}
}
