package ops

import "testing"

// TestByteStreamBounds exercises the past-end guards in getBytes/readInt: once a
// read advances the position beyond the buffer, further reads return empty/zero.
func TestByteStreamBounds(t *testing.T) {
	s := newByteStream([]byte{0x01, 0x02})
	s.readInt(5) // advances pos to 5, past the 2-byte buffer
	if got := s.getBytes(1); got != nil {
		t.Fatalf("getBytes past end = %v, want nil", got)
	}
	if got := s.readInt(1); got != 0 {
		t.Fatalf("readInt past end = %d, want 0", got)
	}
}

// TestByteStreamReadBits covers a multi-byte bit read that spans a byte boundary
// and leaves a partial-byte remainder.
func TestByteStreamReadBits(t *testing.T) {
	s := newByteStream([]byte{0xff, 0xf0})
	if got := s.readBits(12); got != 0xfff {
		t.Fatalf("readBits(12) = %#x, want 0xfff", got)
	}
}

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
