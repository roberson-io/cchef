package ops

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// carvePrefix and carveSuffix pad a sample so the carver has to find where the
// file starts and where it ends rather than reading a whole buffer.
var (
	carvePrefix = append(bytes.Repeat([]byte{0x00}, 7), []byte("PREFIX--")...)
	carveSuffix = append([]byte("--SUFFIX"), bytes.Repeat([]byte{0xee}, 9)...)
)

// readCarveSample reads one of the sample files the carving tests are built on.
func readCarveSample(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "carve", name))
	if err != nil {
		t.Fatalf("reading the sample: %v", err)
	}
	return data
}

// carveOne runs the scanner over a buffer and carves the first file found whose
// extension matches, returning its name and bytes.
func carveOne(t *testing.T, buf []byte, ext string) (string, []byte) {
	t.Helper()
	for _, f := range scanForFileTypes(buf, nil) {
		if firstExtension(f.details.extension) != ext {
			continue
		}
		file, err := extractFile(buf, f.details, f.offset)
		if err != nil {
			t.Fatalf("carving the %s at offset %d: %v", ext, f.offset, err)
		}
		return file.Name, file.Data
	}
	t.Fatalf("no %s was found in the buffer", ext)
	return "", nil
}

// TestCarveRoundTrip covers the carvers against real files of each format: a
// sample padded either side must come back byte for byte, under the name
// CyberChef gives it — which is built from where the signature matched, not
// from where the file starts. The two differ only for Targa, whose signature is
// in its footer. Every case was checked against the CyberChef-server oracle
// carving the same buffer; the four it gets wrong are covered separately by
// TestCarveWhereCyberChefFails.
func TestCarveRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		sample, ext string
		// signatureFromEnd, when set, says how far back from the end of the
		// sample its signature sits.
		signatureFromEnd int
	}{
		{"sample.png", "png", 0},
		{"sample.jpg", "jpg", 0},
		{"sample.bmp", "bmp", 0},
		{"sample.ico", "ico", 0},
		{"sample.webp", "webp", 0},
		{"sample.gif", "gif", 0},
		{"sample.tga", "tga", 18},
		{"sample.zip", "zip", 0},
		{"sample.tar", "tar", 0},
		{"sample.gz", "gz", 0},
		{"sample.bz2", "bz2", 0},
		{"sample.xz", "xz", 0},
		{"sample.zlib", "zlib", 0},
		{"sample.pdf", "pdf", 0},
		{"sample.rtf", "rtf", 0},
		{"sample.plist", "plist", 0},
		{"sample.sqlite", "sqlite", 0},
		{"sample.wav", "wav", 0},
		{"sample.flv", "flv", 0},
		{"sample.mp3", "mp3", 0},
		{"sample.elf", "elf", 0},
		{"sample.exe", "exe", 0},
		{"sample.macho", "dylib", 0},
		{"sample.deb", "deb", 0},
		{"sample.lzo", "lzop", 0},
		{"sample.evtx", "evtx", 0},
		{"sample.evt", "evt", 0},
		{"sample.dmp", "dmp", 0},
		{"sample.pf", "pf", 0},
		{"sample.lnk", "lnk", 0},
		{"sample.keychain", "keychain", 0},
	} {
		t.Run(tc.sample, func(t *testing.T) {
			want := readCarveSample(t, tc.sample)
			buf := append(append(append([]byte{}, carvePrefix...), want...), carveSuffix...)

			name, got := carveOne(t, buf, tc.ext)
			if !bytes.Equal(got, want) {
				t.Errorf("carved %d bytes, want %d", len(got), len(want))
			}
			signatureAt := len(carvePrefix)
			if tc.signatureFromEnd > 0 {
				signatureAt += len(want) - tc.signatureFromEnd
			}
			if wantName := fmt.Sprintf("extracted_at_0x%x.%s", signatureAt, tc.ext); name != wantName {
				t.Errorf("name = %q, want %q", name, wantName)
			}
		})
	}
}

// TestExtractFileWithoutCarver covers a type the scanner recognises but cannot
// cut out, which is most of the signature table.
func TestExtractFileWithoutCarver(t *testing.T) {
	sig := fileSig{name: "Test", extension: "xyz", mime: "application/x-test"}
	_, err := extractFile([]byte{1, 2, 3}, sig, 0)
	if err == nil {
		t.Fatal("a type with no carver was extracted anyway")
	}
	const want = `No extraction algorithm available for "application/x-test" files`
	if err.Error() != want {
		t.Errorf("got %q, want %q", err.Error(), want)
	}
}

// TestFirstExtension covers the name suffix, which is the first of the
// comma-separated extensions a signature lists.
func TestFirstExtension(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"jpg,jpeg,jpe,thm,mpo", "jpg"},
		{"png", "png"},
		{"", ""},
	} {
		if got := firstExtension(tc.in); got != tc.want {
			t.Errorf("firstExtension(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestCarversCoverTheTable covers that every carver the signature table names is
// implemented. The table is generated from CyberChef, so a new upstream
// extractor shows up here rather than being silently skipped.
func TestCarversCoverTheTable(t *testing.T) {
	named := map[string]bool{}
	for _, cat := range fileSignatures {
		for _, ft := range cat.types {
			if ft.carver == "" {
				continue
			}
			named[ft.carver] = true
			if _, ok := carvers[ft.carver]; !ok {
				t.Errorf("%s/%s names carver %q, which is not implemented", cat.name, ft.name, ft.carver)
			}
		}
	}
	for name := range carvers {
		if !named[name] {
			t.Errorf("carver %q is implemented but no signature names it", name)
		}
	}
}

// TestCarvePFWin10Declines covers the one format whose end cannot be found: from
// Windows 10 a prefetch file is compressed and records only the expanded size,
// so there is nothing to carve to. It has to say so rather than guess.
func TestCarvePFWin10Declines(t *testing.T) {
	want := readCarveSample(t, "sample.pf10")
	buf := append(append(append([]byte{}, carvePrefix...), want...), carveSuffix...)

	for _, f := range scanForFileTypes(buf, nil) {
		if f.details.carver != "PFWin10" {
			continue
		}
		_, err := extractFile(buf, f.details, f.offset)
		if err == nil {
			t.Fatal("a Windows 10 prefetch file was carved despite recording no length")
		}
		if !strings.Contains(err.Error(), "records no compressed length") {
			t.Errorf("got %q, which does not explain why", err.Error())
		}
		return
	}
	t.Fatal("the Windows 10 prefetch signature was not found")
}

// TestCarveAbortsCleanly covers a signature that matches but is followed by
// nothing usable: the carve must report a problem rather than run off the buffer.
func TestCarveAbortsCleanly(t *testing.T) {
	// A PNG signature with no chunks after it.
	buf := append(mustHex(t, "89504e470d0a1a0a"), bytes.Repeat([]byte{0x00}, 4)...)
	for _, f := range scanForFileTypes(buf, []string{"Images"}) {
		if f.details.carver != "PNG" {
			continue
		}
		if _, err := extractFile(buf, f.details, f.offset); err == nil {
			t.Error("a truncated PNG was carved without complaint")
		}
		return
	}
	t.Fatal("the PNG signature was not found")
}

// TestCarveStreamPastEnd covers an offset beyond the buffer, which leaves the
// carver an empty stream to fail on rather than an out-of-range slice.
func TestCarveStreamPastEnd(t *testing.T) {
	if got := carveStream([]byte{1, 2, 3}, 99); got.length() != 0 {
		t.Errorf("stream length = %d, want 0", got.length())
	}
}

// TestJPEGSizedSegment covers the marker classification across the whole range.
func TestJPEGSizedSegment(t *testing.T) {
	sized := map[byte]bool{0xdb: true, 0xde: true, 0xfe: true}
	for m := 0xc0; m <= 0xcf; m++ {
		sized[byte(m)] = true
	}
	for m := 0xe0; m <= 0xef; m++ {
		sized[byte(m)] = true
	}
	for m := range 256 {
		if got := jpegSizedSegment(byte(m)); got != sized[byte(m)] {
			t.Errorf("jpegSizedSegment(%#02x) = %v, want %v", m, got, sized[byte(m)])
		}
	}
}

// TestJPEGUnalignedMarker covers a buffer whose signature matches but whose
// following bytes are not a marker.
func TestJPEGUnalignedMarker(t *testing.T) {
	buf := append(mustHex(t, "ffd8ffe0"), bytes.Repeat([]byte{0x41}, 32)...)
	if err := catchStreamError(func() { carveJPEG(buf, 0) }); err == nil {
		t.Error("a JPEG with no end marker was carved without complaint")
	}
}

// TestTarOctal covers the size field, which is octal digits padded with spaces
// or nulls.
func TestTarOctal(t *testing.T) {
	for _, tc := range []struct {
		name  string
		field string
		want  int
	}{
		{"padded with nulls", "00000000144\x00", 100},
		{"padded with spaces", "        144 ", 100},
		{"all zeroes", "00000000000", 0},
		{"empty", "           ", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tarOctal([]byte(tc.field)); got != tc.want {
				t.Errorf("got %d, want %d", got, tc.want)
			}
		})
	}

	if err := catchStreamError(func() { tarOctal([]byte("not octal!!")) }); err == nil {
		t.Error("a size field that is not octal was accepted")
	}
}

// TestArDecimal covers the member size field of an ar archive.
func TestArDecimal(t *testing.T) {
	for _, tc := range []struct {
		field string
		want  int
	}{
		{"12        ", 12},
		{"0         ", 0},
		{"          ", -1},
		{"garbage   ", -1},
		{"-5        ", -1},
	} {
		if got := arDecimal([]byte(tc.field)); got != tc.want {
			t.Errorf("arDecimal(%q) = %d, want %d", tc.field, got, tc.want)
		}
	}
}

// TestCarveByDecompressingRejects covers a stream that does not decompress.
func TestCarveByDecompressingRejects(t *testing.T) {
	for _, tc := range []struct {
		name string
		buf  []byte
	}{
		{"a gzip header with no data", mustHex(t, "1f8b08000000000000ff")},
		{"a gzip header with corrupt data", append(mustHex(t, "1f8b08000000000000ff"), bytes.Repeat([]byte{0xff}, 32)...)},
		{"a zlib header with no data", mustHex(t, "789c")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			carve := carveGZIP
			if tc.buf[0] == 0x78 {
				carve = carveZlib
			}
			if err := catchStreamError(func() { carve(tc.buf, 0) }); err == nil {
				t.Error("a broken stream was carved without complaint")
			}
		})
	}
}

// TestCountingByteReader covers the reader that measures how much of a buffer a
// decompressor consumed, including both ways of reading from it.
func TestCountingByteReader(t *testing.T) {
	r := &countingByteReader{data: []byte{1, 2, 3}}
	if b, err := r.ReadByte(); b != 1 || err != nil {
		t.Errorf("ReadByte = %d, %v", b, err)
	}
	p := make([]byte, 8)
	if n, err := r.Read(p); n != 2 || err != nil {
		t.Errorf("Read = %d, %v; want 2, nil", n, err)
	}
	if _, err := r.ReadByte(); err == nil {
		t.Error("reading past the end gave no error")
	}
	if _, err := r.Read(p); err == nil {
		t.Error("reading past the end gave no error")
	}
}

// TestTargaFromExtensionArea covers a Targa file whose footer points at an
// extension area, which fixes where the file starts without any searching.
func TestTargaFromExtensionArea(t *testing.T) {
	image := readCarveSample(t, "sample.tga")
	body := image[:len(image)-targaFooterWidth]

	// An extension area of the size the format states, then a footer pointing at
	// it from the start of the file.
	area := make([]byte, targaExtensionSz)
	area[0] = byte(targaExtensionSz & 0xff)
	area[1] = byte(targaExtensionSz >> 8)

	extensionOffset := len(body)
	footer := make([]byte, targaFooterWidth)
	footer[0] = byte(extensionOffset & 0xff)
	footer[1] = byte(extensionOffset >> 8)
	footer[2] = byte(extensionOffset >> 16)
	footer[3] = byte(extensionOffset >> 24)
	copy(footer[8:], "TRUEVISION-XFILE.")

	want := append(append(append([]byte{}, body...), area...), footer...)
	buf := append(append(append([]byte{}, carvePrefix...), want...), carveSuffix...)

	_, got := carveOne(t, buf, "tga")
	if !bytes.Equal(got, want) {
		t.Errorf("carved %d bytes, want %d", len(got), len(want))
	}
}

// TestTargaFileSizeRejects covers the headers that do not describe an image
// whose size can be worked out.
func TestTargaFileSizeRejects(t *testing.T) {
	valid := func() []byte {
		h := make([]byte, targaHeaderWidth)
		h[2] = targaTrueColour
		h[12], h[13] = 16, 0 // width
		h[14], h[15] = 16, 0 // height
		h[16] = 24           // pixel depth
		return h
	}
	if _, ok := targaFileSize(valid(), 0); !ok {
		t.Fatal("a well-formed header was rejected")
	}

	for _, tc := range []struct {
		name   string
		damage func([]byte)
	}{
		{"a run-length encoded image", func(h []byte) { h[2] = 10 }},
		{"no width", func(h []byte) { h[12], h[13] = 0, 0 }},
		{"no height", func(h []byte) { h[14], h[15] = 0, 0 }},
		{"a pixel depth that is not whole bytes", func(h []byte) { h[16] = 15 }},
		{"an unknown colour map type", func(h []byte) { h[1] = 7 }},
		{"a colour map with no entry size", func(h []byte) { h[1], h[7] = 1, 0 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := valid()
			tc.damage(h)
			if _, ok := targaFileSize(h, 0); ok {
				t.Error("the header was accepted")
			}
		})
	}

	if _, ok := targaFileSize(make([]byte, 4), 0); ok {
		t.Error("a header shorter than the format allows was accepted")
	}
}

// TestTargaRejects covers buffers where the signature matched but no file can be
// read around it.
func TestTargaRejects(t *testing.T) {
	sig := []byte("TRUEVISION-XFILE.")

	// The signature too near the start for a footer to precede it.
	if err := catchStreamError(func() { carveTARGA(append([]byte{0x00}, sig...), 1) }); err == nil {
		t.Error("a signature with no room for a footer was accepted")
	}
	// A footer running past the end of the buffer.
	buf := append(bytes.Repeat([]byte{0x00}, 8), sig...)
	if err := catchStreamError(func() { carveTARGA(buf, 8) }); err == nil {
		t.Error("a truncated footer was accepted")
	}
	// A whole footer, but nothing before it that reads as an image.
	buf = append(bytes.Repeat([]byte{0xff}, 64), make([]byte, targaFooterWidth)...)
	copy(buf[len(buf)-targaFooterWidth+8:], sig)
	if err := catchStreamError(func() { carveTARGA(buf, len(buf)-targaFooterWidth+8) }); err == nil {
		t.Error("a footer with no image before it was accepted")
	}
}

// TestMP3FrameSizeRejects covers the bytes that do not open a frame.
func TestMP3FrameSizeRejects(t *testing.T) {
	for _, tc := range []struct {
		name  string
		bytes []byte
	}{
		{"too few bytes", []byte{0xff, 0xfb}},
		{"no sync word", []byte{0x00, 0x00, 0x00, 0x00}},
		{"a partial sync word", []byte{0xff, 0x1b, 0x90, 0x00}},
		{"a reserved version", []byte{0xff, 0xeb, 0x90, 0x00}},
		{"a reserved layer", []byte{0xff, 0xf9, 0x90, 0x00}},
		{"a free bit rate", []byte{0xff, 0xfb, 0x00, 0x00}},
		{"an invalid bit rate", []byte{0xff, 0xfb, 0xf0, 0x00}},
		{"a reserved sample rate", []byte{0xff, 0xfb, 0x9c, 0x00}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, ok := mp3FrameSize(newByteStream(tc.bytes)); ok {
				t.Error("the bytes were read as a frame header")
			}
		})
	}
}

// TestMP3Layers covers the frame sizes of the layers and versions, which are
// worked out differently for each.
func TestMP3Layers(t *testing.T) {
	for _, tc := range []struct {
		name   string
		header []byte
		want   int
	}{
		// MPEG-1 layer 3, 128 kbit/s at 44100 Hz.
		{"MPEG-1 layer 3", []byte{0xff, 0xfb, 0x90, 0x00}, 417},
		// MPEG-1 layer 1, 32 kbit/s at 44100 Hz.
		{"MPEG-1 layer 1", []byte{0xff, 0xfe, 0x10, 0x00}, 32},
		// MPEG-2 layer 3, 8 kbit/s at 22050 Hz.
		{"MPEG-2 layer 3", []byte{0xff, 0xf3, 0x10, 0x00}, 26},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := mp3FrameSize(newByteStream(tc.header))
			if !ok {
				t.Fatal("the header was not read as a frame")
			}
			if got != tc.want {
				t.Errorf("frame size = %d, want %d", got, tc.want)
			}
		})
	}
}

// TestByteStreamPeekAndLookingAt covers the two reads that do not move the
// stream, at both ends of the buffer.
func TestByteStreamPeekAndLookingAt(t *testing.T) {
	s := newByteStream([]byte{1, 2, 3})
	if got := s.peek(2); got != 3 {
		t.Errorf("peek(2) = %d, want 3", got)
	}
	if s.pos != 0 {
		t.Errorf("peek moved the position to %d", s.pos)
	}
	for _, at := range []int{-1, 3} {
		if err := catchStreamError(func() { s.peek(at) }); err == nil {
			t.Errorf("peek(%d) was allowed", at)
		}
	}

	if !s.lookingAt([]byte{1, 2}) {
		t.Error("lookingAt did not see the bytes under the position")
	}
	if s.lookingAt([]byte{1, 2, 3, 4}) {
		t.Error("lookingAt saw a sequence longer than the buffer")
	}
	s.moveTo(3)
	if s.lookingAt([]byte{1}) {
		t.Error("lookingAt saw bytes at the end of the buffer")
	}
}

// TestByteStreamAdvance covers the unchecked move: past the end is allowed,
// which is how a carver's loop finishes, but before the start is not.
func TestByteStreamAdvance(t *testing.T) {
	s := newByteStream([]byte{1, 2, 3})
	s.advance(10)
	if s.pos != 10 {
		t.Errorf("position = %d, want 10", s.pos)
	}
	if s.hasMore() {
		t.Error("a stream past its end reports more to read")
	}
	back := newByteStream([]byte{1, 2, 3})
	if err := catchStreamError(func() { back.advance(-1) }); err == nil {
		t.Error("advancing to before the start was allowed")
	}
}

// TestCatchStreamErrorPassesPlainErrors covers a panic carrying an error that is
// neither of the two a carve raises; it must not be mistaken for one.
func TestCatchStreamErrorPassesPlainErrors(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("an unrelated error panic was swallowed")
		}
	}()
	_ = catchStreamError(func() { panic(errors.New("something else")) })
}

// TestCarveRTFEscapes covers a document whose braces are escaped, which must not
// be counted as opening or closing anything.
func TestCarveRTFEscapes(t *testing.T) {
	doc := []byte(`{\rtf1 a \{ b \} c \\ d {\nested }}`)
	buf := append(append(append([]byte{}, carvePrefix...), doc...), carveSuffix...)

	_, got := carveOne(t, buf, "rtf")
	if !bytes.Equal(got, doc) {
		t.Errorf("carved %q, want %q", got, doc)
	}
}

// TestCarvePListNested covers a property list holding another <plist> tag,
// which the tag counting has to pair up rather than stopping at the first close.
func TestCarvePListNested(t *testing.T) {
	// The signature is the DOCTYPE at offset 39, so the declaration before it
	// has to be the width the format is written with.
	prologue := readCarveSample(t, "sample.plist")[:39]
	doc := append(append([]byte{}, prologue...),
		[]byte(`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">`+
			`<plist version="1.0"><data><plist></plist></data></plist>`)...)

	buf := append(append(append([]byte{}, carvePrefix...), doc...), carveSuffix...)

	_, got := carveOne(t, buf, "plist")
	if !bytes.Equal(got, doc) {
		t.Errorf("carved %d bytes, want %d", len(got), len(doc))
	}
}

// TestCarveELF32Bit covers the other word size, whose header is narrower and
// whose offsets are four bytes rather than eight.
func TestCarveELF32Bit(t *testing.T) {
	const shentsize, shnum = 40, 2
	body := bytes.Repeat([]byte{0x90}, 32)
	shoff := 52 + len(body)

	h := make([]byte, 52)
	copy(h, []byte{0x7f, 'E', 'L', 'F', 1, 1, 1}) // 32-bit, little endian
	put16 := func(at, v int) { h[at] = byte(v); h[at+1] = byte(v >> 8) }
	put32 := func(at, v int) {
		for i := range 4 {
			h[at+i] = byte(v >> (8 * i))
		}
	}
	put16(16, 2)     // type
	put16(18, 3)     // machine
	put32(20, 1)     // version
	put32(32, shoff) // section header offset
	put16(46, shentsize)
	put16(48, shnum)

	want := append(append(append([]byte{}, h...), body...), bytes.Repeat([]byte{0}, shentsize*shnum)...)
	buf := append(append(append([]byte{}, carvePrefix...), want...), carveSuffix...)

	_, got := carveOne(t, buf, "elf")
	if !bytes.Equal(got, want) {
		t.Errorf("carved %d bytes, want %d", len(got), len(want))
	}
}

// TestCarveMZPECertificateTable covers a portable executable that declares an
// attribute certificate table. The table lives in the appended data that no
// section header covers, so where one is declared its end is the end of the
// file.
func TestCarveMZPECertificateTable(t *testing.T) {
	const peAt = 0x80
	put32 := func(b []byte, at, v int) {
		for i := range 4 {
			b[at+i] = byte(v >> (8 * i))
		}
	}

	dos := make([]byte, peAt)
	copy(dos, "MZ")
	dos[0x3c] = peAt

	coff := make([]byte, 24)
	copy(coff, "PE\x00\x00")
	coff[4], coff[5] = 0x64, 0x86 // machine
	coff[6] = 1                   // one section
	coff[20] = 240                // size of the optional header

	opt := make([]byte, 240)
	opt[0], opt[1] = 0x0b, 0x02 // PE32+

	section := make([]byte, 40)
	base := len(dos) + len(coff) + len(opt) + len(section)

	// Appended data holding a 16-byte certificate table at its end.
	const overlay = 64
	put32(opt, 112+32, base+overlay-16) // certificate table address
	put32(opt, 112+36, 16)              // and its size

	want := append(append(append(append([]byte{}, dos...), coff...), opt...), section...)
	want = append(want, bytes.Repeat([]byte{0xcc}, overlay)...)

	buf := append(append(append([]byte{}, carvePrefix...), want...), carveSuffix...)
	_, got := carveOne(t, buf, "exe")
	if !bytes.Equal(got, want) {
		t.Errorf("carved %d bytes, want %d", len(got), len(want))
	}
}

// TestCarveJPEGFixedSegments covers the markers that carry no length or a fixed
// one, and the restart markers the scan runs through.
func TestCarveJPEGFixedSegments(t *testing.T) {
	want := []byte{
		0xff, 0xd8, // start of image, no length
		0xff, 0xfe, 0x00, 0x03, 0x41, // a comment, whose length covers itself
		0xff, 0x01, // temporary use, no length
		0xff, 0xdf, 0x00, // expand reference, one byte
		0xff, 0xdc, 0x00, 0x00, // define number of lines, two bytes
		0xff, 0xdd, 0x00, 0x00, // define restart interval, two bytes
		0xff, 0xd0, 0x11, 0x22, // a restart marker, then entropy data
		0xff, 0xd9, // end of image
	}
	buf := append(append(append([]byte{}, carvePrefix...), want...), carveSuffix...)

	_, got := carveOne(t, buf, "jpg")
	if !bytes.Equal(got, want) {
		t.Errorf("carved %d bytes (% x), want %d", len(got), got, len(want))
	}
}

// TestCarveGIFRejects covers a GIF whose blocks do not begin with any of the
// bytes that introduce one.
func TestCarveGIFRejects(t *testing.T) {
	buf := append([]byte("GIF89a"), bytes.Repeat([]byte{0x41}, 32)...)
	if err := catchStreamError(func() { carveGIF(buf, 0) }); err == nil {
		t.Error("a GIF with no recognisable blocks was carved without complaint")
	}
}

// TestCarveByDecompressingPastEnd covers an offset beyond the buffer.
func TestCarveByDecompressingPastEnd(t *testing.T) {
	if err := catchStreamError(func() { carveGZIP([]byte{1, 2, 3}, 99) }); err == nil {
		t.Error("an offset past the buffer was carved without complaint")
	}
}

// TestCarveDEBOddMember covers a member whose contents are an odd number of
// bytes: the next header is padded to an even position.
func TestCarveDEBOddMember(t *testing.T) {
	member := func(name, contents string) []byte {
		h := bytes.Repeat([]byte{' '}, arHeaderWidth)
		copy(h, name)
		copy(h[arSizeOffset:], fmt.Sprint(len(contents)))
		copy(h[arMagicOffset:], arHeaderMagic)
		out := append(append([]byte{}, h...), contents...)
		if len(contents)%2 == 1 {
			out = append(out, '\n')
		}
		return out
	}
	want := append([]byte("!<arch>\n"), member("debian-binary/", "2.0\n")...)
	want = append(want, member("odd/", "abc")...)
	want = append(want, member("last/", "final")...)

	// Bytes after the archive that are not a member header must not be taken in.
	buf := append(append(append([]byte{}, carvePrefix...), want...), carveSuffix...)

	_, got := carveOne(t, buf, "deb")
	if !bytes.Equal(got, want) {
		t.Errorf("carved %d bytes, want %d", len(got), len(want))
	}
}

// TestCarveDEBStopsAtBadSize covers a member header whose size field is not a
// number, which ends the walk rather than being read as one.
func TestCarveDEBStopsAtBadSize(t *testing.T) {
	h := bytes.Repeat([]byte{' '}, arHeaderWidth)
	copy(h, "broken/")
	copy(h[arSizeOffset:], "not-a-size")
	copy(h[arMagicOffset:], arHeaderMagic)

	want := append([]byte("!<arch>\n"), h...)
	got := carveDEB(append(want, []byte("trailing")...), 0)
	if len(got) != 8 {
		t.Errorf("carved %d bytes, want just the 8-byte archive header", len(got))
	}
}

// TestCarveFLVStopsAtBrokenChain covers a tag header whose "size of the previous
// tag" field does not match the tag before it. That says the earlier tag was not
// really part of the file, so the walk backs out over both it and the header
// just read rather than keeping either.
func TestCarveFLVStopsAtBrokenChain(t *testing.T) {
	be := func(v, width int) []byte {
		b := make([]byte, width)
		for i := range width {
			b[width-1-i] = byte(v >> (8 * i))
		}
		return b
	}
	header := append([]byte("FLV\x01\x05"), be(9, 4)...)

	// One tag: previous size 0, type 8, body of 4 bytes.
	tag := append(be(0, 4), 8)
	tag = append(tag, be(4, 3)...)
	tag = append(tag, bytes.Repeat([]byte{0}, 7+4)...)

	// Then a header claiming a previous tag size that does not match.
	bad := append(be(999, 4), 8)
	bad = append(bad, be(4, 3)...)
	bad = append(bad, bytes.Repeat([]byte{0}, 7+4)...)

	buf := append(append(append([]byte{}, header...), tag...), bad...)
	got := carveFLV(buf, 0)

	if len(got) >= len(header)+len(tag) {
		t.Errorf("carved %d bytes, keeping the tag the next header contradicted", len(got))
	}
	if len(got) < len(header) {
		t.Errorf("carved %d bytes, dropping part of the file header", len(got))
	}
}

// TestTargaExtensionAreaRejected covers the extension-area offset not leading
// anywhere: the area must open with the size the format states.
func TestTargaExtensionAreaRejected(t *testing.T) {
	data := make([]byte, 1024)
	footerAt := 900

	// An offset is declared, but there is no extension area where it points.
	if _, ok := targaStartFromExtension(data, footerAt, 16); ok {
		t.Error("an extension area that is not there was accepted")
	}
	// The area is right, but the offset puts the start of the file before the
	// beginning of the buffer.
	areaAt := footerAt - targaExtensionSz
	data[areaAt] = byte(targaExtensionSz & 0xff)
	data[areaAt+1] = byte(targaExtensionSz >> 8)
	if _, ok := targaStartFromExtension(data, footerAt, areaAt+1); ok {
		t.Error("an offset reaching before the buffer was accepted")
	}
	// No offset declared at all.
	if _, ok := targaStartFromExtension(data, footerAt, 0); ok {
		t.Error("an absent extension area was accepted")
	}
	// A footer too near the start for an area to precede it.
	if _, ok := targaStartFromExtension(data, 4, 1); ok {
		t.Error("an area reaching before the buffer was accepted")
	}
}

// TestCarveMP3WithTags covers a file opening with an ID3 tag and closing with
// the fixed-size one, both of which belong to the file.
func TestCarveMP3WithTags(t *testing.T) {
	// An ID3v2 tag whose length is written as four seven-bit groups.
	const tagBody = 32
	id3 := append([]byte("ID3\x04\x00\x00"), 0, 0, 0, byte(tagBody))
	id3 = append(id3, bytes.Repeat([]byte{0}, tagBody)...)

	// One MPEG-1 layer 3 frame at 128 kbit/s and 44100 Hz, which is 417 bytes.
	frame := append([]byte{0xff, 0xfb, 0x90, 0x00}, bytes.Repeat([]byte{0x00}, 413)...)

	v1 := append([]byte("TAG"), bytes.Repeat([]byte{0x20}, mp3TagV1Size-3)...)

	want := append(append(append([]byte{}, id3...), frame...), v1...)
	buf := append(append(append([]byte{}, carvePrefix...), want...), carveSuffix...)

	_, got := carveOne(t, buf, "mp3")
	if !bytes.Equal(got, want) {
		t.Errorf("carved %d bytes, want %d", len(got), len(want))
	}
}

// TestCarveMP3TruncatedFrame covers a final frame cut short by the end of the
// buffer, which is taken as far as it goes.
func TestCarveMP3TruncatedFrame(t *testing.T) {
	buf := append([]byte{0xff, 0xfb, 0x90, 0x00}, bytes.Repeat([]byte{0x00}, 16)...)
	if got := carveMP3(buf, 0); len(got) != len(buf) {
		t.Errorf("carved %d bytes, want the whole %d-byte buffer", len(got), len(buf))
	}
}

// TestCarveLZOPOptionalFields covers a header carrying the optional fields the
// flags can call for: a filter, an extra field, and a file name.
func TestCarveLZOPOptionalFields(t *testing.T) {
	be := func(v, width int) []byte {
		b := make([]byte, width)
		for i := range width {
			b[width-1-i] = byte(v >> (8 * i))
		}
		return b
	}
	// A version with the filter bit set, and flags asking for an extra field.
	want := append([]byte("\x89LZO\x00\x0d\x0a\x1a\x0a"), be(lzopVersion940|lzopHeaderFiler, 2)...)
	want = append(want, be(0x2050, 2)...)         // library version
	want = append(want, be(lzopVersion940, 2)...) // version needed
	want = append(want, 1, 1)                     // method and level
	want = append(want, be(lzopExtraField|lzopAdler32Data|lzopAdler32Comp, 4)...)
	want = append(want, be(0, 4)...)                   // the filter
	want = append(want, bytes.Repeat([]byte{0}, 8)...) // mode and time
	want = append(want, bytes.Repeat([]byte{0}, 4)...) // the later time field
	want = append(want, 4)                             // a four-character name
	want = append(want, []byte("name")...)
	want = append(want, be(6, 4)...) // the extra field's length
	want = append(want, []byte("extra")...)
	want = append(want, 0)
	want = append(want, be(0, 4)...) // the header checksum

	// One block whose compressed size differs from its uncompressed size, so
	// both sets of checksums follow it.
	body := []byte("compressed!!")
	want = append(want, be(32, 4)...)
	want = append(want, be(len(body), 4)...)
	want = append(want, body...)
	// One Adler-32 over the data and one over the compressed block.
	want = append(want, bytes.Repeat([]byte{0}, 2*4)...)
	want = append(want, be(0, 4)...) // the empty block that ends the file

	buf := append(append(append([]byte{}, carvePrefix...), want...), carveSuffix...)
	_, got := carveOne(t, buf, "lzop")
	if !bytes.Equal(got, want) {
		t.Errorf("carved %d bytes, want %d", len(got), len(want))
	}
}

// TestMachoSegmentKinds covers a load command list holding a 32-bit segment and
// one that is not a segment at all.
func TestMachoSegmentKinds(t *testing.T) {
	le := func(v, width int) []byte {
		b := make([]byte, width)
		for i := range width {
			b[i] = byte(v >> (8 * i))
		}
		return b
	}
	// A 32-bit header: magic, cpu, subtype, filetype, ncmds, sizeofcmds, flags.
	// The magic is written in file order; read big-endian it is the byte-swapped
	// 32-bit form, which says the rest of the file is little-endian.
	header := append([]byte{0xce, 0xfa, 0xed, 0xfe}, bytes.Repeat([]byte{0}, 12)...)
	header = append(header, le(2, 4)...) // two load commands
	header = append(header, bytes.Repeat([]byte{0}, 8)...)

	// A command that is not a segment, 16 bytes wide.
	other := append(le(0x2b, 4), le(16, 4)...)
	other = append(other, bytes.Repeat([]byte{0}, 8)...)

	// A 32-bit segment whose size sits at offset 36.
	segment := append(le(machoSegment, 4), le(56, 4)...)
	segment = append(segment, bytes.Repeat([]byte{0}, 28)...)
	segment = append(segment, le(200, 4)...)
	segment = append(segment, bytes.Repeat([]byte{0}, 16)...)

	buf := append(append(append([]byte{}, header...), other...), segment...)
	buf = append(buf, bytes.Repeat([]byte{0xaa}, 300)...)

	if got := carveMACHO(buf, 0); len(got) != 200 {
		t.Errorf("carved %d bytes, want 200 (the one segment's size)", len(got))
	}
}

// TestCarveEVTXChunks covers an event log holding more than one chunk.
func TestCarveEVTXChunks(t *testing.T) {
	le32 := func(v int) []byte {
		b := make([]byte, 4)
		for i := range 4 {
			b[i] = byte(v >> (8 * i))
		}
		return b
	}
	const firstChunk = 0x1000
	header := append([]byte("ElfFile\x00"), bytes.Repeat([]byte{0}, 0x28-8)...)
	header = append(header, le32(firstChunk)...)
	header = append(header, bytes.Repeat([]byte{0}, firstChunk-len(header))...)

	chunk := append([]byte("ElfChnk"), bytes.Repeat([]byte{0x11}, evtxChunkSize)...)

	want := append(append(append([]byte{}, header...), chunk...), chunk...)
	buf := append(append(append([]byte{}, carvePrefix...), want...), carveSuffix...)

	_, got := carveOne(t, buf, "evtx")
	if !bytes.Equal(got, want) {
		t.Errorf("carved %d bytes, want %d", len(got), len(want))
	}
}

// TestCarveRTFPlainText covers a document whose body is ordinary characters,
// which the brace counting steps over without acting on.
func TestCarveRTFPlainText(t *testing.T) {
	doc := []byte(`{\rtf1 plain words with no braces at all}`)
	buf := append(append(append([]byte{}, carvePrefix...), doc...), carveSuffix...)

	_, got := carveOne(t, buf, "rtf")
	if !bytes.Equal(got, doc) {
		t.Errorf("carved %q, want %q", got, doc)
	}
}

// TestCarveRTFRejects covers a buffer that does not open with a brace.
func TestCarveRTFRejects(t *testing.T) {
	if err := catchStreamError(func() { carveRTF([]byte("not rtf"), 0) }); err == nil {
		t.Error("a buffer with no opening brace was carved as RTF")
	}
}

// TestTargaFileSizeAtBufferEdge covers a candidate start so near the end of the
// buffer that no header fits.
func TestTargaFileSizeAtBufferEdge(t *testing.T) {
	if _, ok := targaFileSize(make([]byte, 20), 10); ok {
		t.Error("a header running past the end of the buffer was accepted")
	}
}

// TestCarveDEBTruncatedHeader covers an archive whose last header is cut short
// by the end of the buffer.
func TestCarveDEBTruncatedHeader(t *testing.T) {
	buf := append([]byte("!<arch>\n"), bytes.Repeat([]byte{' '}, 20)...)
	if got := carveDEB(buf, 0); len(got) != 8 {
		t.Errorf("carved %d bytes, want just the archive header", len(got))
	}
}

// TestCarveJPEGOddTail covers a buffer ending mid-marker: one byte is left, so
// the pair the walk needs cannot be read.
func TestCarveJPEGOddTail(t *testing.T) {
	buf := []byte{0xff, 0xd8, 0xff}
	if err := catchStreamError(func() { carveJPEG(buf, 0) }); err == nil {
		t.Error("a buffer ending mid-marker was carved without complaint")
	}
}

// TestCarveDEBStopsAtNonMember covers an archive followed by enough bytes for a
// member header that is not one.
func TestCarveDEBStopsAtNonMember(t *testing.T) {
	h := bytes.Repeat([]byte{' '}, arHeaderWidth)
	copy(h, "only/")
	copy(h[arSizeOffset:], "2")
	copy(h[arMagicOffset:], arHeaderMagic)

	want := append(append([]byte("!<arch>\n"), h...), []byte("ab")...)
	buf := append(append([]byte{}, want...), bytes.Repeat([]byte{0x41}, arHeaderWidth*2)...)

	if got := carveDEB(buf, 0); !bytes.Equal(got, want) {
		t.Errorf("carved %d bytes, want %d", len(got), len(want))
	}
}

// TestTargaColourMapped covers an image carrying a colour map, whose entries
// come between the header and the pixels.
func TestTargaColourMapped(t *testing.T) {
	const entries, entryDepth, width, height, pixelDepth = 4, 24, 8, 8, 8

	h := make([]byte, targaHeaderWidth)
	h[1] = 1 // a colour map is present
	h[2] = targaColourMapped
	h[5], h[6] = entries, 0
	h[7] = entryDepth
	h[12], h[13] = width, 0
	h[14], h[15] = height, 0
	h[16] = pixelDepth

	size, ok := targaFileSize(h, 0)
	if !ok {
		t.Fatal("a colour-mapped header was rejected")
	}
	want := targaHeaderWidth + entries*(entryDepth/8) + width*height*(pixelDepth/8)
	if size != want {
		t.Errorf("size = %d, want %d", size, want)
	}
}

// TestCarveGZIPCorruptBody covers a well-formed header followed by data that is
// not a DEFLATE stream, which fails while being read rather than when opened.
func TestCarveGZIPCorruptBody(t *testing.T) {
	// A gzip header, then a DEFLATE block claiming the reserved block type.
	buf := append(mustHex(t, "1f8b08000000000000ff"), 0x07, 0x00, 0x00, 0x00, 0x00)
	if err := catchStreamError(func() { carveGZIP(buf, 0) }); err == nil {
		t.Error("a corrupt DEFLATE stream was carved without complaint")
	}
}

// TestCarveGZIPBadChecksum covers a stream that decompresses but whose trailer
// does not agree with what came out, which is only found on closing it.
func TestCarveGZIPBadChecksum(t *testing.T) {
	var good []byte
	func() {
		defer func() { _ = recover() }()
		good = readCarveSample(t, "sample.gz")
	}()
	if len(good) < 9 {
		t.Skip("no gzip sample to damage")
	}
	damaged := append([]byte{}, good...)
	// The last eight bytes are the checksum and length; flip a bit in them.
	damaged[len(damaged)-5] ^= 0xff
	if err := catchStreamError(func() { carveGZIP(damaged, 0) }); err == nil {
		t.Error("a stream whose trailer disagrees was carved without complaint")
	}
}
