package ops

import (
	"regexp"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(Jump{})
	core.Register(ConditionalJump{})
}

// labelIndex finds the step index of the Label with this name, or -1. The scan
// covers every step, including disabled ones, as upstream's does.
func labelIndex(state *core.FlowState, name string) int {
	for i, step := range state.Steps {
		if step.Op != "Label" {
			continue
		}
		if len(step.Args) > 0 {
			if got, ok := step.Args[0].(string); ok && got == name {
				return i
			}
		} else if name == "" {
			return i
		}
	}
	return -1
}

// jumpTo moves execution to the named label, reporting whether it went. A jump
// is refused once the recipe's jump allowance is spent or the label does not
// exist, and a refusal resets the count so a later loop starts afresh.
func jumpTo(state *core.FlowState, label string, maxJumps int) bool {
	target := labelIndex(state, label)
	if state.NumJumps >= maxJumps || target == -1 {
		state.NumJumps = 0
		return false
	}
	// Progress is the index execution resumes after, so landing on the label's
	// own index continues with the step below it.
	state.Progress = target
	state.NumJumps++
	return true
}

// Jump moves execution to a label. Ported from CyberChef Jump.mjs.
type Jump struct{}

// Meta returns the operation metadata.
func (Jump) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Jump",
		Module:      "Default",
		Description: "Jump forwards or backwards to the specified Label",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the label and how many times a backwards jump may be taken.
func (Jump) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Label name", Type: core.ArgString, Value: ""},
		{Name: "Maximum jumps (if jumping backwards)", Type: core.ArgNumber, Value: 10},
	}
}

// Run passes the data through; outside a recipe there is nowhere to jump.
func (Jump) Run(in *core.Dish, args []any) (*core.Dish, error) { return in, nil }

// RunFlow jumps.
func (Jump) RunFlow(state *core.FlowState) error {
	jumpTo(state, state.Args[0].(string), int(state.Args[1].(float64)))
	return nil
}

// ConditionalJump moves execution to a label when the data matches. Ported from
// CyberChef ConditionalJump.mjs.
type ConditionalJump struct{}

// Meta returns the operation metadata.
func (ConditionalJump) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Conditional Jump",
		Module:      "Default",
		Description: "Conditionally jump forwards or backwards to the specified Label based on whether the data matches the specified regular expression.",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the test, whether to invert it, the label, and the jump
// allowance.
func (ConditionalJump) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Match (regex)", Type: core.ArgString, Value: ""},
		{Name: "Invert match", Type: core.ArgBoolean, Value: false},
		{Name: "Label name", Type: core.ArgString, Value: ""},
		{Name: "Maximum jumps (if jumping backwards)", Type: core.ArgNumber, Value: 10},
	}
}

// Run passes the data through; outside a recipe there is nowhere to jump.
func (ConditionalJump) Run(in *core.Dish, args []any) (*core.Dish, error) { return in, nil }

// RunFlow jumps when the data matches. An empty test never jumps, and unlike a
// refused jump it leaves the jump count alone, as upstream does.
func (ConditionalJump) RunFlow(state *core.FlowState) error {
	pattern := state.Args[0].(string)
	invert := state.Args[1].(bool)
	label := state.Args[2].(string)
	maxJumps := int(state.Args[3].(float64))

	if pattern == "" {
		return nil
	}
	// The allowance and the label are checked before the data is, so a spent
	// allowance resets the count whether or not the data would have matched.
	if state.NumJumps >= maxJumps || labelIndex(state, label) == -1 {
		state.NumJumps = 0
		return nil
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	if re.MatchString(state.Dish.String()) != invert {
		jumpTo(state, label, maxJumps)
	} else {
		state.NumJumps = 0
	}
	return nil
}
