package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

// TestEscapeSmartCharacters transcribes CyberChef's EscapeSmartCharacters.mjs
// fixtures.
func TestEscapeSmartCharacters(t *testing.T) {
	include := core.Recipe{{Op: "Escape Smart Characters", Args: []any{"Include"}}}
	runCases(t, []opCase{
		{
			"smart quotes and apostrophes",
			"“Hello,” she said, ‘yes.’",
			"\"Hello,\" she said, 'yes.'", include,
		},
		{
			"em dash, en dash and ellipsis",
			"page 1–3 — wait…", "page 1-3 -- wait...", include,
		},
		{
			"trademark symbols",
			"Foo© Bar® Baz™", "Foo(c) Bar(r) Baz(tm)", include,
		},
		{
			"arrows and guillemets",
			"← → ↔ ⇒ « »", "<-- --> <-> ==> << >>", include,
		},
		{
			"math and misc",
			"3 × 4 ÷ 2 = 6, ±0.5 • item",
			"3 x 4 / 2 = 6, +/-0.5 * item", include,
		},
		{
			"NBSP becomes regular space",
			"a b c", "a b c", include,
		},
		{
			"unmappable Include preserves char",
			"warning: ☣ hazard", "warning: ☣ hazard", include,
		},
		{
			"unmappable Remove drops char",
			"warning: ☣ hazard", "warning:  hazard",
			core.Recipe{{Op: "Escape Smart Characters", Args: []any{"Remove"}}},
		},
		{
			"unmappable Replace substitutes dot",
			"warning: ☣ hazard", "warning: . hazard",
			core.Recipe{{Op: "Escape Smart Characters", Args: []any{"Replace with '.'"}}},
		},
		{"pure ASCII passes through", "hello world! 123", "hello world! 123", include},
		{"empty input", "", "", include},
	})
}
