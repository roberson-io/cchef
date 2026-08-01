package ops

import (
	"encoding/hex"
	"testing"
)

// removeEXIFHex runs Remove EXIF on raw bytes decoded from inHex and returns the
// output as lowercase hex.
func removeEXIFHex(t *testing.T, inHex string) (string, error) {
	t.Helper()
	out, err := runOp(t, "Remove EXIF", string(mustHex(t, inHex)))
	if err != nil {
		return "", err
	}
	return hex.EncodeToString([]byte(out)), nil
}

func TestRemoveEXIF(t *testing.T) {
	// SOI | APP1 "Exif\0\0" | SOS+rest  ->  SOI | SOS+rest  (APP1 segment dropped)
	withEXIF := "ffd8ffe10008457869660000ffda000200"
	wantStripped := "ffd8ffda000200"
	if got, err := removeEXIFHex(t, withEXIF); err != nil || got != wantStripped {
		t.Errorf("strip: got %q, %v; want %q", got, err, wantStripped)
	}

	// APP1 EXIF as the second app segment (after APP0) is also removed.
	withEXIF2 := "ffd8ffe0000400" + "00" + "ffe10008457869660000" + "ffda000200"
	// segments: SOI | APP0(ffe0 0004 0000) | APP1 EXIF | SOS ; APP1 dropped
	wantStripped2 := "ffd8ffe000040000ffda000200"
	if got, err := removeEXIFHex(t, withEXIF2); err != nil || got != wantStripped2 {
		t.Errorf("strip after APP0: got %q, %v; want %q", got, err, wantStripped2)
	}
}

func TestRemoveEXIFNotFound(t *testing.T) {
	// A JPEG with no EXIF segment is returned unchanged (CyberChef swallows the
	// "Exif not found." error and returns the input).
	noEXIF := "ffd8ffe000040000ffda000200"
	if got, err := removeEXIFHex(t, noEXIF); err != nil || got != noEXIF {
		t.Errorf("no-exif: got %q, %v; want unchanged %q", got, err, noEXIF)
	}
}

func TestRemoveEXIFEmpty(t *testing.T) {
	if out, err := runOp(t, "Remove EXIF", ""); err != nil || out != "" {
		t.Errorf("empty = %q, %v; want empty", out, err)
	}
}

// Malformed/truncated JPEGs exercise splitJPEGSegments' error and clamp paths.
func TestRemoveEXIFMalformed(t *testing.T) {
	for _, in := range []string{
		"ffd8ff",       // marker cut off (head+2 > len)
		"ffd8ffe1",     // segment length cut off (head+4 > len)
		"ffd8ffe1ffff", // declared length overruns the buffer, then runs out
	} {
		if _, err := runOp(t, "Remove EXIF", string(mustHex(t, in))); err == nil {
			t.Errorf("expected error for malformed JPEG %q", in)
		}
	}
}

func TestRemoveEXIFNotJPEG(t *testing.T) {
	if _, err := runOp(t, "Remove EXIF", "hello world"); err == nil {
		t.Error("expected error for non-JPEG input")
	} else if err.Error() != "Could not remove EXIF data from image: Given data is not jpeg." {
		t.Errorf("unexpected error text: %q", err.Error())
	}
}
