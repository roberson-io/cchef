package audio

import (
	"bytes"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// Helpers for reading the byte layouts the audio containers are built from.
// Ported from CyberChef's src/core/lib/AudioBytes.mjs. Reads that run off the
// end of the buffer give zero or the empty string rather than failing, which is
// how the JavaScript behaves when it indexes past a typed array.

// ascii4 reads the four-character tag that names a chunk or atom.
func ascii4(b []byte, off int) string {
	if off < 0 || off+4 > len(b) {
		return ""
	}
	return string([]rune{rune(b[off]), rune(b[off+1]), rune(b[off+2]), rune(b[off+3])})
}

// indexOfASCII returns the first position of s in b at or after start and before
// end, or -1 if it is not there.
func indexOfASCII(b []byte, s string, start, end int) int {
	if start < 0 {
		start = 0
	}
	if end > len(b) {
		end = len(b)
	}
	limit := end - len(s)
	if limit < start {
		return -1
	}
	i := bytes.Index(b[start:end], []byte(s))
	if i < 0 {
		return -1
	}
	return start + i
}

// audioByteAt returns the byte at off, or zero past either end of the buffer.
func audioByteAt(b []byte, off int) int {
	if off < 0 || off >= len(b) {
		return 0
	}
	return int(b[off])
}

// u32be reads a four-byte big-endian number.
func u32be(b []byte, off int) int {
	return audioByteAt(b, off)<<24 | audioByteAt(b, off+1)<<16 |
		audioByteAt(b, off+2)<<8 | audioByteAt(b, off+3)
}

// u32le reads a four-byte little-endian number.
func u32le(b []byte, off int) int {
	return audioByteAt(b, off) | audioByteAt(b, off+1)<<8 |
		audioByteAt(b, off+2)<<16 | audioByteAt(b, off+3)<<24
}

// u16le reads a two-byte little-endian number.
func u16le(b []byte, off int) int {
	return audioByteAt(b, off) | audioByteAt(b, off+1)<<8
}

// u64le reads an eight-byte little-endian number.
func u64le(b []byte, off int) uint64 {
	// #nosec G115 -- each half is a four-byte field, so narrowing to uint32 is
	// what reading it means; the JavaScript coerces the same way
	return uint64(uint32(u32le(b, off))) | uint64(uint32(u32le(b, off+4)))<<32
}

// synchsafeToInt reads the seven-bit groups an ID3 length is written as.
func synchsafeToInt(b0, b1, b2, b3 int) int {
	return (b0&0x7f)<<21 | (b1&0x7f)<<14 | (b2&0x7f)<<7 | b3&0x7f
}

// The text encodings an ID3 frame can name, in the order the format numbers
// them.
const (
	textLatin1 = 0
	textUTF16  = 1
	textUTF16B = 2
	textUTF8   = 3
)

// decodeText reads bytes under one of the encodings an ID3 frame can name. An
// encoding outside the four defined ones is read as UTF-16, as the JavaScript's
// lookup falls back to.
func decodeText(b []byte, encoding int) string {
	switch encoding {
	case textLatin1:
		return decodeLatin1(b)
	case textUTF16B:
		return audioDecodeUTF16(b, false)
	case textUTF8:
		return safeUtf8(b)
	default:
		return audioDecodeUTF16(b, true)
	}
}

// decodeLatin1 reads each byte as the character of that code point.
func decodeLatin1(b []byte) string {
	var out strings.Builder
	for _, c := range b {
		out.WriteRune(rune(c))
	}
	return out.String()
}

// decodeUTF16 reads pairs of bytes as characters, in the given byte order. A
// byte-order mark at the start overrides that order, as a decoder does.
func audioDecodeUTF16(b []byte, little bool) string {
	if len(b) >= 2 {
		switch {
		case b[0] == 0xff && b[1] == 0xfe:
			b, little = b[2:], true
		case b[0] == 0xfe && b[1] == 0xff:
			b, little = b[2:], false
		}
	}
	units := make([]uint16, 0, len(b)/2)
	for i := 0; i+1 < len(b); i += 2 {
		if little {
			units = append(units, uint16(b[i])|uint16(b[i+1])<<8)
		} else {
			units = append(units, uint16(b[i])<<8|uint16(b[i+1]))
		}
	}
	return string(utf16.Decode(units))
}

// safeUtf8 reads bytes as UTF-8, putting the replacement character in place of
// anything that is not valid rather than refusing the whole run.
func safeUtf8(b []byte) string {
	if utf8.Valid(b) {
		return string(b)
	}
	var out strings.Builder
	for len(b) > 0 {
		r, size := utf8.DecodeRune(b)
		out.WriteRune(r)
		b = b[size:]
	}
	return out.String()
}

// stripNullsAndTrim removes the null bytes a fixed-width field is padded with
// and the space either side of what is left.
func stripNullsAndTrim(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(s, "\x00", ""))
}

// decodeLatin1Trim reads a fixed-width Latin-1 field.
func decodeLatin1Trim(b []byte) string {
	return stripNullsAndTrim(decodeLatin1(b))
}

// decodeUtf16LE reads a counted UTF-16 field, of the kind an ASF object holds.
func decodeUtf16LE(b []byte, off, length int) string {
	if length <= 0 || off < 0 || off+length > len(b) {
		return ""
	}
	return stripNullsAndTrim(audioDecodeUTF16(b[off:off+length], true))
}

// nullTerminated returns the bytes up to the next terminator at or after start,
// and where the field after it begins. A UTF-16 field is terminated by two zero
// bytes rather than one.
func nullTerminated(b []byte, start, encoding int) (value []byte, next int) {
	if start < 0 {
		start = 0
	}
	if start > len(b) {
		return nil, len(b) + 1
	}
	if encoding != textUTF16 && encoding != textUTF16B {
		i := start
		for i < len(b) && b[i] != 0x00 {
			i++
		}
		return b[start:i], i + 1
	}
	// The scan steps two bytes at a time and stops while there is still a whole
	// pair left, so it can never carry i past the end of the buffer.
	i := start
	for i+1 < len(b) && (b[i] != 0x00 || b[i+1] != 0x00) {
		i += 2
	}
	return b[start:i], i + 2
}
