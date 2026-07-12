package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// TestIPv6TransitionFixtures transcribes CyberChef's IPv6Transition.mjs cases:
// IPv4/range → transition addresses, IPv6 → IPv4, and MAC → EUI-64. Args are
// [Ignore ranges, Remove headers].
func TestIPv6TransitionFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"IPv4 to IPv6", "198.51.100.7",
			"6to4: 2002:c633:6407::/48\nIPv4 Mapped: ::ffff:c633:6407\nIPv4 Translated: ::ffff:0:c633:6407\nNat 64: 64:ff9b::c633:6407\n",
			core.Recipe{{Op: "IPv6 Transition Addresses", Args: []any{true, false}}},
		},
		{
			"IPv4 /24 Range to IPv6", "198.51.100.0/24",
			"6to4: 2002:c633:6400::/40\nIPv4 Mapped: ::ffff:c633:6400/120\nIPv4 Translated: ::ffff:0:c633:6400/120\nNat 64: 64:ff9b::c633:6400/120\n",
			core.Recipe{{Op: "IPv6 Transition Addresses", Args: []any{false, false}}},
		},
		{
			"IPv4 to IPv6 Remove headers", "198.51.100.7",
			"2002:c633:6407::/48\n::ffff:c633:6407\n::ffff:0:c633:6407\n64:ff9b::c633:6407\n",
			core.Recipe{{Op: "IPv6 Transition Addresses", Args: []any{true, true}}},
		},
		{
			"IPv6 to IPv4", "64:ff9b::c633:6407", "IPv4: 198.51.100.7\n",
			core.Recipe{{Op: "IPv6 Transition Addresses", Args: []any{true, false}}},
		},
		{
			"MAC to EUI-64", "a1:b2:c3:d4:e5:f6", "EUI-64 Interface ID: a3b2:c3ff:fed4:e5f6",
			core.Recipe{{Op: "IPv6 Transition Addresses", Args: []any{true, false}}},
		},
		// 6to4 (2002::) reverse path, plus a multi-line mixed input.
		{
			"6to4 to IPv4", "2002:c633:6407::", "IPv4: 198.51.100.7\n",
			core.Recipe{{Op: "IPv6 Transition Addresses", Args: []any{true, false}}},
		},
		{
			"multi-line, remove headers", "198.51.100.7\na1:b2:c3:d4:e5:f6",
			"2002:c633:6407::/48\n::ffff:c633:6407\n::ffff:0:c633:6407\n64:ff9b::c633:6407\na3b2:c3ff:fed4:e5f6",
			core.Recipe{{Op: "IPv6 Transition Addresses", Args: []any{true, true}}},
		},
		// An EUI-64 IPv6 address (containing ff:fe) reverses back to a MAC address,
		// undoing the U/L bit flip (a3 -> a1).
		{
			"EUI-64 IPv6 to MAC", "fe80::a3b2:c3ff:fed4:e5f6", "Mac Address: A1:B2:C3:D4:E5:F6\n",
			core.Recipe{{Op: "IPv6 Transition Addresses", Args: []any{true, false}}},
		},
		{
			"EUI-64 IPv6 to MAC remove headers", "fe80::a3b2:c3ff:fed4:e5f6", "A1:B2:C3:D4:E5:F6\n",
			core.Recipe{{Op: "IPv6 Transition Addresses", Args: []any{true, true}}},
		},
		// Unrecognised input yields the usage message (the default switch arm).
		{
			"invalid input", "not an address",
			"Enter compressed or expanded IPv6 address, IPv4 address or MAC Address.",
			core.Recipe{{Op: "IPv6 Transition Addresses", Args: []any{true, false}}},
		},
		// A compressed IPv4-mapped address: short hextets are zero-padded.
		{
			"mapped compressed padding", "::ffff:1:2", "IPv4: 0.1.0.2\n",
			core.Recipe{{Op: "IPv6 Transition Addresses", Args: []any{true, false}}},
		},
		// A blank line between addresses is skipped.
		{
			"blank line skipped", "198.51.100.7\n\n198.51.100.7",
			"6to4: 2002:c633:6407::/48\nIPv4 Mapped: ::ffff:c633:6407\nIPv4 Translated: ::ffff:0:c633:6407\nNat 64: 64:ff9b::c633:6407\n" +
				"6to4: 2002:c633:6407::/48\nIPv4 Mapped: ::ffff:c633:6407\nIPv4 Translated: ::ffff:0:c633:6407\nNat 64: 64:ff9b::c633:6407\n",
			core.Recipe{{Op: "IPv6 Transition Addresses", Args: []any{true, false}}},
		},
		// With "Ignore ranges" on, a CIDR line is skipped entirely.
		{
			"ignore ranges skips CIDR", "198.51.100.0/24", "",
			core.Recipe{{Op: "IPv6 Transition Addresses", Args: []any{true, false}}},
		},
		// An EUI-64 whose reversed-MAC second nibble is a hex letter: cchef
		// lowercases the XOR-table key, so it renders "0e" where CyberChef (which
		// leaves the key upper-case) renders "0undefined".
		{
			"EUI-64 letter nibble (cchef fixes CyberChef bug)", "fe80::0cb2:c3ff:fed4:e5f6",
			"Mac Address: 0e:B2:C3:D4:E5:F6\n",
			core.Recipe{{Op: "IPv6 Transition Addresses", Args: []any{true, false}}},
		},
		// A Nat64 prefix with no embedded IPv4 hextets: cchef's nil-guard returns
		// empty output, whereas CyberChef throws "Cannot read properties of null".
		{
			"nat64 prefix without embedded IPv4 (cchef guards)", "64:ff9b::", "",
			core.Recipe{{Op: "IPv6 Transition Addresses", Args: []any{false, false}}},
		},
	})
}
