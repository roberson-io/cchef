package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

func fletcherRecipe(op string) core.Recipe { return core.Recipe{{Op: op, Args: []any{}}} }

// Fletcher-16/32/64 vectors transcribed from
// ../CyberChef/tests/operations/tests/FletcherChecksum.mjs. Fletcher-8 has no
// upstream fixtures, so those values come from the CyberChef-server oracle.
func TestFletcherFixtures(t *testing.T) {
	runCases(t, []opCase{
		{"Fletcher-8: abcde", "abcde", "50", fletcherRecipe("Fletcher-8 Checksum")},
		{"Fletcher-8: abcdef", "abcdef", "2c", fletcherRecipe("Fletcher-8 Checksum")},
		{"Fletcher-8: abcdefgh", "abcdefgh", "69", fletcherRecipe("Fletcher-8 Checksum")},

		{"Fletcher-16: abcde", "abcde", "c8f0", fletcherRecipe("Fletcher-16 Checksum")},
		{"Fletcher-16: abcdef", "abcdef", "2057", fletcherRecipe("Fletcher-16 Checksum")},
		{"Fletcher-16: abcdefgh", "abcdefgh", "0627", fletcherRecipe("Fletcher-16 Checksum")},

		{"Fletcher-32: abcde", "abcde", "f04fc729", fletcherRecipe("Fletcher-32 Checksum")},
		{"Fletcher-32: abcdef", "abcdef", "56502d2a", fletcherRecipe("Fletcher-32 Checksum")},
		{"Fletcher-32: abcdefgh", "abcdefgh", "ebe19591", fletcherRecipe("Fletcher-32 Checksum")},

		{"Fletcher-64: abcde", "abcde", "c8c6c527646362c6", fletcherRecipe("Fletcher-64 Checksum")},
		{"Fletcher-64: abcdef", "abcdef", "c8c72b276463c8c6", fletcherRecipe("Fletcher-64 Checksum")},
		{"Fletcher-64: abcdefgh", "abcdefgh", "312e2b28cccac8c6", fletcherRecipe("Fletcher-64 Checksum")},
	})
}

// Empty input yields all-zero checksums (the accumulator loop never runs).
func TestFletcherEmpty(t *testing.T) {
	cases := map[string]string{
		"Fletcher-8 Checksum":  "00",
		"Fletcher-16 Checksum": "0000",
		"Fletcher-32 Checksum": "00000000",
		"Fletcher-64 Checksum": "0000000000000000",
	}
	for op, want := range cases {
		got, err := runOp(t, op, "")
		if err != nil || got != want {
			t.Fatalf("%s(\"\") = %q, %v; want %q", op, got, err, want)
		}
	}
}
