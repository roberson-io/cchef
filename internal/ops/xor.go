package ops

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"unicode"

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
		// CyberChef's fromDecimal is called with delim "Auto", which charRep maps
		// to undefined; `str.split(undefined)` yields the whole string as one token
		// and parseInt reads only the leading integer. So a decimal key is a single
		// byte (e.g. "82 226" -> [82]).
		//
		// A key with no leading integer is [NaN] in CyberChef, whose per-operation
		// behaviour can't be reproduced by any single byte (ADD/SUB zero the input
		// while OR leaves it unchanged). We treat it as identity ([0]); the only
		// divergence is ADD/SUB with a fully non-numeric decimal key.
		if v, ok := leadingInt(str); ok {
			return []byte{byte(v)}, nil
		}
		return []byte{0}, nil
	case "binary":
		return fromBinaryKey(str), nil
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

// leadingInt parses a leading base-10 integer the way JavaScript's parseInt
// does: skip leading whitespace, take an optional sign and the following digits,
// and stop at the first non-digit. Returns false if there is no integer (NaN).
func leadingInt(s string) (int, bool) {
	s = strings.TrimLeft(s, " \t\n\r\f\v")
	i := 0
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		i++
	}
	start := i
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == start {
		return 0, false
	}
	v, err := strconv.Atoi(s[:i])
	if err != nil {
		return 0, false
	}
	return v, true
}

// fromBinaryKey decodes a binary key the way CyberChef's fromBinary does when
// called via convertToByteArray: strip all whitespace (the "Space" delimiter is
// the regex /\s+/g) and read fixed 8-bit chunks, so "0110100001101001" becomes
// two bytes. A trailing partial chunk is parsed as-is.
func fromBinaryKey(str string) []byte {
	var sb strings.Builder
	for _, r := range str {
		if !unicode.IsSpace(r) {
			sb.WriteRune(r)
		}
	}
	s := sb.String()
	out := make([]byte, 0, len(s)/8+1)
	for i := 0; i < len(s); i += 8 {
		v, _ := strconv.ParseUint(s[i:min(i+8, len(s))], 2, 32)
		out = append(out, byte(v))
	}
	return out
}

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

// Run applies the XOR. Ported from CyberChef XOR.mjs / lib bitOp.
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
