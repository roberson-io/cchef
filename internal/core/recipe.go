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
func (r Recipe) ExecuteWith(reg *Registry, input *Dish) (*Dish, error) {
	dish := input
	for i, step := range r {
		if step.Disabled {
			continue
		}
		if step.Breakpoint {
			return dish, nil
		}

		op, ok := reg.Get(step.Op)
		if !ok {
			return nil, fmt.Errorf("step %d: unknown operation %q", i+1, step.Op)
		}
		meta := op.Meta()

		args, err := CoerceArgs(op.Args(), step.Args)
		if err != nil {
			return nil, fmt.Errorf("step %d (%s): %w", i+1, meta.Name, err)
		}

		in, err := dish.Get(meta.InputType)
		if err != nil {
			return nil, fmt.Errorf("step %d (%s): %w", i+1, meta.Name, err)
		}
		inDish := NewDish(toBytes(in), meta.InputType)

		out, err := op.Run(inDish, args)
		if err != nil {
			return nil, fmt.Errorf("step %d (%s): %w", i+1, meta.Name, err)
		}
		dish = out
	}
	return dish, nil
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
