package cmd

// opAliases gives short, explicit aliases to the highest-traffic operations,
// keyed by display name. Aliases are deliberately curated (not generated) and
// kept few: clig.dev warns against cryptic or implicit abbreviations, so these
// cover only the common encode/decode pairs where a short form is genuinely
// useful. TestOpAliasesValid ensures they name real operations, are unique, and
// never shadow a canonical subcommand name.
var opAliases = map[string][]string{
	"To Base64":   {"b64e"},
	"From Base64": {"b64d"},
	"To Base32":   {"b32e"},
	"From Base32": {"b32d"},
	"To Hex":      {"hex"},
	"From Hex":    {"unhex"},
	"URL Encode":  {"urle"},
	"URL Decode":  {"urld"},
}
