package ops

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/dlclark/regexp2"
)

// The walk and the orderings the extractor operations share. Ported from
// CyberChef's src/core/lib/Extract.mjs and the comparators in lib/Sort.mjs that
// it uses.

// extractSearch returns every match of re in input, in the order they occur.
// Matches that remove also matches are left out; less, when given, orders what
// is left; unique reduces it to one of each. The pattern is run under regexp2
// rather than the standard library, since several of the extractors need
// lookaround or backreferences.
func extractSearch(
	input string,
	re *regexp2.Regexp,
	remove *regexp2.Regexp,
	less func(a, b string) bool,
	unique bool,
) []string {
	var results []string

	match, err := re.FindStringMatch(input)
	for err == nil && match != nil {
		text := match.String()
		if remove == nil || !matchesRegexp2(remove, text) {
			results = append(results, text)
		}
		match, err = re.FindNextMatch(match)
	}

	if less != nil {
		sort.SliceStable(results, func(i, j int) bool { return less(results[i], results[j]) })
	}
	if unique {
		results = uniqueStrings(results)
	}
	return results
}

// matchesRegexp2 reports whether re matches s anywhere.
func matchesRegexp2(re *regexp2.Regexp, s string) bool {
	ok, err := re.MatchString(s)
	return err == nil && ok
}

// uniqueStrings keeps the first of each repeated value, leaving the order of
// what remains as it was.
func uniqueStrings(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := in[:0]
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// extractResult renders what was found, with a count before it when asked for.
func extractResult(found []string, displayTotal bool) string {
	joined := strings.Join(found, "\n")
	if displayTotal {
		return fmt.Sprintf("Total found: %d\n\n%s", len(found), joined)
	}
	return joined
}

// caseInsensitiveLess orders two strings ignoring the case of either.
func caseInsensitiveLess(a, b string) bool {
	return localeCompareASCII(strings.ToLower(a), strings.ToLower(b)) < 0
}

// extractIPLess orders two addresses by the number their four parts make, so
// that 9.0.0.0 comes before 10.0.0.0 rather than after it as text would have it.
// Anything the four parts do not read as a number sorts after everything that
// does, and among themselves as text.
//
// Note that a part is read as an ordinary decimal number whatever it looks like,
// so the leading zero of an octal address counts for nothing in the ordering
// even though the address itself was matched as octal.
func extractIPLess(a, b string) bool {
	av, aok := extractIPValue(a)
	bv, bok := extractIPValue(b)
	switch {
	case !aok && bok:
		return false
	case aok && !bok:
		return true
	case !aok && !bok:
		return localeCompareASCII(a, b) < 0
	}
	return av < bv
}

// extractIPValue reads a dotted address as the number its parts make, or reports
// that it does not read as one. A part that is not a number, and an address of
// any length other than four parts, both make the whole thing unreadable — as
// the arithmetic over its parts would give a value that is not a number.
func extractIPValue(s string) (float64, bool) {
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return 0, false
	}
	var value float64
	for i, weight := range [4]float64{0x1000000, 0x10000, 0x100, 1} {
		n, err := strconv.ParseFloat(strings.TrimSpace(parts[i]), 64)
		if err != nil || math.IsNaN(n) {
			return 0, false
		}
		value += n * weight
	}
	return value, true
}
