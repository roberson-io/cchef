package ops

import (
	"fmt"
	"testing"

	"github.com/roberson-io/cchef/core"
)

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
