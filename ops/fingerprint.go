package ops

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/bytestream"
)

// catchStreamError runs fn, returning as an error any attempt to move outside
// the buffer. Stream.mjs throws there and the readers in internal/bytestream
// raise in the same places, which keeps the check out of every read; JA3 and
// JA3S let the throw escape rather than turning it into an operation error, so
// CyberChef reports a record that runs out mid-field as a crash. Here it is an
// ordinary error, carrying the message CyberChef throws. Panics of every other
// kind are left alone.
func catchStreamError(fn func() (*core.Dish, error)) (out *core.Dish, err error) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		raised, ok := r.(error)
		if !ok {
			panic(r)
		}
		if _, ok := errors.AsType[bytestream.StreamError](raised); !ok {
			panic(r)
		}
		out, err = nil, raised
	}()
	return fn()
}

// greaseCipherSuites are the GREASE values (RFC 8701) excluded from JA3/JA3S
// cipher, extension and curve lists.
var greaseCipherSuites = map[int]bool{
	0x0a0a: true, 0x1a1a: true, 0x2a2a: true, 0x3a3a: true,
	0x4a4a: true, 0x5a5a: true, 0x6a6a: true, 0x7a7a: true,
	0x8a8a: true, 0x9a9a: true, 0xaaaa: true, 0xbaba: true,
	0xcaca: true, 0xdada: true, 0xeaea: true, 0xfafa: true,
}

// fingerprintBytes decodes the fingerprint operation input according to the
// selected format (Hex, Base64, or Raw). Mirrors Utils.convertToByteArray, which
// is lenient: Base64 and Hex both decode what they can and never fail here.
func fingerprintBytes(s, format string) []byte {
	switch format {
	case "Base64":
		b, _ := fromBase64(s, "A-Za-z0-9+/=", true, false)
		return b
	case "Raw":
		return []byte(s)
	default: // Hex
		return hexToBytes(s)
	}
}

// parseJA3Segment reads size-byte integers from the stream and joins the
// non-GREASE values with "-".
func parseJA3Segment(s *bytestream.Stream, size int) string {
	var segment []string
	for s.HasMore() {
		el := s.ReadInt(size)
		if !greaseCipherSuites[el] {
			segment = append(segment, strconv.Itoa(el))
		}
	}
	return strings.Join(segment, "-")
}

// fingerprintError wraps a fingerprint parsing failure message.
func fingerprintError(msg string) error { return fmt.Errorf("%s", msg) }
