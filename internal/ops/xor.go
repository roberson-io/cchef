package ops

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(XOR{})
}

// xorDelims are the key encoding modes for the toggleString key argument.
var xorDelims = []string{"Hex", "Decimal", "Binary", "Base64", "UTF8", "Latin1"}

// convertToByteArray decodes a key string according to its encoding mode.
// Ported from CyberChef Utils.convertToByteArray.
func convertToByteArray(str, mode string) ([]byte, error) {
	switch strings.ToLower(mode) {
	case "hex":
		var out []byte
		for _, p := range nonHex.Split(str, -1) {
			for j := 0; j+2 <= len(p); j += 2 {
				v, err := strconv.ParseUint(p[j:j+2], 16, 8)
				if err != nil {
					return nil, fmt.Errorf("invalid hex key byte %q: %w", p[j:j+2], err)
				}
				out = append(out, byte(v))
			}
		}
		return out, nil
	case "decimal":
		return parseNumberList(str, 10)
	case "binary":
		return parseNumberList(str, 2)
	case "base64":
		return base64.StdEncoding.DecodeString(strings.TrimSpace(str))
	case "utf8":
		return []byte(str), nil
	default: // latin1
		out := make([]byte, 0, len(str))
		for _, r := range str {
			out = append(out, byte(r))
		}
		return out, nil
	}
}

// parseNumberList parses whitespace/comma-separated numbers in the given base.
func parseNumberList(str string, base int) ([]byte, error) {
	fields := strings.FieldsFunc(str, func(r rune) bool {
		return r == ' ' || r == ',' || r == '\t' || r == '\n'
	})
	out := make([]byte, 0, len(fields))
	for _, f := range fields {
		v, err := strconv.ParseUint(f, base, 16)
		if err != nil {
			return nil, fmt.Errorf("invalid key value %q: %w", f, err)
		}
		out = append(out, byte(v))
	}
	return out, nil
}

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

// Run applies the XOR. Ported from CyberChef XOR.mjs / lib bitOp.
func (XOR) Run(in *core.Dish, args []any) (*core.Dish, error) {
	keyArg := args[0].(core.ToggleString)
	scheme := args[1].(string)
	nullPreserving := args[2].(bool)

	key, err := convertToByteArray(keyArg.Value, keyArg.Option)
	if err != nil {
		return nil, err
	}
	if len(key) == 0 {
		key = []byte{0}
	}

	input := in.Bytes()
	result := make([]byte, 0, len(input))
	for i := 0; i < len(input); i++ {
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
			x = o ^ k
		}
		result = append(result, x)

		if scheme != "Standard" && scheme != "Cascade" && !skip {
			switch scheme {
			case "Input differential":
				key[i%len(key)] = o
			case "Output differential":
				key[i%len(key)] = x
			}
		}
	}
	return core.NewDish(result, core.TypeByteArray), nil
}
