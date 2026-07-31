package ops

import (
	"regexp"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(ToUpperCase{})
	core.Register(ToLowerCase{})
}

// Scope regexes for To Upper case, matching the first word character within
// each scope. Ported from CyberChef ToUpperCase.mjs.
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
		return core.NewDish([]byte(strings.ToUpper(s)), core.TypeString), nil
	case "Word":
		re = reWord
	case "Sentence":
		re = reSentence
	case "Paragraph":
		re = reParagraph
	default:
		return core.NewDish([]byte(s), core.TypeString), nil
	}

	out := re.ReplaceAllStringFunc(s, strings.ToUpper)
	return core.NewDish([]byte(out), core.TypeString), nil
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
	return core.NewDish([]byte(strings.ToLower(dishText(in))), core.TypeString), nil
}
