package ops

import (
	"crypto/sha3"
	"encoding/hex"
	"errors"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(Shake{})
}

// Shake is the SHAKE extendable-output function (XOF) of SHA-3, producing a
// variable-length digest. Ported from CyberChef's Shake: the Size argument is in
// bits and the output is floor(size/8) bytes of SHAKE output, as hex.
type Shake struct{}

// Meta returns the operation metadata.
func (Shake) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Shake",
		Module:      "Crypto",
		Description: "Shake is an Extendable Output Function (XOF) of the SHA-3 hash algorithm, part of the Keccak family, allowing for variable output length/size.",
		InfoURL:     "https://wikipedia.org/wiki/SHA-3#Instances",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (Shake) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Capacity", Type: core.ArgOption, Value: []string{"256", "128"}},
		{Name: "Size", Type: core.ArgNumber, Integer: true, Value: 512},
	}
}

// Run computes the SHAKE digest of the requested size.
func (Shake) Run(in *core.Dish, args []any) (*core.Dish, error) {
	capacity := args[0].(string)
	size := int(args[1].(float64))
	if size < 0 {
		//nolint:staticcheck,revive // verbatim CyberChef OperationError text
		return nil, errors.New("Size must be greater than 0")
	}

	n := size / 8
	var raw []byte
	if capacity == "128" {
		raw = sha3.SumSHAKE128(in.Bytes(), n)
	} else {
		raw = sha3.SumSHAKE256(in.Bytes(), n)
	}
	return core.NewDish([]byte(hex.EncodeToString(raw)), core.TypeString), nil
}
