package ops

// MD6 wraps the self-contained node-md6 package (a standard MD6 implementation)
// and has no CyberChef fixtures, so these vectors are generated from that exact
// package via node — the "md6 FTW" and empty-string values match the published
// MD6-256 test vectors. See PLAN's MD6 note.

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// md6Recipe builds an MD6 recipe: Size, Levels, Key.
func md6Recipe(size, levels int, key string) core.Recipe {
	return core.Recipe{{Op: "MD6", Args: []any{size, levels, key}}}
}

func TestMD6(t *testing.T) {
	long := strings.Repeat("A", 1000)
	runCases(t, []opCase{
		{"empty 256", "", "bca38b24a804aa37d821d31af00f5598230122c5bbfc4c4ad5ed40e4258f04ca", md6Recipe(256, 64, "")},
		{"md6 FTW", "md6 FTW", "7bfaa624f661a683be2a3b2007493006a30a7845ee1670e499927861a8e74cce", md6Recipe(256, 64, "")},
		{"Hello 256", "Hello, World!", "ce5effce32637e6b8edaacc9284b873c3fd4e66f9779a79df67eb4a82dda8230", md6Recipe(256, 64, "")},
		{"Hello 512", "Hello, World!", "1333db8caf3c69ce346f2dacef9805803f9d4c8594e4b20856ce1b0a70ccb0e68028b0b749d4aa25cbe489a2eb51260c0d7bd16d32dd4d7bfbd1f3ae8aa03260", md6Recipe(512, 64, "")},
		{"Hello 128", "Hello, World!", "229b1e1e0a0725416b8ec8cc0911facf", md6Recipe(128, 64, "")},
		{"Hello 224", "Hello, World!", "5383f38e6b88aab5ccdfc68e2b36870a3d78d18c2ac15b058f86934c", md6Recipe(224, 64, "")},
		// Non-multiple-of-8 size exercises crop's remainder masking.
		{"Hello 250", "Hello, World!", "d3c524cb2eb79439fda3b182e9932dce758f5fa30b6c475474da0868b0820c00", md6Recipe(250, 64, "")},
		// Keyed (k>0 selects r = max(80, ...)).
		{"keyed", "Hello, World!", "e9b61d08a55ef1177596b4fa04140a918653de84346da7a7e65b0ded2a5c6a94", md6Recipe(256, 64, "secretkey")},
		{"keyed short", "abc", "10bb35cfaaea9da0125a6a2762d06dbfdbdd6510713170f54ef002fc65925d46", md6Recipe(256, 64, "k")},
		// Small level counts force sequential mode for a multi-block input.
		{"seq levels=1", long, "7b44b0ab258f9f6da032bd39df13a62137b900895c863c3d88e67684ca9f3a0a", md6Recipe(256, 1, "")},
		{"seq levels=0", long, "32a19a4ea884e92c7c5dc3d9d24fa12985bd3077e98ea175d2fe3e78a0655906", md6Recipe(256, 0, "")},
		// Size 0 is clamped to 1 bit by the library.
		{"size 0", "Hello, World!", "00", md6Recipe(0, 64, "")},
		{"size 1", "Hello, World!", "00", md6Recipe(1, 64, "")},
		// Non-ASCII input exercises the multi-byte UTF-8 encoding branches.
		{"2-byte utf8", "café", "4c4085f481366259526c797acb9d6d860130aa298a6c528b4fed94c665560e6a", md6Recipe(256, 64, "")},
		{"3-byte utf8", "中文", "5ad2ce53c330d84f7e8ffea854a11e361244045be17a924533bcc084aeff1fc6", md6Recipe(256, 64, "")},
		// A key longer than 64 bytes is truncated to 64.
		{"long key", "Hello", "00a0811a68149d221aeba1ae4461178a4f4231ef41ef89358e23b9dd1ec27f13", md6Recipe(256, 64, strings.Repeat("K", 70))},
	})
}

// Parameter validation matches CyberChef's error text.
func TestMD6Errors(t *testing.T) {
	cases := []struct {
		size, levels int
		wantErr      string
	}{
		{-1, 64, "Size must be between 0 and 512"},
		{513, 64, "Size must be between 0 and 512"},
		{256, -1, "Levels must be greater than 0"},
	}
	for _, c := range cases {
		_, err := runOp(t, "MD6", "x", c.size, c.levels, "")
		if err == nil || err.Error() != c.wantErr {
			t.Fatalf("size=%d levels=%d: got %v want %q", c.size, c.levels, err, c.wantErr)
		}
	}
}
