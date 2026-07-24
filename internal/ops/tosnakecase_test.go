package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

func snakeRecipe(smart bool) core.Recipe {
	return core.Recipe{{Op: "To Snake case", Args: []any{smart}}}
}

// To Snake case has no CyberChef fixtures; these vectors are verified against the
// CyberChef-server oracle. CyberChef wraps lodash's snakeCase; cchef reimplements
// lodash's word splitter (deburr + words), so output matches byte-for-byte across
// the BMP surface. "Attempt to be context aware" (smart) only transforms
// identifier-like tokens, leaving quoted strings untouched (replaceVariableNames).
func TestToSnakeCaseFixtures(t *testing.T) {
	runCases(t, []opCase{
		{"empty", "", "", snakeRecipe(false)},
		{"camelCase", "fooBar", "foo_bar", snakeRecipe(false)},
		{"spaced words", "Hello World", "hello_world", snakeRecipe(false)},
		{"acronym run", "XMLHttpRequest", "xml_http_request", snakeRecipe(false)},
		{"dashes and edges", "--foo-bar--", "foo_bar", snakeRecipe(false)},
		{"digit boundary", "user123name", "user_123_name", snakeRecipe(false)},
		{"deburr accents", "déjà vu", "deja_vu", snakeRecipe(false)},
		{"already snake", "already_snake", "already_snake", snakeRecipe(false)},
	})
}

// TestToSnakeCaseSmart covers the context-aware mode.
func TestToSnakeCaseSmart(t *testing.T) {
	runCases(t, []opCase{
		{
			"transform identifiers, keep strings",
			`var fooBar = "some String Value";`,
			`var foo_bar = "some String Value";`,
			snakeRecipe(true),
		},
	})
}
