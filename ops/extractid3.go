package ops

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/core"
)

// The layout of an ID3 tag, in bytes.
const (
	id3HeaderWidth = 10
	// The identifier and frame header are narrower in ID3v2.2 than in the
	// versions after it.
	id3v22IDWidth     = 3
	id3v22FrameHeader = 6
	id3IDWidth        = 4
	id3FrameHeader    = 10
	// The version at which frame lengths became seven-bit numbers like the tag
	// length, rather than plain integers.
	id3SyncSafeFramesFrom = 4
	id3v22                = 2
)

// errID3NoHeader is what input that does not open with a tag gets.
//
//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
var errID3NoHeader = errors.New("No valid ID3 header.")

// ExtractID3 reads the ID3 metadata out of an MP3 file.
type ExtractID3 struct{}

// Meta returns the operation metadata.
func (ExtractID3) Meta() core.OpMeta {
	return core.OpMeta{
		Name:   "Extract ID3",
		Module: "Default",
		Description: "This operation extracts ID3 metadata from an MP3 file.<br><br>ID3 is " +
			"a metadata container most often used in conjunction with the MP3 audio file " +
			"format. It allows information such as the title, artist, album, track number, " +
			"and other information about the file to be stored in the file itself.",
		InfoURL:    "https://wikipedia.org/wiki/ID3",
		InputType:  core.TypeArrayBuffer,
		OutputType: core.TypeJSON,
	}
}

// Args returns the argument definitions.
func (ExtractID3) Args() []core.ArgDef { return nil }

// Run reads the tag.
func (ExtractID3) Run(in *core.Dish, _ []any) (*core.Dish, error) {
	data := in.Bytes()
	if len(data) < id3HeaderWidth || string(data[:3]) != "ID3" {
		return nil, errID3NoHeader
	}

	major := int(data[3])
	out := newOMap().
		set("Type", "ID3").
		set("Version", fmt.Sprintf("%d.%d", major, data[4])).
		set("Flags", strconv.Itoa(int(data[5]))).
		set("Size", strconv.Itoa(id3SyncSafe(data[6:id3HeaderWidth])))

	tags, err := id3ReadFrames(data, major)
	if err != nil {
		return nil, err
	}
	out.set("Tags", tags)

	encoded, err := marshalOMap(out)
	if err != nil {
		return nil, err
	}
	return core.NewDish(encoded, core.TypeJSON), nil
}

// id3SyncSafe reads a length written as seven-bit groups, which is how the tag
// length is always stored and how frame lengths are stored from ID3v2.4. The top
// bit of each byte is left clear so that a length can never look like the sync
// word that starts an audio frame.
func id3SyncSafe(b []byte) int {
	size := 0
	for _, c := range b {
		size = size<<7 | int(c&0x7f)
	}
	return size
}

// id3Plain reads a length written as an ordinary big-endian integer, which is
// how frame lengths are stored up to ID3v2.3.
func id3Plain(b []byte) int {
	size := 0
	for _, c := range b {
		size = size<<8 | int(c)
	}
	return size
}

// id3ReadFrames walks the frames of the tag, which run from the end of the
// header to the length the header gives.
func id3ReadFrames(data []byte, major int) (*omap, error) {
	tags := newOMap()
	tagSize := id3SyncSafe(data[6:id3HeaderWidth])

	idWidth, headerWidth := id3IDWidth, id3FrameHeader
	if major == id3v22 {
		idWidth, headerWidth = id3v22IDWidth, id3v22FrameHeader
	}

	// pos counts from the start of the file, and the frames start after the
	// header; the tag length counts only what follows the header.
	pos := id3HeaderWidth
	end := id3HeaderWidth + tagSize

	for pos+headerWidth <= end && pos+headerWidth <= len(data) {
		id := id3FrameID(data[pos : pos+idWidth])
		if id == "" {
			// Padding where an identifier would be: the tag ends here.
			break
		}
		description, known := id3FrameDescriptions[id]
		if !known {
			//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
			return nil, fmt.Errorf("Unknown Frame Identifier: %s", id)
		}

		size := id3FrameSize(data[pos+idWidth:pos+headerWidth], major, idWidth)
		body := pos + headerWidth
		tags.set(id, newOMap().
			set("Size", strconv.Itoa(size)).
			set("Description", description).
			set("Data", id3FrameData(data, body, size)))

		pos = body + size
	}
	return tags, nil
}

// id3FrameID reads a frame identifier, trimming the trailing null that pads a
// three-character identifier out in the later versions. An identifier that is
// all padding names no frame.
func id3FrameID(b []byte) string {
	id := strings.TrimRight(string(b), "\x00")
	if strings.Trim(id, "\x00 ") == "" {
		return ""
	}
	return id
}

// id3FrameSize reads the length of a frame's data. ID3v2.4 writes it as
// seven-bit groups like the tag length; the versions before it write an ordinary
// integer, three bytes wide in ID3v2.2 and four after that.
func id3FrameSize(b []byte, major, idWidth int) int {
	width := idWidth
	if major >= id3SyncSafeFramesFrom {
		return id3SyncSafe(b[:width])
	}
	if width > len(b) {
		width = len(b)
	}
	return id3Plain(b[:width])
}

// id3FrameData reads a frame's contents as one character per byte, leaving out
// the first, which says how the rest is encoded rather than being part of it.
// The bytes are taken as written rather than decoded, as CyberChef does.
func id3FrameData(data []byte, at, size int) string {
	var out strings.Builder
	for i := 1; i < size; i++ {
		if at+i >= len(data) {
			break
		}
		out.WriteRune(rune(data[at+i]))
	}
	return out.String()
}

func init() { core.Register(ExtractID3{}) }
