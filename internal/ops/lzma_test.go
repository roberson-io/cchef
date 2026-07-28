package ops

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/ulikunitz/xz/lzma"

	"github.com/roberson-io/cchef/internal/core"
)

// lzmaGolden is one case from testdata/lzma.jsonl: a stream CyberChef produced,
// and what it should read back as.
type lzmaGolden struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	Spec struct {
		Hex  string `json:"hex"`
		Len  int    `json:"len"`
		N    int    `json:"n"`
		Seed int64  `json:"seed"`
	} `json:"spec"`
	Mode        string `json:"mode"`
	StreamHex   string `json:"streamHex"`
	PlainLen    int    `json:"plainLen"`
	PlainSHA256 string `json:"plainSHA256"`
}

// lzmaSentence is the text the repeated-text cases are built from.
const lzmaSentence = "The quick brown fox jumps over the lazy dog. "

// plain rebuilds the bytes a golden was made from.
func (g lzmaGolden) plain(t *testing.T) []byte {
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
	case "text":
		out := make([]byte, 0, len(lzmaSentence)*g.Spec.N)
		for range g.Spec.N {
			out = append(out, lzmaSentence...)
		}
		return out
	case "random":
		return deflateRandom(g.Spec.Len, g.Spec.Seed)
	}
	t.Fatalf("%s: unknown input kind %q", g.Name, g.Kind)
	return nil
}

// loadLZMAGoldens reads the corpus.
func loadLZMAGoldens(t *testing.T) []lzmaGolden {
	t.Helper()
	f, err := os.Open("testdata/lzma.jsonl")
	if err != nil {
		t.Fatalf("open goldens: %v", err)
	}
	defer func() { _ = f.Close() }()

	var out []lzmaGolden
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<22), 1<<22)
	for sc.Scan() {
		var g lzmaGolden
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

// TestLZMAFixtures covers CyberChef's own cases
// (../CyberChef/tests/operations/tests/Compress.mjs). The two decompression
// cases read streams the lzma command wrote, which state no size in the header.
func TestLZMAFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"LZMA compress & decompress",
			"The cat sat on the mat.", "The cat sat on the mat.",
			core.Recipe{
				{Op: "LZMA Compress", Args: []any{"6"}},
				{Op: "LZMA Decompress", Args: []any{}},
			},
		},
		{
			"LZMA decompress: binary",
			"5d00008000ffffffffffffffff00000052500a84f99bb28021a969d627e03e8a922effffbd160000",
			"00 01 02 03 04 05 06 07 08 09 0a 0b 0c 0d 0e 0f 10",
			core.Recipe{
				{Op: "From Hex", Args: []any{"Space"}},
				{Op: "LZMA Decompress", Args: []any{}},
				{Op: "To Hex", Args: []any{"Space"}},
			},
		},
		{
			"LZMA decompress: string",
			"5d00008000ffffffffffffffff002a1a08a202b1a4b814b912c94c4152e1641907d3fd8cd903ffff4fec0000",
			"The cat sat on the mat.",
			core.Recipe{
				{Op: "From Hex", Args: []any{"Space"}},
				{Op: "LZMA Decompress", Args: []any{}},
			},
		},
	})
}

// TestLZMADecompressGoldens reads streams CyberChef wrote, across four of the
// nine compression modes and a corpus from one byte to sixty kilobytes.
// Decompression has one right answer, so these are exact.
func TestLZMADecompressGoldens(t *testing.T) {
	for _, g := range loadLZMAGoldens(t) {
		t.Run(g.Name, func(t *testing.T) {
			stream, err := hex.DecodeString(g.StreamHex)
			if err != nil {
				t.Fatalf("bad stream hex: %v", err)
			}
			out, err := runOp(t, "LZMA Decompress", string(stream))
			if err != nil {
				t.Fatalf("LZMA Decompress: %v", err)
			}
			if len(out) != g.PlainLen {
				t.Errorf("read back %d bytes, want %d", len(out), g.PlainLen)
			}
			if sum := fmt.Sprintf("%x", sha256.Sum256([]byte(out))); sum != g.PlainSHA256 {
				t.Errorf("digest %s, want %s", sum, g.PlainSHA256)
			}
		})
	}
}

// TestLZMARoundTrips checks each golden's input back through both operations.
func TestLZMARoundTrips(t *testing.T) {
	for _, g := range loadLZMAGoldens(t) {
		t.Run(g.Name, func(t *testing.T) {
			want := string(g.plain(t))
			compressed, err := runOp(t, "LZMA Compress", want, g.Mode)
			if err != nil {
				t.Fatalf("LZMA Compress: %v", err)
			}
			back, err := runOp(t, "LZMA Decompress", compressed)
			if err != nil {
				t.Fatalf("LZMA Decompress: %v", err)
			}
			if back != want {
				t.Errorf("round trip changed the data (%d bytes in, %d out)",
					len(want), len(back))
			}
		})
	}
}

// TestLZMACompressModes covers the compression mode, which sets how far back a
// repeat may reach. The size of that window is recorded in the header, so it
// can be read straight back out.
func TestLZMACompressModes(t *testing.T) {
	// The nine modes, as dictionary sizes in bytes.
	want := []uint32{
		1 << 16, 1 << 20, 1 << 19, 1 << 20, 1 << 21,
		1 << 22, 1 << 23, 1 << 24, 1 << 25,
	}
	for i, dictSize := range want {
		mode := fmt.Sprint(i + 1)
		t.Run("mode "+mode, func(t *testing.T) {
			out, err := runOp(t, "LZMA Compress", "The cat sat on the mat.", mode)
			if err != nil {
				t.Fatalf("LZMA Compress: %v", err)
			}
			if len(out) < lzmaHeaderSize {
				t.Fatalf("output is only %d bytes", len(out))
			}
			got := uint32(out[1]) | uint32(out[2])<<8 | uint32(out[3])<<16 | uint32(out[4])<<24
			if got != dictSize {
				t.Errorf("header records a %d-byte window, want %d", got, dictSize)
			}
			if out[0] != lzmaProperties {
				t.Errorf("properties byte is %#02x, want %#02x", out[0], lzmaProperties)
			}
		})
	}
}

// TestLZMACompressStatesNoSizeInHeader pins the shape of the header this writer
// produces: the size is left unknown and an end marker closes the stream.
//
// That is not only a choice about fidelity. The library underneath has an open
// bug — ulikunitz/xz issue 71 — where asking for the size in the header while
// writing nothing produces a stream its own reader rejects. Nothing here asks
// for it, and this test fails if that ever changes.
func TestLZMACompressStatesNoSizeInHeader(t *testing.T) {
	for _, input := range []string{"", "a", "The cat sat on the mat."} {
		out, err := runOp(t, "LZMA Compress", input, "7")
		if err != nil {
			t.Fatalf("LZMA Compress(%q): %v", input, err)
		}
		if len(out) < lzmaHeaderSize {
			t.Fatalf("output is only %d bytes", len(out))
		}
		size := out[5:lzmaHeaderSize]
		for i := range size {
			if size[i] != 0xff {
				t.Errorf("input %q: header states a size (% x), want it left unknown",
					input, size)
				break
			}
		}

		// And whatever the header says, it must read back.
		back, err := runOp(t, "LZMA Decompress", out)
		if err != nil {
			t.Fatalf("LZMA Decompress(%q): %v", input, err)
		}
		if back != input {
			t.Errorf("round trip of %q gave %q", input, back)
		}
	}
}

// TestLZMADecompressReadsBothHeaderForms covers the two ways the length can be
// recorded: stated outright, as CyberChef does, and left unknown with an end
// marker, as the lzma command does.
func TestLZMADecompressReadsBothHeaderForms(t *testing.T) {
	const want = "The cat sat on the mat."
	for _, tc := range []struct{ name, stream string }{
		{
			// What CyberChef writes: the length stated outright.
			"size stated",
			"5d000080001700000000000000002a1a08a202b1a4b814b912c94c4152e1641907d3fd8cd903ffff4fec0000",
		},
		{
			// What the lzma command writes: the length left unknown.
			"size unknown",
			"5d00008000ffffffffffffffff002a1a08a202b1a4b814b912c94c4152e1641907d3fd8cd903ffff4fec0000",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stream, err := hex.DecodeString(tc.stream)
			if err != nil {
				t.Fatalf("bad hex: %v", err)
			}
			out, err := runOp(t, "LZMA Decompress", string(stream))
			if err != nil {
				t.Fatalf("LZMA Decompress: %v", err)
			}
			if out != want {
				t.Errorf("got %q, want %q", out, want)
			}
		})
	}
}

// TestLZMAErrors covers the input each operation refuses.
func TestLZMAErrors(t *testing.T) {
	good, err := runOp(t, "LZMA Compress", "The cat sat on the mat.", "7")
	if err != nil {
		t.Fatalf("LZMA Compress: %v", err)
	}
	for _, tc := range []struct{ name, input string }{
		{"empty", ""},
		{"not lzma", "this is not an lzma stream"},
		{"truncated", good[:len(good)/2]},
		{"header only", good[:lzmaHeaderSize]},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := runOp(t, "LZMA Decompress", tc.input); err == nil {
				t.Error("expected an error")
			}
		})
	}
}

// TestLZMACompressEmptyInput covers compressing nothing, which has to give a
// stream that reads back as nothing.
func TestLZMACompressEmptyInput(t *testing.T) {
	out, err := runOp(t, "LZMA Compress", "", "7")
	if err != nil {
		t.Fatalf("LZMA Compress: %v", err)
	}
	if len(out) <= lzmaHeaderSize {
		t.Fatalf("produced %d bytes, too few to hold a stream", len(out))
	}
	back, err := runOp(t, "LZMA Decompress", out)
	if err != nil {
		t.Fatalf("LZMA Decompress: %v", err)
	}
	if back != "" {
		t.Errorf("read back %q, want nothing", back)
	}
}

// TestLZMAWindowRejectsUnknownMode covers the guard on the compression mode.
// The argument definition allows only 1 to 9, so this is reached by handing the
// operation an argument the command line could not.
func TestLZMAWindowRejectsUnknownMode(t *testing.T) {
	for _, mode := range []string{"", "0", "10", "seven", "-1"} {
		t.Run(mode, func(t *testing.T) {
			if _, err := lzmaWindow(mode); err == nil {
				t.Error("expected an error from lzmaWindow")
			}
			dish := core.NewDish([]byte("hello"), core.TypeByteArray)
			if _, err := (LZMACompress{}).Run(dish, []any{mode}); err == nil {
				t.Error("expected an error from the operation")
			}
		})
	}
}

// TestLZMAWindowAcceptsEveryMode covers the other side of that guard.
func TestLZMAWindowAcceptsEveryMode(t *testing.T) {
	for i, want := range lzmaWindowBits {
		mode := fmt.Sprint(i + 1)
		got, err := lzmaWindow(mode)
		if err != nil {
			t.Errorf("mode %s: %v", mode, err)
			continue
		}
		if got != want {
			t.Errorf("mode %s gives %d window bits, want %d", mode, got, want)
		}
	}
}

// failingWriter refuses everything written to it.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("no room") }

// TestLZMAEncodeToErrors covers the writer's own failures: settings it will not
// accept, and a destination that will not take the bytes.
func TestLZMAEncodeToErrors(t *testing.T) {
	t.Run("settings refused", func(t *testing.T) {
		// A window far below the smallest the format allows.
		err := lzmaEncodeTo(&bytes.Buffer{}, []byte("hello"), lzma.WriterConfig{DictCap: 1})
		if err == nil {
			t.Error("expected an error for an impossible window size")
		}
	})
	t.Run("destination refuses", func(t *testing.T) {
		// Enough data that the encoder has to flush while writing.
		data := deflateRandom(1<<20, 41)
		err := lzmaEncodeTo(failingWriter{}, data, lzma.WriterConfig{DictCap: 1 << 16})
		if err == nil {
			t.Error("expected an error from a destination that refuses")
		}
	})
}
