package ops

import (
	"math"
	"strings"
	"testing"
	"unicode"

	"github.com/roberson-io/cchef/core"
)

// makeLorem generates placeholder text of the length and unit asked for.
func makeLorem(t *testing.T, length int, unit string) string {
	t.Helper()
	out, err := runOp(t, "Generate Lorem Ipsum", "", length, unit)
	if err != nil {
		t.Fatalf("Generate Lorem Ipsum(%d, %s): %v", length, unit, err)
	}
	return out
}

// loremFields splits text into its words, however it is spaced.
func loremFields(text string) []string { return strings.Fields(text) }

// loremSentences splits text into its sentences, dropping the empty tail after
// the final full stop.
func loremSplitSentences(text string) []string {
	var out []string
	for s := range strings.SplitSeq(strings.ReplaceAll(text, "\n", " "), ".") {
		if strings.TrimSpace(s) != "" {
			out = append(out, strings.TrimSpace(s))
		}
	}
	return out
}

// loremParagraphs splits text into its paragraphs.
func loremSplitParagraphs(text string) []string {
	var out []string
	for p := range strings.SplitSeq(text, "\n\n") {
		if strings.TrimSpace(p) != "" {
			out = append(out, p)
		}
	}
	return out
}

// TestGenerateLoremIpsumLimits covers CyberChef's own cases
// (CyberChef's tests/operations/tests/GenerateLoremIpsum.mjs), which check the
// bounds on the length rather than the text itself.
func TestGenerateLoremIpsumLimits(t *testing.T) {
	for _, tc := range []struct {
		name   string
		length int
		unit   string
		want   string
	}{
		{"more words than it will make", 999999, "Words", "Length must be less than 100000"},
		{"more sentences than it will make", 999999, "Sentences", "Length must be less than 100000"},
		{"more paragraphs than it will make", 999999, "Paragraphs", "Length must be less than 100000"},
		{"more bytes than it will make", 1000001, "Bytes", "Length must be less than 1000000"},
		{"no words at all", 0, "Words", "Length must be greater than 0"},
		{"fewer than none", -1, "Bytes", "Length must be greater than 0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := runOp(t, "Generate Lorem Ipsum", "", tc.length, tc.unit)
			if err == nil {
				t.Fatalf("accepted a length of %d, giving %d characters", tc.length, len(out))
			}
			if err.Error() != tc.want {
				t.Errorf("got %q, want %q", err.Error(), tc.want)
			}
		})
	}
}

// TestGenerateLoremIpsumLimitsAreInclusive covers the far end of each bound,
// which is allowed.
func TestGenerateLoremIpsumLimitsAreInclusive(t *testing.T) {
	if got := len(loremFields(makeLorem(t, maxLoremWords, "Words"))); got != maxLoremWords {
		t.Errorf("asked for %d words and got %d", maxLoremWords, got)
	}
	if got := len(makeLorem(t, maxLoremBytes, "Bytes")); got != maxLoremBytes {
		t.Errorf("asked for %d bytes and got %d", maxLoremBytes, got)
	}
}

// TestGenerateLoremIpsumUnknownUnit covers the guard on the unit. The recipe
// engine checks the option before the operation runs, so the guard only answers
// a direct call.
func TestGenerateLoremIpsumUnknownUnit(t *testing.T) {
	if _, err := (GenerateLoremIpsum{}).Run(
		core.NewDish(nil, core.TypeString), []any{float64(3), "Novels"},
	); err == nil {
		t.Error("accepted a unit the operation does not offer")
	}
}

// TestGenerateLoremIpsumCounts covers the length: asking for a number of words,
// sentences, paragraphs or bytes gives exactly that many.
func TestGenerateLoremIpsumCounts(t *testing.T) {
	for _, tc := range []struct {
		unit  string
		count func(string) int
	}{
		{"Words", func(s string) int { return len(loremFields(s)) }},
		{"Sentences", func(s string) int { return len(loremSplitSentences(s)) }},
		{"Paragraphs", func(s string) int { return len(loremSplitParagraphs(s)) }},
		{"Bytes", func(s string) int { return len(s) }},
	} {
		t.Run(tc.unit, func(t *testing.T) {
			for _, length := range []int{1, 2, 3, 5, 6, 17, 50} {
				if got := tc.count(makeLorem(t, length, tc.unit)); got != length {
					t.Errorf("asked for %d %s and got %d", length, tc.unit, got)
				}
			}
		})
	}
}

// TestGenerateLoremIpsumUsesItsOwnWords covers the vocabulary: every word comes
// from the list, whatever punctuation it carries.
func TestGenerateLoremIpsumUsesItsOwnWords(t *testing.T) {
	known := map[string]bool{}
	for _, word := range loremWordList {
		known[strings.ToLower(word)] = true
	}

	for _, word := range loremFields(makeLorem(t, 2000, "Words")) {
		bare := strings.ToLower(strings.Trim(word, ".,"))
		if !known[bare] {
			t.Errorf("%q is not one of the words the generator knows", word)
		}
	}
}

// TestGenerateLoremIpsumSentenceShape covers how a sentence is written: it opens
// with a capital and closes with a full stop.
func TestGenerateLoremIpsumSentenceShape(t *testing.T) {
	for _, sentence := range loremSplitSentences(makeLorem(t, 40, "Sentences")) {
		first := []rune(sentence)[0]
		if !unicode.IsUpper(first) {
			t.Errorf("sentence %q does not open with a capital", sentence)
		}
	}
	text := makeLorem(t, 40, "Sentences")
	if stops := strings.Count(text, "."); stops != 40 {
		t.Errorf("40 sentences carry %d full stops", stops)
	}
}

// TestGenerateLoremIpsumOpening covers the opening words, which are always the
// same however the rest comes out.
func TestGenerateLoremIpsumOpening(t *testing.T) {
	const opening = "Lorem ipsum dolor sit amet"

	for _, unit := range []string{"Words", "Sentences", "Paragraphs"} {
		t.Run(unit, func(t *testing.T) {
			if got := makeLorem(t, 30, unit); !strings.HasPrefix(got, opening) {
				t.Errorf("began %q", got[:min(len(got), len(opening))])
			}
		})
	}

	// Asked for fewer words than the opening has, it gives as much of it as
	// will fit.
	for length, want := range map[int]string{
		1: "Lorem.", 2: "Lorem ipsum.", 3: "Lorem ipsum dolor.",
		4: "Lorem ipsum dolor sit.", 5: "Lorem ipsum dolor sit amet.",
	} {
		if got := makeLorem(t, length, "Words"); got != want {
			t.Errorf("%d words gave %q, want %q", length, got, want)
		}
	}
}

// TestGenerateLoremIpsumNoWordTwiceRunning covers the one rule the word picker
// keeps: it never gives the same word twice in a row.
func TestGenerateLoremIpsumNoWordTwiceRunning(t *testing.T) {
	words := loremPickWords(5000)
	if len(words) != 5000 {
		t.Fatalf("asked for 5000 words and got %d", len(words))
	}
	for i := 1; i < len(words); i++ {
		if words[i] == words[i-1] {
			t.Fatalf("%q appears twice running at %d", words[i], i)
		}
	}
}

// TestGenerateLoremIpsumSentenceLengths covers the spread of sentence lengths,
// which is drawn about a mean rather than fixed.
func TestGenerateLoremIpsumSentenceLengths(t *testing.T) {
	var lengths []int
	for range 30 {
		// The last sentence holds whatever words are left over, so it is not
		// drawn from the distribution and is left out.
		sentences := loremSplitSentences(makeLorem(t, 400, "Words"))
		for _, s := range sentences[:len(sentences)-1] {
			lengths = append(lengths, len(strings.Fields(s)))
		}
	}
	if len(lengths) < 100 {
		t.Fatalf("only %d sentences to judge by", len(lengths))
	}

	var total int
	for _, n := range lengths {
		total += n
		if n < 1 {
			t.Errorf("a sentence of %d words", n)
		}
	}
	mean := float64(total) / float64(len(lengths))
	if math.Abs(mean-loremSentenceMean) > 2 {
		t.Errorf("sentences average %.2f words, want about %d", mean, loremSentenceMean)
	}

	spread := map[int]bool{}
	for _, n := range lengths {
		spread[n] = true
	}
	if len(spread) < 10 {
		t.Errorf("only %d distinct sentence lengths, which is not much of a spread", len(spread))
	}
}

// TestGenerateLoremIpsumCommas covers the commas, which are scattered through
// the sentences rather than being absent or in every one.
func TestGenerateLoremIpsumCommas(t *testing.T) {
	var withComma, total int
	for range 20 {
		for _, sentence := range loremSplitSentences(makeLorem(t, 60, "Sentences")) {
			total++
			if strings.Contains(sentence, ",") {
				withComma++
			}
		}
	}

	share := float64(withComma) / float64(total)
	if math.Abs(share-loremCommaChance) > 0.1 {
		t.Errorf("%.2f of sentences carry a comma, want about %.2f", share, loremCommaChance)
	}
}

// TestGenerateLoremIpsumVaries covers that the text is drawn afresh each time.
func TestGenerateLoremIpsumVaries(t *testing.T) {
	seen := map[string]bool{}
	for range 20 {
		text := makeLorem(t, 30, "Words")
		if seen[text] {
			t.Fatal("the same text came up twice")
		}
		seen[text] = true
	}
}
