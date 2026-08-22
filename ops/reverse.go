package ops

import (
	"slices"
	"unicode/utf8"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(Reverse{})
}

// Reverse reverses the input by byte, character (UTF-8 rune), or line.
type Reverse struct{}

// Meta returns the operation metadata.
func (Reverse) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Reverse",
		Module:      "Default",
		Description: "Reverses the input by byte, character (UTF-8 rune), or line.",
		InputType:   core.TypeByteArray,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions. Default scope is Character.
func (Reverse) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "By", Type: core.ArgOption, Value: []string{"Byte", "Character", "Line"}, DefaultIndex: 1},
	}
}

// Run reverses the input.
func (Reverse) Run(in *core.Dish, args []any) (*core.Dish, error) {
	data := in.Bytes()
	switch args[0].(string) {
	case "Line":
		return core.NewDish(reverseLines(data), core.TypeByteArray), nil
	case "Character":
		r := bytesToRunesLatin1(data)
		for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
			r[i], r[j] = r[j], r[i]
		}
		return core.NewDish([]byte(string(r)), core.TypeByteArray), nil
	default: // Byte
		out := make([]byte, len(data))
		for i, b := range data {
			out[len(data)-1-i] = b
		}
		return core.NewDish(out, core.TypeByteArray), nil
	}
}

// bytesToRunesLatin1 decodes bytes to runes the way CyberChef's byteArrayToUtf8
// does: a valid UTF-8 sequence becomes its rune, and any byte that is not part
// of one is kept as its Latin-1 code point rather than being replaced with the
// U+FFFD replacement character. This makes reversal of non-UTF-8 (e.g. binary)
// input lossless.
func bytesToRunesLatin1(data []byte) []rune {
	runes := make([]rune, 0, len(data))
	for i := 0; i < len(data); {
		r, size := utf8.DecodeRune(data[i:])
		if r == utf8.RuneError && size == 1 {
			r = rune(data[i]) // invalid byte -> its Latin-1 code point
		}
		runes = append(runes, r)
		i += size
	}
	return runes
}

// reverseLines reverses the order of LF-separated lines, preserving the
// original total length (matching CyberChef's slice behaviour).
func reverseLines(data []byte) []byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			lines = append(lines, data[start:i])
			start = i + 1
		}
	}
	lines = append(lines, data[start:])

	var out []byte
	for _, line := range slices.Backward(lines) {
		out = append(out, line...)
		out = append(out, '\n')
	}
	if len(out) > len(data) {
		out = out[:len(data)]
	}
	return out
}
