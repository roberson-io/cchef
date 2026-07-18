package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// Cases transcribed from ../CyberChef/tests/operations/tests/Ciphers.mjs.
func TestVigenere(t *testing.T) {
	plain := "LUGGAGEBASEMENTVARENNESALLIESCANBECLOTHEDASENEMIESENEMIESCANBECLOTHEDASALLIESALWAYSUSEID"
	cipher := "PXCGRJIEWSVPIQPVRUIQJEJDPOEEJFEQXETOSWDEUDWHJEDLIVANVPMHOCRQFHYLFWLHZAJDPOEEJDPZWYJXWHED"
	runCases(t, []opCase{
		{"encode no input", "", "", core.Recipe{{Op: "Vigenère Encode", Args: []any{"nothing"}}}},
		{"encode normal", plain, cipher, core.Recipe{{Op: "Vigenère Encode", Args: []any{"Edward"}}}},
		{"decode no input", "", "", core.Recipe{{Op: "Vigenère Decode", Args: []any{"nothing"}}}},
		{"decode normal", cipher, plain, core.Recipe{{Op: "Vigenère Decode", Args: []any{"Edward"}}}},
		// Non-alphabet characters pass through and don't advance the key.
		{
			"encode mixed case & punctuation", "Hello, World!", "Rijvs, Uyvjn!",
			core.Recipe{{Op: "Vigenère Encode", Args: []any{"key"}}},
		},
		{
			"decode mixed case & punctuation", "Rijvs, Uyvjn!", "Hello, World!",
			core.Recipe{{Op: "Vigenère Decode", Args: []any{"key"}}},
		},
	})
}

// TestVigenereErrors covers the two validation paths, for both operations.
func TestVigenereErrors(t *testing.T) {
	for _, op := range []string{"Vigenère Encode", "Vigenère Decode"} {
		if _, err := runOp(t, op, "SOMETEXT", ""); err == nil || err.Error() != "No key entered" {
			t.Errorf("%s empty key: got %v, want \"No key entered\"", op, err)
		}
		if _, err := runOp(t, op, "SOMETEXT", "abc123"); err == nil || err.Error() != "The key must consist only of letters" {
			t.Errorf("%s invalid key: got %v, want \"The key must consist only of letters\"", op, err)
		}
	}
}
