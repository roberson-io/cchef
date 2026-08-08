package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
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
		{
			"Simple string global", "foofoo", "barbar",
			core.Recipe{{Op: "Find / Replace", Args: []any{simple("foo"), "bar", true, false, true, false}}},
		},
		{
			"Simple string special chars literal", "a.b.c", "a_b_c",
			core.Recipe{{Op: "Find / Replace", Args: []any{simple("."), "_", true, false, true, false}}},
		},
		{
			"Regex global", "a1b2c3", "a#b#c#",
			core.Recipe{{Op: "Find / Replace", Args: []any{regex(`\d`), "#", true, false, true, false}}},
		},
		{
			"Regex non-global (first only)", "aaa", "baa",
			core.Recipe{{Op: "Find / Replace", Args: []any{regex("a"), "b", false, false, true, false}}},
		},
		{
			"Regex case insensitive", "Hello HELLO", "x x",
			core.Recipe{{Op: "Find / Replace", Args: []any{regex("hello"), "x", true, true, true, false}}},
		},
		{
			"Regex capture groups", "John Smith", "Smith John",
			core.Recipe{{Op: "Find / Replace", Args: []any{regex(`(\w+) (\w+)`), "$2 $1", true, false, true, false}}},
		},
		{
			"Extended escapes", "a\tb", "a b",
			core.Recipe{{Op: "Find / Replace", Args: []any{ext(`\t`), " ", true, false, true, false}}},
		},
		{
			"Extended newline", "a\nb", "a-b",
			core.Recipe{{Op: "Find / Replace", Args: []any{ext(`\n`), "-", true, false, true, false}}},
		},
		{
			"Extended hex escape", "aAb", "a_b",
			core.Recipe{{Op: "Find / Replace", Args: []any{ext(`\x41`), "_", true, false, true, false}}},
		},
		{
			"No match leaves input unchanged", "abc", "abc",
			core.Recipe{{Op: "Find / Replace", Args: []any{regex("z"), "_", false, false, true, false}}},
		},
		// Octal escape in Extended mode (\101 -> 'A').
		{
			"Extended octal escape", "aAb", "a_b",
			core.Recipe{{Op: "Find / Replace", Args: []any{ext(`\101`), "_", true, false, true, false}}},
		},
		// The replacement decodes escape sequences (\n -> newline), matching
		// CyberChef's binaryString handling of the Replace field.
		{
			"Replacement decodes newline", "aXb", "a\nb",
			core.Recipe{{Op: "Find / Replace", Args: []any{regex("X"), `\n`, true, false, true, false}}},
		},
		{
			"Replacement escape coexists with group ref", "John Smith", "Smith\tJohn",
			core.Recipe{{Op: "Find / Replace", Args: []any{regex(`(\w+) (\w+)`), `$2\t$1`, true, false, true, false}}},
		},
		// A lookahead pattern — which RE2 cannot compile — runs via the
		// JavaScript-compatible fallback, group references included.
		{
			"Lookahead find with group ref", "alice@x bob@y", "<alice>@x <bob>@y",
			core.Recipe{{Op: "Find / Replace", Args: []any{regex(`(\w+)(?=@)`), "<$1>", true, false, true, false}}},
		},
		{
			"Lookbehind find, first match only", "$10 and $20", "$N and $20",
			core.Recipe{{Op: "Find / Replace", Args: []any{regex(`(?<=\$)\d+`), "N", false, false, true, false}}},
		},
	})
}

func TestFindReplaceBranches(t *testing.T) {
	regex := func(s string) core.ToggleString { return core.ToggleString{Value: s, Option: "Regex"} }
	// "Dot matches all" adds the s flag so "." spans newlines.
	if out, err := runOp(t, "Find / Replace", "a\nb", regex("a.b"), "X", true, false, true, true); err != nil || out != "X" {
		t.Fatalf("dotall = %q, %v", out, err)
	}
	// An invalid regex pattern is reported as an error.
	if _, err := runOp(t, "Find / Replace", "abc", regex("["), "X", true, false, true, false); err == nil {
		t.Fatal("invalid regex: expected an error")
	}
}
