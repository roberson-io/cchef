package ops

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// xpressHex reads a hex string that may be written in space-separated bytes.
func xpressHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(strings.ReplaceAll(s, " ", ""))
	if err != nil {
		t.Fatalf("decode %q: %v", s, err)
	}
	return b
}

// xpressRecipe reads the hex the fixtures are written in and decompresses it,
// the way CyberChef's own cases are set up.
var xpressRecipe = core.Recipe{
	{Op: "From Hex", Args: []any{"Space"}},
	{Op: "XPRESS Decompress"},
}

// TestXPRESSDecompressFixtures covers CyberChef's own cases
// (CyberChef's tests/operations/tests/XPRESS.mjs), which are in turn the worked
// examples from MS-XCA section 3.1 and the forms around them.
func TestXPRESSDecompressFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"worked example, all literals",
			"0000000047484f53542f2f5245434f5645522064617461207265636f7665727920656e67ffffff07696e652e0a",
			"GHOST//RECOVER data recovery engine.\n",
			xpressRecipe,
		},
		{
			// The low nibble of 0x0f selects a raw length and its high nibble
			// feeds the next match, which reads the trailing LE16 0x0126.
			"shared nibble and LE16 length",
			"ffffff1f61626317000fff2601",
			strings.Repeat("abc", 100),
			xpressRecipe,
		},
		{"nibble length, 13 + 10", "ffffff7f6e07000d", strings.Repeat("n", 24), xpressRecipe},
		{
			// The low nibble of 0x21 gives the first match its length, the
			// literal 'B' does not clear the byte, and the second match takes
			// the high nibble.
			"shared half-byte across a literal",
			"ffffff5f41070021420700",
			strings.Repeat("A", 12) + strings.Repeat("B", 13),
			xpressRecipe,
		},
		{
			"one-byte raw length, 0xd7 + 25",
			"ffff0000413142324333443445354636473748387f000fd7",
			strings.Repeat("A1B2C3D4E5F6G7H8", 16),
			xpressRecipe,
		},
		{
			// 0xfffc + 3 = 65535, then a shared-nibble match with LE16 0x116d
			// + 3 = 4464, for 70000 bytes in all.
			"LE16 raw lengths",
			"ffffff7f570700fffffcffffffff6d11",
			strings.Repeat("W", 70000),
			xpressRecipe,
		},
	})
}

// TestXPRESSDecompressRawLengths covers the boundaries of the raw-length forms
// a nibble of 15 selects: the one-byte form, and the LE16 and LE32 forms with a
// value at the lowest the format allows.
func TestXPRESSDecompressRawLengths(t *testing.T) {
	for _, tc := range []struct {
		name, in string
		want     int
	}{
		{"one-byte, 0xd7 + 25", "ffffff7f 41 0700 0f d7", 241},
		{"LE16 at the lowest value allowed", "ffffff7f 41 0700 0f ff 1600", 26},
		{"LE32 at the lowest value allowed", "ffffff7f 41 0700 0f ff 0000 16000000", 26},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := xpressDecode(xpressHex(t, tc.in))
			if err != nil {
				t.Fatalf("decompress: %v", err)
			}
			if want := bytes.Repeat([]byte("A"), tc.want); !bytes.Equal(got, want) {
				t.Errorf("got %d bytes, want %d", len(got), tc.want)
			}
		})
	}
}

func TestXPRESSDecompressErrors(t *testing.T) {
	for _, tc := range []struct{ name, in, want string }{
		{"empty input", "", "XPRESS: truncated flag group"},
		{"flag group cut short", "000000", "XPRESS: truncated flag group"},
		{"literal with nothing left", "00000000", "XPRESS: truncated literal"},
		{"match word cut in half", "00000080 41", "XPRESS: truncated match"},
		{"shared nibble missing", "00000080 0700", "XPRESS: truncated shared nibble"},
		{"raw length byte missing", "00000080 0700 0f", "XPRESS: truncated raw length"},
		{"raw length LE16 missing", "00000080 0700 0f ff", "XPRESS: truncated raw length"},
		{"raw length LE32 missing", "00000080 0700 0f ff 0000", "XPRESS: truncated raw length"},
		{"raw length below the minimum", "00000080 0700 0f ff 1500", "XPRESS: invalid match length"},
		{"match reaching before the output", "00000080 0000", "XPRESS: match offset out of range"},
		{
			"match longer than the output cap",
			"00000040 41 0700 0f ff 0000 00ffffff",
			"XPRESS: decompression ratio too large",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := runOp(t, "XPRESS Decompress", string(xpressHex(t, tc.in)))
			if err == nil {
				t.Fatal("input was accepted")
			}
			if err.Error() != tc.want {
				t.Errorf("error = %q, want %q", err, tc.want)
			}
		})
	}
}

// xpressHuffmanFixture is the worked example from MS-XCA section 3.1: 256 bytes
// of code lengths and then the coded stream.
const xpressHuffmanFixture = "00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 " +
	"03 00 00 00 00 00 00 05 00 00 00 00 00 00 00 00 " +
	"00 00 00 00 00 00 00 00 00 00 06 00 00 00 00 00 " +
	"50 66 55 55 66 65 55 45 65 55 55 65 55 05 00 00 " +
	"00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 " +
	"00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 " +
	"00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 " +
	"00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 " +
	"05 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 " +
	"00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 " +
	"05 00 00 00 00 00 00 00 00 00 00 00 00 00 00 50 " +
	"00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 " +
	"00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 " +
	"00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 " +
	"00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 " +
	"00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 " +
	"b4 e3 a9 8f 5e e7 62 8e bc 5f ac 28 47 19 40 42 " +
	"98 aa eb 89 7c da 20 5c 61 96 e4 b6 ff 38 01 00 " +
	"00"

// TestXPRESSHuffmanDecompressFixture covers CyberChef's own case
// (CyberChef's tests/operations/tests/XPRESS.mjs).
func TestXPRESSHuffmanDecompressFixture(t *testing.T) {
	want := strings.Repeat("The quick brown fox jumps over the lazy dog. ", 8)
	runCases(t, []opCase{
		{
			"worked example", xpressHuffmanFixture, want,
			core.Recipe{
				{Op: "From Hex", Args: []any{"Space"}},
				{Op: "XPRESS LZ77+Huffman Decompress", Args: []any{360}},
			},
		},
	})
}

// TestXPRESSHuffmanDecompressSize covers the decompressed size, which the
// format does not record and so has to be given.
func TestXPRESSHuffmanDecompressSize(t *testing.T) {
	for _, tc := range []struct {
		name string
		size any
		want string
	}{
		{"short of the real size", 359, "XPRESS: output exceeds declared size"},
		{"past the real size", 361, "XPRESS: corrupt end-of-data marker"},
		{"zero", 0, "XPRESS: invalid decompressed size"},
		{"negative", -1, "XPRESS: invalid decompressed size"},
		{"past the output cap", 32<<20 + 1, "XPRESS: invalid decompressed size"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := string(xpressHex(t, xpressHuffmanFixture))
			_, err := runOp(t, "XPRESS LZ77+Huffman Decompress", in, tc.size)
			if err == nil {
				t.Fatal("input was accepted")
			}
			if err.Error() != tc.want {
				t.Errorf("error = %q, want %q", err, tc.want)
			}
		})
	}
}

// TestXPRESSHuffmanDecompressSizeInteger pins the integer check on the size,
// which counts bytes and so is never fractional. CyberChef leaves the argument
// open and truncates a fractional value instead.
func TestXPRESSHuffmanDecompressSizeInteger(t *testing.T) {
	op, _ := core.Default.Get("XPRESS LZ77+Huffman Decompress")
	_, err := core.CoerceArgs(op.Args(), []any{360.5})
	if err == nil {
		t.Fatal("a fractional decompressed size was accepted")
	}
	if want := "Decompressed size must be an integer."; err.Error() != want {
		t.Errorf("got %q, want %q", err.Error(), want)
	}
}

// xpressHuffmanSymbols are the symbols the crafted streams below use. Sixteen
// symbols of four bits each is the smallest table the format allows, since the
// codes must fill the code space exactly. Their codes are 0 to 15 in this
// order: canonical order is by length and then by symbol.
//
// Symbols 65 to 67 are the literals 'A' to 'C' and 256 is end-of-data. The
// match symbols are read as ((s-256)>>4) offset bits and ((s-256)&15) as the
// length: 257 is a four-byte match at offset 1, 271 takes a raw length at
// offset 1, 274 is a five-byte match whose one offset bit chooses between
// offsets 2 and 3, and 465 takes thirteen offset bits, which is more than the
// register is ever left holding. The rest pad the table out to sixteen.
var xpressHuffmanSymbols = []int{0, 1, 2, 3, 4, 5, 6, 65, 66, 67, 256, 257, 258, 271, 274, 465}

// xpressHuffmanStream prefixes body with the code-length table for
// xpressHuffmanSymbols: 256 bytes of 4-bit lengths, the even symbol of each
// pair in the low nibble and the odd one in the high.
func xpressHuffmanStream(t *testing.T, body string) []byte {
	t.Helper()
	table := make([]byte, xpressTableBytes)
	for _, s := range xpressHuffmanSymbols {
		if s%2 == 0 {
			table[s/2] |= 4
		} else {
			table[s/2] |= 4 << 4
		}
	}
	return append(table, xpressHex(t, body)...)
}

// TestXPRESSHuffmanForms covers each thing a symbol can stand for: a literal,
// end-of-data, a match with and without offset bits, and the three raw-length
// forms a length nibble of 15 selects.
func TestXPRESSHuffmanForms(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		size       int
		want       string
	}{
		{"literals", "9a 78 0000 0000", 3, "ABC"},
		// Symbol 256 before the output is full is a match of length 3 at
		// offset 1 rather than the end of the data.
		{"end-of-data symbol mid-stream", "a0 7a 0000 0000", 4, "AAAA"},
		{"match without offset bits", "a0 7b 0000 0000", 5, "AAAAA"},
		{"match with an offset bit", "9e 78 00 d0 0000", 8, "ABCABCAB"},
		{"raw length byte, 20 + 18", "a0 7d 0000 14 0000", 39, strings.Repeat("A", 39)},
		{"raw length LE16, 40 + 3", "a0 7d 0000 ff 2800 0000", 44, strings.Repeat("A", 44)},
		{"raw length LE32, 40 + 3", "a0 7d 0000 ff 0000 28000000 0000", 44, strings.Repeat("A", 44)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := xpressDecodeHuffman(xpressHuffmanStream(t, tc.body), tc.size)
			if err != nil {
				t.Fatalf("decompress: %v", err)
			}
			if string(got) != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestXPRESSHuffmanErrors(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		size       int
		want       string
	}{
		{"one literal too many", "a0 78 0000 0000", 1, "XPRESS: output exceeds declared size"},
		{"end-of-data with no output", "00 a0 0000 0000", 4, "XPRESS: corrupt end-of-data marker"},
		{
			"end-of-data with less than a match left",
			"a0 78 0000 0000", 4, "XPRESS: corrupt end-of-data marker",
		},
		{"match reaching before the output", "00 ba 0000 0000", 4, "XPRESS: match offset out of range"},
		{"match past the declared size", "a0 7b 0000 0000", 4, "XPRESS: output exceeds declared size"},
		{
			"match longer than the output cap",
			"a0 7d 0000 ff 0000 00000004 0000", 4096, "XPRESS: decompression ratio too large",
		},
		{"bit stream ending mid-symbol", "7777 a077", 16, "XPRESS: truncated bit stream"},
		{"bit stream too short to preload", "00 7a", 1, "XPRESS: truncated bit stream"},
		// Symbol 465 wants thirteen offset bits, more than the register holds
		// after a symbol, so it always calls for another word.
		{"bit stream ending before the offset bits", "7777 00f0", 16, "XPRESS: truncated bit stream"},
		{"raw length with nothing behind it", "00 7d 0000", 4, "XPRESS: truncated raw length"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := xpressDecodeHuffman(xpressHuffmanStream(t, tc.body), tc.size)
			if err == nil {
				t.Fatal("input was accepted")
			}
			if err.Error() != tc.want {
				t.Errorf("error = %q, want %q", err, tc.want)
			}
		})
	}
}

// TestXPRESSHuffmanTableErrors covers the two things that can be wrong with the
// code-length table itself, which the crafted streams above always get right.
func TestXPRESSHuffmanTableErrors(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   []byte
		want string
	}{
		{"table cut short", make([]byte, xpressTableBytes-1), "XPRESS: truncated Huffman table"},
		{"no codes at all", make([]byte, xpressTableBytes+8), "XPRESS: invalid Huffman code lengths"},
		// Every symbol one bit long asks for far more of the code space than
		// there is. CyberChef writes past the end of its table and catches it
		// on the count afterwards; the table here is a fixed size, so the
		// overflow has to be caught as it happens.
		{
			"codes claiming more than the whole code space",
			append(bytes.Repeat([]byte{0x11}, xpressTableBytes), make([]byte, 8)...),
			"XPRESS: invalid Huffman code lengths",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := xpressDecodeHuffman(tc.in, 1)
			if err == nil {
				t.Fatal("input was accepted")
			}
			if err.Error() != tc.want {
				t.Errorf("error = %q, want %q", err, tc.want)
			}
		})
	}
}
