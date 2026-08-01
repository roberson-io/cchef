package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

func csmRecipe(preserve bool) core.Recipe {
	return core.Recipe{{Op: "CSS Minify", Args: []any{preserve}}}
}

// CSS Minify has no CyberChef fixtures; these vectors are verified against the
// CyberChef-server oracle. vkbeautify.cssmin optionally strips comments, then
// collapses whitespace runs to a single space (JS \s) and tightens whitespace
// after {, }, ;, /* and */.
func TestCSSMinifyFixtures(t *testing.T) {
	css := "a {\n  color: red;\n  /* x */\n}"
	runCases(t, []opCase{
		{"empty", "", "", csmRecipe(false)},
		{"strip comment", css, "a {color: red;}", csmRecipe(false)},
		{"preserve comment", css, "a {color: red;/*x */}", csmRecipe(true)},
		{"multiple rules", "a{color:red}  b{color:blue}", "a{color:red}b{color:blue}", csmRecipe(false)},
		{"leading comment + spacing", "/* top */ a { color : red ; }", " a {color : red ;}", csmRecipe(false)},
		{"selector list, trailing space kept", "a,b { color:red }", "a,b {color:red }", csmRecipe(false)},
		{"JS whitespace collapse (NBSP)", "a\u00a0{ color:red}", "a {color:red}", csmRecipe(false)},
	})
}
