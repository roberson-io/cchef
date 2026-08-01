package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

// TestSubsectionFixtures transcribes CyberChef's Subsection.mjs cases, including
// the nested case where an inner Subsection is closed by a partial Merge.
func TestSubsectionFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"Subsection: nothing", "", "",
			core.Recipe{{Op: "Subsection", Args: []any{"", true, true, false}}},
		},
		{
			"Subsection, Full Merge: nothing", "", "",
			core.Recipe{
				{Op: "Subsection", Args: []any{"", true, true, false}},
				{Op: "Merge", Args: []any{true}},
			},
		},
		{
			"Subsection, Partial Merge: nothing", "", "",
			core.Recipe{
				{Op: "Subsection", Args: []any{"", true, true, false}},
				{Op: "Merge", Args: []any{false}},
			},
		},
		{
			"Subsection, Full Merge: Base64 with Hex", "SGVsbG38675629ybGQ=", "Hello World",
			core.Recipe{
				{Op: "Subsection", Args: []any{"386756", true, true, false}},
				{Op: "From Hex", Args: []any{"Auto"}},
				{Op: "Merge", Args: []any{true}},
				{Op: "From Base64", Args: []any{"A-Za-z0-9+/=", true, false}},
			},
		},
		{
			"Subsection, Partial Merge: Base64 with Hex surrounded by binary data.",
			"000000000SGVsbG38675629ybGQ=0000000000",
			"000000000Hello World0000000000",
			core.Recipe{
				{Op: "Subsection", Args: []any{"SGVsbG38675629ybGQ=", true, true, false}},
				{Op: "Subsection", Args: []any{"386756", true, true, false}},
				{Op: "From Hex", Args: []any{"Auto"}},
				{Op: "Merge", Args: []any{false}},
				{Op: "From Base64", Args: []any{"A-Za-z0-9+/=", true, false}},
			},
		},
	})
}

// TestSubsectionOnlyMatchedPartsChange checks that the steps below run over the
// matching sections and leave everything else exactly as it was.
func TestSubsectionOnlyMatchedPartsChange(t *testing.T) {
	recipe := core.Recipe{
		{Op: "Subsection", Args: []any{"[a-z]+", true, true, false}},
		{Op: "To Upper case", Args: []any{"All"}},
	}
	out, err := recipe.Execute(core.NewDish([]byte("1abc2def3"), core.TypeString))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.String() != "1ABC2DEF3" {
		t.Errorf("got %q, want %q", out.String(), "1ABC2DEF3")
	}
}

// TestSubsectionCaptureGroup checks that with a capture group only the group is
// worked on, and the rest of the match is left alone.
func TestSubsectionCaptureGroup(t *testing.T) {
	recipe := core.Recipe{
		{Op: "Subsection", Args: []any{"<([a-z]+)>", true, true, false}},
		{Op: "To Upper case", Args: []any{"All"}},
	}
	out, err := recipe.Execute(core.NewDish([]byte("x<abc>y"), core.TypeString))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.String() != "x<ABC>y" {
		t.Errorf("got %q, want %q", out.String(), "x<ABC>y")
	}
}

// TestSubsectionGlobalFlag checks that with global matching off only the first
// section is worked on.
func TestSubsectionGlobalFlag(t *testing.T) {
	for _, c := range []struct {
		name   string
		global bool
		want   string
	}{
		{"every section", true, "1ABC2DEF3"},
		{"the first only", false, "1ABC2def3"},
	} {
		t.Run(c.name, func(t *testing.T) {
			recipe := core.Recipe{
				{Op: "Subsection", Args: []any{"[a-z]+", true, c.global, false}},
				{Op: "To Upper case", Args: []any{"All"}},
			}
			out, err := recipe.Execute(core.NewDish([]byte("1abc2def3"), core.TypeString))
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if out.String() != c.want {
				t.Errorf("got %q, want %q", out.String(), c.want)
			}
		})
	}
}

// TestSubsectionCaseFlag checks the case-sensitivity flag.
func TestSubsectionCaseFlag(t *testing.T) {
	for _, c := range []struct {
		name          string
		caseSensitive bool
		want          string
	}{
		{"sensitive misses", true, "abc"},
		{"insensitive matches", false, "ABC"},
	} {
		t.Run(c.name, func(t *testing.T) {
			recipe := core.Recipe{
				{Op: "Subsection", Args: []any{"ABC", c.caseSensitive, true, false}},
				{Op: "To Upper case", Args: []any{"All"}},
			}
			out, err := recipe.Execute(core.NewDish([]byte("abc"), core.TypeString))
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if out.String() != c.want {
				t.Errorf("got %q, want %q", out.String(), c.want)
			}
		})
	}
}

// TestSubsectionNoMatchSkipsTheStepsBelow checks that when nothing matches, the
// steps a Subsection governs do not run over the whole input, and the steps
// after its Merge still do.
func TestSubsectionNoMatchSkipsTheStepsBelow(t *testing.T) {
	recipe := core.Recipe{
		{Op: "Subsection", Args: []any{"absent", true, true, false}},
		{Op: "To Upper case", Args: []any{"All"}},
		{Op: "Merge", Args: []any{true}},
		{Op: "Find / Replace", Args: []any{
			core.ToggleString{Value: "z", Option: "Simple string"},
			"!", true, false, true, false,
		}},
	}
	out, err := recipe.Execute(core.NewDish([]byte("abz"), core.TypeString))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.String() != "ab!" {
		t.Errorf("got %q, want %q — uppercase must not have run", out.String(), "ab!")
	}
}

// TestSubsectionIgnoreErrors checks that a failing section is left as it was
// when errors are to be ignored, and reported when they are not.
func TestSubsectionIgnoreErrors(t *testing.T) {
	build := func(ignore bool) core.Recipe {
		return core.Recipe{
			{Op: "Subsection", Args: []any{"[!]+", true, true, ignore}},
			{Op: "From Base64", Args: []any{"A-Za-z0-9+/=", false, true}},
		}
	}
	out, err := build(true).Execute(core.NewDish([]byte("a!!!b"), core.TypeString))
	if err != nil {
		t.Fatalf("ignoring: %v", err)
	}
	if out.String() != "a!!!b" {
		t.Errorf("ignoring: got %q, want the section left alone", out.String())
	}
	if _, err := build(false).Execute(core.NewDish([]byte("a!!!b"), core.TypeString)); err == nil {
		t.Error("not ignoring: want an error")
	}
}

// TestSubsectionBadPattern checks an invalid pattern is reported.
func TestSubsectionBadPattern(t *testing.T) {
	recipe := core.Recipe{{Op: "Subsection", Args: []any{"([", true, true, false}}}
	if _, err := recipe.Execute(core.NewDish([]byte("x"), core.TypeString)); err == nil {
		t.Error("want an error for an invalid pattern")
	}
}
