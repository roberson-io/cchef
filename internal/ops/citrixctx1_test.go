package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// Citrix CTX1 has no CyberChef fixture file; these vectors were produced by the
// CyberChef-server oracle (encode is UTF-16LE keyed with a running XOR chain).

// TestCitrixCTX1Encode covers the encode operation.
func TestCitrixCTX1Encode(t *testing.T) {
	runCases(t, []opCase{
		{
			"Citrix CTX1 Encode: password", "password", "NFHALEBBMHGCLEBBMDGGKMAJNOHLLKBP",
			core.Recipe{{Op: "Citrix CTX1 Encode", Args: []any{}}},
		},
		{
			"Citrix CTX1 Encode: Test", "Test", "PBFEJEDBOHECJDDG",
			core.Recipe{{Op: "Citrix CTX1 Encode", Args: []any{}}},
		},
		{
			"Citrix CTX1 Encode: single char", "a", "MEGB",
			core.Recipe{{Op: "Citrix CTX1 Encode", Args: []any{}}},
		},
		{
			// café exercises a non-ASCII code point through the UTF-16LE encoder.
			"Citrix CTX1 Encode: unicode", "café", "MGGDKHACMBGECIIN",
			core.Recipe{{Op: "Citrix CTX1 Encode", Args: []any{}}},
		},
		{
			"Citrix CTX1 Encode: empty", "", "",
			core.Recipe{{Op: "Citrix CTX1 Encode", Args: []any{}}},
		},
	})
}

// TestCitrixCTX1Decode covers the decode operation (the inverse of encode).
func TestCitrixCTX1Decode(t *testing.T) {
	runCases(t, []opCase{
		{
			"Citrix CTX1 Decode: password", "NFHALEBBMHGCLEBBMDGGKMAJNOHLLKBP", "password",
			core.Recipe{{Op: "Citrix CTX1 Decode", Args: []any{}}},
		},
		{
			"Citrix CTX1 Decode: Test", "PBFEJEDBOHECJDDG", "Test",
			core.Recipe{{Op: "Citrix CTX1 Decode", Args: []any{}}},
		},
		{
			"Citrix CTX1 Decode: single char", "MEGB", "a",
			core.Recipe{{Op: "Citrix CTX1 Decode", Args: []any{}}},
		},
		{
			"Citrix CTX1 Decode: unicode", "MGGDKHACMBGECIIN", "café",
			core.Recipe{{Op: "Citrix CTX1 Decode", Args: []any{}}},
		},
		{
			"Citrix CTX1 Decode: empty", "", "",
			core.Recipe{{Op: "Citrix CTX1 Decode", Args: []any{}}},
		},
	})
}

// TestCitrixCTX1DecodeError covers the invalid-length error path.
func TestCitrixCTX1DecodeError(t *testing.T) {
	if _, err := runOp(t, "Citrix CTX1 Decode", "ABC"); err == nil {
		t.Fatal("expected error for non-multiple-of-4 length")
	} else if !strings.Contains(err.Error(), "Incorrect hash length") {
		t.Fatalf("unexpected error: %v", err)
	}
}
