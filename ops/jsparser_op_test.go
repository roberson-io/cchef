package ops

// Operation-level tests for JavaScript Parser (the AST fidelity is covered by
// TestJSParserAST against the esprima golden corpus).

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
)

func jsParserRecipe() core.Recipe {
	return core.Recipe{{Op: "JavaScript Parser", Args: []any{false, false, false, false, false}}}
}

func TestJavaScriptParserOp(t *testing.T) {
	want := "{\n  \"type\": \"Program\",\n  \"body\": [\n    {\n      \"type\": \"ExpressionStatement\",\n      \"expression\": {\n        \"type\": \"BinaryExpression\",\n        \"operator\": \"+\",\n        \"left\": {\n          \"type\": \"Literal\",\n          \"value\": 1,\n          \"raw\": \"1\"\n        },\n        \"right\": {\n          \"type\": \"Literal\",\n          \"value\": 2,\n          \"raw\": \"2\"\n        }\n      }\n    }\n  ],\n  \"sourceType\": \"script\"\n}"
	runCases(t, []opCase{
		{"JavaScript Parser: 1+2", "1+2", want, jsParserRecipe()},
	})
}

// A malformed program surfaces esprima's error text.
func TestJavaScriptParserError(t *testing.T) {
	_, err := runOp(t, "JavaScript Parser", "var 1 = 2;", false, false, false, false, false)
	if err == nil || !strings.HasPrefix(err.Error(), "Line 1:") {
		t.Fatalf("got %v, want a 'Line 1:' parse error", err)
	}
}

// Each not-yet-ported output option is rejected rather than silently ignored.
func TestJavaScriptParserOptionsRejected(t *testing.T) {
	for i := range 5 {
		args := make([]any, 5)
		for j := range args {
			args[j] = j == i
		}
		_, err := runOp(t, "JavaScript Parser", "x", args...)
		if err == nil || !strings.Contains(err.Error(), "not yet supported") {
			t.Fatalf("option %d: got %v, want 'not yet supported'", i, err)
		}
	}
}
