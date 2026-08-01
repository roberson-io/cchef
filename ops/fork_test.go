package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// TestForkFixtures transcribes CyberChef's Fork.mjs cases. The "(expect) Error"
// case is covered separately in TestForkPropagatesErrors: CyberChef turns an
// operation's error into the recipe's output text, whereas cchef reports it as
// an error so a shell sees a non-zero exit.
func TestForkFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"Fork: nothing", "", "",
			core.Recipe{{Op: "Fork", Args: []any{"\n", "\n", false}}},
		},
		{
			"Fork, Merge: nothing", "", "",
			core.Recipe{
				{Op: "Fork", Args: []any{"\n", "\n", false}},
				{Op: "Merge", Args: []any{true}},
			},
		},
		{
			"Fork, Conditional Jump, Encodings",
			"Some data with a 1 in it\nSome data with a 2 in it",
			"U29tZSBkYXRhIHdpdGggYSAxIGluIGl0\n" +
				"53 6f 6d 65 20 64 61 74 61 20 77 69 74 68 20 61 20 32 20 69 6e 20 69 74",
			core.Recipe{
				{Op: "Fork", Args: []any{"\\n", "\\n", false}},
				{Op: "Conditional Jump", Args: []any{"1", false, "skipReturn", 10.0}},
				{Op: "To Hex", Args: []any{"Space"}},
				{Op: "Return", Args: []any{}},
				{Op: "Label", Args: []any{"skipReturn"}},
				{Op: "To Base64", Args: []any{"A-Za-z0-9+/="}},
			},
		},
		// Upstream passes To Hex a second "Bytes per line" argument of 0, which
		// cchef's To Hex does not have; 0 means no line breaks, which is what it
		// does with the delimiter alone, so the case reads the same.
		{
			"Fork, Partial Merge", "Hello World", "48656c6c6f 576f726c64",
			core.Recipe{
				{Op: "Fork", Args: []any{" ", " ", false}},
				{Op: "Fork", Args: []any{"l", "l", false}},
				{Op: "Merge", Args: []any{false}},
				{Op: "To Hex", Args: []any{"None"}},
			},
		},
	})
}

// TestForkPropagatesErrors covers CyberChef's "Fork, (expect) Error, Merge"
// case. Upstream catches an OperationError at the recipe level and sets the
// message as the output; cchef returns it as an error instead, so the message
// reaches stderr and the exit status is non-zero.
func TestForkPropagatesErrors(t *testing.T) {
	recipe := core.Recipe{
		{Op: "Fork", Args: []any{"\n\n", "\n\n", false}},
		{Op: "Set Union", Args: []any{"\n\n", ","}},
		{Op: "Merge", Args: []any{true}},
	}
	_, err := recipe.Execute(core.NewDish([]byte("1,2,3,4\n\n3,4,5,6"), core.TypeString))
	if err == nil {
		t.Fatal("want an error from the failing branch")
	}
	if !strings.Contains(err.Error(), "Incorrect number of sets") {
		t.Errorf("got %v, want it to mention the set count", err)
	}
}

// TestForkIgnoreErrors checks that a failing branch leaves its own data alone
// and the rest still run, when errors are to be ignored.
func TestForkIgnoreErrors(t *testing.T) {
	recipe := core.Recipe{
		{Op: "Fork", Args: []any{",", ",", true}},
		{Op: "From Base64", Args: []any{"A-Za-z0-9+/=", true, false}},
	}
	out, err := recipe.Execute(core.NewDish([]byte("aGk=,!!!!"), core.TypeString))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.HasPrefix(out.String(), "hi,") {
		t.Errorf("got %q, want the good branch decoded", out.String())
	}
}

// TestForkEmptyPieces checks that a delimiter appearing at the ends produces
// empty pieces, which are processed like any other.
func TestForkEmptyPieces(t *testing.T) {
	recipe := core.Recipe{
		{Op: "Fork", Args: []any{",", "|", false}},
		{Op: "To Upper case", Args: []any{"All"}},
	}
	out, err := recipe.Execute(core.NewDish([]byte(",a,,b,"), core.TypeString))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.String() != "|A||B|" {
		t.Errorf("got %q, want %q", out.String(), "|A||B|")
	}
}

// TestForkThenLaterStepsSeeMergedData checks that steps after the closing Merge
// run once over the rejoined data, not once per branch.
func TestForkThenLaterStepsSeeMergedData(t *testing.T) {
	recipe := core.Recipe{
		{Op: "Fork", Args: []any{",", ",", false}},
		{Op: "To Upper case", Args: []any{"All"}},
		{Op: "Merge", Args: []any{true}},
		{Op: "Find / Replace", Args: []any{
			core.ToggleString{Value: ",", Option: "Simple string"}, "-", true, false, true, false,
		}},
	}
	out, err := recipe.Execute(core.NewDish([]byte("a,b,c"), core.TypeString))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.String() != "A-B-C" {
		t.Errorf("got %q, want %q", out.String(), "A-B-C")
	}
}

// TestForkNestedMergeWithoutArguments checks a Merge written with no arguments
// closing two levels at once: its "Merge All" argument defaults to true, so the
// step below it runs over the whole rejoined data rather than inside a branch.
func TestForkNestedMergeWithoutArguments(t *testing.T) {
	recipe := core.Recipe{
		{Op: "Fork", Args: []any{",", ",", false}},
		{Op: "Fork", Args: []any{"-", "-", false}},
		{Op: "Merge"},
		{Op: "Find / Replace", Args: []any{
			core.ToggleString{Value: ",", Option: "Simple string"},
			";", true, false, true, false,
		}},
	}
	out, err := recipe.Execute(core.NewDish([]byte("a-b,c"), core.TypeString))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	// Both commas would survive had the replace run per branch; merging all
	// first means it sees them.
	if out.String() != "a-b;c" {
		t.Errorf("got %q, want %q", out.String(), "a-b;c")
	}
}
