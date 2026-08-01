// Package lodashcase converts identifiers between camel, kebab and snake case
// the way lodash does.
//
// Splitting a name into words is the whole difficulty: lodash breaks on case
// changes, on the boundary between letters and digits, and on runs of capitals
// followed by a lowercase letter, so HTTPRequest2 is HTTP, Request, 2. The case
// operations rely on that, and [ReplaceVariableNames] applies it to the
// identifiers in a piece of source while leaving everything else alone.
package lodashcase

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/dlclark/regexp2"
)

// Shared from-scratch port of lodash's case helpers, backing To Snake / Camel /
// Kebab case. CyberChef wraps lodash's snakeCase/camelCase/kebabCase; each of
// those is createCompounder(words(deburr(string).replace(apos, ""))) with a
// per-case join. The word splitter's Unicode regex uses lookahead, which Go's RE2
// cannot express, so it is run with github.com/dlclark/regexp2 (already a
// dependency). Reduced fidelity, by design: lodash's regex is UTF-16-oriented, so
// astral characters (emoji, surrogate pairs) may split differently; BMP text is
// byte-for-byte identical.

// deburrLetters maps Latin-1 Supplement and Latin Extended-A letters to basic
// Latin, transcribed from lodash/_deburrLetter.js.
var deburrLetters = map[string]string{
	"\u00c0": "A",
	"\u00c1": "A",
	"\u00c2": "A",
	"\u00c3": "A",
	"\u00c4": "A",
	"\u00c5": "A",
	"\u00e0": "a",
	"\u00e1": "a",
	"\u00e2": "a",
	"\u00e3": "a",
	"\u00e4": "a",
	"\u00e5": "a",
	"\u00c7": "C",
	"\u00e7": "c",
	"\u00d0": "D",
	"\u00f0": "d",
	"\u00c8": "E",
	"\u00c9": "E",
	"\u00ca": "E",
	"\u00cb": "E",
	"\u00e8": "e",
	"\u00e9": "e",
	"\u00ea": "e",
	"\u00eb": "e",
	"\u00cc": "I",
	"\u00cd": "I",
	"\u00ce": "I",
	"\u00cf": "I",
	"\u00ec": "i",
	"\u00ed": "i",
	"\u00ee": "i",
	"\u00ef": "i",
	"\u00d1": "N",
	"\u00f1": "n",
	"\u00d2": "O",
	"\u00d3": "O",
	"\u00d4": "O",
	"\u00d5": "O",
	"\u00d6": "O",
	"\u00d8": "O",
	"\u00f2": "o",
	"\u00f3": "o",
	"\u00f4": "o",
	"\u00f5": "o",
	"\u00f6": "o",
	"\u00f8": "o",
	"\u00d9": "U",
	"\u00da": "U",
	"\u00db": "U",
	"\u00dc": "U",
	"\u00f9": "u",
	"\u00fa": "u",
	"\u00fb": "u",
	"\u00fc": "u",
	"\u00dd": "Y",
	"\u00fd": "y",
	"\u00ff": "y",
	"\u00c6": "Ae",
	"\u00e6": "ae",
	"\u00de": "Th",
	"\u00fe": "th",
	"\u00df": "ss",
	"\u0100": "A",
	"\u0102": "A",
	"\u0104": "A",
	"\u0101": "a",
	"\u0103": "a",
	"\u0105": "a",
	"\u0106": "C",
	"\u0108": "C",
	"\u010a": "C",
	"\u010c": "C",
	"\u0107": "c",
	"\u0109": "c",
	"\u010b": "c",
	"\u010d": "c",
	"\u010e": "D",
	"\u0110": "D",
	"\u010f": "d",
	"\u0111": "d",
	"\u0112": "E",
	"\u0114": "E",
	"\u0116": "E",
	"\u0118": "E",
	"\u011a": "E",
	"\u0113": "e",
	"\u0115": "e",
	"\u0117": "e",
	"\u0119": "e",
	"\u011b": "e",
	"\u011c": "G",
	"\u011e": "G",
	"\u0120": "G",
	"\u0122": "G",
	"\u011d": "g",
	"\u011f": "g",
	"\u0121": "g",
	"\u0123": "g",
	"\u0124": "H",
	"\u0126": "H",
	"\u0125": "h",
	"\u0127": "h",
	"\u0128": "I",
	"\u012a": "I",
	"\u012c": "I",
	"\u012e": "I",
	"\u0130": "I",
	"\u0129": "i",
	"\u012b": "i",
	"\u012d": "i",
	"\u012f": "i",
	"\u0131": "i",
	"\u0134": "J",
	"\u0135": "j",
	"\u0136": "K",
	"\u0137": "k",
	"\u0138": "k",
	"\u0139": "L",
	"\u013b": "L",
	"\u013d": "L",
	"\u013f": "L",
	"\u0141": "L",
	"\u013a": "l",
	"\u013c": "l",
	"\u013e": "l",
	"\u0140": "l",
	"\u0142": "l",
	"\u0143": "N",
	"\u0145": "N",
	"\u0147": "N",
	"\u014a": "N",
	"\u0144": "n",
	"\u0146": "n",
	"\u0148": "n",
	"\u014b": "n",
	"\u014c": "O",
	"\u014e": "O",
	"\u0150": "O",
	"\u014d": "o",
	"\u014f": "o",
	"\u0151": "o",
	"\u0154": "R",
	"\u0156": "R",
	"\u0158": "R",
	"\u0155": "r",
	"\u0157": "r",
	"\u0159": "r",
	"\u015a": "S",
	"\u015c": "S",
	"\u015e": "S",
	"\u0160": "S",
	"\u015b": "s",
	"\u015d": "s",
	"\u015f": "s",
	"\u0161": "s",
	"\u0162": "T",
	"\u0164": "T",
	"\u0166": "T",
	"\u0163": "t",
	"\u0165": "t",
	"\u0167": "t",
	"\u0168": "U",
	"\u016a": "U",
	"\u016c": "U",
	"\u016e": "U",
	"\u0170": "U",
	"\u0172": "U",
	"\u0169": "u",
	"\u016b": "u",
	"\u016d": "u",
	"\u016f": "u",
	"\u0171": "u",
	"\u0173": "u",
	"\u0174": "W",
	"\u0175": "w",
	"\u0176": "Y",
	"\u0177": "y",
	"\u0178": "Y",
	"\u0179": "Z",
	"\u017b": "Z",
	"\u017d": "Z",
	"\u017a": "z",
	"\u017c": "z",
	"\u017e": "z",
	"\u0132": "IJ",
	"\u0133": "ij",
	"\u0152": "Oe",
	"\u0153": "oe",
	"\u0149": "'n",
	"\u017f": "s",
}

// reLatin matches the letters deburrLetters covers; reComboMark matches combining
// diacritical marks (removed). Both from lodash/deburr.js.
var (
	reLatin     = regexp.MustCompile(`[\x{00c0}-\x{00d6}\x{00d8}-\x{00f6}\x{00f8}-\x{00ff}\x{0100}-\x{017f}]`)
	reComboMark = regexp.MustCompile(`[\x{0300}-\x{036f}\x{fe20}-\x{fe2f}\x{20d0}-\x{20ff}]`)
	reApos      = regexp.MustCompile(`['\x{2019}]`)
)

// lodashDeburr converts accented Latin letters to basic Latin and strips combining
// marks (lodash.deburr).
func lodashDeburr(s string) string {
	// reLatin matches exactly the code points deburrLetters covers, so the lookup
	// always hits.
	s = reLatin.ReplaceAllStringFunc(s, func(m string) string {
		return deburrLetters[m]
	})
	return reComboMark.ReplaceAllString(s, "")
}

// buildUnicodeWordPattern composes lodash's reUnicodeWord (from
// lodash/_unicodeWords.js) using raw strings so the \uXXXX / \xNN escapes and the
// surrogate ranges reach regexp2 verbatim.
func buildUnicodeWordPattern() string {
	const (
		rsAstralRange    = `\ud800-\udfff`
		rsComboMarks     = `\u0300-\u036f`
		reComboHalfMarks = `\ufe20-\ufe2f`
		rsComboSymbols   = `\u20d0-\u20ff`
		rsDingbatRange   = `\u2700-\u27bf`
		rsLowerRange     = `a-z\xdf-\xf6\xf8-\xff`
		rsMathOpRange    = `\xac\xb1\xd7\xf7`
		rsNonCharRange   = `\x00-\x2f\x3a-\x40\x5b-\x60\x7b-\xbf`
		rsPunctuationRng = `\u2000-\u206f`
		rsSpaceRange     = ` \t\x0b\f\xa0\ufeff\n\r\u2028\u2029\u1680\u180e\u2000\u2001\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u202f\u205f\u3000`
		rsUpperRange     = `A-Z\xc0-\xd6\xd8-\xde`
		rsVarRange       = `\ufe0e\ufe0f`
		rsApos           = `['\u2019]`
		rsZWJ            = `\u200d`
	)
	rsComboRange := rsComboMarks + reComboHalfMarks + rsComboSymbols
	rsBreakRange := rsMathOpRange + rsNonCharRange + rsPunctuationRng + rsSpaceRange

	rsBreak := `[` + rsBreakRange + `]`
	rsCombo := `[` + rsComboRange + `]`
	rsDigits := `\d+`
	rsDingbat := `[` + rsDingbatRange + `]`
	rsLower := `[` + rsLowerRange + `]`
	rsMisc := `[^` + rsAstralRange + rsBreakRange + rsDigits + rsDingbatRange + rsLowerRange + rsUpperRange + `]`
	rsFitz := `\ud83c[\udffb-\udfff]`
	rsModifier := `(?:` + rsCombo + `|` + rsFitz + `)`
	rsNonAstral := `[^` + rsAstralRange + `]`
	rsRegional := `(?:\ud83c[\udde6-\uddff]){2}`
	rsSurrPair := `[\ud800-\udbff][\udc00-\udfff]`
	rsUpper := `[` + rsUpperRange + `]`

	rsMiscLower := `(?:` + rsLower + `|` + rsMisc + `)`
	rsMiscUpper := `(?:` + rsUpper + `|` + rsMisc + `)`
	rsOptContrLower := `(?:` + rsApos + `(?:d|ll|m|re|s|t|ve))?`
	rsOptContrUpper := `(?:` + rsApos + `(?:D|LL|M|RE|S|T|VE))?`
	reOptMod := rsModifier + `?`
	rsOptVar := `[` + rsVarRange + `]?`
	rsOptJoin := `(?:` + rsZWJ + `(?:` + strings.Join([]string{rsNonAstral, rsRegional, rsSurrPair}, `|`) + `)` + rsOptVar + reOptMod + `)*`
	rsOrdLower := `\d*(?:1st|2nd|3rd|(?![123])\dth)(?=\b|[A-Z_])`
	rsOrdUpper := `\d*(?:1ST|2ND|3RD|(?![123])\dTH)(?=\b|[a-z_])`
	rsSeq := rsOptVar + reOptMod + rsOptJoin
	rsEmoji := `(?:` + strings.Join([]string{rsDingbat, rsRegional, rsSurrPair}, `|`) + `)` + rsSeq

	return strings.Join([]string{
		rsUpper + `?` + rsLower + `+` + rsOptContrLower + `(?=` + strings.Join([]string{rsBreak, rsUpper, `$`}, `|`) + `)`,
		rsMiscUpper + `+` + rsOptContrUpper + `(?=` + strings.Join([]string{rsBreak, rsUpper + rsMiscLower, `$`}, `|`) + `)`,
		rsUpper + `?` + rsMiscLower + `+` + rsOptContrLower,
		rsUpper + `+` + rsOptContrUpper,
		rsOrdUpper,
		rsOrdLower,
		rsDigits,
		rsEmoji,
	}, `|`)
}

var (
	reHasUnicodeWord = regexp.MustCompile(`[a-z][A-Z]|[A-Z]{2}[a-z]|[0-9][a-zA-Z]|[a-zA-Z][0-9]|[^a-zA-Z0-9 ]`)
	reASCIIWord      = regexp.MustCompile(`[^\x00-\x2f\x3a-\x40\x5b-\x60\x7b-\x7f]+`)
	reUnicodeWord    = regexp2.MustCompile(buildUnicodeWordPattern(), regexp2.None)
	// reVarToken tokenises code for the "context aware" mode (lib/Code.mjs).
	reVarToken = regexp.MustCompile(`(?i)\\"|"(?:\\"|[^"])*"|(\b[a-z0-9\-_]+\b)`)
)

// lodashWords splits a string into words (lodash.words): the ASCII path for plain
// input, the Unicode (lookahead) path otherwise.
func lodashWords(s string) []string {
	if reHasUnicodeWord.MatchString(s) {
		return regexp2FindAll(reUnicodeWord, s)
	}
	if m := reASCIIWord.FindAllString(s, -1); m != nil {
		return m
	}
	return []string{}
}

func regexp2FindAll(re *regexp2.Regexp, s string) []string {
	out := []string{}
	m, _ := re.FindStringMatch(s)
	for m != nil {
		out = append(out, m.String())
		m, _ = re.FindNextMatch(m)
	}
	return out
}

// lodashCompound is lodash's createCompounder: split the deburred, apostrophe-
// stripped input into words and fold them with cb.
func lodashCompound(s string, cb func(result, word string, index int) string) string {
	words := lodashWords(reApos.ReplaceAllString(lodashDeburr(s), ""))
	result := ""
	for i, w := range words {
		result = cb(result, w, i)
	}
	return result
}

// SnakeCase joins the words of s with underscores, all lower case.
func SnakeCase(s string) string {
	return lodashCompound(s, func(result, word string, index int) string {
		sep := ""
		if index != 0 {
			sep = "_"
		}
		return result + sep + strings.ToLower(word)
	})
}

// KebabCase joins the words of s with hyphens, all lower case.
func KebabCase(s string) string {
	return lodashCompound(s, func(result, word string, index int) string {
		sep := ""
		if index != 0 {
			sep = "-"
		}
		return result + sep + strings.ToLower(word)
	})
}

// CamelCase joins the words of s with each after the first capitalised.
func CamelCase(s string) string {
	return lodashCompound(s, func(result, word string, index int) string {
		word = strings.ToLower(word)
		if index != 0 {
			word = upperFirst(word)
		}
		return result + word
	})
}

// upperFirst upper-cases the first code point (lodash.upperFirst).
func upperFirst(s string) string {
	if s == "" {
		return ""
	}
	r, size := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + s[size:]
}

// ReplaceVariableNames applies replacer to identifier-like tokens, leaving quoted
// strings and escaped quotes untouched (a port of lib/Code.mjs).
func ReplaceVariableNames(input string, replacer func(string) string) string {
	var b strings.Builder
	last := 0
	for _, m := range reVarToken.FindAllStringSubmatchIndex(input, -1) {
		b.WriteString(input[last:m[0]])
		if m[2] >= 0 { // the identifier capture group matched
			b.WriteString(replacer(input[m[2]:m[3]]))
		} else {
			b.WriteString(input[m[0]:m[1]])
		}
		last = m[1]
	}
	b.WriteString(input[last:])
	return b.String()
}
