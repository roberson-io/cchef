package ops

import (
	"encoding/hex"
	"regexp"
	"testing"
)

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("bad hex %q: %v", s, err)
	}
	return b
}

// TestDetectFileTypeSignatures checks magic-byte detection for a spread of
// categories against the ported FileSignatures table.
func TestDetectFileTypeSignatures(t *testing.T) {
	cases := []struct {
		name, hexInput, wantName, wantMime string
	}{
		{"png", "89504e470d0a1a0a0000000d", "Portable Network Graphics image", "image/png"},
		{"jpeg", "ffd8ffe000104a4649", "Joint Photographic Experts Group image", "image/jpeg"},
		{"gif", "474946383961", "Graphics Interchange Format image", "image/gif"},
		{"bmp", "424d00000000000000000000000028000000", "Bitmap image", "image/bmp"},
		{"wav", "524946460000000057415645", "Waveform Audio", "audio/x-wav"},
		{"pdf", "255044462d312e30", "Portable Document Format", "application/pdf"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := detectFileType(mustHex(t, c.hexInput), nil)
			found := false
			for _, ft := range got {
				if ft.name == c.wantName && ft.mime == c.wantMime {
					found = true
				}
			}
			if !found {
				t.Errorf("detectFileType(%s): want a %q/%q match, got %v", c.name, c.wantName, c.wantMime, got)
			}
		})
	}
}

// TestDetectFileTypeGuards covers the short-buffer and category-filter branches.
func TestDetectFileTypeGuards(t *testing.T) {
	if got := detectFileType([]byte{0x89}, nil); got != nil {
		t.Errorf("buffers shorter than 2 bytes should not match, got %v", got)
	}
	png := mustHex(t, "89504e470d0a1a0a")
	if got := detectFileType(png, []string{"Audio"}); len(got) != 0 {
		t.Errorf("PNG should not match when only Audio category requested, got %v", got)
	}
	if got := detectFileType(png, []string{"Images"}); len(got) == 0 {
		t.Errorf("PNG should match when Images category requested")
	}
}

func TestIsImage(t *testing.T) {
	if mime := isImage(mustHex(t, "89504e470d0a1a0a")); mime != "image/png" {
		t.Errorf("isImage(png) = %q, want image/png", mime)
	}
	if mime := isImage([]byte("hello world, not an image")); mime != "" {
		t.Errorf("isImage(text) = %q, want empty", mime)
	}
}

func TestIsTypeString(t *testing.T) {
	wav := mustHex(t, "524946460000000057415645")
	if mime := isTypeString("audio", wav); mime != "audio/x-wav" {
		t.Errorf("isTypeString(audio, wav) = %q, want audio/x-wav", mime)
	}
	if mime := isTypeString("video", wav); mime != "" {
		t.Errorf("isTypeString(video, wav) = %q, want empty", mime)
	}
}

func TestIsTypeMatch(t *testing.T) {
	re := regexp.MustCompile(`^(audio|video)`)
	wav := mustHex(t, "524946460000000057415645")
	if mime := isTypeMatch(re, wav); mime != "audio/x-wav" {
		t.Errorf("isTypeMatch(audio|video, wav) = %q, want audio/x-wav", mime)
	}
	if mime := isTypeMatch(re, []byte("plain text")); mime != "" {
		t.Errorf("isTypeMatch on text = %q, want empty", mime)
	}
}
