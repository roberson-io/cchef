package ops

import (
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
	})
}
