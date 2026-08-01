package ops

import (
	"fmt"
	"hash/adler32"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(Adler32{})
}

// Adler32 computes the Adler-32 checksum of the input.
type Adler32 struct{}

// Meta returns the operation metadata.
func (Adler32) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Adler-32 Checksum",
		Module:      "Crypto",
		Description: "Computes the Adler-32 checksum of the input, output as an 8-digit hex string.",
		InfoURL:     "https://wikipedia.org/wiki/Adler-32",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (Adler32) Args() []core.ArgDef { return nil }

// Run computes the checksum.
func (Adler32) Run(in *core.Dish, args []any) (*core.Dish, error) {
	sum := adler32.Checksum(in.Bytes())
	return core.NewDish(fmt.Appendf(nil, "%08x", sum), core.TypeString), nil
}
