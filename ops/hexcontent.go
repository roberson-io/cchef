package ops

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/core"
)

// hexContentRe matches a SNORT hex block: pipes around two or more hex digits
// and spaces. Mirrors CyberChef's /\|([a-f\d ]{2,})\|/gi.
var hexContentRe = regexp.MustCompile(`(?i)\|([a-f\d ]{2,})\|`)

func init() {
	core.Register(ToHexContent{})
	core.Register(FromHexContent{})
}

// ToHexContent converts special characters in a string to SNORT hex-content
// notation (e.g. foo=bar -> foo|3d|bar).
type ToHexContent struct{}

// Meta returns the operation metadata.
func (ToHexContent) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "To Hex Content",
		Module:      "Default",
		Description: "Converts special characters in a string to hexadecimal. This format is used by SNORT for representing hex within ASCII text.",
		InfoURL:     "http://manual-snort-org.s3-website-us-east-1.amazonaws.com/node32.html#SECTION00451000000000000000",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ToHexContent) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Convert", Type: core.ArgOption, Value: []string{
			"Only special chars", "Only special chars including spaces", "All chars",
		}},
		{Name: "Print spaces between bytes", Type: core.ArgBoolean, Value: false},
	}
}

// Run encodes the input. Ported from CyberChef ToHexContent.mjs.
func (ToHexContent) Run(in *core.Dish, args []any) (*core.Dish, error) {
	input := in.Bytes()
	convert := args[0].(string)
	spaces := args[1].(bool)

	if convert == "All chars" {
		result := "|" + toHex(input, " ", "") + "|"
		if !spaces {
			result = strings.ReplaceAll(result, " ", "")
		}
		return core.NewDish([]byte(result), core.TypeString), nil
	}

	convertSpaces := convert == "Only special chars including spaces"
	var sb strings.Builder
	inHex := false
	for _, b := range input {
		if hexContentSpecial(b, convertSpaces) {
			if !inHex {
				sb.WriteByte('|')
				inHex = true
			} else if spaces {
				sb.WriteByte(' ')
			}
			fmt.Fprintf(&sb, "%02x", b)
		} else {
			if inHex {
				sb.WriteByte('|')
				inHex = false
			}
			sb.WriteByte(b)
		}
	}
	if inHex {
		sb.WriteByte('|')
	}
	return core.NewDish([]byte(sb.String()), core.TypeString), nil
}

// hexContentSpecial reports whether a byte is hex-encoded rather than kept
// verbatim: everything outside [0-9A-Za-z] is special, plus space when
// convertSpaces is set. Mirrors the byte-range test in ToHexContent.mjs.
func hexContentSpecial(b byte, convertSpaces bool) bool {
	return (b == 32 && convertSpaces) ||
		(b < 48 && b != 32) ||
		(b > 57 && b < 65) ||
		(b > 90 && b < 97) ||
		b > 122
}

// FromHexContent translates SNORT hex-content notation back to raw bytes
// (e.g. foo|3d|bar -> foo=bar).
type FromHexContent struct{}

// Meta returns the operation metadata.
func (FromHexContent) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "From Hex Content",
		Module:      "Default",
		Description: "Translates hexadecimal bytes in text back to raw bytes. This format is used by SNORT for representing hex within ASCII text.",
		InfoURL:     "http://manual-snort-org.s3-website-us-east-1.amazonaws.com/node32.html#SECTION00451000000000000000",
		InputType:   core.TypeString,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (FromHexContent) Args() []core.ArgDef { return nil }

// Run decodes the input. Ported from CyberChef FromHexContent.mjs: each
// "|<hex>|" block is decoded and everything else is passed through as raw bytes.
func (FromHexContent) Run(in *core.Dish, args []any) (*core.Dish, error) {
	input := in.Bytes()
	var out []byte
	i := 0
	for _, m := range hexContentRe.FindAllSubmatchIndex(input, -1) {
		// Raw bytes up to the match.
		out = append(out, input[i:m[0]]...)
		out = append(out, decodeHexContent(input[m[2]:m[3]])...)
		i = m[1]
	}
	out = append(out, input[i:]...)
	return core.NewDish(out, core.TypeByteArray), nil
}

// decodeHexContent decodes the inside of a hex block the way CyberChef's fromHex
// (byteLen 2, "Auto" delimiter) does: split on non-hex runs, then read two-hex-
// digit chunks, keeping a trailing single digit as its own byte.
func decodeHexContent(content []byte) []byte {
	var out []byte
	for _, part := range nonHex.Split(string(content), -1) {
		for j := 0; j < len(part); j += 2 {
			end := min(j+2, len(part))
			v, _ := strconv.ParseUint(part[j:end], 16, 8) // pure-hex chunk: never errors
			out = append(out, byte(v))
		}
	}
	return out
}
