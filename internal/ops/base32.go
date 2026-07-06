package ops

import (
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

const stdBase32Alphabet = "A-Z2-7="

func init() {
	core.Register(ToBase32{})
	core.Register(FromBase32{})
}

// ToBase32 encodes raw data as a Base32 string.
type ToBase32 struct{}

// Meta returns the operation metadata.
func (ToBase32) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "To Base32",
		Module:      "Default",
		Description: "Base32 encodes arbitrary byte data using a restricted set of symbols (usually A-Z and 2-7).",
		InfoURL:     "https://wikipedia.org/wiki/Base32",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ToBase32) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Alphabet", Type: core.ArgEditableOption, Value: stdBase32Alphabet},
	}
}

// Run encodes the input. Ported from CyberChef ToBase32.mjs.
func (ToBase32) Run(in *core.Dish, args []any) (*core.Dish, error) {
	data := in.Bytes()
	if len(data) == 0 {
		return core.NewDish(nil, core.TypeString), nil
	}
	alphabet := []rune(expandAlphRange(args[0].(string)))

	var sb strings.Builder
	for i := 0; i < len(data); i += 5 {
		// Missing bytes are treated as 0 in the bit math (mirroring JS, where
		// NaN coerces to 0 under bitwise operators); n tracks how many are real.
		var c [5]int
		n := 0
		for j := 0; j < 5 && i+j < len(data); j++ {
			c[j] = int(data[i+j])
			n++
		}

		enc := [8]int{
			c[0] >> 3,
			((c[0] & 7) << 2) | (c[1] >> 6),
			(c[1] >> 1) & 31,
			((c[1] & 1) << 4) | (c[2] >> 4),
			((c[2] & 15) << 1) | (c[3] >> 7),
			(c[3] >> 2) & 31,
			((c[3] & 3) << 3) | (c[4] >> 5),
			c[4] & 31,
		}

		// Pad the encodings that correspond to absent input bytes.
		switch n {
		case 1:
			enc[2], enc[3], enc[4], enc[5], enc[6], enc[7] = 32, 32, 32, 32, 32, 32
		case 2:
			enc[4], enc[5], enc[6], enc[7] = 32, 32, 32, 32
		case 3:
			enc[5], enc[6], enc[7] = 32, 32, 32
		case 4:
			enc[7] = 32
		}

		for _, e := range enc {
			// An out-of-range index (e.g. the padding slot 32 when the alphabet
			// has no padding character) contributes nothing, mirroring JS's
			// (alphabetChars[e] || "") in gchq/CyberChef#2380.
			if e < len(alphabet) {
				sb.WriteRune(alphabet[e])
			}
		}
	}
	return core.NewDish([]byte(sb.String()), core.TypeString), nil
}

// FromBase32 decodes a Base32 string back into raw bytes.
type FromBase32 struct{}

// Meta returns the operation metadata.
func (FromBase32) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "From Base32",
		Module:      "Default",
		Description: "Decodes a Base32 string back into its raw byte value.",
		InfoURL:     "https://wikipedia.org/wiki/Base32",
		InputType:   core.TypeString,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (FromBase32) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Alphabet", Type: core.ArgEditableOption, Value: stdBase32Alphabet},
		{Name: "Remove non-alphabet chars", Type: core.ArgBoolean, Value: true},
	}
}

// Run decodes the input. Ported from CyberChef FromBase32.mjs.
func (FromBase32) Run(in *core.Dish, args []any) (*core.Dish, error) {
	data := []rune(in.String())
	if len(data) == 0 {
		return core.NewDish(nil, core.TypeByteArray), nil
	}
	alphabet := []rune(expandAlphRange(args[0].(string)))
	removeNonAlph := args[1].(bool)

	idx := make(map[rune]int, len(alphabet))
	for i, c := range alphabet {
		idx[c] = i
	}
	if removeNonAlph {
		filtered := data[:0]
		for _, c := range data {
			if _, ok := idx[c]; ok {
				filtered = append(filtered, c)
			}
		}
		data = filtered
	}

	// indexOf with "=" fallback for missing chars, matching CyberChef.
	at := func(i int) int {
		c := '='
		if i < len(data) {
			c = data[i]
		}
		if v, ok := idx[c]; ok {
			return v
		}
		return -1
	}

	var out []byte
	for i := 0; i < len(data); i += 8 {
		e := [8]int{}
		for j := range e {
			e[j] = at(i + j)
		}

		chr1 := (e[0] << 3) | (e[1] >> 2)
		chr2 := ((e[1] & 3) << 6) | (e[2] << 1) | (e[3] >> 4)
		chr3 := ((e[3] & 15) << 4) | (e[4] >> 1)
		chr4 := ((e[4] & 1) << 7) | (e[5] << 2) | (e[6] >> 3)
		chr5 := ((e[6] & 7) << 5) | e[7]

		out = append(out, byte(chr1))
		if (e[1]&3) != 0 || e[2] != 32 {
			out = append(out, byte(chr2))
		}
		if (e[3]&15) != 0 || e[4] != 32 {
			out = append(out, byte(chr3))
		}
		if (e[4]&1) != 0 || e[5] != 32 {
			out = append(out, byte(chr4))
		}
		if (e[6]&7) != 0 || e[7] != 32 {
			out = append(out, byte(chr5))
		}
	}
	return core.NewDish(out, core.TypeByteArray), nil
}
