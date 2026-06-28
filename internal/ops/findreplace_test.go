package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// Hand-verified cases for Find / Replace (no upstream fixtures).
// Args: Find{value,option}, Replace, Global, CaseInsensitive, Multiline, DotAll.
func TestFindReplace(t *testing.T) {
	simple := func(v string) core.ToggleString { return core.ToggleString{Value: v, Option: "Simple string"} }
	regex := func(v string) core.ToggleString { return core.ToggleString{Value: v, Option: "Regex"} }
	ext := func(v string) core.ToggleString {
		return core.ToggleString{Value: v, Option: "Extended (\\n, \\t, \\x...)"}
	}

	runCases(t, []opCase{
		{"Simple string global", "foofoo", "barbar",
			core.Recipe{{Op: "Find / Replace", Args: []any{simple("foo"), "bar", true, false, true, false}}}},
		{"Simple string special chars literal", "a.b.c", "a_b_c",
			core.Recipe{{Op: "Find / Replace", Args: []any{simple("."), "_", true, false, true, false}}}},
		{"Regex global", "a1b2c3", "a#b#c#",
			core.Recipe{{Op: "Find / Replace", Args: []any{regex(`\d`), "#", true, false, true, false}}}},
		{"Regex non-global (first only)", "aaa", "baa",
			core.Recipe{{Op: "Find / Replace", Args: []any{regex("a"), "b", false, false, true, false}}}},
		{"Regex case insensitive", "Hello HELLO", "x x",
			core.Recipe{{Op: "Find / Replace", Args: []any{regex("hello"), "x", true, true, true, false}}}},
		{"Regex capture groups", "John Smith", "Smith John",
			core.Recipe{{Op: "Find / Replace", Args: []any{regex(`(\w+) (\w+)`), "$2 $1", true, false, true, false}}}},
		{"Extended escapes", "a\tb", "a b",
			core.Recipe{{Op: "Find / Replace", Args: []any{ext(`\t`), " ", true, false, true, false}}}},
		{"Extended newline", "a\nb", "a-b",
			core.Recipe{{Op: "Find / Replace", Args: []any{ext(`\n`), "-", true, false, true, false}}}},
		{"Extended hex escape", "aAb", "a_b",
			core.Recipe{{Op: "Find / Replace", Args: []any{ext(`\x41`), "_", true, false, true, false}}}},
		{"No match leaves input unchanged", "abc", "abc",
			core.Recipe{{Op: "Find / Replace", Args: []any{regex("z"), "_", false, false, true, false}}}},
		// In Extended mode an unrecognised escape becomes the literal char (\d -> d).
		{"Extended literal fallback", "d1d", "X1X",
			core.Recipe{{Op: "Find / Replace", Args: []any{ext(`\d`), "X", true, false, true, false}}}},
	})
}
