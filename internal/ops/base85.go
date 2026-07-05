package ops

import (
	"fmt"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

// Base85 named alphabet specifications (CyberChef ALPHABET_OPTIONS).
var base85Options = []struct{ name, value string }{
	{"Standard", "!-u"},
	{"Z85 (ZeroMQ)", `0-9a-zA-Z.\-:+=^!/*?&<>()[]{}@%$#`},
	{"IPv6", "0-9A-Za-z!#$%&()*+\\-;<=>?@^_`{|}~"},
}

const base85Standard = "!-u"

func init() {
	core.Register(ToBase85{})
	core.Register(FromBase85{})
}

// alphabetName returns the name of a known Base85 alphabet, or "" if custom.
func base85AlphabetName(expanded string) string {
	for _, o := range base85Options {
		if expandAlphRange(o.value) == expanded {
			return o.name
		}
	}
	return ""
}

// expandB85Alphabet expands and validates an 85-character alphabet.
func expandB85Alphabet(spec string) ([]rune, error) {
	alphabet := []rune(expandAlphRange(spec))
	seen := make(map[rune]bool, len(alphabet))
	for _, c := range alphabet {
		seen[c] = true
	}
	if len(alphabet) != 85 || len(seen) != 85 {
		return nil, fmt.Errorf("alphabet must be of length 85")
	}
	return alphabet, nil
}

// ToBase85 encodes raw bytes as a Base85 (Ascii85) string.
type ToBase85 struct{}

// Meta returns the operation metadata.
func (ToBase85) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "To Base85",
		Module:      "Default",
		Description: "Base85 (Ascii85) encodes arbitrary byte data using 85 printable ASCII characters, more efficient than Base64.",
		InfoURL:     "https://wikipedia.org/wiki/Ascii85",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ToBase85) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Alphabet", Type: core.ArgEditableOption, Value: base85Standard},
		{Name: "Include delimiter", Type: core.ArgBoolean, Value: false},
	}
}

// Run encodes the input. Ported from CyberChef ToBase85.mjs.
func (ToBase85) Run(in *core.Dish, args []any) (*core.Dish, error) {
	alphabet, err := expandB85Alphabet(args[0].(string))
	if err != nil {
		return nil, err
	}
	includeDelim := args[1].(bool)
	encoding := base85AlphabetName(string(alphabet))
	input := in.Bytes()
	if len(input) == 0 {
		return core.NewDish(nil, core.TypeString), nil
	}

	var sb strings.Builder
	for i := 0; i < len(input); i += 4 {
		n := min(len(input)-i, 4)
		var block uint32
		block |= uint32(input[i]) << 24
		if i+1 < len(input) {
			block |= uint32(input[i+1]) << 16
		}
		if i+2 < len(input) {
			block |= uint32(input[i+2]) << 8
		}
		if i+3 < len(input) {
			block |= uint32(input[i+3])
		}

		if encoding != "Standard" || block > 0 {
			digits := make([]int, 5)
			b := block
			for j := 4; j >= 0; j-- {
				digits[j] = int(b % 85)
				b /= 85
			}
			// For a partial final block, keep n+1 digits.
			if n < 4 {
				digits = digits[:n+1]
			}
			for _, d := range digits {
				sb.WriteRune(alphabet[d])
			}
		} else {
			sb.WriteByte('z')
		}
	}

	out := sb.String()
	if includeDelim {
		out = "<~" + out + "~>"
	}
	return core.NewDish([]byte(out), core.TypeString), nil
}

// FromBase85 decodes a Base85 (Ascii85) string back into raw bytes.
type FromBase85 struct{}

// Meta returns the operation metadata.
func (FromBase85) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "From Base85",
		Module:      "Default",
		Description: "Decodes a Base85 (Ascii85) string back into its raw byte value.",
		InfoURL:     "https://wikipedia.org/wiki/Ascii85",
		InputType:   core.TypeString,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (FromBase85) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Alphabet", Type: core.ArgEditableOption, Value: base85Standard},
		{Name: "Remove non-alphabet chars", Type: core.ArgBoolean, Value: true},
		{Name: "All-zero group char", Type: core.ArgString, Value: "z"},
	}
}

// Run decodes the input. Ported from CyberChef FromBase85.mjs.
func (FromBase85) Run(in *core.Dish, args []any) (*core.Dish, error) {
	alphabet, err := expandB85Alphabet(args[0].(string))
	if err != nil {
		return nil, err
	}
	removeNonAlph := args[1].(bool)
	var allZero rune = -1
	if s := args[2].(string); s != "" {
		allZero = []rune(s)[0]
	}
	idx := make(map[rune]int, len(alphabet))
	for i, c := range alphabet {
		idx[c] = i
	}
	if allZero >= 0 {
		if _, ok := idx[allZero]; ok {
			return nil, fmt.Errorf("the all-zero group char cannot appear in the alphabet")
		}
	}

	input := []rune(in.String())
	input = stripBase85Delim(input)
	if removeNonAlph {
		keep := func(c rune) bool {
			if c == '~' || c == allZero {
				return true
			}
			_, ok := idx[c]
			return ok
		}
		filtered := input[:0]
		for _, c := range input {
			if keep(c) {
				filtered = append(filtered, c)
			}
		}
		input = stripBase85Delim(filtered)
	}
	if len(input) == 0 {
		return core.NewDish(nil, core.TypeByteArray), nil
	}

	var res []byte
	for i := 0; i < len(input); {
		if input[i] == allZero {
			res = append(res, 0, 0, 0, 0)
			i++
			continue
		}
		// Up to 5 digits in this group.
		m := min(len(input)-i, 5)
		digit := func(off, fallback int) int {
			if i+off < len(input) {
				c := input[i+off]
				k, ok := idx[c]
				if !ok || k < 0 || k > 84 {
					return -1
				}
				return k
			}
			return fallback
		}
		d0, d1 := digit(0, 84), digit(1, 84)
		d2, d3, d4 := digit(2, 84), digit(3, 84), digit(4, 84)
		if d0 < 0 || d1 < 0 || (i+2 < len(input) && d2 < 0) ||
			(i+3 < len(input) && d3 < 0) || (i+4 < len(input) && d4 < 0) {
			return nil, fmt.Errorf("invalid base85 character in group at index %d", i)
		}

		block := uint32(d0)*52200625 + uint32(d1)*614125 + uint32(d2)*7225 + uint32(d3)*85 + uint32(d4)
		blockBytes := []byte{
			byte(block >> 24), byte(block >> 16), byte(block >> 8), byte(block),
		}
		// Partial final group of m chars yields m-1 bytes.
		if m < 5 {
			blockBytes = blockBytes[:m-1]
		}
		res = append(res, blockBytes...)
		i += 5
	}
	return core.NewDish(res, core.TypeByteArray), nil
}

// stripBase85Delim removes surrounding <~ ~> delimiters if present.
func stripBase85Delim(r []rune) []rune {
	if len(r) >= 4 && r[0] == '<' && r[1] == '~' && r[len(r)-2] == '~' && r[len(r)-1] == '>' {
		return r[2 : len(r)-2]
	}
	return r
}
