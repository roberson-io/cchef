package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// Offset checker output verified against the CyberChef-server oracle.
func TestOffsetChecker(t *testing.T) {
	runCases(t, []opCase{
		{"common prefix", "hello world\nhello there",
			"<span class='hl5'>hello </span>world\n<span class='hl5'>hello </span>there",
			core.Recipe{{Op: "Offset checker", Args: []any{`\n`}}}},
	})
}
