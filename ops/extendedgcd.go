package ops

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(ExtendedGCD{})
}

// ExtendedGCD finds the greatest common divisor of two numbers along with the
// coefficients that express it.
type ExtendedGCD struct{}

// Meta returns the operation metadata.
func (ExtendedGCD) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Extended GCD",
		Module:      "Crypto",
		Description: "Computes the Extended Euclidean Algorithm for integers a and b.\n\nFinds integers x and y (Bezout coefficients) such that:\na*x + b*y = gcd(a, b)\n\nThis is fundamental to many number theory algorithms including modular inverse, solving linear Diophantine equations, and cryptographic operations.\n\nInput handling: If either a or b is left blank, its value is taken from the Input field.",
		InfoURL:     "https://wikipedia.org/wiki/Extended_Euclidean_algorithm",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the two values, either of which may be left blank to take it
// from the input instead.
func (ExtendedGCD) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Value a", Type: core.ArgString, Value: ""},
		{Name: "Value b", Type: core.ArgString, Value: ""},
	}
}

// Run reports the divisor and the coefficients.
func (ExtendedGCD) Run(in *core.Dish, args []any) (*core.Dish, error) {
	aStr, bStr, err := twoValues(in.String(), args[0].(string), args[1].(string), "Value a", "Value b")
	if err != nil {
		return nil, err
	}
	a, err := parseBigInt(aStr, "Value a")
	if err != nil {
		return nil, err
	}
	b, err := parseBigInt(bStr, "Value b")
	if err != nil {
		return nil, err
	}

	g, x, y := egcd(a, b)

	var sb strings.Builder
	fmt.Fprintf(&sb, "gcd: %s\n\n", new(big.Int).Abs(g))
	sb.WriteString("Bezout coefficients:\n")
	fmt.Fprintf(&sb, "x = %s\n", x)
	fmt.Fprintf(&sb, "y = %s\n\n", y)
	return core.NewDish([]byte(sb.String()), core.TypeString), nil
}
