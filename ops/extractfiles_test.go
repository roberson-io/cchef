package ops

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/filecarve"
	"github.com/roberson-io/cchef/internal/filesig"
)

// extractFilesArgs returns the operation's arguments with every category asked
// for, failed extractions ignored, and no size floor.
func extractFilesArgs(minSize float64) []any {
	args := make([]any, 0, len(filesig.Signatures)+2)
	for range filesig.Signatures {
		args = append(args, true)
	}
	return append(args, true, minSize)
}

// runExtractFiles runs the operation over a buffer and returns the files it cut
// out.
func runExtractFiles(t *testing.T, buf []byte, args []any) []core.NamedFile {
	t.Helper()
	op, ok := core.Default.Get("Extract Files")
	if !ok {
		t.Fatal("Extract Files is not registered")
	}
	coerced, err := core.CoerceArgs(op.Args(), args)
	if err != nil {
		t.Fatalf("arguments: %v", err)
	}
	out, err := op.Run(core.NewDish(buf, core.TypeArrayBuffer), coerced)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.Type() != core.TypeFileList {
		t.Fatalf("output type = %q, want %q", out.Type(), core.TypeFileList)
	}
	return out.Files()
}

// carveBlob builds a buffer holding the named samples one after another, with
// padding between them, and returns it along with where each sample starts.
func carveBlob(t *testing.T, names ...string) ([]byte, map[string]int) {
	t.Helper()
	var buf []byte
	at := map[string]int{}
	buf = append(buf, carvePrefix...)
	for _, name := range names {
		at[name] = len(buf)
		buf = append(buf, readCarveSample(t, name)...)
		buf = append(buf, []byte("-----PADDING-----")...)
	}
	return append(buf, carveSuffix...), at
}

// TestExtractFilesFindsEachFile covers the operation over a buffer holding
// several files: each is cut out whole and named for where it was found.
func TestExtractFilesFindsEachFile(t *testing.T) {
	names := []string{"sample.png", "sample.gz", "sample.rtf"}
	buf, at := carveBlob(t, names...)

	files := runExtractFiles(t, buf, extractFilesArgs(1))

	for _, name := range names {
		want := readCarveSample(t, name)
		wantName := fmt.Sprintf("extracted_at_0x%x.%s", at[name], strings.TrimPrefix(name, "sample."))
		found := false
		for _, f := range files {
			if f.Name == wantName {
				found = true
				if !bytes.Equal(f.Data, want) {
					t.Errorf("%s: carved %d bytes, want %d", wantName, len(f.Data), len(want))
				}
			}
		}
		if !found {
			t.Errorf("%s was not among the files extracted", wantName)
		}
	}
}

// TestExtractFilesMinimumSize covers the size floor, which is there to prune
// small false positives.
func TestExtractFilesMinimumSize(t *testing.T) {
	buf, _ := carveBlob(t, "sample.rtf", "sample.sqlite")

	small := runExtractFiles(t, buf, extractFilesArgs(1))
	large := runExtractFiles(t, buf, extractFilesArgs(1024))

	if len(large) >= len(small) {
		t.Errorf("a floor of 1024 kept %d files, no fewer than the %d with no floor", len(large), len(small))
	}
	for _, f := range large {
		if len(f.Data) < 1024 {
			t.Errorf("%s is %d bytes, below the floor", f.Name, len(f.Data))
		}
	}
	if len(large) == 0 {
		t.Error("the floor removed everything, including the 8192-byte database")
	}
}

// TestExtractFilesCategories covers restricting the scan, which is what the
// per-category arguments are for.
func TestExtractFilesCategories(t *testing.T) {
	buf, _ := carveBlob(t, "sample.png", "sample.gz")

	// Every category off but Images.
	args := make([]any, 0, len(filesig.Signatures)+2)
	for _, cat := range filesig.Signatures {
		args = append(args, cat.Name == "Images")
	}
	args = append(args, true, 1.0)

	for _, f := range runExtractFiles(t, buf, args) {
		if strings.HasSuffix(f.Name, ".gz") {
			t.Errorf("%s was extracted when only Images was asked for", f.Name)
		}
	}
}

// TestExtractFilesReportsFailures covers the "ignore failed extractions"
// argument. A signature that matches but cannot be carved is dropped silently
// when it is on, and reported when it is off.
func TestExtractFilesReportsFailures(t *testing.T) {
	// A PNG signature with nothing usable after it.
	buf := append(append([]byte{}, carvePrefix...), mustHex(t, "89504e470d0a1a0a")...)
	buf = append(buf, bytes.Repeat([]byte{0x00}, 8)...)

	if files := runExtractFiles(t, buf, extractFilesArgs(1)); len(files) != 0 {
		t.Errorf("ignoring failures still produced %d files", len(files))
	}

	op, _ := core.Default.Get("Extract Files")
	args := extractFilesArgs(1)
	args[len(args)-2] = false
	coerced, err := core.CoerceArgs(op.Args(), args)
	if err != nil {
		t.Fatalf("arguments: %v", err)
	}
	_, err = op.Run(core.NewDish(buf, core.TypeArrayBuffer), coerced)
	if err == nil {
		t.Fatal("a failed extraction was not reported")
	}
	if !strings.Contains(err.Error(), "Error while attempting to extract") {
		t.Errorf("got %q, want CyberChef's wording", err.Error())
	}
}

// TestExtractFilesIgnoresUncarvableTypes covers a type the scanner recognises
// but that no algorithm can cut out. That is not a failure worth reporting —
// most of the signature table is in that position — so it stays quiet even when
// failures are being reported.
func TestExtractFilesIgnoresUncarvableTypes(t *testing.T) {
	// A WebAssembly binary is detected but has no extraction algorithm.
	buf := append(append([]byte{}, carvePrefix...), mustHex(t, "0061736d01000000")...)
	buf = append(buf, carveSuffix...)

	op, _ := core.Default.Get("Extract Files")
	args := extractFilesArgs(1)
	args[len(args)-2] = false
	coerced, err := core.CoerceArgs(op.Args(), args)
	if err != nil {
		t.Fatalf("arguments: %v", err)
	}
	if _, err := op.Run(core.NewDish(buf, core.TypeArrayBuffer), coerced); err != nil {
		t.Errorf("a type with no algorithm was reported as a failure: %v", err)
	}
}

// TestExtractFilesEmptyInput covers a buffer with nothing in it.
func TestExtractFilesEmptyInput(t *testing.T) {
	if files := runExtractFiles(t, nil, extractFilesArgs(1)); len(files) != 0 {
		t.Errorf("an empty buffer produced %d files", len(files))
	}
}

// TestExtractFilesArgsMatchCategories covers the argument list keeping step with
// the signature table: there is one switch per category, then the two settings.
func TestExtractFilesArgsMatchCategories(t *testing.T) {
	op, ok := core.Default.Get("Extract Files")
	if !ok {
		t.Fatal("Extract Files is not registered")
	}
	args := op.Args()
	if len(args) != len(filesig.Signatures)+2 {
		t.Fatalf("%d arguments for %d categories", len(args), len(filesig.Signatures))
	}
	for i, cat := range filesig.Signatures {
		if args[i].Name != cat.Name {
			t.Errorf("argument %d is %q, want %q", i, args[i].Name, cat.Name)
		}
		// Every category is searched by default except the catch-all one.
		want := cat.Name != "Miscellaneous"
		if args[i].Value != want {
			t.Errorf("%s defaults to %v, want %v", cat.Name, args[i].Value, want)
		}
	}
}

// TestExtractFilesChainsAsConcatenation covers a list-of-files output feeding a
// following operation as its files' contents concatenated.
func TestExtractFilesChainsAsConcatenation(t *testing.T) {
	buf, _ := carveBlob(t, "sample.png")
	files := runExtractFiles(t, buf, extractFilesArgs(1))
	if len(files) == 0 {
		t.Fatal("nothing was extracted")
	}
	// Chained into a following operation, the extracted files feed on as their
	// contents concatenated in order.
	got, err := core.NewFileListDish(files).Get(core.TypeByteArray)
	if err != nil {
		t.Fatalf("Get(ByteArray): %v", err)
	}
	var want []byte
	for _, f := range files {
		want = append(want, f.Data...)
	}
	if !bytes.Equal(got.([]byte), want) {
		t.Errorf("concatenation mismatch: got %d bytes, want %d", len(got.([]byte)), len(want))
	}
}

// TestExtractFilesAdvertisedFormats covers the list of formats the description
// offers. CyberChef builds it from the extension field of every signature that
// has an extraction algorithm, keeping each signature's extensions together and
// dropping repeats, and the operation is only honest if cchef can carve exactly
// what it advertises. The expected list is CyberChef's own, transcribed from the
// operation's description.
func TestExtractFilesAdvertisedFormats(t *testing.T) {
	want := []string{
		"JPG,JPEG,JPE,THM,MPO", "GIF", "PNG", "WEBP", "BMP", "ICO", "TGA",
		"FLV", "WAV", "MP3", "PDF", "RTF", "DOCX,XLSX,PPTX", "EPUB",
		"EXE,DLL,DRV,VXD,SYS,OCX,VBX,COM,FON,SCR", "ELF,BIN,AXF,O,PRX,SO",
		"DYLIB", "ZIP", "TAR", "GZ", "BZ2", "ZLIB", "XZ", "JAR", "LZOP,LZO",
		"DEB", "SQLITE", "EVT", "EVTX", "DMP", "PF", "PLIST", "KEYCHAIN", "LNK",
	}

	got := carvableExtensions()
	if len(got) != len(want) {
		t.Fatalf("advertises %d formats, want %d:\ngot  %v\nwant %v", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("format %d = %q, want %q", i, got[i], want[i])
		}
	}

	// Every one of them must actually be carvable.
	for _, ext := range got {
		found := false
		for _, cat := range filesig.Signatures {
			for _, ft := range cat.Types {
				if strings.EqualFold(ft.Extension, ext) && ft.Carver != "" {
					if filecarve.CanCarve(ft.Carver) {
						found = true
					}
				}
			}
		}
		if !found {
			t.Errorf("%s is advertised but nothing can carve it", ext)
		}
	}
}

// TestExtractFilesDescriptionListsFormats covers the description itself carrying
// the list, which is where a user reads what the operation can recover.
func TestExtractFilesDescriptionListsFormats(t *testing.T) {
	op, ok := core.Default.Get("Extract Files")
	if !ok {
		t.Fatal("Extract Files is not registered")
	}
	description := op.Meta().Description
	for _, ext := range carvableExtensions() {
		if !strings.Contains(description, ext) {
			t.Errorf("the description does not mention %s", ext)
		}
	}
}
