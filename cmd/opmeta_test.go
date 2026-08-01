package cmd

import (
	"sort"
	"testing"
	"unicode/utf8"

	"github.com/roberson-io/cchef/core"
)

// knownCategories is the closed set of CyberChef categories cchef groups by,
// built from the same constants used in opCategories so the two cannot drift.
var knownCategories = map[string]bool{
	catArithmeticLogic:    true,
	catCodeTidy:           true,
	catCompression:        true,
	catDataFormat:         true,
	catDateTime:           true,
	catEncryptionEncoding: true,
	catExtractors:         true,
	catFlowControl:        true,
	catForensics:          true,
	catHashing:            true,
	catLanguage:           true,
	catMultimedia:         true,
	catNetworking:         true,
	catOther:              true,
	catPublicKey:          true,
	catUtils:              true,
}

// TestOpCategoriesMatchRegistry keeps opCategories in exact sync with the
// registered operations: every operation has at least one known category, and
// there are no stale entries for operations that no longer exist.
func TestOpCategoriesMatchRegistry(t *testing.T) {
	registered := map[string]bool{}
	for _, op := range core.Default.All() {
		name := op.Meta().Name
		registered[name] = true
		cats, ok := opCategories[name]
		if !ok || len(cats) == 0 {
			t.Errorf("operation %q has no category in opCategories", name)
			continue
		}
		for _, c := range cats {
			if !knownCategories[c] {
				t.Errorf("operation %q has unknown category %q", name, c)
			}
		}
	}
	for name := range opCategories {
		if !registered[name] {
			t.Errorf("opCategories has stale entry %q (no such registered operation)", name)
		}
	}
}

// TestSummariesFitAndPresent ensures every operation has a non-empty one-line
// summary within the width bound, so help and `cchef list` stay readable.
func TestSummariesFitAndPresent(t *testing.T) {
	for _, op := range core.Default.All() {
		s := summaryOf(op.Meta())
		if s == "" {
			t.Errorf("empty summary for %q", op.Meta().Name)
		}
		// Allow one extra rune for a truncation ellipsis on derived summaries.
		if n := utf8.RuneCountInString(s); n > maxSummaryLen+1 {
			t.Errorf("summary for %q is %d runes (max %d): %q", op.Meta().Name, n, maxSummaryLen, s)
		}
	}
}

func TestCategoriesOfSortedAndFallback(t *testing.T) {
	got := categoriesOf("URL Decode")
	want := []string{"Data format", "Networking"}
	if len(got) != len(want) {
		t.Fatalf("categoriesOf(URL Decode) = %v, want %v", got, want)
	}
	if !sort.StringsAreSorted(got) {
		t.Errorf("categoriesOf result not sorted: %v", got)
	}
	if fb := categoriesOf("No Such Op"); len(fb) != 1 || fb[0] != "Uncategorized" {
		t.Errorf("fallback = %v, want [Uncategorized]", fb)
	}
}
