package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// Hand-verified cases for Filter, Sort and Unique (no upstream fixtures).
func TestFilter(t *testing.T) {
	runCases(t, []opCase{
		{"Filter keep matching", "apple\nbanana\ncherry", "apple\nbanana",
			core.Recipe{{Op: "Filter", Args: []any{"Line feed", "a", false}}}},
		{"Filter invert", "apple\nbanana\ncherry", "cherry",
			core.Recipe{{Op: "Filter", Args: []any{"Line feed", "a", true}}}},
	})
}

func TestSort(t *testing.T) {
	runCases(t, []opCase{
		{"Sort alphabetical", "banana\napple\ncherry", "apple\nbanana\ncherry",
			core.Recipe{{Op: "Sort", Args: []any{"Line feed", false, "Alphabetical (case sensitive)"}}}},
		{"Sort alphabetical reverse", "banana\napple\ncherry", "cherry\nbanana\napple",
			core.Recipe{{Op: "Sort", Args: []any{"Line feed", true, "Alphabetical (case sensitive)"}}}},
		{"Sort case insensitive", "Banana\napple\nCherry", "apple\nBanana\nCherry",
			core.Recipe{{Op: "Sort", Args: []any{"Line feed", false, "Alphabetical (case insensitive)"}}}},
		{"Sort numeric", "10\n2\n1\n20", "1\n2\n10\n20",
			core.Recipe{{Op: "Sort", Args: []any{"Line feed", false, "Numeric"}}}},
		{"Sort length", "ccc\na\nbb", "a\nbb\nccc",
			core.Recipe{{Op: "Sort", Args: []any{"Line feed", false, "Length"}}}},
		{"Sort IP address", "192.168.1.10\n192.168.1.2\n10.0.0.1", "10.0.0.1\n192.168.1.2\n192.168.1.10",
			core.Recipe{{Op: "Sort", Args: []any{"Line feed", false, "IP address"}}}},
	})
}

func TestUnique(t *testing.T) {
	runCases(t, []opCase{
		{"Unique", "a\nb\na\nc\nb", "a\nb\nc",
			core.Recipe{{Op: "Unique", Args: []any{"Line feed", false}}}},
		{"Unique with count", "a\nb\na", "2 a\n1 b",
			core.Recipe{{Op: "Unique", Args: []any{"Line feed", true}}}},
	})
}
