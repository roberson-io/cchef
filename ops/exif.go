package ops

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// EXIF parser ported from the npm "exif-parser" library (lib/jpeg.js, exif.js,
// simplify.js, date.js, bufferstream.js) as used by CyberChef's Extract EXIF.
// The library relies on JS exceptions for control flow; this port mirrors that
// with a scoped panic/recover (exifErr) that parseEXIF converts to an error.

// exifErr is panicked on malformed input and recovered at the parse boundary.
type exifErr struct{ msg string }

func (e exifErr) Error() string { return e.msg }

// rational is an EXIF RATIONAL/SRATIONAL value (numerator, denominator).
type rational struct{ num, den float64 }

// bufStream mirrors exif-parser's BufferStream: a cursor over the whole buffer
// with an end bound and an endianness flag. Reads are bounded by the buffer
// length (as Node's Buffer reads are), not by end.
type bufStream struct {
	buf []byte
	off int
	end int
	big bool
}

func newBufStream(buf []byte, off, length int, big bool) *bufStream {
	return &bufStream{buf: buf, off: off, end: off + length, big: big}
}

func (s *bufStream) need(n int) {
	if s.off < 0 || s.off+n > len(s.buf) {
		panic(exifErr{"Invalid EXIF data"})
	}
}

func (s *bufStream) nextUInt8() int {
	s.need(1)
	v := int(s.buf[s.off])
	s.off++
	return v
}

func (s *bufStream) nextInt8() int {
	v := s.nextUInt8()
	if v >= 0x80 {
		v -= 0x100
	}
	return v
}

func (s *bufStream) nextUInt16() int {
	s.need(2)
	var v int
	if s.big {
		v = int(s.buf[s.off])<<8 | int(s.buf[s.off+1])
	} else {
		v = int(s.buf[s.off+1])<<8 | int(s.buf[s.off])
	}
	s.off += 2
	return v
}

func (s *bufStream) nextUInt32() uint32 {
	s.need(4)
	b := s.buf[s.off : s.off+4]
	var v uint32
	if s.big {
		v = uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
	} else {
		v = uint32(b[3])<<24 | uint32(b[2])<<16 | uint32(b[1])<<8 | uint32(b[0])
	}
	s.off += 4
	return v
}

// #nosec G115 -- deliberate reinterpretation of a 32-bit value as signed (SLONG)
func (s *bufStream) nextInt32() int32 { return int32(s.nextUInt32()) }

func (s *bufStream) nextFloat() float64 {
	return float64(math.Float32frombits(s.nextUInt32()))
}

func (s *bufStream) nextDouble() float64 {
	hi := uint64(s.nextUInt32())
	lo := uint64(s.nextUInt32())
	var bits uint64
	if s.big {
		bits = hi<<32 | lo
	} else {
		// little-endian double: the first 4 bytes are the low word.
		bits = lo<<32 | hi
	}
	return math.Float64frombits(bits)
}

func (s *bufStream) nextString(n int) string {
	s.need(n)
	v := string(s.buf[s.off : s.off+n])
	s.off += n
	return v
}

func (s *bufStream) nextBuffer(n int) []byte {
	s.need(n)
	v := s.buf[s.off : s.off+n]
	s.off += n
	return v
}

func (s *bufStream) skip(n int)     { s.off += n }
func (s *bufStream) remaining() int { return s.end - s.off }
func (s *bufStream) mark() exifMark { return exifMark{parent: s, off: s.off} }
func (s *bufStream) branch(offset, length int) *bufStream {
	return &bufStream{buf: s.buf, off: s.off + offset, end: s.off + offset + length, big: s.big}
}

// exifMark captures an offset plus the parent stream, so openWithOffset reads the
// parent's endianness at call time (exif-parser sets it after marking).
type exifMark struct {
	parent *bufStream
	off    int
}

func (m exifMark) openWithOffset(offset int) *bufStream {
	off := m.off + offset
	return &bufStream{buf: m.parent.buf, off: off, end: m.parent.end, big: m.parent.big}
}

// bytesPerComponent returns the size of one component of the given EXIF format.
func bytesPerComponent(format int) int {
	switch format {
	case 1, 2, 6, 7:
		return 1
	case 3, 8:
		return 2
	case 4, 9, 11:
		return 4
	case 5, 10, 12:
		return 8
	default:
		return 0
	}
}

// readExifValue reads one component of the given format.
func readExifValue(format int, s *bufStream) any {
	switch format {
	case 1:
		return float64(s.nextUInt8())
	case 3:
		return float64(s.nextUInt16())
	case 4:
		return float64(s.nextUInt32())
	case 5:
		return rational{float64(s.nextUInt32()), float64(s.nextUInt32())}
	case 6:
		return float64(s.nextInt8())
	case 8:
		return float64(s.nextUInt16())
	case 9:
		return float64(s.nextUInt32())
	case 10:
		return rational{float64(s.nextInt32()), float64(s.nextInt32())}
	case 11:
		return s.nextFloat()
	case 12:
		return s.nextDouble()
	default:
		panic(exifErr{"Invalid format while decoding: " + strconv.Itoa(format)})
	}
}

// readExifTag reads one 12-byte IFD entry, returning its tag id, value and format.
func readExifTag(tiff exifMark, s *bufStream) (int, any, int) {
	tagType := s.nextUInt16()
	format := s.nextUInt16()
	bpc := bytesPerComponent(format)
	components := int(s.nextUInt32())
	valueBytes := bpc * components

	if valueBytes > 4 {
		s = tiff.openWithOffset(int(s.nextUInt32()))
	}

	var value any
	switch {
	case format == 2:
		str := s.nextString(components)
		if i := strings.IndexByte(str, 0); i != -1 {
			str = str[:i]
		}
		value = str
	case format == 7:
		value = s.nextBuffer(components)
	case format != 0:
		arr := make([]any, 0, components)
		for range components {
			arr = append(arr, readExifValue(format, s))
		}
		value = arr
	}

	if valueBytes < 4 {
		s.skip(4 - valueBytes)
	}
	return tagType, value, format
}

// readIFDSection reads an IFD's entries and calls iterator for each.
func readIFDSection(tiff exifMark, s *bufStream, iterator func(tagType int, value any, format int)) {
	n := s.nextUInt16()
	for range n {
		tagType, value, format := readExifTag(tiff, s)
		iterator(tagType, value, format)
	}
}

// readExifHeader validates the "Exif\0\0" + TIFF header and returns the TIFF
// origin marker. The second return is false for invalid (non-EXIF) headers.
func readExifHeader(s *bufStream) (exifMark, bool) {
	if s.nextString(6) != "Exif\x00\x00" {
		return exifMark{}, false
	}
	tiff := s.mark()
	switch s.nextUInt16() {
	case 0x4949:
		s.big = false
	case 0x4D4D:
		s.big = true
	default:
		return exifMark{}, false
	}
	if s.nextUInt16() != 0x002A {
		return exifMark{}, false
	}
	return tiff, true
}

// EXIF IFD section identifiers (exif-parser's IFD0/GPSIFD/SubIFD/InteropIFD).
const (
	exifIFD0    = 1
	exifIFD1    = 2
	exifGPSIFD  = 3
	exifInterop = 5
)

// exifParseTags parses one APP1 section, invoking iterator(section, tag, value,
// format) for each tag. Returns false for sections without a valid EXIF header.
func exifParseTags(s *bufStream, iterator func(section, tagType int, value any, format int)) bool {
	tiff, ok := readExifHeaderSafe(s)
	if !ok {
		return false
	}

	var subIfdOffset, gpsOffset, interopOffset int
	ifd0 := tiff.openWithOffset(int(s.nextUInt32()))
	readIFDSection(tiff, ifd0, func(tagType int, value any, format int) {
		switch tagType {
		case 0x8825:
			gpsOffset = int(value.([]any)[0].(float64))
		case 0x8769:
			subIfdOffset = int(value.([]any)[0].(float64))
		default:
			iterator(exifIFD0, tagType, value, format)
		}
	})

	if ifd1Offset := int(ifd0.nextUInt32()); ifd1Offset != 0 {
		readIFDSection(tiff, tiff.openWithOffset(ifd1Offset), func(t int, v any, f int) {
			iterator(exifIFD1, t, v, f)
		})
	}
	if gpsOffset != 0 {
		readIFDSection(tiff, tiff.openWithOffset(gpsOffset), func(t int, v any, f int) {
			iterator(exifGPSIFD, t, v, f)
		})
	}
	if subIfdOffset != 0 {
		readIFDSection(tiff, tiff.openWithOffset(subIfdOffset), func(t int, v any, f int) {
			if t == 0xA005 {
				interopOffset = int(v.([]any)[0].(float64))
			} else {
				iterator(exifInterop, t, v, f)
			}
		})
	}
	if interopOffset != 0 {
		readIFDSection(tiff, tiff.openWithOffset(interopOffset), func(t int, v any, f int) {
			iterator(exifInterop, t, v, f)
		})
	}
	return true
}

// readExifHeaderSafe wraps readExifHeader so a malformed header (which
// exif-parser catches) yields ok=false rather than propagating.
func readExifHeaderSafe(s *bufStream) (m exifMark, ok bool) {
	defer func() {
		if recover() != nil {
			m, ok = exifMark{}, false
		}
	}()
	return readExifHeader(s)
}

// jpegParseSections walks JPEG markers (big-endian), calling iterator for each up
// to the Start-of-Scan marker.
func jpegParseSections(s *bufStream, iterator func(markerType int, section *bufStream)) {
	s.big = true
	markerType := -1
	for s.remaining() > 0 && markerType != 0xDA {
		if s.nextUInt8() != 0xFF {
			panic(exifErr{"Invalid JPEG section offset"})
		}
		markerType = s.nextUInt8()
		// Markers with no payload (RSTn, SOS) have no length field.
		var length int
		if (markerType >= 0xD0 && markerType <= 0xD9) || markerType == 0xDA {
			length = 0
		} else {
			length = s.nextUInt16() - 2
		}
		iterator(markerType, s.branch(0, length))
		s.skip(length)
	}
}

// exifTagStore preserves tag insertion order with first-value-wins semantics.
type exifTagStore struct {
	order []string
	vals  map[string]any
}

func newExifTagStore() *exifTagStore { return &exifTagStore{vals: map[string]any{}} }

func (s *exifTagStore) setIfAbsent(name string, v any) {
	if _, ok := s.vals[name]; !ok {
		s.order = append(s.order, name)
		s.vals[name] = v
	}
}

func (s *exifTagStore) get(name string) (any, bool) {
	v, ok := s.vals[name]
	return v, ok
}

func (s *exifTagStore) set(name string, v any) { s.vals[name] = v }

// resolveTagName maps a tag id to its name, GPS section first then the EXIF
// table, matching exif-parser (unknown ids become "undefined").
func resolveTagName(section, tagType int) string {
	id := uint16(tagType) // #nosec G115 -- tagType is a 16-bit IFD tag id
	var name string
	if section == exifGPSIFD {
		name = gpsTagNames[id]
	}
	if name == "" {
		name = exifTagNames[id]
	}
	if name == "" {
		name = "undefined"
	}
	return name
}

// simplifyValue reduces rational components to numbers and unwraps single-element
// arrays, mirroring simplify.simplifyValue.
func simplifyValue(value any, format int) any {
	arr, ok := value.([]any)
	if !ok {
		return value
	}
	out := make([]any, len(arr))
	for i, v := range arr {
		if format == 10 || format == 5 {
			r := v.(rational)
			out[i] = r.num / r.den
		} else {
			out[i] = v
		}
	}
	if len(out) == 1 {
		return out[0]
	}
	return out
}

// degreeTag describes a GPS coordinate tag and its reference tag.
type degreeTag struct {
	name, refName, posVal string
}

var exifDegreeTags = []degreeTag{
	{name: "GPSLatitude", refName: "GPSLatitudeRef", posVal: "N"},
	{name: "GPSLongitude", refName: "GPSLongitudeRef", posVal: "E"},
}

// castDegreeValues converts GPS [deg, min, sec] rationals into signed decimal
// degrees, mirroring simplify.castDegreeValues.
func castDegreeValues(store *exifTagStore) {
	for _, t := range exifDegreeTags {
		v, ok := store.get(t.name)
		if !ok {
			continue
		}
		parts, ok := v.([]any)
		if !ok || len(parts) < 3 {
			continue
		}
		deg := toFloat(parts[0]) + toFloat(parts[1])/60 + toFloat(parts[2])/3600
		ref, _ := store.get(t.refName)
		refStr, _ := ref.(string)
		if refStr != t.posVal {
			deg = -deg
		}
		store.set(t.name, deg)
	}
}

func toFloat(v any) float64 {
	if f, ok := v.(float64); ok {
		return f
	}
	return 0
}

var exifDateTags = []string{"ModifyDate", "DateTimeOriginal", "CreateDate", "ModifyDate"}

// castDateValues converts EXIF date strings to UNIX timestamps, mirroring
// simplify.castDateValues.
func castDateValues(store *exifTagStore) {
	for _, name := range exifDateTags {
		v, ok := store.get(name)
		if !ok {
			continue
		}
		str, isStr := v.(string)
		if !isStr || str == "" {
			continue
		}
		if ts, ok := parseExifDate(str); ok {
			store.set(name, ts)
		}
	}
}

// parseDateTimeParts builds a UTC timestamp (seconds) from date and time parts.
func parseDateTimeParts(dateParts, timeParts []string) (float64, bool) {
	if len(dateParts) < 3 || len(timeParts) < 3 {
		return 0, false
	}
	year, ok1 := jsParseInt(dateParts[0], 10)
	month, ok2 := jsParseInt(dateParts[1], 10)
	day, ok3 := jsParseInt(dateParts[2], 10)
	hh, ok4 := jsParseInt(timeParts[0], 10)
	mm, ok5 := jsParseInt(timeParts[1], 10)
	ss, ok6 := jsParseInt(timeParts[2], 10)
	if ok1 && ok2 && ok3 && ok4 && ok5 && ok6 {
		t := time.Date(year, time.Month(month), day, hh, mm, ss, 0, time.UTC)
		return float64(t.Unix()), true
	}
	return 0, false
}

func parseDateWithSpecFormat(s string) (float64, bool) {
	parts := strings.SplitN(s, " ", 2)
	if len(parts) < 2 {
		return 0, false
	}
	return parseDateTimeParts(strings.Split(parts[0], ":"), strings.Split(parts[1], ":"))
}

func parseDateWithTimezoneFormat(s string) (float64, bool) {
	dateParts := strings.Split(s[0:10], "-")
	timeParts := strings.Split(s[11:19], ":")
	tzParts := strings.Split(s[19:25], ":")
	ts, ok := parseDateTimeParts(dateParts, timeParts)
	if !ok {
		return 0, false
	}
	tzHours, _ := jsParseInt(tzParts[0], 10)
	tzMins := 0
	if len(tzParts) > 1 {
		tzMins, _ = jsParseInt(tzParts[1], 10)
	}
	ts -= float64(tzHours*3600 + tzMins*60)
	return ts, true
}

// parseExifDate detects the "YYYY:MM:DD hh:mm:ss" and timezone-suffixed formats.
func parseExifDate(s string) (float64, bool) {
	if len(s) == 25 && s[10] == 'T' {
		return parseDateWithTimezoneFormat(s)
	}
	if len(s) == 19 && s[4] == ':' {
		return parseDateWithSpecFormat(s)
	}
	return 0, false
}

// parseEXIF parses all EXIF tags from a JPEG/TIFF byte stream.
func parseEXIF(data []byte) (store *exifTagStore, err error) {
	// CyberChef's Extract EXIF wraps the whole parse in a try/catch, so any
	// failure (bad JPEG, unexpected type) becomes an error rather than a crash.
	defer func() {
		if r := recover(); r != nil {
			store, err = nil, fmt.Errorf("%v", r)
		}
	}()

	store = newExifTagStore()
	s := newBufStream(data, 0, len(data), false)
	jpegParseSections(s, func(markerType int, section *bufStream) {
		if markerType != 0xE1 {
			return
		}
		exifParseTags(section, func(sec, tagType int, value any, format int) {
			if format == 7 {
				return // binary field, readBinaryTags disabled
			}
			if tagType == 0x0201 || tagType == 0x0202 || tagType == 0x0103 {
				return // thumbnail pointers, hidePointers enabled
			}
			value = simplifyValue(value, format)
			store.setIfAbsent(resolveTagName(sec, tagType), value)
		})
	})
	castDegreeValues(store)
	castDateValues(store)
	return store, nil
}

// exifValueString renders a simplified tag value the way JS string interpolation
// does (numbers via JS Number formatting, arrays joined by ",").
func exifValueString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return jsNumberString(x)
	case []any:
		parts := make([]string, len(x))
		for i, e := range x {
			parts[i] = exifValueString(e)
		}
		return strings.Join(parts, ",")
	case nil:
		return "undefined"
	default:
		return ""
	}
}

// jsNumberString formats a float as JS String() would (finite via jsFormatNumber,
// which is JS Number.toString for finite values; non-finite keep JS spellings).
func jsNumberString(f float64) string {
	switch {
	case math.IsNaN(f):
		return "NaN"
	case math.IsInf(f, 1):
		return "Infinity"
	case math.IsInf(f, -1):
		return "-Infinity"
	default:
		return jsFormatNumber(f)
	}
}
