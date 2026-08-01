package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

// Transcribed from ../CyberChef/tests/operations/tests/UnescapeUnicodeCharacters.mjs.
func TestUnescapeUnicodeFixtures(t *testing.T) {
	unesc := func(prefix string) core.Recipe {
		return core.Recipe{{Op: "Unescape Unicode Characters", Args: []any{prefix}}}
	}
	runCases(t, []opCase{
		{"backslash-u 4-digit BMP", "\\u03c3\\u03bf\\u03c5", "σου", unesc("\\u")},
		{"percent-u 4-digit BMP", "%u03c3%u03bf%u03c5", "σου", unesc("%u")},
		{"U+ 4-digit BMP", "U+0041", "A", unesc("U+")},
		{"U+ 5-digit astral emoji", "U+1F600", "\U0001F600", unesc("U+")},
		{"U+ 6-digit zero-padded", "U+000041", "A", unesc("U+")},
		{"U+ mixed lengths", "U+0041 U+1F600 U+000042", "A \U0001F600 B", unesc("U+")},
		{"passthrough with no matches", "hello world", "hello world", unesc("\\u")},
	})
}

// Escape Unicode Characters has no upstream fixture file; these cases are
// verified against the CyberChef-server oracle. Args: prefix, encode-all,
// padding, uppercase-hex.
func TestEscapeUnicode(t *testing.T) {
	esc := func(prefix string, all bool, pad float64, upper bool) core.Recipe {
		return core.Recipe{{Op: "Escape Unicode Characters", Args: []any{prefix, all, pad, upper}}}
	}
	runCases(t, []opCase{
		{"default backslash-u", "σου", "\\u03C3\\u03BF\\u03C5", esc("\\u", false, 4, true)},
		{"percent-u prefix", "σου", "%u03C3%u03BF%u03C5", esc("%u", false, 4, true)},
		{"U+ prefix", "σου", "U+03C3U+03BFU+03C5", esc("U+", false, 4, true)},
		{"ASCII kept, non-ASCII escaped", "abcσ", "abc\\u03C3", esc("\\u", false, 4, true)},
		{"encode all chars", "abc", "\\u0061\\u0062\\u0063", esc("\\u", true, 4, true)},
		{"padding 2 (no truncation)", "σ", "\\u3C3", esc("\\u", false, 2, true)},
		{"padding 6", "σ", "\\u0003C3", esc("\\u", false, 6, true)},
		{"lowercase hex", "σ", "\\u03c3", esc("\\u", false, 4, false)},
		{"astral emoji as surrogate pair", "😀", "\\uD83D\\uDE00", esc("\\u", false, 4, true)},
		{"astral emoji with U+ prefix", "😀", "U+D83DU+DE00", esc("U+", false, 4, true)},
	})
}
