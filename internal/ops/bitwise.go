package ops

import (
	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(ADD{})
	core.Register(SUB{})
	core.Register(AND{})
	core.Register(OR{})
	core.Register(NOT{})
	core.Register(BitShiftLeft{})
	core.Register(BitShiftRight{})
}

// bitwise applies a per-byte function against a repeating key. An empty key is
// treated as a single zero byte. Ported from lib/BitwiseOp.mjs bitOp (the plain
// Standard, non-null-preserving form used by ADD/SUB/AND/OR/NOT).
func bitwise(input, key []byte, f func(o, k byte) byte) []byte {
	if len(key) == 0 {
		key = []byte{0}
	}
	out := make([]byte, len(input))
	for i, o := range input {
		out[i] = f(o, key[i%len(key)])
	}
	return out
}

// runBitwise decodes the toggleString key and applies f across the input.
func runBitwise(in *core.Dish, keyArg core.ToggleString, f func(o, k byte) byte) (*core.Dish, error) {
	key, err := convertToByteArray(keyArg.Value, keyArg.Option)
	if err != nil {
		return nil, err
	}
	return core.NewDish(bitwise(in.Bytes(), key, f), core.TypeByteArray), nil
}

// ADD adds a repeating key to each byte, mod 256.
type ADD struct{}

// Meta returns the operation metadata.
func (ADD) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "ADD",
		Module:      "Default",
		Description: "ADD the input with the given key (e.g. fe023da5), MOD 255.",
		InfoURL:     "https://wikipedia.org/wiki/Bitwise_operation#Bitwise_operators",
		InputType:   core.TypeByteArray,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (ADD) Args() []core.ArgDef {
	return []core.ArgDef{{Name: "Key", Type: core.ArgToggleString, Value: "", ToggleValues: xorDelims}}
}

// Run applies the ADD. Ported from CyberChef ADD.mjs / lib bitOp.
func (ADD) Run(in *core.Dish, args []any) (*core.Dish, error) {
	return runBitwise(in, args[0].(core.ToggleString), func(o, k byte) byte { return o + k })
}

// SUB subtracts a repeating key from each byte, mod 256.
type SUB struct{}

// Meta returns the operation metadata.
func (SUB) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "SUB",
		Module:      "Default",
		Description: "SUB the input with the given key (e.g. fe023da5), MOD 255.",
		InfoURL:     "https://wikipedia.org/wiki/Bitwise_operation#Bitwise_operators",
		InputType:   core.TypeByteArray,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (SUB) Args() []core.ArgDef {
	return []core.ArgDef{{Name: "Key", Type: core.ArgToggleString, Value: "", ToggleValues: xorDelims}}
}

// Run applies the SUB. Ported from CyberChef SUB.mjs / lib bitOp.
func (SUB) Run(in *core.Dish, args []any) (*core.Dish, error) {
	return runBitwise(in, args[0].(core.ToggleString), func(o, k byte) byte { return o - k })
}

// AND ANDs the input with a repeating key.
type AND struct{}

// Meta returns the operation metadata.
func (AND) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "AND",
		Module:      "Default",
		Description: "AND the input with the given key (e.g. fe023da5).",
		InfoURL:     "https://wikipedia.org/wiki/Bitwise_operation#AND",
		InputType:   core.TypeByteArray,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (AND) Args() []core.ArgDef {
	return []core.ArgDef{{Name: "Key", Type: core.ArgToggleString, Value: "", ToggleValues: xorDelims}}
}

// Run applies the AND. Ported from CyberChef AND.mjs / lib bitOp.
func (AND) Run(in *core.Dish, args []any) (*core.Dish, error) {
	return runBitwise(in, args[0].(core.ToggleString), func(o, k byte) byte { return o & k })
}

// OR ORs the input with a repeating key.
type OR struct{}

// Meta returns the operation metadata.
func (OR) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "OR",
		Module:      "Default",
		Description: "OR the input with the given key (e.g. fe023da5).",
		InfoURL:     "https://wikipedia.org/wiki/Bitwise_operation#OR",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (OR) Args() []core.ArgDef {
	return []core.ArgDef{{Name: "Key", Type: core.ArgToggleString, Value: "", ToggleValues: xorDelims}}
}

// Run applies the OR. Ported from CyberChef OR.mjs / lib bitOp.
func (OR) Run(in *core.Dish, args []any) (*core.Dish, error) {
	return runBitwise(in, args[0].(core.ToggleString), func(o, k byte) byte { return o | k })
}

// NOT inverts each byte.
type NOT struct{}

// Meta returns the operation metadata.
func (NOT) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "NOT",
		Module:      "Default",
		Description: "Returns the inverse of each byte.",
		InfoURL:     "https://wikipedia.org/wiki/Bitwise_operation#NOT",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (NOT) Args() []core.ArgDef { return nil }

// Run inverts each byte. Ported from CyberChef NOT.mjs / lib bitOp.
func (NOT) Run(in *core.Dish, args []any) (*core.Dish, error) {
	return core.NewDish(bitwise(in.Bytes(), nil, func(o, _ byte) byte { return ^o }), core.TypeByteArray), nil
}

// BitShiftLeft shifts the bits in each byte left by a fixed amount.
type BitShiftLeft struct{}

// Meta returns the operation metadata.
func (BitShiftLeft) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Bit shift left",
		Module:      "Default",
		Description: "Shifts the bits in each byte towards the left by the specified amount.",
		InfoURL:     "https://wikipedia.org/wiki/Bitwise_operation#Bit_shifts",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeArrayBuffer,
	}
}

// Args returns the argument definitions.
func (BitShiftLeft) Args() []core.ArgDef {
	minAmt, maxAmt := 0.0, 7.0
	return []core.ArgDef{{Name: "Amount", Type: core.ArgNumber, Value: 1, Min: &minAmt, Max: &maxAmt}}
}

// Run shifts each byte left. Ported from CyberChef BitShiftLeft.mjs.
func (BitShiftLeft) Run(in *core.Dish, args []any) (*core.Dish, error) {
	amount := uint(args[0].(float64))
	data := in.Bytes()
	out := make([]byte, len(data))
	for i, b := range data {
		out[i] = byte(uint(b) << amount)
	}
	return core.NewDish(out, core.TypeArrayBuffer), nil
}

// BitShiftRight shifts the bits in each byte right by a fixed amount.
type BitShiftRight struct{}

// Meta returns the operation metadata.
func (BitShiftRight) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Bit shift right",
		Module:      "Default",
		Description: "Shifts the bits in each byte towards the right by the specified amount. Logical shifts replace the leftmost bits with zeros; arithmetic shifts preserve the most significant bit.",
		InfoURL:     "https://wikipedia.org/wiki/Bitwise_operation#Bit_shifts",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeArrayBuffer,
	}
}

// Args returns the argument definitions.
func (BitShiftRight) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Amount", Type: core.ArgNumber, Value: 1},
		{Name: "Type", Type: core.ArgOption, Value: []string{"Logical shift", "Arithmetic shift"}},
	}
}

// Run shifts each byte right. Ported from CyberChef BitShiftRight.mjs: logical
// shifts fill with zeros, arithmetic shifts XOR the original sign bit back in.
func (BitShiftRight) Run(in *core.Dish, args []any) (*core.Dish, error) {
	amount := uint(args[0].(float64))
	var mask byte
	if args[1].(string) != "Logical shift" {
		mask = 0x80
	}
	data := in.Bytes()
	out := make([]byte, len(data))
	for i, b := range data {
		out[i] = byte(uint(b)>>amount) ^ (b & mask)
	}
	return core.NewDish(out, core.TypeArrayBuffer), nil
}
