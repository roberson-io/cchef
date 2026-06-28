package core

import "testing"

// upper is a tiny test operation that uppercases ASCII letters.
type upperTestOp struct{}

func (upperTestOp) Meta() OpMeta {
	return OpMeta{Name: "Test Upper", Module: "Test", InputType: TypeByteArray, OutputType: TypeByteArray}
}
func (upperTestOp) Args() []ArgDef { return nil }
func (upperTestOp) Run(in *Dish, args []any) (*Dish, error) {
	b := append([]byte(nil), in.Bytes()...)
	for i, c := range b {
		if c >= 'a' && c <= 'z' {
			b[i] = c - 32
		}
	}
	return NewDish(b, TypeByteArray), nil
}

func TestRegistryRegisterAndGet(t *testing.T) {
	reg := NewRegistry()
	reg.Register(upperTestOp{})

	op, ok := reg.Get("Test Upper")
	if !ok {
		t.Fatal("Get(\"Test Upper\") not found")
	}
	if op.Meta().Name != "Test Upper" {
		t.Fatalf("got %q", op.Meta().Name)
	}
	if _, ok := reg.Get("Missing"); ok {
		t.Fatal("expected Get of missing op to fail")
	}
	if len(reg.All()) != 1 {
		t.Fatalf("All() = %d, want 1", len(reg.All()))
	}
}
