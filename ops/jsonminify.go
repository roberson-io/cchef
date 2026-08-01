package ops

import (
	"fmt"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(JSONMinify{})
}

// JSONMinify struct.
type JSONMinify struct{}

// Meta returns the operation metadata.
func (JSONMinify) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "JSON Minify",
		Module:      "Code",
		Description: "Compresses JavaScript Object Notation (JSON) code.",
		InfoURL:     "https://wikipedia.org/wiki/JSON",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (JSONMinify) Args() []core.ArgDef { return nil }

// Run minifies the JSON input. Ported from vkbeautify.jsonmin =
// JSON.stringify(JSON.parse(text), null, 0); empty input yields "".
func (JSONMinify) Run(in *core.Dish, _ []any) (*core.Dish, error) {
	input := in.String()
	if input == "" {
		return core.NewDish([]byte(""), core.TypeString), nil
	}
	val, err := jsonParseOrdered([]byte(input))
	if err != nil {
		return nil, fmt.Errorf("invalid JSON input: %w", err)
	}
	return core.NewDish([]byte(jsStringify(val, 0)), core.TypeString), nil
}
