package ops

import (
	"math"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/roberson-io/cchef/core"
)

// The analysis behind the Magic operation, ported from CyberChef's
// lib/Magic.mjs. Magic looks at data, works out which operations claim to be
// able to decode something of that shape, runs them, and looks at what comes
// out — recursively, to a given depth — then ranks the recipes it found by how
// much like ordinary text or a known file the result became.

// magicSnippetLen is how much of each result is quoted back in the report.
const magicSnippetLen = 100

// magicBruteForceLen is how much of the data brute forcing works on, since
// trying every byte over a large input is slow and rarely more informative.
const magicBruteForceLen = 100

// magicLangScore is how well the data's byte frequencies fit one language.
type magicLangScore struct {
	Lang        string
	Score       float64
	Probability float64
}

// magicOption is one candidate recipe and what its result looks like.
type magicOption struct {
	Recipe      core.Recipe
	Data        string
	LangScores  []magicLangScore
	FileType    *fileSig
	IsUTF8      bool
	Entropy     float64
	MatchingOps []magicCheck
	Useful      bool
	MatchesCrib bool
}

// magicRun carries the settings for one invocation, so the recursion does not
// have to pass them all down.
type magicRun struct {
	extensive bool
	intensive bool
	crib      *regexp.Regexp
	registry  *core.Registry
}

// magicFreqDist counts how often each byte value appears, as a percentage.
func magicFreqDist(data []byte) [256]float64 {
	var counts [256]int
	for _, b := range data {
		counts[b]++
	}
	var freq [256]float64
	if len(data) == 0 {
		return freq
	}
	for i, c := range counts {
		freq[i] = float64(c) / float64(len(data)) * 100
	}
	return freq
}

// magicEntropy is the Shannon entropy of the data, from 0 to 8. High entropy
// suggests compressed or encrypted data; ordinary text sits around 3.5 to 5.
func magicEntropy(data []byte) float64 {
	freq := magicFreqDist(data)
	entropy := 0.0
	for _, f := range freq {
		p := f / 100
		if p == 0 {
			continue
		}
		entropy += p * math.Log2(p)
	}
	return -entropy
}

// magicUTF8Kind reports whether the data could be text: 0 if not, 1 if it is
// all printable ASCII, 2 if it is valid UTF-8 using more than ASCII. Ported
// from CyberChef's isUTF8, which is stricter than a plain UTF-8 check — a
// control character other than tab, newline or carriage return rules text out.
func magicUTF8Kind(data []byte) int {
	onlyASCII := true
	for i := 0; i < len(data); {
		if isPrintableASCII(data[i]) {
			i++
			continue
		}
		onlyASCII = false
		width := utf8SequenceWidth(data[i:])
		if width == 0 {
			return 0
		}
		i += width
	}
	if onlyASCII {
		return 1
	}
	return 2
}

// isPrintableASCII reports whether b is text on its own.
func isPrintableASCII(b byte) bool {
	return b == 0x09 || b == 0x0A || b == 0x0D || (b >= 0x20 && b <= 0x7E)
}

// utf8Shape is one well-formed shape a multi-byte UTF-8 sequence may take: a
// range the leading byte falls in, the narrower range its follower is allowed,
// and how many bytes the whole sequence takes. The follower's range is what
// rules out the overlong forms, the surrogates, and anything past plane 16;
// every byte after it is an ordinary continuation.
type utf8Shape struct {
	leadLo, leadFrom byte
	nextLo, nextHi   byte
	width            int
}

var utf8Shapes = [...]utf8Shape{
	{0xC2, 0xDF, 0x80, 0xBF, 2},
	{0xE0, 0xE0, 0xA0, 0xBF, 3}, // not overlong
	{0xE1, 0xEC, 0x80, 0xBF, 3},
	{0xED, 0xED, 0x80, 0x9F, 3}, // not a surrogate
	{0xEE, 0xEF, 0x80, 0xBF, 3},
	{0xF0, 0xF0, 0x90, 0xBF, 4}, // not overlong
	{0xF1, 0xF3, 0x80, 0xBF, 4},
	{0xF4, 0xF4, 0x80, 0x8F, 4}, // not past plane 16
}

// utf8SequenceWidth returns the length of the well-formed UTF-8 sequence at the
// start of data, or 0 if there is not one.
func utf8SequenceWidth(data []byte) int {
	for _, shape := range utf8Shapes {
		if data[0] < shape.leadLo || data[0] > shape.leadFrom {
			continue
		}
		if len(data) < shape.width {
			return 0
		}
		if data[1] < shape.nextLo || data[1] > shape.nextHi {
			return 0
		}
		for _, b := range data[2:shape.width] {
			if b < 0x80 || b > 0xBF {
				return 0
			}
		}
		return shape.width
	}
	return 0
}

// magicChiSquared compares the data's byte frequencies against a language's,
// returning the goodness-of-fit score (lower fits better) and the probability
// of seeing a fit that poor by chance.
func magicChiSquared(observed, expected [256]float64) (score, probability float64) {
	for i := range observed {
		diff := observed[i] - expected[i]
		score += diff * diff / expected[i]
	}
	return score, 1 - chiSquaredCDF(score, float64(len(observed)-1))
}

// magicDetectLanguage ranks the languages by how well the data's byte
// frequencies fit each, best first.
func magicDetectLanguage(data []byte, extensive bool) []magicLangScore {
	if len(data) == 0 {
		return []magicLangScore{{Lang: "Unknown", Score: math.MaxFloat64, Probability: 0}}
	}
	langs := magicCommonLangs
	if extensive {
		langs = magicExtensiveLangs
	}
	freq := magicFreqDist(data)
	scores := make([]magicLangScore, 0, len(langs))
	for lang, expected := range langs {
		score, probability := magicChiSquared(freq, expected)
		scores = append(scores, magicLangScore{Lang: lang, Score: score, Probability: probability})
	}
	sort.SliceStable(scores, func(i, j int) bool { return scores[i].Score < scores[j].Score })
	return scores
}

// magicLanguageName gives a language's name, falling back to its code.
func magicLanguageName(code string) string {
	if name, ok := magicLanguageNames[code]; ok {
		return name
	}
	return code
}

// magicDetectFileType reports the first file signature the data matches.
func magicDetectFileType(data []byte) *fileSig {
	types := detectFileType(data, nil)
	if len(types) == 0 {
		return nil
	}
	return &types[0]
}

// magicMatchingChecks finds the operations claiming they can decode this data.
func magicMatchingChecks(data []byte, entropy float64) []magicCheck {
	var matches []magicCheck
	for _, check := range magicChecks {
		if !inEntropyRange(entropy, check.EntropyRange) {
			continue
		}
		if check.Pattern != "" && !magicPatternMatches(check.Pattern, data) {
			continue
		}
		matches = append(matches, check)
	}
	return matches
}

// inEntropyRange reports whether the entropy lies inside the range, which may
// be absent meaning any entropy will do.
func inEntropyRange(entropy float64, r []float64) bool {
	return len(r) != 2 || (entropy >= r[0] && entropy <= r[1])
}

// magicPatternMatches tests one of the generated patterns against the data. A
// pattern that will not compile simply does not match, so one bad entry in the
// table cannot stop the whole analysis.
func magicPatternMatches(pattern string, data []byte) bool {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}
	return re.MatchString(bytesAsText(data))
}

// magicOutputPasses reports whether an operation's result looks like what its
// check promised.
func magicOutputPasses(data []byte, out *magicOutputCheck) bool {
	if out == nil {
		return true
	}
	if out.Pattern != "" && !magicPatternMatches(out.Pattern, data) {
		return false
	}
	if !inEntropyRange(magicEntropy(data), out.EntropyRange) {
		return false
	}
	if out.Mime != "" && !magicIsMime(out.Mime, data) {
		return false
	}
	return true
}

// magicIsMime reports whether the data is detected as a file whose media type
// starts with the given prefix.
func magicIsMime(prefix string, data []byte) bool {
	for _, t := range detectFileType(data, nil) {
		if strings.HasPrefix(t.mime, prefix) {
			return true
		}
	}
	return false
}

// magicSnippet is the first hundred characters of a result, quoted back in the
// report, reading the data the same way the patterns do.
func magicSnippet(data []byte) string {
	runes := []rune(bytesAsText(data))
	if len(runes) > magicSnippetLen {
		return string(runes[:magicSnippetLen])
	}
	return string(runes)
}

// magicDescribe records everything the analysis knows about one piece of data.
func (m *magicRun) describe(data []byte, recipe core.Recipe, useful bool) magicOption {
	entropy := magicEntropy(data)
	return magicOption{
		Recipe:      recipe,
		Data:        magicSnippet(data),
		LangScores:  magicDetectLanguage(data, m.extensive),
		FileType:    magicDetectFileType(data),
		IsUTF8:      magicUTF8Kind(data) != 0,
		Entropy:     entropy,
		MatchingOps: magicMatchingChecks(data, entropy),
		Useful:      useful,
		MatchesCrib: m.crib != nil && m.crib.MatchString(bytesAsText(data)),
	}
}

// magicRunRecipe runs a recipe over the data, returning nothing at all if it
// does not run to completion, which is how a branch is abandoned.
func (m *magicRun) runRecipe(recipe core.Recipe, data []byte) []byte {
	out, err := recipe.ExecuteWith(m.registry, core.NewDish(data, core.TypeArrayBuffer))
	if err != nil {
		return nil
	}
	return out.Bytes()
}

// attempt gives one operation its turn at the data, reporting what it produced
// and whether that is worth looking at further. It is not, if the operation
// could not run or produced nothing, if it merely repeated the operation above
// it without changing anything, or if the result does not look like what the
// check promised.
func (m *magicRun) attempt(data []byte, check magicCheck, prevOp string) ([]byte, bool) {
	step := core.RecipeOp{Op: check.Op, Args: check.Args}
	output := m.runRecipe(core.Recipe{step}, data)
	switch {
	case len(output) == 0:
		return nil, false
	case prevOp == check.Op && slices.Equal(output, data):
		return nil, false
	case !magicOutputPasses(output, check.Output):
		return nil, false
	}
	return output, true
}

// speculate is the recursion: record what this data looks like, then run each
// operation that claims to decode it and look at the result the same way.
func (m *magicRun) speculate(data []byte, depth int, recipe core.Recipe, useful bool) []magicOption {
	if depth < 0 {
		return nil
	}
	options := []magicOption{m.describe(data, recipe, useful)}
	matching := options[0].MatchingOps

	var prevOp string
	if len(recipe) > 0 {
		prevOp = recipe[len(recipe)-1].Op
	}

	for _, check := range matching {
		output, ok := m.attempt(data, check, prevOp)
		if !ok {
			continue
		}
		step := core.RecipeOp{Op: check.Op, Args: check.Args}
		options = append(options,
			m.speculate(output, depth-1, append(slices.Clone(recipe), step), check.Useful)...)
	}

	if m.intensive {
		for _, guess := range m.bruteForce(data) {
			options = append(options,
				m.speculate(guess.data, depth-1, append(slices.Clone(recipe), guess.step), false)...)
		}
	}

	return magicRank(magicPrune(options))
}

// magicPrune drops the candidates that tell us nothing: an empty result, or one
// with no sign of language, file, text, decodable shape or the wanted string.
func magicPrune(options []magicOption) []magicOption {
	kept := options[:0]
	for _, o := range options {
		if !o.Useful && o.Data == "" {
			continue
		}
		if o.LangScores[0].Probability > 0 || o.FileType != nil || o.IsUTF8 ||
			len(o.MatchingOps) > 0 || o.MatchesCrib {
			kept = append(kept, o)
		}
	}
	return kept
}

// magicRank sorts the candidates so the most promising comes first, scoring
// each the way CyberChef does: lower is better, text and known files are
// rewarded, and longer recipes and higher entropy are penalised.
func magicRank(options []magicOption) []magicOption {
	sort.SliceStable(options, func(i, j int) bool {
		a, b := options[i], options[j]
		// A bare result that only suggests further operations is worse than
		// one that has actually run some.
		if len(a.Recipe) == 0 && len(a.MatchingOps) > 0 && len(b.Recipe) > 0 {
			return false
		}
		if len(b.Recipe) == 0 && len(b.MatchingOps) > 0 && len(a.Recipe) > 0 {
			return true
		}
		return magicScore(a) < magicScore(b)
	})
	return options
}

// magicScore is one candidate's ranking score, lower being more promising.
func magicScore(o magicOption) float64 {
	score := o.LangScores[0].Score
	if o.IsUTF8 {
		score -= 100
	}
	if o.FileType != nil && score > 500 {
		score = 500
	}
	if o.Useful && score > 100 {
		score = 100
	}
	return score + float64(len(o.Recipe)) + o.Entropy
}
