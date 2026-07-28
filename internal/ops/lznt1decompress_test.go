package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// cleanHex drops the spaces the streams below are grouped with for reading.
func cleanHex(s string) string { return strings.ReplaceAll(s, " ", "") }

// lznt1Golden is one case from testdata/lznt1.jsonl: a stream, and the bytes it
// was made from.
//
// CyberChef cannot write LZNT1, so the streams were built from data already
// known, and CyberChef was asked what they meant only as a check. Where its
// answer differs from what went in, CyberChefDiffers says what it gave; the
// expected output stays the data itself.
type lznt1Golden struct {
	Name             string `json:"name"`
	StreamHex        string `json:"streamHex"`
	PlainLen         int    `json:"plainLen"`
	PlainSHA256      string `json:"plainSHA256"`
	PlainHex         string `json:"plainHex"`
	CyberChefDiffers string `json:"cyberchefDiffers"`
}

// TestLZNT1Fixtures covers CyberChef's own case
// (../CyberChef/tests/operations/tests/LZNT1Decompress.mjs).
func TestLZNT1Fixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"LZNT1 Decompress",
			"\x1a\xb0\x00compress\x00edtestda\x04ta\x07\x88alot",
			"compressedtestdatacompressedalot",
			core.Recipe{{Op: "LZNT1 Decompress", Args: []any{}}},
		},
	})
}

// TestLZNT1DecompressGoldens reads a corpus of streams back into the data they
// were made from: stored chunks and compressed ones, repeats reaching the
// furthest back the format allows, and inputs either side of a chunk boundary.
func TestLZNT1DecompressGoldens(t *testing.T) {
	for _, g := range readJSONL[lznt1Golden](t, "testdata/lznt1.jsonl") {
		t.Run(g.Name, func(t *testing.T) {
			out, err := runOp(t, "LZNT1 Decompress", string(unhex(t, g.StreamHex)))
			if err != nil {
				t.Fatalf("LZNT1 Decompress: %v", err)
			}
			if len(out) != g.PlainLen {
				t.Errorf("read back %d bytes, want %d", len(out), g.PlainLen)
			}
			if sum := digest([]byte(out)); sum != g.PlainSHA256 {
				t.Errorf("digest %s, want %s", sum, g.PlainSHA256)
			}
		})
	}
}

// TestLZNT1ReadsAChunkOfOneByte pins the first of two faults in CyberChef's
// reader. A chunk records its length less one, so a chunk holding a single byte
// has a size field of zero — which CyberChef takes for the end of the stream,
// dropping that byte and everything after it. MS-XCA ends the stream on a
// header of 0x0000 instead, which is what happens here.
func TestLZNT1ReadsAChunkOfOneByte(t *testing.T) {
	// A stored chunk holding "A", then one holding "BCDEFGHIJK".
	stream := "0030" + "41" + "0930" + "4243444546474849 4a4b"
	out, err := runOp(t, "LZNT1 Decompress", string(unhex(t, cleanHex(stream))))
	if err != nil {
		t.Fatalf("LZNT1 Decompress: %v", err)
	}
	if out != "ABCDEFGHIJK" {
		t.Errorf("got %q, want %q (CyberChef gives nothing at all)", out, "ABCDEFGHIJK")
	}
}

// TestLZNT1EndsOnAZeroHeader covers the terminator the format really has: a
// header of 0x0000, after which anything else in the input is ignored.
func TestLZNT1EndsOnAZeroHeader(t *testing.T) {
	chunk := "0930" + "4243444546474849 4a4b"
	stream := chunk + "0000" + chunk
	out, err := runOp(t, "LZNT1 Decompress", string(unhex(t, cleanHex(stream))))
	if err != nil {
		t.Fatalf("LZNT1 Decompress: %v", err)
	}
	if out != "BCDEFGHIJK" {
		t.Errorf("got %q, want %q", out, "BCDEFGHIJK")
	}
}

// TestLZNT1DecompressRejectsBadInput covers streams that promise more than they
// hold. CyberChef reads the first of these without complaint, a byte short.
func TestLZNT1DecompressRejectsBadInput(t *testing.T) {
	cases := []struct {
		name string
		hex  string
	}{
		{"a stored chunk a byte short", "0930" + "4243444546474849 4a"},
		{"a stored chunk with no data at all", "0930"},
		{"a compressed chunk a byte short", "03b0" + "00 41 42"},
		{"a match distance cut in half", "01b0" + "01 41"},
		{"a match reaching before the start", "03b0" + "02 41 0010"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := runOp(t, "LZNT1 Decompress", string(unhex(t, cleanHex(c.hex)))); err == nil {
				t.Fatal("read a stream that should have been refused")
			}
		})
	}
}

// TestLZNT1DecompressEmptyAndShort covers input with no chunk in it, which is
// not an error: there is simply nothing to read.
func TestLZNT1DecompressEmptyAndShort(t *testing.T) {
	for _, in := range []string{"", "00", "41"} {
		out, err := runOp(t, "LZNT1 Decompress", string(unhex(t, in)))
		if err != nil {
			t.Fatalf("LZNT1 Decompress %q: %v", in, err)
		}
		if out != "" {
			t.Errorf("input %q gave %q, want nothing", in, out)
		}
	}
}
