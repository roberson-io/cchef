package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

func sqmRecipe() core.Recipe {
	return core.Recipe{{Op: "SQL Minify", Args: []any{}}}
}

// SQL Minify has no CyberChef fixtures; these vectors are verified against the
// CyberChef-server oracle. vkbeautify.sqlmin collapses whitespace runs to a single
// space, then removes the whitespace before the FIRST "(" and the FIRST ")" only
// (the lib omits the /g flag on those two replaces) — cchef preserves that quirk.
// JS \s (which includes NBSP and Unicode spaces) is matched, not Go's narrower \s.
func TestSQLMinifyFixtures(t *testing.T) {
	runCases(t, []opCase{
		{"empty", "", "", sqmRecipe()},
		{"collapse whitespace", "SELECT  a,  b   FROM  t", "SELECT a, b FROM t", sqmRecipe()},
		{"first paren only", "SELECT * FROM t WHERE x IN (1, 2)  AND  y = (3)", "SELECT * FROM t WHERE x IN(1, 2) AND y = (3)", sqmRecipe()},
		{"first open and close paren", "a  (  b  )  (  c  )", "a( b) ( c )", sqmRecipe()},
		{"open then close", "foo   ( bar )", "foo( bar)", sqmRecipe()},
		{"newlines and tabs", "line1\n\t line2\n  line3", "line1 line2 line3", sqmRecipe()},
		{"trailing whitespace collapses", "trailing   ", "trailing ", sqmRecipe()},
		{"leading whitespace collapses", "  leading", " leading", sqmRecipe()},
		{"JS whitespace: NBSP", "a\u00a0b", "a b", sqmRecipe()},
		{"JS whitespace: ideographic + space", "x\u3000 y", "x y", sqmRecipe()},
	})
}
