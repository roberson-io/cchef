package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

func kebabRecipe(smart bool) core.Recipe {
	return core.Recipe{{Op: "To Kebab case", Args: []any{smart}}}
}

// To Kebab case: oracle-verified vectors (CyberChef wraps lodash's kebabCase).
func TestToKebabCaseFixtures(t *testing.T) {
	runCases(t, []opCase{
		{"empty", "", "", kebabRecipe(false)},
		{"camelCase", "fooBar", "foo-bar", kebabRecipe(false)},
		{"spaced words", "Hello World", "hello-world", kebabRecipe(false)},
		{"acronym run", "XMLHttpRequest", "xml-http-request", kebabRecipe(false)},
		{"digit boundary", "user123name", "user-123-name", kebabRecipe(false)},
		{"deburr accents", "déjà vu", "deja-vu", kebabRecipe(false)},
	})
}

// TestToKebabCaseSmart covers context-aware mode.
func TestToKebabCaseSmart(t *testing.T) {
	runCases(t, []opCase{
		{"transform identifiers, keep strings", `const fooBar = 1;`, `const foo-bar = 1;`, kebabRecipe(true)},
	})
}
