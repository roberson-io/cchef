package ops

import "regexp"

// Shared helpers for the from-scratch port of the vkbeautify library
// (https://github.com/vkiryukhin/vkBeautify), which backs the CyberChef
// JSON/XML/CSS/SQL Beautify and Minify operations. Each operation lives in its own
// <op>.go; the pieces genuinely shared across them live here.

// jsWSChars is the set of characters JavaScript's \s matches: ASCII whitespace
// plus the Unicode space separators, line separators, NBSP and BOM. Go's RE2 \s is
// narrower ([\t\n\f\r ]), so the vkbeautify ports use this class to stay
// byte-for-byte compatible with the original library's regexes.
const jsWSChars = "\t\n\x0b\f\r \u00a0\u1680\u2000-\u200a\u2028\u2029\u202f\u205f\u3000\ufeff"

// jsWSRun matches a run of one or more JS-whitespace characters (JS /\s{1,}/).
var jsWSRun = regexp.MustCompile("[" + jsWSChars + "]+")

// xmlTagWSRe matches whitespace (JS \s) between a closing '>' and an opening '<',
// shared by XML Beautify and XML Minify (vkbeautify's />\s{0,}</ -> "><").
var xmlTagWSRe = regexp.MustCompile(">[" + jsWSChars + "]*<")

// vkNumericStep matches an indent step for which JS's parseInt(step) is not NaN
// (leading JS whitespace, optional sign, then a digit). vkbeautify then falls
// through a numeric switch that never matches a string, so createShiftArr uses the
// 4-space default for such steps.
var vkNumericStep = regexp.MustCompile("^[" + jsWSChars + "]*[+-]?[0-9]")

// createShiftArr builds the indentation ladder used by XML/CSS Beautify:
// ["\n", "\n"+unit, "\n"+unit+unit, ...] with 101 entries. Ported from
// vkbeautify.createShiftArr, including its numeric-step quirk.
func createShiftArr(step string) []string {
	space := step
	if vkNumericStep.MatchString(step) {
		space = "    "
	}
	shift := make([]string, 1, 101)
	shift[0] = "\n"
	for ix := range 100 {
		shift = append(shift, shift[ix]+space)
	}
	return shift
}
