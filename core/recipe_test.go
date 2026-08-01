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

// strInTestOp / numInTestOp exercise toBytes' string and default (number) cases.
type strInTestOp struct{}

func (strInTestOp) Meta() OpMeta {
	return OpMeta{Name: "Test StrIn", InputType: TypeString, OutputType: TypeString}
}
func (strInTestOp) Args() []ArgDef { return nil }
func (strInTestOp) Run(in *Dish, _ []any) (*Dish, error) {
	return NewDish(in.Bytes(), TypeString), nil
}

type numInTestOp struct{}

func (numInTestOp) Meta() OpMeta {
	return OpMeta{Name: "Test NumIn", InputType: TypeNumber, OutputType: TypeString}
}
func (numInTestOp) Args() []ArgDef { return nil }
func (numInTestOp) Run(in *Dish, _ []any) (*Dish, error) {
	return NewDish(in.Bytes(), TypeString), nil
}

// numArgTestOp has a numeric argument, to exercise the coerce-error branch.
type numArgTestOp struct{}

func (numArgTestOp) Meta() OpMeta {
	return OpMeta{Name: "Test NumArg", InputType: TypeByteArray, OutputType: TypeByteArray}
}
func (numArgTestOp) Args() []ArgDef                       { return []ArgDef{{Name: "n", Type: ArgNumber}} }
func (numArgTestOp) Run(in *Dish, _ []any) (*Dish, error) { return in, nil }

// failValidTestOp errors from Run while declaring a valid input type, so the
// failure surfaces at op.Run rather than at dish.Get.
type failValidTestOp struct{}

func (failValidTestOp) Meta() OpMeta {
	return OpMeta{Name: "Test FailValid", InputType: TypeByteArray, OutputType: TypeByteArray}
}
func (failValidTestOp) Args() []ArgDef                  { return nil }
func (failValidTestOp) Run(*Dish, []any) (*Dish, error) { return nil, fmt.Errorf("boom") }

// TestRecipeExecuteDefaultRegistry covers the Execute wrapper (default registry);
// an empty recipe returns the input dish unchanged.
func TestRecipeExecuteDefaultRegistry(t *testing.T) {
	out, err := (Recipe{}).Execute(NewDish([]byte("x"), TypeString))
	if err != nil || out.String() != "x" {
		t.Fatalf("Execute empty recipe = %q, %v", out.String(), err)
	}
}

// TestRecipeCoerceError covers the CoerceArgs error branch in ExecuteWith.
func TestRecipeCoerceError(t *testing.T) {
	reg := NewRegistry()
	reg.Register(numArgTestOp{})
	if _, err := (Recipe{{Op: "Test NumArg", Args: []any{"bad"}}}).ExecuteWith(reg, NewDish([]byte("x"), TypeByteArray)); err == nil {
		t.Fatal("expected a coercion error")
	}
}

// TestRecipeRunError covers the op.Run error branch (distinct from dish.Get).
func TestRecipeRunError(t *testing.T) {
	reg := NewRegistry()
	reg.Register(failValidTestOp{})
	_, err := (Recipe{{Op: "Test FailValid"}}).ExecuteWith(reg, NewDish([]byte("x"), TypeByteArray))
	if err == nil || !strings.Contains(err.Error(), "step 1 (Test FailValid)") {
		t.Fatalf("got %v, want a step-wrapped Run error", err)
	}
}

// TestRecipeToBytes covers toBytes' string and default (number) branches.
func TestRecipeToBytes(t *testing.T) {
	reg := NewRegistry()
	reg.Register(strInTestOp{})
	reg.Register(numInTestOp{})

	out, err := (Recipe{{Op: "Test StrIn"}}).ExecuteWith(reg, NewDish([]byte("hello"), TypeString))
	if err != nil || out.String() != "hello" {
		t.Fatalf("string input = %q, %v", out.String(), err)
	}
	out, err = (Recipe{{Op: "Test NumIn"}}).ExecuteWith(reg, NewDish([]byte("42"), TypeString))
	if err != nil || out.String() != "42" {
		t.Fatalf("number input = %q, %v", out.String(), err)
	}
}
