package ops

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(HexToPEM{})
	core.Register(PEMToHex{})
}

const pemBase64Map = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

var (
	pemWhitespace = regexp.MustCompile(`\s`)
	pemBeginRe    = regexp.MustCompile(`-----BEGIN ([A-Z][A-Z ]+[A-Z])-----`)
)

// HexToPEM wraps a hex-encoded DER blob in PEM armor.
type HexToPEM struct{}

// Meta returns the operation metadata.
func (HexToPEM) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Hex to PEM",
		Module:      "PublicKey",
		Description: "Converts a hexadecimal DER (Distinguished Encoding Rules) string into PEM (Privacy Enhanced Mail) format.",
		InfoURL:     "https://wikipedia.org/wiki/Privacy-Enhanced_Mail",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (HexToPEM) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Header string", Type: core.ArgString, Value: "CERTIFICATE"},
	}
}

// Run wraps the hex DER as PEM. Ported from CyberChef HexToPEM.mjs, which calls
// jsrsasign's getPEMStringFromHex: the hex is base64-encoded, wrapped at 64
// characters, and armored with CRLF line endings and a trailing CRLF.
func (HexToPEM) Run(in *core.Dish, args []any) (*core.Dish, error) {
	header := args[0].(string)
	clean := pemWhitespace.ReplaceAllString(in.String(), "")
	body := wrapBase64(hexToBase64(clean), 64)
	out := "-----BEGIN " + header + "-----\r\n" + body + "\r\n-----END " + header + "-----\r\n"
	return core.NewDish([]byte(out), core.TypeString), nil
}

// hexToBase64 base64-encodes a hex string the way jsrsasign does: each 2-hex-digit
// group becomes a byte (8 bits) via JavaScript parseInt semantics (a leading
// partial parse, non-hex -> 0), and a trailing odd digit contributes a 4-bit
// nibble. The resulting bit stream is then base64-encoded with '=' padding.
//
// This reproduces CyberChef byte-for-byte for all well-formed (fully hexadecimal)
// input, and is lenient rather than erroring on stray characters. One quirk is
// not reproduced: for input that interleaves hex and non-hex characters,
// jsrsasign routes through CryptoJS's 32-bit word packing whose exact garbage
// bytes we do not emulate, so such malformed input can differ.
func hexToBase64(hexStr string) string {
	var sb strings.Builder
	bitBuf, bitCnt := 0, 0
	for i := 0; i < len(hexStr); {
		var v, nbits int
		if i+2 <= len(hexStr) {
			v, nbits = parseHexLenient(hexStr[i:i+2]), 8
			i += 2
		} else {
			v, nbits = parseHexLenient(hexStr[i:i+1]), 4
			i++
		}
		bitBuf = bitBuf<<nbits | v
		bitCnt += nbits
		for bitCnt >= 6 {
			bitCnt -= 6
			sb.WriteByte(pemBase64Map[(bitBuf>>bitCnt)&0x3f])
		}
		bitBuf &= (1 << bitCnt) - 1
	}
	if bitCnt > 0 {
		sb.WriteByte(pemBase64Map[(bitBuf<<(6-bitCnt))&0x3f])
	}
	for sb.Len()%4 != 0 {
		sb.WriteByte('=')
	}
	return sb.String()
}

// parseHexLenient mirrors JavaScript parseInt(s, 16): it reads the leading run of
// hex digits and stops at the first non-hex character; a string with no leading
// hex digit yields 0 (JavaScript's NaN, coerced to 0 by the bitwise pipeline).
func parseHexLenient(s string) int {
	v := 0
	for i := 0; i < len(s); i++ {
		var d int
		switch c := s[i]; {
		case c >= '0' && c <= '9':
			d = int(c - '0')
		case c >= 'a' && c <= 'f':
			d = int(c-'a') + 10
		case c >= 'A' && c <= 'F':
			d = int(c-'A') + 10
		default:
			return v
		}
		v = v*16 + d
	}
	return v
}

// wrapBase64 joins width-character chunks of s with CRLF.
func wrapBase64(s string, width int) string {
	var lines []string
	for i := 0; i < len(s); i += width {
		end := min(i+width, len(s))
		lines = append(lines, s[i:end])
	}
	return strings.Join(lines, "\r\n")
}

// PEMToHex extracts the DER content of PEM blocks as a hex string.
type PEMToHex struct{}

// Meta returns the operation metadata.
func (PEMToHex) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "PEM to Hex",
		Module:      "Default",
		Description: "Converts PEM (Privacy Enhanced Mail) format to a hexadecimal DER (Distinguished Encoding Rules) string.",
		InfoURL:     "https://wikipedia.org/wiki/Privacy-Enhanced_Mail",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (PEMToHex) Args() []core.ArgDef { return nil }

// Run decodes the PEM blocks to hex. Ported from CyberChef PEMToHex.mjs: each
// "-----BEGIN <type>-----" header is paired with its matching footer, the base64
// body between them is leniently decoded, and the blocks' hex is joined with
// newlines.
func (PEMToHex) Run(in *core.Dish, args []any) (*core.Dish, error) {
	input := in.String()
	var blocks []string
	for _, m := range pemBeginRe.FindAllStringSubmatchIndex(input, -1) {
		header := input[m[2]:m[3]]
		footer := "-----END " + header + "-----"
		start := m[1]
		rel := strings.Index(input[start:], footer)
		if rel < 0 {
			return nil, fmt.Errorf("PEM footer '%s' not found", footer)
		}
		// fromBase64 only errors on an invalid alphabet or in strict mode; the
		// alphabet is a fixed valid constant and strict is off, so it cannot fail.
		bytes, _ := fromBase64(input[start:start+rel], "A-Za-z0-9+/=", true, false)
		blocks = append(blocks, toHexFast(bytes))
	}
	return core.NewDish([]byte(strings.Join(blocks, "\n")), core.TypeString), nil
}
