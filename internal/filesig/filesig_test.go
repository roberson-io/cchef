package filesig

import (
	"encoding/hex"
	"regexp"
	"strings"
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
			got := Detect(mustHex(t, c.hexInput), nil)
			found := false
			for _, ft := range got {
				if ft.Name == c.wantName && ft.MIME == c.wantMime {
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
	if got := Detect([]byte{0x89}, nil); got != nil {
		t.Errorf("buffers shorter than 2 bytes should not match, got %v", got)
	}
	png := mustHex(t, "89504e470d0a1a0a")
	if got := Detect(png, []string{"Audio"}); len(got) != 0 {
		t.Errorf("PNG should not match when only Audio category requested, got %v", got)
	}
	if got := Detect(png, []string{"Images"}); len(got) == 0 {
		t.Errorf("PNG should match when Images category requested")
	}
}

func TestIsImage(t *testing.T) {
	if mime := IsImage(mustHex(t, "89504e470d0a1a0a")); mime != "image/png" {
		t.Errorf("isImage(png) = %q, want image/png", mime)
	}
	if mime := IsImage([]byte("hello world, not an image")); mime != "" {
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
	if mime := IsTypeMatch(re, wav); mime != "audio/x-wav" {
		t.Errorf("isTypeMatch(audio|video, wav) = %q, want audio/x-wav", mime)
	}
	if mime := IsTypeMatch(re, []byte("plain text")); mime != "" {
		t.Errorf("isTypeMatch on text = %q, want empty", mime)
	}
}

// TestScanForFileTypes covers the scanner behind Extract Files: unlike
// detectFileType, which only asks what the buffer starts with, this reports
// every position a signature matches at.
func TestScanForFileTypes(t *testing.T) {
	png := mustHex(t, "89504e470d0a1a0a")
	gif := []byte("GIF89a")

	buf := append([]byte("junkjunk"), png...)
	buf = append(buf, []byte("filler")...)
	buf = append(buf, gif...)

	found := Scan(buf, nil)
	if len(found) < 2 {
		t.Fatalf("found %d signatures, want at least the PNG and the GIF", len(found))
	}
	// Offsets come back in increasing order.
	for i := 1; i < len(found); i++ {
		if found[i].Offset < found[i-1].Offset {
			t.Fatalf("offsets out of order: %d after %d", found[i].Offset, found[i-1].Offset)
		}
	}

	at := func(want int, ext string) {
		t.Helper()
		for _, f := range found {
			if f.Offset == want && strings.HasPrefix(f.Details.Extension, ext) {
				return
			}
		}
		t.Errorf("no %s reported at offset %d", ext, want)
	}
	at(8, "png")
	at(22, "gif")
}

// TestScanForFileTypesNonZeroFirstOffset covers a signature whose first
// constrained byte is not at the start of the file. WEBP is identified by
// "WEBP" at offset 8, so the scanner has to look back from where it finds the
// W to report where the file begins.
func TestScanForFileTypesNonZeroFirstOffset(t *testing.T) {
	buf := append([]byte("RIFF????WEBPVP8 "), make([]byte, 8)...)
	buf = append([]byte("pad!"), buf...)

	for _, f := range Scan(buf, []string{"Images"}) {
		if f.Details.Extension == "webp" {
			if f.Offset != 4 {
				t.Errorf("WEBP reported at offset %d, want 4", f.Offset)
			}
			return
		}
	}
	t.Error("the WEBP was not found")
}

// TestScanForFileTypesCategories covers restricting the scan to some categories.
func TestScanForFileTypesCategories(t *testing.T) {
	buf := append([]byte("junk"), mustHex(t, "89504e470d0a1a0a")...)

	if got := Scan(buf, []string{"Images"}); len(got) == 0 {
		t.Error("the PNG was not found when scanning Images")
	}
	for _, f := range Scan(buf, []string{"Audio"}) {
		if f.Details.Extension == "png" {
			t.Error("a PNG was reported when only Audio was asked for")
		}
	}
	if got := Scan(buf, []string{}); len(got) != 0 {
		t.Errorf("scanning no categories found %d signatures", len(got))
	}
}

// TestScanForFileTypesGuards covers buffers too short to hold a signature.
func TestScanForFileTypesGuards(t *testing.T) {
	for _, buf := range [][]byte{nil, {}, {0x89}} {
		if got := Scan(buf, nil); got != nil {
			t.Errorf("scanning %v found %d signatures, want none", buf, len(got))
		}
	}
}

// TestScanForFileTypesRepeated covers the same signature appearing more than
// once: each position is reported.
func TestScanForFileTypesRepeated(t *testing.T) {
	gif := []byte("GIF89a")
	buf := append(append(append([]byte("a"), gif...), []byte("bb")...), gif...)

	var offsets []int
	for _, f := range Scan(buf, []string{"Images"}) {
		if f.Details.Extension == "gif" {
			offsets = append(offsets, f.Offset)
		}
	}
	if len(offsets) != 2 || offsets[0] != 1 || offsets[1] != 9 {
		t.Errorf("GIF offsets = %v, want [1 9]", offsets)
	}
}

// TestFileSignatureTableWellFormed covers the invariants the scanner relies on.
// Every alternative must constrain at least one byte, because locating a
// candidate starts from the first check; and the checks must be ordered by the
// byte they constrain, because the offset the file starts at is worked out by
// stepping back from that first one.
func TestFileSignatureTableWellFormed(t *testing.T) {
	for _, cat := range Signatures {
		for _, ft := range cat.Types {
			if len(ft.alts) == 0 {
				t.Errorf("%s/%s has no signature", cat.Name, ft.Name)
			}
			for i, alt := range ft.alts {
				if len(alt) == 0 {
					t.Errorf("%s/%s alternative %d constrains no bytes", cat.Name, ft.Name, i)
					continue
				}
				for j := 1; j < len(alt); j++ {
					if alt[j].off < alt[j-1].off {
						t.Errorf("%s/%s alternative %d: offset %d follows %d",
							cat.Name, ft.Name, i, alt[j].off, alt[j-1].off)
					}
				}
			}
		}
	}
}
