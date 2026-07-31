package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// SHA1 takes a round count, defaulting to the standard 80. Reduced-round SHA-1
// is a cryptanalysis exercise rather than a hash anyone should rely on, so the
// only reference for these digests is CyberChef itself; every expected value
// below was recorded from the oracle.

func TestSHA1Rounds(t *testing.T) {
	runCases(t, []opCase{
		{
			"SHA1: default 80 rounds", "hello", "aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d",
			core.Recipe{{Op: "SHA1"}},
		},
		{
			"SHA1: 80 rounds given explicitly", "hello", "aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d",
			core.Recipe{{Op: "SHA1", Args: []any{80}}},
		},
		{
			"SHA1: 16 rounds", "hello", "975648694251393d5a6480aebef493445079135c",
			core.Recipe{{Op: "SHA1", Args: []any{16}}},
		},
		{
			"SHA1: 20 rounds", "hello", "328f1b8dc5590f0c7e055b0279e8cdadcfd72b4a",
			core.Recipe{{Op: "SHA1", Args: []any{20}}},
		},
		{
			"SHA1: 64 rounds", "hello", "e537356b2e2b2ecc0de26c615120ba968a2537d3",
			core.Recipe{{Op: "SHA1", Args: []any{64}}},
		},
		{
			"SHA1: multi-block input, 48 rounds",
			"The quick brown fox jumps over the lazy dog. The quick brown fox jumps over the lazy dog. The quick brown fox jumps over the lazy dog.",
			"a105b35a8fa0ac354f5b0a19f3807ebbf8987f66",
			core.Recipe{{Op: "SHA1", Args: []any{48}}},
		},
		{
			"SHA1: empty input", "", "da39a3ee5e6b4b0d3255bfef95601890afd80709",
			core.Recipe{{Op: "SHA1"}},
		},
	})
}

// TestSHA1RoundsBound pins the minimum CyberChef declares, and the maximum
// cchef adds: past 80 rounds the message schedule keeps extending, so an
// unbounded count is an unbounded allocation.
func TestSHA1RoundsBound(t *testing.T) {
	op, _ := core.Default.Get("SHA1")
	for _, tc := range []struct {
		rounds float64
		want   string
	}{
		{15, "Rounds must be greater than or equal to 16."},
		{81, "Rounds must be less than or equal to 80."},
	} {
		_, err := core.CoerceArgs(op.Args(), []any{tc.rounds})
		if err == nil {
			t.Fatalf("%v rounds was accepted", tc.rounds)
		}
		if err.Error() != tc.want {
			t.Errorf("got %q, want %q", err.Error(), tc.want)
		}
	}
}
