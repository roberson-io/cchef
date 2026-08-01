package ops

import (
	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/lodashcase"
)

func init() {
	core.Register(ToKebabCase{})
}

// ToKebabCase struct.
type ToKebabCase struct{}

// Meta returns the operation metadata.
func (ToKebabCase) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "To Kebab case",
		Module:      "Code",
		Description: "Converts the input string to kebab case. Kebab case is all lower case with dashes as word boundaries, e.g. this-is-kebab-case.",
		InfoURL:     "https://wikipedia.org/wiki/Letter_case#Special_case_styles",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ToKebabCase) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Attempt to be context aware", Type: core.ArgBoolean, Value: false},
	}
}

// Run converts the input to kebab case. With "Attempt to be context aware", only
// identifier-like tokens are transformed.
func (ToKebabCase) Run(in *core.Dish, args []any) (*core.Dish, error) {
	smart := args[0].(bool)
	input := in.String()
	out := lodashcase.KebabCase(input)
	if smart {
		out = lodashcase.ReplaceVariableNames(input, lodashcase.KebabCase)
	}
	return core.NewDish([]byte(out), core.TypeString), nil
}
