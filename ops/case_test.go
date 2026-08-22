package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

// TestCaseFullMapping covers the characters whose case conversion changes
// length or produces a different sequence — Unicode calls this full case
// mapping, as opposed to the simple one-rune-for-one-rune kind. JavaScript's
// toUpperCase and toLowerCase apply it, so CyberChef does too, and these
// expectations are its output for the same input.
func TestCaseFullMapping(t *testing.T) {
	runCases(t, []opCase{
		// One character expanding to several.
		{"upper sharp s", "ß", "SS", upperRecipe},
		{"upper fi ligature", "ﬁ", "FI", upperRecipe},
		{"upper ffl ligature", "ﬄ", "FFL", upperRecipe},
		{"upper st ligature", "ﬅ", "ST", upperRecipe},
		{"upper n preceded by apostrophe", "ŉ", "ʼN", upperRecipe},
		{"upper armenian ech-yiwn", "և", "ԵՒ", upperRecipe},
		// A precomposed character decomposing into base plus combining marks.
		{"upper j with caron", "ǰ", "J̌", upperRecipe},
		{"upper iota with dialytika and tonos", "ΐ", "Ϊ́", upperRecipe},
		// Lower case is not simply the reverse: a capital I with a dot above
		// keeps the dot as a combining mark.
		{"lower capital i with dot above", "İ", "i̇", lowerRecipe},
		// Greek sigma takes its final form at the end of a word, which depends
		// on surrounding context rather than the character alone.
		{"lower sigma at end of word", "ΑΣ", "ας", lowerRecipe},
		{"lower sigma mid word", "ΑΣΑ", "ασα", lowerRecipe},
		{"lower isolated sigma", "Σ", "σ", lowerRecipe},
	})
}

var (
	upperRecipe = core.Recipe{{Op: "To Upper case", Args: []any{"All"}}}
	lowerRecipe = core.Recipe{{Op: "To Lower case"}}
)

// TestSwapCaseFullMapping covers full case mapping in Swap case, which decides
// per character and so converts one character at a time: a sharp s is lower
// case and becomes "SS", and a capital I with a dot above keeps the dot.
//
// A Greek sigma stays in its non-final form here, unlike To Lower case, because
// converting a single character gives the rule no following letter to look at —
// which is what CyberChef does too, and these are its outputs.
func TestSwapCaseFullMapping(t *testing.T) {
	swap := core.Recipe{{Op: "Swap case"}}
	runCases(t, []opCase{
		{"swap sharp s", "ß", "SS", swap},
		{"swap fi ligature", "ﬁ", "FI", swap},
		{"swap capital i with dot above", "İ", "i̇", swap},
		{"swap sharp s in a word", "foo ßar baz", "FOO SSAR BAZ", swap},
		{"swap sigma keeps non-final form", "ΑΣ ΒΕΤΑ", "ασ βετα", swap},
		// Outside the basic plane the input reaches CyberChef as surrogate
		// halves, which have no case, so these come back untouched even though
		// the characters themselves are cased.
		{"swap leaves upper deseret alone", "𐐀", "𐐀", swap},
		{"swap leaves lower deseret alone", "𐐨", "𐐨", swap},
	})
}

// TestLodashCaseFullMapping covers the case conversions built on the lodash
// word splitter. They lower-case whole words, so the Greek final-sigma rule
// applies and a trailing sigma takes its final form.
func TestLodashCaseFullMapping(t *testing.T) {
	runCases(t, []opCase{
		{"snake sigma at end of word", "ΑΣ ΒΕΤΑ", "ας_βετα", core.Recipe{{Op: "To Snake case"}}},
		{"kebab sigma at end of word", "ΑΣ ΒΕΤΑ", "ας-βετα", core.Recipe{{Op: "To Kebab case"}}},
		{"camel sigma at end of word", "ΑΣ ΒΕΤΑ", "αςΒετα", core.Recipe{{Op: "To Camel case"}}},
	})
}

// TestLodashCaseNonASCIIDigits covers the digit class used to split words.
// lodash's pattern spells it "\d", which in JavaScript means the ten ASCII
// digits and nothing else, so a decimal digit from another script is an
// ordinary letter and starts no new word.
func TestLodashCaseNonASCIIDigits(t *testing.T) {
	runCases(t, []opCase{
		// Thaana letter followed by NKo digit nine: one word, not two.
		{"snake keeps a non-ASCII digit in the word", "ޤ߉", "ޤ߉", core.Recipe{{Op: "To Snake case"}}},
		{"kebab keeps a non-ASCII digit in the word", "ޤ߉", "ޤ߉", core.Recipe{{Op: "To Kebab case"}}},
		// An ASCII digit still splits, which is what makes the two differ.
		{"snake splits on an ASCII digit", "ޤ9", "ޤ_9", core.Recipe{{Op: "To Snake case"}}},
	})
}

// TestLodashCaseAstralWords pins the one place these operations knowingly
// differ from CyberChef. lodash splits words with a pattern written against
// UTF-16, where a character outside the basic plane is a surrogate pair that
// one alternative matches on its own, so CyberChef makes it a word of its own:
// "ￍ𐀀" becomes "ￍ_𐀀". Here the pattern runs over code points, where those
// alternatives can never match, and the character joins the letters beside it.
//
// Feeding the pattern surrogates instead does not work: regexp2 will not match
// a lone surrogate, so [\ud800-\udbff] never fires however the input is
// encoded. Closing the gap means rewriting the pattern's surrogate, emoji and
// regional-indicator constructs in terms of code points.
func TestLodashCaseAstralWords(t *testing.T) {
	runCases(t, []opCase{
		{"snake keeps an astral character in the word", "ￍ𐀀", "ￍ𐀀", core.Recipe{{Op: "To Snake case"}}},
		{"kebab keeps an astral character in the word", "ￍ𐀀", "ￍ𐀀", core.Recipe{{Op: "To Kebab case"}}},
	})
}

func TestCaseOps(t *testing.T) {
	runCases(t, []opCase{
		{
			"To Upper All", "Hello, World!", "HELLO, WORLD!",
			core.Recipe{{Op: "To Upper case", Args: []any{"All"}}},
		},
		{
			"To Upper Word", "hello there world", "Hello There World",
			core.Recipe{{Op: "To Upper case", Args: []any{"Word"}}},
		},
		{
			"To Upper Sentence", "hello there. how are you?", "Hello there. How are you?",
			core.Recipe{{Op: "To Upper case", Args: []any{"Sentence"}}},
		},
		{
			"To Upper Paragraph", "hello world\nsecond para", "Hello world\nSecond para",
			core.Recipe{{Op: "To Upper case", Args: []any{"Paragraph"}}},
		},
		{
			"To Lower", "Hello, World!", "hello, world!",
			core.Recipe{{Op: "To Lower case"}},
		},
		{
			"Case round trip", "MiXeD", "MIXED",
			core.Recipe{
				{Op: "To Lower case"},
				{Op: "To Upper case", Args: []any{"All"}},
			},
		},
	})
}
