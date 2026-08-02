package audio

import (
	"testing"

	"github.com/roberson-io/cchef/internal/jsonval"
)

// TestAudioByteHelpers covers the readers the container parsers are built from,
// including the reads that run off the end of the buffer. The JavaScript gets
// undefined there, which its arithmetic turns into zero and its string building
// into nothing.
func TestAudioByteHelpers(t *testing.T) {
	b := []byte{0x01, 0x02, 0x03, 0x04, 0x05}

	if got := ascii4(b, 1); got != "\x02\x03\x04\x05" {
		t.Errorf("ascii4 = %q", got)
	}
	for _, off := range []int{-1, 2, 99} {
		if got := ascii4(b, off); got != "" {
			t.Errorf("ascii4 past the end at %d = %q, want empty", off, got)
		}
	}

	if got := u32be(b, 0); got != 0x01020304 {
		t.Errorf("u32be = %#x", got)
	}
	if got := u32le(b, 0); got != 0x04030201 {
		t.Errorf("u32le = %#x", got)
	}
	if got := u16le(b, 0); got != 0x0201 {
		t.Errorf("u16le = %#x", got)
	}
	if got := u32be(b, 90); got != 0 {
		t.Errorf("u32be past the end = %d, want 0", got)
	}
	if got := u64le([]byte{1, 0, 0, 0, 2, 0, 0, 0}, 0); got != 1|2<<32 {
		t.Errorf("u64le = %#x", got)
	}
	if got := synchsafeToInt(0x00, 0x00, 0x01, 0x02); got != 130 {
		t.Errorf("synchsafeToInt = %d, want 130", got)
	}
	// The top bit of each group is not part of the number.
	if got := synchsafeToInt(0xff, 0xff, 0xff, 0xff); got != 0x0fffffff {
		t.Errorf("synchsafeToInt with every bit set = %#x", got)
	}
}

// TestAudioIndexOfAscii covers the search the container sniffing relies on.

// TestAudioIndexOfAscii covers the search the container sniffing relies on.
func TestAudioIndexOfAscii(t *testing.T) {
	b := []byte("....OpusHead....OpusTags")
	for _, tc := range []struct {
		name             string
		needle           string
		start, end, want int
	}{
		{"found", "OpusHead", 0, len(b), 4},
		{"found after the start", "Opus", 5, len(b), 16},
		{"absent", "vorbis", 0, len(b), -1},
		{"beyond the window", "OpusTags", 0, 10, -1},
		{"a window past the buffer is clamped", "OpusTags", 0, 9999, 16},
		{"a negative start is taken as the beginning", "OpusHead", -5, len(b), 4},
		{"a window with no room", "OpusHead", 20, 22, -1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := indexOfASCII(b, tc.needle, tc.start, tc.end); got != tc.want {
				t.Errorf("got %d, want %d", got, tc.want)
			}
		})
	}
}

// TestAudioDecodeText covers the four encodings an ID3 frame can name, and the
// fallback for one it cannot.

// TestAudioDecodeText covers the four encodings an ID3 frame can name, and the
// fallback for one it cannot.
func TestAudioDecodeText(t *testing.T) {
	for _, tc := range []struct {
		name     string
		bytes    []byte
		encoding int
		want     string
	}{
		{"Latin-1", []byte{0x48, 0x69, 0xe9}, textLatin1, "Hié"},
		{"UTF-16 little endian", []byte{0x48, 0x00, 0x69, 0x00}, textUTF16, "Hi"},
		{"UTF-16 big endian", []byte{0x00, 0x48, 0x00, 0x69}, textUTF16B, "Hi"},
		{"UTF-8", []byte("Hié"), textUTF8, "Hié"},
		{"a byte-order mark wins over the named order", []byte{0xff, 0xfe, 0x48, 0x00}, textUTF16B, "H"},
		{"a big-endian mark", []byte{0xfe, 0xff, 0x00, 0x48}, textUTF16, "H"},
		{"an encoding that is not defined is read as UTF-16", []byte{0x48, 0x00}, 9, "H"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := decodeText(tc.bytes, tc.encoding); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// TestAudioSafeUtf8 covers bytes that are not valid UTF-8, which stand in as the
// replacement character rather than failing the whole run.

// TestAudioSafeUtf8 covers bytes that are not valid UTF-8, which stand in as the
// replacement character rather than failing the whole run.
func TestAudioSafeUtf8(t *testing.T) {
	if got := safeUtf8([]byte("plain")); got != "plain" {
		t.Errorf("got %q", got)
	}
	got := safeUtf8([]byte{0x41, 0xff, 0x42})
	if got != "A�B" {
		t.Errorf("got %q, want %q", got, "A�B")
	}
}

// TestAudioNullTerminated covers finding the end of a field, which is one zero
// byte for the single-byte encodings and two for the UTF-16 ones.

// TestAudioNullTerminated covers finding the end of a field, which is one zero
// byte for the single-byte encodings and two for the UTF-16 ones.
func TestAudioNullTerminated(t *testing.T) {
	single := []byte{'a', 'b', 0x00, 'c'}
	value, next := nullTerminated(single, 0, textLatin1)
	if string(value) != "ab" || next != 3 {
		t.Errorf("got %q, next %d", value, next)
	}

	wide := []byte{'a', 0x00, 'b', 0x00, 0x00, 0x00, 'c', 0x00}
	value, next = nullTerminated(wide, 0, textUTF16)
	if string(value) != "a\x00b\x00" || next != 6 {
		t.Errorf("got %q, next %d", value, next)
	}

	// A field with no terminator runs to the end.
	value, next = nullTerminated([]byte{'a', 'b'}, 0, textLatin1)
	if string(value) != "ab" || next != 3 {
		t.Errorf("unterminated: got %q, next %d", value, next)
	}
	// Starting past the end reads nothing.
	if value, _ := nullTerminated(single, 99, textLatin1); value != nil {
		t.Errorf("starting past the end gave %q", value)
	}
	// A negative start is taken as the beginning.
	if value, _ := nullTerminated(single, -3, textLatin1); string(value) != "ab" {
		t.Errorf("a negative start gave %q", value)
	}
}

// TestAudioNormalizeTlen covers the length frame, which holds either whole
// milliseconds or a number of seconds to be scaled.

// TestAudioNormalizeTlen covers the length frame, which holds either whole
// milliseconds or a number of seconds to be scaled.
func TestAudioNormalizeTlen(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want int
		ok   bool
	}{
		{"214500", 214500, true},
		{"  1000  ", 1000, true},
		{"12.5", 12500, true},
		{"0", 0, true},
		{"", 0, false},
		{"not a number", 0, false},
		{"-5", 0, false},
		// The all-digits form is taken as milliseconds whatever its size; only
		// the seconds form is bounded.
		{"100000", 100000, true},
		{"100000.5", 0, false},
		{"1e6", 0, false},
	} {
		got, ok := normalizeTlen(tc.in)
		if ok != tc.ok || (ok && got != tc.want) {
			t.Errorf("normalizeTlen(%q) = %d, %v; want %d, %v", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}

// TestAudioLookupTables covers reading a value out of one of the fixed tables,
// where an index the format allows but the table does not cover reports nothing.

// TestAudioLookupTables covers reading a value out of one of the fixed tables,
// where an index the format allows but the table does not cover reports nothing.
func TestAudioLookupTables(t *testing.T) {
	if got := lookupInt(aacSampleRates, 4); got != 44100 {
		t.Errorf("got %v, want 44100", got)
	}
	for _, i := range []int{-1, len(aacSampleRates), 99} {
		if got := lookupInt(aacSampleRates, i); got != nil {
			t.Errorf("lookupInt(%d) = %v, want nothing", i, got)
		}
	}
	if got := lookupString(aacChannels, 2); got != "stereo" {
		t.Errorf("got %v, want stereo", got)
	}
	if got := lookupString(aacChannels, 99); got != nil {
		t.Errorf("got %v, want nothing", got)
	}
}

// TestAudioSliceRange covers taking a fixed-width field out of a buffer that may
// be shorter than the field.

// TestAudioSliceRange covers taking a fixed-width field out of a buffer that may
// be shorter than the field.
func TestAudioSliceRange(t *testing.T) {
	b := []byte{1, 2, 3}
	if got := sliceRange(b, 0, 2); len(got) != 2 {
		t.Errorf("got %v", got)
	}
	if got := sliceRange(b, 1, 99); len(got) != 2 {
		t.Errorf("a range past the end gave %v", got)
	}
	if got := sliceRange(b, 99, 100); got != nil {
		t.Errorf("a range starting past the end gave %v", got)
	}
}

// TestAudioIsID3FrameID covers what counts as a frame identifier, which is what
// the walk stops at when it reaches the padding after the last frame.

// TestAudioIsID3FrameID covers what counts as a frame identifier, which is what
// the walk stops at when it reaches the padding after the last frame.
func TestAudioIsID3FrameID(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want bool
	}{
		{"TIT2", true},
		{"WXXX", true},
		{"1234", true},
		{"", false},
		{"TIT", false},
		{"TIT22", false},
		{"tit2", false},
		{"TI T", false},
		{"\x00\x00\x00\x00", false},
	} {
		if got := isID3FrameID(tc.in); got != tc.want {
			t.Errorf("isID3FrameID(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

// TestAudioSniffContainer covers the formats told apart by their opening bytes,
// including the ones no sample exercises.

// TestAudioSniffContainer covers the formats told apart by their opening bytes,
// including the ones no sample exercises.
func TestAudioSniffContainer(t *testing.T) {
	for _, tc := range []struct {
		name       string
		bytes      []byte
		typ, brand string
		mime       string
	}{
		{"BW64", []byte("BW64....WAVE"), "bw64", "", "audio/wav"},
		{"an MP4 that is not audio", []byte("....ftypisom"), "mp4", "isom", "video/mp4"},
		{"an M4B audiobook", []byte("....ftypM4B "), "m4a", "M4B ", "audio/mp4"},
		{"an AIFC", []byte("FORM....AIFC"), "aiff", "AIFC", "audio/aiff"},
		{"a FORM that is not AIFF", []byte("FORM....XXXX"), "unknown", "", ""},
		{"an Ogg with no Opus header", append([]byte("OggS"), make([]byte, 40)...), "ogg", "", "audio/ogg"},
		{"too short for anything", []byte{0x00}, "unknown", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := SniffContainer(tc.bytes)
			if got.Typ != tc.typ || got.brand != tc.brand || got.mime != tc.mime {
				t.Errorf("got %+v, want type %q brand %q mime %q", got, tc.typ, tc.brand, tc.mime)
			}
		})
	}
}

// TestAudioReportBookkeeping covers the report's own accounting: a system is
// recorded once, can be taken back, and payloads are counted by what they are.

// TestAudioReportBookkeeping covers the report's own accounting: a system is
// recorded once, can be taken back, and payloads are counted by what they are.
func TestAudioReportBookkeeping(t *testing.T) {
	r := NewReport("", 0, Container{Typ: "unknown"})

	r.addSystem("one")
	r.addSystem("one")
	r.addSystem("two")
	if len(r.systems) != 2 {
		t.Errorf("recorded %v, want two systems", r.systems)
	}
	r.dropSystem("one")
	if len(r.systems) != 1 || r.systems[0] != "two" {
		t.Errorf("after dropping one: %v", r.systems)
	}
	r.dropSystem("absent")
	if len(r.systems) != 1 {
		t.Errorf("dropping an absent system changed %v", r.systems)
	}

	r.addEmbedded(jsonval.NewOMap().Set("source", "a"))
	r.addEmbedded(jsonval.NewOMap().Set("source", "b"))
	r.addEmbedded(jsonval.NewOMap().Set("source", "a"))
	if n := r.countEmbedded(func(e *jsonval.OMap) bool { return e.Value("source") == "a" }); n != 2 {
		t.Errorf("counted %d, want 2", n)
	}
}

// TestAudioTruthy covers what counts as a value worth mapping onto a common tag,
// which follows the test the JavaScript makes.

// TestAudioTruthy covers what counts as a value worth mapping onto a common tag,
// which follows the test the JavaScript makes.
func TestAudioTruthy(t *testing.T) {
	for _, tc := range []struct {
		value any
		want  bool
	}{
		{nil, false},
		{"", false},
		{"x", true},
		{0, false},
		{5, true},
		{false, false},
		{true, true},
		{1.5, true},
	} {
		if got := audioTruthy(tc.value); got != tc.want {
			t.Errorf("audioTruthy(%#v) = %v, want %v", tc.value, got, tc.want)
		}
	}
}

// TestAudioSetCommonIfEmpty covers a tag being filled in only once: the first
// source to name it wins, which is why the readers run in a fixed order.

// TestAudioSetCommonIfEmpty covers a tag being filled in only once: the first
// source to name it wins, which is why the readers run in a fixed order.
func TestAudioSetCommonIfEmpty(t *testing.T) {
	r := NewReport("", 0, Container{Typ: "unknown"})

	r.setCommonIfEmpty("title", "first")
	r.setCommonIfEmpty("title", "second")
	if got := r.common.Value("title"); got != "first" {
		t.Errorf("title = %v, want first", got)
	}
	r.setCommonIfEmpty("", "orphan")
	r.setCommonIfEmpty("artist", "")
	if got := r.common.Value("artist"); got != nil {
		t.Errorf("artist = %v, want nothing", got)
	}
}

// TestAudioApeWarnings covers the two shapes of APE tag the reader can only
// report a problem about.

// TestAudioApeWarnings covers the two shapes of APE tag the reader can only
// report a problem about.
func TestAudioApeWarnings(t *testing.T) {
	// A footer cut short by the end of the file.
	truncated := append([]byte("padding"), []byte("APETAGEX")...)
	ape := parseApeV2BestEffort(truncated)
	if ape == nil {
		t.Fatal("the truncated footer was not noticed")
	}
	if got := ape.Value("warning"); got != "APETAGEX found but footer truncated." {
		t.Errorf("warning = %v", got)
	}

	// A footer whose declared size puts the tag before the start of the file.
	bad := append([]byte("padding"), []byte("APETAGEX")...)
	bad = append(bad, make([]byte, 24)...)
	// size field, at offset 12 of the footer, larger than everything before it
	copy(bad[len("padding")+12:], []byte{0xff, 0xff, 0x00, 0x00})
	ape = parseApeV2BestEffort(bad)
	if ape == nil {
		t.Fatal("the out-of-bounds tag was not noticed")
	}
	if got := ape.Value("warning"); got != "APEv2 bounds invalid (non-standard placement)." {
		t.Errorf("warning = %v", got)
	}

	if parseApeV2BestEffort([]byte("no tag here")) != nil {
		t.Error("a file with no APE tag reported one")
	}
}

// TestAudioGeobJumbf covers the other content type that marks an encapsulated
// object as carrying content credentials.

// TestAudioGeobJumbf covers the other content type that marks an encapsulated
// object as carrying content credentials.
func TestAudioGeobJumbf(t *testing.T) {
	r := NewReport("", 0, Container{Typ: "mp3"})
	payload := append([]byte{0x00}, []byte("application/x-jumbf\x00name\x00desc\x00")...)
	payload = append(payload, 1, 2, 3, 4)

	processGeobFrame(id3Frame{id: "GEOB", data: payload}, jsonval.NewOMap(), r)

	if got := r.provenanceC2PA().Value("present"); got != true {
		t.Errorf("present = %v, want true", got)
	}
	embedding, _ := r.provenanceC2PA().Value("embedding").([]any)
	if len(embedding) != 1 {
		t.Errorf("recorded %d embeddings, want 1", len(embedding))
	}
}

// TestAudioDecodeFrameGuards covers the frames too short to hold what they
// declare.

// TestAudioDecodeFrameGuards covers the frames too short to hold what they
// declare.
func TestAudioDecodeFrameGuards(t *testing.T) {
	if got := decodeTxxx([]byte{0x00}); got != nil {
		t.Errorf("a one-byte TXXX gave %v, want nothing", got)
	}
	if got := decodeCommFrame([]byte{0x00, 'e', 'n'}); got != nil {
		t.Errorf("a short COMM gave %v, want nothing", got)
	}
	// A description that runs to the end leaves no value behind it.
	if got := decodeTxxx([]byte{0x00, 'k', 'e', 'y'}); got == nil {
		t.Error("a TXXX with no value was refused")
	}
	if got := decodeCommFrame([]byte{0x00, 'e', 'n', 'g', 'd', 'e', 's', 'c'}); got == nil {
		t.Error("a COMM with no body was refused")
	}
	if !isAllDigits("123") || isAllDigits("") || isAllDigits("12a") {
		t.Error("isAllDigits does not agree with what a digit run is")
	}
}

// TestAudioRunIgnoresAnUnusableLimit covers the embedded-text limit being given
// a value that is not a number, which falls back to the default.

// TestAudioNullTerminatedWideRunsOut covers a UTF-16 field with no terminator
// before the end of the frame. The scan steps two bytes at a time, so it stops
// at the last whole character pair rather than taking the odd byte after it.
func TestAudioNullTerminatedWideRunsOut(t *testing.T) {
	value, next := nullTerminated([]byte{'a', 0x00, 'b'}, 0, textUTF16)
	if string(value) != "a\x00" {
		t.Errorf("got %q, want the one whole character", value)
	}
	if next != 4 {
		t.Errorf("next = %d, want 4", next)
	}
}

// TestAudioParserGuards covers the readers being given files too short or too
// odd to hold what they describe. None of these is an error: the report simply
// carries less.

// TestAudioParserGuards covers the readers being given files too short or too
// odd to hold what they describe. None of these is an error: the report simply
// carries less.
func TestAudioParserGuards(t *testing.T) {
	report := func() *Report {
		return NewReport("", 0, Container{Typ: "unknown"})
	}

	t.Run("an AC3 stream too short for its header", func(t *testing.T) {
		r := report()
		parseAc3([]byte{0x0b, 0x77}, r)
		if _, ok := r.raw.Get("ac3"); ok {
			t.Error("fields were reported from a header that is not there")
		}
	})

	t.Run("an ASF file too short for its header", func(t *testing.T) {
		r := report()
		parseWmaAsf([]byte{0x30, 0x26}, r)
		if _, ok := r.raw.Get("asf"); ok {
			t.Error("objects were reported from a header that is not there")
		}
	})

	t.Run("a FLAC picture block that is all header", func(t *testing.T) {
		r := report()
		// A picture block declaring lengths longer than the block itself.
		block := []byte{0x80 | 6, 0x00, 0x00, 0x08}
		block = append(block, make([]byte, 8)...)
		parseFlac(append([]byte("fLaC"), block...), r, 1024)
		if len(r.embedded) != 1 {
			t.Errorf("recorded %d payloads, want 1", len(r.embedded))
		}
	})

	t.Run("an AIFF holding no chunks", func(t *testing.T) {
		r := report()
		parseAiffBestEffort([]byte("FORM\x00\x00\x00\x04AIFF"), r, 1024)
		aiff, _ := r.raw.Value("aiff").(*jsonval.OMap)
		if aiff == nil {
			t.Fatal("no section was written")
		}
		index, _ := aiff.Value("chunk_index").([]any)
		if index == nil || len(index) != 0 {
			t.Errorf("chunk_index = %v, want an empty list", index)
		}
	})

	t.Run("an MP4 whose atoms run past the end", func(t *testing.T) {
		r := report()
		parseMp4BestEffort([]byte("\x00\x00\xff\xffftyp"), r)
		mp4, _ := r.raw.Value("mp4").(*jsonval.OMap)
		atoms, _ := mp4.Value("top_level_atoms").([]any)
		if len(atoms) != 1 {
			t.Errorf("found %d atoms, want 1", len(atoms))
		}
	})

	t.Run("an APE tag whose items run past the footer", func(t *testing.T) {
		// A well-formed footer, but item lengths that overrun.
		body := []byte{0xff, 0xff, 0x00, 0x00, 0, 0, 0, 0}
		body = append(body, []byte("Key\x00")...)
		tag := append(make([]byte, 8), body...)
		footer := append([]byte("APETAGEX"), make([]byte, 24)...)
		copy(footer[12:], []byte{byte(len(body) + 32), 0, 0, 0})
		ape := parseApeV2BestEffort(append(tag, footer...))
		if ape == nil {
			t.Fatal("the tag was not found")
		}
		items, _ := ape.Value("items").([]any)
		if len(items) != 0 {
			t.Errorf("read %d items from a tag that overruns", len(items))
		}
	})
}

// TestAudioApeWithHeader covers the layout an APE tag is normally written in: a
// header, the items, then a footer. The items are only reached when the search
// for the tag lands on the footer rather than the header, which happens once the
// tag is larger than the 32 KB window searched back from the end of the file —
// so the tag below is built past that size on purpose.

// TestAudioApeWithHeader covers the layout an APE tag is normally written in: a
// header, the items, then a footer. The items are only reached when the search
// for the tag lands on the footer rather than the header, which happens once the
// tag is larger than the 32 KB window searched back from the end of the file —
// so the tag below is built past that size on purpose.
func TestAudioApeWithHeader(t *testing.T) {
	item := func(key string, value []byte) []byte {
		out := make([]byte, 8)
		out[0] = byte(len(value))
		out = append(out, key...)
		out = append(out, 0x00)
		return append(out, value...)
	}
	body := append(item("Artist", []byte("An Artist")), item("Year", []byte("2026"))...)
	// Past the search window, so the header falls outside it and the footer is
	// what gets found. The padding reads as an item with no key, which stops the
	// walk cleanly.
	body = append(body, make([]byte, 33000-len(body))...)

	block := func() []byte {
		out := append([]byte("APETAGEX"), make([]byte, 24)...)
		out[8], out[9] = 0xd0, 0x07 // version 2000
		// The recorded size covers the header, the items and the footer, which
		// is what lets the reader step back from the footer to the items.
		size := len(body) + 64
		out[12] = byte(size)
		out[13] = byte(size >> 8)
		out[14] = byte(size >> 16)
		out[16] = 2 // item count
		return out
	}

	tag := append(block(), body...)
	tag = append(tag, block()...)

	ape := parseApeV2BestEffort(tag)
	if ape == nil {
		t.Fatal("the tag was not found")
	}
	if warning, ok := ape.Get("warning"); ok {
		t.Fatalf("the tag was refused: %v", warning)
	}
	items, _ := ape.Value("items").([]any)
	if len(items) != 2 {
		t.Fatalf("read %d items, want 2", len(items))
	}
	first, _ := items[0].(*jsonval.OMap)
	if first.Value("key") != "Artist" || first.Value("value") != "An Artist" {
		t.Errorf("first item = %v / %v", first.Value("key"), first.Value("value"))
	}
	second, _ := items[1].(*jsonval.OMap)
	if second.Value("key") != "Year" || second.Value("value") != "2026" {
		t.Errorf("second item = %v / %v", second.Value("key"), second.Value("value"))
	}
}

// TestAudioMp4StopsAtAnImpossibleAtom covers an atom whose declared size is
// smaller than the eight bytes that declare it, which ends the walk.

// TestAudioMp4StopsAtAnImpossibleAtom covers an atom whose declared size is
// smaller than the eight bytes that declare it, which ends the walk.
func TestAudioMp4StopsAtAnImpossibleAtom(t *testing.T) {
	r := NewReport("", 0, Container{Typ: "m4a"})
	buf := append([]byte("\x00\x00\x00\x10ftypM4A "), make([]byte, 8)...)
	buf = append(buf, []byte("\x00\x00\x00\x02free")...)

	parseMp4BestEffort(buf, r)

	mp4, _ := r.raw.Value("mp4").(*jsonval.OMap)
	atoms, _ := mp4.Value("top_level_atoms").([]any)
	if len(atoms) != 1 {
		t.Errorf("found %d atoms, want just the first", len(atoms))
	}
}

// TestAudioAsfStopsAtABadObject covers a header object whose declared size
// cannot be right, which ends the walk over them.

// TestAudioAsfStopsAtABadObject covers a header object whose declared size
// cannot be right, which ends the walk over them.
func TestAudioAsfStopsAtABadObject(t *testing.T) {
	header := func(objects []byte, count int) []byte {
		out := make([]byte, 30)
		copy(out, []byte{0x30, 0x26, 0xb2, 0x75, 0x8e, 0x66, 0xcf, 0x11})
		size := 30 + len(objects)
		out[16] = byte(size)
		out[17] = byte(size >> 8)
		out[24] = byte(count)
		return append(out, objects...)
	}

	// One object declaring a size smaller than an object header.
	tiny := make([]byte, 24)
	tiny[16] = 4

	r := NewReport("", 0, Container{Typ: "wma"})
	parseWmaAsf(header(tiny, 1), r)

	asf, _ := r.raw.Value("asf").(*jsonval.OMap)
	objects, _ := asf.Value("header_objects").([]any)
	if len(objects) != 0 {
		t.Errorf("recorded %d objects, want none", len(objects))
	}
}

// TestAudioAsfExtStopsAtATruncatedDescriptor covers the two places a descriptor
// can claim more bytes than the object holds.

// TestAudioAsfExtStopsAtATruncatedDescriptor covers the two places a descriptor
// can claim more bytes than the object holds.
func TestAudioAsfExtStopsAtATruncatedDescriptor(t *testing.T) {
	le16 := func(v int) []byte { return []byte{byte(v), byte(v >> 8)} }

	t.Run("a name longer than the object", func(t *testing.T) {
		// Long enough that a descriptor is attempted at all, but the name it
		// declares reaches past the end.
		b := append(le16(1), le16(999)...)
		b = append(b, 0, 0, 0, 0)
		got, _ := parseAsfExtContentDescription(b, 0, len(b))
		if len(got) != 0 {
			t.Errorf("read %d descriptors, want none", len(got))
		}
	})

	t.Run("a value longer than the object", func(t *testing.T) {
		name := []byte{'A', 0x00}
		b := le16(1)
		b = append(b, le16(len(name))...)
		b = append(b, name...)
		b = append(b, le16(asfValueString)...)
		b = append(b, le16(999)...)
		got, _ := parseAsfExtContentDescription(b, 0, len(b))
		if len(got) != 0 {
			t.Errorf("read %d descriptors, want none", len(got))
		}
	})
}
