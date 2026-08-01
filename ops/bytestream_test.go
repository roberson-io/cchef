package ops

import (
	"bytes"
	"testing"
)

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

// TestByteStreamReadIntEndianness covers both byte orders. Ported from
// Stream.readInt, which takes an endianness rather than assuming big.
func TestByteStreamReadIntEndianness(t *testing.T) {
	for _, tc := range []struct {
		name   string
		data   []byte
		n      int
		little bool
		want   int
	}{
		{"big endian, two bytes", []byte{0x01, 0x02}, 2, false, 0x0102},
		{"little endian, two bytes", []byte{0x01, 0x02}, 2, true, 0x0201},
		{"big endian, four bytes", []byte{0x00, 0x00, 0x01, 0x00}, 4, false, 256},
		{"little endian, four bytes", []byte{0x00, 0x01, 0x00, 0x00}, 4, true, 256},
		{"one byte is the same either way", []byte{0x7f}, 1, true, 0x7f},
		{"zero bytes reads nothing", []byte{0x01}, 0, false, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newByteStream(tc.data)
			var got int
			if tc.little {
				got = s.readIntLE(tc.n)
			} else {
				got = s.readInt(tc.n)
			}
			if got != tc.want {
				t.Errorf("got %d, want %d", got, tc.want)
			}
			if s.pos != tc.n {
				t.Errorf("position = %d, want %d", s.pos, tc.n)
			}
		})
	}
}

// TestByteStreamReadStringStopsAtNull covers Stream.readString, which builds the
// string only up to the first null byte but still advances the whole width.
func TestByteStreamReadStringStopsAtNull(t *testing.T) {
	s := newByteStream([]byte{'a', 'b', 0x00, 'c', 'd', 'e'})
	if got := s.readString(5); got != "ab" {
		t.Errorf("got %q, want %q", got, "ab")
	}
	if s.pos != 5 {
		t.Errorf("position = %d, want 5 (the full width is consumed)", s.pos)
	}
	if got := s.readString(1); got != "e" {
		t.Errorf("got %q, want %q", got, "e")
	}
}

// TestByteStreamReadStringToEnd covers the -1 width, which reads everything left.
func TestByteStreamReadStringToEnd(t *testing.T) {
	s := newByteStream([]byte("hello"))
	s.moveForwardsBy(2)
	if got := s.readString(-1); got != "llo" {
		t.Errorf("got %q, want %q", got, "llo")
	}
	if s.hasMore() {
		t.Error("stream still reports more after reading to the end")
	}
}

// TestByteStreamContinueUntilByte covers the single-byte search. It always moves
// at least one byte, so a stream already sitting on the value finds the next one.
func TestByteStreamContinueUntilByte(t *testing.T) {
	for _, tc := range []struct {
		name string
		data []byte
		from int
		val  byte
		want int
	}{
		{"finds the next occurrence", []byte{1, 2, 3, 0xff, 4}, 0, 0xff, 3},
		{"skips the byte it is sitting on", []byte{0xff, 1, 0xff, 2}, 0, 0xff, 2},
		{"runs to the end when absent", []byte{1, 2, 3}, 0, 0xff, 3},
		{"already at the end", []byte{1, 2, 3}, 3, 0xff, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newByteStream(tc.data)
			s.moveTo(tc.from)
			s.continueUntilByte(tc.val)
			if s.pos != tc.want {
				t.Errorf("position = %d, want %d", s.pos, tc.want)
			}
		})
	}
}

// TestByteStreamContinueUntil covers the byte-sequence search. CyberChef's
// version discards the current position and uses a skip table that steps over
// genuine matches; this searches forwards from where the stream actually is, so
// the cases below are what the callers of the JS function meant to ask for.
func TestByteStreamContinueUntil(t *testing.T) {
	for _, tc := range []struct {
		name string
		data []byte
		from int
		seq  []byte
		want int
	}{
		{"finds the sequence", []byte{0, 1, 0x50, 0x4b, 5, 6, 9}, 0, []byte{0x50, 0x4b}, 2},
		{
			"only looks forwards of the position",
			[]byte{1, 1, 1, 0, 0, 0, 0, 0, 1, 1, 1, 0},
			4,
			[]byte{1, 1, 1, 0},
			8,
		},
		{
			"a match behind the position is not reported",
			[]byte{1, 1, 1, 0, 0, 0, 0, 0},
			4,
			[]byte{1, 1, 1, 0},
			8,
		},
		{
			"a match the skip table would step over",
			[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 1, 0, 0, 0},
			0,
			[]byte{1, 0, 0},
			15,
		},
		{"stops at the end when absent", []byte{1, 2, 3, 4}, 0, []byte{9, 9}, 4},
		{"a sequence longer than the data", []byte{1, 2}, 0, []byte{1, 2, 3}, 2},
		{"a match right at the position", []byte{7, 8, 9}, 0, []byte{7, 8}, 0},
		{"a match at the very end", []byte{1, 2, 7, 8}, 0, []byte{7, 8}, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newByteStream(tc.data)
			s.moveTo(tc.from)
			s.continueUntil(tc.seq)
			if s.pos != tc.want {
				t.Errorf("position = %d, want %d", s.pos, tc.want)
			}
		})
	}
}

// TestByteStreamConsume covers consumeWhile and consumeIf.
func TestByteStreamConsume(t *testing.T) {
	s := newByteStream([]byte{0, 0, 0, 5, 7, 7})
	s.consumeWhile(0)
	if s.pos != 3 {
		t.Errorf("after consumeWhile position = %d, want 3", s.pos)
	}
	s.consumeIf(5)
	if s.pos != 4 {
		t.Errorf("after a matching consumeIf position = %d, want 4", s.pos)
	}
	s.consumeIf(0xff)
	if s.pos != 4 {
		t.Errorf("a non-matching consumeIf moved the position to %d", s.pos)
	}

	// consumeWhile at the end of the stream must not read past it.
	end := newByteStream([]byte{1})
	end.moveTo(1)
	end.consumeWhile(1)
	if end.pos != 1 {
		t.Errorf("consumeWhile ran past the end to %d", end.pos)
	}
	end.consumeIf(1)
	if end.pos != 1 {
		t.Errorf("consumeIf ran past the end to %d", end.pos)
	}
}

// TestByteStreamCarve covers the slice-out helper, including the bit position
// rounding up to the containing byte.
func TestByteStreamCarve(t *testing.T) {
	s := newByteStream([]byte{1, 2, 3, 4, 5})
	s.moveTo(3)
	if got := s.carve(0, s.pos); !bytes.Equal(got, []byte{1, 2, 3}) {
		t.Errorf("got %v, want [1 2 3]", got)
	}
	if got := s.carve(1, 4); !bytes.Equal(got, []byte{2, 3, 4}) {
		t.Errorf("got %v, want [2 3 4]", got)
	}

	// A part-read byte is included whole.
	bitp := newByteStream([]byte{1, 2, 3, 4})
	bitp.readBits(4)
	if got := bitp.carve(0, bitp.pos); !bytes.Equal(got, []byte{1}) {
		t.Errorf("got %v, want [1]", got)
	}
}

// TestByteStreamClone covers that a clone reads on independently of its origin.
func TestByteStreamClone(t *testing.T) {
	s := newByteStream([]byte{1, 2, 3, 4})
	s.moveForwardsBy(2)
	c := s.clone()
	if c.pos != 2 {
		t.Fatalf("clone position = %d, want 2", c.pos)
	}
	c.moveForwardsBy(1)
	if s.pos != 2 {
		t.Errorf("moving the clone moved the original to %d", s.pos)
	}
}

// TestByteStreamMoveBackwards covers stepping back, which several carvers do
// after over-reading a header.
func TestByteStreamMoveBackwards(t *testing.T) {
	s := newByteStream([]byte{1, 2, 3, 4})
	s.moveTo(3)
	s.moveBackwardsBy(2)
	if s.pos != 1 {
		t.Errorf("position = %d, want 1", s.pos)
	}
}

// TestByteStreamOutOfBounds covers the positions Stream.mjs throws on. A carver
// that walks off its buffer aborts the carve rather than returning nonsense, so
// these raise a stream error that extractFile turns back into a value.
func TestByteStreamOutOfBounds(t *testing.T) {
	for _, tc := range []struct {
		name string
		run  func(*byteStream)
	}{
		{"moveTo before the start", func(s *byteStream) { s.moveTo(-1) }},
		{"moveTo past the end", func(s *byteStream) { s.moveTo(5) }},
		{"moveForwardsBy past the end", func(s *byteStream) { s.moveForwardsBy(9) }},
		{"moveBackwardsBy before the start", func(s *byteStream) { s.moveBackwardsBy(1) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newByteStream([]byte{1, 2, 3, 4})
			if err := catchStreamError(func() { tc.run(s) }); err == nil {
				t.Errorf("no error for %s (position ended at %d)", tc.name, s.pos)
			}
		})
	}

	// The end of the buffer is a legal position: it is where a full read lands.
	s := newByteStream([]byte{1, 2, 3, 4})
	if err := catchStreamError(func() { s.moveTo(4) }); err != nil {
		t.Errorf("moving to the end reported %v", err)
	}
}

// TestCatchStreamErrorPassesOtherPanics covers that the recovery helper only
// takes back stream errors; a genuine fault must not be swallowed.
func TestCatchStreamErrorPassesOtherPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("a non-stream panic was swallowed")
		}
	}()
	_ = catchStreamError(func() { panic("something else") })
}

// TestCatchStreamErrorClean covers the case where nothing goes wrong.
func TestCatchStreamErrorClean(t *testing.T) {
	if err := catchStreamError(func() {}); err != nil {
		t.Errorf("got %v, want nil", err)
	}
}

// TestByteStreamPastEndGuards covers the reads that give up once an earlier read
// has already carried the position beyond the buffer.
func TestByteStreamPastEndGuards(t *testing.T) {
	past := func() *byteStream {
		s := newByteStream([]byte{0x01, 0x02})
		s.readInt(5) // carries the position to 5, past the 2-byte buffer
		return s
	}

	if got := past().readString(2); got != "" {
		t.Errorf("readString past the end = %q, want empty", got)
	}
	if s := past(); func() int { s.continueUntilByte(0x01); return s.pos }() != 5 {
		t.Errorf("continueUntilByte past the end moved the position to %d, want 5", s.pos)
	}
	if s := past(); func() int { s.continueUntil([]byte{0x01}); return s.pos }() != 5 {
		t.Errorf("continueUntil past the end moved the position to %d, want 5", s.pos)
	}
}

// TestByteStreamCarveClamps covers carve being asked for a region wider than the
// buffer, or an inverted one.
func TestByteStreamCarveClamps(t *testing.T) {
	s := newByteStream([]byte{1, 2, 3})
	if got := s.carve(-4, 99); !bytes.Equal(got, []byte{1, 2, 3}) {
		t.Errorf("got %v, want the whole buffer", got)
	}
	if got := s.carve(2, 1); got != nil {
		t.Errorf("an inverted region gave %v, want nil", got)
	}
}

// TestStreamErrorMessage covers the wording, which reaches the user as the
// reason a carve was abandoned.
func TestStreamErrorMessage(t *testing.T) {
	const want = "Cannot move to position -1 in stream. Out of bounds."
	if got := (streamError{pos: -1}).Error(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
