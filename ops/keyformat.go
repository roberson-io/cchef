package ops

import (
	"encoding/base64"
	"strconv"
	"strings"
	"unicode"
)

// Decoding for key and operand arguments that arrive as a toggleString: the
// user picks the spelling (Hex, Decimal, Binary, Base64, UTF8 or Latin1) and
// the operation needs the raw bytes.

// convertToByteArray decodes a key string according to its encoding mode.
func convertToByteArray(str, mode string) ([]byte, error) {
	switch strings.ToLower(mode) {
	case "hex":
		return splitHexToBytes(str), nil
	case "decimal":
		// CyberChef's fromDecimal is called with delim "Auto", which charRep maps
		// to undefined; `str.split(undefined)` yields the whole string as one token
		// and parseInt reads only the leading integer. So a decimal key is a single
		// byte (e.g. "82 226" -> [82]).
		//
		// A key with no leading integer is [NaN] in CyberChef, whose per-operation
		// behaviour can't be reproduced by any single byte (ADD/SUB zero the input
		// while OR leaves it unchanged). We treat it as identity ([0]); the only
		// divergence is ADD/SUB with a fully non-numeric decimal key.
		if v, ok := leadingInt(str); ok {
			return []byte{byte(v)}, nil // #nosec G115 -- XOR/operand result bounded to a byte
		}
		return []byte{0}, nil
	case "binary":
		return fromBinaryKey(str), nil
	case "base64":
		return base64.StdEncoding.DecodeString(strings.TrimSpace(str))
	case "utf8":
		return []byte(str), nil
	default: // latin1
		out := make([]byte, 0, len(str))
		for _, r := range str {
			out = append(out, byte(r)) // #nosec G115 -- XOR/operand result bounded to a byte
		}
		return out, nil
	}
}

// leadingInt parses a leading base-10 integer the way JavaScript's parseInt
// does: skip leading whitespace, take an optional sign and the following
// digits, and stop at the first non-digit. Unlike jsnum.ParseInt it reports
// false when the digits do not fit an int, which the decimal-key path treats
// the same as no integer at all.
func leadingInt(s string) (int, bool) {
	s = strings.TrimLeft(s, " \t\n\r\f\v")
	i := 0
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		i++
	}
	start := i
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == start {
		return 0, false
	}
	v, err := strconv.Atoi(s[:i])
	if err != nil {
		return 0, false
	}
	return v, true
}

// fromBinaryKey decodes a binary key the way CyberChef's fromBinary does when
// called via convertToByteArray: strip all whitespace (the "Space" delimiter is
// the regex /\s+/g) and read fixed 8-bit chunks, so "0110100001101001" becomes
// two bytes. A trailing partial chunk is parsed as-is.
func fromBinaryKey(str string) []byte {
	var sb strings.Builder
	for _, r := range str {
		if !unicode.IsSpace(r) {
			sb.WriteRune(r)
		}
	}
	s := sb.String()
	out := make([]byte, 0, len(s)/8+1)
	for i := 0; i < len(s); i += 8 {
		v, _ := strconv.ParseUint(s[i:min(i+8, len(s))], 2, 32)
		out = append(out, byte(v)) // #nosec G115 -- XOR/operand result bounded to a byte
	}
	return out
}
