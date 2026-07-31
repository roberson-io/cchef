package ops

import (
	"errors"
	"strconv"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(LuhnChecksum{})
}

// LuhnChecksum computes the Luhn mod-N checksum, check digit, and the validated
// string. Ported from CyberChef's LuhnChecksum, generalised to any even radix in
// [2, 36].
type LuhnChecksum struct{}

// Meta returns the operation metadata.
func (LuhnChecksum) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Luhn Checksum",
		Module:      "Default",
		Description: "The Luhn mod N algorithm using the english alphabet. The Luhn mod N algorithm is an extension to the Luhn algorithm (also known as mod 10 algorithm) that allows it to work with sequences of values in any even-numbered base.",
		InfoURL:     "https://wikipedia.org/wiki/Luhn_mod_N_algorithm",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (LuhnChecksum) Args() []core.ArgDef {
	return []core.ArgDef{{Name: "Radix", Type: core.ArgNumber, Integer: true, Value: 10}}
}

// Run computes the Luhn checksum report.
func (LuhnChecksum) Run(in *core.Dish, args []any) (*core.Dish, error) {
	input := in.String()
	if input == "" {
		return core.NewDish([]byte(""), core.TypeString), nil
	}
	radix := int(args[0].(float64))
	if radix < 2 || radix > 36 {
		//nolint:staticcheck,revive // verbatim CyberChef OperationError text
		return nil, errors.New("Error: Radix argument must be between 2 and 36")
	}
	if radix%2 != 0 {
		//nolint:staticcheck,revive // verbatim CyberChef OperationError text
		return nil, errors.New("Error: Radix argument must be divisible by 2")
	}

	sum, err := luhnSum(input, radix)
	if err != nil {
		return nil, err
	}
	// input is already validated above, and appending "0" cannot introduce an
	// invalid digit, so this call never errors.
	checkDigit, _ := luhnSum(input+"0", radix)
	if checkDigit != 0 {
		checkDigit = radix - checkDigit
	}

	checkSumStr := strconv.FormatInt(int64(sum), radix)
	checkDigitStr := strconv.FormatInt(int64(checkDigit), radix)
	out := "Checksum: " + checkSumStr + "\nCheckdigit: " + checkDigitStr +
		"\nLuhn Validated String: " + input + checkDigitStr
	return core.NewDish([]byte(out), core.TypeString), nil
}

// luhnSum computes the Luhn mod-radix weighted digit sum of s (mod radix),
// doubling every second digit from the right and summing the base-radix digits
// of the doubled value.
func luhnSum(s string, radix int) (int, error) {
	acc := 0
	even := false
	for i := len(s) - 1; i >= 0; i-- {
		d := luhnDigit(s[i])
		if d < 0 || d >= radix {
			return 0, errors.New("Character: " + string(s[i]) + " is not valid in radix " + strconv.Itoa(radix) + ".") //nolint:staticcheck,revive // verbatim CyberChef text
		}
		if even {
			d *= 2
			d = d/radix + d%radix
		}
		even = !even
		acc += d
	}
	return acc % radix, nil
}

// luhnDigit returns the value of an alphanumeric digit (0-9, a-z/A-Z → 10-35),
// or -1 if the byte is not one.
func luhnDigit(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'z':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'Z':
		return int(c-'A') + 10
	}
	return -1
}
