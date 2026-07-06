package ops

import (
	"fmt"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

// stdBase64Alphabet is the RFC 4648 alphabet (the CyberChef default).
const stdBase64Alphabet = "A-Za-z0-9+/="

func init() {
	core.Register(ToBase64{})
	core.Register(FromBase64{})
}

// expandAlphRange expands an alphabet specification such as "A-Za-z0-9+/=" into
// its full character sequence. A backslash escapes a literal dash ("\\-").
// Ported from CyberChef's Utils.expandAlphRange.
func expandAlphRange(alph string) string {
	r := []rune(alph)
	var out strings.Builder
	for i := 0; i < len(r); i++ {
		switch {
		case i < len(r)-2 && r[i+1] == '-' && r[i] != '\\':
			for c := r[i]; c <= r[i+2]; c++ {
				out.WriteRune(c)
			}
			i += 2
		case i < len(r)-1 && r[i] == '\\' && r[i+1] == '-':
			out.WriteRune('-')
			i++
		default:
			out.WriteRune(r[i])
		}
	}
	return out.String()
}

// toBase64 encodes data using the given (possibly custom) alphabet. A 65th
// character is used as padding; a 64-character alphabet produces no padding.
func toBase64(data []byte, alph string) string {
	if len(data) == 0 {
		return ""
	}
	alphabet := []rune(expandAlphRange(alph))
	pad := ""
	if len(alphabet) == 65 {
		pad = string(alphabet[64])
	}

	var out strings.Builder
	for i := 0; i < len(data); i += 3 {
		c0 := int(data[i])
		c1, c2 := -1, -1
		if i+1 < len(data) {
			c1 = int(data[i+1])
		}
		if i+2 < len(data) {
			c2 = int(data[i+2])
		}

		e0 := c0 >> 2
		e1 := (c0 & 3) << 4
		e2, e3 := 64, 64
		if c1 >= 0 {
			e1 |= c1 >> 4
			e2 = (c1 & 15) << 2
			if c2 >= 0 {
				e2 |= c2 >> 6
				e3 = c2 & 63
			}
		}

		out.WriteRune(alphabet[e0])
		out.WriteRune(alphabet[e1])
		if e2 == 64 {
			out.WriteString(pad)
		} else {
			out.WriteRune(alphabet[e2])
		}
		if e3 == 64 {
			out.WriteString(pad)
		} else {
			out.WriteRune(alphabet[e3])
		}
	}
	return out.String()
}

// fromBase64 decodes a base64 string using the given alphabet. When
// removeNonAlph is set, characters outside the alphabet are stripped first.
func fromBase64(data, alph string, removeNonAlph bool) ([]byte, error) {
	if data == "" {
		return nil, nil
	}
	alphabet := []rune(expandAlphRange(alph))
	idx := make(map[rune]int, len(alphabet))
	for i, c := range alphabet {
		idx[c] = i
	}
	padIndex := -1
	if len(alphabet) == 65 {
		padIndex = 64
	}

	r := []rune(data)
	if removeNonAlph {
		filtered := r[:0]
		for _, c := range r {
			if _, ok := idx[c]; ok {
				filtered = append(filtered, c)
			}
		}
		r = filtered
	}

	val := func(i int) int {
		if i >= len(r) {
			return -1
		}
		v, ok := idx[r[i]]
		if !ok {
			return -1
		}
		return v
	}

	var out []byte
	for i := 0; i < len(r); i += 4 {
		e0, e1, e2, e3 := val(i), val(i+1), val(i+2), val(i+3)
		if e0 == -1 || e1 == -1 {
			return nil, fmt.Errorf("invalid base64 input near position %d", i)
		}
		out = append(out, byte((e0<<2)|(e1>>4))) // #nosec G115 -- 6-bit groups recombined into a byte by the Base64 decode
		if e2 != padIndex && e2 != -1 {
			out = append(out, byte((e1<<4)|(e2>>2))) // #nosec G115 -- 6-bit groups recombined into a byte by the Base64 decode
		}
		if e3 != padIndex && e3 != -1 {
			out = append(out, byte((e2<<6)|e3)) // #nosec G115 -- 6-bit groups recombined into a byte by the Base64 decode
		}
	}
	return out, nil
}

// ToBase64 encodes raw data into an ASCII Base64 string.
type ToBase64 struct{}

// Meta returns the operation metadata.
func (ToBase64) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "To Base64",
		Module:      "Default",
		Description: "Encodes raw data into an ASCII Base64 string. e.g. hello becomes aGVsbG8=",
		InfoURL:     "https://wikipedia.org/wiki/Base64",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ToBase64) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Alphabet", Type: core.ArgEditableOption, Value: stdBase64Alphabet},
	}
}

// Run encodes the input.
func (ToBase64) Run(in *core.Dish, args []any) (*core.Dish, error) {
	alph := args[0].(string)
	return core.NewDish([]byte(toBase64(in.Bytes(), alph)), core.TypeString), nil
}

// FromBase64 decodes data from an ASCII Base64 string back into raw bytes.
type FromBase64 struct{}

// Meta returns the operation metadata.
func (FromBase64) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "From Base64",
		Module:      "Default",
		Description: "Decodes data from an ASCII Base64 string back into its raw format. e.g. aGVsbG8= becomes hello",
		InfoURL:     "https://wikipedia.org/wiki/Base64",
		InputType:   core.TypeString,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (FromBase64) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Alphabet", Type: core.ArgEditableOption, Value: stdBase64Alphabet},
		{Name: "Remove non-alphabet chars", Type: core.ArgBoolean, Value: true},
		{Name: "Strict mode", Type: core.ArgBoolean, Value: false},
	}
}

// Run decodes the input.
func (FromBase64) Run(in *core.Dish, args []any) (*core.Dish, error) {
	alph := args[0].(string)
	removeNonAlph := args[1].(bool)
	out, err := fromBase64(in.String(), alph, removeNonAlph)
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeByteArray), nil
}
