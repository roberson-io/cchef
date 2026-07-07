package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// Regular expression outputs verified against the CyberChef-server oracle.
func TestRegularExpression(t *testing.T) {
	runCases(t, []opCase{
		{
			"list matches", "a1b22c333", "1\n22\n333",
			core.Recipe{{Op: "Regular expression", Args: []any{"", `\d+`, false, true, false, false, false, false, "List matches"}}},
		},
		{
			"list with groups", "cat bat hat", "cat\n  Group 1: c\nbat\n  Group 1: b\nhat\n  Group 1: h",
			core.Recipe{{Op: "Regular expression", Args: []any{"", "(.)at", false, true, false, false, false, false, "List matches with capture groups"}}},
		},
		{
			"highlight", "cat bat", "<span class='hl2' title='Offset: 0\n" +
				"Groups:\n\t1: c\n'>cat</span> <span class='hl1' title='Offset: 4\nGroups:\n\t1: b\n'>bat</span>",
			core.Recipe{{Op: "Regular expression", Args: []any{"", "(.)at", false, true, false, false, false, false, "Highlight matches"}}},
		},
		{
			"no match escapes input", "<a>&b", "&lt;a&gt;&amp;b",
			core.Recipe{{Op: "Regular expression", Args: []any{"", "x", false, true, false, false, false, false, "Highlight matches"}}},
		},
		{
			"list capture groups", "2024-03-09 and 1999-12-31", "2024\n03\n09\n1999\n12\n31",
			core.Recipe{{Op: "Regular expression", Args: []any{"", `(\d{4})-(\d{2})-(\d{2})`, false, true, false, false, false, false, "List capture groups"}}},
		},
		{
			"list matches with capture groups", "2024-03-09", "2024-03-09\n  Group 1: 2024\n  Group 2: 03\n  Group 3: 09",
			core.Recipe{{Op: "Regular expression", Args: []any{"", `(\d{4})-(\d{2})-(\d{2})`, false, true, false, false, false, false, "List matches with capture groups"}}},
		},
		{
			"list matches with total", "a1 b22 c333", "Total found: 3\n\n1\n22\n333",
			core.Recipe{{Op: "Regular expression", Args: []any{"", `\d+`, false, true, false, false, false, true, "List matches"}}},
		},
		// Case-insensitive flag (arg 3): "cat" matches every casing.
		{
			"case insensitive", "CAT cat Cat", "CAT\ncat\nCat",
			core.Recipe{{Op: "Regular expression", Args: []any{"", "cat", true, true, false, false, false, false, "List matches"}}},
		},
		// Dot-matches-all flag (arg 5): "." spans the newline, matching the whole input.
		{
			"dot matches all", "aXb\ncXd", "aXb\ncXd",
			core.Recipe{{Op: "Regular expression", Args: []any{"", "a.*d", false, true, true, false, false, false, "List matches"}}},
		},
		// An empty regex is a no-op that just HTML-escapes the input.
		{
			"empty regex escapes input", "a<b", "a&lt;b",
			core.Recipe{{Op: "Regular expression", Args: []any{"", "", true, true, false, false, false, false, "Highlight matches"}}},
		},
	})
}

// TestRegularExpressionBranches covers the invalid-regex error and the
// display-total prefix.
func TestRegularExpressionBranches(t *testing.T) {
	if _, err := runOp(t, "Regular expression", "abc", "", "[", false, true, false, false, false, false, "List matches"); err == nil {
		t.Fatal("expected an error for an invalid regex")
	}
	// "Highlight matches" with Display total prepends the count in regexHighlight.
	out, err := runOp(t, "Regular expression", "a1b2", "", `\d`, false, true, false, false, false, true, "Highlight matches")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Total found: 2") {
		t.Fatalf("display total: %q", out)
	}
}
