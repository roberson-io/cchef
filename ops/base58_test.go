package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

const (
	b58Bitcoin = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
	b58Ripple  = "rpshnaf39wBUDNEGHJKLM4PQRST7VWXYZ2bcdeCg65jkm8oFqi1tuvAxyz"
)

// Cases transcribed from CyberChef tests/operations/tests/Base58.mjs.
func TestBase58Fixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"To Base58 (Bitcoin): nothing", "", "",
			core.Recipe{{Op: "To Base58", Args: []any{b58Bitcoin}}},
		},
		{
			"To Base58 (Bitcoin): hello world", "hello world", "StV1DL6CwTryKyV",
			core.Recipe{{Op: "To Base58", Args: []any{b58Bitcoin}}},
		},
		{
			"To Base58 (Ripple): hello world", "hello world", "StVrDLaUATiyKyV",
			core.Recipe{{Op: "To Base58", Args: []any{b58Ripple}}},
		},
		{
			"To Base58 all null", "\x00\x00\x00\x00\x00\x00", "111111",
			core.Recipe{{Op: "To Base58", Args: []any{b58Bitcoin}}},
		},
		{
			"To Base58 null prefix/suffix", "\x00\x00\x00Hello\x00\x00\x00", "111D7LMXYjHjTu",
			core.Recipe{{Op: "To Base58", Args: []any{b58Bitcoin}}},
		},

		{
			"From Base58 all null", "111111", "\x00\x00\x00\x00\x00\x00",
			core.Recipe{{Op: "From Base58", Args: []any{b58Bitcoin, true}}},
		},
		{
			"From Base58 null prefix/suffix", "111D7LMXYjHjTu", "\x00\x00\x00Hello\x00\x00\x00",
			core.Recipe{{Op: "From Base58", Args: []any{b58Bitcoin, true}}},
		},

		{
			"Base58 round trip", "The quick brown fox", "The quick brown fox",
			core.Recipe{
				{Op: "To Base58", Args: []any{b58Bitcoin}},
				{Op: "From Base58", Args: []any{b58Bitcoin, true}},
			},
		},
	})
}

func TestBase58Errors(t *testing.T) {
	cases := []struct {
		name, op, input string
		args            []any
	}{
		{"To Base58 rejects wrong-length alphabet", "To Base58", "x", []any{"short"}},
		{"From Base58 rejects wrong-length alphabet", "From Base58", "x", []any{"short", false}},
		{"From Base58 rejects char not in alphabet", "From Base58", "0", []any{base58Bitcoin, false}},
	}
	for _, c := range cases {
		if _, err := runOp(t, c.op, c.input, c.args...); err == nil {
			t.Fatalf("%s: expected an error", c.name)
		}
	}
}

func TestBase58ValueBranches(t *testing.T) {
	if out, err := runOp(t, "From Base58", "", base58Bitcoin, false); err != nil || out != "" {
		t.Fatalf("From Base58(\"\") = %q, %v; want empty", out, err)
	}
	if _, err := runOp(t, "From Base58", "0OIl", base58Bitcoin, true); err != nil {
		t.Fatalf("From Base58(strip): %v", err)
	}
}
