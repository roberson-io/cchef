package ops

import (
	"encoding/hex"
	"math/big"

	"github.com/roberson-io/cchef/internal/core"
)

const base62Alphabet = "0-9A-Za-z"

func init() {
	core.Register(ToBase62{})
	core.Register(FromBase62{})
}

// bigIntToBaseN renders a non-negative big.Int in the radix defined by the
// given alphabet (the radix is len(alphabet)).
func bigIntToBaseN(n *big.Int, alphabet []rune) string {
	if n.Sign() == 0 {
		return string(alphabet[0])
	}
	base := big.NewInt(int64(len(alphabet)))
	m := new(big.Int).Set(n)
	mod := new(big.Int)
	var digits []rune
	for m.Sign() > 0 {
		m.DivMod(m, base, mod)
		digits = append(digits, alphabet[mod.Int64()])
	}
	// Reverse (digits were produced least-significant first).
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}
	return string(digits)
}

// baseNToBigInt parses a string in the radix defined by the alphabet into a
// big.Int. Characters outside the alphabet are ignored.
func baseNToBigInt(s string, alphabet []rune) *big.Int {
	idx := make(map[rune]int, len(alphabet))
	for i, c := range alphabet {
		idx[c] = i
	}
	base := big.NewInt(int64(len(alphabet)))
	n := new(big.Int)
	d := new(big.Int)
	for _, c := range s {
		k, ok := idx[c]
		if !ok {
			continue
		}
		n.Mul(n, base)
		n.Add(n, d.SetInt64(int64(k)))
	}
	return n
}

// ToBase62 encodes raw bytes as a Base62 string.
type ToBase62 struct{}

// Meta returns the operation metadata.
func (ToBase62) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "To Base62",
		Module:      "Default",
		Description: "Base62 encodes arbitrary byte data using alphanumeric characters by treating the data as a large integer.",
		InfoURL:     "https://wikipedia.org/wiki/Base62",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ToBase62) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Alphabet", Type: core.ArgString, Value: base62Alphabet},
	}
}

// Run encodes the input. Ported from CyberChef ToBase62.mjs.
func (ToBase62) Run(in *core.Dish, args []any) (*core.Dish, error) {
	input := in.Bytes()
	if len(input) == 0 {
		return core.NewDish(nil, core.TypeString), nil
	}
	alphabet := []rune(expandAlphRange(args[0].(string)))
	n, _ := new(big.Int).SetString(hex.EncodeToString(input), 16)
	return core.NewDish([]byte(bigIntToBaseN(n, alphabet)), core.TypeString), nil
}

// FromBase62 decodes a Base62 string back into raw bytes.
type FromBase62 struct{}

// Meta returns the operation metadata.
func (FromBase62) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "From Base62",
		Module:      "Default",
		Description: "Decodes a Base62 string back into its raw byte value.",
		InfoURL:     "https://wikipedia.org/wiki/Base62",
		InputType:   core.TypeString,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (FromBase62) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Alphabet", Type: core.ArgString, Value: base62Alphabet},
	}
}

// Run decodes the input. Ported from CyberChef FromBase62.mjs.
func (FromBase62) Run(in *core.Dish, args []any) (*core.Dish, error) {
	if len(in.Bytes()) == 0 {
		return core.NewDish(nil, core.TypeByteArray), nil
	}
	alphabet := []rune(expandAlphRange(args[0].(string)))
	n := baseNToBigInt(in.String(), alphabet)

	h := n.Text(16)
	if len(h)%2 != 0 {
		h = "0" + h
	}
	out, err := hex.DecodeString(h)
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeByteArray), nil
}
