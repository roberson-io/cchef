package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

// Binary has no dedicated CyberChef fixture file; these are hand-verified
// vectors (each byte as 8-bit binary) plus round-trips.
func TestBinaryOps(t *testing.T) {
	runCases(t, []opCase{
		{
			"To Binary: nothing", "", "",
			core.Recipe{{Op: "To Binary", Args: []any{"Space", 8}}},
		},
		{
			"To Binary: AB", "AB", "01000001 01000010",
			core.Recipe{{Op: "To Binary", Args: []any{"Space", 8}}},
		},
		{
			"To Binary: Hi!", "Hi!", "01001000 01101001 00100001",
			core.Recipe{{Op: "To Binary", Args: []any{"Space", 8}}},
		},
		{
			"To Binary: None delim", "AB", "0100000101000010",
			core.Recipe{{Op: "To Binary", Args: []any{"None", 8}}},
		},

		{
			"From Binary: AB", "01000001 01000010", "AB",
			core.Recipe{{Op: "From Binary", Args: []any{"Space", 8}}},
		},
		{
			"From Binary: no delim", "0100000101000010", "AB",
			core.Recipe{{Op: "From Binary", Args: []any{"Space", 8}}},
		},

		{
			"Binary round trip", "Hello, World!", "Hello, World!",
			core.Recipe{
				{Op: "To Binary", Args: []any{"Comma", 8}},
				{Op: "From Binary", Args: []any{"Comma", 8}},
			},
		},
	})
}

func TestBinaryBranches(t *testing.T) {
	if _, err := runOp(t, "From Binary", "11111111111111111111111111111111111111111111111111111111111111111", "Space", 65.0); err == nil {
		t.Fatal("From Binary: expected an error for oversized byte length")
	}
	// Direct Run calls reach the ArgDef-guarded width branches: To Binary width 0
	// falls back to 8; From Binary width 0 errors rather than misbehaving.
	if _, err := (ToBinary{}).Run(abytes("A"), []any{"Space", float64(0)}); err != nil {
		t.Fatalf("To Binary width 0: %v", err)
	}
	if _, err := (FromBinary{}).Run(sdish("01000001"), []any{"Space", float64(0)}); err == nil {
		t.Fatal("From Binary width 0: expected an error")
	}
}

// TestToBinaryByteLengthMax pins the upper bound CyberChef declares on To
// Binary's byte length; the lower bound is already declared. From Binary
// carries no upper bound upstream, so it accepts the same value.
func TestToBinaryByteLengthMax(t *testing.T) {
	op, _ := core.Default.Get("To Binary")
	args := core.DefaultArgs(op.Args())
	args[1] = float64(257)
	_, err := core.CoerceArgs(op.Args(), args)
	if err == nil {
		t.Fatal("a byte length of 257 was accepted")
	}
	if want := "Byte Length must be less than or equal to 256."; err.Error() != want {
		t.Errorf("got %q, want %q", err.Error(), want)
	}

	from, _ := core.Default.Get("From Binary")
	fromArgs := core.DefaultArgs(from.Args())
	fromArgs[1] = float64(257)
	if _, err := core.CoerceArgs(from.Args(), fromArgs); err != nil {
		t.Errorf("From Binary has no upper bound upstream, but 257 was refused: %v", err)
	}
}
