package ops

import (
	"math/big"
	"sort"
	"strings"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(Sum{})
	core.Register(Subtract{})
	core.Register(Multiply{})
	core.Register(Divide{})
	core.Register(Mean{})
	core.Register(Median{})
	core.Register(StandardDeviation{})
}

// arithmeticDelims are the delimiter options for the arithmetic operations
// (CyberChef ARITHMETIC_DELIM_OPTIONS). The default is "Line feed".
var arithmeticDelims = []string{"Line feed", "Space", "Comma", "Semi-colon", "Colon", "CRLF"}

// createNumArray splits the input on the named delimiter and parses each token as
// a bignumber, excluding any that are NaN. Ported from lib/Arithmetic.mjs.
func createNumArray(input, delimName string) []bigNum {
	parts := strings.Split(input, charRep(delimName))
	nums := make([]bigNum, 0, len(parts))
	for _, p := range parts {
		if n, ok := parseBigNum(p); ok {
			nums = append(nums, n)
		}
	}
	return nums
}

// reduceNums folds data left-to-right with op, returning NaN for an empty list
// (mirroring CyberChef, where an undefined reduction becomes BigNumber(NaN)).
func reduceNums(data []bigNum, op func(a, b bigNum) bigNum) bigNum {
	if len(data) == 0 {
		return bnNaN
	}
	acc := data[0]
	for _, n := range data[1:] {
		acc = op(acc, n)
	}
	return acc
}

func sumNums(data []bigNum) bigNum { return reduceNums(data, bigNum.plus) }

// meanNums computes sum(data).div(length), matching lib/Arithmetic.mjs (the
// division rounds to 20 decimal places).
func meanNums(data []bigNum) bigNum {
	if len(data) == 0 {
		return bnNaN
	}
	return sumNums(data).div(fromInt(len(data)))
}

// medianNums returns the median, averaging the two middle values for an even
// count. Ported from lib/Arithmetic.mjs.
func medianNums(data []bigNum) bigNum {
	if len(data) == 0 {
		return bnNaN
	}
	sorted := make([]bigNum, len(data))
	copy(sorted, data)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].cmp(sorted[j]) < 0 })
	if len(sorted)%2 == 0 {
		return meanNums([]bigNum{sorted[len(sorted)/2], sorted[len(sorted)/2-1]})
	}
	return sorted[len(sorted)/2]
}

// stdDevNums computes the population standard deviation, matching
// lib/Arithmetic.mjs: mean, then sqrt of the mean squared deviation. Both the
// division and the square root round to 20 decimal places.
func stdDevNums(data []bigNum) bigNum {
	if len(data) == 0 {
		return bnNaN
	}
	// Any ±Infinity element makes a deviation (±Infinity − ±Infinity) NaN, so the
	// whole result is NaN — matching bignumber.js.
	for _, n := range data {
		if n.inf != 0 || n.nan {
			return bnNaN
		}
	}
	avg := meanNums(data)
	devSum := new(big.Rat)
	for _, n := range data {
		d := new(big.Rat).Sub(n.val, avg.val)
		devSum.Add(devSum, d.Mul(d, d))
	}
	variance := round20(new(big.Rat).Quo(devSum, new(big.Rat).SetInt64(int64(len(data)))))
	return finite(sqrtRound20(variance))
}

// runArith parses the input, reduces it, and returns the result as a BigNumber
// dish, mirroring the shape of every arithmetic operation's run().
func runArith(in *core.Dish, delimName string, reduce func([]bigNum) bigNum) *core.Dish {
	res := reduce(createNumArray(in.String(), delimName))
	return core.NewDish([]byte(res.String()), core.TypeBigNumber)
}

// Sum adds a list of numbers.
type Sum struct{}

// Meta returns the operation metadata.
func (Sum) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Sum",
		Module:      "Default",
		Description: "Adds together a list of numbers. If an item in the string is not a number it is excluded from the list.",
		InfoURL:     "https://wikipedia.org/wiki/Summation",
		InputType:   core.TypeString,
		OutputType:  core.TypeBigNumber,
	}
}

// Args returns the argument definitions.
func (Sum) Args() []core.ArgDef {
	return []core.ArgDef{{Name: "Delimiter", Type: core.ArgOption, Value: arithmeticDelims}}
}

// Run computes the result. Ported from CyberChef Sum.mjs.
func (Sum) Run(in *core.Dish, args []any) (*core.Dish, error) {
	return runArith(in, args[0].(string), sumNums), nil
}

// Subtract subtracts a list of numbers.
type Subtract struct{}

// Meta returns the operation metadata.
func (Subtract) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Subtract",
		Module:      "Default",
		Description: "Subtracts a list of numbers. If an item in the string is not a number it is excluded from the list.",
		InfoURL:     "https://wikipedia.org/wiki/Subtraction",
		InputType:   core.TypeString,
		OutputType:  core.TypeBigNumber,
	}
}

// Args returns the argument definitions.
func (Subtract) Args() []core.ArgDef {
	return []core.ArgDef{{Name: "Delimiter", Type: core.ArgOption, Value: arithmeticDelims}}
}

// Run computes the result. Ported from CyberChef Subtract.mjs.
func (Subtract) Run(in *core.Dish, args []any) (*core.Dish, error) {
	return runArith(in, args[0].(string), func(d []bigNum) bigNum { return reduceNums(d, bigNum.minus) }), nil
}

// Multiply multiplies a list of numbers.
type Multiply struct{}

// Meta returns the operation metadata.
func (Multiply) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Multiply",
		Module:      "Default",
		Description: "Multiplies a list of numbers. If an item in the string is not a number it is excluded from the list.",
		InfoURL:     "https://wikipedia.org/wiki/Multiplication",
		InputType:   core.TypeString,
		OutputType:  core.TypeBigNumber,
	}
}

// Args returns the argument definitions.
func (Multiply) Args() []core.ArgDef {
	return []core.ArgDef{{Name: "Delimiter", Type: core.ArgOption, Value: arithmeticDelims}}
}

// Run computes the result. Ported from CyberChef Multiply.mjs.
func (Multiply) Run(in *core.Dish, args []any) (*core.Dish, error) {
	return runArith(in, args[0].(string), func(d []bigNum) bigNum { return reduceNums(d, bigNum.times) }), nil
}

// Divide divides a list of numbers.
type Divide struct{}

// Meta returns the operation metadata.
func (Divide) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Divide",
		Module:      "Default",
		Description: "Divides a list of numbers. If an item in the string is not a number it is excluded from the list.",
		InfoURL:     "https://wikipedia.org/wiki/Division_(mathematics)",
		InputType:   core.TypeString,
		OutputType:  core.TypeBigNumber,
	}
}

// Args returns the argument definitions.
func (Divide) Args() []core.ArgDef {
	return []core.ArgDef{{Name: "Delimiter", Type: core.ArgOption, Value: arithmeticDelims}}
}

// Run computes the result. Ported from CyberChef Divide.mjs.
func (Divide) Run(in *core.Dish, args []any) (*core.Dish, error) {
	return runArith(in, args[0].(string), func(d []bigNum) bigNum { return reduceNums(d, bigNum.div) }), nil
}

// Mean computes the mean (average) of a list of numbers.
type Mean struct{}

// Meta returns the operation metadata.
func (Mean) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Mean",
		Module:      "Default",
		Description: "Computes the mean (average) of a number list. If an item in the string is not a number it is excluded from the list.",
		InfoURL:     "https://wikipedia.org/wiki/Arithmetic_mean",
		InputType:   core.TypeString,
		OutputType:  core.TypeBigNumber,
	}
}

// Args returns the argument definitions.
func (Mean) Args() []core.ArgDef {
	return []core.ArgDef{{Name: "Delimiter", Type: core.ArgOption, Value: arithmeticDelims}}
}

// Run computes the result. Ported from CyberChef Mean.mjs.
func (Mean) Run(in *core.Dish, args []any) (*core.Dish, error) {
	return runArith(in, args[0].(string), meanNums), nil
}

// Median computes the median of a list of numbers.
type Median struct{}

// Meta returns the operation metadata.
func (Median) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Median",
		Module:      "Default",
		Description: "Computes the median of a number list. If an item in the string is not a number it is excluded from the list.",
		InfoURL:     "https://wikipedia.org/wiki/Median",
		InputType:   core.TypeString,
		OutputType:  core.TypeBigNumber,
	}
}

// Args returns the argument definitions.
func (Median) Args() []core.ArgDef {
	return []core.ArgDef{{Name: "Delimiter", Type: core.ArgOption, Value: arithmeticDelims}}
}

// Run computes the result. Ported from CyberChef Median.mjs.
func (Median) Run(in *core.Dish, args []any) (*core.Dish, error) {
	return runArith(in, args[0].(string), medianNums), nil
}

// StandardDeviation computes the standard deviation of a list of numbers.
type StandardDeviation struct{}

// Meta returns the operation metadata.
func (StandardDeviation) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Standard Deviation",
		Module:      "Default",
		Description: "Computes the standard deviation of a number list. If an item in the string is not a number it is excluded from the list.",
		InfoURL:     "https://wikipedia.org/wiki/Standard_deviation",
		InputType:   core.TypeString,
		OutputType:  core.TypeBigNumber,
	}
}

// Args returns the argument definitions.
func (StandardDeviation) Args() []core.ArgDef {
	return []core.ArgDef{{Name: "Delimiter", Type: core.ArgOption, Value: arithmeticDelims}}
}

// Run computes the result. Ported from CyberChef StandardDeviation.mjs.
func (StandardDeviation) Run(in *core.Dish, args []any) (*core.Dish, error) {
	return runArith(in, args[0].(string), stdDevNums), nil
}
