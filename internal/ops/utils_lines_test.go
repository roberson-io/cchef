package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// Hand-verified cases for Filter, Sort and Unique (no upstream fixtures).
func TestFilter(t *testing.T) {
	runCases(t, []opCase{
		{
			"Filter keep matching", "apple\nbanana\ncherry", "apple\nbanana",
			core.Recipe{{Op: "Filter", Args: []any{"Line feed", "a", false}}},
		},
		{
			"Filter invert", "apple\nbanana\ncherry", "cherry",
			core.Recipe{{Op: "Filter", Args: []any{"Line feed", "a", true}}},
		},
	})
}

func TestSort(t *testing.T) {
	runCases(t, []opCase{
		{
			"Sort alphabetical", "banana\napple\ncherry", "apple\nbanana\ncherry",
			core.Recipe{{Op: "Sort", Args: []any{"Line feed", false, "Alphabetical (case sensitive)"}}},
		},
		{
			"Sort alphabetical reverse", "banana\napple\ncherry", "cherry\nbanana\napple",
			core.Recipe{{Op: "Sort", Args: []any{"Line feed", true, "Alphabetical (case sensitive)"}}},
		},
		{
			"Sort case insensitive", "Banana\napple\nCherry", "apple\nBanana\nCherry",
			core.Recipe{{Op: "Sort", Args: []any{"Line feed", false, "Alphabetical (case insensitive)"}}},
		},
		{
			"Sort numeric", "10\n2\n1\n20", "1\n2\n10\n20",
			core.Recipe{{Op: "Sort", Args: []any{"Line feed", false, "Numeric"}}},
		},
		{
			"Sort length", "ccc\na\nbb", "a\nbb\nccc",
			core.Recipe{{Op: "Sort", Args: []any{"Line feed", false, "Length"}}},
		},
		{
			"Sort IP address", "192.168.1.10\n192.168.1.2\n10.0.0.1", "10.0.0.1\n192.168.1.2\n192.168.1.10",
			core.Recipe{{Op: "Sort", Args: []any{"Line feed", false, "IP address"}}},
		},
		{
			"Sort IP with invalid (valid first)", "1.2.3.4\nnotanip\n1.2.3.5", "1.2.3.4\n1.2.3.5\nnotanip",
			core.Recipe{{Op: "Sort", Args: []any{"Line feed", false, "IP address"}}},
		},
		{
			"Sort numeric natural", "file10\nfile2\nfile1", "file1\nfile2\nfile10",
			core.Recipe{{Op: "Sort", Args: []any{"Line feed", false, "Numeric"}}},
		},
		{
			"Sort hexadecimal", "ff\n10\n2\na", "2\na\n10\nff",
			core.Recipe{{Op: "Sort", Args: []any{"Line feed", false, "Numeric (hexadecimal)"}}},
		},
		// Mixed numeric/non-numeric leading segments: text (leading empty segment
		// coerces to 0) sorts before numbers, matching CyberChef's numericSort.
		{
			"Sort numeric mixed types", "x\n10\n9\nabc\n2", "abc\nx\n2\n9\n10",
			core.Recipe{{Op: "Sort", Args: []any{"Line feed", false, "Numeric"}}},
		},
		// Equal numeric segments falling through to the string tie-break (a2 < a2b).
		{
			"Sort numeric segment tiebreak", "a10\na2\na1\nb2\na2b\na2", "a1\na2\na2\na2b\na10\nb2",
			core.Recipe{{Op: "Sort", Args: []any{"Line feed", false, "Numeric"}}},
		},
		// Mixed-case text tie-break: localeCompare orders lowercase before the
		// upper-case form (file before FILE), unlike raw byte order.
		{
			"Sort numeric mixed case", "file9\nfile10\nFILE2\n7", "file9\nfile10\nFILE2\n7",
			core.Recipe{{Op: "Sort", Args: []any{"Line feed", false, "Numeric"}}},
		},
		// Comma delimiter and the "Nothing (separate chars)" delimiter.
		{
			"Sort comma delimited", "banana,apple,cherry", "apple,banana,cherry",
			core.Recipe{{Op: "Sort", Args: []any{"Comma", false, "Alphabetical (case sensitive)"}}},
		},
	})
}

func TestFilterAndUniqueDelimiters(t *testing.T) {
	runCases(t, []opCase{
		// "Nothing (separate chars)" splits into individual characters.
		{
			"Filter separate chars", "a1b2c3", "123",
			core.Recipe{{Op: "Filter", Args: []any{"Nothing (separate chars)", `\d`, false}}},
		},
		{
			"Unique comma delimited", "a,b,a,c,b", "a,b,c",
			core.Recipe{{Op: "Unique", Args: []any{"Comma", false}}},
		},
	})
}

func TestUnique(t *testing.T) {
	runCases(t, []opCase{
		{
			"Unique", "a\nb\na\nc\nb", "a\nb\nc",
			core.Recipe{{Op: "Unique", Args: []any{"Line feed", false}}},
		},
		{
			"Unique with count", "a\nb\na", "2 a\n1 b",
			core.Recipe{{Op: "Unique", Args: []any{"Line feed", true}}},
		},
	})
}

// TestSortComparatorInternals directly exercises the Sort/Filter comparator
// helpers' harder-to-reach branches (non-IP fallback, out-of-range octet, and
// the lowercase-before-uppercase tie-break).
func TestSortComparatorInternals(t *testing.T) {
	if _, ok := ipToUint("999.0.0.1"); ok {
		t.Fatal("ipToUint(999.0.0.1) should be false (octet out of range)")
	}
	if !ipLess("apple", "banana") {
		t.Fatal("ipLess(apple, banana): non-IP inputs should compare lexicographically")
	}
	if ipLess("banana", "apple") {
		t.Fatal("ipLess(banana, apple) should be false")
	}
	if localeCompareASCII("a", "A") >= 0 {
		t.Fatal("localeCompareASCII(a, A) should order lowercase first")
	}
	if localeCompareASCII("A", "a") <= 0 {
		t.Fatal("localeCompareASCII(A, a) should order uppercase second")
	}
}

// TestFilterInvalidRegex covers Filter's regex-compile error path.
func TestFilterInvalidRegex(t *testing.T) {
	if _, err := runOp(t, "Filter", "abc", "Line feed", "[", false); err == nil {
		t.Fatal("Filter with an invalid regex: expected an error")
	}
}
