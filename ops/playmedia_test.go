package ops

import (
	"encoding/base64"
	"testing"
)

// A minimal RIFF/WAVE header is enough: Play Media only validates the magic
// bytes then passes the input through.
const wavMagicHex = "524946460000000057415645"

func wavMagic(t *testing.T) []byte {
	t.Helper()
	return mustHex(t, wavMagicHex)
}

func TestPlayMediaRawPassthrough(t *testing.T) {
	want := string(wavMagic(t))
	hexIn := wavMagicHex
	if out, err := runOp(t, "Play Media", hexIn, "Hex"); err != nil || out != want {
		t.Errorf("Hex/Raw = %q, %v; want %q", out, err, want)
	}
	b64 := base64.StdEncoding.EncodeToString(wavMagic(t))
	if out, err := runOp(t, "Play Media", b64, "Base64"); err != nil || out != want {
		t.Errorf("Base64/Raw = %q, %v; want %q", out, err, want)
	}
	if out, err := runOp(t, "Play Media", want, "Raw"); err != nil || out != want {
		t.Errorf("Raw/Raw = %q, %v; want %q", out, err, want)
	}
}

func TestPlayMediaEmpty(t *testing.T) {
	if out, err := runOp(t, "Play Media", "", "Raw"); err != nil || out != "" {
		t.Errorf("empty = %q, %v; want empty", out, err)
	}
}

func TestPlayMediaInvalid(t *testing.T) {
	if _, err := runOp(t, "Play Media", "hello world, not media", "Raw"); err == nil {
		t.Error("expected error for non-media input")
	}
	// An image is a recognised file type but not audio/video -> rejected.
	if _, err := runOp(t, "Play Media", pngMagicHex, "Hex"); err == nil {
		t.Error("expected error for image input to Play Media")
	}
}
