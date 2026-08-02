package opsutil

import (
	"regexp"
	"strconv"
	"strings"
)

// reEscapedChars matches the backslash escape sequences recognised by
// ParseEscapedChars.
var reEscapedChars = regexp.MustCompile(`\\([abfnrtv'"]|[0-3][0-7]{2}|[0-7]{1,2}|x[0-9a-fA-F]{2}|u[0-9a-fA-F]{4}|u\{[0-9a-fA-F]{1,6}\}|\\)`)

// ParseEscapedChars converts recognised backslash escape sequences into their
// literal characters. Unrecognised sequences (e.g. "\d") are left intact.
func ParseEscapedChars(s string) string {
	return reEscapedChars.ReplaceAllStringFunc(s, func(m string) string {
		a := m[1:] // drop the leading backslash
		switch a[0] {
		case '\\':
			return "\\"
		case 'a':
			return "\x07"
		case 'b':
			return "\b"
		case 't':
			return "\t"
		case 'n':
			return "\n"
		case 'v':
			return "\v"
		case 'f':
			return "\f"
		case 'r':
			return "\r"
		case '"':
			return "\""
		case '\'':
			return "'"
		case 'x':
			v, _ := strconv.ParseInt(a[1:], 16, 32)
			return string(rune(v))
		case 'u':
			if a[1] == '{' {
				v, _ := strconv.ParseInt(a[2:len(a)-1], 16, 32)
				return string(rune(v))
			}
			v, _ := strconv.ParseInt(a[1:], 16, 32)
			return string(rune(v))
		default: // octal 0-7
			v, _ := strconv.ParseInt(a, 8, 32)
			return string(rune(v))
		}
	})
}

// ExpandAlphRange expands an alphabet specification such as "A-Za-z0-9+/=" into
// its full character sequence. A backslash escapes a literal dash ("\\-").
func ExpandAlphRange(alph string) string {
	r := []rune(alph)
	var out strings.Builder
	for i := 0; i < len(r); i++ {
		switch {
		case i < len(r)-2 && r[i+1] == '-' && r[i] != '\\':
			for c := r[i]; c <= r[i+2]; c++ {
				out.WriteRune(c)
			}
			i += 2
		case i < len(r)-1 && r[i] == '\\' && r[i+1] == '-':
			out.WriteRune('-')
			i++
		default:
			out.WriteRune(r[i])
		}
	}
	return out.String()
}

// EscapeWhitespace maps control characters 0x09–0x10 into the Private Use Area
// (0xE000 + code) so they render, matching Utils.EscapeWhitespace.
func EscapeWhitespace(s string) string {
	var sb strings.Builder
	for _, r := range s {
		if r >= 0x09 && r <= 0x10 {
			sb.WriteRune(0xe000 + r)
		} else {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// BytesAsLatin1 maps each byte to its code point, matching CyberChef's
// Utils.byteArrayToChars. Unlike BytesAsText it never tries UTF-8 first.
func BytesAsLatin1(b []byte) string {
	var sb strings.Builder
	for _, by := range b {
		sb.WriteRune(rune(by))
	}
	return sb.String()
}
