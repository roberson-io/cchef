package ops

// LM Hash fixture transcribed from CyberChef's tests/operations/tests/NTLM.mjs.

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

func TestLMHash(t *testing.T) {
	runCases(t, []opCase{
		{
			"LM Hash", "QWERTYUIOPASDFGHJKLZXCVBNM1234567890!@#$%^&*()_+.,?/",
			"6D9DF16655336CA75A3C13DD18BA8156",
			core.Recipe{{Op: "LM Hash", Args: []any{}}},
		},
		// Empty password: the well-known LM hash of the two null halves.
		{
			"LM Hash empty", "",
			"AAD3B435B51404EEAAD3B435B51404EE",
			core.Recipe{{Op: "LM Hash", Args: []any{}}},
		},
		// Lowercase input is uppercased before hashing, so it matches an
		// all-uppercase password of the same letters.
		{
			"LM Hash lowercase", "password",
			"E52CAC67419A9A224A3B108F3FA6CB6D",
			core.Recipe{{Op: "LM Hash", Args: []any{}}},
		},
	})
}
