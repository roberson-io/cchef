package ops

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/binary"
	"fmt"
	"io"
	"unicode/utf16"
)

// This file ports the SheetJS `codepage` (cptable) library that CyberChef wraps
// for its character-encoding operations. cchef reproduces cptable's decode/encode
// byte-for-byte: 140 codepages are table-driven (decode tables embedded in
// codepage_data.bin.gz, encode tables derived from them at load), and seven are
// "magic" algorithmic encodings (UTF-8/16/32, UTF-7, US-ASCII). The five ISO-2022
// charsets CyberChef lists are unsupported by cptable itself and error, exactly
// as upstream does.

//go:embed codepage_data.bin.gz
var codepageData []byte

// codepage holds one codepage's decode and (derived) encode tables. Keys and
// decoded values are UTF-16 code units.
type codepage struct {
	dec map[uint16]uint16 // byte or 2-byte code -> UTF-16 code unit
	enc map[uint16]int    // UTF-16 code unit -> 1- or 2-byte value
}

var codepages map[int]*codepage

// cpKind maps a codepage number to its dispatch kind (see cpCharsets).
var cpKind = func() map[int]string {
	m := make(map[int]string, len(cpCharsets))
	for _, c := range cpCharsets {
		m[c.cp] = c.kind
	}
	return m
}()

func init() {
	cps, err := loadCodepages(codepageData)
	if err != nil {
		panic("codepage: " + err.Error())
	}
	codepages = cps
}

// loadCodepages parses the gzipped decode-table blob into per-codepage tables,
// deriving each encode table from its decode table.
func loadCodepages(data []byte) (map[int]*codepage, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	raw, err := io.ReadAll(gz)
	if err != nil {
		return nil, err
	}
	m := make(map[int]*codepage)
	r := bytes.NewReader(raw)
	var count uint16
	if err := binary.Read(r, binary.BigEndian, &count); err != nil {
		return nil, err
	}
	for c := 0; c < int(count); c++ {
		var cp uint16
		var n uint32
		if err := binary.Read(r, binary.BigEndian, &cp); err != nil {
			return nil, err
		}
		if err := binary.Read(r, binary.BigEndian, &n); err != nil {
			return nil, err
		}
		dec := make(map[uint16]uint16, n)
		enc := make(map[uint16]int, n)
		for i := uint32(0); i < n; i++ {
			var pair [2]uint16
			if err := binary.Read(r, binary.BigEndian, &pair); err != nil {
				return nil, err
			}
			dec[pair[0]] = pair[1]
			// Derive encode: iterate in ascending code order (as stored),
			// last write wins, skipping U+FFFD — identical to cptable.
			if pair[1] != 0xFFFD {
				enc[pair[1]] = int(pair[0])
			}
		}
		m[int(cp)] = &codepage{dec: dec, enc: enc}
	}
	return m, nil
}

// isMagicKind reports whether a dispatch kind is an algorithmic (magic) encoding.
func isMagicKind(kind string) bool {
	switch kind {
	case "utf8", "utf7", "utf16le", "utf16be", "utf32le", "utf32be", "ascii":
		return true
	}
	return false
}

// cptableDecode decodes bytes in the given codepage to a string, reproducing
// cptable.utils.decode — including cptable's cached SBCS/DBCS paths, which
// differ from its general (2-byte-first) path.
func cptableDecode(cp int, data []byte) (string, error) {
	kind := cpKind[cp]
	if isMagicKind(kind) {
		return magicDecode(kind, data)
	}
	c, ok := codepages[cp]
	if !ok {
		return "", fmt.Errorf("unrecognized CP: %d", cp)
	}
	switch kind {
	case "sbcs":
		return sbcsDecode(c, data)
	case "dbcs":
		return dbcsDecode(c, data)
	default: // "table" — general path
		return tableDecode(c, data)
	}
}

// sbcsDecode is cptable's cached single-byte decoder: one byte per character.
func sbcsDecode(c *codepage, data []byte) (string, error) {
	units := make([]uint16, 0, len(data))
	for _, b := range data {
		r, ok := c.dec[uint16(b)]
		if !ok {
			return "", fmt.Errorf("unrecognized code: %d", b)
		}
		units = append(units, r)
	}
	return string(utf16.Decode(units)), nil
}

// dbcsDecode is cptable's cached double-byte decoder: a byte whose single-byte
// entry is absent (or U+FFFD) is a lead byte and consumes a trailing byte;
// unknown sequences yield U+FFFD rather than an error.
func dbcsDecode(c *codepage, data []byte) (string, error) {
	var units []uint16
	for i := 0; i < len(data); {
		b := uint16(data[i])
		if r, ok := c.dec[b]; ok {
			units = append(units, r)
			i++
			continue
		}
		// Absent single-byte entry means a lead byte. With no trailing byte the
		// 2-byte index is NaN in cptable, reading its zeroed buffer -> NUL; an
		// in-range but unmapped pair yields cptable's U+FDFF sentinel.
		if i+1 >= len(data) {
			units = append(units, 0)
		} else if r, ok := c.dec[b<<8|uint16(data[i+1])]; ok {
			units = append(units, r)
		} else {
			units = append(units, 0xFDFF)
		}
		i += 2
	}
	return string(utf16.Decode(units)), nil
}

// tableDecode is cptable's general decoder: try a 2-byte code first, then a
// single byte, else error.
func tableDecode(c *codepage, data []byte) (string, error) {
	var units []uint16
	for i := 0; i < len(data); {
		var s uint16
		matched := false
		j := 1
		if i+1 < len(data) {
			if r, ok := c.dec[uint16(data[i])<<8|uint16(data[i+1])]; ok {
				s, j, matched = r, 2, true
			}
		}
		if !matched {
			if r, ok := c.dec[uint16(data[i])]; ok {
				s, matched = r, true
			}
		}
		if !matched {
			return "", fmt.Errorf("unrecognized code: %d", data[i])
		}
		units = append(units, s)
		i += j
	}
	return string(utf16.Decode(units)), nil
}

// cptableEncode encodes a string into the given codepage, reproducing
// cptable.utils.encode. For table codepages (cached or general) unmappable code
// units become the byte 0x00.
func cptableEncode(cp int, s string) ([]byte, error) {
	kind := cpKind[cp]
	if isMagicKind(kind) {
		return magicEncode(kind, s)
	}
	c, ok := codepages[cp]
	if !ok {
		return nil, fmt.Errorf("unrecognized CP: %d", cp)
	}
	var out []byte
	for _, u := range utf16.Encode([]rune(s)) {
		w := c.enc[u] // 0 if absent (unmappable)
		if w > 255 {
			out = append(out, byte(w>>8), byte(w&255)) // #nosec G115 -- packs cptable's 1- or 2-byte code value
		} else {
			out = append(out, byte(w&255)) // #nosec G115 -- low byte of a codepage value
		}
	}
	return out, nil
}

// --- magic (algorithmic) encodings ---

var (
	utf7Base64 = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	utf7SetD   = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'(),-./:?"
)

// UTF-16 surrogate-pair constants for encoding code points above the BMP.
const (
	supplementaryBase = 0x10000 // first code point above the Basic Multilingual Plane
	surrogateHigh     = 0xD800  // high (leading) surrogate base; also the surrogate range start
	surrogateLow      = 0xDC00  // low (trailing) surrogate base
	surrogateEnd      = 0xDFFF  // last surrogate code unit
	surrogateShift    = 10      // bits carried by the high surrogate
	surrogateMask     = 0x3FF   // 10-bit mask for each surrogate half
)

// magicDecode decodes data under one of the algorithmic ("magic") encodings,
// dispatching to a per-encoding decoder.
func magicDecode(m string, data []byte) (string, error) {
	var units []uint16
	switch m {
	case "utf8":
		units = decodeUTF8(data)
	case "ascii":
		units = decodeASCII(data)
	case "utf16le":
		units = decodeUTF16(data, true)
	case "utf16be":
		units = decodeUTF16(data, false)
	case "utf32le":
		units = decodeUTF32LE(data)
	case "utf32be":
		units = decodeUTF32BE(data)
	case "utf7":
		return utf7Decode(data)
	default:
		return "", fmt.Errorf("unsupported magic: %s", m)
	}
	return string(utf16.Decode(units)), nil
}

// decodeASCII maps each byte directly to a code unit.
func decodeASCII(data []byte) []uint16 {
	var units []uint16
	for _, b := range data {
		units = append(units, uint16(b))
	}
	return units
}

// decodeUTF8 decodes a UTF-8 byte stream (skipping a leading BOM), folding each
// 1-4 byte sequence into a code point emitted via appendUTF32. The lead-byte
// thresholds (128/224/240) and continuation masks are the standard UTF-8 bit
// patterns.
func decodeUTF8(data []byte) []uint16 {
	var units []uint16
	i := 0
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		i = 3
	}
	for i < len(data) {
		var w int
		j := 1
		d := data[i]
		switch {
		case d < 128:
			w = int(d)
		case d < 224:
			w = int(d&31)<<6 + int(at(data, i+1)&63)
			j = 2
		case d < 240:
			w = int(d&15)<<12 + int(at(data, i+1)&63)<<6 + int(at(data, i+2)&63)
			j = 3
		default:
			w = int(d&7)<<18 + int(at(data, i+1)&63)<<12 + int(at(data, i+2)&63)<<6 + int(at(data, i+3)&63)
			j = 4
		}
		units = appendUTF32(units, w)
		i += j
	}
	return units
}

// decodeUTF16 decodes a UTF-16 byte stream in the given byte order, skipping a
// matching leading BOM.
func decodeUTF16(data []byte, littleEndian bool) []uint16 {
	i := 0
	if littleEndian {
		if len(data) >= 2 && data[0] == 0xFF && data[1] == 0xFE {
			i = 2
		}
	} else if len(data) >= 2 && data[0] == 0xFE && data[1] == 0xFF {
		i = 2
	}
	var units []uint16
	for ; i+1 < len(data); i += 2 {
		if littleEndian {
			units = append(units, uint16(data[i+1])<<8|uint16(data[i]))
		} else {
			units = append(units, uint16(data[i])<<8|uint16(data[i+1]))
		}
	}
	return units
}

// decodeUTF32LE decodes a little-endian UTF-32 stream, skipping a leading BOM.
func decodeUTF32LE(data []byte) []uint16 {
	i := 0
	if len(data) >= 4 && data[0] == 0xFF && data[1] == 0xFE && data[2] == 0 && data[3] == 0 {
		i = 4
	}
	var units []uint16
	for ; i < len(data); i += 4 {
		// JS computes (data[i+3]<<24) as a signed int32, so a high leading byte
		// makes the sum negative and fromCharCode keeps the low 16 bits.
		// #nosec G115 -- reproduces JS's signed int32 <<24 (overflow is intentional; see appendUTF32)
		w := int(int32(uint32(at(data, i+3))<<24)) + int(at(data, i+2))<<16 + int(at(data, i+1))<<8 + int(at(data, i))
		units = appendUTF32(units, w)
	}
	return units
}

// decodeUTF32BE decodes a big-endian UTF-32 stream, skipping a leading BOM. A
// short trailing group decodes to NUL, matching JS's undefined-byte behaviour.
func decodeUTF32BE(data []byte) []uint16 {
	i := 0
	if len(data) >= 4 && data[3] == 0xFF && data[2] == 0xFE && data[1] == 0 && data[0] == 0 {
		i = 4
	}
	var units []uint16
	for ; i < len(data); i += 4 {
		if i+3 >= len(data) {
			units = append(units, 0)
			continue
		}
		// #nosec G115 -- reproduces JS's signed int32 <<24 (overflow is intentional; see appendUTF32)
		w := int(int32(uint32(data[i])<<24)) + int(data[i+1])<<16 + int(data[i+2])<<8 + int(data[i+3])
		units = appendUTF32(units, w)
	}
	return units
}

// appendUTF32 appends a code point as UTF-16 code unit(s): a surrogate pair for
// code points above the BMP, otherwise a single unit (low 16 bits).
func appendUTF32(units []uint16, w int) []uint16 {
	if w >= supplementaryBase {
		w -= supplementaryBase
		return append(units, uint16(surrogateHigh+((w>>surrogateShift)&surrogateMask)), uint16(surrogateLow+(w&surrogateMask)))
	}
	return append(units, uint16(w)) // #nosec G115 -- low 16 bits, mirroring JS fromCharCode(ToUint16)
}

// at returns data[i] or 0 if out of range (mirrors JS out-of-bounds -> undefined
// -> 0 in the bitwise expressions).
func at(data []byte, i int) byte {
	if i < len(data) {
		return data[i]
	}
	return 0
}

// magicEncode encodes s under one of the algorithmic ("magic") encodings,
// dispatching to a per-encoding encoder over its UTF-16 code units.
func magicEncode(m string, s string) ([]byte, error) {
	units := utf16.Encode([]rune(s))
	switch m {
	case "utf8":
		return encodeUTF8(units), nil
	case "ascii":
		return encodeASCII(units), nil
	case "utf16le":
		return encodeUTF16(units, true), nil
	case "utf16be":
		return encodeUTF16(units, false), nil
	case "utf32le":
		return encodeUTF32(units, true), nil
	case "utf32be":
		return encodeUTF32(units, false), nil
	case "utf7":
		return utf7Encode(units), nil
	default:
		return nil, fmt.Errorf("unsupported magic: %s", m)
	}
}

// encodeUTF8 encodes code units as UTF-8. The 1/2/3-byte lead bytes and
// continuation masks are the standard UTF-8 patterns; the 4-byte surrogate path
// reproduces cptable's quirk of encoding (codepoint - 0x10000) rather than the
// codepoint itself.
func encodeUTF8(units []uint16) []byte {
	var out []byte
	for i := 0; i < len(units); i++ {
		w := int(units[i])
		switch {
		case w <= 0x7F:
			out = append(out, byte(w))
		case w <= 0x7FF:
			out = append(out, byte(192+(w>>6)), byte(128+(w&63)))
		case w >= surrogateHigh && w <= surrogateEnd:
			w -= surrogateHigh
			i++
			ww := int(getUnit(units, i)) - surrogateLow + (w << surrogateShift)
			out = append(out, byte(240+((ww>>18)&0x07)), byte(144+((ww>>12)&0x3F)),
				byte(128+((ww>>6)&0x3F)), byte(128+(ww&0x3F)))
		default:
			out = append(out, byte(224+(w>>12)), byte(128+((w>>6)&63)), byte(128+(w&63))) // #nosec G115 -- UTF-8 bytes from masked 6-bit groups
		}
	}
	return out
}

// encodeASCII encodes the low byte of each code unit. cptable's ascii encoder
// uses Node's Buffer 'ascii' path, not the browser-only throwing path.
func encodeASCII(units []uint16) []byte {
	out := make([]byte, 0, len(units))
	for _, w := range units {
		out = append(out, byte(w&0xff))
	}
	return out
}

// encodeUTF16 encodes code units in the given byte order.
func encodeUTF16(units []uint16, littleEndian bool) []byte {
	out := make([]byte, 0, len(units)*2)
	for _, w := range units {
		if littleEndian {
			out = append(out, byte(w&255), byte(w>>8))
		} else {
			out = append(out, byte(w>>8), byte(w&255))
		}
	}
	return out
}

// encodeUTF32 encodes code points (combining surrogate pairs) in the given byte
// order, four bytes each.
func encodeUTF32(units []uint16, littleEndian bool) []byte {
	var out []byte
	for i := 0; i < len(units); i++ {
		w := int(units[i])
		if w >= surrogateHigh && w <= surrogateEnd {
			i++
			w = supplementaryBase + ((w - surrogateHigh) << surrogateShift) + (int(getUnit(units, i)) - surrogateLow)
		}
		if littleEndian {
			out = append(out, byte(w&255), byte((w>>8)&255), byte((w>>16)&255), byte((w>>24)&255))
		} else {
			out = append(out, byte((w>>24)&255), byte((w>>16)&255), byte((w>>8)&255), byte(w&255))
		}
	}
	return out
}

func getUnit(units []uint16, i int) uint16 {
	if i < len(units) {
		return units[i]
	}
	return 0
}

// utf7Encode ports cptable's per-character UTF-7 encoder, including its bug: the
// output buffer is only 4*len bytes, so the overflow past that (a byte per
// non-direct character) is silently dropped, truncating the result.
func utf7Encode(units []uint16) []byte {
	var out []byte
	for _, u := range units {
		c := rune(u)
		if c == '+' {
			out = append(out, 0x2b, 0x2d)
			continue
		}
		if c < 0x80 && indexRune(utf7SetD, c) >= 0 {
			out = append(out, byte(c))
			continue
		}
		// two big-endian bytes of the code unit
		t0 := byte(u >> 8)
		t1 := byte(u & 0xff)
		out = append(out, 0x2b,
			utf7Base64[t0>>2],
			utf7Base64[((t0&0x03)<<4)+(t1>>4)],
			utf7Base64[(t1&0x0F)<<2],
			0x2d)
	}
	if limit := 4 * len(units); len(out) > limit {
		out = out[:limit]
	}
	return out
}

// utf7Decode ports cptable's UTF-7 decoder.
func utf7Decode(data []byte) (string, error) {
	i := utf7SkipBOM(data)
	var units []uint16
	for i < len(data) {
		if data[i] != 0x2b { // not '+'
			units = append(units, uint16(data[i]))
			i++
			continue
		}
		if at(data, i+1) == 0x2d { // "+-" -> "+"
			units = append(units, uint16('+'))
			i += 2
			continue
		}
		j := 1
		for i+j < len(data) && isUTF7Base64(data[i+j]) {
			j++
		}
		dash := 0
		if at(data, i+j) == 0x2d {
			j++
			dash = 1
		}
		tt := utf7DecodeBase64Run(data, i, j, dash)
		dec, _ := magicDecode("utf16be", tt) // utf16be decoding never errors
		units = append(units, utf16.Encode([]rune(dec))...)
		i += j
	}
	return string(utf16.Decode(units)), nil
}

// utf7SkipBOM returns the offset past an optional UTF-7 byte-order mark
// ("+/v8-", or "+/v" followed by one of 8/9/+//), or 0 when absent.
func utf7SkipBOM(data []byte) int {
	if len(data) >= 4 && data[0] == 0x2B && data[1] == 0x2F && data[2] == 0x76 {
		if len(data) >= 5 && data[3] == 0x38 && data[4] == 0x2D {
			return 5
		}
		if data[3] == 0x38 || data[3] == 0x39 || data[3] == 0x2B || data[3] == 0x2F {
			return 4
		}
	}
	return 0
}

// utf7DecodeBase64Run decodes the modified-base64 shifted run starting at i
// (spanning j bytes, with dash=1 if it ends in '-') into its raw UTF-16BE bytes.
func utf7DecodeBase64Run(data []byte, i, j, dash int) []byte {
	var tt []byte
	for l := 1; l < j-dash; {
		e1 := indexByte(utf7Base64, at(data, i+l))
		l++
		e2 := indexByte(utf7Base64, at(data, i+l))
		l++
		tt = append(tt, byte(e1<<2|e2>>4)) // #nosec G115 -- base64 sextets assemble a byte
		e3 := indexByte(utf7Base64, at(data, i+l))
		l++
		if e3 == -1 {
			break
		}
		tt = append(tt, byte((e2&15)<<4|e3>>2)) // #nosec G115 -- base64 sextets assemble a byte
		e4 := indexByte(utf7Base64, at(data, i+l))
		l++
		if e4 == -1 {
			break
		}
		if e4 < 64 {
			tt = append(tt, byte((e3&3)<<6|e4)) // #nosec G115 -- base64 sextets assemble a byte
		}
	}
	return tt
}

func isUTF7Base64(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9') || b == '+' || b == '/'
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

func indexRune(s string, r rune) int {
	for i, c := range s {
		if c == r {
			return i
		}
	}
	return -1
}
