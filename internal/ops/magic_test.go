package ops

import (
	"math"
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// TestMagicEntropy checks the Shannon entropy at its extremes and against a
// case that can be worked out by hand.
func TestMagicEntropy(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		want float64
	}{
		{"nothing", nil, 0},
		{"one value repeated", []byte("aaaaaaaa"), 0},
		{"two values evenly", []byte("abab"), 1},
		{"four values evenly", []byte("abcd"), 2},
		{"every byte once", allByteValues(), 8},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := magicEntropy(c.data); math.Abs(got-c.want) > 1e-12 {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

// allByteValues is every byte value exactly once, whose entropy is the maximum.
func allByteValues() []byte {
	out := make([]byte, 256)
	for i := range out {
		out[i] = byte(i)
	}
	return out
}

// TestMagicFreqDist checks the frequency distribution is a percentage per byte.
func TestMagicFreqDist(t *testing.T) {
	freq := magicFreqDist([]byte("aabb"))
	if freq['a'] != 50 || freq['b'] != 50 {
		t.Errorf("got a=%v b=%v, want 50 each", freq['a'], freq['b'])
	}
	if freq['c'] != 0 {
		t.Errorf("absent byte has frequency %v", freq['c'])
	}
	if empty := magicFreqDist(nil); empty != [256]float64{} {
		t.Error("no data should give no frequencies")
	}
}

// TestMagicUTF8Kind covers the three answers and the sequences that are
// rejected: overlong forms, surrogates, anything past plane 16, truncated
// sequences, and control characters that are not tab, newline or return.
func TestMagicUTF8Kind(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		want int
	}{
		{"empty is ascii", nil, 1},
		{"plain ascii", []byte("Hello, world!"), 1},
		{"the allowed control characters", []byte("a\tb\nc\rd"), 1},
		{"two-byte sequence", []byte("é"), 2},
		{"three-byte sequence", []byte("中"), 2},
		{"four-byte sequence", []byte("😀"), 2},
		{"a bell is not text", []byte("a\x07b"), 0},
		{"a lone continuation byte", []byte{0x80}, 0},
		{"a truncated two-byte sequence", []byte{0xC3}, 0},
		{"an overlong two-byte sequence", []byte{0xC0, 0xAF}, 0},
		{"an overlong three-byte sequence", []byte{0xE0, 0x80, 0xAF}, 0},
		{"a surrogate", []byte{0xED, 0xA0, 0x80}, 0},
		{"just below the surrogates", []byte{0xED, 0x9F, 0xBF}, 2},
		{"past plane sixteen", []byte{0xF5, 0x80, 0x80, 0x80}, 0},
		{"an overlong four-byte sequence", []byte{0xF0, 0x80, 0x80, 0x80}, 0},
		{"plane sixteen", []byte{0xF4, 0x8F, 0xBF, 0xBF}, 2},
		{"plane four", []byte{0xF1, 0x80, 0x80, 0x80}, 2},
		{"the last three-byte sequence", []byte{0xEF, 0xBF, 0xBD}, 2},
		{"a private-use area character", []byte{0xEE, 0x80, 0x80}, 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := magicUTF8Kind(c.data); got != c.want {
				t.Errorf("got %d, want %d", got, c.want)
			}
		})
	}
}

// TestMagicText checks how the data is read before patterns are tested: as
// UTF-8 where it is valid, and one character per byte where it is not, which is
// what lets a binary signature written \xff match the byte 0xff.
func TestMagicText(t *testing.T) {
	if got := magicText([]byte("héllo")); got != "héllo" {
		t.Errorf("valid UTF-8 changed: %q", got)
	}
	got := magicText([]byte{0xff, 0xd8, 0xff})
	if want := "ÿØÿ"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if !magicPatternMatches(`^\xff\xd8\xff`, []byte{0xff, 0xd8, 0xff, 0x00}) {
		t.Error("a binary signature did not match")
	}
}

// TestMagicPatternMatchesBadPattern checks that a pattern which will not
// compile simply fails to match, rather than stopping the analysis.
func TestMagicPatternMatchesBadPattern(t *testing.T) {
	if magicPatternMatches("([", []byte("anything")) {
		t.Error("an invalid pattern should not match")
	}
}

// TestMagicDetectLanguage checks English text is recognised, and that data with
// no bytes at all is reported as unknown.
func TestMagicDetectLanguage(t *testing.T) {
	scores := magicDetectLanguage([]byte(strings.Repeat(
		"The quick brown fox jumps over the lazy dog. ", 20)), false)
	if scores[0].Lang != "en" {
		t.Errorf("best guess is %q, want en", scores[0].Lang)
	}
	if scores[0].Score > scores[len(scores)-1].Score {
		t.Error("the scores are not sorted best first")
	}

	empty := magicDetectLanguage(nil, false)
	if len(empty) != 1 || empty[0].Lang != "Unknown" {
		t.Errorf("no data gave %+v", empty)
	}

	// The extensive set is much larger than the common one.
	if len(magicDetectLanguage([]byte("hello"), true)) <= len(magicDetectLanguage([]byte("hello"), false)) {
		t.Error("the extensive set is not larger")
	}
}

// TestMagicLanguageName checks a known code is named and an unknown one is
// reported as itself.
func TestMagicLanguageName(t *testing.T) {
	if got := magicLanguageName("en"); got != "English" {
		t.Errorf("got %q, want English", got)
	}
	if got := magicLanguageName("zzz"); got != "zzz" {
		t.Errorf("got %q, want the code back", got)
	}
}

// TestMagicOutputPasses covers each way an operation's result can fail to look
// like what its check promised.
func TestMagicOutputPasses(t *testing.T) {
	png := "\x89PNG\r\n\x1a\n\x00\x00\x00\x0dIHDR"
	cases := []struct {
		name  string
		data  []byte
		check *magicOutputCheck
		want  bool
	}{
		{"no criteria at all", []byte("anything"), nil, true},
		{"pattern matches", []byte("hello"), &magicOutputCheck{Pattern: "^hel"}, true},
		{"pattern does not", []byte("hello"), &magicOutputCheck{Pattern: "^bye"}, false},
		{"entropy inside the range", []byte("aaaa"), &magicOutputCheck{EntropyRange: []float64{0, 1}}, true},
		{"entropy outside it", allByteValues(), &magicOutputCheck{EntropyRange: []float64{0, 1}}, false},
		{"media type matches", []byte(png), &magicOutputCheck{Mime: "image"}, true},
		{"media type does not", []byte("plain text"), &magicOutputCheck{Mime: "image"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := magicOutputPasses(c.data, c.check); got != c.want {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

// TestMagicScore checks each adjustment the ranking makes.
func TestMagicScore(t *testing.T) {
	base := magicOption{LangScores: []magicLangScore{{Score: 1000}}, Entropy: 0}
	plain := magicScore(base)

	text := base
	text.IsUTF8 = true
	if magicScore(text) != plain-100 {
		t.Error("text is not rewarded")
	}

	file := base
	file.FileType = &fileSig{}
	if got := magicScore(file); got != 500 {
		t.Errorf("a recognised file scored %v, want 500", got)
	}

	useful := base
	useful.Useful = true
	if got := magicScore(useful); got != 100 {
		t.Errorf("a useful operation scored %v, want 100", got)
	}

	// A long recipe and high entropy both count against a candidate.
	longer := base
	longer.Recipe = core.Recipe{{Op: "a"}, {Op: "b"}}
	longer.Entropy = 4
	if magicScore(longer) != plain+6 {
		t.Error("recipe length and entropy are not counted")
	}

	// A good score is not made worse by a file or a useful operation.
	good := magicOption{LangScores: []magicLangScore{{Score: 10}}, FileType: &fileSig{}, Useful: true}
	if magicScore(good) != 10 {
		t.Errorf("a good score was changed to %v", magicScore(good))
	}
}

// TestMagicRankPrefersRecipesOverBareSuggestions checks the tie-break: a result
// that has actually run operations beats one that merely suggests some.
func TestMagicRankPrefersRecipesOverBareSuggestions(t *testing.T) {
	bare := magicOption{
		LangScores:  []magicLangScore{{Score: 1}},
		MatchingOps: []magicCheck{{Op: "From Hex"}},
	}
	ran := magicOption{
		LangScores: []magicLangScore{{Score: 900}},
		Recipe:     core.Recipe{{Op: "From Hex"}},
	}
	ranked := magicRank([]magicOption{bare, ran})
	if len(ranked[0].Recipe) == 0 {
		t.Error("the bare suggestion was ranked first")
	}
	// And the other way round, so both sides of the comparison are exercised.
	ranked = magicRank([]magicOption{ran, bare})
	if len(ranked[0].Recipe) == 0 {
		t.Error("order of input changed the ranking")
	}
}

// TestMagicPrune covers each reason a candidate is dropped or kept.
func TestMagicPrune(t *testing.T) {
	nothing := []magicLangScore{{Probability: 0}}
	cases := []struct {
		name string
		opt  magicOption
		keep bool
	}{
		{"empty and not useful", magicOption{Data: "", LangScores: nothing}, false},
		{"empty but useful", magicOption{Data: "", LangScores: nothing, Useful: true, IsUTF8: true}, true},
		{"nothing found at all", magicOption{Data: "x", LangScores: nothing}, false},
		{"a language was found", magicOption{Data: "x", LangScores: []magicLangScore{{Probability: 0.5}}}, true},
		{"a file was found", magicOption{Data: "x", LangScores: nothing, FileType: &fileSig{}}, true},
		{"the data is text", magicOption{Data: "x", LangScores: nothing, IsUTF8: true}, true},
		{"an operation matched", magicOption{Data: "x", LangScores: nothing, MatchingOps: []magicCheck{{}}}, true},
		{"the crib matched", magicOption{Data: "x", LangScores: nothing, MatchesCrib: true}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := len(magicPrune([]magicOption{c.opt})) == 1
			if got != c.keep {
				t.Errorf("kept = %v, want %v", got, c.keep)
			}
		})
	}
}

// TestMagicIsMime checks the media-type prefix test.
func TestMagicIsMime(t *testing.T) {
	png := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\x0dIHDR")
	if !magicIsMime("image", png) {
		t.Error("a PNG is not an image")
	}
	if !magicIsMime("image/png", png) {
		t.Error("the full media type did not match")
	}
	if magicIsMime("audio", png) {
		t.Error("a PNG is audio")
	}
	if magicIsMime("image", []byte("not a file")) {
		t.Error("plain text is an image")
	}
}

// TestMagicEscape checks the report's rendering of characters that would
// otherwise break a line.
func TestMagicEscape(t *testing.T) {
	got := magicEscape("a\nb\tc\rd\x00e\x7ff")
	if want := `a\nb\tc\rd\x00e\x7ff`; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got := magicEscape("plain é"); got != "plain é" {
		t.Errorf("ordinary text was changed: %q", got)
	}
}

// TestMagicSnippetIsTrimmed checks a long result is quoted only in part.
func TestMagicSnippetIsTrimmed(t *testing.T) {
	long := strings.Repeat("a", magicSnippetLen*2)
	if got := magicSnippet([]byte(long)); len(got) != magicSnippetLen {
		t.Errorf("snippet is %d characters, want %d", len(got), magicSnippetLen)
	}
	if got := magicSnippet([]byte("short")); got != "short" {
		t.Errorf("a short result was changed: %q", got)
	}
}

// TestMagicRunRecipeAbandonsFailures checks that a recipe which cannot run
// yields nothing, which is how a branch is abandoned.
func TestMagicRunRecipeAbandonsFailures(t *testing.T) {
	run := &magicRun{registry: core.Default}
	if out := run.runRecipe(core.Recipe{{Op: "No Such Operation"}}, []byte("x")); out != nil {
		t.Errorf("an unknown operation gave %q", out)
	}
	if out := run.runRecipe(core.Recipe{{Op: "To Upper case", Args: []any{"All"}}}, []byte("x")); string(out) != "X" {
		t.Errorf("a working recipe gave %q", out)
	}
}

// TestMagicEncodingGuessesNeedsTheOperation checks the brute-force encodings
// stop cleanly when the operation they rely on is not registered.
func TestMagicEncodingGuessesNeedsTheOperation(t *testing.T) {
	run := &magicRun{registry: core.NewRegistry()}
	if got := run.encodingGuesses([]byte("hello")); got != nil {
		t.Errorf("got %d guesses from an empty registry", len(got))
	}
}

// TestMagicAttempt covers each reason one operation's turn is abandoned: a
// result that is empty, one that repeats the previous operation without
// changing anything, and one that does not look like what the check promised.
func TestMagicAttempt(t *testing.T) {
	run := &magicRun{registry: core.Default}
	upper := magicCheck{Op: "To Upper case", Args: []any{"All"}}

	t.Run("a working operation is kept", func(t *testing.T) {
		out, ok := run.attempt([]byte("abc"), upper, "")
		if !ok || string(out) != "ABC" {
			t.Errorf("got %q, %v", out, ok)
		}
	})
	t.Run("an operation that cannot run is abandoned", func(t *testing.T) {
		if _, ok := run.attempt([]byte("abc"), magicCheck{Op: "No Such Operation"}, ""); ok {
			t.Error("an unknown operation was kept")
		}
	})
	t.Run("an empty result is abandoned", func(t *testing.T) {
		if _, ok := run.attempt(nil, upper, ""); ok {
			t.Error("an empty result was kept")
		}
	})
	t.Run("repeating itself unchanged is abandoned", func(t *testing.T) {
		// Upper-casing something already upper case changes nothing, so a
		// second turn at it goes nowhere.
		if _, ok := run.attempt([]byte("ABC"), upper, "To Upper case"); ok {
			t.Error("a repeat that changed nothing was kept")
		}
		// The same result is fine when a different operation produced it.
		if _, ok := run.attempt([]byte("ABC"), upper, "From Hex"); !ok {
			t.Error("a different previous operation should not matter")
		}
	})
	t.Run("a result unlike what was promised is abandoned", func(t *testing.T) {
		promising := upper
		promising.Output = &magicOutputCheck{Pattern: "^this will not appear$"}
		if _, ok := run.attempt([]byte("abc"), promising, ""); ok {
			t.Error("a result failing its output check was kept")
		}
	})
}
