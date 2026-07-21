package ops

// SSDEEP / Compare SSDEEP hashes. These wrap the non-standard ssdeep.js npm
// package (no CyberChef fixtures exist), so vectors are generated from that exact
// package via node — see PLAN's fuzzy-hashing note.

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

func ssdeepDigestRecipe() core.Recipe {
	return core.Recipe{{Op: "SSDEEP", Args: []any{}}}
}

func TestSSDEEP(t *testing.T) {
	long := strings.Repeat("The quick brown fox jumps over the lazy dog. ", 20)
	runCases(t, []opCase{
		{"empty", "", "3::", ssdeepDigestRecipe()},
		{"Hello", "Hello", "3:aE:aE", ssdeepDigestRecipe()},
		{"Hello world", "Hello world", "3:agPn:agPn", ssdeepDigestRecipe()},
		{"sentence", "The quick brown fox jumps over the lazy dog", "3:FJKKIUKact:FHIGi", ssdeepDigestRecipe()},
		{
			"long (block scaling)", long,
			"6:FHIG8NIG8NIG8NIG8NIG8NIG8NIG8NIG8NIG8NIG8NIG8NIG8NIG8NIG8NIG8NID:Fg666666666666666666G",
			ssdeepDigestRecipe(),
		},
	})
}

func TestCompareSSDEEP(t *testing.T) {
	const (
		sa = "6:FHIG8NIG8NIG8NIG8NIG8NIG8NIG8NIG8NIG8NIG8NIG8NIG8NIG8NIG8NIG8NID:Fg666666666666666666G"
		sb = "6:FHIG8NIG8NIG8NIG8NIG8NIG8NIG8NIG8NIG8NIG8NIG8NIG8NIG8NIG8NIG8NIB:Fg666666666666666666K"
		sc = "3:agPn:agPn"
	)
	cmp := core.Recipe{{Op: "Compare SSDEEP hashes", Args: []any{"Line feed"}}}
	runCases(t, []opCase{
		{"identical", "3:aE:aE\n3:aE:aE", "100", cmp},
		{"near (same block)", sa + "\n" + sb, "98.4375", cmp},
		// Block sizes differing by more than one give 0.
		{"far block", sa + "\n" + sc, "0", cmp},
	})
}

// A non-default delimiter (Comma) is honoured when splitting the two hashes.
func TestCompareSSDEEPCommaDelim(t *testing.T) {
	got, err := runOp(t, "Compare SSDEEP hashes", "3:aE:aE,3:aE:aE", "Comma")
	if err != nil || got != "100" {
		t.Fatalf("got %q, %v want 100", got, err)
	}
}

func TestCompareSSDEEPBadSampleCount(t *testing.T) {
	_, err := runOp(t, "Compare SSDEEP hashes", "one\ntwo\nthree", "Line feed")
	want := "Incorrect number of samples."
	if err == nil || err.Error() != want {
		t.Fatalf("got %v\nwant %q", err, want)
	}
}
