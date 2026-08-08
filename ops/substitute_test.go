package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

// subRecipe builds a Substitute recipe.
func subRecipe(plain, cipher string, ignoreCase bool) core.Recipe {
	return core.Recipe{{Op: "Substitute", Args: []any{plain, cipher, ignoreCase}}}
}

// Default rotation map used by CyberChef's Substitute defaults.
const (
	subDefPlain  = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	subDefCipher = "XYZABCDEFGHIJKLMNOPQRSTUVW"
)

// TestSubstitute covers the operation's behaviours (all oracle-verified, since
// the op has no upstream fixture file).
func TestSubstitute(t *testing.T) {
	runCases(t, []opCase{
		// Basic substitution (a Caesar-style shift via the defaults).
		{"default shift", "HELLO", "EBIIL", subRecipe(subDefPlain, subDefCipher, false)},
		{"case-sensitive miss", "hello", "hello", subRecipe(subDefPlain, subDefCipher, false)},
		// Range expansion with a hyphen.
		{"range expand", "1234567890", "8765432109", subRecipe("0-9", "9876543210", false)},
		// Length mismatch warning ('c' is unmapped and passes through).
		{
			"length mismatch warning", "abc",
			"Warning: Plaintext and Ciphertext lengths differ\n\nxyc",
			subRecipe("abcd", "xy", false),
		},
		// Escape notation in the key, and an escaped hyphen (a literal '-').
		{"escaped newline key", "a\nb", "XYb", subRecipe(`a\n`, "XY", false)},
		{"escaped hyphen", "a-c", "XYZ", subRecipe(`a\-c`, "XYZ", false)},
		// Unmapped code points pass through; a code point within a byte is written
		// as that single Latin-1 byte (é -> 0xe9), matching CyberChef.
		{"unicode passthrough", "café", "caf\xe9", subRecipe("A", "B", false)},

		// Ignore case: input case is preserved in the output.
		{"ignore case default", "hello", "ebiil", subRecipe(subDefPlain, subDefCipher, true)},
		{"ic upper input, lower keys", "ABC", "XYZ", subRecipe("abc", "xyz", true)},
		{"ic mixed case", "aBcD", "wXyZ", subRecipe("abcd", "WXYZ", true)},
		{"ic digits treated as upper", "012", "ABC", subRecipe("0-9", "abcdefghij", true)},
		{"ic value uppercased", "A", "Z", subRecipe("a", "z", true)},
		{"ic unmapped passthrough", "Q9", "Q9", subRecipe("abc", "xyz", true)},
		// Case-sensitive: the substituted value's own case is preserved.
		{"value case preserved", "A", "z", subRecipe("A", "z", false)},
	})
}
