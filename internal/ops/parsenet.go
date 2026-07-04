package ops

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
)

// byteStream is a faithful subset of CyberChef's lib/Stream.mjs: a big-endian
// byte reader tracking a byte position and an intra-byte bit position, used by
// the packet-parsing operations.
type byteStream struct {
	bytes  []byte
	pos    int
	bitPos int
}

func newByteStream(b []byte) *byteStream { return &byteStream{bytes: b} }

func (s *byteStream) length() int { return len(s.bytes) }

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
	if s.pos > len(s.bytes) {
		return 0
	}
	val := 0
	for i := s.pos; i < s.pos+numBytes && i < len(s.bytes); i++ {
		val = val<<8 | int(s.bytes[i])
	}
	s.pos += numBytes
	s.bitPos = 0
	return val
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

// moveTo sets the stream position (used by the TLS record parser).
func (s *byteStream) moveTo(pos int) { s.pos = pos; s.bitPos = 0 }

// toHexFast renders bytes as lowercase hex with no delimiter (CyberChef toHexFast).
func toHexFast(b []byte) string { return hex.EncodeToString(b) }

// omap is an insertion-ordered JSON object. Go's encoding/json sorts map keys,
// so the packet parsers build omaps to preserve CyberChef's field order.
type omap struct {
	keys []string
	vals map[string]any
}

func newOMap() *omap { return &omap{vals: map[string]any{}} }

func (o *omap) set(key string, v any) *omap {
	if _, ok := o.vals[key]; !ok {
		o.keys = append(o.keys, key)
	}
	o.vals[key] = v
	return o
}

// jsonNoEscape marshals v to compact JSON without escaping <, > and &, matching
// JavaScript's JSON.stringify.
func jsonNoEscape(v any) ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(b.Bytes(), "\n"), nil
}

// MarshalJSON emits the object with keys in insertion order.
func (o *omap) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte('{')
	for i, k := range o.keys {
		if i > 0 {
			b.WriteByte(',')
		}
		kb, err := jsonNoEscape(k)
		if err != nil {
			return nil, err
		}
		b.Write(kb)
		b.WriteByte(':')
		vb, err := jsonNoEscape(o.vals[k])
		if err != nil {
			return nil, err
		}
		b.Write(vb)
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}

// marshalOMap renders an omap as compact JSON text (CyberChef's minified form).
func marshalOMap(o *omap) ([]byte, error) {
	return jsonNoEscape(o)
}
