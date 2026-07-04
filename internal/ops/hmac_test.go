package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// Cases transcribed from CyberChef tests/operations/tests/Hash.mjs.
func TestHMACFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"HMAC: SHA256", "Hello, World!",
			"52589bd80ccfa4acbb3f9512dfaf4f700fa5195008aae0b77a9e47dcca75beac",
			core.Recipe{{Op: "HMAC", Args: []any{
				core.ToggleString{Value: "test", Option: "Latin1"}, "SHA256",
			}}},
		},
		{
			"HMAC: RFC4231 TC1 SHA-256", "Hi There",
			"b0344c61d8db38535ca8afceaf0bf12b881dc200c9833da726e9376c2e32cff7",
			core.Recipe{{Op: "HMAC", Args: []any{
				core.ToggleString{Value: "0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b", Option: "Hex"}, "SHA256",
			}}},
		},
	})
}
