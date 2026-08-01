package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

// These tests exercise each Run's own handling of an out-of-range option value.
// In normal use core.CoerceArgs validates ArgOption values before Run is ever
// called (see Recipe.ExecuteWith), so these default arms are unreachable through
// the engine/CLI. Calling Run directly pins the standalone contract: an
// unvalidated bad option is rejected (or safely falls back) rather than
// panicking. They document behaviour a future direct caller can rely on.

func strDish(s string) *core.Dish { return core.NewDish([]byte(s), core.TypeString) }

// ToUpperCase falls back to returning the input unchanged for an unknown scope
// (the switch default), rather than dereferencing a nil regexp. (ToLowerCase has
// no scope switch — it always lower-cases — so it has no such arm.)
func TestCaseRunUnknownScopeFallsBack(t *testing.T) {
	got, err := (ToUpperCase{}).Run(strDish("Hello World"), []any{"BogusScope"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.String() != "Hello World" {
		t.Fatalf("got %q, want input unchanged", got.String())
	}
}

// Keccak/SHA3 reject an unknown digest size.
func TestHashRunUnknownSizeErrors(t *testing.T) {
	for _, op := range []core.Operation{Keccak{}, SHA3{}} {
		if _, err := op.Run(strDish("x"), []any{"999"}); err == nil {
			t.Fatalf("%T: expected error for unknown size", op)
		}
	}
}

// Diff rejects an unknown "Diff by" mode (input has the two samples the switch
// is reached with).
func TestDiffRunUnknownModeErrors(t *testing.T) {
	args := []any{`\n\n`, "BogusMode", true, true, false, false}
	if _, err := (Diff{}).Run(strDish("a\n\nb"), args); err == nil {
		t.Fatal("expected error for unknown Diff by mode")
	}
}

// ChangeIPFormat rejects unknown input and output formats independently.
func TestChangeIPFormatRunUnknownFormatsError(t *testing.T) {
	if _, err := (ChangeIPFormat{}).Run(strDish("1.2.3.4"), []any{"Bogus", "Decimal"}); err == nil {
		t.Fatal("expected error for unknown input format")
	}
	if _, err := (ChangeIPFormat{}).Run(strDish("1.2.3.4"), []any{"Dotted Decimal", "Bogus"}); err == nil {
		t.Fatal("expected error for unknown output format")
	}
}

// convertCoordinates rejects an unknown input format (the helper's switch
// default, unreachable via the op because Auto resolves to a known format).
func TestConvertCoordinatesUnknownFormatErrors(t *testing.T) {
	if _, err := convertCoordinates("51,0", "BogusFormat", "Comma", "Decimal Degrees", "Comma", "None", 3); err == nil {
		t.Fatal("expected error for unknown input format")
	}
}
