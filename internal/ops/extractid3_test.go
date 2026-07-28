package ops

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// readID3Sample reads one of the tagged MP3 files the tests are built on.
func readID3Sample(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "id3", name))
	if err != nil {
		t.Fatalf("reading the sample: %v", err)
	}
	return data
}

// runExtractID3 runs the operation over a tagged file.
func runExtractID3(t *testing.T, data []byte) (string, error) {
	t.Helper()
	op, ok := core.Default.Get("Extract ID3")
	if !ok {
		t.Fatal("Extract ID3 is not registered")
	}
	out, err := op.Run(core.NewDish(data, core.TypeArrayBuffer), nil)
	if err != nil {
		return "", err
	}
	return out.String(), nil
}

// TestExtractID3Fixtures covers tags CyberChef also reads correctly. The
// expected output is the oracle's, byte for byte.
func TestExtractID3Fixtures(t *testing.T) {
	for _, tc := range []struct{ name, sample, want string }{
		{
			"an ID3v2.4 tag",
			"v24_small.mp3",
			`{"Type":"ID3","Version":"4.0","Flags":"0","Size":"130","Tags":{` +
				`"TIT2":{"Size":"12","Description":"Title/songname/content description","Data":"Test Title\u0000"},` +
				`"TPE1":{"Size":"13","Description":"Lead performer(s)/Soloist(s)","Data":"Test Artist\u0000"},` +
				`"TALB":{"Size":"12","Description":"Album/Movie/Show title","Data":"Test Album\u0000"},` +
				`"TDRC":{"Size":"6","Description":"Recording time","Data":"2026\u0000"},` +
				`"TRCK":{"Size":"3","Description":"Track number/Position in set","Data":"3\u0000"},` +
				`"TSSE":{"Size":"14","Description":"Software/Hardware and settings used for encoding","Data":"Lavf61.1.100\u0000"}}}`,
		},
		{
			"an ID3v2.3 tag whose frames are all short",
			"v23_small.mp3",
			`{"Type":"ID3","Version":"3.0","Flags":"0","Size":"66","Tags":{` +
				`"TIT2":{"Size":"7","Description":"Title/songname/content description","Data":"Small\u0000"},` +
				`"TPE1":{"Size":"5","Description":"Lead performer(s)/Soloist(s)","Data":"Ann\u0000"},` +
				`"TSSE":{"Size":"14","Description":"Software/Hardware and settings used for encoding","Data":"Lavf61.1.100\u0000"}}}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := runExtractID3(t, readID3Sample(t, tc.sample))
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if got != tc.want {
				t.Errorf("got  %s\nwant %s", got, tc.want)
			}
		})
	}
}

// TestExtractID3Corrected covers tags CyberChef cannot read. Its size fields are
// read as though every one were a seven-bit "syncsafe" number, which is only
// true of the tag length and, in ID3v2.4, the frame lengths. The values below
// are what the sizes in the file actually say.
func TestExtractID3Corrected(t *testing.T) {
	for _, tc := range []struct{ name, sample, want string }{
		{
			// The frame is 302 bytes, written as 00 00 01 2e. CyberChef reads
			// that as seven-bit groups, giving 0x01<<7 | 0x2e = 174, lands
			// inside the title, and reports "Unknown Frame Identifier: AAAA".
			"an ID3v2.3 frame longer than 127 bytes",
			"v23_long_frame.mp3",
			`{"Type":"ID3","Version":"3.0","Flags":"0","Size":"361","Tags":{` +
				`"TIT2":{"Size":"302","Description":"Title/songname/content description",` +
				`"Data":"` + repeatString("A", 300) + `\u0000"},` +
				`"TPE1":{"Size":"5","Description":"Lead performer(s)/Soloist(s)","Data":"Bob\u0000"},` +
				`"TSSE":{"Size":"14","Description":"Software/Hardware and settings used for encoding","Data":"Lavf61.1.100\u0000"}}}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := runExtractID3(t, readID3Sample(t, tc.sample))
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if got != tc.want {
				t.Errorf("got  %s\nwant %s", got, tc.want)
			}
		})
	}
}

// TestExtractID3LargeTag covers a tag over 16 KB, where the length needs more
// than fourteen bits. CyberChef shifts each byte of the length by a decreasing
// amount rather than by seven, so anything that large is read as a number about
// a hundred times too big and the walk runs past the end of the tag.
func TestExtractID3LargeTag(t *testing.T) {
	got, err := runExtractID3(t, readID3Sample(t, "v24_large_tag.mp3"))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	// The title alone is 20,000 characters, so the tag is well over the
	// fourteen bits CyberChef manages.
	const wantSize = `"Size":"20061"`
	if !containsString(got, wantSize) {
		t.Errorf("the tag length was not read as 20061:\n%.200s", got)
	}
	if !containsString(got, `"TIT2"`) {
		t.Error("the long title frame was not read")
	}
}

// TestExtractID3Rejects covers input that is not a tagged file.
func TestExtractID3Rejects(t *testing.T) {
	for _, tc := range []struct {
		name string
		data []byte
	}{
		{"nothing at all", nil},
		{"too short to hold a header", []byte("ID")},
		{"some other file", []byte("Not an MP3 at all, just text.")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := runExtractID3(t, tc.data)
			if err == nil {
				t.Fatal("input with no tag was read as one")
			}
			if err.Error() != "No valid ID3 header." {
				t.Errorf("got %q, want %q", err.Error(), "No valid ID3 header.")
			}
		})
	}
}

// TestExtractID3UnknownFrame covers a tag naming a frame the table does not
// list, which is reported rather than skipped.
func TestExtractID3UnknownFrame(t *testing.T) {
	// A v2.4 header declaring a 20-byte tag, then a frame called "ZZZZ".
	tag := []byte{'I', 'D', '3', 4, 0, 0, 0, 0, 0, 20}
	tag = append(tag, 'Z', 'Z', 'Z', 'Z', 0, 0, 0, 1, 0, 0, 0)

	_, err := runExtractID3(t, tag)
	if err == nil {
		t.Fatal("an unknown frame identifier was accepted")
	}
	if err.Error() != "Unknown Frame Identifier: ZZZZ" {
		t.Errorf("got %q, want %q", err.Error(), "Unknown Frame Identifier: ZZZZ")
	}
}

// TestExtractID3EndOfTag covers the padding that can follow the last frame: a
// run of zero bytes where an identifier would be ends the tag.
func TestExtractID3EndOfTag(t *testing.T) {
	// A v2.4 header declaring a 30-byte tag holding one frame and then padding.
	tag := []byte{'I', 'D', '3', 4, 0, 0, 0, 0, 0, 30}
	tag = append(tag, 'T', 'I', 'T', '2', 0, 0, 0, 5, 0, 0)
	tag = append(tag, 0, 'H', 'i', '!', 0)
	tag = append(tag, make([]byte, 15)...)

	got, err := runExtractID3(t, tag)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	const want = `{"Type":"ID3","Version":"4.0","Flags":"0","Size":"30","Tags":{` +
		`"TIT2":{"Size":"5","Description":"Title/songname/content description","Data":"Hi!\u0000"}}}`
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

// TestExtractID3v22 covers the older tag layout, whose identifiers are three
// characters and whose frame headers are six bytes rather than ten.
func TestExtractID3v22(t *testing.T) {
	// A v2.2 header declaring a 24-byte tag holding two frames.
	tag := []byte{'I', 'D', '3', 2, 0, 0, 0, 0, 0, 24}
	tag = append(tag, 'T', 'T', '2', 0, 0, 6)
	tag = append(tag, 0, 'S', 'o', 'n', 'g', 0)
	tag = append(tag, 'T', 'P', '1', 0, 0, 6)
	tag = append(tag, 0, 'A', 'n', 'n', 'e', 0)

	got, err := runExtractID3(t, tag)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	const want = `{"Type":"ID3","Version":"2.0","Flags":"0","Size":"24","Tags":{` +
		`"TT2":{"Size":"6","Description":"Title/Songname/Content description","Data":"Song\u0000"},` +
		`"TP1":{"Size":"6","Description":"Lead artist(s)/Lead performer(s)/Soloist(s)/Performing group","Data":"Anne\u0000"}}}`
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

// repeatString returns s repeated n times.
func repeatString(s string, n int) string {
	out := make([]byte, 0, len(s)*n)
	for range n {
		out = append(out, s...)
	}
	return string(out)
}

// containsString reports whether needle appears in haystack.
func containsString(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

// TestExtractID3TakesNoArguments covers the operation's argument list, which is
// empty: a tag is read the same way whatever the caller wants.
func TestExtractID3TakesNoArguments(t *testing.T) {
	op, ok := core.Default.Get("Extract ID3")
	if !ok {
		t.Fatal("Extract ID3 is not registered")
	}
	if args := op.Args(); len(args) != 0 {
		t.Errorf("declares %d arguments, want none", len(args))
	}
}

// TestExtractID3TruncatedFrame covers a frame whose declared length runs past
// the end of the file, which is read as far as the data goes.
func TestExtractID3TruncatedFrame(t *testing.T) {
	// A v2.4 header declaring a 40-byte tag holding a frame that claims 20 bytes
	// of data, of which only 6 are present.
	tag := []byte{'I', 'D', '3', 4, 0, 0, 0, 0, 0, 40}
	tag = append(tag, 'T', 'I', 'T', '2', 0, 0, 0, 20, 0, 0)
	tag = append(tag, 0, 'C', 'u', 't')

	got, err := runExtractID3(t, tag)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	const want = `{"Type":"ID3","Version":"4.0","Flags":"0","Size":"40","Tags":{` +
		`"TIT2":{"Size":"20","Description":"Title/songname/content description","Data":"Cut"}}}`
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

// TestExtractID3v22FrameSizeWidth covers the three-byte length an ID3v2.2 frame
// header carries, which is narrower than the field the later versions use.
func TestExtractID3v22FrameSizeWidth(t *testing.T) {
	// A frame of 0x010000 bytes would overflow a two-byte length; the three
	// bytes have to be read as one number.
	if got := id3FrameSize([]byte{0x01, 0x00, 0x00}, id3v22, id3v22IDWidth); got != 0x010000 {
		t.Errorf("got %d, want %d", got, 0x010000)
	}
	// A short field is read as far as it goes rather than past the end.
	if got := id3FrameSize([]byte{0x02}, id3v22, id3v22IDWidth); got != 2 {
		t.Errorf("got %d, want 2", got)
	}
}
