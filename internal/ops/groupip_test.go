package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// TestGroupIPAddressesOracle checks Group IP addresses against CyberChef-server
// output (v11.2.0); no upstream fixture. Networks are emitted in ascending order
// and each is followed by its members. Args are [Delimiter, Subnet (CIDR), Only
// show the subnets].
func TestGroupIPAddressesOracle(t *testing.T) {
	runCases(t, []opCase{
		{
			"IPv4 /24", "192.168.1.5\n192.168.1.200\n10.0.0.1",
			"10.0.0.0/24\n  10.0.0.1\n\n192.168.1.0/24\n  192.168.1.5\n  192.168.1.200\n\n",
			core.Recipe{{Op: "Group IP addresses", Args: []any{"Line feed", 24, false}}},
		},
		{
			"IPv6 /64", "ff00::1111:2222\nff00::1111:3333",
			"ff00::/64\n  ff00::1111:2222\n  ff00::1111:3333\n\n",
			core.Recipe{{Op: "Group IP addresses", Args: []any{"Line feed", 64, false}}},
		},
		{
			"IPv4 /24 only subnets", "192.168.1.5\n192.168.1.200\n10.0.0.1",
			"10.0.0.0/24\n192.168.1.0/24\n",
			core.Recipe{{Op: "Group IP addresses", Args: []any{"Line feed", 24, true}}},
		},
	})
}
