package ops

import "github.com/roberson-io/cchef/internal/core"

func init() {
	core.Register(AtbashCipher{})
}

// AtbashCipher maps each letter to its mirror in the alphabet (a<->z, b<->y).
// It is the affine cipher with a=25, b=25, and is its own inverse.
type AtbashCipher struct{}

// Meta returns the operation metadata.
func (AtbashCipher) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Atbash Cipher",
		Module:      "Ciphers",
		Description: "Atbash is a mono-alphabetic substitution cipher originally used to encode the Hebrew alphabet. It has been modified here for use with the Latin alphabet.",
		InfoURL:     "https://wikipedia.org/wiki/Atbash",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (AtbashCipher) Args() []core.ArgDef {
	return nil
}

// Run applies the cipher. Ported from CyberChef AtbashCipher.mjs, which calls
// affineEncode(input, [25, 25]).
func (AtbashCipher) Run(in *core.Dish, args []any) (*core.Dish, error) {
	out := affineMap(in.String(), func(idx int) int {
		return (25*idx + 25) % 26
	})
	return core.NewDish([]byte(out), core.TypeString), nil
}
