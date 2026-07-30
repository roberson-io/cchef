package ops

import (
	"errors"
	"math/big"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(ModularInverse{})
}

// ModularInverse finds the number that multiplies with a to give one, modulo m.
// Ported from CyberChef ModularInverse.mjs.
type ModularInverse struct{}

// Meta returns the operation metadata.
func (ModularInverse) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Modular Inverse",
		Module:      "Crypto",
		Description: "Computes the modular multiplicative inverse of a modulo m.\n\nFinds x such that a*x = 1 (mod m).\n\nInput handling: If either a or m is left blank, its value is taken from the Input field.",
		InfoURL:     "https://wikipedia.org/wiki/Modular_multiplicative_inverse",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the value and the modulus, either of which may be left blank to
// take it from the input instead.
func (ModularInverse) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Value (a)", Type: core.ArgString, Value: ""},
		{Name: "Modulus (m)", Type: core.ArgString, Value: ""},
	}
}

// Run reports the inverse, or says why there is not one.
func (ModularInverse) Run(in *core.Dish, args []any) (*core.Dish, error) {
	aStr, mStr, err := twoValues(in.String(), args[0].(string), args[1].(string), "Value (a)", "Modulus (m)")
	if err != nil {
		return nil, err
	}
	a, err := parseBigInt(aStr, "Value (a)")
	if err != nil {
		return nil, err
	}
	m, err := parseBigInt(mStr, "Modulus (m)")
	if err != nil {
		return nil, err
	}
	if m.Sign() <= 0 {
		return nil, errors.New("Modulus must be greater than zero") //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}

	// Bring the value into the range 0 to m before looking for the inverse.
	aNorm := new(big.Int).Mod(a, m)
	g, x, _ := egcd(aNorm, m)

	// An inverse exists only when the two share no factor.
	if g.CmpAbs(big.NewInt(1)) != 0 {
		return nil, errors.New("Inverse does not exist because gcd(a, m) ≠ 1") //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}
	// The value was brought into the range 0 to m and the modulus is positive,
	// so the divisor comes back positive and the coefficient needs no flip.
	inv := new(big.Int).Mod(x, m)

	return core.NewDish([]byte(inv.String()), core.TypeString), nil
}
