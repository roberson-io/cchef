package ops

import (
	"github.com/roberson-io/cchef/core"
)

// The steering operations that carry no logic of their own. Each is a flow
// operation so that the engine hands it the execution state rather than running
// it as a transform, but three of them leave that state alone: Comment and
// Label exist to annotate and to be jumped to, and Merge is meaningful only as
// the boundary Fork and Subsection scan for.

func init() {
	core.Register(Comment{})
	core.Register(Label{})
	core.Register(Merge{})
	core.Register(Return{})
}

// Comment is a note in a recipe.
type Comment struct{}

// Meta returns the operation metadata.
func (Comment) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Comment",
		Module:      "Default",
		Description: "Provides a place to write comments within a recipe. This operation has no computational effect.",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the comment text.
func (Comment) Args() []core.ArgDef {
	return []core.ArgDef{{Name: "", Type: core.ArgString, Value: ""}}
}

// Run passes the data through.
func (Comment) Run(in *core.Dish, args []any) (*core.Dish, error) { return in, nil }

// RunFlow leaves the recipe alone.
func (Comment) RunFlow(state *core.FlowState) error { return nil }

// Label marks a place in a recipe for a jump to return to.
type Label struct{}

// Meta returns the operation metadata.
func (Label) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Label",
		Module:      "Default",
		Description: "Provides a location for conditional and fixed jumps to jump to.",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the label name.
func (Label) Args() []core.ArgDef {
	return []core.ArgDef{{Name: "Name", Type: core.ArgString, Value: ""}}
}

// Run passes the data through.
func (Label) Run(in *core.Dish, args []any) (*core.Dish, error) { return in, nil }

// RunFlow leaves the recipe alone; the jumps look the label up themselves.
func (Label) RunFlow(state *core.FlowState) error { return nil }

// Merge closes a Fork or Subsection.
type Merge struct{}

// Meta returns the operation metadata.
func (Merge) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Merge",
		Module:      "Default",
		Description: "Consolidate all branches back into a single trunk. The opposite of Fork. Unticking the Merge All checkbox will only consolidate all branches up to the nearest Fork/Subsection.",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns whether every branch is merged or only the nearest one.
func (Merge) Args() []core.ArgDef {
	return []core.ArgDef{{Name: "Merge All", Type: core.ArgBoolean, Value: true}}
}

// Run passes the data through.
func (Merge) Run(in *core.Dish, args []any) (*core.Dish, error) { return in, nil }

// RunFlow does nothing: the Fork or Subsection above has already merged when it
// found this step while collecting its sub-recipe.
func (Merge) RunFlow(state *core.FlowState) error { return nil }

// Return ends a recipe early.
type Return struct{}

// Meta returns the operation metadata.
func (Return) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Return",
		Module:      "Default",
		Description: "End execution of operations at this point in the recipe.",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns no arguments; the operation takes none.
func (Return) Args() []core.ArgDef { return nil }

// Run passes the data through.
func (Return) Run(in *core.Dish, args []any) (*core.Dish, error) { return in, nil }

// RunFlow moves past the last step, so nothing after this runs.
func (Return) RunFlow(state *core.FlowState) error {
	state.Progress = len(state.Steps)
	return nil
}
