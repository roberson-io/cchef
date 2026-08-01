package ops

import (
	"fmt"
	"math/big"
	"slices"
	"strings"

	"github.com/roberson-io/cchef/core"
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

// BCD sign nibble codes: the hex digits C/D (and B, an alternate negative) that
// mark the sign of a signed BCD value.
const (
	bcdSignPlus     = 12 // "C" — credit / positive
	bcdSignMinus    = 13 // "D" — debit / negative
	bcdSignMinusAlt = 11 // "B" — alternate negative
)

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

	nibbles, bytes := bcdEncodeNibbles(n, encoding, packed, signed)
	return bcdFormatOutput(nibbles, bytes, outputFormat), nil
}

// bcdEncodeNibbles converts n to BCD nibble codes using the given scheme,
// optionally appending a sign nibble and packing two nibbles per byte. It
// returns the nibble stream (used for "Nibbles" output) and the byte stream
// (used for "Bytes"/"Raw" output). When unpacked, the byte stream is the raw
// digit nibbles and the nibble stream is interleaved with null high nibbles.
func bcdEncodeNibbles(n *big.Int, encoding []int, packed, signed bool) (nibbles, bytes []int) {
	// Split the absolute value into its decimal digits.
	absStr := new(big.Int).Abs(n).String()
	nibbles = make([]int, 0, len(absStr)+2)
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
			nibbles = append(nibbles, bcdSignPlus)
		} else {
			nibbles = append(nibbles, bcdSignMinus)
		}
	}

	if packed {
		return nibbles, bcdPackNibbles(nibbles)
	}
	bytes = nibbles
	// Add null high nibbles: [n] -> [0, n].
	interleaved := make([]int, 0, len(nibbles)*2)
	for _, nb := range nibbles {
		interleaved = append(interleaved, 0, nb)
	}
	return interleaved, bytes
}

// bcdPackNibbles packs the nibble stream two nibbles per byte, high nibble
// first. An odd final nibble occupies the high half of a trailing byte.
func bcdPackNibbles(nibbles []int) []int {
	var bytes []int
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
	return bytes
}

// bcdFormatOutput renders the nibble/byte streams in the requested output
// format ("Nibbles" as 4-bit groups, "Bytes" as 8-bit groups, "Raw" as bytes).
func bcdFormatOutput(nibbles, bytes []int, outputFormat string) *core.Dish {
	switch outputFormat {
	case "Nibbles":
		parts := make([]string, len(nibbles))
		for i, nb := range nibbles {
			parts[i] = fmt.Sprintf("%04b", nb)
		}
		return core.NewDish([]byte(strings.Join(parts, " ")), core.TypeString)
	case "Bytes":
		parts := make([]string, len(bytes))
		for i, b := range bytes {
			parts[i] = fmt.Sprintf("%08b", b)
		}
		return core.NewDish([]byte(strings.Join(parts, " ")), core.TypeString)
	default: // "Raw"
		raw := make([]byte, len(bytes))
		for i, b := range bytes {
			raw[i] = byte(b) // #nosec G115 -- nibble/digit value bounded to a byte
		}
		return core.NewDish(raw, core.TypeString)
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

	nibbles, err := bcdParseNibbles(in.Bytes(), inputFormat)
	if err != nil {
		return nil, err
	}

	if !packed {
		// Discard each high nibble. Faithfully reproduces CyberChef's
		// splice-in-loop (which shifts indices as it goes).
		for i := 0; i < len(nibbles); i++ {
			nibbles = append(nibbles[:i], nibbles[i+1:]...)
		}
	}

	sign := ""
	if signed && len(nibbles) > 0 {
		last := nibbles[len(nibbles)-1]
		nibbles = nibbles[:len(nibbles)-1]
		if last == bcdSignMinus || last == bcdSignMinusAlt {
			sign = "-"
		}
	}

	digits, err := bcdDecodeDigits(nibbles, encoding)
	if err != nil {
		return nil, err
	}

	n, ok := new(big.Int).SetString(sign+digits, 10)
	if !ok {
		return nil, fmt.Errorf("invalid input")
	}
	return core.NewDish([]byte(n.String()), core.TypeBigNumber), nil
}

// bcdParseNibbles reads the raw BCD bytes into a nibble stream. For the
// "Nibbles"/"Bytes" formats the input is a whitespace-separated binary string
// read four bits at a time; for "Raw" each input byte contributes its two
// nibbles (high then low).
func bcdParseNibbles(input []byte, inputFormat string) ([]int, error) {
	var nibbles []int
	switch inputFormat {
	case "Nibbles", "Bytes":
		s := whitespaceRE.ReplaceAllString(string(input), "")
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
		for _, b := range input {
			nibbles = append(nibbles, int(b>>4), int(b&15))
		}
	}
	return nibbles, nil
}

// bcdDecodeDigits maps each nibble code back to its decimal digit via the
// scheme's lookup table, erroring on any code not in the scheme.
func bcdDecodeDigits(nibbles, encoding []int) (string, error) {
	var out strings.Builder
	for _, nb := range nibbles {
		val := slices.Index(encoding, nb)
		if val < 0 {
			return "", fmt.Errorf("value %04b is not in the encoding scheme", nb)
		}
		// val is a decimal digit 0-9 (index into the 10-entry encoding table).
		out.WriteByte(byte('0' + val)) // #nosec G115 -- nibble/digit value bounded to a byte
	}
	return out.String(), nil
}
