package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

func jmRecipe() core.Recipe {
	return core.Recipe{{Op: "JSON Minify", Args: []any{}}}
}

// Fixtures transcribed from ../CyberChef/tests/operations/tests/JSONMinify.mjs.
// CyberChef's JSON Minify is vkbeautify.jsonmin = JSON.stringify(JSON.parse(text),
// null, 0); cchef reproduces it over the shared order-preserving JSON serialiser
// (jsonvalue.go). Strict JSON only (unlike JSON Beautify, which uses JSON5).
func TestJSONMinifyFixtures(t *testing.T) {
	runCases(t, []opCase{
		{"JSON Minify: ''", "", "", jmRecipe()},
		{"JSON Minify: number", "42", "42", jmRecipe()},
		{"JSON Minify: float", "4.2", "4.2", jmRecipe()},
		{"JSON Minify: string", `"string"`, `"string"`, jmRecipe()},
		{"JSON Minify: boolean", "false", "false", jmRecipe()},
		{"JSON Minify: emptyList", "[\n \n  \t]", "[]", jmRecipe()},
		{"JSON Minify: list", "[2,\n  \t1]", "[2,1]", jmRecipe()},
		{
			"JSON Minify: object", "{\n \"second\": 2,\n \"first\": 3\n}",
			`{"second":2,"first":3}`, jmRecipe(),
		},
		{
			"JSON Minify: tab, nested",
			"[\n\t2,\n\t{\n\t\t\"second\": 2,\n\t\t\"first\": 3,\n\t\t\"beginning\": {\n\t\t\t\"j\": \"3\",\n\t\t\t\"i\": [\n\t\t\t\t2,\n\t\t\t\t3,\n\t\t\t\tfalse\n\t\t\t]\n\t\t}\n\t},\n\t1,\n\t2,\n\t3\n]",
			`[2,{"second":2,"first":3,"beginning":{"j":"3","i":[2,3,false]}},1,2,3]`,
			jmRecipe(),
		},
	})
}

// TestJSONMinifyErrors covers invalid JSON input.
func TestJSONMinifyErrors(t *testing.T) {
	for _, in := range []string{"{bad}", "[1,2", `{"a":}`} {
		if _, err := runOp(t, "JSON Minify", in); err == nil {
			t.Errorf("expected error for %q", in)
		}
	}
}
