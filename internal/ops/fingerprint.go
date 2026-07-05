package ops

import (
	"fmt"
	"strconv"
	"strings"
)

// greaseCipherSuites are the GREASE values (RFC 8701) excluded from JA3/JA3S
// cipher, extension and curve lists.
var greaseCipherSuites = map[int]bool{
	0x0a0a: true, 0x1a1a: true, 0x2a2a: true, 0x3a3a: true,
	0x4a4a: true, 0x5a5a: true, 0x6a6a: true, 0x7a7a: true,
	0x8a8a: true, 0x9a9a: true, 0xaaaa: true, 0xbaba: true,
	0xcaca: true, 0xdada: true, 0xeaea: true, 0xfafa: true,
}

// fingerprintBytes decodes the fingerprint operation input according to the
// selected format (Hex, Base64, or Raw). Mirrors Utils.convertToByteArray.
func fingerprintBytes(s, format string) ([]byte, error) {
	switch format {
	case "Base64":
		return fromBase64(s, "A-Za-z0-9+/=", true)
	case "Raw":
		return []byte(s), nil
	default: // Hex
		return hexToBytes(s), nil
	}
}

// parseJA3Segment reads size-byte integers from the stream and joins the
// non-GREASE values with "-". Ported from JA3Fingerprint.mjs.
func parseJA3Segment(s *byteStream, size int) string {
	var segment []string
	for s.hasMore() {
		el := s.readInt(size)
		if !greaseCipherSuites[el] {
			segment = append(segment, strconv.Itoa(el))
		}
	}
	return strings.Join(segment, "-")
}

// fingerprintError wraps a fingerprint parsing failure message.
func fingerprintError(msg string) error { return fmt.Errorf("%s", msg) }
