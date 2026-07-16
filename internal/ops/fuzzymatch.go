package ops

import (
	"strings"
	"unicode"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(FuzzyMatch{})
}

// fuzzyWeights holds the scoring weights for the fuzzy matcher.
type fuzzyWeights struct {
	sequentialBonus, separatorBonus, camelBonus, firstLetterBonus         float64
	leadingLetterPenalty, maxLeadingLetterPenalty, unmatchedLetterPenalty float64
}

// fuzzyDefaultWeights mirrors DEFAULT_WEIGHTS in FuzzyMatch.mjs.
var fuzzyDefaultWeights = fuzzyWeights{15, 30, 30, 15, -5, -15, -1}

// fuzzyResult is a single global match: its score and matched indices.
type fuzzyResult struct {
	score float64
	idxs  []int
}

// FuzzyMatch performs a fuzzy search for a pattern within the input.
type FuzzyMatch struct{}

// Meta returns the operation metadata.
func (FuzzyMatch) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Fuzzy Match",
		Module:      "Default",
		Description: "Conducts a fuzzy search to find a pattern within the input, highlighting any matches.",
		InfoURL:     "",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (FuzzyMatch) Args() []core.ArgDef {
	w := fuzzyDefaultWeights
	return []core.ArgDef{
		{Name: "Search", Type: core.ArgString, Value: ""},
		{Name: "Sequential bonus", Type: core.ArgNumber, Value: w.sequentialBonus},
		{Name: "Separator bonus", Type: core.ArgNumber, Value: w.separatorBonus},
		{Name: "Camel bonus", Type: core.ArgNumber, Value: w.camelBonus},
		{Name: "First letter bonus", Type: core.ArgNumber, Value: w.firstLetterBonus},
		{Name: "Leading letter penalty", Type: core.ArgNumber, Value: w.leadingLetterPenalty},
		{Name: "Max leading letter penalty", Type: core.ArgNumber, Value: w.maxLeadingLetterPenalty},
		{Name: "Unmatched letter penalty", Type: core.ArgNumber, Value: w.unmatchedLetterPenalty},
	}
}

// Run performs the fuzzy match. Ported from CyberChef FuzzyMatch.mjs + lib/FuzzyMatch.mjs.
func (FuzzyMatch) Run(in *core.Dish, args []any) (*core.Dish, error) {
	search := parseEscapedChars(args[0].(string))
	w := fuzzyWeights{
		sequentialBonus: args[1].(float64), separatorBonus: args[2].(float64),
		camelBonus: args[3].(float64), firstLetterBonus: args[4].(float64),
		leadingLetterPenalty: args[5].(float64), maxLeadingLetterPenalty: args[6].(float64),
		unmatchedLetterPenalty: args[7].(float64),
	}

	input := []rune(in.String())
	matches := fuzzyMatchGlobal([]rune(search), input, w)

	var b strings.Builder
	pos := 0
	hlClass := "hl1"
	for _, m := range matches {
		for i, r := range calcMatchRanges(m.idxs) {
			start, length := r[0], r[1]
			b.WriteString(escapeHTML(string(input[pos:start])))
			if i == 0 {
				b.WriteString(`<span class="` + hlClass + `">`)
			}
			pos = start + length
			b.WriteString("<b>" + escapeHTML(string(input[start:pos])) + "</b>")
		}
		b.WriteString("</span>")
		if hlClass == "hl1" {
			hlClass = "hl2"
		} else {
			hlClass = "hl1"
		}
	}
	b.WriteString(escapeHTML(string(input[pos:])))
	return core.NewDish([]byte(b.String()), core.TypeString), nil
}

// fuzzyMatchGlobal returns all fuzzy matches of pattern in str.
func fuzzyMatchGlobal(pattern, str []rune, w fuzzyWeights) []fuzzyResult {
	const maxMatches = 256
	const recursionLimit = 10
	var results []fuzzyResult
	strCurrIndex := 0
	for {
		found, score, idxs := fuzzyMatchRecursive(pattern, str, 0, strCurrIndex, nil, nil,
			maxMatches, 0, 0, recursionLimit, w)
		if !found {
			break
		}
		results = append(results, fuzzyResult{score, append([]int(nil), idxs...)})
		strCurrIndex = idxs[len(idxs)-1] + 1
	}
	return results
}

func fuzzyMatchRecursive(pattern, str []rune, patternCurIndex, strCurrIndex int, srcMatches, matches []int,
	maxMatches, nextMatch, recursionCount, recursionLimit int, w fuzzyWeights,
) (bool, float64, []int) {
	outScore := 0.0
	recursionCount++
	if recursionCount >= recursionLimit {
		return false, outScore, []int{}
	}
	if patternCurIndex == len(pattern) || strCurrIndex == len(str) {
		return false, outScore, []int{}
	}

	var best fuzzyBest
	firstMatch := true

	for patternCurIndex < len(pattern) && strCurrIndex < len(str) {
		if unicode.ToLower(pattern[patternCurIndex]) == unicode.ToLower(str[strCurrIndex]) {
			if nextMatch >= maxMatches {
				return false, outScore, []int{}
			}
			if firstMatch && srcMatches != nil {
				matches = append([]int(nil), srcMatches...)
				firstMatch = false
			}
			matched, recursiveScore, recMatches := fuzzyMatchRecursive(pattern, str,
				patternCurIndex, strCurrIndex+1, matches, nil,
				maxMatches, nextMatch, recursionCount, recursionLimit, w)
			best.consider(matched, recursiveScore, recMatches)
			matches = fuzzySetIdx(matches, nextMatch, strCurrIndex)
			nextMatch++
			patternCurIndex++
		}
		strCurrIndex++
	}

	if patternCurIndex != len(pattern) {
		return false, outScore, matches
	}

	outScore = fuzzyScore(matches, nextMatch, str, w)
	if best.matched && best.score > outScore {
		return true, best.score, best.matches
	}
	return true, outScore, matches
}

// fuzzyBest tracks the highest-scoring matched result among the recursive
// alternatives tried at a position.
type fuzzyBest struct {
	matched bool
	score   float64
	matches []int
}

// consider updates the tracker with a recursive result, keeping it only if it
// matched and outscores the current best.
func (b *fuzzyBest) consider(matched bool, score float64, matches []int) {
	if !matched {
		return
	}
	if !b.matched || score > b.score {
		b.matches = append([]int(nil), matches...)
		b.score = score
	}
	b.matched = true
}

// fuzzyBaseScore is the score a full match starts from before bonuses/penalties.
const fuzzyBaseScore = 100

// fuzzyScore computes the score for a completed match: the base score, the
// leading- and unmatched-letter penalties, and per-match sequential / camelCase
// / separator / first-letter bonuses. Ported from FuzzyMatch.mjs.
func fuzzyScore(matches []int, nextMatch int, str []rune, w fuzzyWeights) float64 {
	outScore := float64(fuzzyBaseScore)
	penalty := w.leadingLetterPenalty * float64(matches[0])
	if penalty < w.maxLeadingLetterPenalty {
		penalty = w.maxLeadingLetterPenalty
	}
	outScore += penalty
	outScore += w.unmatchedLetterPenalty * float64(len(str)-nextMatch)

	for i := range nextMatch {
		currIdx := matches[i]
		if i > 0 && currIdx == matches[i-1]+1 {
			outScore += w.sequentialBonus
		}
		if currIdx > 0 {
			neighbor := str[currIdx-1]
			curr := str[currIdx]
			if neighbor != unicode.ToUpper(neighbor) && curr != unicode.ToLower(curr) {
				outScore += w.camelBonus
			}
			if neighbor == '_' || neighbor == ' ' {
				outScore += w.separatorBonus
			}
		} else {
			outScore += w.firstLetterBonus
		}
	}
	return outScore
}

func fuzzySetIdx(s []int, i, v int) []int {
	for len(s) <= i {
		s = append(s, 0)
	}
	s[i] = v
	return s
}

// calcMatchRanges turns a list of match indices into [start, length] ranges.
func calcMatchRanges(matches []int) [][2]int {
	var ranges [][2]int
	if len(matches) == 0 {
		return ranges
	}
	start := matches[0]
	curr := start
	for _, m := range matches {
		if m == curr || m == curr+1 {
			curr = m
		} else {
			ranges = append(ranges, [2]int{start, curr - start + 1})
			start = m
			curr = m
		}
	}
	ranges = append(ranges, [2]int{start, curr - start + 1})
	return ranges
}
