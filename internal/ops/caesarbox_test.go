package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// TestCaesarBoxCipherFixtures transcribes the CyberChef CaesarBox.mjs fixtures.
func TestCaesarBoxCipherFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"Caesar Box Cipher: nothing", "", "",
			core.Recipe{{Op: "Caesar Box Cipher", Args: []any{1}}},
		},
		{
			"Caesar Box Cipher: Hello World! encode", "Hello World!", "Hlodeor!lWl",
			core.Recipe{{Op: "Caesar Box Cipher", Args: []any{3}}},
		},
		{
			"Caesar Box Cipher: Hello World! decode", "Hlodeor!lWl", "HelloWorld!",
			core.Recipe{{Op: "Caesar Box Cipher", Args: []any{4}}},
		},
	})
}

// TestCaesarBoxCipherEdge covers space stripping, padding, and the non-positive
// height branch (CyberChef produces empty output there).
func TestCaesarBoxCipherEdge(t *testing.T) {
	runCases(t, []opCase{
		// Height 1 is a pure space-strip (single row read straight across).
		{
			"Caesar Box Cipher: height 1 strips spaces", "a b c", "abc",
			core.Recipe{{Op: "Caesar Box Cipher", Args: []any{1}}},
		},
		// Height >= length: one char per column, padding trimmed, order preserved.
		{
			"Caesar Box Cipher: tall box", "abcd", "abcd",
			core.Recipe{{Op: "Caesar Box Cipher", Args: []any{4}}},
		},
		{
			"Caesar Box Cipher: zero height is empty", "abc", "",
			core.Recipe{{Op: "Caesar Box Cipher", Args: []any{0}}},
		},
		{
			"Caesar Box Cipher: negative height is empty", "abc", "",
			core.Recipe{{Op: "Caesar Box Cipher", Args: []any{-1}}},
		},
	})
}
