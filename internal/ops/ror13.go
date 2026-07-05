package ops

import (
	"fmt"
	"math/bits"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(ROR13{})
}

// ROR13 computes the ROR13 hash used in Windows API-name hashing techniques.
type ROR13 struct{}

// Meta returns the operation metadata.
func (ROR13) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "ROR13",
		Module:      "Default",
		Description: "Computes a ROR13 hash used in API hashing techniques.",
		InfoURL:     "",
		InputType:   core.TypeByteArray,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ROR13) Args() []core.ArgDef { return nil }

// Run computes the hash. Ported from CyberChef ROR13.mjs: rotate the 32-bit
// accumulator right 13 bits then add each byte, emitting "0x" + 8-digit
// uppercase hex.
func (ROR13) Run(in *core.Dish, args []any) (*core.Dish, error) {
	var hash uint32
	for _, chr := range in.Bytes() {
		hash = bits.RotateLeft32(hash, -13) + uint32(chr)
	}
	return core.NewDish(fmt.Appendf(nil, "0x%08X", hash), core.TypeString), nil
}
