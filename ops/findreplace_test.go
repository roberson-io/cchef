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

// TestParseEscapedChars covers the escape-sequence decoder's less-common arms
// (backslash, \x, \u, \u{...}) directly.
func TestParseEscapedChars(t *testing.T) {
	cases := map[string]string{
		`\\`:        "\\",
		`\x41`:      "A",
		`\u0041`:    "A",
		`\u{1F600}`: "\U0001F600",
		`\a`:        "\x07",
	}
	for in, want := range cases {
		if got := parseEscapedChars(in); got != want {
			t.Errorf("parseEscapedChars(%q) = %q, want %q", in, got, want)
		}
	}
}
