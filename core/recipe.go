package core

import "fmt"

// RecipeOp is a single step in a recipe: an operation name plus its argument
// values. It matches CyberChef's JSON recipe entry {op, args, disabled?, breakpoint?}.
type RecipeOp struct {
	Op         string `json:"op"`
	Args       []any  `json:"args,omitempty"`
	Disabled   bool   `json:"disabled,omitempty"`
	Breakpoint bool   `json:"breakpoint,omitempty"`
}

// Recipe is an ordered list of operations executed in sequence.
type Recipe []RecipeOp

// Execute runs the recipe against the input dish using the default registry.
func (r Recipe) Execute(input *Dish) (*Dish, error) {
	return r.ExecuteWith(Default, input)
}

// ExecuteWith runs the recipe against the input dish using the given registry.
// Each step converts the dish to the operation's input type before running and
// stores the result as the operation's output type. Disabled steps are skipped;
// a breakpoint halts execution before that step runs, returning the dish so far.
//
// A flow control step is handed the execution state instead and may move to
// another step, so the steps are copied for this run: one that rewrites a later
// step's arguments cannot affect the caller's recipe.
func (r Recipe) ExecuteWith(reg *Registry, input *Dish) (*Dish, error) {
	state := &FlowState{Steps: copySteps(r), Dish: input, Registry: reg}
	return state.run()
}

// run is the execution loop, shared by a whole recipe and by the sub-recipes
// Fork and Subsection run over their tranches.
func (s *FlowState) run() (*Dish, error) {
	for s.Progress = 0; s.Progress < len(s.Steps); s.Progress++ {
		step := s.Steps[s.Progress]
		if step.Disabled {
			continue
		}
		if step.Breakpoint {
			return s.Dish, nil
		}
		if err := s.runStep(step); err != nil {
			return nil, err
		}
	}
	return s.Dish, nil
}

// runStep runs one step, updating the state's dish or, for a flow control
// operation, letting it update the state itself.
func (s *FlowState) runStep(step RecipeOp) error {
	op, ok := s.Registry.Get(step.Op)
	if !ok {
		return fmt.Errorf("step %d: unknown operation %q", s.Progress+1, step.Op)
	}
	meta := op.Meta()

	args, err := CoerceArgs(op.Args(), step.Args)
	if err != nil {
		return fmt.Errorf("step %d (%s): %w", s.Progress+1, meta.Name, err)
	}

	if flow, isFlow := op.(FlowOperation); isFlow {
		// The state carries the coerced arguments of the step being run, so a
		// flow operation reads them the same way an ordinary one does.
		s.Args = args
		if err := flow.RunFlow(s); err != nil {
			return fmt.Errorf("step %d (%s): %w", s.Progress+1, meta.Name, err)
		}
		return nil
	}

	in, err := s.Dish.Get(meta.InputType)
	if err != nil {
		return fmt.Errorf("step %d (%s): %w", s.Progress+1, meta.Name, err)
	}
	out, err := op.Run(NewDish(toBytes(in), meta.InputType), args)
	if err != nil {
		return fmt.Errorf("step %d (%s): %w", s.Progress+1, meta.Name, err)
	}
	s.Dish = out
	return nil
}

// toBytes renders a value returned by Dish.Get back into canonical byte storage.
func toBytes(v any) []byte {
	switch x := v.(type) {
	case []byte:
		return x
	case string:
		return []byte(x)
	default:
		return fmt.Append(nil, x)
	}
}
