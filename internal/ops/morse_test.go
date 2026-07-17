package ops

// Morse code tests. The two fixtures in ../CyberChef/tests/operations/tests/
// MorseCode.mjs only cover "SOS", so the remaining expected values were produced
// by the CyberChef-server oracle across the format options, delimiters, and the
// UTF-16 code-unit iteration quirk (non-BMP characters split into two "letters").

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

func TestToMorseCode(t *testing.T) {
	to := func(fmt, ld, wd string) core.Recipe {
		return core.Recipe{{Op: "To Morse Code", Args: []any{fmt, ld, wd}}}
	}
	runCases(t, []opCase{
		{"SOS", "SOS", "... --- ...", to("-/.", "Space", "Line feed")},
		{"Hello World", "Hello World", ".... . .-.. .-.. ---\n.-- --- .-. .-.. -..", to("-/.", "Space", "Line feed")},
		{"underscore format", "SOS", "... ___ ...", to("_/.", "Space", "Line feed")},
		{"Dash/Dot format", "SOS", "DotDotDot DashDashDash DotDotDot", to("Dash/Dot", "Space", "Line feed")},
		{"DASH/DOT format", "SOS", "DOTDOTDOT DASHDASHDASH DOTDOTDOT", to("DASH/DOT", "Space", "Line feed")},
		{"dash/dot format", "SOS", "dotdotdot dashdashdash dotdotdot", to("dash/dot", "Space", "Line feed")},
		{"comma/forward-slash delims", "AB CD", ".-,-.../-.-.,-..", to("-/.", "Comma", "Forward slash")},
		{"space/backslash delims", "AB CD", ".- -...\\-.-. -..", to("-/.", "Space", "Backslash")},
		{"digits and punctuation", "Hi 123!", ".... ..\n.---- ..--- ...-- -.-.--", to("-/.", "Space", "Line feed")},
		// Non-Morse characters map to empty letters but still emit a delimiter.
		{"non-Morse BMP char", "a€b", ".-  -...", to("-/.", "Space", "Line feed")},
		// UTF-16 quirk: a non-BMP char is two code units -> two empty letters.
		{"non-BMP char alone", "😀", " ", to("-/.", "Space", "Line feed")},
		{"non-BMP char in word", "a😀b", ".-   -...", to("-/.", "Space", "Line feed")},
	})
}

func TestFromMorseCode(t *testing.T) {
	from := func(ld, wd string) core.Recipe {
		return core.Recipe{{Op: "From Morse Code", Args: []any{ld, wd}}}
	}
	runCases(t, []opCase{
		{"SOS", "... --- ...", "SOS", from("Space", "Line feed")},
		{"multi-word (line feed)", ".... . .-.. .-.. ---\n.-- --- .-. .-.. -..", "HELLO WORLD", from("Space", "Line feed")},
		{"digits", ".---- ..--- ...--", "123", from("Space", "Line feed")},
		{"dash/dot words", "dash dot dot", "TEE", from("Space", "Line feed")},
		{"dot words", "dot dot dot", "EEE", from("Space", "Line feed")},
		{"comma/forward-slash delims", ".-,-.../-.-.,-..", "AB CD", from("Comma", "Forward slash")},
		// Unicode middle-dot and minus-sign variants decode too.
		{"unicode dot/dash variants", "·−·−·−", ".", from("Space", "Line feed")},
		// Unknown signals are dropped.
		{"unknown signal dropped", "unknownsig ...", "S", from("Space", "Line feed")},
	})
}

// TestCharRepSlashes covers the delimiter names Morse adds to charRep.
func TestCharRepSlashes(t *testing.T) {
	if got := charRep("Forward slash"); got != "/" {
		t.Fatalf("Forward slash: got %q", got)
	}
	if got := charRep("Backslash"); got != "\\" {
		t.Fatalf("Backslash: got %q", got)
	}
}
