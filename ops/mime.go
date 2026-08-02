package ops

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/codepage"
	"github.com/roberson-io/cchef/internal/jsnum"
)

func init() {
	core.Register(MIMEDecoding{})
}

// CyberChef surfaces these OperationError messages verbatim.
// The codepages the three supported charset families decode through. The
// ISO-8859 parts sit consecutively from one above the base, so the part number
// gives the codepage directly.
const (
	mimeCodepageUTF8        = 65001
	mimeCodepageASCII       = 20127
	mimeISO8859BaseCodepage = 28590
)

var (
	errMIMEUnhandledCharset = errors.New("Unhandled Charset")        //nolint:staticcheck // verbatim CyberChef message
	errMIMEEncodedWord      = errors.New("Incorrectly Encoded Word") //nolint:staticcheck // verbatim CyberChef message
)

// MIMEDecoding decodes RFC 2047 MIME encoded-word header extensions.
type MIMEDecoding struct{}

// Meta returns the operation metadata.
func (MIMEDecoding) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "MIME Decoding",
		Module:      "Default",
		Description: "Enables the decoding of MIME message header extensions for non-ASCII text",
		InfoURL:     "https://tools.ietf.org/html/rfc2047",
		InputType:   core.TypeByteArray,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (MIMEDecoding) Args() []core.ArgDef { return nil }

// Run decodes MIME encoded-word header extensions.
func (MIMEDecoding) Run(in *core.Dish, _ []any) (*core.Dish, error) {
	text := mimeByteArrayToUtf8(in.Bytes())
	text = strings.ReplaceAll(text, "\r\n", "\n")
	out, err := mimeDecodeHeaders(text)
	if err != nil {
		return nil, err
	}
	return core.NewDish([]byte(out), core.TypeString), nil
}

// mimeByteArrayToUtf8 ports Utils.byteArrayToUtf8: valid UTF-8 bytes decode as a
// UTF-8 string, and anything else falls back to a byte-per-character (Latin-1)
// reading.
func mimeByteArrayToUtf8(b []byte) string {
	if utf8.Valid(b) {
		return string(b)
	}
	return mimeLatin1(b)
}

// mimeLatin1 reads each byte as the code point of the same value.
func mimeLatin1(b []byte) string {
	r := make([]rune, len(b))
	for i, c := range b {
		r[i] = rune(c)
	}
	return string(r)
}

// mimeWord is one located RFC 2047 encoded word (=?charset?enc?text?=): the
// index it starts at, its charset and encoding byte, the raw encoded text, and
// the index just past its closing "?=".
type mimeWord struct {
	start   int
	charset string
	enc     rune
	text    []rune
	end     int
}

// mimeLocateWord finds the next encoded word in header, returning ok=false when
// no complete word is present (mirroring the loop's original break conditions).
func mimeLocateWord(header []rune) (mimeWord, bool) {
	start := mimeIndexOf(header, "=?")
	if start == -1 {
		return mimeWord{}, false
	}
	cur := start + 2

	idx := mimeIndexOf(header[cur:], "?")
	if idx == -1 {
		return mimeWord{}, false
	}
	charset := string(header[cur : cur+idx])
	cur += idx + 1

	if len(header) < cur+len("Q??=") {
		return mimeWord{}, false
	}
	enc := header[cur]
	cur++
	if header[cur] != '?' {
		return mimeWord{}, false
	}
	cur++

	j := mimeIndexOf(header[cur:], "?=")
	if j == -1 {
		return mimeWord{}, false
	}
	return mimeWord{
		start:   start,
		charset: charset,
		enc:     enc,
		text:    header[cur : cur+j],
		end:     cur + j + 2,
	}, true
}

// mimeDecodeHeaders ports MIMEDecoding.decodeHeaders: it finds RFC 2047 encoded
// words (=?charset?encoding?text?=), decodes each, and drops the whitespace
// between adjacent encoded words.
func mimeDecodeHeaders(headerStr string) (string, error) {
	full := []rune(headerStr)
	i := mimeIndexOf(full, "=?")
	if i == -1 {
		return headerStr, nil
	}

	var decoded strings.Builder
	decoded.WriteString(string(full[:i]))
	header := full[i:]
	isBetweenWords := false

	for {
		w, ok := mimeLocateWord(header)
		if !ok {
			break
		}

		var data []byte
		var err error
		switch unicode.ToLower(w.enc) {
		case 'b':
			data, err = fromBase64(string(w.text), stdBase64Alphabet, true, false)
			if err != nil {
				return "", err
			}
		case 'q':
			data, err = mimeParseQEncodedWord(w.text)
			if err != nil {
				return "", err
			}
		default:
			isBetweenWords = false
			decoded.WriteString(string(mimeSlice(header, 0, w.start+2)))
			header = mimeSliceFrom(header, w.start+2)
			data = mimeRunesToBytes(w.text)
		}

		if w.start > 0 && (!isBetweenWords || runeSearchNonWS(mimeSlice(header, 0, w.start)) > -1) {
			decoded.WriteString(string(mimeSlice(header, 0, w.start)))
		}
		s, err := mimeConvertFromCharset(w.charset, data)
		if err != nil {
			return "", err
		}
		decoded.WriteString(s)

		header = mimeSliceFrom(header, w.end)
		isBetweenWords = true
	}

	if len(header) > 0 {
		decoded.WriteString(string(header))
	}
	return decoded.String(), nil
}

// mimeParseQEncodedWord ports parseQEncodedWord: "_" becomes a space, "=XX" a
// hex byte, printable ASCII and CR/LF/TAB pass through, and anything else is an
// error. The result is the raw bytes to be charset-decoded.
func mimeParseQEncodedWord(word []rune) ([]byte, error) {
	var out []byte
	for i := 0; i < len(word); i++ {
		c := word[i]
		switch {
		case c == '_':
			out = append(out, ' ')
		case c == '=':
			if i+2 >= len(word) {
				return nil, errMIMEEncodedWord
			}
			if b, ok := hexByte(word[i+1], word[i+2]); ok {
				out = append(out, b)
			}
			i += 2
		case (c >= ' ' && c <= '~') || c == '\n' || c == '\r' || c == '\t':
			out = append(out, byte(c)) // #nosec G115 -- guarded to printable ASCII / CR-LF-TAB, all < 0x80
		default:
			return nil, errMIMEEncodedWord
		}
	}
	return out, nil
}

// mimeConvertFromCharset ports convertFromCharset: it decodes the given bytes
// for UTF-8, US-ASCII and the ISO-8859-* family, throwing on anything else.
func mimeConvertFromCharset(charset string, data []byte) (string, error) {
	charset = strings.ToLower(charset)
	parts := strings.Split(charset, "-")
	switch {
	case len(parts) == 2 && parts[0] == "utf" && charset == "utf-8":
		return codepage.Decode(mimeCodepageUTF8, data)
	case len(parts) == 2 && charset == "us-ascii":
		return codepage.Decode(mimeCodepageASCII, data)
	case len(parts) == 3 && parts[0] == "iso" && parts[1] == "8859":
		if n, ok := parseLeadingInt(parts[2]); ok && n >= 1 && n <= 16 {
			// The ISO-8859 parts sit consecutively in the codepage numbering,
			// so the part number gives the codepage directly. Part 12 was never
			// standardized and has no codepage, which the lookup reports.
			return codepage.Decode(mimeISO8859BaseCodepage+n, data)
		}
	}
	return "", errMIMEUnhandledCharset
}

// mimeRunesToBytes takes the low byte of each rune, matching cptable's treatment
// of a string as a byte sequence for the unknown-encoding path.
func mimeRunesToBytes(r []rune) []byte {
	b := make([]byte, len(r))
	for i, c := range r {
		b[i] = byte(c) // #nosec G115 -- low byte only, matching cptable's charCode &0xFF
	}
	return b
}

// --- small helpers ---

// mimeIndexOf returns the index of the first occurrence of the ASCII sub within r,
// or -1, mirroring JavaScript String.indexOf on UTF-16 for BMP text.
func mimeIndexOf(r []rune, sub string) int {
	s := []rune(sub)
	if len(s) == 0 {
		return 0
	}
	for i := 0; i+len(s) <= len(r); i++ {
		match := true
		for k, c := range s {
			if r[i+k] != c {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// mimeSlice clamps like JavaScript String.slice for the non-negative indices used
// here: out-of-range bounds are clamped and an inverted range yields empty.
func mimeSlice(r []rune, start, end int) []rune {
	if start < 0 {
		start = 0
	}
	if end > len(r) {
		end = len(r)
	}
	if start >= end {
		return nil
	}
	return r[start:end]
}

// mimeSliceFrom is mimeSlice(r, start, len(r)).
func mimeSliceFrom(r []rune, start int) []rune {
	if start >= len(r) {
		return nil
	}
	if start < 0 {
		start = 0
	}
	return r[start:]
}

// runeSearchNonWS returns the index of the first non-whitespace rune (JS \S), or
// -1 if there is none.
func runeSearchNonWS(r []rune) int {
	for i, c := range r {
		if !jsnum.IsSpace(c) {
			return i
		}
	}
	return -1
}

// hexByte parses two hex-digit runes into a byte, reporting whether both were
// valid hex.
func hexByte(hi, lo rune) (byte, bool) {
	h, ok1 := hexNibble(hi)
	l, ok2 := hexNibble(lo)
	if !ok1 || !ok2 {
		return 0, false
	}
	return h<<4 | l, true
}

func hexNibble(c rune) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return byte(c - '0'), true
	case c >= 'a' && c <= 'f':
		return byte(c-'a') + 10, true
	case c >= 'A' && c <= 'F':
		return byte(c-'A') + 10, true
	}
	return 0, false
}

// parseLeadingInt parses the leading decimal digits of s (like JS parseInt),
// reporting false if there are none.
func parseLeadingInt(s string) (int, bool) {
	n := 0
	found := false
	for _, c := range s {
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int(c-'0')
		found = true
	}
	return n, found
}
