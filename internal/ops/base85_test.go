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
