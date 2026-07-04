package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// TestFuzzyMatch uses CyberChef's official browser fixture (tests/browser/02_ops.js):
// input "test input", search "tein", default weights, where the last <b> element is "in".
// The full output is derived from the algorithm: match indices [0,1,5,6] -> ranges
// [[0,2],[5,2]] -> "<span class=\"hl1\"><b>te</b>st <b>in</b></span>put".
func TestFuzzyMatch(t *testing.T) {
	runCases(t, []opCase{
		{
			"fixture", "test input", `<span class="hl1"><b>te</b>st <b>in</b></span>put`,
			core.Recipe{{Op: "Fuzzy Match", Args: []any{"tein", 15.0, 30.0, 30.0, 15.0, -5.0, -15.0, -1.0}}},
		},
	})
}

// The last <b> element must be "in", matching the fixture's "b:last-child" assertion.
func TestFuzzyMatchLastBold(t *testing.T) {
	out, err := runOp(t, "Fuzzy Match", "test input", "tein", 15.0, 30.0, 30.0, 15.0, -5.0, -15.0, -1.0)
	if err != nil {
		t.Fatal(err)
	}
	last := strings.LastIndex(out, "<b>")
	end := strings.Index(out[last:], "</b>")
	if got := out[last+3 : last+end]; got != "in" {
		t.Errorf("b:last-child = %q, want %q", got, "in")
	}
}
