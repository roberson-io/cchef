package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

func phpDesRecipe(validJSON bool) core.Recipe {
	return core.Recipe{{Op: "PHP Deserialize", Args: []any{validJSON}}}
}

// Fixtures transcribed from CyberChef's tests/operations/tests/PHP.mjs. "Output
// valid JSON" quotes integer keys ("0":) vs leaving them bare (0:).
func TestPHPDeserializeFixtures(t *testing.T) {
	arr := `a:2:{s:1:"a";i:10;i:0;a:1:{s:2:"ab";b:1;}}`
	runCases(t, []opCase{
		{"empty array", "a:0:{}", "{}", phpDesRecipe(true)},
		{"integer", "i:10;", "10", phpDesRecipe(true)},
		{"string", `s:17:"PHP Serialization";`, `"PHP Serialization"`, phpDesRecipe(true)},
		{"array (JSON)", arr, `{"a": 10,"0": {"ab": true}}`, phpDesRecipe(true)},
		{"array (non-JSON)", arr, `{"a": 10,0: {"ab": true}}`, phpDesRecipe(false)},
		{"null", "N;", "null", phpDesRecipe(true)},
		{"float", "d:3.14;", "3.14", phpDesRecipe(true)},
		{"empty string", `s:0:"";`, `""`, phpDesRecipe(true)},
		{"boolean true (top level)", "b:1;", "true", phpDesRecipe(true)},
		{"boolean false in object", `a:1:{s:3:"key";b:0;}`, `{"key": false}`, phpDesRecipe(true)},
		{"integer keys quoted (JSON)", `a:2:{i:5;s:1:"a";i:10;s:1:"b";}`, `{"5": "a","10": "b"}`, phpDesRecipe(true)},
		{"integer keys bare (non-JSON)", `a:2:{i:5;s:1:"a";i:10;s:1:"b";}`, `{5: "a",10: "b"}`, phpDesRecipe(false)},
	})
}

// TestPHPDeserializeErrors covers the parser's error and defensive branches with
// malformed input (verbatim CyberChef error text where checked).
func TestPHPDeserializeErrors(t *testing.T) {
	for _, in := range []string{
		"x:1;",           // unknown type
		"i-5;",           // missing ':'
		"i:10",           // no terminating ';' -> end of input
		`s:5:"ab";`,      // declared length exceeds available input
		`a:1:{s:1:"a"`,   // truncated array
		"",               // empty input -> end of input
		"N",              // null missing terminating ';'
		"a:1:x",          // array missing '{'
		`s:2:"abcd";`,    // string missing closing '";'
		`a:1:{i:0;i:0;X`, // array missing closing '}'
		"a1",             // array missing ':' after 'a'
		"a:",             // array count reaches end of input
		"s1",             // string missing ':' after 's'
		"s:",             // string length reaches end of input
		"s:5:X",          // string missing opening '"'
	} {
		if _, err := runOp(t, "PHP Deserialize", in, true); err == nil {
			t.Errorf("expected error for %q", in)
		}
	}
	// Direct coverage of phpAllDigits edge cases unreachable via the parser.
	if phpAllDigits("") || phpAllDigits("12a") {
		t.Error("phpAllDigits should reject empty and mixed strings")
	}
	if !phpAllDigits("07") {
		t.Error("phpAllDigits should accept all-digit strings")
	}
}
