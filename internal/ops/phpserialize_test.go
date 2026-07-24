package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

func phpSerRecipe() core.Recipe {
	return core.Recipe{{Op: "PHP Serialize", Args: []any{}}}
}

// Fixtures transcribed from ../CyberChef/tests/operations/tests/PHPSerialize.mjs.
// The input is JSON; strings use JS UTF-16 code-unit length.
func TestPHPSerializeFixtures(t *testing.T) {
	runCases(t, []opCase{
		{"empty array", "[]", "a:0:{}", phpSerRecipe()},
		{"empty object", "{}", "a:0:{}", phpSerRecipe()},
		{"null", "null", "N;", phpSerRecipe()},
		{"integer", "10", "i:10;", phpSerRecipe()},
		{"float", "14.523", "d:14.523;", phpSerRecipe()},
		{"boolean array", "[true, false]", "a:2:{i:0;b:1;i:1;b:0;}", phpSerRecipe()},
		{"string", `"Test string to serialize"`, `s:24:"Test string to serialize";`, phpSerRecipe()},
		{"object with nested array", `{"a":10,"b":[1,2]}`, `a:2:{s:1:"a";i:10;s:1:"b";a:2:{i:0;i:1;i:1;i:2;}}`, phpSerRecipe()},
		{"object of strings and int", `{"name":"Bob","age":30}`, `a:2:{s:4:"name";s:3:"Bob";s:3:"age";i:30;}`, phpSerRecipe()},
	})
}

// TestPHPSerializeErrorsAndHelpers covers the invalid-JSON path and the defensive
// helper branches (an unhandled value type; parseInt on a non-numeric prefix).
func TestPHPSerializeErrorsAndHelpers(t *testing.T) {
	if _, err := runOp(t, "PHP Serialize", "{invalid"); err == nil {
		t.Error("expected error for invalid JSON")
	}
	if got := phpSerialize(int(42)); got != "" {
		t.Errorf("phpSerialize(unhandled type) = %q, want empty", got)
	}
	if p := phpParseIntPrefix("+abc"); !mathIsNaN(p) {
		t.Errorf("phpParseIntPrefix(%q) = %v, want NaN", "+abc", p)
	}
	// A digit prefix that overflows float64 range -> ParseFloat errors -> NaN.
	if p := phpParseIntPrefix("1" + strings.Repeat("0", 400)); !mathIsNaN(p) {
		t.Errorf("phpParseIntPrefix(huge) = %v, want NaN", p)
	}
}

func mathIsNaN(f float64) bool { return f != f }
