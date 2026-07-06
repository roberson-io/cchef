package ops

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(ToHexdump{})
	core.Register(FromHexdump{})
}

// maxHexdumpWidth caps the To Hexdump line width (matches CyberChef's MAX_WIDTH).
const maxHexdumpWidth = 65536

// ToHexdump renders the input as a classic hexdump.
type ToHexdump struct{}

// Meta returns the operation metadata.
func (ToHexdump) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "To Hexdump",
		Module:      "Default",
		Description: "Creates a hexdump of the input data, displaying both the hexadecimal values of each byte and an ASCII representation alongside. The 'UNIX format' argument defines which subset of printable characters are displayed in the preview column.",
		InfoURL:     "https://wikipedia.org/wiki/Hex_dump",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ToHexdump) Args() []core.ArgDef {
	minWidth, maxWidth := 1.0, float64(maxHexdumpWidth)
	return []core.ArgDef{
		{Name: "Width", Type: core.ArgNumber, Value: float64(16), Min: &minWidth, Max: &maxWidth},
		{Name: "Upper case hex", Type: core.ArgBoolean, Value: false},
		{Name: "Include final length", Type: core.ArgBoolean, Value: false},
		{Name: "UNIX format", Type: core.ArgBoolean, Value: false},
	}
}

// Run renders the hexdump. Ported from ToHexdump.mjs.
func (ToHexdump) Run(in *core.Dish, args []any) (*core.Dish, error) {
	length := args[0].(float64)
	upperCase := args[1].(bool)
	includeFinalLength := args[2].(bool)
	unixFormat := args[3].(bool)

	// Width is constrained to [1, maxHexdumpWidth] by the ArgDef Min/Max; only
	// the integer requirement is left to check here.
	if math.Round(length) != length {
		return nil, fmt.Errorf("width must be a positive integer")
	}
	width := int(length)

	data := in.Bytes()
	var lines []string
	for i := 0; i < len(data); i += width {
		end := min(i+width, len(data))
		buff := data[i:end]

		hex := make([]string, len(buff))
		for j, b := range buff {
			hex[j] = fmt.Sprintf("%02x", b)
		}
		// Pad the joined hex to width*(padding+1) columns (padding = 2).
		hexStr := padEndSpace(strings.Join(hex, " "), width*3)

		var ascii strings.Builder
		for _, b := range buff {
			ascii.WriteRune(hexdumpPrintable(b, unixFormat))
		}

		lineNo := fmt.Sprintf("%08x", i)
		if upperCase {
			hexStr = strings.ToUpper(hexStr)
			lineNo = strings.ToUpper(lineNo)
		}

		lines = append(lines, fmt.Sprintf("%s  %s |%s|", lineNo, hexStr, ascii.String()))

		if includeFinalLength && end == len(data) {
			lines = append(lines, fmt.Sprintf("%08x", end))
		}
	}

	return core.NewDish([]byte(strings.Join(lines, "\n")), core.TypeString), nil
}

// hexdumpPrintable maps a byte to the character shown in the ASCII preview
// column, replacing non-printable values with a dot. Ported from Utils.printable
// restricted to the Latin-1 range (the only inputs a byte buffer produces): the
// combined printable + whitespace regexes dot 0x00-0x1F, 0x7F-0x9F, and 0xAD.
// With UNIX format only 0x20-0x7E are kept.
func hexdumpPrintable(b byte, unixFormat bool) rune {
	if unixFormat {
		if b < 0x20 || b > 0x7e {
			return '.'
		}
		return rune(b)
	}
	if b <= 0x1f || (b >= 0x7f && b <= 0x9f) || b == 0xad {
		return '.'
	}
	return rune(b)
}

// FromHexdump attempts to convert a hexdump back into raw data.
type FromHexdump struct{}

// Meta returns the operation metadata.
func (FromHexdump) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "From Hexdump",
		Module:      "Default",
		Description: "Attempts to convert a hexdump back into raw data. This operation supports many different hexdump variations, but probably not all. Make sure you verify that the data it gives you is correct before continuing analysis.",
		InfoURL:     "https://wikipedia.org/wiki/Hex_dump",
		InputType:   core.TypeString,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (FromHexdump) Args() []core.ArgDef {
	return nil
}

// fromHexdumpRE extracts the hex-byte region of each hexdump line, tolerating
// the many tool formats (xxd, Wireshark, 010, Linux, ...). Ported verbatim from
// FromHexdump.mjs; it uses no lookahead/backreferences, so RE2 handles it.
var fromHexdumpRE = regexp.MustCompile(`(?im)^\s*(?:[\dA-F]{4,16}h?:?)?[ \t]+((?:[\dA-F]{2} ){1,8}(?:[ \t]|[\dA-F]{2}-)(?:[\dA-F]{2} ){1,8}|(?:[\dA-F]{4} )+(?:[\dA-F]{2})?|(?:[\dA-F]{2} )*[\dA-F]{2})`)

// Run parses the hexdump. Ported from FromHexdump.mjs. The upstream width/CR
// detection only toggles UI highlighting, so it is omitted here.
func (FromHexdump) Run(in *core.Dish, args []any) (*core.Dish, error) {
	var out []byte
	for _, m := range fromHexdumpRE.FindAllStringSubmatch(in.String(), -1) {
		out = append(out, fromHexAuto(strings.ReplaceAll(m[1], "-", " "))...)
	}
	return core.NewDish(out, core.TypeByteArray), nil
}

// fromHexAuto decodes a run of hex bytes, splitting on any non-hex separator
// (CyberChef's fromHex with the "Auto" delimiter).
func fromHexAuto(s string) []byte {
	var out []byte
	for _, part := range nonHex.Split(s, -1) {
		for j := 0; j+2 <= len(part); j += 2 {
			if v, err := strconv.ParseUint(part[j:j+2], 16, 8); err == nil {
				out = append(out, byte(v))
			}
		}
	}
	return out
}
