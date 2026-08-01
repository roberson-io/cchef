package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

// Cases transcribed from CyberChef tests/operations/tests/ByteRepr.mjs.
func TestOctalFixtures(t *testing.T) {
	const utf8In = "Γειά σου"
	const utf8Out = "316 223 316 265 316 271 316 254 40 317 203 316 277 317 205"
	runCases(t, []opCase{
		{
			"To Octal: nothing", "", "",
			core.Recipe{{Op: "To Octal", Args: []any{"Space"}}},
		},
		{
			"To Octal: hello world", "hello world", "150 145 154 154 157 40 167 157 162 154 144",
			core.Recipe{{Op: "To Octal", Args: []any{"Space"}}},
		},
		{
			"To Octal: UTF-8", utf8In, utf8Out,
			core.Recipe{{Op: "To Octal", Args: []any{"Space"}}},
		},

		{
			"From Octal: nothing", "", "",
			core.Recipe{{Op: "From Octal", Args: []any{"Space"}}},
		},
		{
			"From Octal: hello world", "150 145 154 154 157 40 167 157 162 154 144", "hello world",
			core.Recipe{{Op: "From Octal", Args: []any{"Space"}}},
		},
		{
			"From Octal: UTF-8", utf8Out, utf8In,
			core.Recipe{{Op: "From Octal", Args: []any{"Space"}}},
		},

		{
			"Octal round trip (Comma)", "Hello, World!", "Hello, World!",
			core.Recipe{
				{Op: "To Octal", Args: []any{"Comma"}},
				{Op: "From Octal", Args: []any{"Comma"}},
			},
		},
	})
}

func TestOctalBranches(t *testing.T) {
	if _, err := runOp(t, "From Octal", "999", "Space"); err == nil {
		t.Fatal("From Octal: expected an error for an invalid digit")
	}
	// A delimiter that maps to empty treats the input as one token (shared decode
	// path's None handling; octal exposes no such option).
	if _, err := (FromOctal{}).Run(sdish("77"), []any{"None"}); err != nil {
		t.Fatalf("From Octal None delim: %v", err)
	}
}
