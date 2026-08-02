package ops

import (
	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(XOR{})
}

// xorDelims are the key encoding modes for the toggleString key argument.
var xorDelims = []string{"Hex", "Decimal", "Binary", "Base64", "UTF8", "Latin1"}

// bitOp applies f across the input against a repeating key, mirroring
// lib/BitwiseOp.mjs bitOp: the Cascade scheme keys each byte off its successor,
// and the differential schemes rewrite the key in place from the input/output
// (so callers must pass a fresh key per invocation). Null preserving leaves a
// byte untouched when it is 0 or equal to the key byte.
func bitOp(input, key []byte, f func(o, k byte) byte, nullPreserving bool, scheme string) []byte {
	if len(key) == 0 {
		key = []byte{0}
	}
	out := make([]byte, 0, len(input))
	for i := range input {
		k := key[i%len(key)]
		if scheme == "Cascade" {
			if i+1 < len(input) {
				k = input[i+1]
			} else {
				k = 0
			}
		}
		o := input[i]

		var x byte
		skip := nullPreserving && (o == 0 || o == k)
		if skip {
			x = o
		} else {
			x = f(o, k)
		}
		out = append(out, x)

		if scheme != "Standard" && scheme != "Cascade" && !skip {
			switch scheme {
			case "Input differential":
				key[i%len(key)] = o
			case "Output differential":
				key[i%len(key)] = x
			}
		}
	}
	return out
}

// xorByte is the XOR bitwise calculation (lib/BitwiseOp.mjs xor).
func xorByte(o, k byte) byte { return o ^ k }

// XOR XORs the input with a repeating key, supporting CyberChef's key schemes.
type XOR struct{}

// Meta returns the operation metadata.
func (XOR) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "XOR",
		Module:      "Default",
		Description: "XOR the input with the given key. Supports Standard, Input differential, Output differential and Cascade schemes.",
		InfoURL:     "https://wikipedia.org/wiki/XOR",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (XOR) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Key", Type: core.ArgToggleString, Value: "", ToggleValues: xorDelims},
		{Name: "Scheme", Type: core.ArgOption, Value: []string{"Standard", "Input differential", "Output differential", "Cascade"}},
		{Name: "Null preserving", Type: core.ArgBoolean, Value: false},
	}
}

// Run applies the XOR.
func (XOR) Run(in *core.Dish, args []any) (*core.Dish, error) {
	keyArg := args[0].(core.ToggleString)
	scheme := args[1].(string)
	nullPreserving := args[2].(bool)

	key, err := convertToByteArray(keyArg.Value, keyArg.Option)
	if err != nil {
		return nil, err
	}
	return core.NewDish(bitOp(in.Bytes(), key, xorByte, nullPreserving, scheme), core.TypeByteArray), nil
}
