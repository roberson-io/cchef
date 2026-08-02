package audio

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/internal/jsnum"
	"github.com/roberson-io/cchef/internal/jsonval"
)

// Readers for each audio container. Ported from CyberChef's
// src/core/lib/AudioParsers.mjs. Each fills in the report it is given rather
// than returning one, so that a container carrying several metadata systems
// contributes all of them.

// The tables mapping each format's own tag names onto the common ones the report
// boils everything down to.
var (
	id3FrameNames = map[string]string{
		"TIT2": "Title/songname/content description", "TPE1": "Lead performer(s)/Soloist(s)",
		"TRCK": "Track number/Position in set", "TALB": "Album/Movie/Show title",
		"TDRC": "Recording time", "TYER": "Year", "TCON": "Content type",
		"TPE2": "Band/orchestra/accompaniment", "TLEN": "Length (ms)", "TCOM": "Composer",
		"COMM": "Comments", "APIC": "Attached picture", "GEOB": "General encapsulated object",
		"TXXX": "User defined text information frame", "UFID": "Unique file identifier",
		"PRIV": "Private frame",
	}
	id3ToCommon = map[string]string{
		"TIT2": "title", "TPE1": "artist", "TALB": "album", "TDRC": "date", "TYER": "date",
		"TRCK": "track", "TCON": "genre", "COMM": "comment", "TCOM": "composer",
		"TCOP": "copyright", "TLAN": "language",
	}
	vorbisToCommon = map[string]string{
		"TITLE": "title", "ARTIST": "artist", "ALBUM": "album", "DATE": "date",
		"TRACKNUMBER": "track", "GENRE": "genre", "COMMENT": "comment",
		"COMPOSER": "composer", "LANGUAGE": "language",
	}
	wmaToCommon = map[string]string{
		"WM/ALBUMTITLE": "album", "WM/GENRE": "genre", "WM/YEAR": "date",
		"WM/TRACKNUMBER": "track", "WM/COMPOSER": "composer", "WM/LANGUAGE": "language",
	}
)

// The mappings that are walked in a fixed order, since a later source must not
// overwrite what an earlier one already set.
var (
	id3v1ToCommon = [][2]string{
		{"title", "title"},
		{"artist", "artist"},
		{"album", "album"},
		{"year", "date"},
		{"comment", "comment"},
		{"genre", "genre"},
		{"track", "track"},
	}
	riffToCommon = [][2]string{
		{"INAM", "title"},
		{"IART", "artist"},
		{"ICMT", "comment"},
		{"IGNR", "genre"},
		{"ICRD", "date"},
		{"ICOP", "copyright"},
	}
	asfContentToCommon = [][2]string{
		{"title", "title"},
		{"author", "artist"},
		{"copyright", "copyright"},
		{"description", "comment"},
	}
	vorbisCommonOrder = []string{
		"TITLE", "ARTIST", "ALBUM", "DATE", "TRACKNUMBER",
		"GENRE", "COMMENT", "COMPOSER", "LANGUAGE",
	}
)

// mapCommonFrom fills in the common tags from a source that names them its own
// way, leaving alone anything already set.
func mapCommonFrom(r *Report, source map[string]string, mapping [][2]string) {
	for _, pair := range mapping {
		r.setCommonIfEmpty(pair[1], source[pair[0]])
	}
}

// ---------------------------------------------------------------- MP3 -------

// parseMp3 reads the tags an MP3 can carry at either end of the file.
func parseMp3(b []byte, r *Report) {
	processID3v2(b, r)
	processID3v1(b, r)

	if ape := parseApeV2BestEffort(b); ape != nil {
		r.addSystem("apev2")
		r.raw.Set("apev2", ape)
	}
}

// processID3v2 reads the tag at the start of the file.
func processID3v2(b []byte, r *Report) {
	r.addSystem("id3v2")

	tag := parseID3v2Tag(b)
	if tag == nil {
		r.raw.Set("id3v2", nil)
		r.dropSystem("id3v2")
		return
	}

	frames := []any{}
	raw := jsonval.NewOMap().Set("header", tag.header).Set("frames", frames)
	r.raw.Set("id3v2", raw)

	var txxx []any
	for _, f := range tag.frames {
		entry := jsonval.NewOMap().
			Set("id", f.id).
			Set("size", f.size).
			Set("description", nullIfEmpty(id3FrameNames[f.id]))

		switch {
		case strings.HasPrefix(f.id, "T") && f.id != "TXXX":
			text := ""
			if len(f.data) >= 1 {
				text = stripNullsAndTrim(decodeText(f.data[1:], int(f.data[0])))
			}
			entry.Set("decoded", text)
			if f.id == "TLEN" {
				if ms, ok := normalizeTlen(text); ok {
					entry.Set("normalized_ms", ms)
				}
			}
			r.setCommonIfEmpty(id3ToCommon[f.id], text)

		case f.id == "TXXX":
			value := decodeTxxx(f.data)
			entry.Set("decoded", value)
			txxx = append(txxx, value)
			raw.Set("txxx", txxx)

		case f.id == "COMM":
			comm := decodeCommFrame(f.data)
			entry.Set("decoded", comm)
			if comm != nil {
				if text, ok := comm.Value("text").(string); ok {
					r.setCommonIfEmpty("comment", text)
				}
			}

		case f.id == "GEOB":
			processGeobFrame(f, entry, r)
		}

		frames = append(frames, entry)
		raw.Set("frames", frames)
	}
}

// processGeobFrame reads an encapsulated object and records what it carries.
func processGeobFrame(f id3Frame, entry *jsonval.OMap, r *Report) {
	d := f.data
	encoding := audioByteAt(d, 0)

	mimeBytes, off := nullTerminated(d, 1, textLatin1)
	mimeType := decodeLatin1Trim(mimeBytes)
	fileBytes, off := nullTerminated(d, off, encoding)
	filename := stripNullsAndTrim(decodeText(fileBytes, encoding))
	descBytes, off := nullTerminated(d, off, encoding)
	description := stripNullsAndTrim(decodeText(descBytes, encoding))

	objectBytes := len(d) - off

	entry.Set("geob", jsonval.NewOMap().
		Set("mimeType", mimeType).
		Set("filename", filename).
		Set("description", description).
		Set("object_bytes", objectBytes))

	n := r.countEmbedded(func(e *jsonval.OMap) bool { return e.Value("source") == "id3v2:GEOB" })
	r.addEmbedded(jsonval.NewOMap().
		Set("id", geobIdentifier(n)).
		Set("source", "id3v2:GEOB").
		Set("content_type", nullIfEmpty(mimeType)).
		Set("byte_length", objectBytes).
		Set("description", nullIfEmpty(description)).
		Set("filename", nullIfEmpty(filename)))

	mt := strings.ToLower(mimeType)
	if strings.Contains(mt, "c2pa") || strings.Contains(mt, "jumbf") {
		c2pa := r.provenanceC2PA()
		c2pa.Set("present", true)
		embedding, _ := c2pa.Value("embedding").([]any)
		c2pa.Set("embedding", append(embedding, jsonval.NewOMap().
			Set("carrier", "id3v2:GEOB").
			Set("content_type", nullIfEmpty(mimeType)).
			Set("byte_length", objectBytes)))
	}
}

// processID3v1 reads the fixed-size tag at the end of the file.
func processID3v1(b []byte, r *Report) {
	tag, fields := parseID3v1(b)
	if tag == nil {
		return
	}
	r.addSystem("id3v1")
	r.raw.Set("id3v1", tag)
	mapCommonFrom(r, fields, id3v1ToCommon)
}

// id3Frame is one frame of an ID3v2 tag.
type id3Frame struct {
	id   string
	size int
	data []byte
}

// id3Tag is a whole ID3v2 tag.
type id3Tag struct {
	header *jsonval.OMap
	frames []id3Frame
}

// parseID3v2Tag reads the tag at the start of the file, or reports that there is
// none.
func parseID3v2Tag(b []byte) *id3Tag {
	if len(b) < 10 || b[0] != 0x49 || b[1] != 0x44 || b[2] != 0x33 {
		return nil
	}
	major, minor, flags := int(b[3]), int(b[4]), int(b[5])
	tagSize := synchsafeToInt(int(b[6]), int(b[7]), int(b[8]), int(b[9]))

	tag := &id3Tag{
		header: jsonval.NewOMap().
			Set("version", fmt.Sprintf("%d.%d", major, minor)).
			Set("flags", flags).
			Set("tag_size", tagSize),
	}

	offset, end := 10, 10+tagSize
	for offset+10 <= end {
		id := ascii4(b, offset)
		if !isID3FrameID(id) {
			break
		}
		var size int
		if major == 4 {
			size = synchsafeToInt(audioByteAt(b, offset+4), audioByteAt(b, offset+5),
				audioByteAt(b, offset+6), audioByteAt(b, offset+7))
		} else {
			size = u32be(b, offset+4)
		}
		offset += 10
		if size <= 0 || offset+size > len(b) {
			break
		}
		tag.frames = append(tag.frames, id3Frame{id: id, size: size, data: b[offset : offset+size]})
		offset += size
	}
	return tag
}

// isID3FrameID reports whether s is four capital letters or digits, which is
// what a frame identifier looks like.
func isID3FrameID(s string) bool {
	if len(s) != 4 {
		return false
	}
	for i := range len(s) {
		c := s[i]
		if (c < 'A' || c > 'Z') && (c < '0' || c > '9') {
			return false
		}
	}
	return true
}

// parseID3v1 reads the 128-byte tag at the end of the file, returning both the
// report section and the fields to map the common tags from.
func parseID3v1(b []byte) (*jsonval.OMap, map[string]string) {
	if len(b) < 128 {
		return nil, nil
	}
	off := len(b) - 128
	if b[off] != 0x54 || b[off+1] != 0x41 || b[off+2] != 0x47 {
		return nil, nil
	}

	var track any
	trackText := ""
	if b[off+125] == 0x00 && b[off+126] != 0x00 {
		trackText = strconv.Itoa(int(b[off+126]))
		track = trackText
	}

	fields := map[string]string{
		"title":   decodeLatin1Trim(b[off+3 : off+33]),
		"artist":  decodeLatin1Trim(b[off+33 : off+63]),
		"album":   decodeLatin1Trim(b[off+63 : off+93]),
		"year":    decodeLatin1Trim(b[off+93 : off+97]),
		"comment": decodeLatin1Trim(b[off+97 : off+127]),
		"track":   trackText,
		"genre":   strconv.Itoa(int(b[off+127])),
	}

	return jsonval.NewOMap().
		Set("title", fields["title"]).
		Set("artist", fields["artist"]).
		Set("album", fields["album"]).
		Set("year", fields["year"]).
		Set("comment", fields["comment"]).
		Set("track", track).
		Set("genre", fields["genre"]), fields
}

// decodeTxxx reads a user-defined text frame, which is a description and a value.
func decodeTxxx(d []byte) any {
	if len(d) < 2 {
		return nil
	}
	encoding := int(d[0])
	descBytes, next := nullTerminated(d, 1, encoding)
	desc := stripNullsAndTrim(decodeText(descBytes, encoding))
	value := ""
	if next <= len(d) {
		value = stripNullsAndTrim(decodeText(d[next:], encoding))
	}
	return jsonval.NewOMap().Set("description", nullIfEmpty(desc)).Set("value", nullIfEmpty(value))
}

// decodeCommFrame reads a comment frame, which names the language it is in.
func decodeCommFrame(d []byte) *jsonval.OMap {
	if len(d) < 5 {
		return nil
	}
	encoding := int(d[0])
	language := string([]rune{rune(d[1]), rune(d[2]), rune(d[3])})
	descBytes, next := nullTerminated(d, 4, encoding)
	short := stripNullsAndTrim(decodeText(descBytes, encoding))
	text := ""
	if next <= len(d) {
		text = stripNullsAndTrim(decodeText(d[next:], encoding))
	}
	return jsonval.NewOMap().
		Set("language", language).
		Set("short_description", nullIfEmpty(short)).
		Set("text", nullIfEmpty(text))
}

// normalizeTlen reads a length frame, which holds either whole milliseconds or a
// number of seconds.
func normalizeTlen(s string) (int, bool) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return 0, false
	}
	if whole, err := strconv.Atoi(trimmed); err == nil && isAllDigits(trimmed) {
		return whole, true
	}
	f, err := strconv.ParseFloat(trimmed, 64)
	if err != nil || f <= 0 || f >= 100000 {
		return 0, false
	}
	return jsnum.Round(f * 1000), true
}

// isAllDigits reports whether s is nothing but decimal digits.
func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := range len(s) {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// parseApeV2BestEffort looks for an APE tag near the end of the file.
func parseApeV2BestEffort(b []byte) *jsonval.OMap {
	scanStart := max(0, len(b)-32768)
	idx := indexOfASCII(b, "APETAGEX", scanStart, len(b))
	if idx < 0 {
		return nil
	}
	if idx+32 > len(b) {
		return jsonval.NewOMap().
			Set("present", true).
			Set("warning", "APETAGEX found but footer truncated.")
	}

	version, size := u32le(b, idx+8), u32le(b, idx+12)
	count, flags := u32le(b, idx+16), u32le(b, idx+20)

	tagStart := idx + 32 - size
	if tagStart < 0 || tagStart >= len(b) {
		return jsonval.NewOMap().
			Set("present", true).Set("version", version).Set("size", size).
			Set("count", count).Set("flags", flags).
			Set("warning", "APEv2 bounds invalid (non-standard placement).")
	}

	items := []any{}
	off, end := tagStart+32, min(len(b), idx)
	for off+8 < end && len(items) < 5000 {
		valueSize, itemFlags := u32le(b, off), u32le(b, off+4)
		off += 8
		keyEnd := off
		for keyEnd < end && b[keyEnd] != 0x00 {
			keyEnd++
		}
		key := decodeLatin1Trim(b[off:keyEnd])
		off = keyEnd + 1
		if key == "" || off+valueSize > end {
			break
		}
		value := stripNullsAndTrim(safeUtf8(b[off : off+valueSize]))
		off += valueSize
		items = append(items, jsonval.NewOMap().Set("key", key).Set("value", value).Set("flags", itemFlags))
	}

	return jsonval.NewOMap().
		Set("present", true).Set("version", version).Set("size", size).
		Set("count", count).Set("flags", flags).Set("items", items)
}

// --------------------------------------------------------------- RIFF -------

// riffChunk is one chunk of a RIFF file.
type riffChunk struct {
	id      string
	size    int
	dataOff int
}

// enumerateChunks walks the chunks between start and end.
func enumerateChunks(b []byte, start, end, maxCount int) []riffChunk {
	var chunks []riffChunk
	off := start
	for off+8 <= end && len(chunks) < maxCount {
		id := ascii4(b, off)
		size := u32le(b, off+4)
		dataOff := off + 8
		if dataOff+size > end {
			break
		}
		chunks = append(chunks, riffChunk{id: id, size: size, dataOff: dataOff})
		off = dataOff + size + size%2
	}
	return chunks
}

// parseRiffWave reads a WAV file's chunks.
func parseRiffWave(b []byte, r *Report, maxText int) {
	r.addSystem("riff_info")

	riff := jsonval.NewOMap().
		Set("chunks", []any{}).
		Set("info", nil).
		Set("bext", nil).
		Set("ixml", nil).
		Set("axml", nil).
		Set("ds64", nil)

	chunkList := []any{}
	info := jsonval.NewOMap()
	for _, c := range enumerateChunks(b, 12, len(b), 50000) {
		chunkList = append(chunkList, jsonval.NewOMap().
			Set("id", c.id).Set("size", c.size).Set("offset", c.dataOff))
		processRiffChunk(b, c, riff, info, r, maxText)
	}
	riff.Set("chunks", chunkList)

	if len(info.Keys()) > 0 {
		riff.Set("info", info)
	}
	r.raw.Set("riff", riff)

	if len(info.Keys()) > 0 {
		values := map[string]string{}
		for _, k := range info.Keys() {
			if s, ok := info.Value(k).(string); ok {
				values[k] = s
			}
		}
		mapCommonFrom(r, values, riffToCommon)
	}
}

// processRiffChunk reads whichever of the chunks carry metadata.
func processRiffChunk(b []byte, c riffChunk, riff, info *jsonval.OMap, r *Report, maxText int) {
	switch {
	case c.id == "ds64":
		riff.Set("ds64", jsonval.NewOMap().Set("present", true).Set("size", c.size))
		r.addSystem("bw64_ds64")
		if r.container.Value("type") == "wav" {
			r.setContainerType("bw64")
		}

	case c.id == "LIST" && ascii4(b, c.dataOff) == "INFO":
		for _, s := range enumerateChunks(b, c.dataOff+4, c.dataOff+c.size, 10000) {
			info.Set(s.id, decodeLatin1Trim(b[s.dataOff:min(s.dataOff+s.size, len(b))]))
		}

	case c.id == "bext":
		r.addSystem("bwf_bext")
		riff.Set("bext", parseBext(b, c.dataOff, c.size))

	case c.id == "iXML", c.id == "axml":
		key := "axml"
		if c.id == "iXML" {
			key = "ixml"
		}
		r.addSystem(key)
		payload := b[min(c.dataOff, len(b)):min(c.dataOff+c.size, len(b))]
		riff.Set(key, jsonval.NewOMap().
			Set("xml", safeUtf8(payload[:min(len(payload), maxText)])).
			Set("truncated", len(payload) > maxText))
		r.addEmbedded(jsonval.NewOMap().
			Set("id", key+"_0").
			Set("source", "riff:"+c.id).
			Set("content_type", "application/xml").
			Set("byte_length", len(payload)).
			Set("description", c.id+" chunk").
			Set("filename", nil))
	}
}

// parseBext reads the broadcast extension chunk.
func parseBext(b []byte, off, size int) *jsonval.OMap {
	end := min(off+size, len(b))
	slice := b[min(off, len(b)):end]
	low, high := u32le(slice, 338), u32le(slice, 342)
	// #nosec G115 -- the two halves are four-byte fields being put back together
	// into the eight-byte sample count they were split from
	reference := uint64(uint32(high))<<32 | uint64(uint32(low))

	return jsonval.NewOMap().
		Set("description", nullIfEmpty(decodeLatin1Trim(sliceRange(slice, 0, 256)))).
		Set("originator", nullIfEmpty(decodeLatin1Trim(sliceRange(slice, 256, 288)))).
		Set("originator_reference", nullIfEmpty(decodeLatin1Trim(sliceRange(slice, 288, 320)))).
		Set("origination_date", nullIfEmpty(decodeLatin1Trim(sliceRange(slice, 320, 330)))).
		Set("origination_time", nullIfEmpty(decodeLatin1Trim(sliceRange(slice, 330, 338)))).
		Set("time_reference_samples", strconv.FormatUint(reference, 10))
}

// sliceRange returns the bytes between from and to, clamped to what is there.
func sliceRange(b []byte, from, to int) []byte {
	if from > len(b) {
		return nil
	}
	return b[from:min(to, len(b))]
}

// --------------------------------------------------------------- FLAC -------

// flacTypeNames name the metadata blocks a FLAC file opens with.
var flacTypeNames = map[int]string{
	0: "STREAMINFO", 1: "PADDING", 2: "APPLICATION", 3: "SEEKTABLE",
	4: "VORBIS_COMMENT", 5: "CUESHEET", 6: "PICTURE",
}

// flacBlock is one metadata block.
type flacBlock struct {
	typeName string
	length   int
	data     []byte
}

// parseFlacMetaBlocks walks the blocks between the signature and the audio.
func parseFlacMetaBlocks(b []byte) []flacBlock {
	var blocks []flacBlock
	off := 4
	for off+4 <= len(b) && len(blocks) < 10000 {
		header := b[off]
		isLast := header&0x80 != 0
		typ := int(header & 0x7f)
		length := int(b[off+1])<<16 | int(b[off+2])<<8 | int(b[off+3])
		off += 4
		if off+length > len(b) {
			break
		}
		name, ok := flacTypeNames[typ]
		if !ok {
			name = fmt.Sprintf("TYPE_%d", typ)
		}
		blocks = append(blocks, flacBlock{typeName: name, length: length, data: b[off : off+length]})
		off += length
		if isLast {
			break
		}
	}
	return blocks
}

// parseFlac reads a FLAC file's metadata blocks.
func parseFlac(b []byte, r *Report, maxText int) {
	r.addSystem("flac_metablocks")

	blockList := []any{}
	flac := jsonval.NewOMap().Set("blocks", blockList)
	r.raw.Set("flac", flac)

	for _, blk := range parseFlacMetaBlocks(b) {
		blockList = append(blockList, jsonval.NewOMap().
			Set("type", blk.typeName).Set("length", blk.length))
		flac.Set("blocks", blockList)

		switch blk.typeName {
		case "VORBIS_COMMENT":
			r.addSystem("vorbis_comments")
			vc, comments := parseVorbisComment(blk.data)
			r.raw.Set("vorbis_comments", vc)
			mapVorbisCommon(r, comments)

		case "PICTURE":
			pic := parseFlacPicture(blk.data, maxText)
			n := r.countEmbedded(func(e *jsonval.OMap) bool {
				id, _ := e.Value("id").(string)
				return strings.HasPrefix(id, "cover_art_")
			})
			r.addEmbedded(jsonval.NewOMap().
				Set("id", fmt.Sprintf("cover_art_%d", n)).
				Set("source", "flac:PICTURE").
				Set("content_type", nullIfEmpty(pic.mime)).
				Set("byte_length", pic.dataLength).
				Set("description", nullIfEmpty(pic.description)).
				Set("filename", nil))
		}
	}
}

// flacPicture describes an image carried in a FLAC file.
type flacPicture struct {
	mime        string
	description string
	dataLength  int
}

// parseFlacPicture reads a picture block's header.
func parseFlacPicture(d []byte, maxText int) flacPicture {
	off := 4
	mimeLen := u32be(d, off)
	off += 4
	mime := safeUtf8(sliceRange(d, off, off+min(mimeLen, maxText)))
	off += mimeLen
	descLen := u32be(d, off)
	off += 4
	description := safeUtf8(sliceRange(d, off, off+min(descLen, maxText)))
	off += descLen + 16
	return flacPicture{mime: mime, description: description, dataLength: u32be(d, off)}
}

// ---------------------------------------------------------- Vorbis / OGG ----

// vorbisComment is one key and value from a comment block.
type vorbisComment struct {
	key   string
	value string
}

// parseVorbisComment reads a comment block, returning both the report section
// and the comments to map the common tags from.
func parseVorbisComment(buf []byte) (*jsonval.OMap, []vorbisComment) {
	off := 0
	vendorLen := u32le(buf, off)
	off += 4
	if off+vendorLen > len(buf) {
		return jsonval.NewOMap().
			Set("vendor", nil).
			Set("comments", []any{}).
			Set("warning", "vendor_len out of bounds"), nil
	}
	vendor := safeUtf8(buf[off : off+vendorLen])
	off += vendorLen
	count := u32le(buf, off)
	off += 4

	list := []any{}
	var comments []vorbisComment
	for i := 0; i < count && off+4 <= len(buf) && len(list) < 20000; i++ {
		length := u32le(buf, off)
		off += 4
		if off+length > len(buf) {
			break
		}
		s := safeUtf8(buf[off : off+length])
		off += length
		if eq := strings.Index(s, "="); eq > 0 {
			c := vorbisComment{key: strings.ToUpper(s[:eq]), value: s[eq+1:]}
			comments = append(comments, c)
			list = append(list, jsonval.NewOMap().Set("key", c.key).Set("value", c.value))
		}
	}
	return jsonval.NewOMap().Set("vendor", vendor).Set("comments", list), comments
}

// mapVorbisCommon fills in the common tags from a comment block.
func mapVorbisCommon(r *Report, comments []vorbisComment) {
	for _, key := range vorbisCommonOrder {
		for _, c := range comments {
			if c.key == key {
				r.setCommonIfEmpty(vorbisToCommon[key], c.value)
				break
			}
		}
	}
}

// parseOgg reads the comment block of an Ogg or Opus stream.
func parseOgg(b []byte, r *Report) {
	r.addSystem("ogg_opus_tags")

	scanEnd := min(len(b), 1024*1024)
	var tags *jsonval.OMap
	var comments []vorbisComment

	opusIdx := indexOfASCII(b, "OpusTags", 0, scanEnd)
	if opusIdx >= 0 {
		r.setContainerType("opus")
		tags, comments = parseVorbisComment(b[min(opusIdx+8, len(b)):scanEnd])
	} else if vorbisIdx := indexOfASCII(b, "\x03vorbis", 0, scanEnd); vorbisIdx >= 0 {
		tags, comments = parseVorbisComment(b[min(vorbisIdx+7, len(b)):scanEnd])
	}

	r.raw.Set("ogg", jsonval.NewOMap().
		Set("has_opustags", opusIdx >= 0).
		Set("has_vorbis_comment", tags != nil))

	if tags != nil {
		r.addSystem("vorbis_comments")
		r.raw.Set("vorbis_comments", tags)
		mapVorbisCommon(r, comments)
	}
}

// ---------------------------------------------------------- MP4 / AIFF ------

// parseMp4BestEffort walks the top-level atoms of an MP4 file.
func parseMp4BestEffort(b []byte, r *Report) {
	r.addSystem("mp4_atoms")

	var atoms []*jsonval.OMap
	seen := map[string]bool{}
	off := 0
	for off+8 <= len(b) && len(atoms) < 2000 {
		size := u32be(b, off)
		typ := ascii4(b, off+4)
		if size < 8 {
			break
		}
		atoms = append(atoms, jsonval.NewOMap().Set("type", typ).Set("size", size).Set("offset", off))
		seen[typ] = true
		off += size
	}

	topLevel := []any{}
	for _, a := range atoms[:min(len(atoms), 200)] {
		topLevel = append(topLevel, a)
	}

	r.raw.Set("mp4", jsonval.NewOMap().
		Set("top_level_atoms", topLevel).
		Set("hints", jsonval.NewOMap().
			Set("hasMoov", seen["moov"]).
			Set("hasUdta", seen["udta"]).
			Set("hasMeta", seen["meta"]).
			Set("hasIlst", seen["ilst"])))
}

// aiffTextChunks are the chunks of an AIFF file that hold readable text. The
// copyright chunk's name is three characters padded to four with a space, which
// is how the format writes it.
const aiffCopyrightChunk = "(c) "

var aiffTextChunks = map[string]bool{
	"NAME": true, "AUTH": true, "ANNO": true, aiffCopyrightChunk: true,
}

// parseAiffBestEffort walks the chunks of an AIFF file.
func parseAiffBestEffort(b []byte, r *Report, maxText int) {
	r.addSystem("aiff_chunks")

	aiff := jsonval.NewOMap()
	textChunks := []any{}
	haveText := false
	var index []any
	title := ""

	off := 12
	for off+8 <= len(b) && len(index) < 2000 {
		id := ascii4(b, off)
		size := u32be(b, off+4)
		dataOff := off + 8
		index = append(index, jsonval.NewOMap().Set("id", id).Set("size", size).Set("offset", off))

		if aiffTextChunks[id] {
			text := safeUtf8(sliceRange(b, dataOff, dataOff+min(size, maxText)))
			textChunks = append(textChunks, jsonval.NewOMap().
				Set("id", id).Set("value", text).Set("truncated", size > maxText))
			if !haveText {
				haveText = true
			}
			aiff.Set("chunks", textChunks)
			if id == "NAME" && title == "" {
				title = text
			}
		}

		off = dataOff + size + size%2
	}

	// The index is always written, even when the file held no chunks at all.
	if index == nil {
		index = []any{}
	}
	aiff.Set("chunk_index", index[:min(len(index), 500)])
	r.raw.Set("aiff", aiff)

	r.setCommonIfEmpty("title", title)
}

// ---------------------------------------------------------- AAC / AC3 ------

var (
	aacSampleRates = []int{96000, 88200, 64000, 48000, 44100, 32000, 24000, 22050, 16000, 12000, 11025, 8000, 7350}
	aacProfiles    = []string{"Main", "LC", "SSR", "LTP"}
	aacChannels    = []string{"defined in AOT", "mono", "stereo", "3.0", "4.0", "5.0", "5.1", "7.1"}
)

// parseAacAdts reads the frame header of a raw AAC stream.
func parseAacAdts(b []byte, r *Report) {
	r.addSystem("adts_header")
	if len(b) < 7 {
		return
	}

	id := int(b[1]>>3) & 0x01
	profile := int(b[2]>>6) & 0x03
	freqIdx := int(b[2]>>2) & 0x0f
	chanCfg := int(b[2]&0x01)<<2 | int(b[3]>>6)&0x03

	version := "MPEG-4"
	if id == 1 {
		version = "MPEG-2"
	}
	profileName := fmt.Sprintf("Profile %d", profile)
	if profile < len(aacProfiles) {
		profileName = aacProfiles[profile]
	}

	r.raw.Set("aac", jsonval.NewOMap().
		Set("mpeg_version", version).
		Set("profile", profileName).
		Set("sample_rate", lookupInt(aacSampleRates, freqIdx)).
		Set("sample_rate_index", freqIdx).
		Set("channel_configuration", chanCfg).
		Set("channel_description", lookupString(aacChannels, chanCfg)))
}

var (
	ac3SampleRates = []int{48000, 44100, 32000}
	ac3Bitrates    = []int{32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 384, 448, 512, 576, 640}
	ac3AcModes     = []string{
		"2.0 (Ch1+Ch2)", "1.0 (C)", "2.0 (L R)", "3.0 (L C R)",
		"2.1 (L R S)", "3.1 (L C R S)", "2.2 (L R SL SR)", "3.2 (L C R SL SR)",
	}
)

// parseAc3 reads the bit-stream information of a Dolby Digital stream.
func parseAc3(b []byte, r *Report) {
	r.addSystem("ac3_bsi")
	if len(b) < 8 {
		return
	}

	fscod := int(b[4]>>6) & 0x03
	frmsizecod := int(b[4]) & 0x3f
	bsid := int(b[5]>>3) & 0x1f
	bsmod := int(b[5]) & 0x07
	acmod := int(b[6]>>5) & 0x07

	r.raw.Set("ac3", jsonval.NewOMap().
		Set("sample_rate", lookupInt(ac3SampleRates, fscod)).
		Set("fscod", fscod).
		Set("bitrate_kbps", lookupInt(ac3Bitrates, frmsizecod>>1)).
		Set("frmsizecod", frmsizecod).
		Set("bsid", bsid).
		Set("bsmod", bsmod).
		Set("acmod", acmod).
		Set("channel_layout", lookupString(ac3AcModes, acmod)))
}

// lookupInt returns the entry at i, or the absent value when there is none. A
// zero entry is also reported as absent, as the JavaScript's `|| null` does.
func lookupInt(table []int, i int) any {
	if i < 0 || i >= len(table) || table[i] == 0 {
		return nil
	}
	return table[i]
}

// lookupString returns the entry at i, or the absent value when there is none.
func lookupString(table []string, i int) any {
	if i < 0 || i >= len(table) || table[i] == "" {
		return nil
	}
	return table[i]
}

// ----------------------------------------------------------------- WMA ------

// The first bytes of the identifiers naming the two ASF objects that carry tags.
var (
	asfContentDescriptionGUID = []byte{0x33, 0x26, 0xb2, 0x75}
	asfExtendedContentGUID    = []byte{0x40, 0xa4, 0xd0, 0xd2}
)

// parseWmaAsf walks the header objects of an ASF file.
func parseWmaAsf(b []byte, r *Report) {
	r.addSystem("asf_header")
	if len(b) < 30 {
		return
	}

	// #nosec G115 -- an ASF header is bounded by the buffer it was read from, so
	// its declared size fits the position it is used as
	headerSize := int(u64le(b, 16))
	numObjects := u32le(b, 24)
	headerEnd := min(len(b), headerSize)

	var asf *jsonval.OMap
	ensureASF := func() *jsonval.OMap {
		if asf == nil {
			asf = jsonval.NewOMap()
			r.raw.Set("asf", asf)
		}
		return asf
	}

	objects := []any{}
	off := 30
	for i := 0; i < numObjects && off+24 <= headerEnd; i++ {
		// #nosec G115 -- checked against the end of the header below before it is
		// used to step to the next object
		objSize := int(u64le(b, off+16))
		if objSize < 24 || off+objSize > headerEnd {
			break
		}

		readAsfObject(b, r, ensureASF, off, objSize)

		guid4 := b[off : off+4]
		objects = append(objects, jsonval.NewOMap().
			Set("guid_prefix", fmt.Sprintf("%02x%02x%02x%02x", guid4[0], guid4[1], guid4[2], guid4[3])).
			Set("size", objSize))
		off += objSize
	}

	ensureASF().Set("header_objects", objects)
}

// readAsfObject reads one header object, when it is one of the two that carry
// tags. Anything else is only recorded in the list of objects.
func readAsfObject(b []byte, r *Report, ensureASF func() *jsonval.OMap, off, objSize int) {
	guid4 := b[off : off+4]
	dataOff := off + 24
	dataLen := objSize - 24

	switch {
	case bytes.Equal(guid4, asfContentDescriptionGUID) && dataLen >= 10:
		cd, fields := parseAsfContentDescription(b, dataOff)
		r.addSystem("asf_content_desc")
		ensureASF().Set("content_description", cd)
		mapCommonFrom(r, fields, asfContentToCommon)

	case bytes.Equal(guid4, asfExtendedContentGUID) && dataLen >= 2:
		ext, names := parseAsfExtContentDescription(b, dataOff, dataOff+dataLen)
		r.addSystem("asf_ext_content_desc")
		ensureASF().Set("extended_content", ext)
		for _, d := range names {
			r.setCommonIfEmptyAny(wmaToCommon[strings.ToUpper(d.key)], d.value)
		}
	}
}

// parseAsfContentDescription reads the object holding the main tags, returning
// both the report section and the fields to map the common tags from.
func parseAsfContentDescription(b []byte, off int) (*jsonval.OMap, map[string]string) {
	titleLen, authorLen := u16le(b, off), u16le(b, off+2)
	copyrightLen, descLen, ratingLen := u16le(b, off+4), u16le(b, off+6), u16le(b, off+8)

	pos := off + 10
	title := decodeUtf16LE(b, pos, titleLen)
	pos += titleLen
	author := decodeUtf16LE(b, pos, authorLen)
	pos += authorLen
	copyright := decodeUtf16LE(b, pos, copyrightLen)
	pos += copyrightLen
	description := decodeUtf16LE(b, pos, descLen)
	pos += descLen
	rating := decodeUtf16LE(b, pos, ratingLen)

	fields := map[string]string{
		"title": title, "author": author, "copyright": copyright,
		"description": description, "rating": rating,
	}
	return jsonval.NewOMap().
		Set("title", title).
		Set("author", author).
		Set("copyright", copyright).
		Set("description", description).
		Set("rating", rating), fields
}

// asfDescriptor is one entry of the extended content description. The value
// keeps whatever kind the entry declared, since a descriptor mapped onto a
// common tag contributes that value as it stands — a track number written as a
// number stays a number.
type asfDescriptor struct {
	key   string
	value any
}

// The kinds a descriptor's value can be.
const (
	asfValueString = 0
	asfValueBool   = 2
	asfValueDword  = 3
	asfValueWord   = 5
)

// parseAsfExtContentDescription reads the object holding the extra tags.
func parseAsfExtContentDescription(b []byte, off, end int) ([]any, []asfDescriptor) {
	count := u16le(b, off)
	pos := off + 2

	list := []any{}
	var named []asfDescriptor
	for i := 0; i < count && pos+6 <= end && len(list) < 5000; i++ {
		nameLen := u16le(b, pos)
		pos += 2
		if pos+nameLen > end {
			break
		}
		name := decodeUtf16LE(b, pos, nameLen)
		pos += nameLen
		valueType := u16le(b, pos)
		pos += 2
		valueLen := u16le(b, pos)
		pos += 2
		if pos+valueLen > end {
			break
		}

		var value any
		switch valueType {
		case asfValueString:
			value = decodeUtf16LE(b, pos, valueLen)
		case asfValueDword:
			value = u32le(b, pos)
		case asfValueWord:
			value = u16le(b, pos)
		case asfValueBool:
			value = u32le(b, pos) != 0
		default:
			value = fmt.Sprintf("(%d bytes, type %d)", valueLen, valueType)
		}
		pos += valueLen

		list = append(list, jsonval.NewOMap().
			Set("name", name).Set("value_type", valueType).Set("value", value))
		named = append(named, asfDescriptor{key: name, value: value})
	}
	return list, named
}
