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

// TestRegistryAllSorted covers the All() comparator, which only runs with two or
// more registered operations.
func TestRegistryAllSorted(t *testing.T) {
	reg := NewRegistry()
	reg.Register(upperTestOp{}) // "Test Upper"
	reg.Register(failTestOp{})  // "Test Fail"
	all := reg.All()
	if len(all) != 2 || all[0].Meta().Name != "Test Fail" || all[1].Meta().Name != "Test Upper" {
		t.Fatalf("All() not sorted by name: got %d ops", len(all))
	}
}

// pkgRegTestOp is a uniquely-named throwaway op for the package-level Register.
type pkgRegTestOp struct{}

func (pkgRegTestOp) Meta() OpMeta                         { return OpMeta{Name: "Test PkgRegister"} }
func (pkgRegTestOp) Args() []ArgDef                       { return nil }
func (pkgRegTestOp) Run(in *Dish, _ []any) (*Dish, error) { return in, nil }

// TestPackageRegister covers the package-level Register wrapper (adds to Default).
func TestPackageRegister(t *testing.T) {
	Register(pkgRegTestOp{})
	if _, ok := Default.Get("Test PkgRegister"); !ok {
		t.Fatal("package Register did not add the op to Default")
	}
}
