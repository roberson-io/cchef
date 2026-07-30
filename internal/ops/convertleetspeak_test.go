package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// TestConvertLeetSpeakFixtures runs CyberChef's four fixture cases.
func TestConvertLeetSpeakFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"Convert to Leet Speak: basic text", "leet", "l337",
			core.Recipe{{Op: "Convert Leet Speak", Args: []any{"To Leet Speak"}}},
		},
		{
			"Convert from Leet Speak: basic leet", "l337", "leet",
			core.Recipe{{Op: "Convert Leet Speak", Args: []any{"From Leet Speak"}}},
		},
		{
			"Convert to Leet Speak: basic text, keep case", "HELLO", "H3LL0",
			core.Recipe{{Op: "Convert Leet Speak", Args: []any{"To Leet Speak"}}},
		},
		{
			"Convert from Leet Speak: basic leet, keep case", "H3LL0", "HeLLo",
			core.Recipe{{Op: "Convert Leet Speak", Args: []any{"From Leet Speak"}}},
		},
	})
}

// TestConvertLeetSpeakEdges covers the behaviour recorded from CyberChef's
// Node API: digits already present are kept, a letter that leet turns into a
// digit loses its case coming back (4 is always a), letters with no digit form
// pass through in either direction, and anything outside ASCII letters and the
// leet digits — punctuation, accents — is untouched.
func TestConvertLeetSpeakEdges(t *testing.T) {
	cases := []struct {
		name, input, direction, want string
	}{
		{"mixed case to leet", "Mixed Case Text!", "To Leet Speak", "M1x3d C453 73x7!"},
		{"digits kept to leet", "abc123", "To Leet Speak", "4bc123"},
		{"upper letters from leet", "B4D C0D3", "From Leet Speak", "BaD CoDe"},
		{"unmapped digits from leet", "62 8 9", "From Leet Speak", "62 8 9"},
		{"plain word from leet", "good", "From Leet Speak", "good"},
		{"punctuation to leet", "a-e_i.o s,t", "To Leet Speak", "4-3_1.0 5,7"},
		{"punctuation from leet", "4-3_1.0 5,7", "From Leet Speak", "a-e_i.o s,t"},
		{"accents untouched", "café", "To Leet Speak", "c4fé"},
		{"empty", "", "To Leet Speak", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := runOp(t, "Convert Leet Speak", c.input, c.direction)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}
