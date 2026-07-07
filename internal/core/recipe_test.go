package core

import (
	"fmt"
	"strings"
	"testing"
)

// failTestOp always errors, to exercise Recipe.Execute's step-error wrapping.
type failTestOp struct{}

func (failTestOp) Meta() OpMeta   { return OpMeta{Name: "Test Fail"} }
func (failTestOp) Args() []ArgDef { return nil }
func (failTestOp) Run(*Dish, []any) (*Dish, error) {
	return nil, fmt.Errorf("boom")
}

// TestRecipeStepError checks that a failing step's error is wrapped with the step
// number and operation name.
func TestRecipeStepError(t *testing.T) {
	reg := NewRegistry()
	reg.Register(failTestOp{})
	if _, err := (Recipe{{Op: "Test Fail"}}).ExecuteWith(reg, NewDish(nil, TypeString)); err == nil ||
		!strings.Contains(err.Error(), "step 1 (Test Fail)") {
		t.Fatalf("got %v, want a step-wrapped error", err)
	}
}

func TestRecipeExecute(t *testing.T) {
	reg := NewRegistry()
	reg.Register(upperTestOp{})

	r := Recipe{{Op: "Test Upper"}}
	out, err := r.ExecuteWith(reg, NewDish([]byte("hello"), TypeString))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.String() != "HELLO" {
		t.Fatalf("got %q, want HELLO", out.String())
	}
}

func TestRecipeSkipsDisabled(t *testing.T) {
	reg := NewRegistry()
	reg.Register(upperTestOp{})

	r := Recipe{{Op: "Test Upper", Disabled: true}}
	out, err := r.ExecuteWith(reg, NewDish([]byte("hello"), TypeString))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.String() != "hello" {
		t.Fatalf("disabled op ran: got %q", out.String())
	}
}

func TestRecipeUnknownOp(t *testing.T) {
	reg := NewRegistry()
	r := Recipe{{Op: "Nope"}}
	if _, err := r.ExecuteWith(reg, NewDish(nil, TypeString)); err == nil {
		t.Fatal("expected error for unknown operation")
	}
}

func TestRecipeBreakpointHalts(t *testing.T) {
	reg := NewRegistry()
	reg.Register(upperTestOp{})

	// Second op would uppercase, but the breakpoint on it should halt before it runs.
	r := Recipe{
		{Op: "Test Upper"},
		{Op: "Test Upper", Breakpoint: true},
	}
	out, err := r.ExecuteWith(reg, NewDish([]byte("hello"), TypeString))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.String() != "HELLO" {
		t.Fatalf("breakpoint did not halt correctly: got %q", out.String())
	}
}
