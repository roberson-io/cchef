package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

func gcbRecipe() core.Recipe {
	return core.Recipe{{Op: "Generic Code Beautify", Args: []any{}}}
}

// Generic Code Beautify has no CyberChef fixtures; these vectors are oracle-
// verified (the port was differentially checked byte-for-byte against the
// CyberChef-server oracle across 30 varied inputs). Strings and comments are
// preserved verbatim; if/else/for get the library's characteristic spacing.
func TestGenericCodeBeautifyFixtures(t *testing.T) {
	runCases(t, []opCase{
		{"empty", "", "", gcbRecipe()},
		{"if block", "if(x){y=1;}", "if (x)  {\n    y = 1;\n}", gcbRecipe()},
		{"statements and comment", "a=1;b=2;// c\nd=3;", "a = 1;\nb = 2;\n// c\nd = 3;", gcbRecipe()},
		{"string preserved", `var s="a;b{}";x=1;`, "var s = \"a;b{}\";\nx = 1;", gcbRecipe()},
		{"if else", "if(a){b=1;}else{c=2;}", "if (a)  {\n    b = 1;\n} else {\n    c = 2;\n}", gcbRecipe()},
		{"for loop", "for(i=0;i<n;i++){f(i);}", "for(i = 0;\ni < n;\ni++) {\n    f(i);\n}", gcbRecipe()},
	})
}

// TestGCBIndentTrailingNewline directly exercises gcbIndent's end-of-input guard
// for a trailing newline (unreachable through Run, which strips trailing
// whitespace before indenting, but a faithful port of the library's guard).
func TestGCBIndentTrailingNewline(t *testing.T) {
	if got := gcbIndent("a\n"); got != "a\n" {
		t.Errorf("gcbIndent(%q) = %q, want unchanged", "a\n", got)
	}
}
