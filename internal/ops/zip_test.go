package ops

import (
	"archive/zip"
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// zipGolden is one case from testdata/zip.jsonl: an input described rather than
// stored, the arguments Zip was given, and the archive CyberChef produced with
// the times blanked out.
type zipGolden struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	Spec struct {
		Hex  string `json:"hex"`
		Len  int    `json:"len"`
		N    int    `json:"n"`
		Seed int64  `json:"seed"`
	} `json:"spec"`
	Args   []any  `json:"args"`
	OutLen int    `json:"outLen"`
	SHA256 string `json:"sha256"`
	Hex    string `json:"hex"`
}

// zipSentence is the text the repeated-text cases are built from.
const zipSentence = "The quick brown fox jumps over the lazy dog. "

// input rebuilds the bytes a golden was made from.
func (g zipGolden) input(t *testing.T) []byte {
	t.Helper()
	switch g.Kind {
	case "raw":
		b, err := hex.DecodeString(g.Spec.Hex)
		if err != nil {
			t.Fatalf("%s: bad hex: %v", g.Name, err)
		}
		return b
	case "text":
		out := make([]byte, 0, len(zipSentence)*g.Spec.N)
		for range g.Spec.N {
			out = append(out, zipSentence...)
		}
		return out
	case "random":
		return deflateRandom(g.Spec.Len, g.Spec.Seed)
	}
	t.Fatalf("%s: unknown input kind %q", g.Name, g.Kind)
	return nil
}

// zipBlankTimes blanks the two places an archive records when it was written,
// so that two runs can be compared. The central directory is found by its
// signature rather than by counting.
func zipBlankTimes(data []byte) []byte {
	out := append([]byte{}, data...)
	if len(out) >= 14 {
		copy(out[10:14], make([]byte, 4))
	}
	if at := bytes.Index(out, []byte("PK\x01\x02")); at >= 0 && len(out) >= at+16 {
		copy(out[at+12:at+16], make([]byte, 4))
	}
	return out
}

// loadZipGoldens reads the corpus.
func loadZipGoldens(t *testing.T) []zipGolden {
	t.Helper()
	f, err := os.Open("testdata/zip.jsonl")
	if err != nil {
		t.Fatalf("open goldens: %v", err)
	}
	defer func() { _ = f.Close() }()

	var out []zipGolden
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		var g zipGolden
		if err := json.Unmarshal(sc.Bytes(), &g); err != nil {
			t.Fatalf("parse goldens: %v", err)
		}
		out = append(out, g)
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("read goldens: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("no goldens loaded")
	}
	return out
}

// TestZipGoldens checks the archive bytes against CyberChef's own, across both
// compression methods, every operating system, and the filename and comment
// fields.
func TestZipGoldens(t *testing.T) {
	for _, g := range loadZipGoldens(t) {
		t.Run(g.Name, func(t *testing.T) {
			out, err := runOp(t, "Zip", string(g.input(t)), g.Args...)
			if err != nil {
				t.Fatalf("Zip: %v", err)
			}
			got := zipBlankTimes([]byte(out))
			if len(got) != g.OutLen {
				t.Errorf("produced %d bytes, want %d", len(got), g.OutLen)
			}
			if sum := fmt.Sprintf("%x", sha256.Sum256(got)); sum != g.SHA256 {
				t.Errorf("digest %s, want %s", sum, g.SHA256)
				if g.Hex != "" {
					t.Errorf("got  %x\nwant %s", got, g.Hex)
				}
			}
		})
	}
}

// TestZipReadableByStandardLibrary checks that every archive is one any reader
// will take, by opening it with the standard library rather than with cchef's
// own reader.
func TestZipReadableByStandardLibrary(t *testing.T) {
	for _, g := range loadZipGoldens(t) {
		t.Run(g.Name, func(t *testing.T) {
			want := g.input(t)
			out, err := runOp(t, "Zip", string(want), g.Args...)
			if err != nil {
				t.Fatalf("Zip: %v", err)
			}
			r, err := zip.NewReader(bytes.NewReader([]byte(out)), int64(len(out)))
			if err != nil {
				t.Fatalf("archive/zip: %v", err)
			}
			if len(r.File) != 1 {
				t.Fatalf("archive holds %d files, want 1", len(r.File))
			}
			if name := r.File[0].Name; name != g.Args[0] {
				t.Errorf("filename is %q, want %q", name, g.Args[0])
			}
			rc, err := r.File[0].Open()
			if err != nil {
				t.Fatalf("open entry: %v", err)
			}
			defer func() { _ = rc.Close() }()
			got, err := io.ReadAll(rc)
			if err != nil {
				t.Fatalf("read entry: %v", err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("entry holds %d bytes, want %d", len(got), len(want))
			}
		})
	}
}

// TestZipRoundTrips checks every archive back through Unzip.
func TestZipRoundTrips(t *testing.T) {
	for _, g := range loadZipGoldens(t) {
		t.Run(g.Name, func(t *testing.T) {
			want := string(g.input(t))
			out, err := runOp(t, "Zip", want, g.Args...)
			if err != nil {
				t.Fatalf("Zip: %v", err)
			}
			files, err := runFileListOp(t, "Unzip", out, "", false)
			if err != nil {
				t.Fatalf("Unzip: %v", err)
			}
			if len(files) != 1 {
				t.Fatalf("unzipped %d files, want 1", len(files))
			}
			if files[0].Name != g.Args[0] {
				t.Errorf("filename is %q, want %q", files[0].Name, g.Args[0])
			}
			if string(files[0].Data) != want {
				t.Errorf("round trip changed the data (%d bytes in, %d out)",
					len(want), len(files[0].Data))
			}
		})
	}
}

// TestZipPassword covers a passworded archive. Its encryption header is filled
// with random bytes, so the same input never gives the same archive twice and
// there is nothing to pin — what is checked is that it reads back, both here
// and through a reader that was not written alongside it.
func TestZipPassword(t *testing.T) {
	const want = "The quick brown fox jumped over the slow dog"
	for _, method := range []string{"Deflate", "None (Store)"} {
		t.Run(method, func(t *testing.T) {
			out, err := runOp(t, "Zip", want,
				"a.txt", "", "secret", method, "MSDOS", "Dynamic Huffman Coding")
			if err != nil {
				t.Fatalf("Zip: %v", err)
			}

			files, err := runFileListOp(t, "Unzip", out, "secret", false)
			if err != nil {
				t.Fatalf("Unzip: %v", err)
			}
			if len(files) != 1 || string(files[0].Data) != want {
				t.Fatalf("round trip failed: %d files", len(files))
			}

			// The header flag says the entry is encrypted.
			if out[6]&1 != 1 {
				t.Errorf("the encrypted flag is not set: flags are %#02x%02x", out[7], out[6])
			}

			// Two archives of the same input must differ, because the encryption
			// header is random.
			again, err := runOp(t, "Zip", want,
				"a.txt", "", "secret", method, "MSDOS", "Dynamic Huffman Coding")
			if err != nil {
				t.Fatalf("Zip: %v", err)
			}
			if string(zipBlankTimes([]byte(out))) == string(zipBlankTimes([]byte(again))) {
				t.Error("two passworded archives came out identical, so the header is not random")
			}
		})
	}
}

// TestZipPasswordCheckByte covers the byte a reader uses to tell a wrong
// password from a right one. It is the top byte of the entry's checksum, and it
// is what lets other tools reject a bad password rather than hand back rubbish.
func TestZipPasswordCheckByte(t *testing.T) {
	out, err := runOp(t, "Zip", "hello",
		"a.txt", "", "secret", "None (Store)", "MSDOS", "Dynamic Huffman Coding")
	if err != nil {
		t.Fatalf("Zip: %v", err)
	}
	header, crc := zipEncryptionHeader(t, []byte(out), "secret")
	if header[11] != byte(crc>>24) {
		t.Errorf("check byte is %#02x, want %#02x", header[11], byte(crc>>24))
	}
}

// zipEncryptionHeader decrypts an archive's twelve-byte encryption header and
// returns it along with the entry's recorded checksum.
func zipEncryptionHeader(t *testing.T, archive []byte, password string) ([]byte, uint32) {
	t.Helper()
	if len(archive) < 30 {
		t.Fatal("archive is too short to hold a local header")
	}
	crc := uint32(archive[14]) | uint32(archive[15])<<8 |
		uint32(archive[16])<<16 | uint32(archive[17])<<24
	nameLen := int(archive[26]) | int(archive[27])<<8
	extraLen := int(archive[28]) | int(archive[29])<<8
	at := 30 + nameLen + extraLen
	if len(archive) < at+12 {
		t.Fatal("archive is too short to hold an encryption header")
	}

	keys := newZipCrypto(password)
	header := make([]byte, 12)
	for i := range header {
		header[i] = keys.decrypt(archive[at+i])
	}
	return header, crc
}

// TestUnzipWrongPassword covers being given the wrong password, which must be
// reported rather than handing back the rubbish it decrypts to.
func TestUnzipWrongPassword(t *testing.T) {
	out, err := runOp(t, "Zip", "The quick brown fox jumped over the slow dog",
		"a.txt", "", "secret", "Deflate", "MSDOS", "Dynamic Huffman Coding")
	if err != nil {
		t.Fatalf("Zip: %v", err)
	}
	if _, err := runFileListOp(t, "Unzip", out, "wrong", false); err == nil {
		t.Error("expected an error for the wrong password")
	}
	if _, err := runFileListOp(t, "Unzip", out, "", false); err == nil {
		t.Error("expected an error when no password is given")
	}
}

// TestUnzipErrors covers the input the reader refuses.
func TestUnzipErrors(t *testing.T) {
	for _, tc := range []struct{ name, input string }{
		{"empty", ""},
		{"not an archive", "this is not a zip file at all"},
		{"truncated", "PK\x03\x04\x14\x00\x00\x00\x08\x00"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := runFileListOp(t, "Unzip", tc.input, "", false); err == nil {
				t.Error("expected an error")
			}
		})
	}
}

// TestUnzipSeveralFiles covers an archive holding more than one entry, which
// cchef never writes but often has to read.
func TestUnzipSeveralFiles(t *testing.T) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	want := map[string]string{
		"first.txt":      "the first file",
		"dir/second.txt": "the second file",
		"dir/third.bin":  "\x00\x01\x02\xff",
	}
	for _, name := range []string{"first.txt", "dir/second.txt", "dir/third.bin"} {
		f, err := w.Create(name)
		if err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
		if _, err := f.Write([]byte(want[name])); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	files, err := runFileListOp(t, "Unzip", buf.String(), "", false)
	if err != nil {
		t.Fatalf("Unzip: %v", err)
	}
	if len(files) != len(want) {
		t.Fatalf("unzipped %d files, want %d", len(files), len(want))
	}
	for _, f := range files {
		if got, ok := want[f.Name]; !ok {
			t.Errorf("unexpected file %q", f.Name)
		} else if string(f.Data) != got {
			t.Errorf("%s holds %q, want %q", f.Name, f.Data, got)
		}
	}
}

// TestUnzipVerify covers the switch that checks each entry against the checksum
// recorded for it.
func TestUnzipVerify(t *testing.T) {
	out, err := runOp(t, "Zip", "The quick brown fox jumped over the slow dog",
		"a.txt", "", "", "None (Store)", "MSDOS", "Dynamic Huffman Coding")
	if err != nil {
		t.Fatalf("Zip: %v", err)
	}
	spoiled := []byte(out)
	// The stored data begins after the local header and the filename.
	spoiled[30+len("a.txt")] ^= 0xff

	if _, err := runFileListOp(t, "Unzip", string(spoiled), "", true); err == nil {
		t.Error("expected a checksum error with verification on")
	}
	if _, err := runFileListOp(t, "Unzip", string(spoiled), "", false); err != nil {
		t.Errorf("expected no error with verification off, got %v", err)
	}
}

// cyberChefPasswordedZip is an archive CyberChef produced, holding
// "The quick brown fox jumped over the slow dog" as a.txt under the password
// "secret". Its encryption header ends in a random byte where the format asks
// for the top byte of the checksum, which is why other tools reject it — and
// why cchef's reader identifies a password by the contents rather than by that
// byte.
const cyberChefPasswordedZip = "504b03041400010008008655fc5c509cbf8c3c0000002c00000005000000612e747874c3599fab4a9bb81cbf5529" +
	"cdbea578f9a47af869afaf436d76662e98692b17c4eada055800c62f90561d56a069d7b0d28cb8d01249dcf4bcfd" +
	"721cd2504b010214001400010008008655fc5c509cbf8c3c0000002c000000050000000000000000000000000000" +
	"000000612e747874504b05060000000001000100330000005f0000000000"

// TestUnzipReadsCyberChefArchive covers reading an archive CyberChef wrote. Its
// encryption header does not carry the check byte the format asks for, so a
// reader that insisted on one would turn it away.
func TestUnzipReadsCyberChefArchive(t *testing.T) {
	archive, err := hex.DecodeString(cyberChefPasswordedZip)
	if err != nil {
		t.Fatalf("bad hex: %v", err)
	}

	files, err := runFileListOp(t, "Unzip", string(archive), "secret", false)
	if err != nil {
		t.Fatalf("Unzip: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("unzipped %d files, want 1", len(files))
	}
	if files[0].Name != "a.txt" {
		t.Errorf("filename is %q, want %q", files[0].Name, "a.txt")
	}
	const want = "The quick brown fox jumped over the slow dog"
	if string(files[0].Data) != want {
		t.Errorf("holds %q, want %q", files[0].Data, want)
	}

	// A wrong password is still turned away, by the checksum rather than the byte.
	if _, err := runFileListOp(t, "Unzip", string(archive), "wrong", false); err == nil {
		t.Error("expected an error for the wrong password")
	}
}

// TestZipRejectsUnknownOptions covers the guards on the two option arguments.
// The argument definitions allow only the named settings, so these are reached
// by handing the operation arguments the command line could not.
func TestZipRejectsUnknownOptions(t *testing.T) {
	dish := core.NewDish([]byte("hello"), core.TypeByteArray)
	for _, tc := range []struct {
		name string
		args []any
	}{
		{"compression method", []any{"a.txt", "", "", "Brotli", "MSDOS", "Dynamic Huffman Coding"}},
		{"compression type", []any{"a.txt", "", "", "Deflate", "MSDOS", "Brotli"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := (Zip{}).Run(dish, tc.args); err == nil {
				t.Error("expected an error")
			}
		})
	}
}

// TestUnzipSkipsDirectories covers an archive carrying entries for the
// directories themselves, which most writers add and which hold no data.
func TestUnzipSkipsDirectories(t *testing.T) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	if _, err := w.Create("sub/"); err != nil {
		t.Fatalf("create directory entry: %v", err)
	}
	f, err := w.Create("sub/file.txt")
	if err != nil {
		t.Fatalf("create file entry: %v", err)
	}
	if _, err := f.Write([]byte("contents")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	files, err := runFileListOp(t, "Unzip", buf.String(), "", false)
	if err != nil {
		t.Fatalf("Unzip: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("unzipped %d entries, want just the one file", len(files))
	}
	if files[0].Name != "sub/file.txt" || string(files[0].Data) != "contents" {
		t.Errorf("got %q holding %q", files[0].Name, files[0].Data)
	}
}

// TestUnzipUnsupportedMethod covers an entry stored by a method neither cchef
// nor CyberChef can unpack.
func TestUnzipUnsupportedMethod(t *testing.T) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, err := w.CreateHeader(&zip.FileHeader{Name: "a.bin", Method: zip.Store})
	if err != nil {
		t.Fatalf("create entry: %v", err)
	}
	if _, err := f.Write([]byte("whatever")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// Method 12 is bzip2, which the format allows and this reader does not. The
	// standard library's writer will not produce one, so the field is changed
	// afterwards, in the local header and in the directory that repeats it.
	archive := buf.Bytes()
	archive[8] = 12
	central := bytes.Index(archive, []byte("PK\x01\x02"))
	if central < 0 {
		t.Fatal("no central directory")
	}
	archive[central+10] = 12

	if _, err := runFileListOp(t, "Unzip", string(archive), "", false); err == nil {
		t.Error("expected an error for an unsupported compression method")
	}
}

// TestUnzipCorruptEntry covers an entry whose compressed data does not decode.
func TestUnzipCorruptEntry(t *testing.T) {
	out, err := runOp(t, "Zip", "The quick brown fox jumped over the slow dog",
		"a.txt", "", "", "Deflate", "MSDOS", "Dynamic Huffman Coding")
	if err != nil {
		t.Fatalf("Zip: %v", err)
	}
	spoiled := []byte(out)
	// Wreck the compressed data, which starts after the header and filename.
	for i := 30 + len("a.txt"); i < 30+len("a.txt")+8 && i < len(spoiled); i++ {
		spoiled[i] = 0xff
	}
	if _, err := runFileListOp(t, "Unzip", string(spoiled), "", false); err == nil {
		t.Error("expected an error for an entry that does not decode")
	}
}

// TestUnzipShortEncryptedEntry covers an encrypted entry too short to hold even
// the header the cipher needs.
func TestUnzipShortEncryptedEntry(t *testing.T) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, err := w.CreateHeader(&zip.FileHeader{
		Name: "a.bin", Method: zip.Store, Flags: zipEncryptedFlag,
	})
	if err != nil {
		t.Fatalf("create entry: %v", err)
	}
	if _, err := f.Write([]byte("short")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	if _, err := runFileListOp(t, "Unzip", buf.String(), "secret", false); err == nil {
		t.Error("expected an error for an entry too short to be encrypted")
	}
}

// TestUnzipEntryWithoutLocalHeader covers an archive whose directory lists an
// entry that is not where it says it is. The directory is read first, so the
// archive opens; the entry only fails when it is reached.
func TestUnzipEntryWithoutLocalHeader(t *testing.T) {
	out, err := runOp(t, "Zip", "hello",
		"a.txt", "", "", "None (Store)", "MSDOS", "Dynamic Huffman Coding")
	if err != nil {
		t.Fatalf("Zip: %v", err)
	}
	spoiled := []byte(out)
	// Wreck the local header's signature, leaving the directory pointing at it.
	spoiled[0], spoiled[1] = 'X', 'Y'

	if _, err := runFileListOp(t, "Unzip", string(spoiled), "", false); err == nil {
		t.Error("expected an error for an entry with no local header")
	}
}
