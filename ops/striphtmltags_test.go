package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

func shtRecipe(removeIndent, removeBreaks bool) core.Recipe {
	return core.Recipe{{Op: "Strip HTML tags", Args: []any{removeIndent, removeBreaks}}}
}

// Strip HTML tags has no CyberChef fixtures; these vectors are oracle-verified.
// CyberChef removes <...> tags recursively (Utils.stripHtmlTags), then optionally
// removes indentation and collapses excess line breaks.
func TestStripHTMLTagsFixtures(t *testing.T) {
	runCases(t, []opCase{
		{"empty", "", "", shtRecipe(true, true)},
		{"basic tags", "<p>Hello <b>World</b></p>", "Hello World", shtRecipe(true, true)},
		{"indentation removed", "<div>\n    <span>indented</span>\n</div>", "indented\n", shtRecipe(true, true)},
		{"inline tag", "a<b>c", "ac", shtRecipe(false, false)},
		{"recursive removal", "<<b>>text", ">text", shtRecipe(false, false)},
		{"collapse blank lines", "<p>one</p>\n\n\n<p>two</p>", "one\ntwo", shtRecipe(false, true)},
		{"remove indentation only", "line1\n    line2\n\tline3", "line1\nline2\nline3", shtRecipe(true, false)},
		{"plain text unchanged", "plain text", "plain text", shtRecipe(true, true)},
	})
}
