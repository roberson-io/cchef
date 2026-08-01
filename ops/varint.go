package ops

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(VarIntDecode{})
	core.Register(VarIntEncode{})
}

// VarIntDecode decodes a Protobuf-style LEB128 variable-length integer.
type VarIntDecode struct{}

// Meta returns the operation metadata.
func (VarIntDecode) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "VarInt Decode",
		Module:      "Default",
		Description: "Decodes a VarInt encoded integer. VarInt is an efficient way of encoding variable length integers and is commonly used with Protobuf.",
		InfoURL:     "https://developers.google.com/protocol-buffers/docs/encoding#varints",
		InputType:   core.TypeByteArray,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (VarIntDecode) Args() []core.ArgDef { return nil }

// Run decodes the VarInt. Ported from CyberChef VarIntDecode.mjs.
func (VarIntDecode) Run(in *core.Dish, args []any) (*core.Dish, error) {
	input := in.Bytes()
	result := new(big.Int)
	var offset uint
	for i := range input {
		part := new(big.Int).Lsh(big.NewInt(int64(input[i]&0x7f)), offset)
		result.Or(result, part)
		if input[i]&0x80 == 0 {
			break
		}
		offset += 7
	}
	return core.NewDish([]byte(result.String()), core.TypeString), nil
}

// VarIntEncode encodes a non-negative integer as an LEB128 variable-length int.
type VarIntEncode struct{}

// Meta returns the operation metadata.
func (VarIntEncode) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "VarInt Encode",
		Module:      "Default",
		Description: "Encodes a Vaint encoded integer. VarInt is an efficient way of encoding variable length integers and is commonly used with Protobuf.",
		InfoURL:     "https://developers.google.com/protocol-buffers/docs/encoding#varints",
		InputType:   core.TypeString,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (VarIntEncode) Args() []core.ArgDef { return nil }

// Run encodes the VarInt. Ported from CyberChef VarIntEncode.mjs.
func (VarIntEncode) Run(in *core.Dish, args []any) (*core.Dish, error) {
	value, ok := new(big.Int).SetString(strings.TrimSpace(in.String()), 10)
	if !ok {
		return nil, fmt.Errorf("input is not a valid integer")
	}
	if value.Sign() < 0 {
		return nil, fmt.Errorf("negative values cannot be represented as VarInt")
	}

	x80 := big.NewInt(0x80)
	mask := big.NewInt(0x7f)
	v := new(big.Int).Set(value)
	var out []byte
	for v.Cmp(x80) >= 0 {
		out = append(out, byte(new(big.Int).And(v, mask).Int64())|0x80) // #nosec G115 -- VarInt group masked to 7 bits
		v.Rsh(v, 7)
	}
	out = append(out, byte(v.Int64())) // #nosec G115 -- VarInt group masked to 7 bits
	return core.NewDish(out, core.TypeByteArray), nil
}
