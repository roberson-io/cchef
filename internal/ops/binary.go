package ops

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

// binDelims are the delimiter options for the binary operations (CyberChef
// BIN_DELIM_OPTIONS).
var binDelims = []string{"Space", "Comma", "Semi-colon", "Colon", "Line feed", "CRLF", "None"}

func init() {
	core.Register(ToBinary{})
	core.Register(FromBinary{})
}

// ToBinary converts input bytes to a binary string.
type ToBinary struct{}

// Meta returns the operation metadata.
func (ToBinary) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "To Binary",
		Module:      "Default",
		Description: "Displays the input data as a binary string, with each byte zero-padded to the given length.",
		InfoURL:     "https://wikipedia.org/wiki/Binary_number",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ToBinary) Args() []core.ArgDef {
	min := 1.0
	return []core.ArgDef{
		{Name: "Delimiter", Type: core.ArgOption, Value: binDelims},
		{Name: "Byte Length", Type: core.ArgNumber, Value: 8, Min: &min},
	}
}

// Run encodes the input. Ported from CyberChef lib/Binary.mjs toBinary.
func (ToBinary) Run(in *core.Dish, args []any) (*core.Dish, error) {
	delim := charRep(args[0].(string))
	padding := int(args[1].(float64))
	if padding < 1 {
		padding = 8
	}
	data := in.Bytes()
	parts := make([]string, len(data))
	for i, b := range data {
		parts[i] = leftPad(strconv.FormatUint(uint64(b), 2), padding)
	}
	return core.NewDish([]byte(strings.Join(parts, delim)), core.TypeString), nil
}

// FromBinary converts a binary string back into its raw byte value.
type FromBinary struct{}

// Meta returns the operation metadata.
func (FromBinary) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "From Binary",
		Module:      "Default",
		Description: "Converts a binary string back into its raw form.",
		InfoURL:     "https://wikipedia.org/wiki/Binary_number",
		InputType:   core.TypeString,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (FromBinary) Args() []core.ArgDef {
	min := 1.0
	return []core.ArgDef{
		{Name: "Delimiter", Type: core.ArgOption, Value: binDelims},
		{Name: "Byte Length", Type: core.ArgNumber, Value: 8, Min: &min},
	}
}

// Run decodes the input. Ported from CyberChef lib/Binary.mjs fromBinary.
func (FromBinary) Run(in *core.Dish, args []any) (*core.Dish, error) {
	byteLen := int(args[1].(float64))
	if byteLen < 1 {
		return nil, fmt.Errorf("byte length must be a positive integer")
	}
	// Keep only binary digits, discarding any delimiter characters.
	var bitsOnly strings.Builder
	for _, r := range in.String() {
		if r == '0' || r == '1' {
			bitsOnly.WriteRune(r)
		}
	}
	bitStr := bitsOnly.String()

	var out []byte
	for i := 0; i+byteLen <= len(bitStr); i += byteLen {
		v, err := strconv.ParseUint(bitStr[i:i+byteLen], 2, 64)
		if err != nil {
			return nil, err
		}
		out = append(out, byte(v))
	}
	return core.NewDish(out, core.TypeByteArray), nil
}

// leftPad zero-pads s on the left to at least width characters.
func leftPad(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return strings.Repeat("0", width-len(s)) + s
}
