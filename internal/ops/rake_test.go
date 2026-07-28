package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// rakeRecipe builds a recipe with the delimiters and stop words given.
func rakeRecipe(word, sentence, stop string) core.Recipe {
	return core.Recipe{{Op: "RAKE", Args: []any{word, sentence, stop}}}
}

// TestRAKEFixture covers CyberChef's own case
// (../CyberChef/tests/operations/tests/RAKE.mjs).
func TestRAKEFixture(t *testing.T) {
	runCases(t, []opCase{{
		"RAKE: Basic Example",
		"test1 test2. test2",
		"Scores: , Keywords: \n3.5, test1 test2\n1.5, test2",
		rakeRecipe(`\s`, `\.\s|\n`, "i,me,my,myself,we,our"),
	}})
}

// TestRAKECases covers the operation against the CyberChef-server oracle.
func TestRAKECases(t *testing.T) {
	for _, tc := range []struct{ name, input, want string }{
		// The oracle will not take an empty input, so this one is checked
		// against what the algorithm must give: no phrases, so no rows.
		{"nothing at all", "", "Scores: , Keywords: "},
		{"a single word", "keyword", "Scores: , Keywords: \n1, keyword"},
		{
			"a stop word breaking a sentence in two",
			"alpha beta the gamma delta",
			"Scores: , Keywords: \n4, alpha beta\n4, gamma delta",
		},
		{
			"a repeated phrase counted once",
			"alpha beta. alpha beta",
			"Scores: , Keywords: \n2, alpha beta",
		},
		{
			"a word shared between phrases",
			"alpha beta. beta gamma",
			"Scores: , Keywords: \n4, alpha beta\n4, beta gamma",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runCases(t, []opCase{{
				tc.name, tc.input, tc.want,
				rakeRecipe(`\s`, `\.\s|\n`, "the,and,a,of"),
			}})
		})
	}
}

// TestRAKEDelimiters covers the two patterns the text is split on.
func TestRAKEDelimiters(t *testing.T) {
	runCases(t, []opCase{
		{
			"words split on a comma",
			"alpha,beta,the,gamma",
			"Scores: , Keywords: \n4, alpha beta\n1, gamma",
			rakeRecipe(`,`, `\n`, "the"),
		},
		{
			"sentences split on a semicolon",
			"alpha beta; gamma delta",
			"Scores: , Keywords: \n4, alpha beta\n4, gamma delta",
			rakeRecipe(`\s`, `;\s`, "the"),
		},
	})
}

// TestRAKERejectsBadPatterns covers a delimiter that is not a pattern at all.
func TestRAKERejectsBadPatterns(t *testing.T) {
	for _, args := range [][]any{
		{`[`, `\n`, "the"},
		{`\s`, `[`, "the"},
	} {
		if _, err := runOp(t, "RAKE", "some text", args...); err == nil {
			t.Errorf("args %v were accepted", args)
		}
	}
}

// TestRAKEPhraseWithUnknownWord covers a phrase holding a word that was never
// counted, which happens when the same word is both a stop word and not,
// depending on where the sentence split fell. It contributes nothing.
func TestRAKEPhraseWithUnknownWord(t *testing.T) {
	got := rakeScorePhrases(
		[]string{"alpha"}, []int{1},
		[][]string{{"alpha", "unknown"}},
	)
	if len(got) != 1 {
		t.Fatalf("scored %d phrases, want 1", len(got))
	}
	if got[0].phrase != "alpha unknown" {
		t.Errorf("phrase = %q", got[0].phrase)
	}
	// alpha's degree is the length of the one phrase it appears in.
	if got[0].score != 2 {
		t.Errorf("score = %v, want 2", got[0].score)
	}
}
