package ops

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/core"
)

// decimalDelims are the delimiter options for the decimal operations (CyberChef
// DELIM_OPTIONS).
var decimalDelims = []string{"Space", "Comma", "Semi-colon", "Colon", "Line feed", "CRLF"}

func init() {
	core.Register(ToDecimal{})
	core.Register(FromDecimal{})
}

// ToDecimal converts input bytes to a delimited list of decimal values.
type ToDecimal struct{}

// Meta returns the operation metadata.
func (ToDecimal) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "To Decimal",
		Module:      "Default",
		Description: "Converts the input data to an ordinal integer array, optionally treating each byte as a signed value.",
		InfoURL:     "https://wikipedia.org/wiki/Decimal",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ToDecimal) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Delimiter", Type: core.ArgOption, Value: decimalDelims},
		{Name: "Support signed values", Type: core.ArgBoolean, Value: false},
	}
}

// Run encodes the input. Ported from CyberChef ToDecimal.mjs.
func (ToDecimal) Run(in *core.Dish, args []any) (*core.Dish, error) {
	delim := charRep(args[0].(string))
	signed := args[1].(bool)
	data := in.Bytes()
	parts := make([]string, len(data))
	for i, b := range data {
		v := int(b)
		if signed && v > 127 {
			v -= 256
		}
		parts[i] = strconv.Itoa(v)
	}
	return core.NewDish([]byte(strings.Join(parts, delim)), core.TypeString), nil
}

// FromDecimal converts a delimited list of decimal values back into bytes.
type FromDecimal struct{}

// Meta returns the operation metadata.
func (FromDecimal) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "From Decimal",
		Module:      "Default",
		Description: "Converts the data from an ordinal integer array back into its raw form.",
		InfoURL:     "https://wikipedia.org/wiki/Decimal",
		InputType:   core.TypeString,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (FromDecimal) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Delimiter", Type: core.ArgOption, Value: decimalDelims},
		{Name: "Support signed values", Type: core.ArgBoolean, Value: false},
	}
}

// Run decodes the input. Ported from CyberChef FromDecimal.mjs.
func (FromDecimal) Run(in *core.Dish, args []any) (*core.Dish, error) {
	delim := charRep(args[0].(string))
	signed := args[1].(bool)

	fields := strings.FieldsFunc(in.String(), func(r rune) bool {
		return strings.ContainsRune(delim, r) || r == ' ' || r == '\n' || r == '\r' || r == '\t'
	})

	var out []byte
	for _, f := range fields {
		v, err := strconv.Atoi(f)
		if err != nil {
			return nil, fmt.Errorf("invalid decimal value %q: %w", f, err)
		}
		if signed && v < 0 {
			v += 256
		}
		out = append(out, byte(v)) // #nosec G115 -- ordinal narrowed to a byte, matching CyberChef byteArray semantics
	}
	return core.NewDish(out, core.TypeByteArray), nil
}
