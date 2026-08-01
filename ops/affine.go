package ops

import (
	"errors"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(AffineCipherEncode{})
	core.Register(AffineCipherDecode{})
}

// affineArgs validates the a and b arguments, mirroring CyberChef's checks: both
// must be non-negative integers, and a must be coprime to 26.
func affineArgs(args []any) (a, b int, err error) {
	af, bf := args[0].(float64), args[1].(float64)
	// CyberChef coerces each number to a string and tests it against
	// /^\+?(0|[1-9]\d*)$/, so only non-negative integers are accepted.
	if af != float64(int(af)) || bf != float64(int(bf)) || af < 0 || bf < 0 {
		return 0, 0, errors.New("The values of a and b can only be integers.") //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}
	a, b = int(af), int(bf)
	if gcd(a, 26) != 1 {
		return 0, 0, errors.New("The value of `a` must be coprime to 26.") //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}
	return a, b, nil
}

// AffineCipherEncode maps each letter to (a*x + b) % 26.
type AffineCipherEncode struct{}

// Meta returns the operation metadata.
func (AffineCipherEncode) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Affine Cipher Encode",
		Module:      "Ciphers",
		Description: "The Affine cipher is a type of monoalphabetic substitution cipher, wherein each letter in an alphabet is mapped to its numeric equivalent, encrypted using simple mathematical function, <code>(ax + b) % 26</code>, and converted back to a letter.",
		InfoURL:     "https://wikipedia.org/wiki/Affine_cipher",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (AffineCipherEncode) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "a", Type: core.ArgNumber, Value: 1},
		{Name: "b", Type: core.ArgNumber, Value: 0},
	}
}

// Run encodes the input. Ported from CyberChef affineEncode (lib/Ciphers.mjs).
func (AffineCipherEncode) Run(in *core.Dish, args []any) (*core.Dish, error) {
	a, b, err := affineArgs(args)
	if err != nil {
		return nil, err
	}
	out := affineMap(in.String(), func(idx int) int {
		return (a*idx + b) % 26
	})
	return core.NewDish([]byte(out), core.TypeString), nil
}

// AffineCipherDecode inverts the affine encoding: (y - b) * a⁻¹ % 26.
type AffineCipherDecode struct{}

// Meta returns the operation metadata.
func (AffineCipherDecode) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Affine Cipher Decode",
		Module:      "Ciphers",
		Description: "The Affine cipher is a type of monoalphabetic substitution cipher. To decrypt, each letter in an alphabet is mapped to its numeric equivalent, decrypted by a mathematical function, and converted back to a letter.",
		InfoURL:     "https://wikipedia.org/wiki/Affine_cipher",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (AffineCipherDecode) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "a", Type: core.ArgNumber, Value: 1},
		{Name: "b", Type: core.ArgNumber, Value: 0},
	}
}

// Run decodes the input. Ported from CyberChef AffineCipherDecode.mjs.
func (AffineCipherDecode) Run(in *core.Dish, args []any) (*core.Dish, error) {
	a, b, err := affineArgs(args)
	if err != nil {
		return nil, err
	}
	aModInv := modInv(a, 26)
	out := affineMap(in.String(), func(idx int) int {
		return mod26((idx-b)*aModInv, 26)
	})
	return core.NewDish([]byte(out), core.TypeString), nil
}

// affineMap applies fn to the 0-25 index of each alphabetic character, leaving
// case intact and passing non-alphabetic characters through unchanged.
func affineMap(input string, fn func(idx int) int) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyz"
	out := make([]rune, 0, len(input))
	for _, r := range input {
		switch {
		case r >= 'a' && r <= 'z':
			out = append(out, rune(alphabet[fn(int(r-'a'))]))
		case r >= 'A' && r <= 'Z':
			out = append(out, rune(alphabet[fn(int(r-'A'))]-32))
		default:
			out = append(out, r)
		}
	}
	return string(out)
}

// gcd returns the greatest common divisor of x and y.
func gcd(x, y int) int {
	if y == 0 {
		return x
	}
	return gcd(y, x%y)
}

// modInv returns the modular inverse of x mod 26, mirroring CyberChef
// Utils.modInv. The caller has verified gcd(x, 26) == 1, so an inverse in
// [1, 25] always exists and the search terminates.
func modInv(x, y int) int {
	x %= y
	for i := 1; ; i++ {
		if (x*i)%26 == 1 {
			return i
		}
	}
}

// mod26 is CyberChef Utils.mod: a positive modulo (((x % y) + y) % y).
func mod26(x, y int) int {
	return ((x % y) + y) % y
}
