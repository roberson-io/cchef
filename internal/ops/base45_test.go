package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

const b45Alph = `0-9A-Z $%*+\-./:`

// Cases transcribed from CyberChef tests/operations/tests/Base45.mjs.
func TestBase45Fixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"To Base45: nothing", "", "",
			core.Recipe{{Op: "To Base45", Args: []any{b45Alph}}},
		},
		{
			"To Base45: AB", "AB", "BB8",
			core.Recipe{{Op: "To Base45", Args: []any{b45Alph}}},
		},
		{
			"To Base45: Hello!!", "Hello!!", "%69 VD92EX0",
			core.Recipe{{Op: "To Base45", Args: []any{b45Alph}}},
		},
		{
			"To Base45: base-45", "base-45", "UJCLQE7W581",
			core.Recipe{{Op: "To Base45", Args: []any{b45Alph}}},
		},

		{
			"From Base45: nothing", "", "",
			core.Recipe{{Op: "From Base45", Args: []any{b45Alph, true}}},
		},
		{
			"From Base45: ietf", "QED8WEX0", "ietf!",
			core.Recipe{{Op: "From Base45", Args: []any{b45Alph, true}}},
		},

		{
			"Base45 round trip", "Hello, World!", "Hello, World!",
			core.Recipe{
				{Op: "To Base45", Args: []any{b45Alph}},
				{Op: "From Base45", Args: []any{b45Alph, true}},
			},
		},
	})
}

func TestBase45Errors(t *testing.T) {
	cases := []struct {
		name, op, input string
		args            []any
	}{
		{"From Base45 rejects char not in alphabet", "From Base45", "~~~", []any{base45Alphabet, false}},
		{"From Base45 rejects oversized triplet", "From Base45", ":::", []any{base45Alphabet, false}},
	}
	for _, c := range cases {
		if _, err := runOp(t, c.op, c.input, c.args...); err == nil {
			t.Fatalf("%s: expected an error", c.name)
		}
	}
}

func TestBase45ZeroPairPadding(t *testing.T) {
	// A two-byte zero pair needs padding to three symbols.
	if _, err := runOp(t, "To Base45", "\x00\x00", base45Alphabet); err != nil {
		t.Fatalf("To Base45(0,0): %v", err)
	}
}
