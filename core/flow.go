package core

// Flow control operations do not transform a dish; they steer the recipe that
// contains them. Instead of Run they are handed the execution state and may
// move the instruction pointer, rewrite later steps' arguments, or run part of
// the recipe over a dish of their own.

// FlowState is the state of a recipe execution, as seen by a flow operation.
type FlowState struct {
	// Steps is the recipe being executed. A flow operation may rewrite the
	// arguments of later steps; the slice is a copy made for this execution, so
	// doing so cannot affect the caller's recipe or a later run of it.
	Steps []RecipeOp
	// Progress is the index of the flow operation itself. Setting it moves
	// execution: the next step run is the one after the index left here, so
	// setting it to a step's own index resumes just after that step, and
	// setting it past the end stops the recipe.
	Progress int
	// Dish is the data as it stands. A flow operation that changes the data
	// (Fork, Subsection) replaces it.
	Dish *Dish
	// Args holds the coerced arguments of the step being run, so a flow
	// operation reads its own arguments the way an ordinary one does.
	Args []any
	// NumJumps counts jumps taken so far, shared by every jump in the recipe so
	// that a backwards jump terminates. It lasts for one execution only.
	NumJumps int
	// NumRegisters is how many registers earlier Register steps have claimed,
	// so that a second Register continues the numbering.
	NumRegisters int
	// Registry resolves step names, so a flow operation can run a sub-recipe.
	Registry *Registry
}

// FlowOperation is implemented by operations that steer the recipe. An
// operation implementing it still satisfies Operation — its Run is what a
// standalone invocation outside a recipe does.
type FlowOperation interface {
	Operation
	RunFlow(state *FlowState) error
}

// RunSteps runs a sub-recipe over its own dish and reports how many steps ran,
// which is what Fork and Subsection need for each tranche. The steps are copied
// per call, so any argument rewriting inside one tranche does not carry to the
// next.
func (s *FlowState) RunSteps(steps []RecipeOp, dish *Dish) (*Dish, int, error) {
	sub := &FlowState{
		Steps:        copySteps(steps),
		Dish:         dish,
		NumRegisters: s.NumRegisters,
		Registry:     s.Registry,
	}
	out, err := sub.run()
	if err != nil {
		return nil, 0, err
	}
	return out, len(steps), nil
}

// copySteps copies a step list deeply enough that rewriting an argument — or a
// toggle string's value inside one — cannot be seen by the original.
func copySteps(steps []RecipeOp) []RecipeOp {
	out := make([]RecipeOp, len(steps))
	for i, step := range steps {
		out[i] = step
		if step.Args != nil {
			out[i].Args = make([]any, len(step.Args))
			copy(out[i].Args, step.Args)
		}
	}
	return out
}
