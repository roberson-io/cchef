package core

import (
	"errors"
	"strings"
	"testing"
)

// The engine changes flow control needs, each tested through a stub operation
// so the tests do not depend on any ported operation.

// upper is an ordinary operation, for checking that the normal path is intact.
type upper struct{}

func (upper) Meta() OpMeta {
	return OpMeta{Name: "test-upper", InputType: TypeString, OutputType: TypeString}
}
func (upper) Args() []ArgDef { return nil }
func (upper) Run(in *Dish, args []any) (*Dish, error) {
	return NewDish([]byte(strings.ToUpper(in.String())), TypeString), nil
}

// appendArg appends its string argument, so register substitution is visible.
type appendArg struct{}

func (appendArg) Meta() OpMeta {
	return OpMeta{Name: "test-append", InputType: TypeString, OutputType: TypeString}
}

func (appendArg) Args() []ArgDef {
	return []ArgDef{{Name: "Suffix", Type: ArgString, Value: ""}}
}

func (appendArg) Run(in *Dish, args []any) (*Dish, error) {
	return NewDish([]byte(in.String()+args[0].(string)), TypeString), nil
}

// setProgress is a flow operation that moves execution to a fixed step.
type setProgress struct{ to int }

func (setProgress) Meta() OpMeta {
	return OpMeta{Name: "test-goto", InputType: TypeString, OutputType: TypeString}
}
func (setProgress) Args() []ArgDef { return nil }
func (setProgress) Run(in *Dish, args []any) (*Dish, error) {
	return in, nil
}

func (s setProgress) RunFlow(state *FlowState) error {
	state.Progress = s.to
	return nil
}

// countJumps is a flow operation that jumps back to before the first step until
// the shared jump counter reaches its limit, so the counter's lifetime is
// observable. Progress is the index execution resumes *after*, so -1 re-runs
// step 0 — the same arithmetic that makes a jump to a label resume just after
// the label.
type countJumps struct{ max int }

func (countJumps) Meta() OpMeta {
	return OpMeta{Name: "test-loop", InputType: TypeString, OutputType: TypeString}
}
func (countJumps) Args() []ArgDef { return nil }
func (countJumps) Run(in *Dish, args []any) (*Dish, error) {
	return in, nil
}

func (c countJumps) RunFlow(state *FlowState) error {
	if state.NumJumps >= c.max {
		state.NumJumps = 0
		return nil
	}
	state.NumJumps++
	state.Progress = -1
	return nil
}

// failFlow is a flow operation that fails, so the error path is covered.
type failFlow struct{}

func (failFlow) Meta() OpMeta {
	return OpMeta{Name: "test-flowfail", InputType: TypeString, OutputType: TypeString}
}
func (failFlow) Args() []ArgDef { return nil }
func (failFlow) Run(in *Dish, args []any) (*Dish, error) {
	return in, nil
}

func (failFlow) RunFlow(state *FlowState) error {
	return errors.New("flow step failed")
}

// flowRegistry is a registry holding only the stub operations above.
func flowRegistry(t *testing.T, extra ...Operation) *Registry {
	t.Helper()
	reg := NewRegistry()
	reg.Register(upper{})
	reg.Register(appendArg{})
	for _, op := range extra {
		reg.Register(op)
	}
	return reg
}

// TestFlowStateReachesFlowOperation checks that a flow operation is handed the
// execution state — the step list, the current index and the dish — rather than
// being run as an ordinary transform.
func TestFlowStateReachesFlowOperation(t *testing.T) {
	var seen *FlowState
	reg := flowRegistry(t, spyFlow{seen: &seen})
	recipe := Recipe{{Op: "test-upper"}, {Op: "test-spy"}, {Op: "test-append", Args: []any{"!"}}}

	out, err := recipe.ExecuteWith(reg, NewDish([]byte("ab"), TypeString))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.String() != "AB!" {
		t.Errorf("got %q, want %q", out.String(), "AB!")
	}
	if seen == nil {
		t.Fatal("flow operation never ran")
	}
	if seen.Progress != 1 {
		t.Errorf("Progress = %d, want 1", seen.Progress)
	}
	if len(seen.Steps) != 3 {
		t.Errorf("Steps = %d, want 3", len(seen.Steps))
	}
	if seen.Dish.String() != "AB" {
		t.Errorf("Dish = %q, want %q", seen.Dish.String(), "AB")
	}
}

// spyFlow records the state it was handed.
type spyFlow struct{ seen **FlowState }

func (spyFlow) Meta() OpMeta {
	return OpMeta{Name: "test-spy", InputType: TypeString, OutputType: TypeString}
}
func (spyFlow) Args() []ArgDef { return nil }
func (spyFlow) Run(in *Dish, args []any) (*Dish, error) {
	return in, nil
}

func (s spyFlow) RunFlow(state *FlowState) error {
	copied := *state
	*s.seen = &copied
	return nil
}

// TestFlowProgressMovesExecution checks that setting Progress redirects the
// loop: execution continues at the step after the one named.
func TestFlowProgressMovesExecution(t *testing.T) {
	// Step 1 jumps to step 2, so the append at step 2 is skipped and the one at
	// step 3 runs.
	reg := flowRegistry(t, setProgress{to: 2})
	recipe := Recipe{
		{Op: "test-upper"},
		{Op: "test-goto"},
		{Op: "test-append", Args: []any{"skipped"}},
		{Op: "test-append", Args: []any{"!"}},
	}
	out, err := recipe.ExecuteWith(reg, NewDish([]byte("ab"), TypeString))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.String() != "AB!" {
		t.Errorf("got %q, want %q", out.String(), "AB!")
	}
}

// TestFlowProgressPastEndStops checks that moving Progress past the last step
// ends execution, which is how Return works.
func TestFlowProgressPastEndStops(t *testing.T) {
	reg := flowRegistry(t, setProgress{to: 99})
	recipe := Recipe{
		{Op: "test-upper"},
		{Op: "test-goto"},
		{Op: "test-append", Args: []any{"never"}},
	}
	out, err := recipe.ExecuteWith(reg, NewDish([]byte("ab"), TypeString))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.String() != "AB" {
		t.Errorf("got %q, want %q", out.String(), "AB")
	}
}

// TestFlowJumpCounterIsShared checks that the jump counter lives for the whole
// execution, so a backwards jump terminates, and that it starts again at zero
// on the next execution rather than leaking between runs.
func TestFlowJumpCounterIsShared(t *testing.T) {
	reg := flowRegistry(t, countJumps{max: 3})
	recipe := Recipe{
		{Op: "test-append", Args: []any{"x"}},
		{Op: "test-loop"},
	}
	for run := range 2 {
		out, err := recipe.ExecuteWith(reg, NewDish([]byte(""), TypeString))
		if err != nil {
			t.Fatalf("run %d: %v", run, err)
		}
		// One pass plus three jumps, so the append runs four times.
		if out.String() != "xxxx" {
			t.Errorf("run %d: got %q, want %q", run, out.String(), "xxxx")
		}
	}
}

// TestFlowOperationErrorIsReported checks that a failing flow operation stops
// the recipe and names the step, as ordinary operations do.
func TestFlowOperationErrorIsReported(t *testing.T) {
	reg := flowRegistry(t, failFlow{})
	recipe := Recipe{{Op: "test-upper"}, {Op: "test-flowfail"}}
	if _, err := recipe.ExecuteWith(reg, NewDish([]byte("ab"), TypeString)); err == nil ||
		!strings.Contains(err.Error(), "flow step failed") ||
		!strings.Contains(err.Error(), "step 2") {
		t.Errorf("got %v, want a step 2 failure", err)
	}
}

// TestFlowStepsAreCopiedPerExecution checks that a flow operation rewriting a
// later step's arguments — as Register does — cannot affect the caller's recipe
// or a later execution of it.
func TestFlowStepsAreCopiedPerExecution(t *testing.T) {
	reg := flowRegistry(t, rewriteArgs{})
	recipe := Recipe{
		{Op: "test-rewrite"},
		{Op: "test-append", Args: []any{"original"}},
	}
	for run := range 2 {
		out, err := recipe.ExecuteWith(reg, NewDish([]byte("v="), TypeString))
		if err != nil {
			t.Fatalf("run %d: %v", run, err)
		}
		if out.String() != "v=rewritten" {
			t.Errorf("run %d: got %q", run, out.String())
		}
	}
	if got := recipe[1].Args[0]; got != "original" {
		t.Errorf("caller's recipe was modified: arg is now %q", got)
	}
}

// rewriteArgs replaces the next step's first argument, the way Register does.
type rewriteArgs struct{}

func (rewriteArgs) Meta() OpMeta {
	return OpMeta{Name: "test-rewrite", InputType: TypeString, OutputType: TypeString}
}
func (rewriteArgs) Args() []ArgDef { return nil }
func (rewriteArgs) Run(in *Dish, args []any) (*Dish, error) {
	return in, nil
}

func (rewriteArgs) RunFlow(state *FlowState) error {
	state.Steps[state.Progress+1].Args[0] = "rewritten"
	return nil
}

// TestFlowSubRecipeRuns checks running part of a recipe over a fresh dish, which
// is what Fork and Subsection need, and that it reports how many steps it ran.
func TestFlowSubRecipeRuns(t *testing.T) {
	reg := flowRegistry(t)
	steps := []RecipeOp{{Op: "test-upper"}, {Op: "test-append", Args: []any{"!"}}}
	state := &FlowState{Registry: reg}

	out, ran, err := state.RunSteps(steps, NewDish([]byte("ab"), TypeString))
	if err != nil {
		t.Fatalf("RunSteps: %v", err)
	}
	if out.String() != "AB!" {
		t.Errorf("got %q, want %q", out.String(), "AB!")
	}
	if ran != 2 {
		t.Errorf("ran = %d, want 2", ran)
	}
}

// TestFlowSubRecipeIsIndependent checks that each sub-recipe run starts from the
// steps as given, so a register set in one tranche does not carry to the next.
func TestFlowSubRecipeIsIndependent(t *testing.T) {
	reg := flowRegistry(t, rewriteArgs{})
	steps := []RecipeOp{{Op: "test-rewrite"}, {Op: "test-append", Args: []any{"original"}}}
	state := &FlowState{Registry: reg}

	for run := range 2 {
		out, _, err := state.RunSteps(steps, NewDish([]byte("v="), TypeString))
		if err != nil {
			t.Fatalf("run %d: %v", run, err)
		}
		if out.String() != "v=rewritten" {
			t.Errorf("run %d: got %q", run, out.String())
		}
	}
	if got := steps[1].Args[0]; got != "original" {
		t.Errorf("the steps given were modified: arg is now %q", got)
	}
}

// TestDisabledAndBreakpointStillWork checks the existing behaviour is intact
// alongside the flow-control loop.
func TestDisabledAndBreakpointStillWork(t *testing.T) {
	reg := flowRegistry(t)
	disabled := Recipe{
		{Op: "test-append", Args: []any{"a"}},
		{Op: "test-append", Args: []any{"b"}, Disabled: true},
		{Op: "test-append", Args: []any{"c"}},
	}
	out, err := disabled.ExecuteWith(reg, NewDish([]byte(""), TypeString))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.String() != "ac" {
		t.Errorf("disabled: got %q, want %q", out.String(), "ac")
	}

	breakpoint := Recipe{
		{Op: "test-append", Args: []any{"a"}},
		{Op: "test-append", Args: []any{"b"}, Breakpoint: true},
	}
	out, err = breakpoint.ExecuteWith(reg, NewDish([]byte(""), TypeString))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.String() != "a" {
		t.Errorf("breakpoint: got %q, want %q", out.String(), "a")
	}
}

// TestFlowSubRecipeReportsFailure checks that a sub-recipe's failure reaches the
// caller, which is how Fork and Subsection decide whether to stop.
func TestFlowSubRecipeReportsFailure(t *testing.T) {
	reg := flowRegistry(t)
	state := &FlowState{Registry: reg}
	_, _, err := state.RunSteps([]RecipeOp{{Op: "test-missing"}}, NewDish(nil, TypeString))
	if err == nil {
		t.Error("want the sub-recipe's failure reported")
	}
}
