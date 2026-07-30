package ops

import (
	"maps"
	"regexp"
	"strconv"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(Register{})
}

// registerRef finds a reference to a register, with any run of backslashes in
// front of it so an escaped reference can be told from a live one.
var registerRef = regexp.MustCompile(`(\\*)\$R(\d{1,2})`)

// Register extracts data from the input into registers that later steps can
// refer to. Ported from CyberChef Register.mjs.
type Register struct{}

// Meta returns the operation metadata.
func (Register) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Register",
		Module:      "Default",
		Description: "Extract data from the input and store it in registers which can then be passed into subsequent operations as arguments. Regular expression capture groups are used to select the data to extract.\n\nTo use registers in arguments, refer to them using the notation $Rn where n is the register number, starting at 0.\n\nFor example:\nInput: Test\nExtractor: (.*)\nArgument: $R0 becomes Test\n\nRegisters can be escaped in arguments using a backslash. e.g. \\$R0 would become $R0 rather than Test.",
		InfoURL:     "https://wikipedia.org/wiki/Regular_expression#Syntax",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the extractor and its flags.
func (Register) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Extractor", Type: core.ArgString, Value: `([\s\S]*)`},
		{Name: "Case insensitive", Type: core.ArgBoolean, Value: true},
		{Name: "Multiline matching", Type: core.ArgBoolean, Value: false},
		{Name: "Dot matches all", Type: core.ArgBoolean, Value: false},
	}
}

// Run passes the data through; outside a recipe there are no later steps to
// pass registers to.
func (Register) Run(in *core.Dish, args []any) (*core.Dish, error) { return in, nil }

// RunFlow extracts the capture groups and writes them into the arguments of
// every later step. The data itself is untouched.
func (Register) RunFlow(state *core.FlowState) error {
	pattern := state.Args[0].(string)
	flags := ""
	if state.Args[1].(bool) {
		flags += "i"
	}
	if state.Args[2].(bool) {
		flags += "m"
	}
	if state.Args[3].(bool) {
		flags += "s"
	}
	if flags != "" {
		pattern = "(?" + flags + ")" + pattern
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	found := re.FindStringSubmatch(state.Dish.String())
	if found == nil {
		return nil
	}
	// found[0] is the whole match; the registers are the capture groups.
	registers := found[1:]

	for i := state.Progress + 1; i < len(state.Steps); i++ {
		if state.Steps[i].Disabled {
			continue
		}
		for j, arg := range state.Steps[i].Args {
			state.Steps[i].Args[j] = fillRegisters(arg, registers, state.NumRegisters)
		}
	}
	state.NumRegisters += len(registers)
	return nil
}

// fillRegisters replaces register references inside one argument. Strings are
// rewritten directly; a toggle string has its value rewritten and its option
// left alone; anything else is returned as it came.
func fillRegisters(arg any, registers []string, offset int) any {
	switch v := arg.(type) {
	case string:
		return replaceRegisters(v, registers, offset)
	case core.ToggleString:
		v.Value = replaceRegisters(v.Value, registers, offset)
		return v
	case map[string]any:
		// A recipe read from JSON carries a toggle string as an object.
		if s, ok := v["string"].(string); ok {
			copied := make(map[string]any, len(v))
			maps.Copy(copied, v)
			copied["string"] = replaceRegisters(s, registers, offset)
			return copied
		}
	}
	return arg
}

// replaceRegisters swaps every live $Rn reference for its register's contents.
// A reference numbered outside this Register's own range is left for another
// Register to fill, and one behind an odd number of backslashes is an escape:
// one backslash is removed and the reference itself is kept.
func replaceRegisters(s string, registers []string, offset int) string {
	return registerRef.ReplaceAllStringFunc(s, func(match string) string {
		parts := registerRef.FindStringSubmatch(match)
		slashes, digits := parts[1], parts[2]
		// The pattern matches one or two digits, so this always parses.
		num, _ := strconv.Atoi(digits)
		if num < offset || num >= offset+len(registers) {
			return match
		}
		if len(slashes)%2 != 0 {
			return match[1:]
		}
		return slashes + registers[num-offset]
	})
}
