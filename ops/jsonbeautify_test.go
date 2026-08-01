package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// jbRecipe builds a JSON Beautify recipe. Args are
// [indentString, sortObjectKeys, formatted]; the third is inert in cchef (it only
// drives CyberChef's browser HTML tree view, which the CLI does not produce).
func jbRecipe(args ...any) core.Recipe {
	return core.Recipe{{Op: "JSON Beautify", Args: args}}
}

// Fixtures transcribed from ../CyberChef/tests/operations/tests/JSONBeautify.mjs.
// CyberChef parses with JSON5 (lenient) and re-emits with JSON.stringify(_, null,
// indent); cchef reproduces run()'s plain-string output. The upstream fixtures
// pipe the string/object cases through "HTML To Text" to reverse the browser
// present() escaping — cchef emits the plain text directly, so the expected
// values here are the post-HTML-To-Text results.
func TestJSONBeautifyFixtures(t *testing.T) {
	nested := `[2,{"second":2,"first":3,"beginning":{"j":"3","i":[2,3,false]}},1,2,3]`
	runCases(t, []opCase{
		{"JSON Beautify: space, ''", "", "", jbRecipe(" ", false, false)},
		{"JSON Beautify: space, number", "42", "42", jbRecipe(" ", false, false)},
		{"JSON Beautify: space, string", `"string"`, `"string"`, jbRecipe(" ", false, false)},
		{"JSON Beautify: space, boolean", "false", "false", jbRecipe(" ", false, false)},
		{"JSON Beautify: space, emptyList", "[]", "[]", jbRecipe(" ", false, false)},
		{"JSON Beautify: space, list", "[2,1]", "[\n 2,\n 1\n]", jbRecipe(" ", false, false)},
		{"JSON Beautify: tab, list", "[2,1]", "[\n\t2,\n\t1\n]", jbRecipe("\t", false, false)},
		{
			"JSON Beautify: space, object", `{"second":2,"first":3}`,
			"{\n \"second\": 2,\n \"first\": 3\n}", jbRecipe(" ", false, false),
		},
		{
			"JSON Beautify: tab, nested", nested,
			"[\n\t2,\n\t{\n\t\t\"second\": 2,\n\t\t\"first\": 3,\n\t\t\"beginning\": {\n\t\t\t\"j\": \"3\",\n\t\t\t\"i\": [\n\t\t\t\t2,\n\t\t\t\t3,\n\t\t\t\tfalse\n\t\t\t]\n\t\t}\n\t},\n\t1,\n\t2,\n\t3\n]",
			jbRecipe("\t", false, false),
		},
		{
			"JSON Beautify: tab, nested, sorted", nested,
			"[\n\t2,\n\t{\n\t\t\"beginning\": {\n\t\t\t\"i\": [\n\t\t\t\t2,\n\t\t\t\t3,\n\t\t\t\tfalse\n\t\t\t],\n\t\t\t\"j\": \"3\"\n\t\t},\n\t\t\"first\": 3,\n\t\t\"second\": 2\n\t},\n\t1,\n\t2,\n\t3\n]",
			jbRecipe("\t", true, false),
		},
	})
}

// TestJSONBeautifyJSON5 exercises the from-scratch JSON5 parser's lenient
// features (unquoted/single-quoted keys, trailing commas, comments, hex numbers,
// non-finite numbers, leading/trailing decimal points, signs) and the ES integer-
// key ordering. All expected outputs are verified against the CyberChef-server
// oracle.
func TestJSONBeautifyJSON5(t *testing.T) {
	runCases(t, []opCase{
		{"unquoted keys + trailing comma", "{a:1,b:2,}", "{\n \"a\": 1,\n \"b\": 2\n}", jbRecipe(" ", false, false)},
		{"single-quoted key", `{'x':1,"y":2}`, "{\n \"x\": 1,\n \"y\": 2\n}", jbRecipe(" ", false, false)},
		{"hex numbers", `{"a":0xFF,"b":-0x10}`, "{\n \"a\": 255,\n \"b\": -16\n}", jbRecipe(" ", false, false)},
		{"non-finite -> null", `{"a":Infinity,"b":-Infinity,"c":NaN}`, "{\n \"a\": null,\n \"b\": null,\n \"c\": null\n}", jbRecipe(" ", false, false)},
		{"number forms", `{"a":.5,"b":5.,"c":+3,"d":1e3,"e":-2.5e-2}`, "{\n \"a\": 0.5,\n \"b\": 5,\n \"c\": 3,\n \"d\": 1000,\n \"e\": -0.025\n}", jbRecipe(" ", false, false)},
		{"array trailing comma", "[1,2,3,]", "[\n 1,\n 2,\n 3\n]", jbRecipe(" ", false, false)},
		{"comments", "{/*c*/\"a\":1,//x\n\"b\":2}", "{\n \"a\": 1,\n \"b\": 2\n}", jbRecipe(" ", false, false)},
		{"integer keys first, ascending", `{"10":1,"2":2,"1":3,"x":4}`, "{\n \"1\": 3,\n \"2\": 2,\n \"10\": 1,\n \"x\": 4\n}", jbRecipe(" ", false, false)},
		{"duplicate key: last wins, first position", `{"d":1,"d":2}`, "{\n \"d\": 2\n}", jbRecipe(" ", false, false)},
	})
}

// TestJSONBeautifyEscapes covers the string-escape branches (\t \n \r \b \f \v,
// \x, \u with surrogate pairs, identity escapes and line continuations),
// oracle-verified.
func TestJSONBeautifyEscapes(t *testing.T) {
	runCases(t, []opCase{
		{"named escapes", `{"s":"a\tb\nc\rd"}`, "{\n \"s\": \"a\\tb\\nc\\rd\"\n}", jbRecipe(" ", false, false)},
		{"control escapes", `{"c":"\b\f\v"}`, "{\n \"c\": \"\\b\\f\\u000b\"\n}", jbRecipe(" ", false, false)},
		{"hex escape", `{"h":"\x41\x42"}`, "{\n \"h\": \"AB\"\n}", jbRecipe(" ", false, false)},
		{"unicode escape", `{"u":"\u0041"}`, "{\n \"u\": \"A\"\n}", jbRecipe(" ", false, false)},
		{"unicode surrogate pair", `{"e":"\uD83D\uDE00"}`, "{\n \"e\": \"😀\"\n}", jbRecipe(" ", false, false)},
		{"null escape", `{"z":"\0"}`, "{\n \"z\": \"\\u0000\"\n}", jbRecipe(" ", false, false)},
		{"identity escapes", `{"i":"\q\Z\/"}`, "{\n \"i\": \"qZ/\"\n}", jbRecipe(" ", false, false)},
		{"LF line continuation", "{\"s\":\"a\\\nb\"}", "{\n \"s\": \"ab\"\n}", jbRecipe(" ", false, false)},
		{"CRLF line continuation", "{\"s\":\"a\\\r\nb\"}", "{\n \"s\": \"ab\"\n}", jbRecipe(" ", false, false)},
		{"U+2028 line continuation", "{\"s\":\"a\\\u2028b\"}", "{\n \"s\": \"ab\"\n}", jbRecipe(" ", false, false)},
	})
}

// TestJSONBeautifyMoreNumbers covers the true/null literals and the large-hex
// (big.Int -> double) path, oracle-verified.
func TestJSONBeautifyMoreNumbers(t *testing.T) {
	runCases(t, []opCase{
		{"true literal", "true", "true", jbRecipe(" ", false, false)},
		{"null literal", "null", "null", jbRecipe(" ", false, false)},
		{"hex overflowing uint64", `{"a":0x1FFFFFFFFFFFFFFFFFF}`, "{\n \"a\": 9.44473296573929e+21\n}", jbRecipe(" ", false, false)},
	})
}

// TestJSONBeautifyEmptyAndScalars covers the empty-input short-circuit, empty
// containers, and a custom indent unit.
func TestJSONBeautifyEmptyAndScalars(t *testing.T) {
	runCases(t, []opCase{
		{"empty input", "", "", jbRecipe(" ", false, false)},
		{"empty object", "{}", "{}", jbRecipe(" ", false, false)},
		{"empty array", "[]", "[]", jbRecipe(" ", false, false)},
		{"custom indent unit", `{"a":[1]}`, "{\n>>\"a\": [\n>>>>1\n>>]\n}", jbRecipe(">>", false, false)},
		{"default indent (4 spaces)", "[1]", "[\n    1\n]", jbRecipe()},
	})
}

// TestJSONBeautifyErrors exercises the parser's error branches. CyberChef prefixes
// parse failures with "Unable to parse input as JSON."; the detail wording is not
// reproduced.
func TestJSONBeautifyErrors(t *testing.T) {
	bad := []string{
		"{bad",            // key with no ':'
		"{",               // unterminated object
		"[1,2",            // unterminated array
		`{"a":1 "b":2}`,   // missing comma
		"[1 2]",           // missing comma in array
		"nul",             // bad literal
		"{:1}",            // invalid identifier as key
		`"unterminated`,   // unterminated string
		"0x",              // empty hex
		"{}garbage",       // trailing content
		"   ",             // whitespace only
		"/* unterminated", // unterminated block comment -> EOF
		`{"a":`,           // value expected, got EOF
		`"\xZZ"`,          // invalid hex escape
		`"\uAB"`,          // truncated unicode escape (too few hex digits)
		`"\uD83D\uAB"`,    // valid high surrogate then truncated low surrogate
		`"\`,              // backslash then EOF (unterminated escape)
		"1.2.3",           // malformed number
		`{"a":1`,          // unterminated object after value
		"[1,@]",           // invalid value inside array
		`{"k`,             // unterminated quoted key
	}
	for _, in := range bad {
		if _, err := runOp(t, "JSON Beautify", in, " ", false, false); err == nil {
			t.Errorf("expected error for %q", in)
		} else if !strings.HasPrefix(err.Error(), "Unable to parse input as JSON.") {
			t.Errorf("input %q: error %q lacks expected prefix", in, err.Error())
		}
	}
}
