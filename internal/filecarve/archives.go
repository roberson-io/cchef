package filecarve

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"io"
	"math"
	"strconv"
	"strings"
)

// carveZIP ends the file after the end-of-central-directory record and whatever
// comment follows it.
func carveZIP(data []byte, offset int) []byte {
	s := carveStream(data, offset)

	s.ContinueUntil(zipEndOfCentralDirectory)

	// Over the record to the comment length, then over the comment.
	s.MoveForwardsBy(zipEOCDCommentLength)
	comment := s.ReadIntLE(2)
	s.MoveForwardsBy(comment)
	return s.Carve(0, s.Pos)
}

var zipEndOfCentralDirectory = []byte{0x50, 0x4b, 0x05, 0x06}

// zipEOCDCommentLength is how far into the end-of-central-directory record the
// two-byte comment length sits.
const zipEOCDCommentLength = 20

// The layout of a tar header, in bytes: where the format identifier and the file
// size sit, and the block size everything is rounded up to.
const (
	TarMagicOffset = 0x101
	TarSizeOffset  = 0x7c
	TarSizeWidth   = 11
	tarHeaderWidth = 512
	TarBlockSize   = 512
)

// carveTAR walks the member headers until one is not a tar header, then takes
// the run of padding zeroes that ends the archive.
func carveTAR(data []byte, offset int) []byte {
	s := carveStream(data, offset)

	for s.HasMore() {
		s.MoveForwardsBy(TarMagicOffset)
		if !bytes.Equal(s.GetBytes(5), []byte("ustar")) {
			// Back to the end of the last member.
			s.MoveBackwardsBy(TarMagicOffset + 5)
			break
		}

		// Back to the size field, which is written as octal digits.
		s.MoveBackwardsBy(TarMagicOffset + 5 - TarSizeOffset)
		size := tarOctal(s.GetBytes(TarSizeWidth))

		// Members are padded out to a whole number of blocks.
		padded := int(math.Ceil(float64(size)/TarBlockSize)) * TarBlockSize
		s.MoveForwardsBy(padded + tarHeaderWidth - TarSizeOffset - TarSizeWidth)
	}
	s.ConsumeWhile(0x00)
	return s.Carve(0, s.Pos)
}

// tarOctal reads a tar size field, which holds octal digits and may be padded
// with spaces or nulls.
func tarOctal(field []byte) int {
	digits := strings.TrimRight(strings.TrimSpace(string(field)), "\x00")
	digits = strings.TrimSpace(digits)
	if digits == "" {
		return 0
	}
	size, err := strconv.ParseInt(digits, 8, 64)
	if err != nil {
		panic(carveFailure{msg: "Invalid file size while parsing TAR"})
	}
	return int(size)
}

// countingByteReader reads a buffer one byte at a time and remembers how far it
// got. compress/flate and its callers take the byte-at-a-time path when the
// reader offers one, so the count is exactly the number of bytes the compressed
// stream occupies — which is what a carve needs and what CyberChef works out by
// stepping through the DEFLATE blocks itself.
type countingByteReader struct {
	data []byte
	pos  int
}

func (r *countingByteReader) ReadByte() (byte, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	b := r.data[r.pos]
	r.pos++
	return b, nil
}

func (r *countingByteReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

// carveGZIP ends the file after the member's footer.
//
// CyberChef walks the DEFLATE blocks itself to find where the compressed data
// stops (src/core/lib/FileSignatures.mjs). Reading the member with the standard
// decompressor and noting how many bytes it consumed answers the same question,
// and does not share that walk's inability to handle a header carrying an extra
// field.
func carveGZIP(data []byte, offset int) []byte {
	return carveByDecompressing(data, offset, func(r io.Reader) (io.ReadCloser, error) {
		zr, err := gzip.NewReader(r)
		if err != nil {
			return nil, err
		}
		// A carved file is one member; anything after it is the next file.
		zr.Multistream(false)
		return zr, nil
	})
}

// carveZlib ends the file after the stream's checksum.
func carveZlib(data []byte, offset int) []byte {
	return carveByDecompressing(data, offset, zlib.NewReader)
}

// carveByDecompressing reads one compressed stream starting at offset and
// returns the bytes it occupied.
func carveByDecompressing(data []byte, offset int, open func(io.Reader) (io.ReadCloser, error)) []byte {
	if offset > len(data) {
		offset = len(data)
	}
	reader := &countingByteReader{data: data[offset:]}
	decoder, err := open(reader)
	if err != nil {
		panic(carveFailure{msg: "Unable to parse the compressed stream"})
	}

	// Reading to the end is what makes the count meaningful; closing is what
	// checks the trailer. Either can be where a damaged stream shows itself, and
	// neither leaves a length worth carving to.
	_, readErr := io.Copy(io.Discard, decoder)
	closeErr := decoder.Close()
	if readErr != nil || closeErr != nil {
		panic(carveFailure{msg: "Unable to parse the compressed stream"})
	}
	return data[offset : offset+reader.pos]
}

// bzip2EndOfStream holds the end-of-stream marker as it appears at each bit
// offset it can be shifted to: a bzip2 stream is a bit stream, so the marker
// does not have to start on a byte boundary. Taken from CyberChef's own list.
var bzip2EndOfStream = [][]byte{
	{0x77, 0x24, 0x53, 0x85, 0x09},
	{0xee, 0x48, 0xa7, 0x0a, 0x12},
	{0xdc, 0x91, 0x4e, 0x14, 0x24},
	{0xb9, 0x22, 0x9c, 0x28, 0x48},
	{0x72, 0x45, 0x38, 0x50, 0x90},
	{0xbb, 0x92, 0x29, 0xc2, 0x84},
	{0x5d, 0xc9, 0x14, 0xe1, 0x42},
	{0x2e, 0xe4, 0x8a, 0x70, 0xa1},
	{0x17, 0x72, 0x45, 0x38, 0x50},
}

// bzip2FooterWidth is how much of the footer follows the marker: the rest of the
// stream checksum and the bits padding the last byte out.
const bzip2FooterWidth = 4

// carveBZIP2 ends the file after the end-of-stream marker and the checksum
// following it.
func carveBZIP2(data []byte, offset int) []byte {
	s := carveStream(data, offset)

	for _, marker := range bzip2EndOfStream {
		s.ContinueUntil(marker)
		if bytes.Equal(s.GetBytes(len(marker)), marker) {
			break
		}
		// Not this shift; start again looking for the next one.
		s.MoveTo(0)
	}
	s.MoveForwardsBy(bzip2FooterWidth)
	return s.Carve(0, s.Pos)
}

// xzEndOfStream is the stream footer, whose last two bytes are the format's
// "YZ" identifier.
var xzEndOfStream = []byte{0x00, 0x00, 0x00, 0x00, 0x04, 0x59, 0x5a}

// carveXZ ends the file after the stream footer.
func carveXZ(data []byte, offset int) []byte {
	s := carveStream(data, offset)

	s.ContinueUntil(xzEndOfStream)
	s.MoveForwardsBy(len(xzEndOfStream))
	return s.Carve(0, s.Pos)
}

// The layout of an ar member header, which is what a Debian package is a
// sequence of: a fixed-width text header whose last two bytes are a marker, with
// the member's size written as decimal digits near the end.
const (
	arHeaderWidth = 60
	arSizeOffset  = 48
	arSizeWidth   = 10
	arMagicOffset = 58
)

// arHeaderMagic ends every member header.
var arHeaderMagic = []byte{0x60, 0x0a}

// carveDEB walks the archive's members to the end of the last one.
//
// CyberChef reads members until the buffer runs out
// (src/core/lib/FileSignatures.mjs), which for a package embedded in a larger
// buffer swallows whatever follows it. Each member header ends with a two-byte
// marker, so this stops at the first thing that is not one — and members are
// padded to an even length, which the walk also has to allow for.
func carveDEB(data []byte, offset int) []byte {
	s := carveStream(data, offset)

	// Over the "!<arch>\n" that opens the archive.
	s.MoveForwardsBy(8)

	for s.Pos+arHeaderWidth <= s.Length() {
		if !bytes.Equal(s.Bytes[s.Pos+arMagicOffset:s.Pos+arHeaderWidth], arHeaderMagic) {
			break
		}
		size := arDecimal(s.Bytes[s.Pos+arSizeOffset : s.Pos+arSizeOffset+arSizeWidth])
		if size < 0 {
			break
		}
		// Members are padded to an even length.
		s.MoveForwardsBy(arHeaderWidth + size + size%2)
	}
	return s.Carve(0, s.Pos)
}

// arDecimal reads a space-padded decimal field, or returns -1 when it does not
// hold one.
func arDecimal(field []byte) int {
	digits := strings.TrimSpace(string(field))
	size, err := strconv.Atoi(digits)
	if err != nil || size < 0 {
		return -1
	}
	return size
}

// The LZOP header flags naming which checksums each block carries, and whether
// the header itself carries optional fields.
const (
	lzopAdler32Data = 0x00000001
	lzopAdler32Comp = 0x00000002
	lzopCRC32Data   = 0x00000100
	lzopCRC32Comp   = 0x00000200
	lzopHeaderFiler = 0x00000800
	lzopExtraField  = 0x00000040
	lzopVersion940  = 0x0940
)

// carveLZOP walks the compressed blocks to the empty one that ends the file.
func carveLZOP(data []byte, offset int) []byte {
	s := carveStream(data, offset)

	// Over the nine magic bytes to the version.
	s.MoveForwardsBy(9)
	version := s.ReadInt(2)

	// Over the library and method fields to the flags.
	s.MoveForwardsBy(6)
	flags := s.ReadInt(4)

	if version&lzopHeaderFiler != 0 {
		s.MoveForwardsBy(4)
	}

	compSums := countBits(flags, lzopAdler32Comp, lzopCRC32Comp)
	dataSums := countBits(flags, lzopAdler32Data, lzopCRC32Data)

	// Over the mode and modification times.
	s.MoveForwardsBy(8)
	if version >= lzopVersion940 {
		s.MoveForwardsBy(4)
	}

	// The original file name, then any extra field.
	s.MoveForwardsBy(s.ReadInt(1))
	if flags&lzopExtraField != 0 {
		s.MoveForwardsBy(s.ReadInt(4))
	}

	// Over the header checksum.
	s.MoveForwardsBy(4)

	for s.HasMore() {
		uncompressed := s.ReadInt(4)
		if uncompressed == 0 {
			break
		}
		compressed := s.ReadInt(4)

		// A block stored rather than compressed carries only the data checksums.
		sums := dataSums
		if uncompressed != compressed {
			sums += compSums
		}
		s.MoveForwardsBy(compressed + sums*4)
	}
	return s.Carve(0, s.Pos)
}

// countBits returns how many of the given flags are set.
func countBits(flags int, of ...int) int {
	n := 0
	for _, f := range of {
		if flags&f != 0 {
			n++
		}
	}
	return n
}
