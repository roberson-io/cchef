package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// TestNetBIOSFixtures transcribes CyberChef's NetBIOS.mjs cases. NetBIOS
// (level-1 encoding) maps each byte to two nibble characters offset by 65 ('A').
func TestNetBIOSFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"Encode NetBIOS name", "The NetBIOS name", "FEGIGFCAEOGFHEECEJEPFDCAGOGBGNGF",
			core.Recipe{{Op: "Encode NetBIOS Name", Args: []any{65}}},
		},
		{
			"Decode NetBIOS Name", "FEGIGFCAEOGFHEECEJEPFDCAGOGBGNGF", "The NetBIOS name",
			core.Recipe{{Op: "Decode NetBIOS Name", Args: []any{65}}},
		},
		// A short name is space-padded to 16 bytes on encode; decode trims the
		// trailing padding back off (verified against the CyberChef-server oracle).
		{
			"NetBIOS short round trip", "AB", "AB",
			core.Recipe{
				{Op: "Encode NetBIOS Name", Args: []any{65}},
				{Op: "Decode NetBIOS Name", Args: []any{65}},
			},
		},
	})
}

func TestNetBIOSEncodeTooLong(t *testing.T) {
	// Input longer than 16 bytes encodes to nothing (matches the oracle).
	if out, err := runOp(t, "Encode NetBIOS Name", strings.Repeat("A", 17), 65); err != nil || out != "" {
		t.Fatalf("encode(>16) = %q, %v", out, err)
	}
}
