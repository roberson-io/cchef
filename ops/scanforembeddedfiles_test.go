package ops

import (
	"strings"
	"testing"
)

// scanHeader opens every report, whatever is found. The wording (and the
// trailing newline before the first result) is CyberChef's.
const scanHeader = "Scanning data for 'magic bytes' which may indicate embedded files. " +
	"The following results may be false positives and should not be treated as reliable. " +
	"Any sufficiently long file is likely to contain these magic bytes coincidentally.\n"

// scanNothing is the whole report when no signature matches.
const scanNothing = scanHeader + "\nNo embedded files were found."

// scanArgs builds the seven category booleans in declared order, with the
// operation's defaults: everything on except Miscellaneous.
func scanArgs(overrides map[string]bool) []any {
	out := make([]any, len(detectFileTypeCats))
	for i, cat := range detectFileTypeCats {
		on := cat != "Miscellaneous"
		if v, there := overrides[cat]; there {
			on = v
		}
		out[i] = on
	}
	return out
}

// The sample files below are the smallest byte strings the corresponding
// signatures accept. Expected outputs are recorded from CyberChef
// (Node API, its own FileSignatures table).
var (
	scanPNG = "\x89PNG\r\n\x1a\n\x00\x00\x00\x0dIHDR"
	scanGIF = "GIF89a\x00\x00\x00\x00"
	scanZIP = "PK\x03\x04\x14\x00\x00\x00"
	scanELF = "\x7fELF\x02\x01\x01" + strings.Repeat("\x00", 9)
	scanBOM = "\xef\xbb\xbfhello"
)

// TestScanForEmbeddedFilesFinds covers matches at several offsets, the hex
// rendering of an offset, and the Description line.
func TestScanForEmbeddedFilesFinds(t *testing.T) {
	pngBlock := "  File type:   Portable Network Graphics image\n" +
		"  Extension:   png\n" +
		"  MIME type:   image/png\n"
	cases := []struct {
		name  string
		input string
		args  []any
		want  string
	}{
		{
			"png at zero", scanPNG, scanArgs(nil),
			scanHeader + "\nOffset 0 (0x00):\n" + pngBlock,
		},
		{
			"png at five", "12345" + scanPNG, scanArgs(nil),
			scanHeader + "\nOffset 5 (0x05):\n" + pngBlock,
		},
		{
			"two files in offset order", "x" + scanPNG + "pad" + scanGIF, scanArgs(nil),
			scanHeader + "\nOffset 1 (0x01):\n" + pngBlock +
				"\nOffset 20 (0x14):\n" +
				"  File type:   Graphics Interchange Format image\n" +
				"  Extension:   gif\n" +
				"  MIME type:   image/gif\n",
		},
		{
			"offset above 0xff keeps lowercase hex", strings.Repeat("\x00", 300) + scanPNG, scanArgs(nil),
			scanHeader + "\nOffset 300 (0x12c):\n" + pngBlock,
		},
		{
			"description line when the type has one", "pad" + scanELF, scanArgs(nil),
			scanHeader + "\nOffset 3 (0x03):\n" +
				"  File type:   Executable and Linkable Format\n" +
				"  Extension:   elf,bin,axf,o,prx,so\n" +
				"  MIME type:   application/x-executable\n" +
				"  Description: Executable and Linkable Format file. No standard file extension.\n",
		},
		{
			"zip found with everything on", "x" + scanZIP + "y", scanArgs(map[string]bool{"Miscellaneous": true}),
			scanHeader + "\nOffset 1 (0x01):\n" +
				"  File type:   PKZIP archive\n" +
				"  Extension:   zip\n" +
				"  MIME type:   application/zip\n",
		},
		{
			"utf-8 mark once miscellaneous is on", scanBOM, scanArgs(map[string]bool{"Miscellaneous": true}),
			scanHeader + "\nOffset 0 (0x00):\n" +
				"  File type:   UTF-8 text\n" +
				"  Extension:   txt\n" +
				"  MIME type:   text/plain\n" +
				"  Description: UTF-8 encoded Unicode byte order mark, commonly but not exclusively seen in text files.\n",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := runOp(t, "Scan for Embedded Files", c.input, c.args...)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if got != c.want {
				t.Errorf("got  %q\nwant %q", got, c.want)
			}
		})
	}
}

// TestScanForEmbeddedFilesFindsNothing covers empty and too-short input, input
// with no signatures in it, and categories switched off.
func TestScanForEmbeddedFilesFindsNothing(t *testing.T) {
	cases := []struct {
		name  string
		input string
		args  []any
	}{
		{"empty input", "", scanArgs(nil)},
		{"single byte", "A", scanArgs(nil)},
		{"no signatures present", "just some text with nothing in it", scanArgs(nil)},
		{"gif with images off", scanGIF, scanArgs(map[string]bool{"Images": false})},
		{"png with every category off", scanPNG, scanArgs(map[string]bool{
			"Images": false, "Video": false, "Audio": false, "Documents": false,
			"Applications": false, "Archives": false,
		})},
		{"utf-8 mark under the defaults", scanBOM, scanArgs(nil)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := runOp(t, "Scan for Embedded Files", c.input, c.args...)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if got != scanNothing {
				t.Errorf("got  %q\nwant %q", got, scanNothing)
			}
		})
	}
}
