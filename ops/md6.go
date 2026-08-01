package ops

import (
	"encoding/hex"
	"errors"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(MD6{})
}

const (
	md6MinSize = 0   // minimum requested digest size (bits)
	md6MaxSize = 512 // maximum requested digest size (bits)
)

// MD6 computes the MD6 hash. Ported from CyberChef MD6.mjs (which wraps the
// self-contained node-md6 package); the algorithm is reimplemented from that
// package, which is a standard MD6 implementation.
type MD6 struct{}

// Meta returns the operation metadata.
func (MD6) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "MD6",
		Module:      "Crypto",
		Description: "The MD6 (Message-Digest 6) algorithm is a cryptographic hash function. It uses a Merkle tree-like structure to allow for immense parallel computation of hashes for very long inputs.",
		InfoURL:     "https://wikipedia.org/wiki/MD6",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (MD6) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Size", Type: core.ArgNumber, Integer: true, Value: 256},
		{Name: "Levels", Type: core.ArgNumber, Integer: true, Value: 64},
		{Name: "Key", Type: core.ArgString, Value: ""},
	}
}

// Run computes the MD6 digest.
func (MD6) Run(in *core.Dish, args []any) (*core.Dish, error) {
	size := int(args[0].(float64))
	levels := int(args[1].(float64))
	key := args[2].(string)
	if size < md6MinSize || size > md6MaxSize {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return nil, errors.New("Size must be between 0 and 512")
	}
	if levels < 0 {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return nil, errors.New("Levels must be greater than 0")
	}
	digest := md6Hash(size, md6Bytes(in.String()), md6Bytes(key), levels)
	return core.NewDish([]byte(hex.EncodeToString(digest)), core.TypeString), nil
}
