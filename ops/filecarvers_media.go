package ops

// carveWAV reads the RIFF chunk size, which covers everything after it.
func carveWAV(data []byte, offset int) []byte {
	s := carveStream(data, offset)

	s.moveTo(4)
	s.moveTo(s.readIntLE(4) + 8)
	return s.carve(0, s.pos)
}

// carveFLV walks the tags of a Flash video to the first one that does not
// follow on from the last.
func carveFLV(data []byte, offset int) []byte {
	s := carveStream(data, offset)

	// Over the signature, version and flags to the header size.
	s.moveForwardsBy(5)
	headerSize := s.readInt(4)
	s.moveForwardsBy(headerSize - 9)

	// The size a previous tag would have had before the first one.
	tagSize := -11
	for s.hasMore() {
		previous := s.readInt(4)
		tagType := s.readInt(1)

		if tagType != flvAudio && tagType != flvVideo && tagType != flvScript {
			// Not a tag: step back over the type and stop.
			s.moveBackwardsBy(1)
			break
		}
		if previous != tagSize+flvTagHeader {
			// The tag before this one did not end here, so it was not part of
			// the file: step back over it and its header as well.
			s.moveBackwardsBy(tagSize + flvTagHeader + 5)
			break
		}

		tagSize = s.readInt(3)

		// Over the rest of the tag header and its payload.
		s.moveForwardsBy(7 + tagSize)
	}
	return s.carve(0, s.pos)
}

// The tag types a Flash video is made of, and the width of a tag header.
const (
	flvAudio     = 8
	flvVideo     = 9
	flvScript    = 18
	flvTagHeader = 11
)

// pdfEndOfFile is the marker that closes a document.
var pdfEndOfFile = []byte("%%EOF")

// carvePDF ends the file after its end-of-file marker and any line ending.
func carvePDF(data []byte, offset int) []byte {
	s := carveStream(data, offset)

	s.continueUntil(pdfEndOfFile)
	s.moveForwardsBy(len(pdfEndOfFile))
	s.consumeIf(0x0d)
	s.consumeIf(0x0a)
	return s.carve(0, s.pos)
}

// carveRTF ends the file at the brace that closes the one it opens with.
func carveRTF(data []byte, offset int) []byte {
	s := carveStream(data, offset)

	if s.readInt(1) != '{' {
		panic(carveFailure{msg: "Not a valid RTF file"})
	}

	for open := 1; open > 0 && s.hasMore(); {
		switch s.readInt(1) {
		case '{':
			open++
		case '}':
			open--
		case '\\':
			// An escape: take any further backslash, then the character the
			// escape applies to, so neither can be read as a brace.
			s.consumeIf('\\')
			s.advance(1)
		}
	}
	return s.carve(0, s.pos)
}

// The tags a property list opens and closes with.
var (
	plistOpenTag  = []byte("<plist")
	plistCloseTag = []byte("</plist>")
)

// carvePListXML ends the file at the tag closing the one it opens with.
func carvePListXML(data []byte, offset int) []byte {
	s := carveStream(data, offset)

	s.continueUntil(plistOpenTag)
	s.moveForwardsBy(len(plistOpenTag))

	for open := 1; open > 0 && s.hasMore(); {
		if s.readInt(1) != '<' {
			continue
		}
		s.moveBackwardsBy(1)
		switch {
		case s.lookingAt(plistOpenTag):
			s.moveForwardsBy(len(plistOpenTag))
			open++
		case s.lookingAt(plistCloseTag):
			s.moveForwardsBy(len(plistCloseTag))
			open--
		default:
			s.moveForwardsBy(1)
		}
	}
	s.consumeIf(0x0a)
	return s.carve(0, s.pos)
}

// The layout of a Targa image, in bytes: the header that opens the file, and the
// footer that a version 2 file ends with.
const (
	targaHeaderWidth = 18
	targaFooterWidth = 26
	targaSignatureAt = 8
	targaExtensionSz = 495
)

// The Targa image types whose data size can be worked out from the header. The
// run-length encoded types cannot be, as their size depends on the pixels.
const (
	targaColourMapped = 1
	targaTrueColour   = 2
	targaGreyscale    = 3
)

// carveTARGA works out where a Targa image begins and ends.
//
// A Targa file has no length field and nothing identifying its start — the only
// thing that can be found is the "TRUEVISION-XFILE." signature in the footer of
// a version 2 file, which is why the scanner matches there. CyberChef then walks
// backwards looking for a position whose bytes happen to equal the distance
// walked (src/core/lib/FileSignatures.mjs); that search does not describe
// anything about the format and does not find the start of a real file.
//
// This instead uses what the format does state. The end is certain: the footer
// is a fixed width and the signature sits at a fixed place inside it. For the
// start, the footer's extension-area offset is measured from the first byte of
// the file, which fixes it directly; failing that, a candidate start is accepted
// only when the dimensions in its header account for exactly the bytes between
// it and the footer.
func carveTARGA(data []byte, offset int) []byte {
	footerAt := offset - targaSignatureAt
	if footerAt < 0 {
		panic(carveFailure{msg: "Not a valid Targa file"})
	}
	end := footerAt + targaFooterWidth
	if end > len(data) {
		panic(carveFailure{msg: "Not a valid Targa file"})
	}

	s := newByteStream(data)
	s.moveTo(footerAt)
	extensionOffset := s.readIntLE(4)

	if start, ok := targaStartFromExtension(data, footerAt, extensionOffset); ok {
		return data[start:end]
	}
	if start, ok := targaStartFromImageSize(data, footerAt); ok {
		return data[start:end]
	}
	panic(carveFailure{msg: "Unable to find the start of the Targa file"})
}

// targaStartFromExtension locates the start of the file from the footer's
// extension-area offset, which is counted from the first byte of the file. The
// area is written immediately before the footer and opens with its own size, so
// finding that size where it is expected confirms the reading.
func targaStartFromExtension(data []byte, footerAt, extensionOffset int) (int, bool) {
	if extensionOffset <= 0 {
		return 0, false
	}
	areaAt := footerAt - targaExtensionSz
	if areaAt < 0 {
		return 0, false
	}
	if int(data[areaAt])|int(data[areaAt+1])<<8 != targaExtensionSz {
		return 0, false
	}
	start := areaAt - extensionOffset
	if start < 0 || start > footerAt {
		return 0, false
	}
	return start, true
}

// targaStartFromImageSize looks for the position whose header accounts for
// exactly the bytes between it and the footer.
func targaStartFromImageSize(data []byte, footerAt int) (int, bool) {
	for start := footerAt - targaHeaderWidth; start >= 0; start-- {
		if size, ok := targaFileSize(data, start); ok && start+size == footerAt {
			return start, true
		}
	}
	return 0, false
}

// targaFileSize returns how many bytes the image at start occupies before its
// footer, or false when its header does not describe one whose size can be
// worked out.
func targaFileSize(data []byte, start int) (int, bool) {
	if start+targaHeaderWidth > len(data) {
		return 0, false
	}
	h := data[start:]
	le16 := func(at int) int { return int(h[at]) | int(h[at+1])<<8 }

	idLength := int(h[0])
	colourMapType := int(h[1])
	imageType := int(h[2])
	colourMapLength := le16(5)
	colourMapDepth := int(h[7])
	width := le16(12)
	height := le16(14)
	pixelDepth := int(h[16])

	switch imageType {
	case targaColourMapped, targaTrueColour, targaGreyscale:
	default:
		return 0, false
	}
	if width == 0 || height == 0 || pixelDepth == 0 || pixelDepth%8 != 0 {
		return 0, false
	}
	if colourMapType != 0 && colourMapType != 1 {
		return 0, false
	}

	colourMap := 0
	if colourMapType == 1 {
		if colourMapDepth == 0 || colourMapDepth%8 != 0 {
			return 0, false
		}
		colourMap = colourMapLength * (colourMapDepth / 8)
	}
	return targaHeaderWidth + idLength + colourMap + width*height*(pixelDepth/8), true
}

// The MPEG audio versions and layers a frame header can name.
const (
	mpegVersion25 = 0
	mpegVersion2  = 2
	mpegVersion1  = 3
	mpegLayer3    = 1
	mpegLayer2    = 2
	mpegLayer1    = 3
)

// mp3Bitrates gives the bit rate in kbit/s for each index, by whether the frame
// is MPEG-1 and which layer it is. Index 0 means "free" and index 15 is invalid;
// both are left at zero and rejected.
var mp3Bitrates = map[[2]int][16]int{
	{1, mpegLayer1}: {0, 32, 64, 96, 128, 160, 192, 224, 256, 288, 320, 352, 384, 416, 448, 0},
	{1, mpegLayer2}: {0, 32, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 384, 0},
	{1, mpegLayer3}: {0, 32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 0},
	{0, mpegLayer1}: {0, 32, 48, 56, 64, 80, 96, 112, 128, 144, 160, 176, 192, 224, 256, 0},
	{0, mpegLayer2}: {0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160, 0},
	{0, mpegLayer3}: {0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160, 0},
}

// mp3SampleRates gives the sampling rate in Hz for each index, by version.
var mp3SampleRates = map[int][4]int{
	mpegVersion1:  {44100, 48000, 32000, 0},
	mpegVersion2:  {22050, 24000, 16000, 0},
	mpegVersion25: {11025, 12000, 8000, 0},
}

// mp3TagV1Size is the width of the fixed-size tag some files end with.
const mp3TagV1Size = 128

// carveMP3 walks the frame headers to the first thing that is not one.
//
// CyberChef's extractMP3 reads three bytes before testing for a frame header
// (src/core/lib/FileSignatures.mjs), so the sync word it compares against is
// three bytes past where a frame actually begins, and it only accepts one of the
// several header forms. This reads each header where it is and takes the
// version and layer from it rather than assuming them.
func carveMP3(data []byte, offset int) []byte {
	s := carveStream(data, offset)

	// A leading ID3 tag carries its length as four seven-bit groups.
	if s.lookingAt([]byte("ID3")) {
		s.moveTo(6)
		size := 0
		for range 4 {
			size = size<<7 | s.readInt(1)
		}
		s.moveForwardsBy(size)
	}

	for s.hasMore() {
		if s.lookingAt([]byte("TAG")) {
			// The fixed-size tag that ends the file.
			s.moveForwardsBy(mp3TagV1Size)
			break
		}
		size, ok := mp3FrameSize(s)
		if !ok {
			break
		}
		if s.pos+size > s.length() {
			// The last frame is cut short; take what there is.
			s.moveTo(s.length())
			break
		}
		s.moveForwardsBy(size)
	}
	return s.carve(0, s.pos)
}

// mp3FrameSize reads the frame header at the current position and returns how
// many bytes the whole frame occupies, or false when there is no header there.
func mp3FrameSize(s *byteStream) (int, bool) {
	if s.pos+4 > s.length() {
		return 0, false
	}
	h := s.bytes[s.pos:]

	// Eleven set bits open a frame.
	if h[0] != 0xff || h[1]&0xe0 != 0xe0 {
		return 0, false
	}
	version := int(h[1]>>3) & 0x03
	layer := int(h[1]>>1) & 0x03
	if version == 1 || layer == 0 {
		// Both of those values are reserved.
		return 0, false
	}

	isMPEG1 := 0
	if version == mpegVersion1 {
		isMPEG1 = 1
	}
	bitrate := mp3Bitrates[[2]int{isMPEG1, layer}][int(h[2]>>4)] * 1000
	sampleRate := mp3SampleRates[version][int(h[2]>>2)&0x03]
	if bitrate == 0 || sampleRate == 0 {
		return 0, false
	}
	padding := int(h[2]>>1) & 0x01

	// A layer 1 frame holds 384 samples in four-byte slots; the other layers
	// hold 1152 samples on MPEG-1 and 576 on the later versions.
	if layer == mpegLayer1 {
		return (12*bitrate/sampleRate + padding) * 4, true
	}
	samples := 144
	if version != mpegVersion1 && layer == mpegLayer3 {
		samples = 72
	}
	return samples*bitrate/sampleRate + padding, true
}
