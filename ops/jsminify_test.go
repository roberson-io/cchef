package ops

// JavaScript Minify wraps esbuild's minifier. CyberChef wraps terser, so the
// output is NOT byte-identical (different mangler and compression passes); these
// golden cases pin cchef's own esbuild-backed output.

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
)

func jsMinifyRecipe() core.Recipe {
	return core.Recipe{{Op: "JavaScript Minify"}}
}

func TestJavaScriptMinify(t *testing.T) {
	runCases(t, []opCase{
		{
			"Minify: function", "function add(a, b) { return a + b; }  var x = add(1, 2);",
			"function add(n,r){return n+r}var x=add(1,2);", jsMinifyRecipe(),
		},
		{
			"Minify: arrow", "const f = (a) => { return a * 2; };",
			"const f=n=>n*2;", jsMinifyRecipe(),
		},
		{
			"Minify: if/else to ternary", "if (x) { y(); } else { z(); }",
			"x?y():z();", jsMinifyRecipe(),
		},
		{"Minify: no trailing newline", "var x = 1;", "var x=1;", jsMinifyRecipe()},
	})
}

// Invalid JavaScript surfaces an "Error minifying JavaScript." error.
func TestJavaScriptMinifyError(t *testing.T) {
	_, err := runOp(t, "JavaScript Minify", "function (")
	if err == nil || !strings.Contains(err.Error(), "Error minifying JavaScript.") {
		t.Fatalf("got %v, want an 'Error minifying JavaScript.' error", err)
	}
}
