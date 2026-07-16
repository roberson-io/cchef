package ops

import (
	"strings"
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

// TestGroupIPErrors covers the CIDR-range guard and the strToIpv4 error for an
// octet that passes the routing regex but exceeds 255.
func TestGroupIPErrors(t *testing.T) {
	if _, err := runOp(t, "Group IP addresses", "1.2.3.4", "Comma", 200, false); err == nil {
		t.Fatal("expected an error for an out-of-range CIDR")
	}
	if _, err := runOp(t, "Group IP addresses", "999.0.0.0", "Comma", 24, false); err == nil {
		t.Fatal("expected an error for an out-of-range octet")
	}
}

// TestStrToIPValidators exercises the strToIpv4/strToIpv6 parser guards directly
// (callers pre-validate with a regex, so these only fire for arbitrary input):
// a wrong block count is an error, and a non-hex IPv6 block is treated as an
// empty "::" segment rather than an error.
func TestStrToIPValidators(t *testing.T) {
	if _, err := strToIpv4("1.2.3"); err == nil {
		t.Error("strToIpv4(1.2.3): expected a block-count error")
	}
	if _, err := strToIpv6("1:2"); err == nil {
		t.Error("strToIpv6(1:2): expected a block-count error")
	}
	got, err := strToIpv6("1:2:zzzz")
	if err != nil {
		t.Fatalf("strToIpv6(non-hex block): unexpected error %v", err)
	}
	if got[0] != 1 || got[1] != 2 {
		t.Errorf("strToIpv6(1:2:zzzz) = %v; want a leading 1, 2", got)
	}
}

// --- direct tests for the helpers extracted from GroupIPAddresses.Run ---

// TestGroupIPTokens documents parsing tokens into per-network groups: two IPv4s
// in the same /24 collapse to one network with both members.
func TestGroupIPTokens(t *testing.T) {
	g, err := groupIPTokens("10.0.0.1,10.0.0.2", ",", 0xFFFFFF00, [8]uint32{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(g.ipv4) != 1 {
		t.Fatalf("expected 1 network, got %d", len(g.ipv4))
	}
	for _, ips := range g.ipv4 {
		if len(ips) != 2 {
			t.Fatalf("expected 2 members, got %d", len(ips))
		}
	}
}

// TestWriteIPv4Networks documents the IPv4 rendering (network line then indented
// members).
func TestWriteIPv4Networks(t *testing.T) {
	var out strings.Builder
	writeIPv4Networks(&out, map[uint32][]uint32{0x0A000000: {0x0A000001}}, 24, false)
	if out.String() != "10.0.0.0/24\n  10.0.0.1\n\n" {
		t.Fatalf("got %q", out.String())
	}
}

// TestWriteIPv6Networks documents the IPv6 rendering in first-seen order; with
// onlySubnets only the network lines are emitted.
func TestWriteIPv6Networks(t *testing.T) {
	var out strings.Builder
	writeIPv6Networks(&out, map[string][][8]int{}, []string{"2001:db8::"}, 32, true)
	if out.String() != "2001:db8::/32\n" {
		t.Fatalf("got %q", out.String())
	}
}
