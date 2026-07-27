package ops

import (
	"math"
	"strconv"
	"strings"
	"time"
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

// jsTrimSpace pares space off both ends the way String#trim does. It takes a
// byte order mark, which Go does not count as space, and leaves a zero width
// space, which is not space to either of them.
func jsTrimSpace(s string) string {
	return strings.TrimFunc(s, mimeIsJSSpace)
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
