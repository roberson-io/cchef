package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

func csbRecipe(indent string) core.Recipe {
	return core.Recipe{{Op: "CSS Beautify", Args: []any{indent}}}
}

// CSS Beautify has no CyberChef fixtures; these vectors are verified against the
// vkbeautify library directly (node -e "require('vkbeautify').css(...)"). The
// library marks boundaries at {, }, ; and comments, then indents each segment by
// nesting depth. It leaves a trailing newline (only leading newlines are trimmed)
// and, for unbalanced "}", reproduces JS's out-of-range array access as the
// literal string "undefined" — cchef preserves both quirks.
func TestCSSBeautifyFixtures(t *testing.T) {
	runCases(t, []opCase{
		{"empty", "", "", csbRecipe("  ")},
		{"declarations", "a{color:red;font:bold}", "a{\n  color:red;\n  font:bold\n}\n", csbRecipe("  ")},
		{"two rules", "a{color:red}b{color:blue}", "a{\n  color:red\n}\nb{\n  color:blue\n}\n", csbRecipe("  ")},
		{
			"tab indent, spaced input", "a { color : red ; margin : 0 ; }",
			"a {\n\t color : red ;\n\t margin : 0 ;\n}\n", csbRecipe("\t"),
		},
		{"leading comment", "/* header */ a { color: red; }", "/* header */\n a {\n   color: red;\n}\n", csbRecipe("  ")},
		{"nested at-rule", "@media screen { a { color: red; } }", "@media screen {\n   a {\n     color: red;\n  }\n}\n", csbRecipe("  ")},
		{"numeric-string indent -> 4 spaces", "a{color:red}", "a{\n    color:red\n}\n", csbRecipe("2")},
		{"three rules", ".x{a:1}.y{b:2}.z{c:3}", ".x{\n  a:1\n}\n.y{\n  b:2\n}\n.z{\n  c:3\n}\n", csbRecipe("  ")},
		{"unbalanced braces -> undefined (JS quirk)", "}}", "undefined}undefined}undefined", csbRecipe("  ")},
	})
}
