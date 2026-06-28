package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

func TestSplitOp(t *testing.T) {
	runCases(t, []opCase{
		{"Split comma to newline", "a,b,c", "a\nb\nc",
			core.Recipe{{Op: "Split", Args: []any{",", "\\n"}}}},
		{"Split space to comma", "a b c", "a,b,c",
			core.Recipe{{Op: "Split", Args: []any{" ", ","}}}},
	})
}

func TestCountOccurrences(t *testing.T) {
	simple := func(v string) core.ToggleString { return core.ToggleString{Value: v, Option: "Simple string"} }
	regex := func(v string) core.ToggleString { return core.ToggleString{Value: v, Option: "Regex"} }
	runCases(t, []opCase{
		{"Count simple", "abcabcabc", "3",
			core.Recipe{{Op: "Count occurrences", Args: []any{simple("abc")}}}},
		{"Count regex", "abcabcabc", "6",
			core.Recipe{{Op: "Count occurrences", Args: []any{regex("[ac]")}}}},
		{"Count empty search", "abc", "0",
			core.Recipe{{Op: "Count occurrences", Args: []any{simple("")}}}},
	})
}

func TestLineNumbers(t *testing.T) {
	runCases(t, []opCase{
		{"Add line numbers", "a\nb\nc", "1 a\n2 b\n3 c",
			core.Recipe{{Op: "Add line numbers", Args: []any{0}}}},
		{"Add line numbers offset", "a\nb", "6 a\n7 b",
			core.Recipe{{Op: "Add line numbers", Args: []any{5}}}},
		{"Remove line numbers", "1 a\n2 b\n3 c", "a\nb\nc",
			core.Recipe{{Op: "Remove line numbers"}}},
	})
}

func TestAlternatingCaps(t *testing.T) {
	runCases(t, []opCase{
		{"Alternating caps", "hello world", "hElLo WoRlD",
			core.Recipe{{Op: "Alternating Caps"}}},
	})
}

func TestRemoveANSI(t *testing.T) {
	runCases(t, []opCase{
		{"Remove ANSI", "\x1b[31mred\x1b[0m text", "red text",
			core.Recipe{{Op: "Remove ANSI Escape Codes"}}},
	})
}

func TestExpandAlphabetRange(t *testing.T) {
	runCases(t, []opCase{
		{"Expand a-e no delim", "a-e", "abcde",
			core.Recipe{{Op: "Expand alphabet range", Args: []any{""}}}},
		{"Expand a-c comma", "a-c", "a,b,c",
			core.Recipe{{Op: "Expand alphabet range", Args: []any{","}}}},
	})
}
