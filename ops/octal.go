package ops

import (
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/core"
)

// octalDelims are the delimiter options for the octal operations.
var octalDelims = []string{"Space", "Comma", "Semi-colon", "Colon", "Line feed", "CRLF"}

func init() {
	core.Register(ToOctal{})
	core.Register(FromOctal{})
}

// ToOctal converts the input to octal bytes separated by the chosen delimiter.
type ToOctal struct{}

// Meta returns the operation metadata.
func (ToOctal) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "To Octal",
		Module:      "Default",
		Description: "Converts the input to octal bytes separated by the specified delimiter.",
		InfoURL:     "https://wikipedia.org/wiki/Octal",
		InputType:   core.TypeByteArray,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ToOctal) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Delimiter", Type: core.ArgOption, Value: octalDelims},
	}
}

// Run encodes the input.
func (ToOctal) Run(in *core.Dish, args []any) (*core.Dish, error) {
	delim := charRep(args[0].(string))
	data := in.Bytes()
	parts := make([]string, len(data))
	for i, b := range data {
		parts[i] = strconv.FormatUint(uint64(b), 8)
	}
	return core.NewDish([]byte(strings.Join(parts, delim)), core.TypeString), nil
}

// FromOctal converts an octal byte string back into its raw value.
type FromOctal struct{}

// Meta returns the operation metadata.
func (FromOctal) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "From Octal",
		Module:      "Default",
		Description: "Converts an octal byte string back into its raw value.",
		InfoURL:     "https://wikipedia.org/wiki/Octal",
		InputType:   core.TypeString,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (FromOctal) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Delimiter", Type: core.ArgOption, Value: octalDelims},
	}
}

// Run decodes the input.
func (FromOctal) Run(in *core.Dish, args []any) (*core.Dish, error) {
	data := in.String()
	if len(data) == 0 {
		return core.NewDish(nil, core.TypeByteArray), nil
	}
	delim := charRep(args[0].(string))

	var parts []string
	if delim == "" {
		parts = []string{data}
	} else {
		parts = strings.Split(data, delim)
	}

	out := make([]byte, 0, len(parts))
	for _, p := range parts {
		v, err := strconv.ParseUint(p, 8, 8)
		if err != nil {
			return nil, err
		}
		out = append(out, byte(v))
	}
	return core.NewDish(out, core.TypeByteArray), nil
}
