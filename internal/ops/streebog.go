package ops

import (
	"encoding/hex"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(Streebog{})
}

// Streebog computes the GOST R 34.11-2012 "Streebog" hash. Ported from
// CyberChef's Streebog operation, which wraps the same GOST digest engine as
// GOST Hash (see gosthash.go's gostDigest2012).
type Streebog struct{}

// Meta returns the operation metadata.
func (Streebog) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Streebog",
		Module:      "Hashing",
		Description: "Streebog is a cryptographic hash function defined in the Russian national standard GOST R 34.11-2012 Information Technology – Cryptographic Information Security – Hash Function. It was created to replace an obsolete GOST hash function defined in the old standard GOST R 34.11-94, and as an asymmetric reply to SHA-3 competition by the US National Institute of Standards and Technology.",
		InfoURL:     "https://wikipedia.org/wiki/Streebog",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (Streebog) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Digest length", Type: core.ArgOption, Value: []string{"256", "512"}},
	}
}

// Run computes the Streebog digest.
func (Streebog) Run(in *core.Dish, args []any) (*core.Dish, error) {
	bitLength := 256
	if args[0].(string) == "512" {
		bitLength = 512
	}
	digest := gostDigest2012(in.Bytes(), bitLength)
	return core.NewDish([]byte(hex.EncodeToString(digest)), core.TypeString), nil
}
