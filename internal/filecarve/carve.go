// Package filecarve cuts an embedded file out of a larger buffer.
//
// Finding where a file starts is [github.com/roberson-io/cchef/internal/filesig]'s
// job; finding where it ends is this package's, and it takes a different walk
// for every format — following chunk lengths, block counts or end markers until
// the file is accounted for. A walk that runs off the end of the buffer has
// misread the data, and [ExtractFile] reports that as an error rather than
// returning a truncated file.
package filecarve

import (
	"errors"
	"fmt"
	"strings"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/bytestream"
	"github.com/roberson-io/cchef/internal/filesig"
)

// carveFunc cuts one file out of bytes, starting at offset, and returns just
// that file's bytes. It reports a problem by raising a stream error through the
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

// CanCarve reports whether an algorithm of that name exists. Signatures name
// their carver as a string, so a table entry can outlive its implementation;
// this is how a caller checks before promising a file type is recoverable.
func CanCarve(name string) bool {
	_, ok := carvers[name]
	return ok
}

// ExtractFile cuts the file described by details out of bytes at offset. A type
// with no carver, or one whose carve walks outside the buffer, is reported as an
// error; CyberChef throws in both cases.
func ExtractFile(data []byte, details filesig.Sig, offset int) (core.NamedFile, error) {
	carve, ok := carvers[details.Carver]
	if !ok {
		//nolint:staticcheck,revive // CyberChef's verbatim Error text
		return core.NamedFile{}, fmt.Errorf(
			"No extraction algorithm available for %q files", details.MIME)
	}

	var carved []byte
	if err := catchStreamError(func() { carved = carve(data, offset) }); err != nil {
		return core.NamedFile{}, err
	}
	return core.NamedFile{
		Name: fmt.Sprintf("extracted_at_0x%x.%s", offset, firstExtension(details.Extension)),
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
func carveStream(data []byte, offset int) *bytestream.Stream {
	if offset > len(data) {
		offset = len(data)
	}
	return bytestream.New(data[offset:])
}

// carvePNG walks the chunk list to the IEND chunk that ends the image.
func carvePNG(data []byte, offset int) []byte {
	s := carveStream(data, offset)

	// Past the eight-byte signature to the first chunk.
	s.MoveForwardsBy(8)

	for chunkType := ""; chunkType != "IEND"; {
		chunkSize := s.ReadInt(4)
		chunkType = s.ReadString(4)

		// The chunk's data, then its four-byte checksum.
		s.MoveForwardsBy(chunkSize + 4)
	}
	return s.Carve(0, s.Pos)
}

// carveJPEG walks the marker segments to the end-of-image marker.
func carveJPEG(data []byte, offset int) []byte {
	s := carveStream(data, offset)

	for s.HasMore() {
		marker := s.GetBytes(2)
		if len(marker) < 2 || marker[0] != 0xff {
			panic(bytestream.StreamError{Pos: s.Pos})
		}

		switch marker[1] {
		case jpegStartOfImage, jpegArithmeticTemp:
			// Carries no length.
		case jpegEndOfImage:
			return s.Carve(0, s.Pos)
		case jpegExpandReference:
			s.Advance(1)
		case jpegDefineNumberOfLines, jpegDefineRestartInterval:
			s.Advance(2)
		case jpegStartOfScan:
			s.Advance(s.ReadInt(2) - 2)
			s.ContinueUntilByte(0xff)
		default:
			if jpegSizedSegment(marker[1]) {
				s.Advance(s.ReadInt(2) - 2)
				break
			}
			// Byte stuffing, a restart marker, or anything unrecognised: the
			// next marker is the next 0xff.
			s.ContinueUntilByte(0xff)
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
	s.MoveForwardsBy(2)
	size := s.ReadIntLE(4)

	// The size counts the whole file, including the six bytes already read.
	s.MoveForwardsBy(size - 6)
	return s.Carve(0, s.Pos)
}

// carveWEBP reads the RIFF chunk size, which covers everything after it.
func carveWEBP(data []byte, offset int) []byte {
	s := carveStream(data, offset)

	// Past "RIFF" to the size field.
	s.MoveForwardsBy(4)
	size := s.ReadIntLE(4)

	// The size runs from the end of the field, so the eight bytes already read
	// need no adjustment.
	s.MoveForwardsBy(size)
	return s.Carve(0, s.Pos)
}

// carveICO ends the file at the end of the last image it holds, which the
// directory gives the offset and size of.
func carveICO(data []byte, offset int) []byte {
	s := carveStream(data, offset)

	// Past the reserved and type fields to the image count.
	s.MoveTo(4)
	count := s.ReadIntLE(2)

	// On to the last directory entry, which is 16 bytes wide.
	s.MoveForwardsBy(icoEntrySize/2 + (count-1)*icoEntrySize)
	size := s.ReadIntLE(4)
	start := s.ReadIntLE(4)

	s.MoveTo(start + size)
	return s.Carve(0, s.Pos)
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
	s.MoveForwardsBy(gifHeaderAndScreen + gifColourTableSize(s.Peek(gifScreenFlags)))

	for {
		switch s.ReadInt(1) {
		case gifTrailer:
			return s.Carve(0, s.Pos)

		case gifExtension:
			// A label naming the kind of extension, then its sub-blocks.
			s.MoveForwardsBy(1)
			gifSkipSubBlocks(s)

		case gifImageDescriptor:
			// The rest of the descriptor, whose last byte says whether a local
			// colour table follows.
			flags := s.Peek(s.Pos + gifDescriptorFlags)
			s.MoveForwardsBy(gifDescriptorWidth - 1 + gifColourTableSize(flags))
			// The LZW minimum code size, then the image's own sub-blocks.
			s.MoveForwardsBy(1)
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
func gifSkipSubBlocks(s *bytestream.Stream) {
	for {
		size := s.ReadInt(1)
		if size == 0 {
			return
		}
		s.MoveForwardsBy(size)
	}
}

// carveFailure is raised by a carver that has read a well-formed buffer but
// cannot find the end of the file. Unlike a stream error it carries its own
// wording, which CyberChef reports as the reason the extraction failed.
type carveFailure struct{ msg string }

func (e carveFailure) Error() string { return e.msg }

// catchStreamError runs fn, returning as an error any out-of-bounds move, or any
// other complaint a reader raised about the shape of what it was reading. Panics
// of every other kind are left alone.
func catchStreamError(fn func()) (err error) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		raised, ok := r.(error)
		if !ok {
			panic(r)
		}
		var se bytestream.StreamError
		var cf carveFailure
		if !errors.As(raised, &se) && !errors.As(raised, &cf) {
			panic(r)
		}
		err = raised
	}()
	fn()
	return nil
}
