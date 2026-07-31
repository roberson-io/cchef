package ops

import (
	"strings"
	"unicode"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(SwapCase{})
	core.Register(RemoveWhitespace{})
	core.Register(RemoveNullBytes{})
	core.Register(PadLines{})
}

// SwapCase swaps the case of each character.
type SwapCase struct{}

// Meta returns the operation metadata.
func (SwapCase) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Swap case",
		Module:      "Default",
		Description: "Converts uppercase characters to lowercase and vice versa.",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (SwapCase) Args() []core.ArgDef { return nil }

// Run swaps case. Ported from CyberChef SwapCase.mjs.
func (SwapCase) Run(in *core.Dish, args []any) (*core.Dish, error) {
	var sb strings.Builder
	for _, r := range in.String() {
		if r == unicode.ToUpper(r) {
			sb.WriteRune(unicode.ToLower(r))
		} else {
			sb.WriteRune(unicode.ToUpper(r))
		}
	}
	return core.NewDish([]byte(sb.String()), core.TypeString), nil
}

// RemoveWhitespace removes selected whitespace (and optionally full stops).
type RemoveWhitespace struct{}

// Meta returns the operation metadata.
func (RemoveWhitespace) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Remove whitespace",
		Module:      "Default",
		Description: "Optionally removes spaces, carriage returns, line feeds, tabs, form feeds and full stops from the input.",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (RemoveWhitespace) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Spaces", Type: core.ArgBoolean, Value: true},
		{Name: "Carriage returns (\\r)", Type: core.ArgBoolean, Value: true},
		{Name: "Line feeds (\\n)", Type: core.ArgBoolean, Value: true},
		{Name: "Tabs", Type: core.ArgBoolean, Value: true},
		{Name: "Form feeds (\\f)", Type: core.ArgBoolean, Value: true},
		{Name: "Full stops", Type: core.ArgBoolean, Value: false},
	}
}

// Run removes the selected characters. Ported from CyberChef RemoveWhitespace.mjs.
func (RemoveWhitespace) Run(in *core.Dish, args []any) (*core.Dish, error) {
	data := in.String()
	repl := func(on bool, old string) {
		if on {
			data = strings.ReplaceAll(data, old, "")
		}
	}
	repl(args[0].(bool), " ")
	repl(args[1].(bool), "\r")
	repl(args[2].(bool), "\n")
	repl(args[3].(bool), "\t")
	repl(args[4].(bool), "\f")
	repl(args[5].(bool), ".")
	return core.NewDish([]byte(data), core.TypeString), nil
}

// RemoveNullBytes strips all 0x00 bytes from the input.
type RemoveNullBytes struct{}

// Meta returns the operation metadata.
func (RemoveNullBytes) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Remove null bytes",
		Module:      "Default",
		Description: "Removes all null bytes (0x00) from the input.",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (RemoveNullBytes) Args() []core.ArgDef { return nil }

// Run removes null bytes.
func (RemoveNullBytes) Run(in *core.Dish, args []any) (*core.Dish, error) {
	data := in.Bytes()
	out := make([]byte, 0, len(data))
	for _, b := range data {
		if b != 0 {
			out = append(out, b)
		}
	}
	return core.NewDish(out, core.TypeByteArray), nil
}

// PadLines pads each line at the start or end by a number of characters.
type PadLines struct{}

// Meta returns the operation metadata.
func (PadLines) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Pad lines",
		Module:      "Default",
		Description: "Adds the specified number of padding characters to the start or end of each line.",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (PadLines) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Position", Type: core.ArgOption, Value: []string{"Start", "End"}},
		{Name: "Length", Type: core.ArgNumber, Integer: true, Value: 5},
		{Name: "Character", Type: core.ArgString, Value: " "},
	}
}

// Run pads each line. Ported from CyberChef PadLines.mjs.
func (PadLines) Run(in *core.Dish, args []any) (*core.Dish, error) {
	position := args[0].(string)
	n := int(args[1].(float64))
	chr := args[2].(string)
	pad := padString(chr, n)

	lines := strings.Split(in.String(), "\n")
	for i, line := range lines {
		if position == "Start" {
			lines[i] = pad + line
		} else {
			lines[i] = line + pad
		}
	}
	return core.NewDish([]byte(strings.Join(lines, "\n")), core.TypeString), nil
}

// padString builds a pad of n characters from the (possibly multi-char) pattern,
// matching JS padStart/padEnd which repeat then truncate the pattern.
func padString(pattern string, n int) string {
	if n <= 0 || pattern == "" {
		return ""
	}
	r := []rune(pattern)
	out := make([]rune, n)
	for i := range n {
		out[i] = r[i%len(r)]
	}
	return string(out)
}
