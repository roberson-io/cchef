package ops

import (
	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/uregex"
)

func init() {
	core.Register(Subsection{})
}

// Subsection runs the rest of the recipe over only the parts of the data that
// match a pattern.
type Subsection struct{}

// Meta returns the operation metadata.
func (Subsection) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Subsection",
		Module:      "Default",
		Description: "Select a part of the input data using a regular expression (regex), and run all subsequent operations on each match separately.\n\nYou can use up to one capture group, where the recipe will only be run on the data in the capture group. If there's more than one capture group, only the first one will be operated on.",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the pattern, its flags, and whether a failing section stops the
// recipe.
func (Subsection) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Section (regex)", Type: core.ArgString, Value: ""},
		{Name: "Case sensitive matching", Type: core.ArgBoolean, Value: true},
		{Name: "Global matching", Type: core.ArgBoolean, Value: true},
		{Name: "Ignore errors", Type: core.ArgBoolean, Value: false},
	}
}

// Run passes the data through; outside a recipe there are no steps to run over
// the matched sections.
func (Subsection) Run(in *core.Dish, args []any) (*core.Dish, error) { return in, nil }

// RunFlow runs the steps below over each matching section, splicing the results
// back between the parts that did not match.
func (Subsection) RunFlow(state *core.FlowState) error {
	pattern := state.Args[0].(string)
	caseSensitive := state.Args[1].(bool)
	global := state.Args[2].(bool)
	ignoreErrors := state.Args[3].(bool)

	input := state.Dish.String()
	if input == "" || pattern == "" {
		return nil
	}
	if !caseSensitive {
		pattern = "(?i)" + pattern
	}
	re, err := uregex.Compile(pattern)
	if err != nil {
		return err
	}

	sub := forkSubRecipe(state)
	found := re.FindAllStringSubmatchIndex(input)
	if !global && len(found) > 1 {
		found = found[:1]
	}
	if len(found) == 0 {
		// Nothing matched, so the steps below must not run over the whole input;
		// skip past them and the Merge that closed this Subsection.
		state.Progress += len(sub) + 1
		return nil
	}

	var out []byte
	at := 0
	ran := 0
	for _, m := range found {
		// With a capture group the section is the group; the rest of the match
		// is left as it was.
		start, end := m[0], m[1]
		if len(m) >= 4 && m[2] >= 0 {
			start, end = m[2], m[3]
		}
		out = append(out, input[at:start]...)
		section, n, err := runTranche(state, sub, input[start:end], ignoreErrors)
		if err != nil {
			return err
		}
		out = append(out, section...)
		at = end
		ran = n
	}
	out = append(out, input[at:]...)

	state.Dish = core.NewDish(out, core.TypeString)
	state.Progress += ran
	return nil
}
