package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

// The Braille lookup has no CyberChef test fixture; these vectors were produced
// by and differential-tested against the CyberChef-server oracle. The two
// strings are the full ASCII/dot6 lookup tables, so the round-trip cases below
// exercise every one of the 64 mappings at once.
const (
	brailleASCIITable = ` A1B'K2L@CIF/MSP"E3H9O6R^DJG>NTQ,*5<-U8V.%[$+X!&;:4\0Z7(_?W]#Y)=`
	brailleDot6Table  = `⠀⠁⠂⠃⠄⠅⠆⠇⠈⠉⠊⠋⠌⠍⠎⠏⠐⠑⠒⠓⠔⠕⠖⠗⠘⠙⠚⠛⠜⠝⠞⠟⠠⠡⠢⠣⠤⠥⠦⠧⠨⠩⠪⠫⠬⠭⠮⠯⠰⠱⠲⠳⠴⠵⠶⠷⠸⠹⠺⠻⠼⠽⠾⠿`
)

// TestToBrailleOracle covers To Braille against oracle-verified outputs.
func TestToBrailleOracle(t *testing.T) {
	runCases(t, []opCase{
		{"Hello", "Hello", "⠓⠑⠇⠇⠕", core.Recipe{{Op: "To Braille"}}},
		{"lowercase is upcased", "hello", "⠓⠑⠇⠇⠕", core.Recipe{{Op: "To Braille"}}},
		{"mixed text", "Hello, World! 123", "⠓⠑⠇⠇⠕⠠⠀⠺⠕⠗⠇⠙⠮⠀⠂⠆⠒", core.Recipe{{Op: "To Braille"}}},
		{"non-ASCII passthrough", "cч", "⠉ч", core.Recipe{{Op: "To Braille"}}},
		{"full lookup table", brailleASCIITable, brailleDot6Table, core.Recipe{{Op: "To Braille"}}},
		{"empty", "", "", core.Recipe{{Op: "To Braille"}}},
	})
}

// TestFromBrailleOracle covers From Braille against oracle-verified outputs.
func TestFromBrailleOracle(t *testing.T) {
	runCases(t, []opCase{
		{"HELLO", "⠓⠑⠇⠇⠕", "HELLO", core.Recipe{{Op: "From Braille"}}},
		{"mixed with plain chars", "⠓⠑⠇⠇⠕ x", "HELLO x", core.Recipe{{Op: "From Braille"}}},
		{"8-dot braille passthrough", "⣿", "⣿", core.Recipe{{Op: "From Braille"}}},
		{"full lookup table", brailleDot6Table, brailleASCIITable, core.Recipe{{Op: "From Braille"}}},
		{"empty", "", "", core.Recipe{{Op: "From Braille"}}},
	})
}

// TestBrailleRoundTrip confirms text survives a To/From round-trip (upper-cased,
// since the lookup only stores upper-case letters).
func TestBrailleRoundTrip(t *testing.T) {
	runCases(t, []opCase{
		{
			"round-trip", "HELLO, WORLD! 123", "HELLO, WORLD! 123",
			core.Recipe{{Op: "To Braille"}, {Op: "From Braille"}},
		},
	})
}
