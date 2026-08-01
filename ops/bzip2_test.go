package ops

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// bzip2Golden is one case from testdata/bzip2.jsonl. The input is described
// rather than stored, and the answer is pinned by its digest, so that a corpus
// running to hundreds of kilobytes costs a few lines of fixture.
type bzip2Golden struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	Spec struct {
		Hex  string `json:"hex"`
		Byte int    `json:"byte"`
		Len  int    `json:"len"`
		N    int    `json:"n"`
		Seed int64  `json:"seed"`
	} `json:"spec"`
	BlockSize     int    `json:"blockSize"`
	CompressedLen int    `json:"compressedLen"`
	SHA256        string `json:"sha256"`
	Hex           string `json:"hex"`
}

// bzip2Sentence is the text the repeated-text cases are built from.
const bzip2Sentence = "The cat sat on the mat. "

// bzip2Random is the byte source the corpus generator used: an ordinary 64-bit
// linear congruential step, taking the top byte of each state.
func bzip2Random(n int, seed int64) []byte {
	x := uint64(seed)
	out := make([]byte, n)
	for i := range out {
		x = x*6364136223846793005 + 1442695040888963407
		out[i] = byte(x >> 56)
	}
	return out
}

// input rebuilds the bytes a golden was made from.
func (g bzip2Golden) input(t *testing.T) []byte {
	t.Helper()
	switch g.Kind {
	case "raw":
		b, err := hex.DecodeString(g.Spec.Hex)
		if err != nil {
			t.Fatalf("%s: bad hex: %v", g.Name, err)
		}
		return b
	case "range":
		b := make([]byte, g.Spec.Len)
		for i := range b {
			b[i] = byte(i)
		}
		return b
	case "repeat":
		b := make([]byte, g.Spec.Len)
		for i := range b {
			b[i] = byte(g.Spec.Byte)
		}
		return b
	case "text":
		out := make([]byte, 0, len(bzip2Sentence)*g.Spec.N)
		for range g.Spec.N {
			out = append(out, bzip2Sentence...)
		}
		return out
	case "random":
		return bzip2Random(g.Spec.Len, g.Spec.Seed)
	}
	t.Fatalf("%s: unknown input kind %q", g.Name, g.Kind)
	return nil
}

// loadBzip2Goldens reads the corpus.
func loadBzip2Goldens(t *testing.T) []bzip2Golden {
	t.Helper()
	f, err := os.Open("testdata/bzip2.jsonl")
	if err != nil {
		t.Fatalf("open goldens: %v", err)
	}
	defer func() { _ = f.Close() }()

	var out []bzip2Golden
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		var g bzip2Golden
		if err := json.Unmarshal(sc.Bytes(), &g); err != nil {
			t.Fatalf("parse goldens: %v", err)
		}
		out = append(out, g)
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("read goldens: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("no goldens loaded")
	}
	return out
}

// TestBzip2CompressGoldens checks the compressed bytes against libbzip2's own,
// taken from CyberChef through its Node API. The corpus covers the run-length
// boundaries, a block of every byte value, incompressible input, and inputs
// long enough to span several blocks at the smaller block sizes.
func TestBzip2CompressGoldens(t *testing.T) {
	for _, g := range loadBzip2Goldens(t) {
		t.Run(g.Name, func(t *testing.T) {
			in := g.input(t)
			out, err := runOp(t, "Bzip2 Compress", string(in), float64(g.BlockSize), float64(30))
			if err != nil {
				t.Fatalf("Bzip2 Compress: %v", err)
			}
			if len(out) != g.CompressedLen {
				t.Errorf("compressed to %d bytes, want %d", len(out), g.CompressedLen)
			}
			if got := fmt.Sprintf("%x", sha256.Sum256([]byte(out))); got != g.SHA256 {
				t.Errorf("digest %s, want %s", got, g.SHA256)
				if g.Hex != "" {
					t.Errorf("got  %x\nwant %s", out, g.Hex)
				}
			}
		})
	}
}

// TestBzip2CompressBlockSize checks that the block size reaches the stream
// header, where it is written as the digit itself.
func TestBzip2CompressBlockSize(t *testing.T) {
	for size := 1; size <= 9; size++ {
		out, err := runOp(t, "Bzip2 Compress", "The cat sat on the mat.", float64(size), float64(30))
		if err != nil {
			t.Fatalf("block size %d: %v", size, err)
		}
		want := fmt.Sprintf("BZh%d", size)
		if len(out) < len(want) {
			t.Errorf("block size %d: output is %d bytes, too short to hold a header", size, len(out))
			continue
		}
		if got := out[:len(want)]; got != want {
			t.Errorf("block size %d: header %q, want %q", size, got, want)
		}
	}
}

// TestBzip2CompressWorkFactorIsInert pins that the work factor changes nothing
// about the output. In libbzip2 it only decides when the block sort gives up on
// its faster path and falls back to a slower one, and both paths produce the
// same transform, so it is a speed control and nothing more.
func TestBzip2CompressWorkFactorIsInert(t *testing.T) {
	in := "The cat sat on the mat. " + string(bzip2Random(4096, 7))
	first := ""
	for _, wf := range []float64{0, 1, 30, 100, 250} {
		out, err := runOp(t, "Bzip2 Compress", in, float64(9), wf)
		if err != nil {
			t.Fatalf("work factor %v: %v", wf, err)
		}
		if out == "" {
			t.Fatalf("work factor %v: no output", wf)
		}
		if first == "" {
			first = out
			continue
		}
		if out != first {
			t.Errorf("work factor %v changed the output", wf)
		}
	}
}

// TestBzip2CompressEmptyInput covers CyberChef's refusal to compress nothing.
func TestBzip2CompressEmptyInput(t *testing.T) {
	_, err := runOp(t, "Bzip2 Compress", "", float64(9), float64(30))
	if err == nil {
		t.Fatal("expected an error for empty input")
	}
	if err.Error() != "Please provide an input." {
		t.Errorf("error %q, want %q", err.Error(), "Please provide an input.")
	}
}

// TestBzip2CompressRoundTrips checks each golden back through the decompressor.
func TestBzip2CompressRoundTrips(t *testing.T) {
	for _, g := range loadBzip2Goldens(t) {
		t.Run(g.Name, func(t *testing.T) {
			in := string(g.input(t))
			compressed, err := runOp(t, "Bzip2 Compress", in, float64(g.BlockSize), float64(30))
			if err != nil {
				t.Fatalf("Bzip2 Compress: %v", err)
			}
			back, err := runOp(t, "Bzip2 Decompress", compressed, false)
			if err != nil {
				t.Fatalf("Bzip2 Decompress: %v", err)
			}
			if back != in {
				t.Errorf("round trip changed the data (%d bytes in, %d out)", len(in), len(back))
			}
		})
	}
}

// bzip2Stream is CyberChef's own fixture: "The cat sat on the mat." compressed.
const bzip2Stream = "425a6839314159265359b218ed630000031380400104002a438c00200021a9ea6" +
	"01a10003202185d5ed68ca6442f1e177245385090b218ed63"

// bzip2Bytes decodes one of the hex streams above.
func bzip2Bytes(t *testing.T, s string) string {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("bad hex: %v", err)
	}
	return string(b)
}

// TestBzip2DecompressFixture covers CyberChef's own case
// (../CyberChef/tests/operations/tests/Compress.mjs).
func TestBzip2DecompressFixture(t *testing.T) {
	runCases(t, []opCase{
		{
			"Bzip2 decompress",
			bzip2Bytes(t, bzip2Stream),
			"The cat sat on the mat.",
			core.Recipe{{Op: "Bzip2 Decompress", Args: []any{false}}},
		},
	})
}

// TestBzip2DecompressLowMemoryIsInert pins that the low-memory switch changes
// nothing about the result. It chooses between two decoders inside libbzip2
// that differ in how much they allocate, not in what they produce.
func TestBzip2DecompressLowMemoryIsInert(t *testing.T) {
	in := bzip2Bytes(t, bzip2Stream)
	for _, low := range []bool{false, true} {
		out, err := runOp(t, "Bzip2 Decompress", in, low)
		if err != nil {
			t.Fatalf("low memory %v: %v", low, err)
		}
		if out != "The cat sat on the mat." {
			t.Errorf("low memory %v: got %q", low, out)
		}
	}
}

// TestBzip2DecompressTrailingBytes checks that whatever follows a finished
// stream is ignored, as it is by the bzip2 command.
func TestBzip2DecompressTrailingBytes(t *testing.T) {
	out, err := runOp(t, "Bzip2 Decompress", bzip2Bytes(t, bzip2Stream)+"XXXX", false)
	if err != nil {
		t.Fatalf("Bzip2 Decompress: %v", err)
	}
	if out != "The cat sat on the mat." {
		t.Errorf("got %q", out)
	}
}

// TestBzip2DecompressConcatenatedStreams checks that every stream in the input
// is read, which is what the bzip2 command and Python's bz2 module both do.
// CyberChef stops after the first and silently drops the rest.
func TestBzip2DecompressConcatenatedStreams(t *testing.T) {
	one := bzip2Bytes(t, bzip2Stream)
	out, err := runOp(t, "Bzip2 Decompress", one+one+one, false)
	if err != nil {
		t.Fatalf("Bzip2 Decompress: %v", err)
	}
	want := strings.Repeat("The cat sat on the mat.", 3)
	if out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}

// TestBzip2DecompressErrors covers the input the operation refuses.
func TestBzip2DecompressErrors(t *testing.T) {
	full := bzip2Bytes(t, bzip2Stream)
	for _, tc := range []struct {
		name, input, want string
	}{
		{"empty", "", "Please provide an input."},
		{"not bzip2", "not bzip2 at all", "not a bzip2 stream"},
		{"header only", full[:4], "truncated bzip2 stream"},
		{"truncated", full[:20], "truncated bzip2 stream"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := runOp(t, "Bzip2 Decompress", tc.input, false)
			if err == nil {
				t.Fatal("expected an error")
			}
			if err.Error() != tc.want {
				t.Errorf("error %q, want %q", err.Error(), tc.want)
			}
		})
	}
}

// TestBzip2DecompressCorruptStream checks that a stream which starts well and
// then does not hold together is reported in the reader's own words, which say
// more about what went wrong than a general message could.
func TestBzip2DecompressCorruptStream(t *testing.T) {
	for _, tc := range []struct {
		name  string
		spoil func([]byte)
		want  string
	}{
		{
			"checksum", func(b []byte) { b[len(b)-3] ^= 0xff },
			"bzip2 data invalid: file checksum mismatch",
		},
		{
			"coded data", func(b []byte) { b[32] ^= 0xff },
			"bzip2 data invalid: Huffman length out of range",
		},
		{
			"origin pointer", func(b []byte) { b[15] ^= 0xff },
			"bzip2 data invalid: origPtr out of bounds",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spoiled := []byte(bzip2Bytes(t, bzip2Stream))
			tc.spoil(spoiled)
			_, err := runOp(t, "Bzip2 Decompress", string(spoiled), false)
			if err == nil {
				t.Fatal("expected an error")
			}
			if err.Error() != tc.want {
				t.Errorf("error %q, want %q", err.Error(), tc.want)
			}
		})
	}
}

// TestBzip2EncodeEmpty covers the encoder given nothing to do. The operation
// refuses empty input before reaching this, but the encoder still has to write
// a well-formed empty stream, which is what the bzip2 command produces for an
// empty file.
func TestBzip2EncodeEmpty(t *testing.T) {
	out := bzip2Encode(nil, 9)
	if got, want := string(out[:4]), "BZh9"; got != want {
		t.Errorf("header %q, want %q", got, want)
	}
	back, err := bzip2Decode(out)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(back) != 0 {
		t.Errorf("decoded %d bytes, want none", len(back))
	}
}

// TestBzip2GroupCount covers the thresholds deciding how many Huffman tables a
// block uses, including the boundaries between them.
func TestBzip2GroupCount(t *testing.T) {
	for _, tc := range []struct {
		symbols, want int
	}{
		{1, 2},
		{199, 2},
		{200, 3},
		{599, 3},
		{600, 4},
		{1199, 4},
		{1200, 5},
		{2399, 5},
		{2400, 6},
		{100000, 6},
	} {
		if got := bzGroupCount(tc.symbols); got != tc.want {
			t.Errorf("bzGroupCount(%d) = %d, want %d", tc.symbols, got, tc.want)
		}
	}
}

// TestBzip2CodeLengthsAreLimited covers the retry that keeps a code from
// growing too long. Frequencies rising like the Fibonacci numbers are the worst
// case for a Huffman tree — each symbol pairs with the whole of the rest — so
// they drive the depth past the limit and force the weights to be flattened.
func TestBzip2CodeLengthsAreLimited(t *testing.T) {
	const alphaSize = 28
	freq := make([]int32, alphaSize)
	freq[0], freq[1] = 1, 1
	for i := 2; i < alphaSize; i++ {
		freq[i] = freq[i-1] + freq[i-2]
	}

	lengths := bzCodeLengths(freq, alphaSize, bzMaxCodeLen)
	if len(lengths) != alphaSize {
		t.Fatalf("got %d lengths, want %d", len(lengths), alphaSize)
	}
	for i, l := range lengths {
		if l < 1 || int(l) > bzMaxCodeLen {
			t.Errorf("symbol %d has length %d, want between 1 and %d", i, l, bzMaxCodeLen)
		}
	}

	// The lengths must still describe a usable code: Kraft's sum over them
	// cannot exceed one, and for a complete code it is exactly one.
	var kraft float64
	for _, l := range lengths {
		kraft += 1 / float64(int64(1)<<l)
	}
	if kraft > 1.0000001 {
		t.Errorf("Kraft sum is %v, which describes no code", kraft)
	}
}

// TestBzip2TreeDepths covers the walk from each symbol up to the root, on a
// tree built by hand: symbols 0 and 1 pair under node 3, and that pairs with
// symbol 2 under the root, so the depths are 2, 2 and 1.
func TestBzip2TreeDepths(t *testing.T) {
	// parent is one-based, as the tree is built: entries 1..3 are the symbols.
	parent := []int32{-2, 4, 4, 5, 5, -1}
	lengths, tooLong := bzTreeDepths(parent, 3, bzMaxCodeLen)
	if want := []byte{2, 2, 1}; string(lengths) != string(want) {
		t.Errorf("lengths %v, want %v", lengths, want)
	}
	if tooLong {
		t.Error("reported too long for a tree two deep")
	}
	if _, tooLong := bzTreeDepths(parent, 3, 1); !tooLong {
		t.Error("did not report too long against a limit of 1")
	}
}

// TestBzip2BuildTree covers the tree building on its own: the two rarest
// symbols must end up sharing a parent, and every symbol must reach the root.
func TestBzip2BuildTree(t *testing.T) {
	const alphaSize = 4
	weight := make([]int64, alphaSize*2+2)
	parent := make([]int32, alphaSize*2+2)
	for i, w := range []int64{100, 1, 1, 100} {
		weight[i+1] = w << 8
	}
	bzBuildTree(weight, parent, alphaSize)

	if parent[2] != parent[3] {
		t.Errorf("the two rarest symbols have parents %d and %d, want the same",
			parent[2], parent[3])
	}
	for i := 1; i <= alphaSize; i++ {
		steps := 0
		for node := int32(i); parent[node] >= 0; node = parent[node] {
			steps++
			if steps > alphaSize*2 {
				t.Fatalf("symbol %d never reaches the root", i-1)
			}
		}
		if steps == 0 {
			t.Errorf("symbol %d has no parent", i-1)
		}
	}
}
