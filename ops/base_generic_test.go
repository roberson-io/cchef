package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

// To Base / From Base have no CyberChef fixture file; these are hand-verified
// integer radix conversions (Go big.Int and JS Number.toString share the
// lower-case digit alphabet for radixes 2-36).
func TestBaseGeneric(t *testing.T) {
	runCases(t, []opCase{
		{
			"To Base 16", "255", "ff",
			core.Recipe{{Op: "To Base", Args: []any{16}}},
		},
		{
			"To Base 2", "255", "11111111",
			core.Recipe{{Op: "To Base", Args: []any{2}}},
		},
		{
			"To Base 36", "1295", "zz",
			core.Recipe{{Op: "To Base", Args: []any{36}}},
		},

		{
			"From Base 16", "ff", "255",
			core.Recipe{{Op: "From Base", Args: []any{16}}},
		},
		{
			"From Base 2", "11111111", "255",
			core.Recipe{{Op: "From Base", Args: []any{2}}},
		},
		{
			"From Base 36 (whitespace stripped)", "z z", "1295",
			core.Recipe{{Op: "From Base", Args: []any{36}}},
		},

		{
			"Base round trip", "123456789", "123456789",
			core.Recipe{
				{Op: "To Base", Args: []any{16}},
				{Op: "From Base", Args: []any{16}},
			},
		},
	})
}

func TestBaseGenericErrors(t *testing.T) {
	cases := []struct {
		name, op, input string
		args            []any
	}{
		{"To Base rejects empty", "To Base", "", []any{16.0}},
		{"To Base rejects non-integer", "To Base", "abc", []any{16.0}},
		{"From Base rejects out-of-range radix", "From Base", "ff", []any{99.0}},
		{"From Base rejects fractional", "From Base", "1.5", []any{10.0}},
		{"From Base rejects invalid digits", "From Base", "xyz", []any{2.0}},
	}
	for _, c := range cases {
		if _, err := runOp(t, c.op, c.input, c.args...); err == nil {
			t.Fatalf("%s: expected an error", c.name)
		}
	}
}
