package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

func msdRecipe() core.Recipe {
	return core.Recipe{{Op: "Microsoft Script Decoder", Args: []any{}}}
}

// Microsoft Script Decoder has no CyberChef fixtures; these vectors are verified
// against the CyberChef-server oracle (the decode logic was differentially checked
// on 14 valid-format samples, byte-for-byte). Non-matching input yields "".
func TestMicrosoftScriptDecoderFixtures(t *testing.T) {
	runCases(t, []opCase{
		{"not encoded", "no encoded script here", "", msdRecipe()},
		{"malformed marker", "#@~^short", "", msdRecipe()},
		{"decode 1", "#@~^AAAAAA==,ACfs$BBBBBB==^#~@", "[EH3m[", msdRecipe()},
		{"decode 2", "#@~^AAAAAA==K+C-VI`wT5US;{H}x*sBBBBBB==^#~@", "P6HvGRUp]Q~wu_yZ=5m", msdRecipe()},
		{"pass-through escapes", "#@~^AAAAAA==ab@*cd@&ef@$ghBBBBBB==^#~@", "xi>(L\n!D@Nw", msdRecipe()},
	})
}

// TestMSDecodeHighByte covers the non-ASCII (byte >= 128) branch directly: such
// bytes do not advance the substitution index and pass through unchanged.
func TestMSDecodeHighByte(t *testing.T) {
	if got := msDecode("\xff"); got != "\xff" {
		t.Errorf("msDecode(0xff) = %q, want 0xff", got)
	}
}
