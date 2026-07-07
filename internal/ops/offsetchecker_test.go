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

// TestOffsetCheckerTrailingMatch covers the span-close when a match reaches the
// end of a shorter sample. The CyberChef-server oracle rejects these inputs, but
// the output matches CyberChef's OffsetChecker.mjs algorithm exactly (including
// its quirky trailing "</span>").
func TestOffsetCheckerTrailingMatch(t *testing.T) {
	out, err := runOp(t, "Offset checker", "abc\nxb", `\n`)
	if err != nil {
		t.Fatal(err)
	}
	want := "a<span class='hl5'>b</span>c</span>\nx<span class='hl5'>b</span></span>"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}
