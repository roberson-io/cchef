package ops

import (
	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/lodashcase"
)

func init() {
	core.Register(ToCamelCase{})
}

// ToCamelCase struct.
type ToCamelCase struct{}

// Meta returns the operation metadata.
func (ToCamelCase) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "To Camel case",
		Module:      "Code",
		Description: "Converts the input string to camel case. Camel case is all lower case except letters after word boundaries which are uppercase, e.g. thisIsCamelCase.",
		InfoURL:     "https://wikipedia.org/wiki/Camel_case",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ToCamelCase) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Attempt to be context aware", Type: core.ArgBoolean, Value: false},
	}
}

// Run converts the input to camel case. With "Attempt to be context aware", only
// identifier-like tokens are transformed.
func (ToCamelCase) Run(in *core.Dish, args []any) (*core.Dish, error) {
	smart := args[0].(bool)
	input := in.String()
	out := lodashcase.CamelCase(input)
	if smart {
		out = lodashcase.ReplaceVariableNames(input, lodashcase.CamelCase)
	}
	return core.NewDish([]byte(out), core.TypeString), nil
}
