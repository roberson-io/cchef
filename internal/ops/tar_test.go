package ops

import (
	"archive/tar"
	"bytes"
	"encoding/hex"
	"io"
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// tarGolden is one case from testdata/tar.jsonl: a filename, the data packed
// under it, and the archive CyberChef made.
//
// An archive records when it was written, in the header and again in the
// checksum over it, so both are blanked before comparing. The archive is kept
// with its trailing zeroes trimmed, which lets one golden stand for both what
// CyberChef writes and the whole-block archive written here.
type tarGolden struct {
	Name string `json:"name"`
	File string `json:"filename"`
	Kind string `json:"kind"`
	Spec struct {
		Hex  string `json:"hex"`
		Len  int    `json:"len"`
		Val  int    `json:"val"`
		Seed int64  `json:"seed"`
	} `json:"spec"`
	TrimmedHex   string `json:"trimmedHex"`
	WantLen      int    `json:"wantLen"`
	CyberChefLen int    `json:"cyberchefLen"`
}

// content rebuilds the bytes a golden packed.
func (g tarGolden) content(t *testing.T) []byte {
	t.Helper()
	switch g.Kind {
	case "raw":
		return unhex(t, g.Spec.Hex)
	case "byte":
		b := make([]byte, g.Spec.Len)
		for i := range b {
			b[i] = byte(g.Spec.Val)
		}
		return b
	case "text":
		n := g.Spec.Len/len(lz4Sentence) + 1
		return []byte(strings.Repeat(lz4Sentence, n))[:g.Spec.Len]
	case "random":
		return deflateRandom(g.Spec.Len, g.Spec.Seed)
	}
	t.Fatalf("%s: unknown input kind %q", g.Name, g.Kind)
	return nil
}

// untarGolden is one archive from testdata/untar.jsonl, written by the tar
// command, with the files it should give up.
type untarGolden struct {
	Name   string `json:"name"`
	TarHex string `json:"tarHex"`
	Files  []struct {
		Name   string `json:"name"`
		Len    int    `json:"len"`
		SHA256 string `json:"sha256"`
	} `json:"files"`
}

// blankTarTime clears the two header fields that say when the archive was
// written: the modification time, and the checksum that covers it.
func blankTarTime(data []byte) []byte {
	out := bytes.Clone(data)
	if len(out) < tarChecksumOffset+tarChecksumWidth {
		return out
	}
	for i := tarMTimeOffset; i < tarMTimeOffset+tarMTimeWidth; i++ {
		out[i] = 0
	}
	for i := tarChecksumOffset; i < tarChecksumOffset+tarChecksumWidth; i++ {
		out[i] = 0
	}
	return out
}

// TestTarFixtures covers CyberChef's own cases
// (../CyberChef/tests/node/tests/operations.mjs).
func TestTarFixtures(t *testing.T) {
	out, err := runOp(t, "Tar", "some file content", "test.txt")
	if err != nil {
		t.Fatalf("Tar: %v", err)
	}
	if len(out) != 2048 {
		t.Errorf("archive is %d bytes, want 2048", len(out))
	}
	if got := out[:8]; got != "test.txt" {
		t.Errorf("archive opens with %q, want %q", got, "test.txt")
	}

	files, err := runFileListOp(t, "Untar", out)
	if err != nil {
		t.Fatalf("Untar: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("Untar gave %d files, want 1", len(files))
	}
	if files[0].Name != "test.txt" || string(files[0].Data) != "some file content" {
		t.Errorf("got %q holding %q", files[0].Name, files[0].Data)
	}
}

// TestTarGoldens packs each case and compares the archive with CyberChef's,
// once the time it was written has been blanked out of both.
func TestTarGoldens(t *testing.T) {
	for _, g := range readJSONL[tarGolden](t, "testdata/tar.jsonl") {
		t.Run(g.Name, func(t *testing.T) {
			out, err := runOp(t, "Tar", string(g.content(t)), g.File)
			if err != nil {
				t.Fatalf("Tar: %v", err)
			}
			if len(out) != g.WantLen {
				t.Errorf("archive is %d bytes, want %d", len(out), g.WantLen)
			}
			trimmed := bytes.TrimRight(blankTarTime([]byte(out)), "\x00")
			if got := hex.EncodeToString(trimmed); got != g.TrimmedHex {
				t.Errorf("archive differs from CyberChef's\n got %s\nwant %s",
					got, g.TrimmedHex)
			}
		})
	}
}

// TestTarPadsToWholeBlocks pins a fault in CyberChef's writer. A tar archive is
// a run of 512-byte blocks, but CyberChef pads the data to one block and no
// further, so anything longer than a block that is not a multiple of one leaves
// the archive ending mid-block. Go's own tar reader refuses such an archive.
func TestTarPadsToWholeBlocks(t *testing.T) {
	for _, g := range readJSONL[tarGolden](t, "testdata/tar.jsonl") {
		if g.CyberChefLen%tarBlockSize == 0 {
			continue
		}
		t.Run(g.Name, func(t *testing.T) {
			out, err := runOp(t, "Tar", string(g.content(t)), g.File)
			if err != nil {
				t.Fatalf("Tar: %v", err)
			}
			if len(out)%tarBlockSize != 0 {
				t.Errorf("archive is %d bytes, which is not whole blocks", len(out))
			}
			if _, err := tar.NewReader(strings.NewReader(out)).Next(); err != nil {
				t.Errorf("Go's tar reader refused the archive: %v", err)
			}
		})
	}
}

// TestTarChecksumIsCorrect works the header checksum out the way every tar
// reader does — the sum of all 512 bytes with the checksum field taken as
// spaces — and holds the archive to it.
func TestTarChecksumIsCorrect(t *testing.T) {
	out, err := runOp(t, "Tar", "some file content", "test.txt")
	if err != nil {
		t.Fatalf("Tar: %v", err)
	}
	header := []byte(out[:tarBlockSize])

	want := 0
	for i, b := range header {
		if i >= tarChecksumOffset && i < tarChecksumOffset+tarChecksumWidth {
			b = ' '
		}
		want += int(b)
	}

	field := string(header[tarChecksumOffset : tarChecksumOffset+tarChecksumWidth])
	if got := strings.Trim(field, "\x00 "); got != tarOctalField(int64(want), tarChecksumDigits) {
		t.Errorf("checksum field is %q, want %q", got, tarOctalField(int64(want), tarChecksumDigits))
	}
}

// TestTarRejectsALongFilename covers a name too big for the field it goes in.
// CyberChef writes it anyway, pushing every field after it out of place and
// leaving an archive nothing can read.
func TestTarRejectsALongFilename(t *testing.T) {
	if _, err := runOp(t, "Tar", "hi", strings.Repeat("a", tarNameWidth+1)); err == nil {
		t.Fatal("packed a name too long for the header")
	}
	if _, err := runOp(t, "Tar", "hi", strings.Repeat("a", tarNameWidth)); err != nil {
		t.Errorf("a name of exactly %d bytes was refused: %v", tarNameWidth, err)
	}
}

// TestTarWritesNamesAsUTF8 covers a name outside ASCII. CyberChef narrows each
// UTF-16 code unit to a single byte, which turns "café.txt" into Latin-1 and
// loses anything above the basic plane outright.
func TestTarWritesNamesAsUTF8(t *testing.T) {
	for _, name := range []string{"café.txt", "日本語.txt", "😀.txt"} {
		t.Run(name, func(t *testing.T) {
			out, err := runOp(t, "Tar", "hi", name)
			if err != nil {
				t.Fatalf("Tar: %v", err)
			}
			got := strings.TrimRight(out[:tarNameWidth], "\x00")
			if got != name {
				t.Errorf("header holds %q, want %q", got, name)
			}
			files, err := runFileListOp(t, "Untar", out)
			if err != nil {
				t.Fatalf("Untar: %v", err)
			}
			if len(files) != 1 || files[0].Name != name {
				t.Errorf("read back %v, want one file named %q", files, name)
			}
		})
	}
}

// TestUntarGoldens reads archives written by the tar command, holding several
// files, a directory tree, an empty file, a symbolic link and a name too long
// for the header. Only the regular files come back.
func TestUntarGoldens(t *testing.T) {
	for _, g := range readJSONL[untarGolden](t, "testdata/untar.jsonl") {
		t.Run(g.Name, func(t *testing.T) {
			files, err := runFileListOp(t, "Untar", string(unhex(t, g.TarHex)))
			if err != nil {
				t.Fatalf("Untar: %v", err)
			}
			if len(files) != len(g.Files) {
				t.Fatalf("gave %d files, want %d: %v", len(files), len(g.Files), names(files))
			}
			for i, want := range g.Files {
				if files[i].Name != want.Name {
					t.Errorf("file %d is named %q, want %q", i, files[i].Name, want.Name)
				}
				if len(files[i].Data) != want.Len {
					t.Errorf("%s is %d bytes, want %d", want.Name, len(files[i].Data), want.Len)
				}
				if sum := digest(files[i].Data); sum != want.SHA256 {
					t.Errorf("%s digest %s, want %s", want.Name, sum, want.SHA256)
				}
			}
		})
	}
}

// names lists what a file list holds, for a failure message.
func names(files []core.NamedFile) []string {
	out := make([]string, len(files))
	for i, f := range files {
		out[i] = f.Name
	}
	return out
}

// TestUntarReadsCyberChefArchives covers the archives CyberChef itself writes,
// including the ones it leaves ending mid-block. Refusing those would mean
// refusing every archive CyberChef has ever produced from data over 512 bytes.
func TestUntarReadsCyberChefArchives(t *testing.T) {
	for _, g := range readJSONL[tarGolden](t, "testdata/tar.jsonl") {
		t.Run(g.Name, func(t *testing.T) {
			// The archives agree byte for byte up to the point CyberChef stops,
			// which TestTarGoldens holds them to, so cutting this one there
			// gives an archive laid out exactly as CyberChef lays one out.
			archive, err := runOp(t, "Tar", string(g.content(t)), g.File)
			if err != nil {
				t.Fatalf("Tar: %v", err)
			}

			files, err := runFileListOp(t, "Untar", archive[:g.CyberChefLen])
			if err != nil {
				t.Fatalf("Untar: %v", err)
			}
			want := g.content(t)
			if len(files) != 1 {
				t.Fatalf("gave %d files, want 1", len(files))
			}
			if files[0].Name != g.File {
				t.Errorf("named %q, want %q", files[0].Name, g.File)
			}
			if !bytes.Equal(files[0].Data, want) {
				t.Errorf("holds %d bytes, want %d", len(files[0].Data), len(want))
			}
		})
	}
}

// TestUntarRoundTrips packs each golden's data and reads it straight back.
func TestUntarRoundTrips(t *testing.T) {
	for _, g := range readJSONL[tarGolden](t, "testdata/tar.jsonl") {
		t.Run(g.Name, func(t *testing.T) {
			want := g.content(t)
			archive, err := runOp(t, "Tar", string(want), g.File)
			if err != nil {
				t.Fatalf("Tar: %v", err)
			}
			files, err := runFileListOp(t, "Untar", archive)
			if err != nil {
				t.Fatalf("Untar: %v", err)
			}
			if len(files) != 1 || !bytes.Equal(files[0].Data, want) {
				t.Errorf("round trip changed the data")
			}
		})
	}
}

// TestUntarRejectsBadInput covers input that is not an archive at all.
func TestUntarRejectsBadInput(t *testing.T) {
	for _, in := range []string{"hello, this is not a tarball", strings.Repeat("\x01", 600)} {
		if _, err := runFileListOp(t, "Untar", in); err == nil {
			t.Errorf("read %q as an archive", in[:min(20, len(in))])
		}
	}
}

// TestUntarOfNothing covers empty input, which holds no files rather than
// being an error.
func TestUntarOfNothing(t *testing.T) {
	files, err := runFileListOp(t, "Untar", "")
	if err != nil {
		t.Fatalf("Untar: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("gave %d files, want none", len(files))
	}
}

// TestTarHeaderFieldBounds covers the size and the time, both of which the
// header keeps as eleven octal digits and no more.
func TestTarHeaderFieldBounds(t *testing.T) {
	header, err := tarHeader("big.bin", 1<<32, 0)
	if err != nil {
		t.Fatalf("tarHeader: %v", err)
	}
	size := strings.TrimRight(string(header[tarSizeOffset:tarSizeOffset+tarSizeWidth+1]), "\x00")
	if size != "40000000000" {
		t.Errorf("size field is %q, want %q", size, "40000000000")
	}
	if _, err := tarHeader("big.bin", 1<<34, 0); err == nil {
		t.Error("accepted a size too big for the field")
	}
	if _, err := tarHeader("big.bin", 0, 1<<40); err == nil {
		t.Error("accepted a time too big for the field")
	}
}

// TestUntarRejectsATruncatedFile covers an archive that stops partway through a
// file's data, rather than partway through the zeroes that close it.
func TestUntarRejectsATruncatedFile(t *testing.T) {
	archive, err := runOp(t, "Tar", "some file content", "test.txt")
	if err != nil {
		t.Fatalf("Tar: %v", err)
	}
	if _, err := runFileListOp(t, "Untar", archive[:tarBlockSize+8]); err == nil {
		t.Fatal("read an archive that stops partway through its file")
	}
}

// tarReadAll is the shape the reader is held to: what Go's own tar reader makes
// of an archive, so that the two agree on ordinary input.
func tarReadAll(t *testing.T, archive string) []string {
	t.Helper()
	r := tar.NewReader(strings.NewReader(archive))
	var out []string
	for {
		h, err := r.Next()
		if err == io.EOF {
			return out
		}
		if err != nil {
			t.Fatalf("Go's tar reader: %v", err)
		}
		out = append(out, h.Name)
	}
}

// TestTarAgreesWithGoReader checks cchef's archives against Go's tar reader,
// which is stricter than CyberChef's and refuses a short final block.
func TestTarAgreesWithGoReader(t *testing.T) {
	for _, g := range readJSONL[tarGolden](t, "testdata/tar.jsonl") {
		t.Run(g.Name, func(t *testing.T) {
			archive, err := runOp(t, "Tar", string(g.content(t)), g.File)
			if err != nil {
				t.Fatalf("Tar: %v", err)
			}
			if got := tarReadAll(t, archive); len(got) != 1 || got[0] != g.File {
				t.Errorf("Go's tar reader saw %v, want [%q]", got, g.File)
			}
		})
	}
}
