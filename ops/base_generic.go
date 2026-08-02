package ops

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(ToBase{})
	core.Register(FromBase{})
}

// ToBase converts an arbitrary-precision integer to a string in the given radix.
type ToBase struct{}

// Meta returns the operation metadata.
func (ToBase) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "To Base",
		Module:      "Default",
		Description: "Converts a decimal number to a different numerical base (radix 2-36).",
		InfoURL:     "https://wikipedia.org/wiki/Radix",
		InputType:   core.TypeBigNumber,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ToBase) Args() []core.ArgDef {
	minRadix, maxRadix := 2.0, 36.0
	return []core.ArgDef{
		{Name: "Radix", Type: core.ArgNumber, Integer: true, Value: 36, Min: &minRadix, Max: &maxRadix},
	}
}

// Run converts the input number to the target radix.
func (ToBase) Run(in *core.Dish, args []any) (*core.Dish, error) {
	s := strings.TrimSpace(in.String())
	if s == "" {
		return nil, fmt.Errorf("input must be a number")
	}
	n, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return nil, fmt.Errorf("input %q is not a valid integer", s)
	}
	radix := int(args[0].(float64))
	return core.NewDish([]byte(n.Text(radix)), core.TypeString), nil
}

// FromBase converts a string in the given radix to a decimal integer.
type FromBase struct{}

// Meta returns the operation metadata.
func (FromBase) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "From Base",
		Module:      "Default",
		Description: "Converts a number from a given numerical base (radix 2-36) to decimal. Only integer values are supported.",
		InfoURL:     "https://wikipedia.org/wiki/Radix",
		InputType:   core.TypeString,
		OutputType:  core.TypeBigNumber,
	}
}

// Args returns the argument definitions.
func (FromBase) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Radix", Type: core.ArgNumber, Integer: true, Value: 36},
	}
}

// Run converts the input from the given radix to decimal.
func (FromBase) Run(in *core.Dish, args []any) (*core.Dish, error) {
	radix := int(args[0].(float64))
	if radix < 2 || radix > 36 {
		return nil, fmt.Errorf("radix argument must be between 2 and 36")
	}
	// Strip all whitespace, matching CyberChef's input.replace(/\s/g, "").
	s := strings.Join(strings.Fields(in.String()), "")
	if strings.ContainsRune(s, '.') {
		return nil, fmt.Errorf("fractional input is not supported")
	}
	n, ok := new(big.Int).SetString(s, radix)
	if !ok {
		return nil, fmt.Errorf("input %q is not valid in radix %d", s, radix)
	}
	return core.NewDish([]byte(n.Text(10)), core.TypeBigNumber), nil
}
