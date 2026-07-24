package ops

import (
	"strings"
	"testing"
)

// allCats enables every category (the CyberChef default).
var allCats = []any{true, true, true, true, true, true, true}

func TestDetectFileType(t *testing.T) {
	png := string(mustHex(t, pngMagicHex))
	want := "File type:   Portable Network Graphics image\n" +
		"Extension:   png\n" +
		"MIME type:   image/png\n"
	if out, err := runOp(t, "Detect File Type", png, allCats...); err != nil || out != want {
		t.Errorf("PNG:\n got %q\nwant %q (err %v)", out, want, err)
	}

	// A type carrying a description emits a Description line (ELF).
	elf := string(mustHex(t, "7f454c46000102"))
	out, err := runOp(t, "Detect File Type", elf, allCats...)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Description: Executable and Linkable Format file.") {
		t.Errorf("ELF output missing Description line:\n%s", out)
	}
}

func TestDetectFileTypeUnknown(t *testing.T) {
	const unknownMsg = "Unknown file type. Have you tried checking the entropy of this data to determine whether it might be encrypted or compressed?"
	if out, err := runOp(t, "Detect File Type", "hello world, nothing here", allCats...); err != nil || out != unknownMsg {
		t.Errorf("unknown = %q, %v; want the unknown-type message", out, err)
	}
}

func TestDetectFileTypeCategoryFilter(t *testing.T) {
	png := string(mustHex(t, pngMagicHex))
	// Images disabled -> PNG no longer detected.
	args := []any{false, true, true, true, true, true, true}
	out, err := runOp(t, "Detect File Type", png, args...)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "Portable Network Graphics") {
		t.Errorf("PNG should not be detected with Images disabled:\n%s", out)
	}
}
