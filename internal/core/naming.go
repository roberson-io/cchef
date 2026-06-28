package core

import "strings"

// Kebab converts an operation name to a CLI subcommand name: lower-cased with
// spaces replaced by hyphens (e.g. "To Base64" -> "to-base64").
func Kebab(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, " ", "-"))
}
