package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// Transcribed from ../CyberChef/tests/operations/tests/BCD.mjs.
func TestBCDFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"To BCD: default 0", "0", "0000",
			core.Recipe{{Op: "To BCD", Args: []any{"8 4 2 1", true, false, "Nibbles"}}},
		},
		{
			"To BCD: unpacked nibbles", "1234567890",
			"0000 0001 0000 0010 0000 0011 0000 0100 0000 0101 0000 0110 0000 0111 0000 1000 0000 1001 0000 0000",
			core.Recipe{{Op: "To BCD", Args: []any{"8 4 2 1", false, false, "Nibbles"}}},
		},
		{
			"To BCD: packed, signed bytes", "1234567890",
			"00000001 00100011 01000101 01100111 10001001 00001100",
			core.Recipe{{Op: "To BCD", Args: []any{"8 4 2 1", true, true, "Bytes"}}},
		},
		{
			"To BCD: packed, signed nibbles, 8 4 -2 -1", "-1234567890",
			"0000 0111 0110 0101 0100 1011 1010 1001 1000 1111 0000 1101",
			core.Recipe{{Op: "To BCD", Args: []any{"8 4 -2 -1", true, true, "Nibbles"}}},
		},
		{
			"From BCD: default 0", "0000", "0",
			core.Recipe{{Op: "From BCD", Args: []any{"8 4 2 1", true, false, "Nibbles"}}},
		},
		{
			"From BCD: packed, signed bytes",
			"00000001 00100011 01000101 01100111 10001001 00001101", "-1234567890",
			core.Recipe{{Op: "From BCD", Args: []any{"8 4 2 1", true, true, "Bytes"}}},
		},
		{
			"From BCD: Excess-3, unpacked, unsigned",
			"00000100 00000101 00000110 00000111 00001000 00001001 00001010 00001011 00001100 00000011",
			"1234567890",
			core.Recipe{{Op: "From BCD", Args: []any{"Excess-3", false, false, "Nibbles"}}},
		},
		{
			"BCD: raw 4 2 2 1, packed, signed", "1234567890", "1234567890",
			core.Recipe{
				{Op: "To BCD", Args: []any{"4 2 2 1", true, true, "Raw"}},
				{Op: "From BCD", Args: []any{"4 2 2 1", true, true, "Raw"}},
			},
		},
	})
}

// TestBCDErrors exercises the operations' error and validation branches, which
// the upstream fixtures do not cover.
func TestBCDErrors(t *testing.T) {
	cases := []struct {
		name  string
		op    string
		input string
		args  []any
	}{
		{"To BCD rejects fractional values", "To BCD", "12.5", []any{"8 4 2 1", true, false, "Nibbles"}},
		{"To BCD rejects non-numeric input", "To BCD", "abc", []any{"8 4 2 1", true, false, "Nibbles"}},
		{"From BCD rejects non-binary characters", "From BCD", "0002", []any{"8 4 2 1", true, false, "Nibbles"}},
		{"From BCD rejects nibbles absent from the scheme", "From BCD", "1010", []any{"8 4 2 1", true, false, "Nibbles"}},
		{"From BCD rejects empty input", "From BCD", "", []any{"8 4 2 1", true, false, "Nibbles"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := runOp(t, c.op, c.input, c.args...); err == nil {
				t.Fatalf("%s: expected an error, got none", c.name)
			}
		})
	}
}
