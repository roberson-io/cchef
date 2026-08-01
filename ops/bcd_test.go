package ops

import (
	"math/big"
	"slices"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// bcd8421 is the plain "8 4 2 1" scheme (digit d encodes to nibble d), which
// keeps the helper tests below easy to read.
var bcd8421 = bcdEncodingLookup["8 4 2 1"]

// TestBcdPackNibbles documents the packing helper: two nibbles per byte, high
// nibble first, with an odd final nibble occupying the high half of a trailing
// byte.
func TestBcdPackNibbles(t *testing.T) {
	cases := []struct {
		name    string
		nibbles []int
		want    []int
	}{
		{"empty", nil, nil},
		{"one pair", []int{0x1, 0x2}, []int{0x12}},
		{"odd count keeps high nibble", []int{0x1, 0x2, 0x3}, []int{0x12, 0x30}},
		{"sign nibble C", []int{0x1, 0xC}, []int{0x1C}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := bcdPackNibbles(c.nibbles); !slices.Equal(got, c.want) {
				t.Fatalf("bcdPackNibbles(%v) = %v, want %v", c.nibbles, got, c.want)
			}
		})
	}
}

// TestBcdEncodeNibbles documents the core encoder: digit nibbles, the optional
// sign nibble (with the even-length leading-zero rule), and packed vs unpacked
// output streams.
func TestBcdEncodeNibbles(t *testing.T) {
	cases := []struct {
		name         string
		n            int64
		packed       bool
		signed       bool
		wantNibbles  []int
		wantByteVals []int
	}{
		{"packed unsigned odd", 123, true, false, []int{1, 2, 3}, []int{0x12, 0x30}},
		{"unpacked interleaves null high nibbles", 12, false, false, []int{0, 1, 0, 2}, []int{1, 2}},
		{"signed positive appends C", 1, true, true, []int{1, 0xC}, []int{0x1C}},
		{"signed negative appends D", -1, true, true, []int{1, 0xD}, []int{0x1D}},
		{"signed even prepends leading zero", 12, true, true, []int{0, 1, 2, 0xC}, []int{0x01, 0x2C}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			nibbles, bytes := bcdEncodeNibbles(big.NewInt(c.n), bcd8421, c.packed, c.signed)
			if !slices.Equal(nibbles, c.wantNibbles) {
				t.Fatalf("nibbles = %v, want %v", nibbles, c.wantNibbles)
			}
			if !slices.Equal(bytes, c.wantByteVals) {
				t.Fatalf("bytes = %v, want %v", bytes, c.wantByteVals)
			}
		})
	}
}

// TestBcdParseNibbles documents parsing raw input into a nibble stream for each
// input format.
func TestBcdParseNibbles(t *testing.T) {
	nib, err := bcdParseNibbles([]byte("0001 0010"), "Nibbles")
	if err != nil || !slices.Equal(nib, []int{1, 2}) {
		t.Fatalf("Nibbles: got %v, err %v", nib, err)
	}
	raw, err := bcdParseNibbles([]byte{0x12, 0x30}, "Raw")
	if err != nil || !slices.Equal(raw, []int{1, 2, 3, 0}) {
		t.Fatalf("Raw: got %v, err %v", raw, err)
	}
	if _, err := bcdParseNibbles([]byte("0002"), "Bytes"); err == nil {
		t.Fatal("expected error on non-binary character")
	}
}

// TestBcdDecodeDigits documents mapping nibble codes back to decimal digits and
// the error for a code absent from the scheme.
func TestBcdDecodeDigits(t *testing.T) {
	got, err := bcdDecodeDigits([]int{1, 2, 3}, bcd8421)
	if err != nil || got != "123" {
		t.Fatalf("got %q, err %v", got, err)
	}
	if _, err := bcdDecodeDigits([]int{10}, bcd8421); err == nil {
		t.Fatal("expected error for nibble not in scheme")
	}
}

// TestBcdFormatOutput documents the three output renderings.
func TestBcdFormatOutput(t *testing.T) {
	if got := bcdFormatOutput([]int{1, 2}, nil, "Nibbles").String(); got != "0001 0010" {
		t.Fatalf("Nibbles: %q", got)
	}
	if got := bcdFormatOutput(nil, []int{0x12}, "Bytes").String(); got != "00010010" {
		t.Fatalf("Bytes: %q", got)
	}
	if got := bcdFormatOutput(nil, []int{0x12, 0x30}, "Raw").Bytes(); !slices.Equal(got, []byte{0x12, 0x30}) {
		t.Fatalf("Raw: %v", got)
	}
}

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
