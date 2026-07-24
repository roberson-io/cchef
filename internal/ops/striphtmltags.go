package ops

import (
	"regexp"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(StripHTMLTags{})
}

// StripHTMLTags struct.
type StripHTMLTags struct{}

// Meta returns the operation metadata.
func (StripHTMLTags) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Strip HTML tags",
		Module:      "Default",
		Description: "Removes all HTML tags from a string.",
		InfoURL:     "https://wikipedia.org/wiki/HTML_element",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (StripHTMLTags) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Remove indentation", Type: core.ArgBoolean, Value: true},
		{Name: "Remove excess line breaks", Type: core.ArgBoolean, Value: true},
	}
}

var (
	reHTMLTag        = regexp.MustCompile(`<[^>]+>`)
	reHTMLIndent     = regexp.MustCompile(`\n[ \f\t]+`)
	reHTMLFirstLine  = regexp.MustCompile(`^\s*\n`)
	reHTMLBlankLines = regexp.MustCompile(`(\n\s*){2,}`)
)

// stripHTMLTags removes <...> tags, repeating until no match to avoid incomplete
// sanitisation (Utils.stripHtmlTags's recursiveRemove).
func stripHTMLTags(s string) string {
	for {
		n := reHTMLTag.ReplaceAllString(s, "")
		if n == s {
			return n
		}
		s = n
	}
}

// Run strips HTML tags and optionally cleans up whitespace.
func (StripHTMLTags) Run(in *core.Dish, args []any) (*core.Dish, error) {
	removeIndentation := args[0].(bool)
	removeLineBreaks := args[1].(bool)
	s := stripHTMLTags(in.String())
	if removeIndentation {
		s = reHTMLIndent.ReplaceAllString(s, "\n")
	}
	if removeLineBreaks {
		s = reHTMLFirstLine.ReplaceAllString(s, "")
		s = reHTMLBlankLines.ReplaceAllString(s, "\n")
	}
	return core.NewDish([]byte(s), core.TypeString), nil
}
