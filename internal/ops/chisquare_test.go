package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// TestChiSquareValues covers the distribution against the oracle. CyberChef
// ships no case for this operation, so the values come from the running server.
func TestChiSquareValues(t *testing.T) {
	for _, tc := range []struct{ name, input, want string }{
		{"a single byte", "a", "254.00390625"},
		{"one byte repeated", "aaaa", "1016.015625"},
		{"a short phrase", "Hello world!", "403.0885416666666"},
		{
			"a longer one", "Hello world, this is a test to determine the correct IC value.",
			"1227.3926411290324",
		},
		{"a pangram", "The quick brown fox jumps over the lazy dog", "650.9821947674421"},
		{"digits", "0123456789", "236.39062499999997"},
		{"two bytes in equal measure", "AAAAAAAAAABBBBBBBBBB", "2520.15625"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := core.Recipe{{Op: "Chi Square", Args: []any{}}}.
				Execute(core.NewDish([]byte(tc.input), core.TypeString))
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if out.String() != tc.want {
				t.Errorf("got %s, want %s", out.String(), tc.want)
			}
		})
	}
}

// TestChiSquareEmptyInput covers input with no bytes at all, which leaves every
// term of the sum out and so scores nothing.
func TestChiSquareEmptyInput(t *testing.T) {
	out, err := core.Recipe{{Op: "Chi Square", Args: []any{}}}.
		Execute(core.NewDish(nil, core.TypeArrayBuffer))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.String() != "0" {
		t.Errorf("got %s, want 0", out.String())
	}
}
