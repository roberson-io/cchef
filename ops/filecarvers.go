package ops

import (
	"fmt"
	"strings"

	"github.com/roberson-io/cchef/core"
)

// carveFunc cuts one file out of bytes, starting at offset, and returns just
// that file's bytes. It reports a problem by raising a streamError through the
// stream it reads with, which extractFile turns back into an error value.
type carveFunc func(bytes []byte, offset int) []byte

// carvers maps the names the signature table uses to the algorithms themselves.
// The table is generated from CyberChef and names carvers as strings so that it
// stays free of Go identifiers; a test checks every name here resolves.
var carvers = map[string]carveFunc{
	"BMP":            carveBMP,
	"BZIP2":          carveBZIP2,
	"DEB":            carveDEB,
	"DMP":            carveDMP,
	"ELF":            carveELF,
	"EVT":            carveEVT,
	"EVTX":           carveEVTX,
	"FLV":            carveFLV,
	"GIF":            carveGIF,
	"GZIP":           carveGZIP,
	"ICO":            carveICO,
	"JPEG":           carveJPEG,
	"LNK":            carveLNK,
	"LZOP":           carveLZOP,
	"MACHO":          carveMACHO,
	"MP3":            carveMP3,
	"MZPE":           carveMZPE,
	"MacOSXKeychain": carveMacOSXKeychain,
	"PDF":            carvePDF,
	"PF":             carvePF,
	"PFWin10":        carvePFWin10,
	"PListXML":       carvePListXML,
	"PNG":            carvePNG,
	"RTF":            carveRTF,
	"SQLITE":         carveSQLITE,
	"TAR":            carveTAR,
	"TARGA":          carveTARGA,
	"WAV":            carveWAV,
	"WEBP":           carveWEBP,
	"XZ":             carveXZ,
	"ZIP":            carveZIP,
	"Zlib":           carveZlib,
}

// extractFile cuts the file described by details out of bytes at offset. A type
// with no carver, or one whose carve walks outside the buffer, is reported as an
// error; CyberChef throws in both cases.
func extractFile(data []byte, details fileSig, offset int) (core.NamedFile, error) {
	carve, ok := carvers[details.carver]
	if !ok {
		//nolint:staticcheck,revive // CyberChef's verbatim Error text
		return core.NamedFile{}, fmt.Errorf(
			"No extraction algorithm available for %q files", details.mime)
	}

	var carved []byte
	if err := catchStreamError(func() { carved = carve(data, offset) }); err != nil {
		return core.NamedFile{}, err
	}
	return core.NamedFile{
		Name: fmt.Sprintf("extracted_at_0x%x.%s", offset, firstExtension(details.extension)),
		Data: carved,
	}, nil
}

// firstExtension returns the first of a signature's comma-separated extensions,
// which is the one the carved file is named with.
func firstExtension(ext string) string {
	first, _, _ := strings.Cut(ext, ",")
	return first
}

// carveStream opens a reader over the buffer from offset onwards, which is where
// most carvers start: they work in positions relative to the file they are
// cutting out rather than to the buffer holding it.
func carveStream(data []byte, offset int) *byteStream {
	if offset > len(data) {
		offset = len(data)
	}
	return newByteStream(data[offset:])
}

// carvePNG walks the chunk list to the IEND chunk that ends the image.
func carvePNG(data []byte, offset int) []byte {
	s := carveStream(data, offset)

	// Past the eight-byte signature to the first chunk.
	s.moveForwardsBy(8)

	for chunkType := ""; chunkType != "IEND"; {
		chunkSize := s.readInt(4)
		chunkType = s.readString(4)

		// The chunk's data, then its four-byte checksum.
		s.moveForwardsBy(chunkSize + 4)
	}
	return s.carve(0, s.pos)
}

// carveJPEG walks the marker segments to the end-of-image marker.
func carveJPEG(data []byte, offset int) []byte {
	s := carveStream(data, offset)

	for s.hasMore() {
		marker := s.getBytes(2)
		if len(marker) < 2 || marker[0] != 0xff {
			panic(streamError{pos: s.pos})
		}

		switch marker[1] {
		case jpegStartOfImage, jpegArithmeticTemp:
			// Carries no length.
		case jpegEndOfImage:
			return s.carve(0, s.pos)
		case jpegExpandReference:
			s.advance(1)
		case jpegDefineNumberOfLines, jpegDefineRestartInterval:
			s.advance(2)
		case jpegStartOfScan:
			s.advance(s.readInt(2) - 2)
			s.continueUntilByte(0xff)
		default:
			if jpegSizedSegment(marker[1]) {
				s.advance(s.readInt(2) - 2)
				break
			}
			// Byte stuffing, a restart marker, or anything unrecognised: the
			// next marker is the next 0xff.
			s.continueUntilByte(0xff)
		}
	}
	//nolint:staticcheck,revive // CyberChef's verbatim Error text
	panic(carveFailure{msg: "Unable to parse JPEG successfully"})
}

// The JPEG markers that carry no length, or a fixed one.
const (
	jpegStartOfImage          = 0xd8
	jpegArithmeticTemp        = 0x01
	jpegEndOfImage            = 0xd9
	jpegExpandReference       = 0xdf
	jpegDefineNumberOfLines   = 0xdc
	jpegDefineRestartInterval = 0xdd
	jpegStartOfScan           = 0xda
)

// jpegSizedSegment reports whether a marker introduces a segment whose length is
// given by the two bytes following it: the frame and table markers (0xc0–0xcf),
// the quantisation and hierarchical-progression tables, the sixteen
// application-specific markers, and the comment.
func jpegSizedSegment(marker byte) bool {
	switch {
	case marker >= 0xc0 && marker <= 0xcf:
		return true
	case marker >= 0xe0 && marker <= 0xef:
		return true
	case marker == 0xdb, marker == 0xde, marker == 0xfe:
		return true
	}
	return false
}

// carveBMP reads the file size out of the header.
func carveBMP(data []byte, offset int) []byte {
	s := carveStream(data, offset)

	// Past the two-byte "BM" to the size field.
	s.moveForwardsBy(2)
	size := s.readIntLE(4)

	// The size counts the whole file, including the six bytes already read.
	s.moveForwardsBy(size - 6)
	return s.carve(0, s.pos)
}

// carveWEBP reads the RIFF chunk size, which covers everything after it.
func carveWEBP(data []byte, offset int) []byte {
	s := carveStream(data, offset)

	// Past "RIFF" to the size field.
	s.moveForwardsBy(4)
	size := s.readIntLE(4)

	// The size runs from the end of the field, so the eight bytes already read
	// need no adjustment.
	s.moveForwardsBy(size)
	return s.carve(0, s.pos)
}

// carveICO ends the file at the end of the last image it holds, which the
// directory gives the offset and size of.
func carveICO(data []byte, offset int) []byte {
	s := carveStream(data, offset)

	// Past the reserved and type fields to the image count.
	s.moveTo(4)
	count := s.readIntLE(2)

	// On to the last directory entry, which is 16 bytes wide.
	s.moveForwardsBy(icoEntrySize/2 + (count-1)*icoEntrySize)
	size := s.readIntLE(4)
	start := s.readIntLE(4)

	s.moveTo(start + size)
	return s.carve(0, s.pos)
}

// icoEntrySize is the width of one entry in an icon directory. The first eight
// bytes of an entry describe the image; the size and offset follow.
const icoEntrySize = 16

// The bytes that introduce each kind of GIF block.
const (
	gifExtension       = 0x21
	gifImageDescriptor = 0x2c
	gifTrailer         = 0x3b
)

// The fixed widths in a GIF, in bytes: the header and logical screen descriptor
// that open the file, and the image descriptor that opens each frame. Each of
// those two carries a flags byte saying whether a colour table follows it —
// gifScreenFlags counts from the start of the file, gifDescriptorFlags from the
// byte after the descriptor's introducer.
const (
	gifHeaderAndScreen = 13
	gifScreenFlags     = 10
	gifDescriptorWidth = 10
	gifDescriptorFlags = 8
)

// carveGIF walks the blocks to the trailer that ends the file.
//
// CyberChef's extractGIF instead seeks to the first graphic control extension
// and then steps over each frame by a fixed eleven bytes
// (src/core/lib/FileSignatures.mjs). That width has no room for a local colour
// table, so the walk derails on the first frame carrying one — which is most
// animations — and runs off the end of the buffer. This reads the block
// structure instead, which is what the fixed widths were standing in for.
func carveGIF(data []byte, offset int) []byte {
	s := carveStream(data, offset)

	// The header and logical screen descriptor, then the global colour table
	// when the descriptor's flags call for one.
	s.moveForwardsBy(gifHeaderAndScreen + gifColourTableSize(s.peek(gifScreenFlags)))

	for {
		switch s.readInt(1) {
		case gifTrailer:
			return s.carve(0, s.pos)

		case gifExtension:
			// A label naming the kind of extension, then its sub-blocks.
			s.moveForwardsBy(1)
			gifSkipSubBlocks(s)

		case gifImageDescriptor:
			// The rest of the descriptor, whose last byte says whether a local
			// colour table follows.
			flags := s.peek(s.pos + gifDescriptorFlags)
			s.moveForwardsBy(gifDescriptorWidth - 1 + gifColourTableSize(flags))
			// The LZW minimum code size, then the image's own sub-blocks.
			s.moveForwardsBy(1)
			gifSkipSubBlocks(s)

		default:
			panic(carveFailure{msg: "Unable to parse GIF successfully"})
		}
	}
}

// gifColourTableSize returns the width in bytes of the colour table a screen or
// image descriptor's flags call for, or zero when there is none. The low three
// bits hold one less than the power of two the table has entries of, and each
// entry is a red, green and blue byte.
func gifColourTableSize(flags byte) int {
	if flags&0x80 == 0 {
		return 0
	}
	return 3 * (1 << ((flags & 0x07) + 1))
}

// gifSkipSubBlocks moves past a run of length-prefixed sub-blocks and the empty
// block that ends it.
func gifSkipSubBlocks(s *byteStream) {
	for {
		size := s.readInt(1)
		if size == 0 {
			return
		}
		s.moveForwardsBy(size)
	}
}

// carveFailure is raised by a carver that has read a well-formed buffer but
// cannot find the end of the file. Unlike a streamError it carries its own
// wording, which CyberChef reports as the reason the extraction failed.
type carveFailure struct{ msg string }

func (e carveFailure) Error() string { return e.msg }
