package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

func camelRecipe(smart bool) core.Recipe {
	return core.Recipe{{Op: "To Camel case", Args: []any{smart}}}
}

// To Camel case: oracle-verified vectors (CyberChef wraps lodash's camelCase).
func TestToCamelCaseFixtures(t *testing.T) {
	runCases(t, []opCase{
		{"empty", "", "", camelRecipe(false)},
		{"snake to camel", "foo_bar", "fooBar", camelRecipe(false)},
		{"spaced words", "Hello World", "helloWorld", camelRecipe(false)},
		{"acronym run", "XMLHttpRequest", "xmlHttpRequest", camelRecipe(false)},
		{"dashes and edges", "--foo-bar--", "fooBar", camelRecipe(false)},
		{"digit boundary", "user123name", "user123Name", camelRecipe(false)},
		{"mixed acronym+digit", "IPv4 address", "iPv4Address", camelRecipe(false)},
		{"deburr accents", "déjà vu", "dejaVu", camelRecipe(false)},
	})
}

// TestToCamelCaseSmart covers context-aware mode.
func TestToCamelCaseSmart(t *testing.T) {
	runCases(t, []opCase{
		{"transform identifiers, keep strings", `let my_var = "keep This";`, `let myVar = "keep This";`, camelRecipe(true)},
	})
}
