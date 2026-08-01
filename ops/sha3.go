package ops

import (
	"crypto/sha3"
	"encoding/hex"
	"fmt"
	"hash"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(SHA3{})
}

// SHA3 computes a SHA-3 digest at the selected output size.
type SHA3 struct{}

// Meta returns the operation metadata.
func (SHA3) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "SHA3",
		Module:      "Crypto",
		Description: "Computes the SHA-3 (Keccak) hash digest of the input at the selected size, output as a lower-case hex string.",
		InfoURL:     "https://wikipedia.org/wiki/SHA-3",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (SHA3) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Size", Type: core.ArgOption, Value: []string{"512", "384", "256", "224"}},
	}
}

// Run computes the digest. Ported from CyberChef SHA3.mjs.
func (SHA3) Run(in *core.Dish, args []any) (*core.Dish, error) {
	var h hash.Hash
	switch args[0].(string) {
	case "224":
		h = sha3.New224()
	case "256":
		h = sha3.New256()
	case "384":
		h = sha3.New384()
	case "512":
		h = sha3.New512()
	default:
		return nil, fmt.Errorf("invalid size %q", args[0])
	}
	h.Write(in.Bytes())
	return core.NewDish([]byte(hex.EncodeToString(h.Sum(nil))), core.TypeString), nil
}
