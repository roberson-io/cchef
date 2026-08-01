package ops

import (
	"encoding/hex"
	"errors"

	"github.com/roberson-io/cchef/core"
)

// blake3KeyLen is the exact key length BLAKE3 keyed mode requires.
const blake3KeyLen = 32

func init() {
	core.Register(BLAKE3{})
}

// blake3SizeMin / blake3SizeMax bound the requested output length in bytes (the
// max is CyberChef's arbitrary resource-exhaustion limit).
var (
	blake3SizeMin float64 = 1
	blake3SizeMax float64 = 65535
)

// BLAKE3 hashes the input with BLAKE3, with an optional 32-byte key, producing a
// hex digest of the requested length. Ported from CyberChef BLAKE3.mjs (which
// wraps @noble/hashes); the algorithm is reimplemented from the BLAKE3 spec.
type BLAKE3 struct{}

// Meta returns the operation metadata.
func (BLAKE3) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "BLAKE3",
		Module:      "Hashing",
		Description: "Hashes the input using BLAKE3 (UTF-8 encoded), with an optional key (also UTF-8), and outputs the result in hexadecimal format.",
		InfoURL:     "https://en.wikipedia.org/wiki/BLAKE_(hash_function)#BLAKE3",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (BLAKE3) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Size (bytes)", Type: core.ArgNumber, Integer: true, Value: 16, Min: &blake3SizeMin, Max: &blake3SizeMax},
		{Name: "Key", Type: core.ArgString, Value: ""},
	}
}

// Run computes the BLAKE3 digest.
func (BLAKE3) Run(in *core.Dish, args []any) (*core.Dish, error) {
	size := int(args[0].(float64))
	key := args[1].(string)

	var h *blake3Hasher
	if key == "" {
		h = blake3New()
	} else {
		if len(key) != blake3KeyLen {
			//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
			return nil, errors.New("The key must be exactly 32 bytes long")
		}
		h = blake3NewKeyed([]byte(key))
	}
	h.update(in.Bytes())
	return core.NewDish([]byte(hex.EncodeToString(h.finalize(size))), core.TypeString), nil
}
