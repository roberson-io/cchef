package ops

// NT Hash fixture transcribed from CyberChef's tests/operations/tests/NTLM.mjs.

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

func TestNTHash(t *testing.T) {
	runCases(t, []opCase{
		{
			"NT Hash", "QWERTYUIOPASDFGHJKLZXCVBNM1234567890!@#$%^&*()_+.,?/",
			"C5FA1C40E55734A8E528DBFE21766D23",
			core.Recipe{{Op: "NT Hash", Args: []any{}}},
		},
		// Empty input is the MD4 of an empty UTF-16LE buffer.
		{
			"NT Hash empty", "",
			"31D6CFE0D16AE931B73C59D7E0C089C0",
			core.Recipe{{Op: "NT Hash", Args: []any{}}},
		},
	})
}
