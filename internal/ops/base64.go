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

// fromBase64 decodes a base64 string using the given alphabet, ported from
// CyberChef's fromBase64. When removeNonAlph is set, characters outside the
// alphabet are stripped first. Without strict mode the decode is lenient:
// invalid or partial input yields the bytes it can and never errors (the
// negative index of a missing character propagates into an out-of-range byte
// that is simply dropped). Strict mode rejects 4n+1 lengths, misplaced padding,
// and non-alphabet characters, matching CyberChef.
func fromBase64(data, alph string, removeNonAlph, strict bool) ([]byte, error) {
	alphabet := []rune(expandAlphRange(alph))
	if len(alphabet) != 64 && len(alphabet) != 65 {
		return nil, fmt.Errorf("Base64 alphabet must be 64 characters, or 65 with padding; got %d", len(alphabet))
	}
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
		if v, ok := idx[r[i]]; ok {
			return v
		}
		return -1
	}

	if strict {
		if len(r)%4 == 1 {
			return nil, fmt.Errorf("invalid Base64 input length (%d): cannot be 4n+1, even without padding characters", len(r))
		}
		if padIndex >= 0 {
			pad := alphabet[padIndex]
			if p := runeIndex(r, pad); p >= 0 {
				if p < len(r)-2 || r[len(r)-1] != pad {
					return nil, fmt.Errorf("Base64 padding character (%c) not used in the correct place", pad)
				}
				if len(r)%4 != 0 {
					return nil, fmt.Errorf("Base64 not padded to a multiple of 4")
				}
			}
		}
	}

	var out []byte
	for i := 0; i < len(r); i += 4 {
		e1, e2, e3, e4 := val(i), val(i+1), val(i+2), val(i+3)
		if strict && (e1 < 0 || e2 < 0 || e3 < 0 || e4 < 0) {
			return nil, fmt.Errorf("Base64 input contains non-alphabet character(s)")
		}
		chr1 := (e1 << 2) | (e2 >> 4)
		chr2 := ((e2 & 15) << 4) | (e3 >> 2)
		chr3 := ((e3 & 3) << 6) | e4
		if chr1 >= 0 && chr1 < 256 {
			out = append(out, byte(chr1)) // #nosec G115 -- range-checked to [0,256)
		}
		if chr2 >= 0 && chr2 < 256 && e3 != padIndex {
			out = append(out, byte(chr2)) // #nosec G115 -- range-checked to [0,256)
		}
		if chr3 >= 0 && chr3 < 256 && e4 != padIndex {
			out = append(out, byte(chr3)) // #nosec G115 -- range-checked to [0,256)
		}
	}
	return out, nil
}

// runeIndex returns the index of the first occurrence of c in r, or -1.
func runeIndex(r []rune, c rune) int {
	for i, x := range r {
		if x == c {
			return i
		}
	}
	return -1
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
	strict := args[2].(bool)
	out, err := fromBase64(in.String(), alph, removeNonAlph, strict)
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeByteArray), nil
}
