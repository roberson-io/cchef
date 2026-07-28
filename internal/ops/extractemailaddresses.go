package ops

import (
	"github.com/dlclark/regexp2"

	"github.com/roberson-io/cchef/internal/core"
)

// The shape of an email address. The part before the @ is either a run of
// dot-separated atoms or a quoted string; the part after is either a run of
// dot-separated labels or a bracketed IPv4 address. Everything from U+00A0
// upwards counts as an ordinary letter on both sides, which is what lets an
// internationalised address through, with the surrogate range left out so that a
// character outside the basic plane is not taken apart.
const (
	// The wide range: everything from U+00A0 up, minus the surrogates, so that a
	// character outside the basic plane is not taken apart into halves.
	emailWide = `\u00A0-\uD7FF\uE000-\uFFFF`

	emailAtomChar = "[" + emailWide + "a-z0-9!#$%&'*+/=?^_`{|}~-]"
	emailQuoted   = `"(?:[\x01-\x08\x0b\x0c\x0e-\x1f\x21\x23-\x5b\x5d-\x7f]` +
		`|\\[\x01-\x09\x0b\x0c\x0e-\x7f])*"`
	emailLocal = `(?:` + emailAtomChar + `+(?:\.` + emailAtomChar + `+)*|` + emailQuoted + `)`

	emailLabelEdge  = `[` + emailWide + `a-z0-9]`
	emailLabelInner = `[` + emailWide + `a-z0-9-]`
	emailLabel      = emailLabelEdge + `(?:` + emailLabelInner + `*` + emailLabelEdge + `)?`

	emailIPByte  = `(?:2(?:5[0-5]|[0-4][0-9])|1[0-9][0-9]|[1-9]?[0-9])`
	emailLiteral = `\[(?:` + emailIPByte + `\.){3}` + emailIPByte + `\]`

	emailHost    = `(?:(?:` + emailLabel + `\.)+` + emailLabel + `|` + emailLiteral + `)`
	emailPattern = emailLocal + `@` + emailHost
)

var emailRegexp = jsRegex(emailPattern, regexp2.IgnoreCase)

// ExtractEmailAddresses pulls the email addresses out of the input.
type ExtractEmailAddresses struct{}

// Meta returns the operation metadata.
func (ExtractEmailAddresses) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Extract email addresses",
		Module:      "Regex",
		Description: "Extracts all email addresses from the input.",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ExtractEmailAddresses) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Display total", Type: core.ArgBoolean, Value: false},
		{Name: "Sort", Type: core.ArgBoolean, Value: false},
		{Name: "Unique", Type: core.ArgBoolean, Value: false},
	}
}

// Run finds the addresses.
func (ExtractEmailAddresses) Run(in *core.Dish, args []any) (*core.Dish, error) {
	displayTotal, _ := args[0].(bool)
	sortResults, _ := args[1].(bool)
	unique, _ := args[2].(bool)

	var less func(a, b string) bool
	if sortResults {
		less = caseInsensitiveLess
	}

	found := extractSearch(in.String(), emailRegexp, nil, less, unique)
	return core.NewDish([]byte(extractResult(found, displayTotal)), core.TypeString), nil
}

func init() { core.Register(ExtractEmailAddresses{}) }
