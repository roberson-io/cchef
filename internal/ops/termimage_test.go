package ops

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestTermProtocolFrom(t *testing.T) {
	cases := []struct {
		name, termProgram, term, kittyWindowID string
		want                                   termProtocol
	}{
		{"iterm", "iTerm.app", "xterm-256color", "", termITerm},
		{"wezterm", "WezTerm", "xterm-256color", "", termITerm},
		{"kitty via TERM", "", "xterm-kitty", "", termKitty},
		{"kitty via window id", "", "xterm-256color", "1", termKitty},
		{"unknown", "", "xterm-256color", "", termNone},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := termProtocolFrom(c.termProgram, c.term, c.kittyWindowID); got != c.want {
				t.Errorf("termProtocolFrom(%q,%q,%q) = %v, want %v", c.termProgram, c.term, c.kittyWindowID, got, c.want)
			}
		})
	}
}

func TestEncodeTerminalImageITerm(t *testing.T) {
	data := []byte("\x89PNGfake")
	out, err := encodeTerminalImage(termITerm, "image/png", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b64 := base64.StdEncoding.EncodeToString(data)
	want := "\x1b]1337;File=inline=1;size=8:" + b64 + "\x07\n"
	if string(out) != want {
		t.Errorf("iTerm2 encoding =\n%q\nwant\n%q", out, want)
	}
}

func TestEncodeTerminalImageKittyPNG(t *testing.T) {
	data := []byte("\x89PNG\r\n\x1a\nsmall")
	out, err := encodeTerminalImage(termKitty, "image/png", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b64 := base64.StdEncoding.EncodeToString(data)
	want := "\x1b_Gf=100,a=T,m=0;" + b64 + "\x1b\\\n"
	if string(out) != want {
		t.Errorf("kitty encoding =\n%q\nwant\n%q", out, want)
	}
}

// A payload longer than one kitty chunk must be split with continuation frames.
func TestEncodeTerminalImageKittyChunked(t *testing.T) {
	data := make([]byte, 4000) // base64 ~5336 chars > 4096 chunk
	for i := range data {
		data[i] = byte(i)
	}
	out, err := encodeTerminalImage(termKitty, "image/png", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := string(out)
	if !strings.HasPrefix(s, "\x1b_Gf=100,a=T,m=1;") {
		t.Errorf("first chunk should be a continuation frame (m=1), got prefix %q", s[:40])
	}
	if strings.Count(s, "\x1b_G") < 2 {
		t.Errorf("expected multiple kitty frames, got %d", strings.Count(s, "\x1b_G"))
	}
	if !strings.Contains(s, "m=0;") {
		t.Error("final frame should carry m=0")
	}
}

func TestEncodeTerminalImageErrors(t *testing.T) {
	if _, err := encodeTerminalImage(termNone, "image/png", []byte("x")); err == nil {
		t.Error("expected error for undetected terminal")
	}
	if _, err := encodeTerminalImage(termKitty, "image/jpeg", []byte("x")); err == nil {
		t.Error("expected error: kitty transmission requires PNG")
	}
}
