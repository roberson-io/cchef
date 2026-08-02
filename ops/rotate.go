package ops

import (
	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(RotateLeft{})
	core.Register(RotateRight{})
}

// rotBytes rotates each byte independently by applying algo `amount` times,
// mirroring CyberChef's lib/Rotate.mjs rot().
func rotBytes(data []byte, amount int, algo func(byte) byte) []byte {
	out := make([]byte, len(data))
	for i, b := range data {
		for range amount {
			b = algo(b)
		}
		out[i] = b
	}
	return out
}

// rotlByte rotates a single byte left by one bit; rotrByte rotates right.
func rotlByte(b byte) byte { return (b << 1) | (b >> 7) }
func rotrByte(b byte) byte { return (b >> 1) | (b << 7) }

// rotlCarry rotates the whole byte array left by amount bits, wrapping the bits
// that fall off the front around to the end.
func rotlCarry(data []byte, amount int) []byte {
	out := make([]byte, len(data))
	if len(data) == 0 {
		return out
	}
	amount %= 8
	var carry byte
	for i := len(data) - 1; i >= 0; i-- {
		old := data[i]
		out[i] = (old << amount) | carry
		carry = byte((int(old) >> (8 - amount)) & ((1 << amount) - 1)) // #nosec G115 -- bit-rotation carry masked to a byte
	}
	out[len(data)-1] |= carry
	return out
}

// rotrCarry rotates the whole byte array right by amount bits, wrapping the bits
// that fall off the end around to the front.
func rotrCarry(data []byte, amount int) []byte {
	out := make([]byte, len(data))
	if len(data) == 0 {
		return out
	}
	amount %= 8
	var carry byte
	for i := range data {
		old := data[i]
		out[i] = (old >> amount) | carry
		carry = byte((int(old) & ((1 << amount) - 1)) << (8 - amount)) // #nosec G115 -- bit-rotation carry masked to a byte
	}
	out[0] |= carry
	return out
}

// RotateLeft rotates the bits of each byte left, optionally carrying overflow
// bits into the next byte.
type RotateLeft struct{}

// Meta returns the operation metadata.
func (RotateLeft) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Rotate left",
		Module:      "Default",
		Description: "Rotates each byte to the left by the number of bits specified, optionally carrying the excess bits over to the next byte. Currently only supports 8-bit values.",
		InfoURL:     "https://wikipedia.org/wiki/Bitwise_operation#Bit_shifts",
		InputType:   core.TypeByteArray,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (RotateLeft) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Amount", Type: core.ArgNumber, Integer: true, Value: 1},
		{Name: "Carry through", Type: core.ArgBoolean, Value: false},
	}
}

// Run rotates left.
func (RotateLeft) Run(in *core.Dish, args []any) (*core.Dish, error) {
	amount := int(args[0].(float64))
	if args[1].(bool) {
		return core.NewDish(rotlCarry(in.Bytes(), amount), core.TypeByteArray), nil
	}
	return core.NewDish(rotBytes(in.Bytes(), amount, rotlByte), core.TypeByteArray), nil
}

// RotateRight rotates the bits of each byte right, optionally carrying overflow
// bits into the next byte.
type RotateRight struct{}

// Meta returns the operation metadata.
func (RotateRight) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Rotate right",
		Module:      "Default",
		Description: "Rotates each byte to the right by the number of bits specified, optionally carrying the excess bits over to the next byte. Currently only supports 8-bit values.",
		InfoURL:     "https://wikipedia.org/wiki/Bitwise_operation#Bit_shifts",
		InputType:   core.TypeByteArray,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (RotateRight) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Amount", Type: core.ArgNumber, Integer: true, Value: 1},
		{Name: "Carry through", Type: core.ArgBoolean, Value: false},
	}
}

// Run rotates right.
func (RotateRight) Run(in *core.Dish, args []any) (*core.Dish, error) {
	amount := int(args[0].(float64))
	if args[1].(bool) {
		return core.NewDish(rotrCarry(in.Bytes(), amount), core.TypeByteArray), nil
	}
	return core.NewDish(rotBytes(in.Bytes(), amount, rotrByte), core.TypeByteArray), nil
}
