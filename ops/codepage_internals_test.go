package ops

import (
	"bytes"
	"compress/gzip"
	"testing"
)

// gzipOf returns the gzip encoding of payload.
func gzipOf(t *testing.T, payload []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(payload); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	return buf.Bytes()
}

// TestLoadCodepagesErrors covers the blob-parsing error paths.
func TestLoadCodepagesErrors(t *testing.T) {
	truncated := gzipOf(t, make([]byte, 4096))
	truncated = truncated[:len(truncated)-6] // valid header, broken deflate stream
	cases := []struct {
		name string
		blob []byte
	}{
		{"bad gzip", []byte{0x00, 0x01, 0x02}},
		{"truncated stream", truncated},
		{"empty payload", gzipOf(t, nil)},                                                                  // no count
		{"missing codepage", gzipOf(t, []byte{0x00, 0x01})},                                                // count=1, no cp
		{"missing entry count", gzipOf(t, []byte{0x00, 0x01, 0x03, 0xE8})},                                 // cp but no n
		{"truncated entry", gzipOf(t, []byte{0x00, 0x01, 0x03, 0xE8, 0x00, 0x00, 0x00, 0x01, 0x00, 0x41})}, // n=1, half a pair
	}
	for _, c := range cases {
		if _, err := loadCodepages(c.blob); err == nil {
			t.Errorf("%s: expected error", c.name)
		}
	}
}

// TestCodepageHelperBranches covers helper branches unreachable through the
// normal 152-charset dispatch.
func TestCodepageHelperBranches(t *testing.T) {
	if _, err := magicDecode("bogus", nil); err == nil {
		t.Error("magicDecode(bogus): expected error")
	}
	if _, err := magicEncode("bogus", ""); err == nil {
		t.Error("magicEncode(bogus): expected error")
	}
	// sbcsDecode errors when a byte has no table entry (unreachable for real
	// full 256-entry SBCS tables).
	if _, err := sbcsDecode(&codepage{dec: map[uint16]uint16{}}, []byte{0x41}); err == nil {
		t.Error("sbcsDecode with empty table: expected error")
	}
	// getUnit returns 0 past the end (mirrors a lone trailing surrogate).
	if getUnit([]uint16{1}, 5) != 0 {
		t.Error("getUnit out of range should be 0")
	}
}
