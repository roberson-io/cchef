package opsutil

import (
	"strings"
	"unicode/utf8"
)

// BytesAsText reads bytes as CyberChef reads a dish: as UTF-8 where that
// works, and otherwise one character per byte, so a byte that is not valid
// UTF-8 keeps its value instead of becoming U+FFFD.
func BytesAsText(data []byte) string {
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

// TextAsBytes writes text back out as CyberChef writes it: one byte per
// character while every character fits in a byte, and the whole string as
// UTF-8 as soon as one does not. It is the inverse of BytesAsText, and what
// decides whether charcode 255 leaves as the byte 0xFF or as its UTF-8 form.
func TextAsBytes(s string) []byte {
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
