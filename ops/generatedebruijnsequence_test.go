package ops

import (
	"bufio"
	"encoding/json"
	"math"
	"os"
	"strconv"
	"testing"
)

// deBruijnVector is one recorded sequence: the size of the alphabet, the length
// of the key, and the sequence itself. The two CyberChef ships as fixtures are
// among them; the rest come from the oracle.
type deBruijnVector struct {
	K    int    `json:"k"`
	N    int    `json:"n"`
	Want string `json:"want"`
}

// TestDeBruijnVectors covers the sequences themselves.
func TestDeBruijnVectors(t *testing.T) {
	file, err := os.Open("testdata/de_bruijn.jsonl")
	if err != nil {
		t.Fatalf("open vectors: %v", err)
	}
	defer func() { _ = file.Close() }()

	seen := 0
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		if len(scanner.Bytes()) == 0 {
			continue
		}
		var v deBruijnVector
		if err := json.Unmarshal(scanner.Bytes(), &v); err != nil {
			t.Fatalf("parse vector: %v", err)
		}
		seen++

		t.Run("k="+strconv.Itoa(v.K)+"/n="+strconv.Itoa(v.N), func(t *testing.T) {
			out, err := runOp(t, "Generate De Bruijn Sequence", "", float64(v.K), float64(v.N))
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if out != v.Want {
				t.Errorf("got %d characters, want %d", len(out), len(v.Want))
			}
		})
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("read vectors: %v", err)
	}
	if seen == 0 {
		t.Fatal("no vectors were read")
	}
}

// TestDeBruijnIsACyclicCover covers what makes a sequence De Bruijn: read as a
// loop, every key of that length over that alphabet appears exactly once.
func TestDeBruijnIsACyclicCover(t *testing.T) {
	for _, tc := range []struct{ k, n int }{
		{2, 3}, {2, 5}, {3, 3}, {4, 4}, {5, 3}, {9, 2},
	} {
		t.Run("k="+strconv.Itoa(tc.k)+"/n="+strconv.Itoa(tc.n), func(t *testing.T) {
			out, err := runOp(t, "Generate De Bruijn Sequence", "", float64(tc.k), float64(tc.n))
			if err != nil {
				t.Fatalf("Run: %v", err)
			}

			want := int(math.Pow(float64(tc.k), float64(tc.n)))
			if len(out) != want {
				t.Fatalf("the sequence is %d long, want %d", len(out), want)
			}
			// Reading past the end wraps to the beginning, which is what makes
			// the sequence a loop.
			looped := out + out[:tc.n-1]
			keys := map[string]bool{}
			for i := range want {
				keys[looped[i:i+tc.n]] = true
			}
			if len(keys) != want {
				t.Errorf("the sequence holds %d distinct keys, want %d", len(keys), want)
			}
			for _, c := range out {
				if c < '0' || int(c-'0') >= tc.k {
					t.Fatalf("%q is not a digit of an alphabet of %d", c, tc.k)
				}
			}
		})
	}
}

// TestDeBruijnRejects covers the sizes it will not build a sequence for.
func TestDeBruijnRejects(t *testing.T) {
	for _, tc := range []struct {
		name string
		k, n float64
		want string
	}{
		{"an alphabet of one", 1, 3, "Invalid alphabet size, required to be between 2 and 9 (inclusive)."},
		{"an alphabet of none", 0, 3, "Invalid alphabet size, required to be between 2 and 9 (inclusive)."},
		{"an alphabet of ten", 10, 2, "Invalid alphabet size, required to be between 2 and 9 (inclusive)."},
		{"an alphabet that is not whole", 2.5, 3, "Invalid alphabet size, required to be integer."},
		{"a key length that is not whole", 3, 2.5, "Invalid key length, required to be integer."},
		{"a key length of one", 3, 1, "Invalid key length, required to be at least 2."},
		{"a key length of none", 2, 0, "Invalid key length, required to be at least 2."},
		{"a key length below none", 3, -1, "Invalid key length, required to be at least 2."},
		{"more keys than it will make", 2, 16, "Too many permutations, please reduce k^n to under 50,000."},
		{"far more", 9, 6, "Too many permutations, please reduce k^n to under 50,000."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := runOp(t, "Generate De Bruijn Sequence", "", tc.k, tc.n)
			if err == nil {
				t.Fatalf("built a sequence of %d characters anyway", len(out))
			}
			if err.Error() != tc.want {
				t.Errorf("got  %q\nwant %q", err.Error(), tc.want)
			}
		})
	}
}

// TestDeBruijnPermutationLimit covers the two sides of the limit on how many
// keys it will cover.
func TestDeBruijnPermutationLimit(t *testing.T) {
	// 2^15 is 32768, which is under the limit; 2^16 is 65536, which is over.
	if _, err := runOp(t, "Generate De Bruijn Sequence", "", 2.0, 15.0); err != nil {
		t.Errorf("a sequence within the limit was refused: %v", err)
	}
	if _, err := runOp(t, "Generate De Bruijn Sequence", "", 2.0, 16.0); err == nil {
		t.Error("a sequence past the limit was built")
	}
}

// TestDeBruijnIgnoresItsInput covers the input, which plays no part.
func TestDeBruijnIgnoresItsInput(t *testing.T) {
	first, err := runOp(t, "Generate De Bruijn Sequence", "", 2.0, 3.0)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	second, err := runOp(t, "Generate De Bruijn Sequence", "anything at all", 2.0, 3.0)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if first != second {
		t.Errorf("the input changed the answer: %q against %q", first, second)
	}
}
