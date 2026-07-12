package ops

// Tests for the Normalise Unicode operation.
//
// CyberChef wraps the `unorm` library; cchef uses golang.org/x/text/unicode/norm.
// The four fixture cases come from
// ../CyberChef/tests/operations/tests/NormaliseUnicode.mjs (transcribed to exact
// expected strings); the extras are oracle-verified.

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

func nuRecipe(form string) core.Recipe {
	return core.Recipe{{Op: "Normalise Unicode", Args: []any{form}}}
}

// TestNormaliseUnicodeFixtures transcribes CyberChef's NormaliseUnicode.mjs
// cases. Input is U+00C7, U+0043, combining cedilla U+0327 and Roman numeral one
// U+2160.
func TestNormaliseUnicodeFixtures(t *testing.T) {
	const in = "\u00c7\u0043\u0327\u2160"
	runCases(t, []opCase{
		{"NFD", in, "\u0043\u0327\u0043\u0327\u2160", nuRecipe("NFD")},
		{"NFC", in, "\u00c7\u00c7\u2160", nuRecipe("NFC")},
		{"NFKD", in, "\u0043\u0327\u0043\u0327\u0049", nuRecipe("NFKD")},
		{"NFKC", in, "\u00c7\u00c7\u0049", nuRecipe("NFKC")},
	})
}

// TestNormaliseUnicodeVectors covers a compatibility ligature (U+FB01), a
// fraction (U+00BD) and empty input across the forms.
func TestNormaliseUnicodeVectors(t *testing.T) {
	runCases(t, []opCase{
		{"ligature NFKD", "\ufb01", "fi", nuRecipe("NFKD")},
		{"ligature NFC unchanged", "\ufb01", "\ufb01", nuRecipe("NFC")},
		{"fraction NFKC", "\u00bd", "\u0031\u2044\u0032", nuRecipe("NFKC")},
		{"ascii unchanged NFC", "hello", "hello", nuRecipe("NFC")},
		{"empty NFD", "", "", nuRecipe("NFD")},
		{"empty NFKC", "", "", nuRecipe("NFKC")},
	})
}

// TestNormaliseUnicodeUnknownForm covers the error for an unrecognised form,
// which the ArgOption coercion normally rejects before Run is reached.
func TestNormaliseUnicodeUnknownForm(t *testing.T) {
	if _, err := (NormaliseUnicode{}).Run(sdish("x"), []any{"NFZ"}); err == nil {
		t.Fatal("expected an error for an unknown normalisation form")
	}
}
