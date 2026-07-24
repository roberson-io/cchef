package ops

import (
	"math"
	"testing"
)

func exifStream(t *testing.T, hexStr string, big bool) *bufStream {
	t.Helper()
	b := mustHex(t, hexStr)
	return newBufStream(b, 0, len(b), big)
}

func TestReadExifValueFormats(t *testing.T) {
	if v := readExifValue(1, exifStream(t, "7f", false)); v != float64(127) {
		t.Errorf("format 1 = %v", v)
	}
	if v := readExifValue(3, exifStream(t, "0102", true)); v != float64(0x0102) {
		t.Errorf("format 3 = %v", v)
	}
	if v := readExifValue(4, exifStream(t, "00000102", true)); v != float64(0x0102) {
		t.Errorf("format 4 = %v", v)
	}
	if v := readExifValue(5, exifStream(t, "0000000100000002", true)); v != (rational{1, 2}) {
		t.Errorf("format 5 = %v", v)
	}
	if v := readExifValue(6, exifStream(t, "ff", false)); v != float64(-1) {
		t.Errorf("format 6 = %v", v)
	}
	if v := readExifValue(8, exifStream(t, "0003", true)); v != float64(3) {
		t.Errorf("format 8 = %v", v)
	}
	if v := readExifValue(9, exifStream(t, "00000004", true)); v != float64(4) {
		t.Errorf("format 9 = %v", v)
	}
	if v := readExifValue(10, exifStream(t, "ffffffff00000002", true)); v != (rational{-1, 2}) {
		t.Errorf("format 10 = %v", v)
	}
	if v := readExifValue(11, exifStream(t, "3f800000", true)); v != float64(1) {
		t.Errorf("format 11 = %v", v)
	}
	if v := readExifValue(12, exifStream(t, "3ff0000000000000", true)); v != float64(1) {
		t.Errorf("format 12 BE = %v", v)
	}
	if v := readExifValue(12, exifStream(t, "000000000000f03f", false)); v != float64(1) {
		t.Errorf("format 12 LE = %v", v)
	}
}

func TestReadExifValueInvalidFormat(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for invalid format")
		} else if _, ok := r.(exifErr); !ok {
			t.Errorf("expected exifErr, got %T", r)
		}
	}()
	readExifValue(99, exifStream(t, "00", false))
}

func TestBytesPerComponentDefault(t *testing.T) {
	if bytesPerComponent(0) != 0 || bytesPerComponent(99) != 0 {
		t.Error("unknown formats should have 0 bytes per component")
	}
}

func TestBufStreamNeedPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic reading past end")
		}
	}()
	s := exifStream(t, "00", false)
	s.nextUInt32() // only 1 byte available
}

func TestSimplifyValue(t *testing.T) {
	// Single rational -> scalar division.
	if v := simplifyValue([]any{rational{1, 4}}, 5); v != float64(0.25) {
		t.Errorf("single rational = %v", v)
	}
	// Multi non-rational -> stays an array.
	v := simplifyValue([]any{float64(1), float64(2)}, 3)
	arr, ok := v.([]any)
	if !ok || len(arr) != 2 {
		t.Errorf("multi value = %v", v)
	}
	// Non-array passes through.
	if v := simplifyValue("text", 2); v != "text" {
		t.Errorf("string = %v", v)
	}
}

func TestCastDegreeValues(t *testing.T) {
	store := newExifTagStore()
	store.setIfAbsent("GPSLatitude", []any{float64(10), float64(30), float64(0)})
	store.setIfAbsent("GPSLatitudeRef", "S")
	store.setIfAbsent("GPSLongitude", []any{float64(0), float64(7), float64(30)})
	store.setIfAbsent("GPSLongitudeRef", "E")
	castDegreeValues(store)
	if v, _ := store.get("GPSLatitude"); v != float64(-10.5) {
		t.Errorf("south latitude = %v, want -10.5", v)
	}
	if v, _ := store.get("GPSLongitude"); v != float64(0.125) {
		t.Errorf("east longitude = %v, want 0.125", v)
	}

	// Missing tag and malformed (too few parts) are skipped.
	store2 := newExifTagStore()
	store2.setIfAbsent("GPSLatitude", []any{float64(10)}) // only 1 part
	castDegreeValues(store2)
	if v, _ := store2.get("GPSLatitude"); len(v.([]any)) != 1 {
		t.Error("malformed latitude should be left unchanged")
	}
}

func TestToFloat(t *testing.T) {
	if toFloat("x") != 0 || toFloat(float64(3)) != 3 {
		t.Error("toFloat")
	}
}

func TestResolveTagName(t *testing.T) {
	if n := resolveTagName(exifGPSIFD, 0x0002); n != "GPSLatitude" {
		t.Errorf("gps name = %q", n)
	}
	if n := resolveTagName(exifGPSIFD, 0x010F); n != "Make" { // GPS falls back to exif table
		t.Errorf("gps->exif fallback = %q", n)
	}
	if n := resolveTagName(exifIFD0, 0x010F); n != "Make" {
		t.Errorf("exif name = %q", n)
	}
	if n := resolveTagName(exifIFD0, 0xFFFF); n != "undefined" {
		t.Errorf("unknown name = %q", n)
	}
}

func TestExifValueString(t *testing.T) {
	if exifValueString("s") != "s" {
		t.Error("string")
	}
	if exifValueString(float64(1.5)) != "1.5" {
		t.Error("float")
	}
	if exifValueString([]any{float64(1), float64(2), float64(3)}) != "1,2,3" {
		t.Error("array join")
	}
	if exifValueString(nil) != "undefined" {
		t.Error("nil")
	}
	if exifValueString(rational{1, 2}) != "" {
		t.Error("default")
	}
}

func TestJSNumberString(t *testing.T) {
	if jsNumberString(math.NaN()) != "NaN" ||
		jsNumberString(math.Inf(1)) != "Infinity" ||
		jsNumberString(math.Inf(-1)) != "-Infinity" ||
		jsNumberString(2) != "2" {
		t.Error("jsNumberString")
	}
}

func TestParseExifDate(t *testing.T) {
	// Timezone format "2004-09-04T23:39:06-08:00" -> UTC 2004-09-05T07:39:06.
	ts, ok := parseExifDate("2004-09-04T23:39:06-08:00")
	if !ok || ts != 1094369946 {
		t.Errorf("timezone date = %v, %v", ts, ok)
	}
	// Spec format.
	if ts, ok := parseExifDate("2010:07:04 00:00:00"); !ok || ts != 1278201600 {
		t.Errorf("spec date = %v, %v", ts, ok)
	}
	// Unrecognised formats.
	if _, ok := parseExifDate("not a date"); ok {
		t.Error("expected no match for junk")
	}
	// Spec-length string with non-numeric parts -> not a valid timestamp.
	if _, ok := parseExifDate("abcd:ef:gh ij:kl:mn"); ok {
		t.Error("expected no match for non-numeric spec date")
	}
	// Spec-length string with the ':' at index 4 but no space separator.
	if _, ok := parseExifDate("2010:07:04-00:00:00"); ok {
		t.Error("expected no match for spec date without a space")
	}
	// Timezone-length string with non-numeric parts.
	if _, ok := parseExifDate("abcd-ef-ghTij:kl:mn-08:00"); ok {
		t.Error("expected no match for non-numeric timezone date")
	}
}

func TestParseDateTimePartsShort(t *testing.T) {
	if _, ok := parseDateTimeParts([]string{"2010"}, []string{"1", "2", "3"}); ok {
		t.Error("too few date parts should fail")
	}
}

func TestReadExifHeaderErrors(t *testing.T) {
	if _, ok := readExifHeader(exifStream(t, "000000000000", false)); ok {
		t.Error("non-Exif header should fail")
	}
	// "Exif\0\0" then bad TIFF byte-order mark (0x0000).
	if _, ok := readExifHeader(exifStream(t, "4578696600000000", false)); ok {
		t.Error("bad TIFF magic should fail")
	}
	// Valid MM but wrong 0x002A.
	if _, ok := readExifHeader(exifStream(t, "4578696600004d4d0000", false)); ok {
		t.Error("bad TIFF 42 marker should fail")
	}
}

func TestReadExifHeaderSafeRecovers(t *testing.T) {
	// "Exif\0\0" then truncated -> OOB panic recovered as ok=false.
	if _, ok := readExifHeaderSafe(exifStream(t, "457869660000", false)); ok {
		t.Error("truncated header should recover to false")
	}
}
