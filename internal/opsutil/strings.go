// Package opsutil holds helpers shared by several operations and engines that
// belong to no one domain.
package opsutil

import "strings"

// SplitTopLevel splits s on sep, ignoring separators inside brackets, parentheses
// or quotes. Selector and path syntaxes both nest that way, so a naive split
// would cut a bracketed subexpression in half.
func SplitTopLevel(s string, sep byte) []string {
	var out []string
	depth := 0
	var quote byte
	start := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '[' || c == '(':
			depth++
		case c == ']' || c == ')':
			depth--
		case c == sep && depth == 0:
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}

// htmlEscaper is CyberChef's Utils.escapeHtml replacement map. It differs
// from the standard library's: backticks are escaped, forward slashes are
// not, and a null byte becomes U+E000 so it stays visible when rendered.
var htmlEscaper = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
	`"`, "&quot;",
	"'", "&#x27;",
	"`", "&#x60;",
	"\x00", "",
)

// EscapeHTML escapes HTML-significant characters the way CyberChef's
// Utils.escapeHtml does.
func EscapeHTML(s string) string { return htmlEscaper.Replace(s) }
