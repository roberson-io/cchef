package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// TestRotateFixtures transcribes the Rotate left / Rotate right cases from
// CyberChef's Rotate.mjs. Each fixture brackets the op with From Hex / To Hex
// (Space delimiter) so the byte-array result is compared as readable hex.
func TestRotateFixtures(t *testing.T) {
	fromHex := core.RecipeOp{Op: "From Hex", Args: []any{"Space"}}
	toHex := core.RecipeOp{Op: "To Hex", Args: []any{"Space"}}
	runCases(t, []opCase{
		{
			"Rotate left: nothing", "", "",
			core.Recipe{fromHex, {Op: "Rotate left", Args: []any{1, false}}, toHex},
		},
		{
			"Rotate left: normal", "61 62 63 31 32 33", "c2 c4 c6 62 64 66",
			core.Recipe{fromHex, {Op: "Rotate left", Args: []any{1, false}}, toHex},
		},
		{
			"Rotate left: carry", "61 62 63 31 32 33", "85 89 8c c4 c8 cd",
			core.Recipe{fromHex, {Op: "Rotate left", Args: []any{2, true}}, toHex},
		},

		{
			"Rotate right: nothing", "", "",
			core.Recipe{fromHex, {Op: "Rotate right", Args: []any{1, false}}, toHex},
		},
		{
			"Rotate right: normal", "61 62 63 31 32 33", "b0 31 b1 98 19 99",
			core.Recipe{fromHex, {Op: "Rotate right", Args: []any{1, false}}, toHex},
		},
		{
			"Rotate right: carry", "61 62 63 31 32 33", "d8 58 98 cc 4c 8c",
			core.Recipe{fromHex, {Op: "Rotate right", Args: []any{2, true}}, toHex},
		},
	})
}

func TestRotateEmpty(t *testing.T) {
	for _, op := range []string{"Rotate left", "Rotate right"} {
		if out, err := runOp(t, op, "", 1.0, false); err != nil || out != "" {
			t.Fatalf("%s(empty) = %q, %v; want empty", op, out, err)
		}
	}
}

// TestRotateCarryEmpty covers the empty-input guard in the carry-through rotate
// helpers (rotlCarry/rotrCarry).
func TestRotateCarryEmpty(t *testing.T) {
	if out, err := runOp(t, "Rotate left", "", 1, true); err != nil || out != "" {
		t.Fatalf("Rotate left carry empty = %q, %v", out, err)
	}
	if out, err := runOp(t, "Rotate right", "", 1, true); err != nil || out != "" {
		t.Fatalf("Rotate right carry empty = %q, %v", out, err)
	}
}
