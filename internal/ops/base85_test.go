package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

const b85Std = "!-u"

// Base85 cases. "Man " -> "9jqo^" is the canonical Wikipedia/spec example
// (hand-verified); the "z" all-zero compression is spec-authoritative for the
// Standard alphabet. Round-trips cover the rest.
func TestBase85Fixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"To Base85: Man", "Man ", "9jqo^",
			core.Recipe{{Op: "To Base85", Args: []any{b85Std, false}}},
		},
		{
			"To Base85: all-zero block -> z", "\x00\x00\x00\x00", "z",
			core.Recipe{{Op: "To Base85", Args: []any{b85Std, false}}},
		},
		{
			"To Base85: include delimiter", "Man ", "<~9jqo^~>",
			core.Recipe{{Op: "To Base85", Args: []any{b85Std, true}}},
		},

		{
			"From Base85: Man", "9jqo^", "Man ",
			core.Recipe{{Op: "From Base85", Args: []any{b85Std, true, "z"}}},
		},
		{
			"From Base85: z -> all-zero block", "z", "\x00\x00\x00\x00",
			core.Recipe{{Op: "From Base85", Args: []any{b85Std, true, "z"}}},
		},
		{
			"From Base85: with delimiters", "<~9jqo^~>", "Man ",
			core.Recipe{{Op: "From Base85", Args: []any{b85Std, true, "z"}}},
		},

		{
			"Base85 round trip", "The quick brown fox jumps", "The quick brown fox jumps",
			core.Recipe{
				{Op: "To Base85", Args: []any{b85Std, false}},
				{Op: "From Base85", Args: []any{b85Std, true, "z"}},
			},
		},
	})
}

func TestBase85Errors(t *testing.T) {
	cases := []struct {
		name, op, input string
		args            []any
	}{
		{"To Base85 rejects wrong-length alphabet", "To Base85", "x", []any{"short", false}},
		{"From Base85 rejects wrong-length alphabet", "From Base85", "abc", []any{"short", true, "z"}},
		{"From Base85 rejects all-zero char in alphabet", "From Base85", "abc", []any{base85Standard, true, "!"}},
		{"From Base85 rejects char not in alphabet", "From Base85", "~~~~~", []any{base85Standard, false, "z"}},
	}
	for _, c := range cases {
		if _, err := runOp(t, c.op, c.input, c.args...); err == nil {
			t.Fatalf("%s: expected an error", c.name)
		}
	}
}

func TestBase85ValueBranches(t *testing.T) {
	if out, err := runOp(t, "To Base85", "", base85Standard, false); err != nil || out != "" {
		t.Fatalf("To Base85(\"\") = %q, %v; want empty", out, err)
	}
	if out, err := runOp(t, "From Base85", "", base85Standard, false, "z"); err != nil || out != "" {
		t.Fatalf("From Base85(\"\") = %q, %v; want empty", out, err)
	}
	// base85AlphabetName returns "" for an unrecognised alphabet.
	if got := base85AlphabetName("not a known 85-char alphabet"); got != "" {
		t.Fatalf("base85AlphabetName(unknown) = %q, want \"\"", got)
	}
}
