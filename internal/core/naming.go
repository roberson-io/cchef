package core

import "strings"

// Kebab converts an operation name to a CLI subcommand name: lower-cased, with
// spaces and separators collapsed to single hyphens and other punctuation
// dropped (e.g. "To Base64" -> "to-base64", "Find / Replace" -> "find-replace").
func Kebab(name string) string {
	var sb strings.Builder
	prevHyphen := false
	for _, r := range strings.ToLower(name) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			sb.WriteRune(r)
			prevHyphen = false
		case r == ' ' || r == '-' || r == '_' || r == '/':
			if !prevHyphen && sb.Len() > 0 {
				sb.WriteByte('-')
				prevHyphen = true
			}
		}
	}
	return strings.TrimRight(sb.String(), "-")
}
