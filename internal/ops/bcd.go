package ops

import (
	"fmt"
	"math/big"
	"slices"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(ToBCD{})
	core.Register(FromBCD{})
}

// bcdEncodingSchemes lists the supported BCD encoding schemes (lib/BCD.mjs).
var bcdEncodingSchemes = []string{
	"8 4 2 1", "7 4 2 1", "4 2 2 1", "2 4 2 1", "8 4 -2 -1", "Excess-3", "IBM 8 4 2 1",
}

// bcdEncodingLookup maps each scheme to the binary value of digits 0-9.
var bcdEncodingLookup = map[string][]int{
	"8 4 2 1":     {0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
	"7 4 2 1":     {0, 1, 2, 3, 4, 5, 6, 8, 9, 10},
	"4 2 2 1":     {0, 1, 4, 5, 8, 9, 12, 13, 14, 15},
	"2 4 2 1":     {0, 1, 2, 3, 4, 11, 12, 13, 14, 15},
	"8 4 -2 -1":   {0, 7, 6, 5, 4, 11, 10, 9, 8, 15},
	"Excess-3":    {3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
	"IBM 8 4 2 1": {10, 1, 2, 3, 4, 5, 6, 7, 8, 9},
}

// bcdFormats lists the BCD input/output formats.
var bcdFormats = []string{"Nibbles", "Bytes", "Raw"}

// ToBCD encodes a decimal number as Binary-Coded Decimal.
type ToBCD struct{}

// Meta returns the operation metadata.
func (ToBCD) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "To BCD",
		Module:      "Default",
		Description: "Binary-Coded Decimal (BCD) is a class of binary encodings of decimal numbers where each decimal digit is represented by a fixed number of bits, usually four or eight. Special bit patterns are sometimes used for a sign.",
		InfoURL:     "https://wikipedia.org/wiki/Binary-coded_decimal",
		InputType:   core.TypeBigNumber,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ToBCD) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Scheme", Type: core.ArgOption, Value: bcdEncodingSchemes},
		{Name: "Packed", Type: core.ArgBoolean, Value: true},
		{Name: "Signed", Type: core.ArgBoolean, Value: false},
		{Name: "Output format", Type: core.ArgOption, Value: bcdFormats},
	}
}

// Run encodes the input. Ported from ToBCD.mjs.
func (ToBCD) Run(in *core.Dish, args []any) (*core.Dish, error) {
	s := strings.TrimSpace(in.String())
	if strings.ContainsRune(s, '.') {
		return nil, fmt.Errorf("fractional values are not supported by BCD")
	}
	n, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return nil, fmt.Errorf("invalid input")
	}

	encoding := bcdEncodingLookup[args[0].(string)]
	packed := args[1].(bool)
	signed := args[2].(bool)
	outputFormat := args[3].(string)

	// Split the absolute value into its decimal digits.
	absStr := new(big.Int).Abs(n).String()
	nibbles := make([]int, 0, len(absStr)+2)
	for _, d := range absStr {
		nibbles = append(nibbles, encoding[int(d-'0')])
	}

	if signed {
		if packed && len(absStr)%2 == 0 {
			// Prepend a leading 0 so the sign nibble doesn't sit alone in a byte,
			// which would be ambiguous with a trailing 0 digit.
			nibbles = append([]int{encoding[0]}, nibbles...)
		}
		if n.Sign() > 0 {
			nibbles = append(nibbles, 12) // "C" for + (credit)
		} else {
			nibbles = append(nibbles, 13) // "D" for - (debit)
		}
	}

	var bytes []int
	if packed {
		encoded, little := 0, false
		for _, nb := range nibbles {
			if little {
				encoded ^= nb
			} else {
				encoded ^= nb << 4
			}
			if little {
				bytes = append(bytes, encoded)
				encoded = 0
			}
			little = !little
		}
		if little {
			bytes = append(bytes, encoded)
		}
	} else {
		bytes = nibbles
		// Add null high nibbles: [n] -> [0, n].
		interleaved := make([]int, 0, len(nibbles)*2)
		for _, nb := range nibbles {
			interleaved = append(interleaved, 0, nb)
		}
		nibbles = interleaved
	}

	switch outputFormat {
	case "Nibbles":
		parts := make([]string, len(nibbles))
		for i, nb := range nibbles {
			parts[i] = fmt.Sprintf("%04b", nb)
		}
		return core.NewDish([]byte(strings.Join(parts, " ")), core.TypeString), nil
	case "Bytes":
		parts := make([]string, len(bytes))
		for i, b := range bytes {
			parts[i] = fmt.Sprintf("%08b", b)
		}
		return core.NewDish([]byte(strings.Join(parts, " ")), core.TypeString), nil
	default: // "Raw"
		raw := make([]byte, len(bytes))
		for i, b := range bytes {
			raw[i] = byte(b)
		}
		return core.NewDish(raw, core.TypeString), nil
	}
}

// FromBCD decodes a Binary-Coded Decimal value into a decimal number.
type FromBCD struct{}

// Meta returns the operation metadata.
func (FromBCD) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "From BCD",
		Module:      "Default",
		Description: "Binary-Coded Decimal (BCD) is a class of binary encodings of decimal numbers where each decimal digit is represented by a fixed number of bits, usually four or eight. Special bit patterns are sometimes used for a sign.",
		InfoURL:     "https://wikipedia.org/wiki/Binary-coded_decimal",
		InputType:   core.TypeString,
		OutputType:  core.TypeBigNumber,
	}
}

// Args returns the argument definitions.
func (FromBCD) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Scheme", Type: core.ArgOption, Value: bcdEncodingSchemes},
		{Name: "Packed", Type: core.ArgBoolean, Value: true},
		{Name: "Signed", Type: core.ArgBoolean, Value: false},
		{Name: "Input format", Type: core.ArgOption, Value: bcdFormats},
	}
}

// Run decodes the input. Ported from FromBCD.mjs.
func (FromBCD) Run(in *core.Dish, args []any) (*core.Dish, error) {
	encoding := bcdEncodingLookup[args[0].(string)]
	packed := args[1].(bool)
	signed := args[2].(bool)
	inputFormat := args[3].(string)

	var nibbles []int
	switch inputFormat {
	case "Nibbles", "Bytes":
		s := whitespaceRE.ReplaceAllString(in.String(), "")
		for i := 0; i < len(s); i += 4 {
			end := min(i+4, len(s))
			var v int
			for _, c := range s[i:end] {
				if c != '0' && c != '1' {
					return nil, fmt.Errorf("invalid input")
				}
				v = v<<1 | int(c-'0')
			}
			nibbles = append(nibbles, v)
		}
	default: // "Raw"
		for _, b := range in.Bytes() {
			nibbles = append(nibbles, int(b>>4), int(b&15))
		}
	}

	if !packed {
		// Discard each high nibble. Faithfully reproduces CyberChef's
		// splice-in-loop (which shifts indices as it goes).
		for i := 0; i < len(nibbles); i++ {
			nibbles = append(nibbles[:i], nibbles[i+1:]...)
		}
	}

	var output strings.Builder
	if signed && len(nibbles) > 0 {
		sign := nibbles[len(nibbles)-1]
		nibbles = nibbles[:len(nibbles)-1]
		if sign == 13 || sign == 11 {
			output.WriteByte('-')
		}
	}

	for _, nb := range nibbles {
		val := slices.Index(encoding, nb)
		if val < 0 {
			return nil, fmt.Errorf("value %04b is not in the encoding scheme", nb)
		}
		// val is a decimal digit 0-9 (index into the 10-entry encoding table).
		output.WriteByte(byte('0' + val))
	}

	n, ok := new(big.Int).SetString(output.String(), 10)
	if !ok {
		return nil, fmt.Errorf("invalid input")
	}
	return core.NewDish([]byte(n.String()), core.TypeBigNumber), nil
}
