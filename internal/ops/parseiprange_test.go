package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// pirRecipe is Parse IP range with the default args (network info + enumerate,
// no large-query allowance).
func pirRecipe() core.Recipe {
	return core.Recipe{{Op: "Parse IP range", Args: []any{true, true, false}}}
}

// TestParseIPRangeFixtures transcribes CyberChef's ParseIPRange.mjs success cases.
func TestParseIPRangeFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"IPv4 CIDR", "10.0.0.0/30",
			"Network: 10.0.0.0\nCIDR: 30\nMask: 255.255.255.252\nRange: 10.0.0.0 - 10.0.0.3\nTotal addresses in range: 4\n\n10.0.0.0\n10.0.0.1\n10.0.0.2\n10.0.0.3", pirRecipe(),
		},
		{
			"IPv4 hyphenated", "10.0.0.0 - 10.0.0.3",
			"Minimum subnet required to hold this range:\n\tNetwork: 10.0.0.0\n\tCIDR: 30\n\tMask: 255.255.255.252\n\tSubnet range: 10.0.0.0 - 10.0.0.3\n\tTotal addresses in subnet: 4\n\nRange: 10.0.0.0 - 10.0.0.3\nTotal addresses in range: 4\n\n10.0.0.0\n10.0.0.1\n10.0.0.2\n10.0.0.3", pirRecipe(),
		},
		{
			"IPv4 list", "10.0.0.8\n10.0.0.5/30\n10.0.0.1\n10.0.0.3",
			"Minimum subnet required to hold this range:\n\tNetwork: 10.0.0.0\n\tCIDR: 28\n\tMask: 255.255.255.240\n\tSubnet range: 10.0.0.0 - 10.0.0.15\n\tTotal addresses in subnet: 16\n\nRange: 10.0.0.1 - 10.0.0.8\nTotal addresses in range: 8\n\n10.0.0.1\n10.0.0.2\n10.0.0.3\n10.0.0.4\n10.0.0.5\n10.0.0.6\n10.0.0.7\n10.0.0.8", pirRecipe(),
		},
		{
			"IPv6 CIDR full", "2404:6800:4001:0000:0000:0000:0000:0000/48",
			"Network: 2404:6800:4001:0000:0000:0000:0000:0000\nShorthand: 2404:6800:4001::\nCIDR: 48\nMask: ffff:ffff:ffff:0000:0000:0000:0000:0000\nRange: 2404:6800:4001:0000:0000:0000:0000:0000 - 2404:6800:4001:ffff:ffff:ffff:ffff:ffff\nTotal addresses in range: 1.2089258196146292e+24\n\n", pirRecipe(),
		},
		{
			"IPv6 CIDR collapsed", "2404:6800:4001::/48",
			"Network: 2404:6800:4001:0000:0000:0000:0000:0000\nShorthand: 2404:6800:4001::\nCIDR: 48\nMask: ffff:ffff:ffff:0000:0000:0000:0000:0000\nRange: 2404:6800:4001:0000:0000:0000:0000:0000 - 2404:6800:4001:ffff:ffff:ffff:ffff:ffff\nTotal addresses in range: 1.2089258196146292e+24\n\n", pirRecipe(),
		},
		{
			"IPv6 hyphenated", "2404:6800:4001:: - 2404:6800:4001:ffff:ffff:ffff:ffff:ffff",
			"Range: 2404:6800:4001:0000:0000:0000:0000:0000 - 2404:6800:4001:ffff:ffff:ffff:ffff:ffff\nShorthand range: 2404:6800:4001:: - 2404:6800:4001:ffff:ffff:ffff:ffff:ffff\nTotal addresses in range: 1.2089258196146292e+24\n\n", pirRecipe(),
		},
		{
			"IPv6 list", "2404:6800:4001:ffff:ffff:ffff:ffff:ffff\n2404:6800:4001::ffff\n2404:6800:4001:ffff:ffff::1111\n2404:6800:4001::/64",
			"Range: 2404:6800:4001:0000:0000:0000:0000:0000 - 2404:6800:4001:ffff:ffff:ffff:ffff:ffff\nShorthand range: 2404:6800:4001:: - 2404:6800:4001:ffff:ffff:ffff:ffff:ffff\nTotal addresses in range: 1.2089258196146292e+24\n\n", pirRecipe(),
		},
		// Large range without "allow large" hits the large-range warning.
		{
			"IPv4 large range", "10.0.0.0/8",
			"Network: 10.0.0.0\nCIDR: 8\nMask: 255.0.0.0\nRange: 10.0.0.0 - 10.255.255.255\nTotal addresses in range: 16777216\n\n" + largeRangeError,
			core.Recipe{{Op: "Parse IP range", Args: []any{true, true, false}}},
		},
		// Neither network info nor enumeration produces empty output.
		{
			"IPv4 no info no enumerate", "10.0.0.0/30", "",
			core.Recipe{{Op: "Parse IP range", Args: []any{false, false, false}}},
		},
		// A reversed hyphenated range: the address count wraps (uint32 underflow),
		// matching CyberChef, and trips the large-range warning.
		{
			"IPv4 reversed range", "10.0.0.3 - 10.0.0.0",
			"Minimum subnet required to hold this range:\n\tNetwork: 10.0.0.0\n\tCIDR: 30\n\tMask: 255.255.255.252\n\tSubnet range: 10.0.0.0 - 10.0.0.3\n\tTotal addresses in subnet: 4\n\nRange: 10.0.0.3 - 10.0.0.0\nTotal addresses in range: 4294967294\n\n" + largeRangeError,
			core.Recipe{{Op: "Parse IP range", Args: []any{true, true, false}}},
		},
	})
}

// TestParseIPRangeErrors covers the validation error cases (surfaced as errors).
func TestParseIPRangeErrors(t *testing.T) {
	cases := []struct{ name, input, wantErr string }{
		{"IPv4 subnet out of range", "10.1.1.1/34", "IPv4 CIDR must be less than 32"},
		{"invalid IPv4 address", "444.1.1.1/30", "block out of range"},
		{"IPv6 subnet out of range", "2404:6800:4001::/129", "IPv6 CIDR must be less than 128"},
		{"invalid input", "2404:6800:4001:/12", "invalid input"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := pirRecipe().Execute(core.NewDish([]byte(c.input), core.TypeString))
			if err == nil || !strings.Contains(err.Error(), c.wantErr) {
				t.Fatalf("got err %v, want containing %q", err, c.wantErr)
			}
		})
	}
}
