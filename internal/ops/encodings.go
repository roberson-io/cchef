package ops

import (
	"errors"
	"unicode/utf16"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/encoding/unicode/utf32"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(DecodeText{})
	core.Register(EncodeText{})
}

// textEncoding pairs a CyberChef charset name (from lib/ChrEnc.mjs, verbatim so
// recipes round-trip) with the golang.org/x/text encoding that reproduces
// CyberChef's `codepage` (cptable) behaviour. utf marks Unicode target encodings
// (UTF-8/16/32), which encode astral characters directly rather than through the
// per-code-unit fallback the byte charsets use.
type textEncoding struct {
	name string
	enc  encoding.Encoding
	utf  bool
}

// textEncodings is the subset of CyberChef's 152 charsets that x/text supports
// and that agree with cptable on graphic characters, in CyberChef's listing
// order. Charsets x/text lacks (most EBCDIC/OEM/DOS pages, ISCII, UTF-7,
// ISO-2022, Johab, Taiwan legacy, Mac pages) are omitted, as are the few that
// disagree on graphic characters (EBCDIC 1047, Mac Roman/Cyrillic). For a few
// kept charsets (ISO-8859-2..16, Windows-1255, KOI8-R/U) x/text and cptable
// differ only on bytes the standard leaves undefined — chiefly the C1 range
// 0x80-0x9F, which cptable maps to C1 control characters and x/text to U+FFFD.
// Real text (graphic characters) is byte-exact. See PLAN.md.
var textEncodings = []textEncoding{
	{"UTF-8 (65001)", unicode.UTF8, true},
	{"UTF-16LE (1200)", unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM), true},
	{"UTF-16BE (1201)", unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM), true},
	{"UTF-32LE (12000)", utf32.UTF32(utf32.LittleEndian, utf32.IgnoreBOM), true},
	{"UTF-32BE (12001)", utf32.UTF32(utf32.BigEndian, utf32.IgnoreBOM), true},
	{"IBM EBCDIC US-Canada (37)", charmap.CodePage037, false},
	{"IBM EBCDIC US-Canada (037 + Euro symbol) (1140)", charmap.CodePage1140, false},
	{"OEM United States (437)", charmap.CodePage437, false},
	{"OEM Multilingual Latin 1; Western European (DOS) (850)", charmap.CodePage850, false},
	{"OEM Latin 2; Central European (DOS) (852)", charmap.CodePage852, false},
	{"OEM Cyrillic (primarily Russian) (855)", charmap.CodePage855, false},
	{"OEM Multilingual Latin 1 + Euro symbol (858)", charmap.CodePage858, false},
	{"OEM Portuguese; Portuguese (DOS) (860)", charmap.CodePage860, false},
	{"OEM Hebrew; Hebrew (DOS) (862)", charmap.CodePage862, false},
	{"OEM French Canadian; French Canadian (DOS) (863)", charmap.CodePage863, false},
	{"OEM Nordic; Nordic (DOS) (865)", charmap.CodePage865, false},
	{"OEM Russian; Cyrillic (DOS) (866)", charmap.CodePage866, false},
	{"Windows-874 Thai (874)", charmap.Windows874, false},
	{"Windows-1250 Central European (1250)", charmap.Windows1250, false},
	{"Windows-1251 Cyrillic (1251)", charmap.Windows1251, false},
	{"Windows-1252 Latin (1252)", charmap.Windows1252, false},
	{"Windows-1253 Greek (1253)", charmap.Windows1253, false},
	{"Windows-1254 Turkish (1254)", charmap.Windows1254, false},
	{"Windows-1255 Hebrew (1255)", charmap.Windows1255, false},
	{"Windows-1256 Arabic (1256)", charmap.Windows1256, false},
	{"Windows-1257 Baltic (1257)", charmap.Windows1257, false},
	{"Windows-1258 Vietnam (1258)", charmap.Windows1258, false},
	{"ISO-8859-1 Latin 1 Western European (28591)", charmap.ISO8859_1, false},
	{"ISO-8859-2 Latin 2 Central European (28592)", charmap.ISO8859_2, false},
	{"ISO-8859-3 Latin 3 South European (28593)", charmap.ISO8859_3, false},
	{"ISO-8859-4 Latin 4 North European (28594)", charmap.ISO8859_4, false},
	{"ISO-8859-5 Latin/Cyrillic (28595)", charmap.ISO8859_5, false},
	{"ISO-8859-6 Latin/Arabic (28596)", charmap.ISO8859_6, false},
	{"ISO-8859-7 Latin/Greek (28597)", charmap.ISO8859_7, false},
	{"ISO-8859-8 Latin/Hebrew (28598)", charmap.ISO8859_8, false},
	{"ISO-8859-9 Latin 5 Turkish (28599)", charmap.ISO8859_9, false},
	{"ISO-8859-10 Latin 6 Nordic (28600)", charmap.ISO8859_10, false},
	{"ISO-8859-13 Latin 7 Baltic Rim (28603)", charmap.ISO8859_13, false},
	{"ISO-8859-14 Latin 8 Celtic (28604)", charmap.ISO8859_14, false},
	{"ISO-8859-15 Latin 9 (28605)", charmap.ISO8859_15, false},
	{"ISO-8859-16 Latin 10 (28606)", charmap.ISO8859_16, false},
	{"EUC Japanese (51932)", japanese.EUCJP, false},
	{"EUC Korean (51949)", korean.EUCKR, false},
	{"KOI8-R Russian Cyrillic (20866)", charmap.KOI8R, false},
	{"KOI8-U Ukrainian Cyrillic (21866)", charmap.KOI8U, false},
	{"Japanese Shift-JIS (932)", japanese.ShiftJIS, false},
	{"Simplified Chinese GBK (936)", simplifiedchinese.GBK, false},
	{"Korean (949)", korean.EUCKR, false},
	{"Traditional Chinese Big5 (950)", traditionalchinese.Big5, false},
}

// textEncodingNames lists the charset option values in order.
var textEncodingNames = func() []string {
	names := make([]string, len(textEncodings))
	for i, e := range textEncodings {
		names[i] = e.name
	}
	return names
}()

// textEncodingByName looks up a charset by its option value.
var textEncodingByName = func() map[string]textEncoding {
	m := make(map[string]textEncoding, len(textEncodings))
	for _, e := range textEncodings {
		m[e.name] = e
	}
	return m
}()

// DecodeText decodes bytes from a chosen character encoding into text.
type DecodeText struct{}

// Meta returns the operation metadata.
func (DecodeText) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Decode text",
		Module:      "Encodings",
		Description: "Decodes text from the chosen character encoding.",
		InfoURL:     "https://wikipedia.org/wiki/Character_encoding",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (DecodeText) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Encoding", Type: core.ArgOption, Value: textEncodingNames},
	}
}

// Run decodes the input bytes using the chosen encoding.
func (DecodeText) Run(in *core.Dish, args []any) (*core.Dish, error) {
	e, ok := textEncodingByName[args[0].(string)]
	if !ok {
		return nil, errors.New("Invalid encoding") //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}
	out, err := e.enc.NewDecoder().Bytes(in.Bytes())
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeString), nil
}

// EncodeText encodes text into a chosen character encoding.
type EncodeText struct{}

// Meta returns the operation metadata.
func (EncodeText) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Encode text",
		Module:      "Encodings",
		Description: "Encodes text into the chosen character encoding.",
		InfoURL:     "https://wikipedia.org/wiki/Character_encoding",
		InputType:   core.TypeString,
		OutputType:  core.TypeArrayBuffer,
	}
}

// Args returns the argument definitions.
func (EncodeText) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Encoding", Type: core.ArgOption, Value: textEncodingNames},
	}
}

// Run encodes the input text using the chosen encoding.
func (EncodeText) Run(in *core.Dish, args []any) (*core.Dish, error) {
	e, ok := textEncodingByName[args[0].(string)]
	if !ok {
		return nil, errors.New("Invalid encoding") //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}
	return core.NewDish(encodeText(e, in.String()), core.TypeArrayBuffer), nil
}

// encodeText encodes s, reproducing cptable's substitution of an unmappable
// UTF-16 code unit with a single 0x00 byte. The whole-string fast path matches
// cptable byte-for-byte when every character is representable; only inputs with
// unmappable characters take the per-code-unit path.
func encodeText(e textEncoding, s string) []byte {
	out, err := e.enc.NewEncoder().Bytes([]byte(s))
	if err == nil {
		return out
	}
	var buf []byte
	for _, u := range utf16.Encode([]rune(s)) {
		if u >= 0xd800 && u <= 0xdfff { // lone surrogate: unmappable
			buf = append(buf, 0)
			continue
		}
		b, err := e.enc.NewEncoder().Bytes([]byte(string(rune(u))))
		if err != nil {
			buf = append(buf, 0)
		} else {
			buf = append(buf, b...)
		}
	}
	return buf
}
