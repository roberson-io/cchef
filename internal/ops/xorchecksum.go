package ops

import (
	"errors"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(XORChecksum{})
}

// XORChecksum computes a block-wise XOR checksum: the input is split into blocks
// of the given size, and the checksum is the XOR of all blocks (a short final
// block leaves the trailing positions unchanged). Ported from CyberChef's
// XORChecksum; the result is returned as hex.
type XORChecksum struct{}

// Meta returns the operation metadata.
func (XORChecksum) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "XOR Checksum",
		Module:      "Default",
		Description: "XOR Checksum splits the input into blocks of a configurable block size, and then XORs together all of these blocks.",
		InfoURL:     "https://wikipedia.org/wiki/XOR",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (XORChecksum) Args() []core.ArgDef {
	return []core.ArgDef{{Name: "Blocksize", Type: core.ArgNumber, Value: 4}}
}

// Run computes the block-wise XOR checksum.
func (XORChecksum) Run(in *core.Dish, args []any) (*core.Dish, error) {
	bs := args[0].(float64)
	if bs != float64(int(bs)) || bs <= 0 {
		//nolint:staticcheck,revive // verbatim CyberChef OperationError text
		return nil, errors.New("Blocksize must be a positive integer.")
	}
	blocksize := int(bs)

	res := make([]byte, blocksize)
	data := in.Bytes()
	for start := 0; start < len(data); start += blocksize {
		for i := 0; i < blocksize && start+i < len(data); i++ {
			res[i] ^= data[start+i]
		}
	}
	return core.NewDish([]byte(toHex(res, "", "")), core.TypeString), nil
}
