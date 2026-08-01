package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// TestIndexOfCoincidenceFixture covers CyberChef's own case. Its test asserts
// the presented text rather than the number, so the two lines below are what a
// faithful port has to produce.
func TestIndexOfCoincidenceFixture(t *testing.T) {
	runCases(t, []opCase{
		{
			"Index of Coincidence",
			"Hello world, this is a test to determine the correct IC value.",
			"Index of Coincidence: 0.07142857142857142\nNormalized: 1.857142857142857",
			core.Recipe{{Op: "Index of Coincidence", Args: []any{}}},
		},
	})
}

// TestIndexOfCoincidenceValues covers the coincidence itself against the
// oracle, over inputs that exercise each part of the formula.
func TestIndexOfCoincidenceValues(t *testing.T) {
	for _, tc := range []struct{ name, input, want string }{
		{"a single letter, which cannot coincide with itself", "a", "0"},
		{"the same letter four times", "aaaa", "1"},
		{"a short phrase", "Hello world!", "0.08888888888888889"},
		{
			"a longer one", "Hello world, this is a test to determine the correct IC value.",
			"0.07142857142857142",
		},
		{
			"a pangram, whose letters are spread evenly",
			"The quick brown fox jumps over the lazy dog", "0.021848739495798318",
		},
		{"digits, which hold no letters at all", "0123456789", "0"},
		{"two letters in equal measure", "AAAAAAAAAABBBBBBBBBB", "0.47368421052631576"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := core.Recipe{{Op: "Index of Coincidence", Args: []any{}}}.
				Execute(core.NewDish([]byte(tc.input), core.TypeString))
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			got, _, _ := strings.Cut(strings.TrimPrefix(out.String(), "Index of Coincidence: "), "\n")
			if got != tc.want {
				t.Errorf("coincidence %s, want %s", got, tc.want)
			}
		})
	}
}
