package deflateraw

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"testing"
)

// deflateCase is one recorded vector: some input, and the bytes zlib produces
// for it at its highest compression level.
type deflateCase struct {
	Name  string `json:"name"`
	Input string `json:"input"`
	Want  string `json:"want"`
}

// TestZlibDeflateMatchesZlib checks the encoder against zlib's own output. The
// vectors come from zlib itself and cover each decision the encoder makes: a
// block cheaper stored than coded, one cheaper under the fixed trees than its
// own, matches at the greatest length, a match far enough away to be worth
// less than a literal, more symbols than one block holds, and more input than
// the window holds.
func TestZlibDeflateMatchesZlib(t *testing.T) {
	for _, c := range loadDeflateCases(t) {
		t.Run(c.Name, func(t *testing.T) {
			in, err := hex.DecodeString(c.Input)
			if err != nil {
				t.Fatalf("decode input: %v", err)
			}
			want, err := hex.DecodeString(c.Want)
			if err != nil {
				t.Fatalf("decode expected output: %v", err)
			}
			got := Deflate(in)
			if !bytes.Equal(got, want) {
				t.Errorf("compressed %d bytes to %d, want %d%s",
					len(in), len(got), len(want), deflateFirstDifference(got, want))
			}
		})
	}
}

// TestZlibDeflateRoundTrips checks that whatever the encoder produces is a
// stream the standard decoder reads back, which the recorded vectors alone
// would not prove if they were ever regenerated from the encoder itself.
func TestZlibDeflateRoundTrips(t *testing.T) {
	for _, c := range loadDeflateCases(t) {
		t.Run(c.Name, func(t *testing.T) {
			in, _ := hex.DecodeString(c.Input)
			r, err := zlib.NewReader(bytes.NewReader(Deflate(in)))
			if err != nil {
				t.Fatalf("open the compressed stream: %v", err)
			}
			defer func() { _ = r.Close() }()
			got, err := io.ReadAll(r)
			if err != nil {
				t.Fatalf("read the compressed stream: %v", err)
			}
			if !bytes.Equal(got, in) {
				t.Errorf("decompressed to %d bytes, want %d", len(got), len(in))
			}
		})
	}
}

// loadDeflateCases reads the recorded vectors.
func loadDeflateCases(t *testing.T) []deflateCase {
	t.Helper()
	file, err := os.Open("testdata/deflate_zlib.jsonl")
	if err != nil {
		t.Fatalf("open vectors: %v", err)
	}
	defer func() { _ = file.Close() }()

	var cases []deflateCase
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for scanner.Scan() {
		if len(scanner.Bytes()) == 0 {
			continue
		}
		var c deflateCase
		if err := json.Unmarshal(scanner.Bytes(), &c); err != nil {
			t.Fatalf("parse vector: %v", err)
		}
		cases = append(cases, c)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("read vectors: %v", err)
	}
	return cases
}

// deflateFirstDifference locates where two streams part company, which is more
// use than their lengths when a vector fails.
func deflateFirstDifference(got, want []byte) string {
	for i := range min(len(got), len(want)) {
		if got[i] != want[i] {
			return "; first difference at byte " + hex.EncodeToString([]byte{byte(i >> 8), byte(i)}) +
				": got " + hex.EncodeToString(got[i:i+1]) + " want " + hex.EncodeToString(want[i:i+1])
		}
	}
	return ""
}

// TestZlibDeflateLengthOverflow covers the redistribution that runs when a
// tree comes out deeper than the format allows. Eight equally rare symbols sit
// under a balanced subtree with a chain of heavier ones above, which puts their
// leaves below the limit and leaves the level above them empty, so the search
// for a level to move a leaf down from has to walk past it.
//
// The result is checked as a Huffman tree rather than against recorded bytes:
// no code may exceed the limit, and the code space must be exactly filled.
func TestZlibDeflateLengthOverflow(t *testing.T) {
	s := &dfState{
		dynLTree: make([]int, dfHeapSize*2),
		dynDTree: make([]int, (2*dfDCodes+1)*2),
		blTree:   make([]int, (2*dfBLCodes+1)*2),
	}
	s.lDesc = dfTreeDesc{s.dynLTree, 0, dfLiteralDesc}

	// Seven rare symbols, which the end marker joins to make eight.
	for symbol := range 7 {
		s.dynLTree[symbol*2] = 1
	}
	s.dynLTree[dfEndBlock*2] = 1
	weight := 9
	for k := range 13 {
		s.dynLTree[(100+k)*2] = weight
		weight *= 2
	}

	s.buildTree(&s.lDesc)

	space := 0.0
	deepest := 0
	for symbol := 0; symbol <= s.lDesc.maxCode; symbol++ {
		length := s.dynLTree[symbol*2+1]
		if length == 0 {
			continue
		}
		if length > dfMaxBits {
			t.Fatalf("symbol %d has a code of %d bits, which is past the limit of %d",
				symbol, length, dfMaxBits)
		}
		deepest = max(deepest, length)
		space += 1 / float64(int(1)<<length)
	}
	if deepest != dfMaxBits {
		t.Errorf("deepest code is %d bits; the tree should have reached the limit", deepest)
	}
	if space != 1 {
		t.Errorf("the codes fill %v of the code space, want exactly 1", space)
	}
}
