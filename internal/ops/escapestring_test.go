package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// Escape string is a from-scratch implementation (CyberChef wraps the jsesc npm
// library, which has no Go port). All expected outputs below were verified
// against the real CyberChef via the local CyberChef-server oracle.
// args order: level, quote, jsonCompat, es6, uppercaseHex.
func TestEscapeString(t *testing.T) {
	special := func(quote string) []any { return []any{"Special chars", quote, false, true, false} }
	runCases(t, []opCase{
		{"single quote and newline", "it's\n", `it\'s\n`,
			core.Recipe{{Op: "Escape string", Args: special("Single")}}},
		{"non-ASCII to hex", "café", `caf\xe9`,
			core.Recipe{{Op: "Escape string", Args: special("Single")}}},
		{"single quote raw under double", "it's", "it's",
			core.Recipe{{Op: "Escape string", Args: special("Double")}}},
		{"escape double quotes", `say "hi"`, `say \"hi\"`,
			core.Recipe{{Op: "Escape string", Args: special("Double")}}},
		{"backslash and tab", "a\\b\tc", `a\\b\tc`,
			core.Recipe{{Op: "Escape string", Args: special("Single")}}},

		{"everything escapes ASCII", "AB", `\x41\x42`,
			core.Recipe{{Op: "Escape string", Args: []any{"Everything", "Single", false, true, false}}}},
		{"everything keeps named quote escape", "a'b", `\x61\'\x62`,
			core.Recipe{{Op: "Escape string", Args: []any{"Everything", "Single", false, true, false}}}},

		{"minimal escapes tab/quote but not non-ASCII", "a\tb'c\né", "a\\tb\\'c\\né",
			core.Recipe{{Op: "Escape string", Args: []any{"Minimal", "Single", false, true, false}}}},

		{"uppercase hex", "é", `\xE9`,
			core.Recipe{{Op: "Escape string", Args: []any{"Special chars", "Single", false, true, true}}}},

		{"es6 astral", "\U0001F600", `\u{1f600}`,
			core.Recipe{{Op: "Escape string", Args: []any{"Special chars", "Single", false, true, false}}}},
		{"non-es6 astral surrogate pair", "\U0001F600", "\\ud83d\\ude00",
			core.Recipe{{Op: "Escape string", Args: []any{"Special chars", "Single", false, false, false}}}},

		// JSON mode wraps in double quotes, uses \u escapes for non-ASCII, and
		// still honours es6 / surrogate pairs (verified via the oracle).
		{"json astral es6", "\U0001F600", `"\u{1f600}"`,
			core.Recipe{{Op: "Escape string", Args: []any{"Special chars", "Double", true, true, false}}}},
		{"json astral surrogate", "\U0001F600", "\"\\ud83d\\ude00\"",
			core.Recipe{{Op: "Escape string", Args: []any{"Special chars", "Double", true, false, false}}}},
	})
}
