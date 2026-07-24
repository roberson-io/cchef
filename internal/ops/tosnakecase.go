package ops

import (
	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(ToSnakeCase{})
}

// ToSnakeCase struct.
type ToSnakeCase struct{}

// Meta returns the operation metadata.
func (ToSnakeCase) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "To Snake case",
		Module:      "Code",
		Description: "Converts the input string to snake case. Snake case is all lower case with underscores as word boundaries, e.g. this_is_snake_case.",
		InfoURL:     "https://wikipedia.org/wiki/Snake_case",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ToSnakeCase) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Attempt to be context aware", Type: core.ArgBoolean, Value: false},
	}
}

// Run converts the input to snake case. With "Attempt to be context aware", only
// identifier-like tokens are transformed (via replaceVariableNames).
func (ToSnakeCase) Run(in *core.Dish, args []any) (*core.Dish, error) {
	smart := args[0].(bool)
	input := in.String()
	out := lodashSnakeCase(input)
	if smart {
		out = replaceVariableNames(input, lodashSnakeCase)
	}
	return core.NewDish([]byte(out), core.TypeString), nil
}
