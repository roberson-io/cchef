package ops

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// failReader simulates a crypto/rand entropy-source failure.
type failReader struct{}

func (failReader) Read([]byte) (int, error) { return 0, fmt.Errorf("no entropy") }

// TestShuffleRandFailurePanics injects a failing crypto/rand.Reader (a
// reassignable package var) so rand.Int returns an error, which Shuffle's randInt
// treats as unrecoverable and panics on.
//
// The PRNG op's rand.Read path (misc.go) is deliberately not covered here: since
// Go 1.24 crypto/rand.Read never returns an error — an entropy failure triggers a
// fatal (uncatchable) error rather than a returned one — so its err != nil branch
// is dead by the standard library's contract and cannot be exercised in a test.
func TestShuffleRandFailurePanics(t *testing.T) {
	orig := rand.Reader
	rand.Reader = failReader{}
	defer func() { rand.Reader = orig }()

	defer func() {
		if recover() == nil {
			t.Error("Shuffle: expected a panic when the entropy source fails")
		}
	}()
	_, _ = runOp(t, "Shuffle", "a,b,c", "Comma")
}

func TestParseObjectIDTimestamp(t *testing.T) {
	runCases(t, []opCase{
		{
			"epoch zero", "000000000000000000000000", "1970-01-01T00:00:00.000Z",
			core.Recipe{{Op: "Parse ObjectID timestamp"}},
		},
		{
			"real objectid", "507f1f77bcf86cd799439011", "2012-10-17T21:13:27.000Z",
			core.Recipe{{Op: "Parse ObjectID timestamp"}},
		},
	})
}

func TestParseObjectIDTimestampErrors(t *testing.T) {
	// Wrong length.
	if _, err := runOp(t, "Parse ObjectID timestamp", "abc"); err == nil {
		t.Error("expected error for non-24-character input")
	}
	// 24 chars but not hex.
	if _, err := runOp(t, "Parse ObjectID timestamp", "zzzzzzzz0000000000000000"); err == nil {
		t.Error("expected error for non-hex input")
	}
}

func TestFileTree(t *testing.T) {
	want := "home\n|---a.txt\n|---b\n|   |---c.txt\n|   |---d.txt"
	runCases(t, []opCase{
		{
			"basic tree", "/home/a.txt\n/home/b/c.txt\n/home/b/d.txt", want,
			core.Recipe{{Op: "File Tree", Args: []any{"/", "Line feed"}}},
		},
	})
}

func TestSleepPassesThrough(t *testing.T) {
	runCases(t, []opCase{
		{
			"sleep passthrough", "hello", "hello",
			core.Recipe{{Op: "Sleep", Args: []any{1}}},
		},
	})
}

// Shuffle and PRNG are non-deterministic; test their structural properties.
func TestShuffle(t *testing.T) {
	out, err := runOp(t, "Shuffle", "a,b,c,d,e", "Comma")
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Split(out, ",")
	sort.Strings(got)
	if strings.Join(got, ",") != "a,b,c,d,e" {
		t.Fatalf("shuffle is not a permutation: %q", out)
	}
	// Empty and single inputs are unchanged.
	if o, _ := runOp(t, "Shuffle", "", "Comma"); o != "" {
		t.Fatalf("empty shuffle = %q", o)
	}
	if o, _ := runOp(t, "Shuffle", "solo", "Comma"); o != "solo" {
		t.Fatalf("single shuffle = %q", o)
	}
}

func TestPRNG(t *testing.T) {
	// Hex: 2 hex chars per byte.
	if o, _ := runOp(t, "Pseudo-Random Number Generator", "", 16, "Hex"); len(o) != 32 || !regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(o) {
		t.Fatalf("hex PRNG = %q", o)
	}
	// Byte array: JSON array of 8 integers in 0..255.
	o, _ := runOp(t, "Pseudo-Random Number Generator", "", 8, "Byte array")
	var arr []int
	if err := json.Unmarshal([]byte(o), &arr); err != nil || len(arr) != 8 {
		t.Fatalf("byte-array PRNG = %q (%v)", o, err)
	}
	for _, v := range arr {
		if v < 0 || v > 255 {
			t.Fatalf("byte out of range: %d", v)
		}
	}
	// Integer: a non-negative decimal string.
	o, _ = runOp(t, "Pseudo-Random Number Generator", "", 4, "Integer")
	if _, err := strconv.ParseUint(o, 10, 64); err != nil {
		t.Fatalf("integer PRNG = %q (%v)", o, err)
	}
	// Raw: exactly N bytes.
	o, _ = runOp(t, "Pseudo-Random Number Generator", "", 10, "Raw")
	if len([]byte(o)) != 10 {
		t.Fatalf("raw PRNG length = %d", len(o))
	}
}

// TestPRNGByteCountBound pins the minimum CyberChef declares on the byte count.
func TestPRNGByteCountBound(t *testing.T) {
	op, _ := core.Default.Get("Pseudo-Random Number Generator")
	args := core.DefaultArgs(op.Args())
	args[0] = float64(0)
	_, err := core.CoerceArgs(op.Args(), args)
	if err == nil {
		t.Fatal("a zero byte count was accepted")
	}
	if want := "Number of bytes must be greater than or equal to 1."; err.Error() != want {
		t.Errorf("got %q, want %q", err.Error(), want)
	}
}
