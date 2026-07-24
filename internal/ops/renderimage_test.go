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
	if out, err := runOp(t, "Render Image", pngMagicHex, "Hex", "Raw"); err != nil || out != want {
		t.Errorf("Hex/Raw = %q, %v; want %q", out, err, want)
	}
	// Input format Base64.
	b64 := base64.StdEncoding.EncodeToString(rawPNG(t))
	if out, err := runOp(t, "Render Image", b64, "Base64", "Raw"); err != nil || out != want {
		t.Errorf("Base64/Raw = %q, %v; want %q", out, err, want)
	}
	// Input format Raw: the input string bytes are the image itself.
	if out, err := runOp(t, "Render Image", want, "Raw", "Raw"); err != nil || out != want {
		t.Errorf("Raw/Raw = %q, %v; want %q", out, err, want)
	}
}

func TestRenderImageEmpty(t *testing.T) {
	if out, err := runOp(t, "Render Image", "", "Raw", "Raw"); err != nil || out != "" {
		t.Errorf("empty input = %q, %v; want empty", out, err)
	}
}

func TestRenderImageBase64Output(t *testing.T) {
	out, err := runOp(t, "Render Image", pngMagicHex, "Hex", "Base64")
	if err != nil {
		t.Fatal(err)
	}
	want := "data:image/png;base64," + base64.StdEncoding.EncodeToString(rawPNG(t))
	if out != want {
		t.Errorf("Base64 output = %q, want %q", out, want)
	}
}

func TestRenderImageTerminalOutput(t *testing.T) {
	png := rawPNG(t)

	t.Run("iterm", func(t *testing.T) {
		t.Setenv("TERM_PROGRAM", "iTerm.app")
		t.Setenv("TERM", "xterm-256color")
		t.Setenv("KITTY_WINDOW_ID", "")
		out, err := runOp(t, "Render Image", pngMagicHex, "Hex", "Terminal")
		if err != nil {
			t.Fatal(err)
		}
		want, _ := encodeTerminalImage(termITerm, "image/png", png)
		if out != string(want) {
			t.Errorf("iterm terminal output mismatch")
		}
	})

	t.Run("kitty", func(t *testing.T) {
		t.Setenv("TERM_PROGRAM", "")
		t.Setenv("TERM", "xterm-256color")
		t.Setenv("KITTY_WINDOW_ID", "3")
		out, err := runOp(t, "Render Image", pngMagicHex, "Hex", "Terminal")
		if err != nil {
			t.Fatal(err)
		}
		want, _ := encodeTerminalImage(termKitty, "image/png", png)
		if out != string(want) {
			t.Errorf("kitty terminal output mismatch")
		}
	})

	t.Run("unsupported terminal", func(t *testing.T) {
		t.Setenv("TERM_PROGRAM", "")
		t.Setenv("TERM", "dumb")
		t.Setenv("KITTY_WINDOW_ID", "")
		if _, err := runOp(t, "Render Image", pngMagicHex, "Hex", "Terminal"); err == nil {
			t.Error("expected error on unsupported terminal")
		}
	})
}

func TestRenderImageInvalid(t *testing.T) {
	if _, err := runOp(t, "Render Image", "hello world, not an image", "Raw", "Raw"); err == nil {
		t.Error("expected error for non-image input")
	}
}
