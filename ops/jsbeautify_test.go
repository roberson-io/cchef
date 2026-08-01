package ops

// Operation-level tests for JavaScript Beautify (AST-to-source fidelity is
// covered by TestJSBeautifyGolden against the escodegen golden corpus).

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
)

func jsBeautifyRecipe(indent, quotes string, semis, comments bool) core.Recipe {
	return core.Recipe{{Op: "JavaScript Beautify", Args: []any{indent, quotes, semis, comments}}}
}

func TestJavaScriptBeautifyOp(t *testing.T) {
	runCases(t, []opCase{
		{
			"Beautify: default", "function f(a,b){return a+b}",
			"function f(a, b) {\n\treturn a + b;\n}", jsBeautifyRecipe("\t", "Auto", true, true),
		},
		{
			"Beautify: 2-space indent", "if(x){y()}",
			"if (x) {\n  y();\n}", jsBeautifyRecipe("  ", "Auto", true, true),
		},
		{
			"Beautify: escaped-tab indent", "if(x){y()}",
			"if (x) {\n\ty();\n}", jsBeautifyRecipe("\\t", "Auto", true, true),
		},
		{
			"Beautify: double quotes", "var s='a';",
			"var s = \"a\";", jsBeautifyRecipe("\t", "Double", true, true),
		},
		{
			"Beautify: single quotes", "var s=\"a\";",
			"var s = 'a';", jsBeautifyRecipe("\t", "Single", true, true),
		},
		{
			"Beautify: JSON array", "[1,2,3]",
			"[\n\t1,\n\t2,\n\t3\n];", jsBeautifyRecipe("\t", "Auto", true, true),
		},
	})
}

// An empty (or all-escape-to-empty) indent argument falls back to a tab.
func TestJavaScriptBeautifyEmptyIndent(t *testing.T) {
	got, err := runOp(t, "JavaScript Beautify", "if(x){y()}", "", "Auto", true, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := "if (x) {\n\ty();\n}"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// jsUnescapeIndent interprets the backslash escapes CyberChef accepts in the
// indent argument.
func TestJSUnescapeIndent(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"  ", "  "},
		{"\\t", "\t"},
		{"\\n", "\n"},
		{"\\r", "\r"},
		{"\\f", "\f"},
		{"\\v", "\v"},
		{"\\0", "\x00"},
		{"\\\\", "\\"},
		{"\\q", "\\q"},
		{"a\\", "a\\"},
	}
	for _, c := range cases {
		if got := jsUnescapeIndent(c.in); got != c.want {
			t.Errorf("jsUnescapeIndent(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// Direct unit tests for pure generator helpers whose remaining branches are not
// reachable through escodegen-shaped ASTs (empty inputs, faithful-port defaults).
func TestJavaScriptBeautifySemicolons(t *testing.T) {
	got, err := runOp(t, "JavaScript Beautify", "function f(){return 1}", "\t", "Auto", false, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := "function f() {\n\treturn 1\n}"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// A malformed program surfaces the wrapped parse error.
func TestJavaScriptBeautifyError(t *testing.T) {
	_, err := runOp(t, "JavaScript Beautify", "var 1 = 2;", "\t", "Auto", true, true)
	if err == nil || !strings.Contains(err.Error(), "Unable to parse JavaScript.") {
		t.Fatalf("got %v, want an 'Unable to parse JavaScript.' error", err)
	}
}
