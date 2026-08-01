package ops

// CTPH / Compare CTPH hashes. These ops wrap the non-standard ctph.js npm
// package (no CyberChef fixtures exist), so vectors are generated from that exact
// package via node — see PLAN's fuzzy-hashing note.

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
)

func ctphDigestRecipe() core.Recipe {
	return core.Recipe{{Op: "CTPH", Args: []any{}}}
}

func TestCTPH(t *testing.T) {
	long := strings.Repeat("The quick brown fox jumps over the lazy dog. ", 20)
	runCases(t, []opCase{
		{"empty", "", "A::", ctphDigestRecipe()},
		{"Hello", "Hello", "A:Y3:Y3", ctphDigestRecipe()},
		{"Hello world", "Hello world", "A:YX8:YX8", ctphDigestRecipe()},
		{"sentence", "The quick brown fox jumps over the lazy dog", "A:5ELdJ9Ludf:5TJ1n", ctphDigestRecipe()},
		{
			"long (block scaling)", long,
			"B:5TJ10NJ10NJ10NJ10NJ10NJ10NJ10NJ10NJ10NJ10NJ10NJ10NJ10NJ10NJ10NJ10NJ10NJ10NJ10NJ1g:5k4444444444444444446",
			ctphDigestRecipe(),
		},
	})
}

// Compare CTPH hashes: two hashes joined by the delimiter, similarity 0..100.
func TestCompareCTPH(t *testing.T) {
	const (
		ha = "B:5TJ10NJ10NJ10NJ10NJ10NJ10NJ10NJ10NJ10NJ10NJ10NJ10NJ10NJ10NJ10NJ10NJ10NJ10NJ10NJ1g:5k4444444444444444446"
		hb = "B:5TJ10NJ10NJ10NJ10NJ10NJ10NJ10NJ10NJ10NJ10NJ10NJ10NJ10NJ10NJ10NJ10NJ10NJ10NJ10NJ1Y:5k444444444444444444o"
		hc = "A:YX8:YX8"
	)
	cmp := func(a, b string) core.Recipe {
		return core.Recipe{{Op: "Compare CTPH hashes", Args: []any{"Line feed"}}}
	}
	runCases(t, []opCase{
		{"identical", "A:Y3:Y3\nA:Y3:Y3", "100", cmp("", "")},
		{"near (same block)", ha + "\n" + hb, "98.76543209876543", cmp("", "")},
		// ha starts 'B', hc starts 'A': triggers the b1>b2 swap then the
		// different-block branch.
		{"different block", ha + "\n" + hc, "0", cmp("", "")},
		// Empty-signature comparisons exercise the Levenshtein base cases.
		{"empty vs sig", "A::\nA:Y3:Y3", "0", cmp("", "")},
		{"sig vs empty", "A:Y3:Y3\nA::", "0", cmp("", "")},
	})
}

// A delimiter that does not split the input into exactly two hashes errors.
func TestCompareCTPHBadSampleCount(t *testing.T) {
	_, err := runOp(t, "Compare CTPH hashes", "only-one-hash", "Line feed")
	want := "Incorrect number of samples."
	if err == nil || err.Error() != want {
		t.Fatalf("got %v\nwant %q", err, want)
	}
}
