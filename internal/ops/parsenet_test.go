package ops

import "testing"

// TestOMapMarshalErrors covers the encoder error paths in jsonNoEscape and the
// omap value marshalling; the packet parsers only store ints/strings, so these
// forwards are unreachable through the operations themselves.
func TestOMapMarshalErrors(t *testing.T) {
	if _, err := jsonNoEscape(make(chan int)); err == nil {
		t.Fatal("jsonNoEscape(chan): expected an error")
	}
	o := newOMap()
	o.set("bad", make(chan int))
	if _, err := marshalOMap(o); err == nil {
		t.Fatal("marshalOMap(chan value): expected an error")
	}
}
