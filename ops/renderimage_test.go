package ops

import (
	"encoding/base64"
	"testing"
)

// Render Image only validates magic bytes then passes the bytes through, so a
// short valid magic prefix exercises every code path.
const pngMagicHex = "89504e470d0a1a0a0102030405"

func rawPNG(t *testing.T) []byte {
	t.Helper()
	return mustHex(t, pngMagicHex)
}

func TestRenderImageRawPassthrough(t *testing.T) {
	want := string(rawPNG(t))
	// Input format Hex.
	if out, err := runOp(t, "Render Image", pngMagicHex, "Hex"); err != nil || out != want {
		t.Errorf("Hex/Raw = %q, %v; want %q", out, err, want)
	}
	// Input format Base64.
	b64 := base64.StdEncoding.EncodeToString(rawPNG(t))
	if out, err := runOp(t, "Render Image", b64, "Base64"); err != nil || out != want {
		t.Errorf("Base64/Raw = %q, %v; want %q", out, err, want)
	}
	// Input format Raw: the input string bytes are the image itself.
	if out, err := runOp(t, "Render Image", want, "Raw"); err != nil || out != want {
		t.Errorf("Raw/Raw = %q, %v; want %q", out, err, want)
	}
}

func TestRenderImageEmpty(t *testing.T) {
	if out, err := runOp(t, "Render Image", "", "Raw"); err != nil || out != "" {
		t.Errorf("empty input = %q, %v; want empty", out, err)
	}
}

func TestRenderImageInvalid(t *testing.T) {
	if _, err := runOp(t, "Render Image", "hello world, not an image", "Raw"); err == nil {
		t.Error("expected error for non-image input")
	}
}
