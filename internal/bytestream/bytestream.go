// Package bytestream reads binary formats a field at a time.
//
// A [Stream] tracks a byte position and an intra-byte bit position, so
// sub-byte fields compose without the caller doing the shifting. Moving
// outside the buffer panics with a [StreamError] rather than returning one:
// a parser that has walked off the end has misread its input and cannot
// usefully continue, so the caller recovers once at the point where giving up
// makes sense instead of checking after every read.
package bytestream

import (
	"bytes"
	"fmt"
)

// Stream is a port of CyberChef's lib/Stream.mjs: a byte reader tracking a
// byte position and an intra-byte bit position. The packet parsers use the
// reading half; the file carvers additionally seek about and slice pieces out.
type Stream struct {
	Bytes  []byte
	Pos    int
	bitPos int
}

// New returns a stream positioned at the start of b. The stream reads b in
// place and never copies it.
func New(b []byte) *Stream { return &Stream{Bytes: b} }

// Length returns the size of the underlying buffer.
func (s *Stream) Length() int { return len(s.Bytes) }

// Clone returns a reader over the same bytes at the same position. The two then
// move independently, which lets a carver look ahead without losing its place.
func (s *Stream) Clone() *Stream {
	return &Stream{Bytes: s.Bytes, Pos: s.Pos, bitPos: s.bitPos}
}

// StreamError is raised when a stream is asked to move outside its buffer.
// Stream.mjs throws in the same places, and a carver that walks off the end of
// its buffer has misread the file and must abandon the carve; unwinding to
// the boundary that owns the read keeps that check out of every read, where a
// recovery turns it back into an ordinary error value.
type StreamError struct{ Pos int }

func (e StreamError) Error() string {
	return fmt.Sprintf("Cannot move to position %d in stream. Out of bounds.", e.Pos)
}

// GetBytes returns numBytes bytes from the current position (all remaining when
// numBytes < 0), advancing the position and clearing the bit position.
func (s *Stream) GetBytes(numBytes int) []byte {
	if s.Pos > len(s.Bytes) {
		return nil
	}
	newPos := len(s.Bytes)
	if numBytes >= 0 {
		newPos = s.Pos + numBytes
	}
	if newPos > len(s.Bytes) {
		newPos = len(s.Bytes)
	}
	b := s.Bytes[s.Pos:newPos]
	s.Pos = newPos
	s.bitPos = 0
	return b
}

// ReadInt reads a big-endian integer of numBytes bytes.
func (s *Stream) ReadInt(numBytes int) int {
	return s.ReadIntOrder(numBytes, false)
}

// ReadIntLE reads a little-endian integer of numBytes bytes.
func (s *Stream) ReadIntLE(numBytes int) int {
	return s.ReadIntOrder(numBytes, true)
}

// ReadIntOrder reads an integer in either byte order. Bytes past the end of the
// buffer read as zero, as indexing past a Uint8Array yields undefined and the
// JS bitwise operators coerce that to 0.
func (s *Stream) ReadIntOrder(numBytes int, little bool) int {
	if s.Pos > len(s.Bytes) {
		return 0
	}
	val := 0
	if little {
		for i := s.Pos + numBytes - 1; i >= s.Pos; i-- {
			val = val<<8 | s.At(i)
		}
	} else {
		for i := s.Pos; i < s.Pos+numBytes; i++ {
			val = val<<8 | s.At(i)
		}
	}
	s.Pos += numBytes
	s.bitPos = 0
	return val
}

// Peek returns the byte at absolute position i without moving the stream. A
// position outside the buffer is an out-of-bounds read, as for a move.
func (s *Stream) Peek(i int) byte {
	if i < 0 || i >= len(s.Bytes) {
		panic(StreamError{Pos: i})
	}
	return s.Bytes[i]
}

// At returns the byte At i, or 0 past either end of the buffer.
func (s *Stream) At(i int) int {
	if i < 0 || i >= len(s.Bytes) {
		return 0
	}
	return int(s.Bytes[i])
}

// ReadBits reads numBits big-endian bits, tracking the intra-byte position so
// successive sub-byte reads compose correctly.
func (s *Stream) ReadBits(numBits int) int {
	bitBuf := int(s.Bytes[s.Pos]) & ((1 << (8 - s.bitPos)) - 1)
	s.Pos++
	bitBufLen := 8 - s.bitPos
	s.bitPos = 0

	for bitBufLen < numBits {
		bitBuf = bitBuf<<bitBufLen | int(s.Bytes[s.Pos])
		s.Pos++
		bitBufLen += 8
	}
	if bitBufLen > numBits {
		excess := bitBufLen - numBits
		bitBuf >>= excess
		s.Pos--
		s.bitPos = 8 - excess
	}
	return bitBuf
}

// HasMore reports whether any byte remains to be read.
func (s *Stream) HasMore() bool { return s.Pos < len(s.Bytes) }

// MoveTo sets the stream position, which must lie within the buffer. The end of
// the buffer counts as within it: that is where a read of the last byte lands.
func (s *Stream) MoveTo(pos int) {
	if pos < 0 || pos > len(s.Bytes) {
		panic(StreamError{Pos: pos})
	}
	s.Pos = pos
	s.bitPos = 0
}

// MoveForwardsBy advances the stream position by n bytes.
func (s *Stream) MoveForwardsBy(n int) { s.MoveTo(s.Pos + n) }

// Advance moves the position by n without requiring the result to lie inside
// the buffer. Stream.mjs has no such method: the carvers that need it assign to
// `position` directly, which skips the bounds check that moveForwardsBy makes,
// and rely on running off the end to finish a loop. Moving to before the start
// is still refused, as no read could follow it.
func (s *Stream) Advance(n int) {
	if s.Pos+n < 0 {
		panic(StreamError{Pos: s.Pos + n})
	}
	s.Pos += n
	s.bitPos = 0
}

// MoveBackwardsBy retreats the stream position by n bytes.
func (s *Stream) MoveBackwardsBy(n int) { s.MoveTo(s.Pos - n) }

// ReadString reads numBytes bytes as a string, stopping the string at the first
// null byte but still consuming the whole width. A negative width reads to the
// end.
func (s *Stream) ReadString(numBytes int) string {
	if s.Pos > len(s.Bytes) {
		return ""
	}
	if numBytes < 0 {
		numBytes = len(s.Bytes) - s.Pos
	}
	var out []byte
	for i := s.Pos; i < s.Pos+numBytes && i < len(s.Bytes); i++ {
		if s.Bytes[i] == 0 {
			break
		}
		out = append(out, s.Bytes[i])
	}
	s.Pos += numBytes
	s.bitPos = 0
	return string(out)
}

// ContinueUntilByte moves to the next occurrence of val, or to the end of the
// buffer if there is none. The byte under the current position is skipped, so
// repeated calls walk successive occurrences rather than standing still.
func (s *Stream) ContinueUntilByte(val byte) {
	if s.Pos > len(s.Bytes) {
		return
	}
	s.bitPos = 0
	for {
		s.Pos++
		if s.Pos >= len(s.Bytes) {
			s.Pos = len(s.Bytes)
			return
		}
		if s.Bytes[s.Pos] == val {
			return
		}
	}
}

// ContinueUntil moves to the start of the next occurrence of seq at or after the
// current position, or to the end of the buffer if there is none.
//
// CyberChef's Stream.ContinueUntil takes a different path for a sequence than
// for a single byte: it abandons the current position and scans from the length
// of the sequence, and its skip table is indexed by the pattern byte that failed
// rather than the text byte found, so the scan can step over a match that is
// there. Both are defects — the callers all mean "find the next one from here" —
// so this searches plainly forwards instead. On well-formed input the two agree.
func (s *Stream) ContinueUntil(seq []byte) {
	if s.Pos > len(s.Bytes) {
		return
	}
	s.bitPos = 0
	if i := bytes.Index(s.Bytes[s.Pos:], seq); i >= 0 {
		s.Pos += i
		return
	}
	s.Pos = len(s.Bytes)
}

// LookingAt reports whether seq sits at the current position, without moving.
func (s *Stream) LookingAt(seq []byte) bool {
	if s.Pos < 0 || s.Pos+len(seq) > len(s.Bytes) {
		return false
	}
	return bytes.Equal(s.Bytes[s.Pos:s.Pos+len(seq)], seq)
}

// ConsumeWhile advances over a run of val.
func (s *Stream) ConsumeWhile(val byte) {
	for s.Pos < len(s.Bytes) && s.Bytes[s.Pos] == val {
		s.Pos++
	}
	s.bitPos = 0
}

// ConsumeIf advances over the next byte when it is val.
func (s *Stream) ConsumeIf(val byte) {
	if s.Pos < len(s.Bytes) && s.Bytes[s.Pos] == val {
		s.Pos++
		s.bitPos = 0
	}
}

// Carve returns the bytes between start and finish. A part-read byte at finish
// is taken whole, as the bits already consumed belong to the carved region.
func (s *Stream) Carve(start, finish int) []byte {
	if s.bitPos > 0 {
		finish++
	}
	if start < 0 {
		start = 0
	}
	if finish > len(s.Bytes) {
		finish = len(s.Bytes)
	}
	if start > finish {
		return nil
	}
	return s.Bytes[start:finish]
}
