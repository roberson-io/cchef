package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// Offset checker output verified against the CyberChef-server oracle.
func TestOffsetChecker(t *testing.T) {
	runCases(t, []opCase{
		{
			"common prefix", "hello world\nhello there",
			"<span class='hl5'>hello </span>world\n<span class='hl5'>hello </span>there",
			core.Recipe{{Op: "Offset checker", Args: []any{`\n`}}},
		},
		// Varied-length samples exercise the span-close paths when a sample runs
		// out mid-match or the match reaches a sample's end. The doubled </span>
		// is a CyberChef quirk this port reproduces (oracle-verified).
		{
			"second sample shorter", "ABCDEF\nABC",
			"<span class='hl5'>ABC</span>DEF\n<span class='hl5'>ABC</span></span>",
			core.Recipe{{Op: "Offset checker", Args: []any{`\n`}}},
		},
		{
			"identical samples match through end", "match\nmatch",
			"<span class='hl5'>match</span></span>\n<span class='hl5'>match</span></span>",
			core.Recipe{{Op: "Offset checker", Args: []any{`\n`}}},
		},
		{
			"first sample shorter", "ABC\nABCDEF",
			"<span class='hl5'>ABC</span></span>\n<span class='hl5'>ABC</span>DEF",
			core.Recipe{{Op: "Offset checker", Args: []any{`\n`}}},
		},
	})

	// Fewer than two samples cannot be compared.
	if _, err := runOp(t, "Offset checker", "onlyone", `\n`); err == nil {
		t.Error("single sample: expected error")
	}
}
