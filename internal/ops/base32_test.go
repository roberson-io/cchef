package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// Cases transcribed from CyberChef tests/operations/tests/Base32.mjs.
func TestBase32Fixtures(t *testing.T) {
	std := "A-Z2-7="
	ext := "0-9A-V="
	runCases(t, []opCase{
		{
			"To Base32 Standard: nothing", "", "",
			core.Recipe{{Op: "To Base32", Args: []any{std}}},
		},
		{
			"To Base32 Standard", "HELLO BASE32", "JBCUYTCPEBBECU2FGMZA====",
			core.Recipe{{Op: "To Base32", Args: []any{std}}},
		},
		{
			"To Base32 Hex Extended", "HELLO BASE32 EXTENDED", "912KOJ2F41142KQ56CP20HAOAH2KSH258G======",
			core.Recipe{{Op: "To Base32", Args: []any{ext}}},
		},
		// Non-BMP Unicode custom alphabets (gchq/CyberChef#2380). Mahjong tiles.
		{
			"To Base32: non-BMP Unicode alphabet", "hello", "🀝🀈🀐🀔🀖🀀🀊🀟",
			core.Recipe{{Op: "To Base32", Args: []any{"🀇🀈🀉🀊🀋🀌🀍🀎🀏🀙🀚🀛🀜🀝🀞🀟🀠🀡🀐🀑🀒🀓🀔🀕🀖🀗🀘🀀🀁🀂🀃🀅"}}},
		},
		{
			"To Base32: 32-char Unicode alphabet omits padding", "hell", "🀝🀈🀐🀔🀖🀀🀇",
			core.Recipe{{Op: "To Base32", Args: []any{"🀇🀈🀉🀊🀋🀌🀍🀎🀏🀙🀚🀛🀜🀝🀞🀟🀠🀡🀐🀑🀒🀓🀔🀕🀖🀗🀘🀀🀁🀂🀃🀅"}}},
		},

		{
			"From Base32 Standard: nothing", "", "",
			core.Recipe{{Op: "From Base32", Args: []any{std, false}}},
		},
		{
			"From Base32 Standard", "JBCUYTCPEBBECU2FGMZA====", "HELLO BASE32",
			core.Recipe{{Op: "From Base32", Args: []any{std, false}}},
		},
		{
			"From Base32 Hex Extended", "912KOJ2F41142KQ56CP20HAOAH2KSH258G======", "HELLO BASE32 EXTENDED",
			core.Recipe{{Op: "From Base32", Args: []any{ext, false}}},
		},
		// Remove-non-alphabet strips the "!!!" before decoding (JBSWY3DP -> "Hello").
		{
			"From Base32 remove non-alphabet", "JBSWY3DP!!!", "Hello",
			core.Recipe{{Op: "From Base32", Args: []any{std, true}}},
		},

		{
			"Base32 round trip", "The quick brown fox", "The quick brown fox",
			core.Recipe{
				{Op: "To Base32", Args: []any{std}},
				{Op: "From Base32", Args: []any{std, false}},
			},
		},
	})
}

func TestBase32Branches(t *testing.T) {
	// Three input bytes exercise the 3-byte padding case.
	if _, err := runOp(t, "To Base32", "abc", "A-Z2-7="); err != nil {
		t.Fatalf("To Base32(abc): %v", err)
	}
	// A non-alphabet char with stripping disabled reaches the -1 lookup.
	if _, err := runOp(t, "From Base32", "!!!!!!!!", "A-Z2-7=", false); err != nil {
		t.Fatalf("From Base32(!): %v", err)
	}
}

// TestBase32EmitGroup documents decoding one 8-symbol group (as decoded indices,
// 32 = the "=" pad) into its bytes, honouring padding: "MY"=f, "MZXW6"=foo.
func TestBase32EmitGroup(t *testing.T) {
	if got := base32EmitGroup(nil, [8]int{12, 24, 32, 32, 32, 32, 32, 32}); string(got) != "f" {
		t.Fatalf("f: %q", got)
	}
	if got := base32EmitGroup(nil, [8]int{12, 25, 23, 22, 30, 32, 32, 32}); string(got) != "foo" {
		t.Fatalf("foo: %q", got)
	}
}
