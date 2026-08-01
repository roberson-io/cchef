package ops

import "github.com/roberson-io/cchef/core"

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

// Run reverses the input. Ported from CyberChef Reverse.mjs.
func (Reverse) Run(in *core.Dish, args []any) (*core.Dish, error) {
	data := in.Bytes()
	switch args[0].(string) {
	case "Line":
		return core.NewDish(reverseLines(data), core.TypeByteArray), nil
	case "Character":
		r := []rune(string(data))
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
	for i := len(lines) - 1; i >= 0; i-- {
		out = append(out, lines[i]...)
		out = append(out, '\n')
	}
	if len(out) > len(data) {
		out = out[:len(data)]
	}
	return out
}
