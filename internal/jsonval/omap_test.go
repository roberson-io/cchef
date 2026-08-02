package jsonval

import "testing"

// TestOMapMarshalErrors covers the encoder error paths in jsonNoEscape and the
// omap value marshalling; the packet parsers only store ints/strings, so these
// forwards are unreachable through the operations themselves.
func TestOMapMarshalErrors(t *testing.T) {
	if _, err := MarshalNoEscape(make(chan int)); err == nil {
		t.Fatal("jsonNoEscape(chan): expected an error")
	}
	o := NewOMap()
	o.Set("bad", make(chan int))
	if _, err := MarshalOMap(o); err == nil {
		t.Fatal("marshalOMap(chan value): expected an error")
	}
}

// TestOMapAccessors covers reading back what was set: values by key, and the
// keys in insertion order with a repeated Set not moving its key.
func TestOMapAccessors(t *testing.T) {
	o := NewOMap().Set("b", 1).Set("a", "x").Set("b", 2)
	if v, ok := o.Get("a"); !ok || v != "x" {
		t.Errorf("Get(a) = %v, %v", v, ok)
	}
	if v, ok := o.Get("b"); !ok || v != 2 {
		t.Errorf("Get(b) = %v, %v; a second Set must overwrite", v, ok)
	}
	if _, ok := o.Get("missing"); ok {
		t.Error("Get of an absent key reported ok")
	}
	if got := o.Keys(); len(got) != 2 || got[0] != "b" || got[1] != "a" {
		t.Errorf("Keys() = %v, want [b a]", got)
	}
}

// TestOMapValue covers the single-value read, which returns nil for an absent
// key the way a Go map read returns the zero value.
func TestOMapValue(t *testing.T) {
	o := NewOMap().Set("k", 7)
	if got := o.Value("k"); got != 7 {
		t.Errorf("Value(k) = %v, want 7", got)
	}
	if got := o.Value("absent"); got != nil {
		t.Errorf("Value(absent) = %v, want nil", got)
	}
}

// TestOMapMerge covers appending one map's entries into another: src's keys
// arrive in order after dst's, and a shared key takes src's value without
// moving position.
func TestOMapMerge(t *testing.T) {
	dst := NewOMap().Set("a", 1).Set("b", 2)
	src := NewOMap().Set("c", 3).Set("b", 20)
	dst.Merge(src)
	keys := dst.Keys()
	if len(keys) != 3 || keys[0] != "a" || keys[1] != "b" || keys[2] != "c" {
		t.Fatalf("Keys() = %v, want [a b c]", keys)
	}
	if got := dst.Value("b"); got != 20 {
		t.Errorf("Value(b) = %v, want 20 (src wins)", got)
	}
}
