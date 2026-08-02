package ops

import (
	"fmt"
	"math/big"
	"regexp"
	"strings"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(TextIntegerConversion{})
}

// TextIntegerConversion converts between text and large integers (decimal or
// hexadecimal), interpreting text as a big-endian sequence of character codes.
type TextIntegerConversion struct{}

// Meta returns the operation metadata.
func (TextIntegerConversion) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Text-Integer Conversion",
		Module:      "Default",
		Description: "Converts between text strings and large integers (decimal or hexadecimal). Text is interpreted as a big-endian sequence of character codes, e.g. ABC is 0x414243 (hex) is 4276803 (decimal). Input is detected as decimal (digits only), hexadecimal (0x prefix), or text (quoted or unquoted). Text may only contain ASCII and Latin-1 characters (code point < 256); multi-byte Unicode characters generate an error.",
		InfoURL:     "https://wikipedia.org/wiki/Endianness",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (TextIntegerConversion) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Output format", Type: core.ArgOption, Value: []string{"String", "Decimal", "Hexadecimal"}},
	}
}

// textIntHexRE and textIntDecRE detect an integer input (hex with 0x prefix, or
// signed decimal), mirroring the regexes in TextIntegerConverter.mjs.
var (
	textIntHexRE   = regexp.MustCompile(`^0[xX][0-9a-fA-F]+$`)
	textIntDecRE   = regexp.MustCompile(`^[+-]?[0-9]+$`)
	textIntQuoteRE = regexp.MustCompile(`^["'].*["']$`)
)

// Run performs the conversion.
func (TextIntegerConversion) Run(in *core.Dish, args []any) (*core.Dish, error) {
	outputFormat := args[0].(string)
	trimmed := strings.TrimSpace(in.String())

	value := new(big.Int)
	switch {
	case trimmed == "":
		// Null input - treat as zero.
	case textIntHexRE.MatchString(trimmed):
		value.SetString(trimmed[2:], 16)
	case textIntDecRE.MatchString(trimmed):
		value.SetString(trimmed, 10)
	case textIntQuoteRE.MatchString(trimmed):
		// Quoted string: strip the surrounding quotes, then convert.
		v, err := textToBigInt([]rune(trimmed)[1 : len([]rune(trimmed))-1])
		if err != nil {
			return nil, err
		}
		value = v
	default:
		v, err := textToBigInt([]rune(trimmed))
		if err != nil {
			return nil, err
		}
		value = v
	}

	switch outputFormat {
	case "Decimal":
		return core.NewDish([]byte(value.Text(10)), core.TypeString), nil
	case "Hexadecimal":
		return core.NewDish([]byte("0x"+value.Text(16)), core.TypeString), nil
	default: // "String"
		return core.NewDish([]byte(bigIntToText(value)), core.TypeString), nil
	}
}

// textToBigInt interprets runes as a big-endian sequence of character codes.
// Ported from TextIntegerConverter.mjs; code points above 255 are rejected.
func textToBigInt(runes []rune) (*big.Int, error) {
	result := new(big.Int)
	for i, r := range runes {
		if r > 255 {
			return nil, fmt.Errorf(
				"character at position %d exceeds Latin-1 range (0-255).\n"+
					"Only ASCII and Latin-1 characters are supported", i,
			)
		}
		result.Lsh(result, 8).Or(result, big.NewInt(int64(r)))
	}
	return result, nil
}

// bigIntToText reverses textToBigInt: it emits the big-endian bytes of a
// positive value as characters. Zero and negative values yield the empty string,
// matching the upstream while (num > 0n) loop.
func bigIntToText(value *big.Int) string {
	if value.Sign() <= 0 {
		return ""
	}
	num := new(big.Int).Set(value)
	mask := big.NewInt(0xff)
	var bytes []rune
	for num.Sign() > 0 {
		b := new(big.Int).And(num, mask).Int64()
		bytes = append([]rune{rune(b)}, bytes...) // #nosec G115 -- big.Int byte (masked to 0xff) is 0-255
		num.Rsh(num, 8)
	}
	var sb strings.Builder
	for _, r := range bytes {
		sb.WriteRune(r)
	}
	return sb.String()
}
