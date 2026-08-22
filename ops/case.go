package ops

import (
	"regexp"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/opsutil"
)

// upperAll and lowerAll apply Unicode full case mapping, where one character
// may become several ("ß" upper-cases to "SS") and where the result depends on
// surrounding context (a Greek sigma takes its final form at the end of a
// word). strings.ToUpper and strings.ToLower implement only the simple
// one-rune-for-one-rune mapping and would leave those characters alone.
// JavaScript's toUpperCase and toLowerCase are full mappings, so this is what
// matching CyberChef requires.
//
// The language is explicitly undefined rather than a locale: case conversion
// here must not depend on where it is run, and locale rules would change the
// answer for Turkish dotless i among others.
var (
	upperAll = cases.Upper(language.Und)
	lowerAll = cases.Lower(language.Und)
)

func init() {
	core.Register(ToUpperCase{})
	core.Register(ToLowerCase{})
}

// Scope regexes for To Upper case, matching the first word character within
// each scope.
var (
	reWord      = regexp.MustCompile(`\b\w`)
	reSentence  = regexp.MustCompile(`(?:\.|^)\s*\b\w`)
	reParagraph = regexp.MustCompile(`(?:\n|^)\s*\b\w`)
)

// ToUpperCase converts the input to upper case, optionally limited in scope.
type ToUpperCase struct{}

// Meta returns the operation metadata.
func (ToUpperCase) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "To Upper case",
		Module:      "Default",
		Description: "Converts the input to upper case, optionally limiting scope to the first character of each word, sentence or paragraph.",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ToUpperCase) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Scope", Type: core.ArgOption, Value: []string{"All", "Word", "Sentence", "Paragraph"}},
	}
}

// Run upper-cases the input within the chosen scope.
func (ToUpperCase) Run(in *core.Dish, args []any) (*core.Dish, error) {
	scope := args[0].(string)
	s := dishText(in)

	var re *regexp.Regexp
	switch scope {
	case "All":
		return core.NewDish(opsutil.TextAsBytes(upperAll.String(s)), core.TypeString), nil
	case "Word":
		re = reWord
	case "Sentence":
		re = reSentence
	case "Paragraph":
		re = reParagraph
	default:
		return core.NewDish(opsutil.TextAsBytes(s), core.TypeString), nil
	}

	out := re.ReplaceAllStringFunc(s, upperAll.String)
	return core.NewDish(opsutil.TextAsBytes(out), core.TypeString), nil
}

// ToLowerCase converts every character in the input to lower case.
type ToLowerCase struct{}

// Meta returns the operation metadata.
func (ToLowerCase) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "To Lower case",
		Module:      "Default",
		Description: "Converts every character in the input to lower case.",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ToLowerCase) Args() []core.ArgDef { return nil }

// Run lower-cases the input.
func (ToLowerCase) Run(in *core.Dish, args []any) (*core.Dish, error) {
	return core.NewDish(opsutil.TextAsBytes(lowerAll.String(dishText(in))), core.TypeString), nil
}
