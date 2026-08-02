package ops

import (
	"errors"
	"strings"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/jsnum"
)

func init() {
	core.Register(MOD{})
}

// MOD reduces every number in a list by a modulus.
type MOD struct{}

// Meta returns the operation metadata.
func (MOD) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "MOD",
		Module:      "Default",
		Description: "Computes the modulo of each number in a list with a given modulus value. Numbers are extracted from the input based on the delimiter, and non-numeric values are ignored.\n\ne.g. 15 4 7 with modulus 3 becomes 0 1 1",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the modulus and the delimiter the input is split on.
func (MOD) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Modulus", Type: core.ArgNumber, Value: 2},
		{Name: "Delimiter", Type: core.ArgOption, Value: arithmeticDelims},
	}
}

// Run reduces each number, reporting the results separated by spaces whatever
// the input was separated by.
func (MOD) Run(in *core.Dish, args []any) (*core.Dish, error) {
	// The argument arrives as a number, and bignumber.js builds from a number
	// through its printed form, so the same route is taken here rather than
	// converting the binary value directly.
	modulus, ok := parseBigNum(jsnum.Format(args[0].(float64)))
	if !ok || modulus.isZero() {
		return nil, errors.New("Modulus cannot be zero") //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}

	numbers := createNumArray(in.String(), args[1].(string))
	results := make([]string, len(numbers))
	for i, n := range numbers {
		results[i] = n.mod(modulus).String()
	}
	return core.NewDish([]byte(strings.Join(results, " ")), core.TypeString), nil
}
