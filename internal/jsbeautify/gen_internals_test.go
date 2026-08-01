package jsbeautify

// Operation-level tests for JavaScript Beautify (AST-to-source fidelity is
// covered by TestJSBeautifyGolden against the escodegen golden corpus).

import (
	"testing"

	"github.com/roberson-io/cchef/internal/jsonval"
)

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
// (escodegen's `this[type]` miss); no AST jsparse.Parse produces reaches these.
func TestJSGenUnknownNode(t *testing.T) {
	g := &jsGen{indent: "\t", newline: "\n", space: " ", quotes: "auto", semicolons: true}
	g.buildHandlers()
	bogus := jsonval.Object{{K: "type", V: "Bogus"}}
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
	stmt := jsonval.Object{{K: "type", V: "ExpressionStatement"}}
	if got := g.maybeBlockSuffix(stmt, "x\n"); got != "x\n  " {
		t.Errorf("maybeBlockSuffix = %q, want %q", got, "x\n  ")
	}
}

// genCatchClause's binding-less branch: esprima v4 requires a catch parameter
// (no optional catch binding), so this is only reachable with a crafted node.
func TestJSGenCatchClauseNoParam(t *testing.T) {
	g := &jsGen{indent: "\t", newline: "\n", space: " ", quotes: "auto", semicolons: true}
	g.buildHandlers()
	catch := jsonval.Object{
		{K: "type", V: "CatchClause"},
		{K: "body", V: jsonval.Object{{K: "type", V: "BlockStatement"}, {K: "body", V: []any{}}}},
	}
	if got := g.genCatchClause(catch, sTFFF); got != "catch {\n}" {
		t.Errorf("genCatchClause(no param) = %q, want %q", got, "catch {\n}")
	}
}

// Semicolons=false drops the optional semicolon before a closing brace.
