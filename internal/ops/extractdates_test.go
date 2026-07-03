package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// TestExtractDatesOracle checks Extract dates against CyberChef-server output
// (v11.2.0); CyberChef ships no fixture file for it. The op matches yyyy-mm-dd,
// dd/mm/yyyy and mm/dd/yyyy date shapes (any of - / . as separators), in order.
func TestExtractDatesOracle(t *testing.T) {
	const in = "Meeting on 2024-02-20 and 01/04/1999, also 12/31/2020."
	runCases(t, []opCase{
		{"Extract dates", in, "2024-02-20\n01/04/1999\n12/31/2020",
			core.Recipe{{Op: "Extract dates", Args: []any{false}}}},
		{"Extract dates: display total", in, "Total found: 3\n\n2024-02-20\n01/04/1999\n12/31/2020",
			core.Recipe{{Op: "Extract dates", Args: []any{true}}}},
	})
}
