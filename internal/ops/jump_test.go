package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// TestJumpFixtures transcribes CyberChef's Jump.mjs cases.
func TestJumpFixtures(t *testing.T) {
	runCases(t, []opCase{
		// No Label exists, so the jump is refused and the encode runs.
		{
			"Jump: Empty Label", "should be changed", "c2hvdWxkIGJlIGNoYW5nZWQ=",
			core.Recipe{
				{Op: "Jump", Args: []any{"", 10.0}},
				{Op: "To Base64", Args: []any{"A-Za-z0-9+/="}},
			},
		},
		{
			"Jump: skips 1", "shouldnt be changed", "shouldnt be changed",
			core.Recipe{
				{Op: "Jump", Args: []any{"skipReplace", 10.0}},
				{Op: "To Base64", Args: []any{"A-Za-z0-9+/="}},
				{Op: "Label", Args: []any{"skipReplace"}},
			},
		},
	})
}

// TestConditionalJumpFixtures transcribes CyberChef's ConditionalJump.mjs cases.
func TestConditionalJumpFixtures(t *testing.T) {
	runCases(t, []opCase{
		// An allowance of zero refuses the jump, so both encodes run.
		{
			"Conditional Jump: Skips 0", "should be changed",
			"YzJodmRXeGtJR0psSUdOb1lXNW5aV1E9",
			core.Recipe{
				{Op: "Conditional Jump", Args: []any{"match", false, "", 0.0}},
				{Op: "To Base64", Args: []any{"A-Za-z0-9+/="}},
				{Op: "To Base64", Args: []any{"A-Za-z0-9+/="}},
			},
		},
		{
			"Conditional Jump: Skips 1", "should be changed",
			"ONUG65LMMQQGEZJAMNUGC3THMVSA====",
			core.Recipe{
				{Op: "Conditional Jump", Args: []any{"should", false, "skip match", 10.0}},
				{Op: "To Base64", Args: []any{"A-Za-z0-9+/="}},
				{Op: "Label", Args: []any{"skip match"}},
				{Op: "To Base32", Args: []any{"A-Z2-7="}},
			},
		},
		// A backwards jump: MD2 runs on the second pass, and the data no longer
		// matches, so the loop ends.
		{
			"Conditional Jump: Skips backwards", "match",
			"f7cf556f7f4fc6635db8c314f7a81f2a",
			core.Recipe{
				{Op: "Label", Args: []any{"back to the beginning"}},
				{Op: "Jump", Args: []any{"skip replace"}},
				{Op: "MD2", Args: []any{}},
				{Op: "Label", Args: []any{"skip replace"}},
				{Op: "Conditional Jump", Args: []any{"match", false, "back to the beginning", 10.0}},
			},
		},
	})
}

// TestConditionalJumpInvert checks the invert flag: the jump is taken when the
// data does not match.
func TestConditionalJumpInvert(t *testing.T) {
	recipe := core.Recipe{
		{Op: "Conditional Jump", Args: []any{"absent", true, "skip", 10.0}},
		{Op: "To Base64", Args: []any{"A-Za-z0-9+/="}},
		{Op: "Label", Args: []any{"skip"}},
	}
	out, err := recipe.Execute(core.NewDish([]byte("plain"), core.TypeString))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.String() != "plain" {
		t.Errorf("got %q, want %q", out.String(), "plain")
	}
}

// TestConditionalJumpEmptyPatternNeverJumps checks that an empty test does
// nothing at all, even with a label that exists.
func TestConditionalJumpEmptyPatternNeverJumps(t *testing.T) {
	recipe := core.Recipe{
		{Op: "Conditional Jump", Args: []any{"", false, "skip", 10.0}},
		{Op: "To Upper case", Args: []any{"All"}},
		{Op: "Label", Args: []any{"skip"}},
	}
	out, err := recipe.Execute(core.NewDish([]byte("hi"), core.TypeString))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.String() != "HI" {
		t.Errorf("got %q, want %q", out.String(), "HI")
	}
}

// TestConditionalJumpBadPattern checks that a pattern which is not a valid
// regular expression is reported rather than silently ignored.
func TestConditionalJumpBadPattern(t *testing.T) {
	recipe := core.Recipe{
		{Op: "Label", Args: []any{"l"}},
		{Op: "Conditional Jump", Args: []any{"([", false, "l", 10.0}},
	}
	if _, err := recipe.Execute(core.NewDish([]byte("x"), core.TypeString)); err == nil {
		t.Error("want an error for an invalid pattern")
	}
}

// TestJumpAllowanceLimitsALoop checks that the allowance bounds a backwards
// loop: with the data always matching, the loop runs exactly that many times.
func TestJumpAllowanceLimitsALoop(t *testing.T) {
	recipe := core.Recipe{
		{Op: "Label", Args: []any{"top"}},
		{Op: "Find / Replace", Args: []any{
			core.ToggleString{Value: "$", Option: "Regex"}, "x", true, false, true, false,
		}},
		{Op: "Conditional Jump", Args: []any{"^x*$", false, "top", 3.0}},
	}
	out, err := recipe.Execute(core.NewDish([]byte(""), core.TypeString))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	// One pass plus three jumps.
	if out.String() != "xxxx" {
		t.Errorf("got %q, want %q", out.String(), "xxxx")
	}
}

// TestJumpToLabelWithNoName checks that a Label written with no arguments at all
// is found by a jump to the empty label name.
func TestJumpToLabelWithNoName(t *testing.T) {
	recipe := core.Recipe{
		{Op: "Jump", Args: []any{"", 10.0}},
		{Op: "To Base64", Args: []any{"A-Za-z0-9+/="}},
		{Op: "Label"},
	}
	out, err := recipe.Execute(core.NewDish([]byte("keep"), core.TypeString))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.String() != "keep" {
		t.Errorf("got %q, want the encode skipped", out.String())
	}
}
