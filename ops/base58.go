package ops

import (
	"fmt"
	"slices"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/opsutil"
)

const base58Bitcoin = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

func init() {
	core.Register(ToBase58{})
	core.Register(FromBase58{})
}

// expandB58Alphabet expands and validates a 58-character alphabet.
func expandB58Alphabet(spec string) ([]rune, error) {
	alphabet := []rune(opsutil.ExpandAlphRange(spec))
	seen := make(map[rune]bool, len(alphabet))
	for _, c := range alphabet {
		seen[c] = true
	}
	if len(alphabet) != 58 || len(seen) != 58 {
		return nil, fmt.Errorf("alphabet must be of length 58")
	}
	return alphabet, nil
}

// ToBase58 encodes raw bytes as a Base58 string.
type ToBase58 struct{}

// Meta returns the operation metadata.
func (ToBase58) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "To Base58",
		Module:      "Default",
		Description: "Base58 encodes arbitrary byte data using a restricted alphabet that omits easily-confused characters. Commonly used for cryptocurrency addresses.",
		InfoURL:     "https://wikipedia.org/wiki/Binary-to-text_encoding#Base58",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ToBase58) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Alphabet", Type: core.ArgEditableOption, Value: base58Bitcoin},
	}
}

// Run encodes the input.
func (ToBase58) Run(in *core.Dish, args []any) (*core.Dish, error) {
	alphabet, err := expandB58Alphabet(args[0].(string))
	if err != nil {
		return nil, err
	}
	input := in.Bytes()
	if len(input) == 0 {
		return core.NewDish(nil, core.TypeString), nil
	}

	zeroPrefix := 0
	for zeroPrefix < len(input) && input[zeroPrefix] == 0 {
		zeroPrefix++
	}

	var result []int
	for _, b := range input {
		carry := int(b)
		for i := range result {
			carry += result[i] << 8
			result[i] = carry % 58
			carry /= 58
		}
		for carry > 0 {
			result = append(result, carry%58)
			carry /= 58
		}
	}

	out := make([]rune, 0, len(result)+zeroPrefix)
	for i := 0; i < zeroPrefix; i++ {
		out = append(out, alphabet[0])
	}
	for _, r := range slices.Backward(result) {
		out = append(out, alphabet[r])
	}
	return core.NewDish([]byte(string(out)), core.TypeString), nil
}

// FromBase58 decodes a Base58 string back into raw bytes.
type FromBase58 struct{}

// Meta returns the operation metadata.
func (FromBase58) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "From Base58",
		Module:      "Default",
		Description: "Decodes a Base58 string back into its raw byte value.",
		InfoURL:     "https://wikipedia.org/wiki/Binary-to-text_encoding#Base58",
		InputType:   core.TypeString,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (FromBase58) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Alphabet", Type: core.ArgEditableOption, Value: base58Bitcoin},
		{Name: "Remove non-alphabet chars", Type: core.ArgBoolean, Value: true},
	}
}

// Run decodes the input.
func (FromBase58) Run(in *core.Dish, args []any) (*core.Dish, error) {
	alphabet, err := expandB58Alphabet(args[0].(string))
	if err != nil {
		return nil, err
	}
	removeNonAlph := args[1].(bool)
	idx := make(map[rune]int, len(alphabet))
	for i, c := range alphabet {
		idx[c] = i
	}

	input := []rune(in.String())
	if len(input) == 0 {
		return core.NewDish(nil, core.TypeByteArray), nil
	}

	zeroPrefix := 0
	for zeroPrefix < len(input) && input[zeroPrefix] == alphabet[0] {
		zeroPrefix++
	}

	var result []int
	for pos, c := range input {
		index, ok := idx[c]
		if !ok {
			if removeNonAlph {
				continue
			}
			return nil, fmt.Errorf("char %q at position %d not in alphabet", c, pos)
		}
		carry := index
		for i := range result {
			carry += result[i] * 58
			result[i] = carry & 0xff
			carry >>= 8
		}
		for carry > 0 {
			result = append(result, carry&0xff)
			carry >>= 8
		}
	}

	for i := 0; i < zeroPrefix; i++ {
		result = append(result, 0)
	}

	out := make([]byte, len(result))
	for i, v := range result {
		out[len(result)-1-i] = byte(v) // #nosec G115 -- big.Int remainder is 0-255
	}
	return core.NewDish(out, core.TypeByteArray), nil
}
