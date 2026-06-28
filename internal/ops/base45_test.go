package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

const b45Alph = `0-9A-Z $%*+\-./:`

// Cases transcribed from CyberChef tests/operations/tests/Base45.mjs.
func TestBase45Fixtures(t *testing.T) {
	runCases(t, []opCase{
		{"To Base45: nothing", "", "",
			core.Recipe{{Op: "To Base45", Args: []any{b45Alph}}}},
		{"To Base45: AB", "AB", "BB8",
			core.Recipe{{Op: "To Base45", Args: []any{b45Alph}}}},
		{"To Base45: Hello!!", "Hello!!", "%69 VD92EX0",
			core.Recipe{{Op: "To Base45", Args: []any{b45Alph}}}},
		{"To Base45: base-45", "base-45", "UJCLQE7W581",
			core.Recipe{{Op: "To Base45", Args: []any{b45Alph}}}},

		{"From Base45: nothing", "", "",
			core.Recipe{{Op: "From Base45", Args: []any{b45Alph, true}}}},
		{"From Base45: ietf", "QED8WEX0", "ietf!",
			core.Recipe{{Op: "From Base45", Args: []any{b45Alph, true}}}},

		{"Base45 round trip", "Hello, World!", "Hello, World!",
			core.Recipe{
				{Op: "To Base45", Args: []any{b45Alph}},
				{Op: "From Base45", Args: []any{b45Alph, true}},
			}},
	})
}
