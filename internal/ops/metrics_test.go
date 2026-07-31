package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// Levenshtein cases transcribed from CyberChef LevenshteinDistance.mjs.
func TestLevenshteinDistance(t *testing.T) {
	runCases(t, []opCase{
		{
			"kitten/sitting", "kitten\nsitting", "3",
			core.Recipe{{Op: "Levenshtein Distance", Args: []any{`\n`, 1, 1, 1}}},
		},
		{
			"saturday/sunday", "saturday\nsunday", "3",
			core.Recipe{{Op: "Levenshtein Distance", Args: []any{`\n`, 1, 1, 1}}},
		},
		{
			"substitution cost 2", "kitten\nsitting", "5",
			core.Recipe{{Op: "Levenshtein Distance", Args: []any{`\n`, 1, 1, 2}}},
		},
	})
}

// Wrap cases transcribed from CyberChef Wrap.mjs.
func TestWrap(t *testing.T) {
	runCases(t, []opCase{
		{
			"wrap at 64", strings.Repeat("A", 128), strings.Repeat("A", 64) + "\n" + strings.Repeat("A", 64),
			core.Recipe{{Op: "Wrap", Args: []any{64}}},
		},
		{
			"wrap at 10", strings.Repeat("1234567890", 3), "1234567890\n1234567890\n1234567890",
			core.Recipe{{Op: "Wrap", Args: []any{10}}},
		},
		{
			"wrap empty", "", "",
			core.Recipe{{Op: "Wrap", Args: []any{64}}},
		},
		// A width above Go's regexp repeat cap (1000) must still wrap, not panic.
		{
			"wrap at large width", strings.Repeat("A", 2500),
			strings.Repeat("A", 2000) + "\n" + strings.Repeat("A", 500),
			core.Recipe{{Op: "Wrap", Args: []any{2000}}},
		},
	})
}

// TestWrapValidation covers the Line Width bounds added upstream (CyberChef
// #2606); previously these inputs panicked in cchef via a bad regexp repeat.
func TestWrapValidation(t *testing.T) {
	for _, width := range []float64{1.1, 0, -1, 65537} {
		if _, err := core.CoerceArgs(Wrap{}.Args(), []any{width}); err == nil {
			t.Fatalf("expected line width %v to be rejected by the declared bounds", width)
		}
	}
}

// Hamming cases: "karolin"/"kathrin" is the classic example (3 byte / 9 bit
// differences); verified against the CyberChef-server oracle.
func TestHammingDistance(t *testing.T) {
	runCases(t, []opCase{
		{
			"byte distance", "karolin\n\nkathrin", "3",
			core.Recipe{{Op: "Hamming Distance", Args: []any{`\n\n`, "Byte", "Raw string"}}},
		},
		{
			"bit distance", "karolin\n\nkathrin", "9",
			core.Recipe{{Op: "Hamming Distance", Args: []any{`\n\n`, "Bit", "Raw string"}}},
		},
		{
			"hex byte distance", "0102\n\n0103", "1",
			core.Recipe{{Op: "Hamming Distance", Args: []any{`\n\n`, "Byte", "Hex"}}},
		},
	})
}

// TestWrapLineWidthInteger pins the integer check CyberChef declares on the
// line width.
func TestWrapLineWidthInteger(t *testing.T) {
	op, _ := core.Default.Get("Wrap")
	_, err := core.CoerceArgs(op.Args(), []any{20.5})
	if err == nil {
		t.Fatal("a fractional line width was accepted")
	}
	if want := "Line Width must be an integer."; err.Error() != want {
		t.Errorf("got %q, want %q", err.Error(), want)
	}
}
