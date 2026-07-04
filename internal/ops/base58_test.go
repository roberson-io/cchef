package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
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
