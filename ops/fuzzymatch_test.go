package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// TestFuzzyMatch uses CyberChef's official browser fixture (tests/browser/02_ops.js):
// input "test input", search "tein", default weights, where the last <b> element is "in".
// The full output is derived from the algorithm: match indices [0,1,5,6] -> ranges
// [[0,2],[5,2]] -> "<span class=\"hl1\"><b>te</b>st <b>in</b></span>put".
func TestFuzzyMatch(t *testing.T) {
	runCases(t, []opCase{
		{
			"fixture", "test input", `<span class="hl1"><b>te</b>st <b>in</b></span>put`,
			core.Recipe{{Op: "Fuzzy Match", Args: []any{"tein", 15.0, 30.0, 30.0, 15.0, -5.0, -15.0, -1.0}}},
		},
	})
}

// The last <b> element must be "in", matching the fixture's "b:last-child" assertion.
func TestFuzzyMatchLastBold(t *testing.T) {
	out, err := runOp(t, "Fuzzy Match", "test input", "tein", 15.0, 30.0, 30.0, 15.0, -5.0, -15.0, -1.0)
	if err != nil {
		t.Fatal(err)
	}
	last := strings.LastIndex(out, "<b>")
	end := strings.Index(out[last:], "</b>")
	if got := out[last+3 : last+end]; got != "in" {
		t.Errorf("b:last-child = %q, want %q", got, "in")
	}
}

// TestFuzzyMatchBranches exercises the matcher's scoring/recursion branches
// (camelCase bonus, leading-letter penalty clamp, recursive backtracking, and
// the hl1/hl2 highlight alternation). Fuzzy Match has no working oracle, so these
// outputs are derived from the algorithm like the primary fixture above.
func TestFuzzyMatchBranches(t *testing.T) {
	def := func(s string) []any { return []any{s, 15.0, 30.0, 30.0, 15.0, -5.0, -15.0, -1.0} }
	runCases(t, []opCase{
		// Two matches (hl1 then hl2) with a camelCase boundary bonus on 'B'.
		{
			"camel + alternating highlight", "fooBar fooBar",
			`<span class="hl1"><b>f</b>oo<b>B</b></span>ar <span class="hl2"><b>f</b>oo<b>B</b></span>ar`,
			core.Recipe{{Op: "Fuzzy Match", Args: def("fb")}},
		},
		// A match starting at index 4 drives the leading-letter penalty past its cap.
		{
			"leading letter penalty clamp", "xxxxhello", `xxxx<span class="hl1"><b>hello</b></span>`,
			core.Recipe{{Op: "Fuzzy Match", Args: def("hello")}},
		},
		// Ambiguous positions force recursive backtracking to pick the best match.
		{
			"recursive backtracking", "a b ab",
			`<span class="hl1"><b>a</b> <b>b</b></span> <span class="hl2"><b>ab</b></span>`,
			core.Recipe{{Op: "Fuzzy Match", Args: def("ab")}},
		},
		// A highly ambiguous run exercises deep recursion.
		{
			"deep recursion", "aaaaaaaaaaaa",
			`<span class="hl1"><b>aaaa</b></span><span class="hl2"><b>aaaa</b></span><span class="hl1"><b>aaaa</b></span>`,
			core.Recipe{{Op: "Fuzzy Match", Args: def("aaaa")}},
		},
		// The separator-boosted later alignment outscores the greedy first-letter
		// match, so the recursive result is chosen (match starts at index 3).
		{
			"recursion beats greedy", "a _abc", `a _<span class="hl1"><b>abc</b></span>`,
			core.Recipe{{Op: "Fuzzy Match", Args: def("abc")}},
		},
	})
}

// TestFuzzyMatchInternals covers guards not reachable through ordinary inputs:
// an empty match list and the maxMatches abort.
func TestFuzzyMatchInternals(t *testing.T) {
	if r := calcMatchRanges(nil); len(r) != 0 {
		t.Fatalf("calcMatchRanges(nil) = %v, want empty", r)
	}
	if ok, _, _ := fuzzyMatchRecursive([]rune("a"), []rune("a"), 0, 0, nil, nil, 0, 0, 0, 10, fuzzyDefaultWeights); ok {
		t.Fatal("fuzzyMatchRecursive with maxMatches=0 should abort")
	}
}

// TestFuzzyScore documents the scoring phase extracted from fuzzyMatchRecursive:
// base score, the first-letter bonus, and the sequential bonus for adjacent
// matches. For "abc" fully matched at [0,1,2] with the default weights:
// 100 + firstLetter(15) + 2*sequential(15) = 145.
func TestFuzzyScore(t *testing.T) {
	got := fuzzyScore([]int{0, 1, 2}, 3, []rune("abc"), fuzzyDefaultWeights)
	if got != 145 {
		t.Fatalf("fuzzyScore = %v, want 145", got)
	}
	// A camelCase boundary (lower then upper) adds the camel bonus.
	// "aB" matched at [1]: 100 + leadingPenalty(-5) + unmatched(-1) + camel(30) = 124.
	if got := fuzzyScore([]int{1}, 1, []rune("aB"), fuzzyDefaultWeights); got != 124 {
		t.Fatalf("camel fuzzyScore = %v, want 124", got)
	}
}

// TestFuzzyBest documents the "best recursive alternative" tracker: it keeps the
// highest-scoring matched result and ignores unmatched or lower-scoring ones.
func TestFuzzyBest(t *testing.T) {
	var b fuzzyBest
	b.consider(true, 50, []int{1, 2})
	if !b.matched || b.score != 50 {
		t.Fatalf("first: %+v", b)
	}
	b.consider(true, 60, []int{3}) // higher score wins
	if b.score != 60 || len(b.matches) != 1 || b.matches[0] != 3 {
		t.Fatalf("higher: %+v", b)
	}
	b.consider(true, 40, []int{4}) // lower score ignored
	if b.score != 60 {
		t.Fatalf("lower changed score: %+v", b)
	}
	b.consider(false, 99, []int{9}) // unmatched ignored
	if b.score != 60 {
		t.Fatalf("unmatched changed score: %+v", b)
	}
}
