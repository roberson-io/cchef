package ops

import (
	"fmt"

	"github.com/roberson-io/cchef/core"
)

// base45Alphabet is the default Base45 alphabet specification (RFC 9285).
const base45Alphabet = `0-9A-Z $%*+\-./:`

func init() {
	core.Register(ToBase45{})
	core.Register(FromBase45{})
}

// ToBase45 encodes raw bytes as a Base45 string.
type ToBase45 struct{}

// Meta returns the operation metadata.
func (ToBase45) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "To Base45",
		Module:      "Default",
		Description: "Base45 encodes arbitrary byte data, used notably in QR codes (RFC 9285).",
		InfoURL:     "https://wikipedia.org/wiki/Base45",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ToBase45) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Alphabet", Type: core.ArgString, Value: base45Alphabet},
	}
}

// Run encodes the input. Ported from CyberChef ToBase45.mjs.
func (ToBase45) Run(in *core.Dish, args []any) (*core.Dish, error) {
	data := in.Bytes()
	if len(data) == 0 {
		return core.NewDish(nil, core.TypeString), nil
	}
	alphabet := []rune(expandAlphRange(args[0].(string)))

	var res []rune
	for i := 0; i < len(data); i += 2 {
		pair := data[i:min(i+2, len(data))]
		b := 0
		for _, e := range pair {
			b = b*256 + int(e)
		}
		chars := 0
		for {
			res = append(res, alphabet[b%45])
			chars++
			b /= 45
			if b == 0 {
				break
			}
		}
		if chars < 2 {
			res = append(res, alphabet[0])
			chars++
		}
		if len(pair) > 1 && chars < 3 {
			res = append(res, alphabet[0])
		}
	}
	return core.NewDish([]byte(string(res)), core.TypeString), nil
}

// FromBase45 decodes a Base45 string back into raw bytes.
type FromBase45 struct{}

// Meta returns the operation metadata.
func (FromBase45) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "From Base45",
		Module:      "Default",
		Description: "Decodes a Base45 string back into its raw byte value (RFC 9285).",
		InfoURL:     "https://wikipedia.org/wiki/Base45",
		InputType:   core.TypeString,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (FromBase45) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Alphabet", Type: core.ArgString, Value: base45Alphabet},
		{Name: "Remove non-alphabet chars", Type: core.ArgBoolean, Value: true},
	}
}

// Run decodes the input. Ported from CyberChef FromBase45.mjs.
func (FromBase45) Run(in *core.Dish, args []any) (*core.Dish, error) {
	if len(in.Bytes()) == 0 {
		return core.NewDish(nil, core.TypeByteArray), nil
	}
	alphabet := expandAlphRange(args[0].(string))
	removeNonAlph := args[1].(bool)

	idx := make(map[rune]int)
	for i, c := range alphabet {
		idx[c] = i
	}

	input := []rune(in.String())
	if removeNonAlph {
		filtered := input[:0]
		for _, c := range input {
			if _, ok := idx[c]; ok {
				filtered = append(filtered, c)
			}
		}
		input = filtered
	}

	var res []byte
	for i := 0; i < len(input); i += 3 {
		triple := input[i:min(i+3, len(input))]
		b := 0
		// Iterate the triple in reverse (matching triple.reverse()).
		for j := len(triple) - 1; j >= 0; j-- {
			c := triple[j]
			k, ok := idx[c]
			if !ok {
				return nil, fmt.Errorf("character not in alphabet: %q", c)
			}
			b = b*45 + k
		}
		if b > 65535 {
			return nil, fmt.Errorf("triplet too large: %q", string(triple))
		}
		if len(triple) > 2 {
			res = append(res, byte(b>>8))
		}
		res = append(res, byte(b&0xff))
	}
	return core.NewDish(res, core.TypeByteArray), nil
}
