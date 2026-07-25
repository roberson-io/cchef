package ops

import (
	"image"
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// runFileListOp runs an operation whose output is a file list.
func runFileListOp(t *testing.T, name, input string, args ...any) ([]core.NamedFile, error) {
	t.Helper()
	op, ok := core.Default.Get(name)
	if !ok {
		t.Fatalf("op %q not registered", name)
	}
	coerced, err := core.CoerceArgs(op.Args(), args)
	if err != nil {
		t.Fatalf("coerce args: %v", err)
	}
	out, err := op.Run(core.NewDish([]byte(input), op.Meta().InputType), coerced)
	if err != nil {
		return nil, err
	}
	if out.Type() != core.TypeFileList {
		t.Fatalf("dish type = %q, want %q", out.Type(), core.TypeFileList)
	}
	return out.Files(), nil
}

// Each output keeps its own channel and zeroes the other two; alpha is
// untouched, matching Jimp's colour({apply: "…", params: [-255]}).
func TestSplitColourChannels(t *testing.T) {
	input := loadPNGBytes(t, "resize_input.png")
	src := decodePNGOut(t, input)

	files, err := runFileListOp(t, "Split Colour Channels", input)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"red.png", "green.png", "blue.png"}
	if len(files) != 3 {
		t.Fatalf("got %d files, want 3", len(files))
	}
	for i, f := range files {
		if f.Name != want[i] {
			t.Fatalf("file %d name = %q, want %q", i, f.Name, want[i])
		}
		got := decodePNGOut(t, string(f.Data))
		if got.Bounds() != src.Bounds() {
			t.Fatalf("%s: bounds %v != source %v", f.Name, got.Bounds(), src.Bounds())
		}
		for p := 0; p < len(src.Pix); p += 4 {
			for ch := range 3 {
				want := byte(0)
				if ch == i {
					want = src.Pix[p+ch]
				}
				if got.Pix[p+ch] != want {
					t.Fatalf("%s: pixel %d channel %d = %d, want %d",
						f.Name, p/4, ch, got.Pix[p+ch], want)
				}
			}
			if got.Pix[p+3] != src.Pix[p+3] {
				t.Fatalf("%s: pixel %d alpha = %d, want %d (unchanged)",
					f.Name, p/4, got.Pix[p+3], src.Pix[p+3])
			}
		}
	}
}

// Each channel matches a golden produced by the real Jimp. This also pins that
// the channels are split from the original rather than from an already-blanked
// intermediate.
func TestSplitColourChannelsGolden(t *testing.T) {
	files, err := runFileListOp(t, "Split Colour Channels", loadPNGBytes(t, "resize_input.png"))
	if err != nil {
		t.Fatal(err)
	}
	for i, golden := range []string{"split_red.png", "split_green.png", "split_blue.png"} {
		assertSamePixels(t, golden, decodePNGOut(t, string(files[i].Data)),
			decodePNGOut(t, loadPNGBytes(t, golden)))
	}
}

func TestSplitColourChannelsInvalid(t *testing.T) {
	_, err := runFileListOp(t, "Split Colour Channels", "not an image")
	if err == nil || err.Error() != "Invalid file type." {
		t.Errorf("error = %v, want Invalid file type.", err)
	}
}

// A zero-size image cannot be PNG-encoded. Decoding always yields at least one
// pixel, so this guard is only reachable by calling the encoder directly.
func TestEncodeChannelError(t *testing.T) {
	if _, err := encodeChannel(image.NewNRGBA(image.Rect(0, 0, 0, 0)), 0); err == nil {
		t.Error("expected an encode error for a 0x0 image")
	}
}

// The same failure surfaced through Run names the channel that failed.
func TestSplitColourChannelsEncodeError(t *testing.T) {
	_, err := splitChannelFiles(image.NewNRGBA(image.Rect(0, 0, 0, 0)))
	if err == nil || !strings.Contains(err.Error(), "Could not split red channel") {
		t.Errorf("error = %v, want one naming the red channel", err)
	}
}
