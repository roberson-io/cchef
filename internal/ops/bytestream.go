package ops

import (
	"bytes"
	"errors"
	"fmt"
)

// byteStream is a port of CyberChef's lib/Stream.mjs: a byte reader tracking a
// byte position and an intra-byte bit position. The packet parsers use the
// reading half; the file carvers additionally seek about and slice pieces out.
type byteStream struct {
	bytes  []byte
	pos    int
	bitPos int
}

func newByteStream(b []byte) *byteStream { return &byteStream{bytes: b} }

func (s *byteStream) length() int { return len(s.bytes) }

// clone returns a reader over the same bytes at the same position. The two then
// move independently, which lets a carver look ahead without losing its place.
func (s *byteStream) clone() *byteStream {
	return &byteStream{bytes: s.bytes, pos: s.pos, bitPos: s.bitPos}
}

// streamError is raised when a stream is asked to move outside its buffer.
// Stream.mjs throws in the same places, and a carver that walks off the end of
// its buffer has misread the file and must abandon the carve; unwinding to
// extractFile keeps that check out of every read. catchStreamError turns it back
// into an ordinary error value at that boundary.
type streamError struct{ pos int }

func (e streamError) Error() string {
	return fmt.Sprintf("Cannot move to position %d in stream. Out of bounds.", e.pos)
}

// catchStreamError runs fn, returning as an error any out-of-bounds move, or any
// other complaint a reader raised about the shape of what it was reading. Panics
// of every other kind are left alone.
func catchStreamError(fn func()) (err error) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		raised, ok := r.(error)
		if !ok {
			panic(r)
		}
		var se streamError
		var cf carveFailure
		if !errors.As(raised, &se) && !errors.As(raised, &cf) {
			panic(r)
		}
		err = raised
	}()
	fn()
	return nil
}

// getBytes returns numBytes bytes from the current position (all remaining when
// numBytes < 0), advancing the position and clearing the bit position.
func (s *byteStream) getBytes(numBytes int) []byte {
	if s.pos > len(s.bytes) {
		return nil
	}
	newPos := len(s.bytes)
	if numBytes >= 0 {
		newPos = s.pos + numBytes
	}
	if newPos > len(s.bytes) {
		newPos = len(s.bytes)
	}
	b := s.bytes[s.pos:newPos]
	s.pos = newPos
	s.bitPos = 0
	return b
}

// readInt reads a big-endian integer of numBytes bytes.
func (s *byteStream) readInt(numBytes int) int {
	return s.readIntOrder(numBytes, false)
}

// readIntLE reads a little-endian integer of numBytes bytes.
func (s *byteStream) readIntLE(numBytes int) int {
	return s.readIntOrder(numBytes, true)
}

// readIntOrder reads an integer in either byte order. Bytes past the end of the
// buffer read as zero, as indexing past a Uint8Array yields undefined and the
// JS bitwise operators coerce that to 0.
func (s *byteStream) readIntOrder(numBytes int, little bool) int {
	if s.pos > len(s.bytes) {
		return 0
	}
	val := 0
	if little {
		for i := s.pos + numBytes - 1; i >= s.pos; i-- {
			val = val<<8 | s.at(i)
		}
	} else {
		for i := s.pos; i < s.pos+numBytes; i++ {
			val = val<<8 | s.at(i)
		}
	}
	s.pos += numBytes
	s.bitPos = 0
	return val
}

// peek returns the byte at absolute position i without moving the stream. A
// position outside the buffer is an out-of-bounds read, as for a move.
func (s *byteStream) peek(i int) byte {
	if i < 0 || i >= len(s.bytes) {
		panic(streamError{pos: i})
	}
	return s.bytes[i]
}

// at returns the byte at i, or 0 past either end of the buffer.
func (s *byteStream) at(i int) int {
	if i < 0 || i >= len(s.bytes) {
		return 0
	}
	return int(s.bytes[i])
}

// readBits reads numBits big-endian bits, tracking the intra-byte position so
// successive sub-byte reads compose correctly. Ported from Stream.readBits (be).
func (s *byteStream) readBits(numBits int) int {
	bitBuf := int(s.bytes[s.pos]) & ((1 << (8 - s.bitPos)) - 1)
	s.pos++
	bitBufLen := 8 - s.bitPos
	s.bitPos = 0

	for bitBufLen < numBits {
		bitBuf = bitBuf<<bitBufLen | int(s.bytes[s.pos])
		s.pos++
		bitBufLen += 8
	}
	if bitBufLen > numBits {
		excess := bitBufLen - numBits
		bitBuf >>= excess
		s.pos--
		s.bitPos = 8 - excess
	}
	return bitBuf
}

func (s *byteStream) hasMore() bool { return s.pos < len(s.bytes) }

// moveTo sets the stream position, which must lie within the buffer. The end of
// the buffer counts as within it: that is where a read of the last byte lands.
func (s *byteStream) moveTo(pos int) {
	if pos < 0 || pos > len(s.bytes) {
		panic(streamError{pos: pos})
	}
	s.pos = pos
	s.bitPos = 0
}

// moveForwardsBy advances the stream position by n bytes.
func (s *byteStream) moveForwardsBy(n int) { s.moveTo(s.pos + n) }

// advance moves the position by n without requiring the result to lie inside
// the buffer. Stream.mjs has no such method: the carvers that need it assign to
// `position` directly, which skips the bounds check that moveForwardsBy makes,
// and rely on running off the end to finish a loop. Moving to before the start
// is still refused, as no read could follow it.
func (s *byteStream) advance(n int) {
	if s.pos+n < 0 {
		panic(streamError{pos: s.pos + n})
	}
	s.pos += n
	s.bitPos = 0
}

// moveBackwardsBy retreats the stream position by n bytes.
func (s *byteStream) moveBackwardsBy(n int) { s.moveTo(s.pos - n) }

// readString reads numBytes bytes as a string, stopping the string at the first
// null byte but still consuming the whole width. A negative width reads to the
// end. Ported from Stream.readString.
func (s *byteStream) readString(numBytes int) string {
	if s.pos > len(s.bytes) {
		return ""
	}
	if numBytes < 0 {
		numBytes = len(s.bytes) - s.pos
	}
	var out []byte
	for i := s.pos; i < s.pos+numBytes && i < len(s.bytes); i++ {
		if s.bytes[i] == 0 {
			break
		}
		out = append(out, s.bytes[i])
	}
	s.pos += numBytes
	s.bitPos = 0
	return string(out)
}

// continueUntilByte moves to the next occurrence of val, or to the end of the
// buffer if there is none. The byte under the current position is skipped, so
// repeated calls walk successive occurrences rather than standing still.
func (s *byteStream) continueUntilByte(val byte) {
	if s.pos > len(s.bytes) {
		return
	}
	s.bitPos = 0
	for {
		s.pos++
		if s.pos >= len(s.bytes) {
			s.pos = len(s.bytes)
			return
		}
		if s.bytes[s.pos] == val {
			return
		}
	}
}

// continueUntil moves to the start of the next occurrence of seq at or after the
// current position, or to the end of the buffer if there is none.
//
// CyberChef's Stream.continueUntil takes a different path for a sequence than
// for a single byte: it abandons the current position and scans from the length
// of the sequence, and its skip table is indexed by the pattern byte that failed
// rather than the text byte found, so the scan can step over a match that is
// there. Both are defects — the callers all mean "find the next one from here" —
// so this searches plainly forwards instead. On well-formed input the two agree.
func (s *byteStream) continueUntil(seq []byte) {
	if s.pos > len(s.bytes) {
		return
	}
	s.bitPos = 0
	if i := bytes.Index(s.bytes[s.pos:], seq); i >= 0 {
		s.pos += i
		return
	}
	s.pos = len(s.bytes)
}

// lookingAt reports whether seq sits at the current position, without moving.
func (s *byteStream) lookingAt(seq []byte) bool {
	if s.pos < 0 || s.pos+len(seq) > len(s.bytes) {
		return false
	}
	return bytes.Equal(s.bytes[s.pos:s.pos+len(seq)], seq)
}

// consumeWhile advances over a run of val.
func (s *byteStream) consumeWhile(val byte) {
	for s.pos < len(s.bytes) && s.bytes[s.pos] == val {
		s.pos++
	}
	s.bitPos = 0
}

// consumeIf advances over the next byte when it is val.
func (s *byteStream) consumeIf(val byte) {
	if s.pos < len(s.bytes) && s.bytes[s.pos] == val {
		s.pos++
		s.bitPos = 0
	}
}

// carve returns the bytes between start and finish. A part-read byte at finish
// is taken whole, as the bits already consumed belong to the carved region.
func (s *byteStream) carve(start, finish int) []byte {
	if s.bitPos > 0 {
		finish++
	}
	if start < 0 {
		start = 0
	}
	if finish > len(s.bytes) {
		finish = len(s.bytes)
	}
	if start > finish {
		return nil
	}
	return s.bytes[start:finish]
}
