package ops

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// runOp runs a single registered operation against input, returning output/error.
func runOp(t *testing.T, name, input string, args ...any) (string, error) {
	t.Helper()
	op, ok := core.Default.Get(name)
	if !ok {
		t.Fatalf("op %q not registered", name)
	}
	coerced, err := core.CoerceArgs(op.Args(), args)
	if err != nil {
		t.Fatalf("coerce args: %v", err)
	}
	out, err := op.Run(core.NewDish([]byte(input), op.Meta().InputType), coerced)
	if err != nil {
		return "", err
	}
	return out.String(), nil
}

// mustHex decodes a hex literal for tests that need exact bytes, which is most
// of the binary-format ones.
func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("bad hex %q: %v", s, err)
	}
	return b
}

// carvePrefix and carveSuffix pad a sample so a carver has to find where the
// file starts and where it ends rather than reading a whole buffer.
var (
	carvePrefix = append(bytes.Repeat([]byte{0x00}, 7), []byte("PREFIX--")...)
	carveSuffix = append([]byte("--SUFFIX"), bytes.Repeat([]byte{0xee}, 9)...)
)

// readCarveSample reads one of the sample files the carving tests are built on.
// They live with the carvers, which is what defines what a valid sample is.
func readCarveSample(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "internal", "filecarve", "testdata", "carve", name))
	if err != nil {
		t.Fatalf("reading the sample: %v", err)
	}
	return data
}

// derTLV builds a DER TLV (tag + length + value) from a tag byte and value, both
// hex, using short-form or long-form length encoding as needed.
func derTLV(tag, value string) string {
	n := len(value) / 2
	if n < 128 {
		return tag + fmt.Sprintf("%02x", n) + value
	}
	lenHex := fmt.Sprintf("%x", n)
	if len(lenHex)%2 != 0 {
		lenHex = "0" + lenHex
	}
	return tag + fmt.Sprintf("%02x", 0x80|len(lenHex)/2) + lenHex + value
}

// opCase mirrors a CyberChef fixture case
// (../CyberChef/tests/operations/tests/<Op>.mjs): an input, the expected output,
// and the recipe that produces it. recipeConfig maps directly onto core.Recipe.
type opCase struct {
	name   string
	input  string
	want   string
	recipe core.Recipe
}

// runCases executes each transcribed fixture case through the engine and
// compares the result against CyberChef's expected output.
func runCases(t *testing.T, cases []opCase) {
	t.Helper()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := c.recipe.Execute(core.NewDish([]byte(c.input), core.TypeString))
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if out.String() != c.want {
				t.Fatalf("got %q\nwant %q", out.String(), c.want)
			}
		})
	}
}

// allBytes is the ALL_BYTES fixture from CyberChef's Base64 tests: every byte
// value 0x00..0xff in order.
func allBytes() string {
	b := make([]byte, 256)
	for i := range b {
		b[i] = byte(i)
	}
	return string(b)
}

// abytes/sdish wrap a raw string as an ArrayBuffer / string dish for direct Run
// calls in tests (used to reach branches guarded on the normal path by CoerceArgs).
func abytes(s string) *core.Dish { return core.NewDish([]byte(s), core.TypeArrayBuffer) }
func sdish(s string) *core.Dish  { return core.NewDish([]byte(s), core.TypeString) }
