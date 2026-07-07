package ops

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(ToHex{})
	core.Register(FromHex{})
}

// toHexDelims are the named delimiter options for To Hex. Ported from
// CyberChef's TO_HEX_DELIM_OPTIONS / Utils.charRep.
var toHexDelims = []string{
	"Space", "Percent", "Comma", "Semi-colon", "Colon",
	"Line feed", "CRLF", "0x", "0x with comma", "\\x", "None",
}

// charRep maps a delimiter option name to its literal string.
func charRep(token string) string {
	switch token {
	case "Space":
		return " "
	case "Percent":
		return "%"
	case "Comma":
		return ","
	case "Semi-colon":
		return ";"
	case "Colon":
		return ":"
	case "Line feed":
		return "\n"
	case "CRLF":
		return "\r\n"
	case "0x":
		return "0x"
	case "\\x":
		return "\\x"
	default: // "None", "Nothing (separate chars)"
		return ""
	}
}

// nonHex matches any run of characters that cannot be part of hex bytes; used as
// the "Auto" delimiter when decoding. Mirrors CyberChef's /[^a-f\d]|0x/gi.
var nonHex = regexp.MustCompile(`(?i)[^a-f\d]|0x`)

// splitHexToBytes decodes hex "Auto" input: it splits on non-hex runs and reads
// whole byte pairs within each run (an odd trailing nibble is ignored). Because
// nonHex.Split yields only hex characters, every pair parses, so this cannot
// fail — unlike hexToBytes, which strips separators and pairs across them.
func splitHexToBytes(s string) []byte {
	var out []byte
	for _, part := range nonHex.Split(s, -1) {
		for j := 0; j+2 <= len(part); j += 2 {
			v, _ := strconv.ParseUint(part[j:j+2], 16, 8) // pure-hex pair: never errors
			out = append(out, byte(v))
		}
	}
	return out
}

// toHex encodes bytes to hex with the given delimiter and optional extra
// delimiter (for "0x with comma"). Ported from lib/Hex.mjs toHex.
func toHex(data []byte, delim, extraDelim string) string {
	if len(data) == 0 {
		return ""
	}
	prepend := delim == "0x" || delim == "\\x" || delim == "%"

	var sb strings.Builder
	for _, b := range data {
		h := fmt.Sprintf("%02x", b)
		if prepend {
			sb.WriteString(delim + h)
		} else {
			sb.WriteString(h + delim)
		}
		if extraDelim != "" {
			sb.WriteString(extraDelim)
		}
	}

	out := sb.String()
	trunc := len(extraDelim)
	if !prepend {
		trunc += len(delim)
	}
	if trunc > 0 {
		out = out[:len(out)-trunc]
	}
	return out
}

// ToHex converts input bytes to hexadecimal separated by the chosen delimiter.
type ToHex struct{}

// Meta returns the operation metadata.
func (ToHex) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "To Hex",
		Module:      "Default",
		Description: "Converts the input to hexadecimal bytes separated by the specified delimiter.",
		InfoURL:     "https://wikipedia.org/wiki/Hexadecimal",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ToHex) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Delimiter", Type: core.ArgOption, Value: toHexDelims},
	}
}

// Run encodes the input.
func (ToHex) Run(in *core.Dish, args []any) (*core.Dish, error) {
	opt := args[0].(string)
	delim, extra := charRep(opt), ""
	if opt == "0x with comma" {
		delim, extra = "0x", ","
	}
	return core.NewDish([]byte(toHex(in.Bytes(), delim, extra)), core.TypeString), nil
}

// FromHex converts a hexadecimal byte string back to its raw value.
type FromHex struct{}

// Meta returns the operation metadata.
func (FromHex) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "From Hex",
		Module:      "Default",
		Description: "Converts a hexadecimal byte string back into its raw value.",
		InfoURL:     "https://wikipedia.org/wiki/Hexadecimal",
		InputType:   core.TypeString,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (FromHex) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Delimiter", Type: core.ArgOption, Value: append([]string{"Auto"}, toHexDelims...)},
	}
}

// Run decodes the input. Ported from lib/Hex.mjs fromHex.
func (FromHex) Run(in *core.Dish, args []any) (*core.Dish, error) {
	delim := args[0].(string)
	data := in.String()

	var parts []string
	switch delim {
	case "None":
		parts = []string{data}
	case "Auto":
		parts = nonHex.Split(data, -1)
	default:
		parts = strings.Split(data, charRep(delim))
	}

	var out []byte
	for _, p := range parts {
		for j := 0; j+2 <= len(p); j += 2 {
			v, err := strconv.ParseUint(p[j:j+2], 16, 8)
			if err != nil {
				return nil, fmt.Errorf("invalid hex byte %q: %w", p[j:j+2], err)
			}
			out = append(out, byte(v))
		}
	}
	return core.NewDish(out, core.TypeByteArray), nil
}
