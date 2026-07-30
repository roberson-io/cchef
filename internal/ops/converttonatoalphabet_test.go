package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// TestConvertToNATOAlphabetFixtures runs CyberChef's two fixture cases.
func TestConvertToNATOAlphabetFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"Convert to NATO alphabet: nothing", "", "",
			core.Recipe{{Op: "Convert to NATO alphabet", Args: []any{}}},
		},
		{
			"Convert to NATO alphabet: full alphabet with numbers",
			"abcdefghijklmnopqrstuvwxyz0123456789,/.",
			"Alfa Bravo Charlie Delta Echo Foxtrot Golf Hotel India Juliett Kilo Lima Mike " +
				"November Oscar Papa Quebec Romeo Sierra Tango Uniform Victor Whiskey X-ray " +
				"Yankee Zulu Zero One Two Three Four Five Six Seven Eight Nine Comma " +
				"Fraction bar Full stop ",
			core.Recipe{{Op: "Convert to NATO alphabet", Args: []any{}}},
		},
	})
}

// TestConvertToNATOAlphabetEdges covers behaviour recorded from CyberChef's
// Node API: uppercase reads the same as lowercase, characters outside the
// table pass through (so a space after a mapped character doubles up), and
// each mapped character carries its own trailing space.
func TestConvertToNATOAlphabetEdges(t *testing.T) {
	cases := []struct {
		name, input, want string
	}{
		{
			"mixed with unmapped characters", "ab, c/d.",
			"Alfa Bravo Comma  Charlie Fraction bar Delta Full stop ",
		},
		{"uppercase", "AbC", "Alfa Bravo Charlie "},
		{"only unmapped characters", "!? -", "!? -"},
		{"beyond ascii untouched", "é日", "é日"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := runOp(t, "Convert to NATO alphabet", c.input)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}
