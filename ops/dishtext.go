package ops

import (
	"strings"
	"unicode/utf8"

	"github.com/roberson-io/cchef/core"
)

// dishText reads a dish as CyberChef reads one: as UTF-8 where that works,
// and otherwise one character per byte. Operations that walk their input as
// text must use this rather than the raw string, or a byte that is not valid
// UTF-8 becomes U+FFFD and its value is lost — 0xFF has to arrive as U+00FF
// for "To Charcode" to report 255 and for "To HTML Entity" to write &yuml;.
func dishText(in *core.Dish) string {
	return bytesAsText(in.Bytes())
}

// bytesAsText is dishText for bytes that are already in hand.
func bytesAsText(data []byte) string {
	if utf8.Valid(data) {
		return string(data)
	}
	var sb strings.Builder
	sb.Grow(len(data))
	for _, b := range data {
		sb.WriteRune(rune(b))
	}
	return sb.String()
}

// textAsBytes writes text back out as CyberChef writes it: one byte per
// character while every character fits in a byte, and the whole string as
// UTF-8 as soon as one does not. It is the inverse of dishText, and what
// decides whether charcode 255 leaves as the byte 0xFF or as its UTF-8 form.
func textAsBytes(s string) []byte {
	if s == "" {
		return nil
	}
	out := make([]byte, 0, len(s))
	for _, r := range s {
		if r > 0xFF {
			return []byte(s)
		}
		out = append(out, byte(r)) // #nosec G115 -- the loop above returns early for anything above 0xFF
	}
	return out
}
