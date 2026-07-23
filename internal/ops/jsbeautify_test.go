package ops

// Operation-level tests for JavaScript Beautify (AST-to-source fidelity is
// covered by TestJSBeautifyGolden against the escodegen golden corpus).

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
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
func TestJSGenHelpers(t *testing.T) {
	if got := lastRune(""); got != 0 {
		t.Errorf("lastRune(\"\") = %q, want 0", got)
	}
	if got := firstRune(""); got != 0 {
		t.Errorf("firstRune(\"\") = %q, want 0", got)
	}
	g := &jsGen{indent: "\t", newline: "\n", space: " ", quotes: "auto", semicolons: true}
	if got := g.join("a", ""); got != "a" {
		t.Errorf("join(\"a\", \"\") = %q, want \"a\"", got)
	}
	if got := g.join("", "b"); got != "b" {
		t.Errorf("join(\"\", \"b\") = %q, want \"b\"", got)
	}
	// jsEscapeDisallowed only receives line terminators or backslash from
	// escapeStringBody; the default (return "") is a faithful-port guard.
	if got := jsEscapeDisallowed(0x41); got != "" {
		t.Errorf("jsEscapeDisallowed('A') = %q, want \"\"", got)
	}
	if got := needsNumberDot("abc"); got {
		t.Errorf("needsNumberDot(\"abc\") = true, want false")
	}
	if got := needsNumberDot("5"); !got {
		t.Errorf("needsNumberDot(\"5\") = false, want true")
	}
	// isPrefixedBy with fragment exactly equal to the keyword (ok(0) path).
	if isClassPrefixed("class") {
		t.Errorf("isClassPrefixed(\"class\") = true, want false")
	}
	// isAsyncPrefixed's whitespace-skip loop (escodegen only emits one space).
	if !isAsyncPrefixed("async  function(){}") {
		t.Errorf("isAsyncPrefixed(\"async  function(){}\") = false, want true")
	}
}

// generateExpression / generateStatement return "" for unknown node types
// (escodegen's `this[type]` miss); no AST jsParse produces reaches these.
func TestJSGenUnknownNode(t *testing.T) {
	g := &jsGen{indent: "\t", newline: "\n", space: " ", quotes: "auto", semicolons: true}
	g.buildHandlers()
	bogus := jsObject{{k: "type", v: "Bogus"}}
	if got := g.generateExpression(bogus, 0, 0); got != "" {
		t.Errorf("generateExpression(Bogus) = %q, want \"\"", got)
	}
	if got := g.generateStatement(bogus, 0); got != "" {
		t.Errorf("generateStatement(Bogus) = %q, want \"\"", got)
	}
}

// maybeBlockSuffix's line-terminator branch fires when the block fragment
// already ends with a newline (escodegen reaches it via trailing comments/blank
// lines, which this port does not emit).
func TestJSGenMaybeBlockSuffixEndsWithNewline(t *testing.T) {
	g := &jsGen{indent: "\t", base: "  ", newline: "\n", space: " ", quotes: "auto", semicolons: true}
	stmt := jsObject{{k: "type", v: "ExpressionStatement"}}
	if got := g.maybeBlockSuffix(stmt, "x\n"); got != "x\n  " {
		t.Errorf("maybeBlockSuffix = %q, want %q", got, "x\n  ")
	}
}

// genCatchClause's binding-less branch: esprima v4 requires a catch parameter
// (no optional catch binding), so this is only reachable with a crafted node.
func TestJSGenCatchClauseNoParam(t *testing.T) {
	g := &jsGen{indent: "\t", newline: "\n", space: " ", quotes: "auto", semicolons: true}
	g.buildHandlers()
	catch := jsObject{
		{k: "type", v: "CatchClause"},
		{k: "body", v: jsObject{{k: "type", v: "BlockStatement"}, {k: "body", v: []any{}}}},
	}
	if got := g.genCatchClause(catch, sTFFF); got != "catch {\n}" {
		t.Errorf("genCatchClause(no param) = %q, want %q", got, "catch {\n}")
	}
}

// Semicolons=false drops the optional semicolon before a closing brace.
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
