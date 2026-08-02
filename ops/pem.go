package ops

import (
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/jsnum"
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

// hexToBase64 base64-encodes a hex string the way jsrsasign's hex2b64 does: the
// hex is read three digits at a time, twelve bits becoming two base64
// characters, with a one- or two-digit remainder handled separately and the
// result padded to a multiple of four.
//
// Reading in threes rather than in bytes is what makes malformed input behave
// as it does. Each group goes through JavaScript's parseInt, which takes the
// leading run of hex digits and stops, so a stray character truncates its group
// rather than shifting the digits after it along.
func hexToBase64(hexStr string) string {
	var sb strings.Builder
	i := 0
	for ; i+3 <= len(hexStr); i += 3 {
		e := int(jsToInt32(jsnum.ParseHex(hexStr[i : i+3])))
		sb.WriteString(base64CharAt(e >> 6))
		sb.WriteString(base64CharAt(e & 0x3f))
	}
	switch len(hexStr) - i {
	case 1:
		sb.WriteString(base64CharAt(int(jsToInt32(jsnum.ParseHex(hexStr[i:]))) << 2))
	case 2:
		e := int(jsToInt32(jsnum.ParseHex(hexStr[i:])))
		sb.WriteString(base64CharAt(e >> 2))
		sb.WriteString(base64CharAt((e & 3) << 4))
	}
	for sb.Len()%4 != 0 {
		sb.WriteByte('=')
	}
	return sb.String()
}

// base64CharAt indexes the base64 alphabet, returning nothing for an index
// outside it. JavaScript's charAt does the same, which is how a negative value
// — reachable because parseInt accepts a sign — drops a character rather than
// erroring.
func base64CharAt(i int) string {
	if i < 0 || i >= len(pemBase64Map) {
		return ""
	}
	return string(pemBase64Map[i])
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
		blocks = append(blocks, hex.EncodeToString(bytes))
	}
	return core.NewDish([]byte(strings.Join(blocks, "\n")), core.TypeString), nil
}
