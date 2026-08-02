package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

func parityRecipe(mode, position, mode3, delim string) core.Recipe {
	return core.Recipe{{Op: "Parity Bit", Args: []any{mode, position, mode3, delim}}}
}

// Fixtures transcribed from CyberChef's tests/operations/tests/ParityBit.mjs.
func TestParityBitFixtures(t *testing.T) {
	toBinThenParity := func(mode, position, delim string) core.Recipe {
		return core.Recipe{
			{Op: "To Binary", Args: []any{"Space"}},
			{Op: "Parity Bit", Args: []any{mode, position, "Encode", delim}},
		}
	}
	runCases(t, []opCase{
		{"even, prepend, even 1s", "01010101 10101010", "001010101 10101010", parityRecipe("Even Parity", "Start", "Encode", "")},
		{"even, prepend, odd 1s", "01010101 10101011", "101010101 10101011", parityRecipe("Even Parity", "Start", "Encode", "")},
		{"even, append, odd 1s", "01010101 10101011", "01010101 101010111", parityRecipe("Even Parity", "End", "Encode", "")},
		{"odd, prepend, even 1s", "01010101 10101010", "101010101 10101010", parityRecipe("Odd Parity", "Start", "Encode", "")},
		{"odd, prepend, odd 1s", "01010101 10101011", "001010101 10101011", parityRecipe("Odd Parity", "Start", "Encode", "")},
		{"odd, append, odd 1s", "01010101 10101011", "01010101 101010110", parityRecipe("Odd Parity", "End", "Encode", "")},
		{
			"even, prepend per byte", "hello world!",
			"101101000 001100101 001101100 001101100 001101111 100100000 001110111 001101111 001110010 001101100 101100100 000100001",
			toBinThenParity("Even Parity", "Start", " "),
		},
		{
			"odd, append per byte", "hello world!",
			"011010000 011001011 011011001 011011001 011011111 001000000 011101111 011011111 011100101 011011001 011001000 001000011",
			toBinThenParity("Odd Parity", "End", " "),
		},
	})
}

// Decode removes the parity bit; empty input is returned unchanged; a non-binary
// character is rejected.
func TestParityBitDecodeAndErrors(t *testing.T) {
	if got, err := runOp(t, "Parity Bit", "001010101", "Even Parity", "Start", "Decode", ""); err != nil || got != "01010101" {
		t.Fatalf("decode Start = %q, %v", got, err)
	}
	if got, err := runOp(t, "Parity Bit", "010101011", "Even Parity", "End", "Decode", ""); err != nil || got != "01010101" {
		t.Fatalf("decode End = %q, %v", got, err)
	}
	if got, err := runOp(t, "Parity Bit", "", "Even Parity", "Start", "Encode", ""); err != nil || got != "" {
		t.Fatalf("empty = %q, %v", got, err)
	}
	if got, err := runOp(t, "Parity Bit", "10x01", "Even Parity", "Start", "Encode", ""); err == nil || got != "" || err.Error() != `unexpected character encountered: "x"` {
		t.Fatalf("bad char: got %q, %v", got, err)
	}
	// Per-token decode with a delimiter.
	if got, err := runOp(t, "Parity Bit", "101010101 010101010", "Even Parity", "Start", "Decode", " "); err != nil || got != "01010101 10101010" {
		t.Fatalf("delimited decode = %q, %v", got, err)
	}
	// A bad character inside a delimited token surfaces the encode error.
	if _, err := runOp(t, "Parity Bit", "1010 10x0", "Even Parity", "Start", "Encode", " "); err == nil || err.Error() != `unexpected character encountered: "x"` {
		t.Fatalf("delimited bad char: got %v", err)
	}
}
