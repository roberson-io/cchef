package ops

import (
	"math"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/roberson-io/cchef/internal/jsnum"
	"github.com/roberson-io/cchef/internal/jsonval"
)

// Shims for the JavaScript built-ins operations lean on where Go's own answer
// differs.

// jsISOYearMin and jsISOYearMax bound the years JavaScript writes plainly.
// Anything outside gets a sign and six digits, which is how a date past the
// four-figure years stays unambiguous.
const (
	jsISOYearMin = 0
	jsISOYearMax = 9999
)

// jsISOTimestamp formats a moment the way Date#toISOString does, from a count of
// milliseconds since the epoch.
func jsISOTimestamp(ms int64) string {
	t := time.UnixMilli(ms).UTC()

	year := strconv.Itoa(t.Year())
	switch {
	case t.Year() < jsISOYearMin:
		year = "-" + leftPad(strconv.Itoa(-t.Year()), 6)
	case t.Year() > jsISOYearMax:
		year = "+" + leftPad(year, 6)
	default:
		year = leftPad(year, 4)
	}

	return year + t.Format("-01-02T15:04:05.000Z")
}

// jsToInt32 puts a number through the conversion JavaScript applies before a
// bitwise operator sees it: the fraction is dropped towards zero and what is
// left is read as a signed 32-bit value, so a large number wraps round rather
// than stopping at the limit. Anything that is not a number at all becomes
// zero.
func jsToInt32(f float64) int64 {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0
	}
	// #nosec G115 -- wrapping into a signed 32-bit value is exactly what this
	// conversion is for.
	return int64(int32(uint32(math.Mod(math.Trunc(f), 1<<32))))
}

// jsChr turns a number into the character it stands for. A value above the
// basic plane is split into the pair of sixteen-bit units JavaScript stores it
// as and joined back up; anything else is taken as one such unit, so a negative
// number wraps round to the top of the plane rather than being refused. A value
// that names half of a pair on its own cannot be written down, and comes back
// as the replacement character.
func jsChr(code float64) string {
	if code > 0xFFFF {
		astral := int64(code) - 0x10000
		return string(utf16.Decode([]uint16{
			uint16(0xD800 | (astral>>10)&0x3FF), // #nosec G115 -- masked to ten bits
			uint16(0xDC00 | astral&0x3FF),       // #nosec G115 -- as above
		}))
	}
	return string(utf16.Decode([]uint16{uint16(jsToInt32(code) & 0xFFFF)})) // #nosec G115 -- masked to sixteen bits
}

// jsTrimSpace pares space off both ends the way String#trim does. It takes a
// byte order mark, which Go does not count as space, and leaves a zero width
// space, which is not space to either of them.
func jsTrimSpace(s string) string {
	return strings.TrimFunc(s, jsnum.IsSpace)
}

// jsEncodeURIComponent percent-encodes every byte except the unreserved set and
// the few marks JavaScript's encodeURIComponent leaves alone. It works over the
// UTF-8 bytes, which is what the built-in does.
func jsEncodeURIComponent(s string) string {
	const keep = "-_.!~*'()"
	const hexDigits = "0123456789ABCDEF"
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || strings.IndexByte(keep, c) >= 0 {
			b.WriteByte(c)
			continue
		}
		b.WriteByte('%')
		b.WriteByte(hexDigits[c>>4])
		b.WriteByte(hexDigits[c&0x0f])
	}
	return b.String()
}

// jsSubstr mirrors JavaScript String.prototype.substr(start, length).
func jsSubstr(s string, start, length int) string {
	n := len(s)
	if start < 0 {
		start = max(n+start, 0)
	}
	if start > n {
		start = n
	}
	if length < 0 {
		length = 0
	}
	end := max(min(start+length, n), start)
	return s[start:end]
}

// jsSubstrFrom mirrors String.prototype.substr(start) (to end of string).
func jsSubstrFrom(s string, start int) string {
	n := len(s)
	if start < 0 {
		start = max(n+start, 0)
	}
	if start > n {
		start = n
	}
	return s[start:]
}

// jsToUint32 reproduces ECMAScript's ToUint32: truncate toward zero, then reduce
// modulo 2^32 into [0, 2^32).
func jsToUint32(f float64) uint32 {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0
	}
	m := math.Mod(math.Trunc(f), two32)
	if m < 0 {
		m += two32
	}
	return uint32(m) // #nosec G115 -- m is in [0, 2^32) by construction
}

// jsToString reproduces JavaScript's String(value) for the values a decoded
// MessagePack map key can take: objects become "[object Object]", arrays join
// their elements with commas (null/undefined as empty), and everything else
// follows its primitive conversion.
func jsToString(v any) string {
	switch x := v.(type) {
	case nil:
		return "null"
	case bool:
		if x {
			return "true"
		}
		return "false"
	case int64:
		return strconv.FormatInt(x, 10)
	case float64:
		return jsNumberToString(x)
	case string:
		return x
	case jsonval.Undefined:
		return "undefined"
	case []any:
		return jsArrayJoin(x)
	default: // jsonval.Object (including Buffer/ArrayBuffer)
		return "[object Object]"
	}
}

// jsNumberToString reproduces JavaScript's Number.prototype.toString for the
// finite/non-finite cases; it differs from jsonval.FormatNumber (JSON.stringify) only
// in rendering NaN and ±Infinity literally rather than as null.
func jsNumberToString(f float64) string {
	switch {
	case math.IsNaN(f):
		return "NaN"
	case math.IsInf(f, 1):
		return "Infinity"
	case math.IsInf(f, -1):
		return "-Infinity"
	}
	return jsonval.FormatNumber(f)
}

// byteSliceFrom / byteSliceRange clamp to bounds, mirroring JS Array.slice.
func byteSliceFrom(b []byte, start int) []byte {
	if start < 0 {
		start = 0
	}
	if start > len(b) {
		start = len(b)
	}
	return b[start:]
}

func byteSliceRange(b []byte, start, end int) []byte {
	if start < 0 {
		start = 0
	}
	if start > len(b) {
		start = len(b)
	}
	if end > len(b) {
		end = len(b)
	}
	if end < start {
		end = start
	}
	return b[start:end]
}
