package ops

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// SM3 known-answer vectors from GM/T 0004-2012 (the two worked examples in the
// standard).
func TestSM3(t *testing.T) {
	cases := []struct{ in, want string }{
		{"abc", "66c7f0f462eeedd9d1f2d46bdc10e4e24167c4875cf2f7a2297da02b8f4ba8e0"},
		{strings.Repeat("abcd", 16), "debe9ff92275b8a138604889c18e5a4d6fdb70e5387e5765293dcba39c0c5732"},
		{"", "1ab21d8355cfa17f8e61194831e81a8f22bec8c728fefb747ed035eb5082aa2b"},
	}
	for _, c := range cases {
		if got := hex.EncodeToString(sm3Sum([]byte(c.in))); got != c.want {
			t.Fatalf("sm3(%q) = %s, want %s", c.in, got, c.want)
		}
	}
}

func sm3Recipe(length, rounds int) core.Recipe {
	return core.Recipe{{Op: "SM3", Args: []any{length, rounds}}}
}

// Vectors derived from the CyberChef-server oracle (the exact crypto-api SM3 the
// operation wraps), covering the default hash plus the Length-truncation and
// Rounds behaviours — including the > 64 rounds out-of-bounds quirk where the
// state collapses toward the SM3 initial value.
func TestSM3Op(t *testing.T) {
	full := "66c7f0f462eeedd9d1f2d46bdc10e4e24167c4875cf2f7a2297da02b8f4ba8e0"
	hundredA := strings.Repeat("a", 100)
	runCases(t, []opCase{
		{"SM3: abc default", "abc", full, sm3Recipe(256, 64)},
		{"SM3: length 128", "abc", "66c7f0f462eeedd9d1f2d46bdc10e4e2", sm3Recipe(128, 64)},
		{"SM3: length 200 truncates to 6 words", "abc", "66c7f0f462eeedd9d1f2d46bdc10e4e24167c4875cf2f7a2", sm3Recipe(200, 64)},
		{"SM3: length 96", "abc", "66c7f0f462eeedd9d1f2d46b", sm3Recipe(96, 64)},
		{"SM3: length 32", "abc", "66c7f0f4", sm3Recipe(32, 64)},
		{"SM3: length below one word gives full digest", "abc", full, sm3Recipe(16, 64)},
		{"SM3: length zero gives full digest", "abc", full, sm3Recipe(0, 64)},
		{"SM3: negative length gives empty output", "abc", "", sm3Recipe(-32, 64)},
		{"SM3: length 512 zero-extends", "abc", full + strings.Repeat("0", 64), sm3Recipe(512, 64)},
		{"SM3: rounds 16", "abc", "47517f71354625c185913a9e4d10b713c896a6fa28d86199c92d3c1c6be815c8", sm3Recipe(256, 16)},
		{"SM3: rounds 17", "abc", "0aeaecde7dc5dba7b20ab22f483f7e49af1a604977c8aeec2a1419079a5bdc1f", sm3Recipe(256, 17)},
		{"SM3: rounds 65 (one over)", "abc", "7380166f5c535422e39a82801c5c90bc8e961195fe39cc919bcfb8537a0b4028", sm3Recipe(256, 65)},
		{"SM3: rounds 100", "abc", "7380166f4914b2b9172442d7da8a06009be1390eb2bbe34856d99cc9a1e4c569", sm3Recipe(256, 100)},
		{"SM3: rounds 128", "abc", "7380166f4914b2b9172442d7da8a0600351b8f5f9ddc8b2f074cfeb79889ac88", sm3Recipe(256, 128)},
		{"SM3: rounds 200 collapses to IV", "abc", "7380166f4914b2b9172442d7da8a0600a96f30bc163138aae38dee4db0fb0e4e", sm3Recipe(256, 200)},
		{"SM3: empty input", "", "1ab21d8355cfa17f8e61194831e81a8f22bec8c728fefb747ed035eb5082aa2b", sm3Recipe(256, 64)},
		{"SM3: multi-block input", hundredA, "0c105d5a46a65fdf0a0938283db2517ea87f176de84786f443cb78802aaa03de", sm3Recipe(256, 64)},
		{"SM3: multi-block, rounds 32", hundredA, "264b64e05166dfc187094cd245001b5f2d7ef59524ef0410e3b3c6104085e707", sm3Recipe(256, 32)},
	})
}
