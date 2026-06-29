package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// Regular expression outputs verified against the CyberChef-server oracle.
func TestRegularExpression(t *testing.T) {
	runCases(t, []opCase{
		{"list matches", "a1b22c333", "1\n22\n333",
			core.Recipe{{Op: "Regular expression", Args: []any{"", `\d+`, false, true, false, false, false, false, "List matches"}}}},
		{"list with groups", "cat bat hat", "cat\n  Group 1: c\nbat\n  Group 1: b\nhat\n  Group 1: h",
			core.Recipe{{Op: "Regular expression", Args: []any{"", "(.)at", false, true, false, false, false, false, "List matches with capture groups"}}}},
		{"highlight", "cat bat", "<span class='hl2' title='Offset: 0\n" +
			"Groups:\n\t1: c\n'>cat</span> <span class='hl1' title='Offset: 4\nGroups:\n\t1: b\n'>bat</span>",
			core.Recipe{{Op: "Regular expression", Args: []any{"", "(.)at", false, true, false, false, false, false, "Highlight matches"}}}},
		{"no match escapes input", "<a>&b", "&lt;a&gt;&amp;b",
			core.Recipe{{Op: "Regular expression", Args: []any{"", "x", false, true, false, false, false, false, "Highlight matches"}}}},
	})
}
