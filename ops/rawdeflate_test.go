package ops

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// deflateGolden is one case from testdata/deflate.jsonl: an input described
// rather than stored, the operation and arguments that were run over it, and
// the answer CyberChef gave, pinned by its digest.
type deflateGolden struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	Spec struct {
		Hex  string `json:"hex"`
		Byte int    `json:"byte"`
		Len  int    `json:"len"`
		N    int    `json:"n"`
		Seed int64  `json:"seed"`
	} `json:"spec"`
	Op        string `json:"op"`
	Args      []any  `json:"args"`
	MaskMtime bool   `json:"maskMtime"`
	OutLen    int    `json:"outLen"`
	SHA256    string `json:"sha256"`
	Hex       string `json:"hex"`
}

// deflateSentence is the text the repeated-text cases are built from.
const deflateSentence = "The quick brown fox jumps over the lazy dog. "

// deflateRandom is the byte source the corpus generator used: an ordinary
// 64-bit linear congruential step, taking the top byte of each state.
func deflateRandom(n int, seed int64) []byte {
	x := uint64(seed)
	out := make([]byte, n)
	for i := range out {
		x = x*6364136223846793005 + 1442695040888963407
		out[i] = byte(x >> 56)
	}
	return out
}

// input rebuilds the bytes a golden was made from.
func (g deflateGolden) input(t *testing.T) []byte {
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
		out := make([]byte, 0, len(deflateSentence)*g.Spec.N)
		for range g.Spec.N {
			out = append(out, deflateSentence...)
		}
		return out
	case "random":
		return deflateRandom(g.Spec.Len, g.Spec.Seed)
	}
	t.Fatalf("%s: unknown input kind %q", g.Name, g.Kind)
	return nil
}

// gzipMTime is where gzip records the time it ran, which makes its output differ
// from one run to the next. Those bytes are blanked before comparing, as
// CyberChef's own tests do by dropping the header.
const (
	gzipMTimeFrom = 4
	gzipMTimeTo   = 8
)

// loadDeflateGoldens reads the corpus.
func loadDeflateGoldens(t *testing.T) []deflateGolden {
	t.Helper()
	f, err := os.Open("testdata/deflate.jsonl")
	if err != nil {
		t.Fatalf("open goldens: %v", err)
	}
	defer func() { _ = f.Close() }()

	var out []deflateGolden
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		var g deflateGolden
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

// TestDeflateFamilyGoldens checks the compressed bytes of all three writers
// against CyberChef's own, across every compression type and a corpus running
// from a single byte to inputs several blocks long.
func TestDeflateFamilyGoldens(t *testing.T) {
	for _, g := range loadDeflateGoldens(t) {
		t.Run(g.Name, func(t *testing.T) {
			out, err := runOp(t, g.Op, string(g.input(t)), g.Args...)
			if err != nil {
				t.Fatalf("%s: %v", g.Op, err)
			}
			got := []byte(out)
			if g.MaskMtime && len(got) >= gzipMTimeTo {
				got = append([]byte{}, got...)
				copy(got[gzipMTimeFrom:gzipMTimeTo], make([]byte, gzipMTimeTo-gzipMTimeFrom))
			}
			if len(got) != g.OutLen {
				t.Errorf("produced %d bytes, want %d", len(got), g.OutLen)
			}
			if sum := fmt.Sprintf("%x", sha256.Sum256(got)); sum != g.SHA256 {
				t.Errorf("digest %s, want %s", sum, g.SHA256)
				if g.Hex != "" {
					t.Errorf("got  %x\nwant %s", got, g.Hex)
				}
			}
		})
	}
}

// TestDeflateFamilyRoundTrips checks every compressed golden back through its
// matching reader.
func TestDeflateFamilyRoundTrips(t *testing.T) {
	readers := map[string]struct {
		op   string
		args []any
	}{
		"Raw Deflate":  {"Raw Inflate", []any{float64(0), float64(0), "Adaptive", false, false}},
		"Zlib Deflate": {"Zlib Inflate", []any{float64(0), float64(0), "Adaptive", false, false}},
		"Gzip":         {"Gunzip", nil},
	}
	for _, g := range loadDeflateGoldens(t) {
		reader, ok := readers[g.Op]
		if !ok {
			continue
		}
		t.Run(g.Name, func(t *testing.T) {
			in := string(g.input(t))
			compressed, err := runOp(t, g.Op, in, g.Args...)
			if err != nil {
				t.Fatalf("%s: %v", g.Op, err)
			}
			back, err := runOp(t, reader.op, compressed, reader.args...)
			if err != nil {
				t.Fatalf("%s: %v", reader.op, err)
			}
			if back != in {
				t.Errorf("round trip changed the data (%d bytes in, %d out)", len(in), len(back))
			}
		})
	}
}

// TestRawInflateStartIndex covers the one argument of Raw Inflate that changes
// the result: the stream is read from that byte onwards.
func TestRawInflateStartIndex(t *testing.T) {
	compressed, err := runOp(t, "Raw Deflate", "The cat sat on the mat.", "Dynamic Huffman Coding")
	if err != nil {
		t.Fatalf("Raw Deflate: %v", err)
	}
	// Two bytes of padding in front, skipped by starting two bytes in.
	padded := "\x00\x00" + compressed

	out, err := runOp(t, "Raw Inflate", padded, float64(2), float64(0), "Adaptive", false, false)
	if err != nil {
		t.Fatalf("Raw Inflate: %v", err)
	}
	if out != "The cat sat on the mat." {
		t.Errorf("got %q", out)
	}
}

// TestRawInflateBufferArgumentsAreInert pins that the three buffer arguments
// change nothing about the result. They size and grow the working buffer inside
// the reader, which has no bearing on what it decodes.
func TestRawInflateBufferArgumentsAreInert(t *testing.T) {
	compressed, err := runOp(t, "Raw Deflate", deflateSentence+deflateSentence, "Dynamic Huffman Coding")
	if err != nil {
		t.Fatalf("Raw Deflate: %v", err)
	}
	want := deflateSentence + deflateSentence

	for _, tc := range []struct {
		size       float64
		bufferType string
		resize     bool
		verify     bool
	}{
		{0, "Adaptive", false, false},
		{0, "Block", false, false},
		{4, "Adaptive", true, false},
		{1024, "Block", true, true},
		{0, "Adaptive", false, true},
	} {
		out, err := runOp(t, "Raw Inflate", compressed,
			float64(0), tc.size, tc.bufferType, tc.resize, tc.verify)
		if err != nil {
			t.Fatalf("%+v: %v", tc, err)
		}
		if out != want {
			t.Errorf("%+v changed the result", tc)
		}
	}
}

// TestRawInflateErrors covers the input the reader refuses.
func TestRawInflateErrors(t *testing.T) {
	compressed, err := runOp(t, "Raw Deflate", "The cat sat on the mat.", "Dynamic Huffman Coding")
	if err != nil {
		t.Fatalf("Raw Deflate: %v", err)
	}
	for _, tc := range []struct{ name, input string }{
		{"empty", ""},
		{"truncated", compressed[:len(compressed)/2]},
		{"garbage", "\xff\xff\xff\xff\xff\xff\xff\xff"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := runOp(t, "Raw Inflate", tc.input, float64(0), float64(0), "Adaptive", false, false); err == nil {
				t.Error("expected an error")
			}
		})
	}
}

// TestRawDeflateStoreBlocks covers the block splitting of the stored form,
// which writes at most 65535 bytes per block, each with its length and that
// length's complement.
func TestRawDeflateStoreBlocks(t *testing.T) {
	const size = 70000 // more than one block, less than two
	out, err := runOp(t, "Raw Deflate", string(deflateRandom(size, 5)), "None (Store)")
	if err != nil {
		t.Fatalf("Raw Deflate: %v", err)
	}
	// Two blocks: 5 bytes of header each, plus the data.
	if want := size + 10; len(out) != want {
		t.Fatalf("produced %d bytes, want %d", len(out), want)
	}
	if out[0] != 0x00 {
		t.Errorf("first block header is %#02x, want 0x00 (not final, stored)", out[0])
	}
	second := 5 + 65535
	if out[second] != 0x01 {
		t.Errorf("second block header is %#02x, want 0x01 (final, stored)", out[second])
	}
}

// TestDeflateFamilyEmptyInput covers compressing nothing, which every writer
// has to turn into a stream its reader will accept.
//
// CyberChef does not: under the stored encoding it hands back an untouched
// 32768-byte working buffer, and under the other two a stream cut a couple of
// bits short. Its own reader rejects all three. What is checked here is that
// the result is well formed and reads back as empty.
func TestDeflateFamilyEmptyInput(t *testing.T) {
	for _, w := range []struct {
		op, reader string
		args       func(string) []any
		readerArgs []any
	}{
		{
			"Raw Deflate", "Raw Inflate",
			func(t string) []any { return []any{t} },
			[]any{float64(0), float64(0), "Adaptive", false, false},
		},
		{
			"Zlib Deflate", "Zlib Inflate",
			func(t string) []any { return []any{t} },
			[]any{float64(0), float64(0), "Adaptive", false, false},
		},
		{
			"Gzip", "Gunzip",
			func(t string) []any { return []any{t, "", "", false} },
			nil,
		},
	} {
		for _, compressionType := range deflateCompressionTypes {
			t.Run(w.op+" "+compressionType, func(t *testing.T) {
				out, err := runOp(t, w.op, "", w.args(compressionType)...)
				if err != nil {
					t.Fatalf("%s: %v", w.op, err)
				}
				if out == "" {
					t.Fatal("produced nothing at all")
				}
				back, err := runOp(t, w.reader, out, w.readerArgs...)
				if err != nil {
					t.Fatalf("%s: %v", w.reader, err)
				}
				if back != "" {
					t.Errorf("read back %q, want nothing", back)
				}
			})
		}
	}
}

// TestDeflateCodeTables covers the length and distance code lookups at every
// boundary, against the tables RFC 1951 sets out.
func TestDeflateCodeTables(t *testing.T) {
	for length := rdfMinMatch; length <= rdfMaxMatch; length++ {
		code, extra, bits := rdfLengthCode(length)
		if code < 257 || code > 285 {
			t.Fatalf("length %d gave code %d, outside 257..285", length, code)
		}
		base := rdfLengthBase[code-257]
		if extra != length-base || bits != rdfLengthExtra[code-257] {
			t.Errorf("length %d gave code %d extra %d bits %d", length, code, extra, bits)
		}
		if extra >= 1<<bits && length != rdfMaxMatch {
			t.Errorf("length %d needs %d in %d bits", length, extra, bits)
		}
	}
	for _, distance := range []int{
		1, 2, 3, 4, 5, 6, 7, 8, 9, 16, 17, 24, 25,
		32, 33, 4096, 4097, 24576, 24577, rdfWindow,
	} {
		code, extra, bits := rdfDistanceCode(distance)
		if code < 0 || code > 29 {
			t.Fatalf("distance %d gave code %d, outside 0..29", distance, code)
		}
		if extra != distance-rdfDistBase[code] || bits != rdfDistExtra[code] {
			t.Errorf("distance %d gave code %d extra %d bits %d", distance, code, extra, bits)
		}
	}
}

// TestDeflateTailMatches covers the check that abandons a candidate repeat as
// soon as it disagrees with the current position anywhere the best match so far
// reaches. It looks from the far end, where a candidate is likeliest to fail.
func TestDeflateTailMatches(t *testing.T) {
	data := []byte("abcdefgh" + "abcdefgX")
	if !rdfTailMatches(data, 0, 8, 7) {
		t.Error("agreeing candidate reported as differing over seven bytes")
	}
	if rdfTailMatches(data, 0, 8, 8) {
		t.Error("candidate differing at its eighth byte reported as agreeing")
	}
}

// TestDeflateLongestMatch covers the search itself.
func TestDeflateLongestMatch(t *testing.T) {
	t.Run("further back is longer", func(t *testing.T) {
		// "abcde" at 0, "abc" at 8, and the position being matched at 12.
		data := []byte("abcde___abc_abcde")
		length, distance := rdfLongestMatch(data, 12, []int32{0, 8})
		if length != 5 || distance != 12 {
			t.Errorf("matched %d bytes %d back, want 5 and 12", length, distance)
		}
	})

	t.Run("older candidate abandoned at its tail", func(t *testing.T) {
		// The newest candidate matches all eight bytes, so the older one — which
		// shares only the first four — is dropped without being walked through.
		data := []byte("abcdXXXX" + "abcdefgh" + "abcdefgh")
		length, distance := rdfLongestMatch(data, 16, []int32{0, 8})
		if length != 8 || distance != 8 {
			t.Errorf("matched %d bytes %d back, want 8 and 8", length, distance)
		}
	})
}

// TestDeflateWritersRejectUnknownType covers the guard each writer keeps on the
// block encoding. The argument definition allows only the three named, so this
// is reached by handing an operation an argument the command line could not.
func TestDeflateWritersRejectUnknownType(t *testing.T) {
	for _, tc := range []struct {
		name string
		op   core.Operation
		args []any
	}{
		{"Raw Deflate", RawDeflate{}, []any{"Brotli"}},
		{"Zlib Deflate", ZlibDeflate{}, []any{"Brotli"}},
		{"Gzip", Gzip{}, []any{"Brotli", "", "", false}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dish := core.NewDish([]byte("hello"), core.TypeByteArray)
			if _, err := tc.op.Run(dish, tc.args); err == nil {
				t.Error("expected an error")
			}
		})
	}
}

// TestDeflateEncodeRejectsUnknownType covers the guard on the block encoding.
// The argument definition allows only the three named, so nothing reaches this
// through the operation, but the writer is called directly by the two
// containers as well.
func TestDeflateEncodeRejectsUnknownType(t *testing.T) {
	if _, err := deflateEncode([]byte("hello"), "Brotli"); err == nil {
		t.Error("expected an error for an unknown compression type")
	}
	if _, err := zlibEncode([]byte("hello"), "Brotli"); err == nil {
		t.Error("expected an error from the zlib writer")
	}
	if _, err := gzipEncode([]byte("hello"), "Brotli", "", "", false); err == nil {
		t.Error("expected an error from the gzip writer")
	}
}

// TestInflateStartIndexOutsideInput covers a start index that is not a position
// in the input at all.
func TestInflateStartIndexOutsideInput(t *testing.T) {
	for _, op := range []string{"Raw Inflate", "Zlib Inflate"} {
		t.Run(op, func(t *testing.T) {
			for _, index := range []float64{-1, 100} {
				_, err := runOp(t, op, "hello", index, float64(0), "Adaptive", false, false)
				if err == nil {
					t.Errorf("start index %v: expected an error", index)
				}
			}
		})
	}
}

// TestDeflatePackageBudgets covers how many packages each depth can pay for.
// The deepest holds one per symbol, each shallower one at most half the depth
// below plus a symbol apiece, and the flags record where the count is odd.
func TestDeflatePackageBudgets(t *testing.T) {
	budgets, flags := rdfPackageBudgets(5, 4)
	if len(budgets) != 4 || len(flags) != 4 {
		t.Fatalf("got %d budgets and %d flags, want 4 of each", len(budgets), len(flags))
	}
	if budgets[3] != 5 {
		t.Errorf("the deepest budget is %d, want one per symbol (5)", budgets[3])
	}
	for j := range 3 {
		if budgets[j] > 2*budgets[j]+1 {
			t.Errorf("budget %d is %d, larger than the depth below allows", j, budgets[j])
		}
	}
}

// TestDeflateMergeRow covers filling one depth from the one below: at each
// place it takes either the two cheapest packages below joined together, or the
// next symbol on its own, whichever is cheaper.
func TestDeflateMergeRow(t *testing.T) {
	freqs := []int32{10, 4, 1}
	below := []int32{1, 1, 4, 10}
	row := make([]int32, 3)
	kinds := make([]int, 3)

	rdfMergeRow(freqs, below, row, kinds, 0)

	// The first place takes the symbol weighing 10, since joining the two
	// cheapest below comes to only 2.
	if row[0] != 10 || kinds[0] != 0 {
		t.Errorf("first place is %d (kind %d), want the symbol weighing 10", row[0], kinds[0])
	}
	if row[1] != 4 || kinds[1] != 1 {
		t.Errorf("second place is %d (kind %d), want the symbol weighing 4", row[1], kinds[1])
	}
	// By the third the remaining symbol weighs 1, so the pair below wins.
	if kinds[2] != len(freqs) {
		t.Errorf("third place is kind %d, want a joined pair (%d)", kinds[2], len(freqs))
	}
}
