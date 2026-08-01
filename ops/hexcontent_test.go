package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

// TestToHexContent covers the SNORT hex-content encoder. Outputs are oracle-
// verified (no upstream fixtures exist).
func TestToHexContent(t *testing.T) {
	special := func(sp bool) core.Recipe {
		return core.Recipe{{Op: "To Hex Content", Args: []any{"Only special chars", sp}}}
	}
	inclSpaces := func(sp bool) core.Recipe {
		return core.Recipe{{Op: "To Hex Content", Args: []any{"Only special chars including spaces", sp}}}
	}
	allChars := func(sp bool) core.Recipe {
		return core.Recipe{{Op: "To Hex Content", Args: []any{"All chars", sp}}}
	}
	runCases(t, []opCase{
		{"special one byte", "foo=bar", "foo|3d|bar", special(false)},
		{"special all printable kept", "abc123", "abc123", special(false)},
		{"special consecutive no spaces", "a==b", "a|3d3d|b", special(false)},
		{"special consecutive spaces", "a==b", "a|3d 3d|b", special(true)},
		{"special three run spaces", "a=<>b", "a|3d 3c 3e|b", special(true)},
		{"special tab", "x\ty", "x|09|y", special(false)},
		{"trailing special closes block", "foo=", "foo|3d|", special(false)},
		{"trailing consecutive special spaces", "a==", "a|3d 3d|", special(true)},
		{"space kept by default", "hello world", "hello world", special(false)},
		{"space converted", "hello world", "hello|20|world", inclSpaces(false)},
		{"consecutive spaces converted", "a  b", "a|20 20|b", inclSpaces(true)},
		{"all chars no spaces", "foo=bar", "|666f6f3d626172|", allChars(false)},
		{"all chars spaces", "foo=bar", "|66 6f 6f 3d 62 61 72|", allChars(true)},
	})
}

// TestFromHexContent covers the SNORT hex-content decoder (byteArray output shown
// as hex via To Hex). Outputs are oracle-verified.
func TestFromHexContent(t *testing.T) {
	dec := core.Recipe{{Op: "From Hex Content"}, {Op: "To Hex", Args: []any{"None"}}}
	runCases(t, []opCase{
		{"basic", "foo|3d|bar", "666f6f3d626172", dec},
		{"spaced hex content", "foo|3d 3e|bar", "666f6f3d3e626172", dec},
		{"no pipes is raw", "hello", "68656c6c6f", dec},
		// An odd trailing nibble is kept as its own byte (fromHex byteLen 2).
		{"odd nibble kept", "a|3d5|b", "613d0562", dec},
		// Whitespace-only content decodes to nothing.
		{"whitespace content", "x|  |y", "7879", dec},
		{"single byte", "|3d|", "3d", dec},
		// Non-hex pipe content is left raw (the regex does not match).
		{"non-hex pipes raw", "|GG|", "7c47477c", dec},
		{"uppercase hex", "up|3D|low", "75703d6c6f77", dec},
	})
}

// TestHexContentRoundTrip round-trips through both operations.
func TestHexContentRoundTrip(t *testing.T) {
	runCases(t, []opCase{
		{
			"round trip", "foo=bar", "foo=bar",
			core.Recipe{
				{Op: "To Hex Content", Args: []any{"Only special chars", false}},
				{Op: "From Hex Content"},
			},
		},
	})
}
