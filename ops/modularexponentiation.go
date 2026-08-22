package ops

import (
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/opsutil"
)

// reModExpHex and reModExpDec are the two forms a value may take, transcribed
// from CyberChef's parseBigInt. A decimal may carry a sign; a hex literal may
// not.
var (
	reModExpHex = regexp.MustCompile(`(?i)^0x[0-9a-f]+$`)
	reModExpDec = regexp.MustCompile(`^[+-]?[0-9]+$`)
)

func init() { core.Register(ModularExponentiation{}) }

// ModularExponentiation raises a base to an exponent modulo a modulus.
type ModularExponentiation struct{}

// Meta returns the operation metadata.
func (ModularExponentiation) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Modular Exponentiation",
		Module:      "Crypto",
		Description: "Performs modular exponentiation, as used in Diffie-Hellman and RSA.",
		InfoURL:     "https://wikipedia.org/wiki/Modular_exponentiation",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ModularExponentiation) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Base", Type: core.ArgString},
		{Name: "Modulus", Type: core.ArgString, Value: "1"},
		{Name: "Exponent", Type: core.ArgString},
	}
}

// Run computes the base raised to the exponent, modulo the modulus.
func (ModularExponentiation) Run(in *core.Dish, args []any) (*core.Dish, error) {
	baseArg, _ := args[0].(string)
	modArg, _ := args[1].(string)
	expArg, _ := args[2].(string)

	baseArg, modArg, expArg = strings.TrimSpace(baseArg), strings.TrimSpace(modArg), strings.TrimSpace(expArg)
	input := strings.TrimSpace(dishText(in))

	if modArg == "" {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return nil, errors.New("Modulus must be defined")
	}

	baseText, expText, err := modExpOperands(baseArg, expArg, input)
	if err != nil {
		return nil, err
	}

	base, err := modExpParse(baseText, "Base")
	if err != nil {
		return nil, err
	}
	exponent, err := modExpParse(expText, "Exponent")
	if err != nil {
		return nil, err
	}
	modulus, err := modExpParse(modArg, "Modulus")
	if err != nil {
		return nil, err
	}
	if modulus.Sign() == 0 {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return nil, errors.New("Modulus must be greater than zero")
	}

	out := modExpPow(base, exponent, modulus)
	return core.NewDish(opsutil.TextAsBytes(out.String()), core.TypeString), nil
}

// modExpOperands settles which of the base and the exponent comes from the
// input. Either box may be left empty to take the input, but not both: with
// neither given there is one value for two slots, which CyberChef refuses
// rather than guessing at.
func modExpOperands(base, exponent, input string) (string, string, error) {
	switch {
	case base != "" && exponent != "":
		return base, exponent, nil
	case base == "" && exponent != "":
		if input == "" {
			//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
			return "", "", errors.New("Base must be defined")
		}
		return input, exponent, nil
	case base != "" && exponent == "":
		if input == "" {
			//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
			return "", "", errors.New("Exponent must be defined")
		}
		return base, input, nil
	case input == "":
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return "", "", errors.New("Base and Exponent must be defined")
	}
	//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	return "", "", errors.New("Ambiguous input: specify either Base or Exponent when using Input")
}

// modExpParse reads a value written in decimal, optionally signed, or in hex
// with an "0x" prefix.
func modExpParse(value, param string) (*big.Int, error) {
	switch {
	case reModExpHex.MatchString(value):
		n, _ := new(big.Int).SetString(value[2:], 16)
		return n, nil
	case reModExpDec.MatchString(value):
		n, _ := new(big.Int).SetString(value, 10)
		return n, nil
	}
	return nil, fmt.Errorf("%s must be decimal or hex (0x...)", param)
}

// modExpPow squares and multiplies its way to the answer, reproducing
// CyberChef's loop rather than deferring to [big.Int.Exp].
//
// Two things differ from the textbook. Reduction uses a remainder whose sign
// follows the dividend, as JavaScript's "%" does, so a negative base gives a
// negative residue instead of the one in [0, modulus). And the loop runs only
// while the exponent is above zero, so a negative exponent does no work at all
// and the answer is 1, where [big.Int.Exp] would return a modular inverse.
func modExpPow(base, exponent, modulus *big.Int) *big.Int {
	result := big.NewInt(1)
	b := new(big.Int).Rem(base, modulus)
	e := new(big.Int).Set(exponent)

	for e.Sign() > 0 {
		if e.Bit(0) == 1 {
			result.Rem(result.Mul(result, b), modulus)
		}
		b.Rem(b.Mul(b, b), modulus)
		e.Rsh(e, 1)
	}
	return result
}
