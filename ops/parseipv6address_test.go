package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// TestParseIPv6AddressOracle checks Parse IPv6 address against CyberChef-server
// output (v11.2.0); there is no upstream fixture. Covers the distinct
// address-type branches: unspecified, loopback, IPv4-mapped, documentation,
// link-local + EUI-64, multicast, 6to4, Teredo and unique-local.
func TestParseIPv6AddressOracle(t *testing.T) {
	p := core.Recipe{{Op: "Parse IPv6 address", Args: []any{}}}
	runCases(t, []opCase{
		{
			"unspecified", "::",
			"Longhand:  0000:0000:0000:0000:0000:0000:0000:0000\nShorthand: ::\n\nUnspecified address corresponding to 0.0.0.0/32 in IPv4.\nUnspecified address range: ::/128", p,
		},
		{
			"loopback", "::1",
			"Longhand:  0000:0000:0000:0000:0000:0000:0000:0001\nShorthand: ::1\n\nLoopback address to the local host corresponding to 127.0.0.1/8 in IPv4.\nLoopback addresses range: ::1/128", p,
		},
		{
			"ipv4-mapped", "::ffff:c0a8:0101",
			"Longhand:  0000:0000:0000:0000:0000:ffff:c0a8:0101\nShorthand: ::ffff:c0a8:101\n\nIPv4-mapped IPv6 address detected. IPv6 clients will be handled natively by default, and IPv4 clients appear as IPv6 clients at their IPv4-mapped IPv6 address.\nMapped IPv4 address: 192.168.1.1\nIPv4-mapped IPv6 addresses range: ::ffff:0:0/96", p,
		},
		{
			"documentation", "2001:db8::1",
			"Longhand:  2001:0db8:0000:0000:0000:0000:0000:0001\nShorthand: 2001:db8::1\n\nThis is a documentation IPv6 address. This range should be used whenever an example IPv6 address is given or to model networking scenarios. Corresponds to 192.0.2.0/24, 198.51.100.0/24, and 203.0.113.0/24 in IPv4.\nDocumentation range: 2001:db8::/32", p,
		},
		{
			"link-local + EUI-64", "fe80::213:c3ff:fedf:ae18",
			"Longhand:  fe80:0000:0000:0000:0213:c3ff:fedf:ae18\nShorthand: fe80::213:c3ff:fedf:ae18\n\nThis is a link-local address comparable to the auto-configuration addresses 169.254.0.0/16 in IPv4.\nLink-local addresses range: fe80::/10\n\nThis IPv6 address contains a modified EUI-64 address, identified by the presence of FF:FE in the 12th and 13th octets.\nInterface identifier: 02:13:c3:ff:fe:df:ae:18\nMAC address:          00:13:c3:df:ae:18", p,
		},
		{
			"multicast all-nodes", "ff02::1",
			"Longhand:  ff02:0000:0000:0000:0000:0000:0000:0001\nShorthand: ff02::1\n\nThis is a reserved multicast address.\nMulticast addresses range: ff00::/8\n\nReserved Multicast Block for Link Local Scope\nReserved Multicast Address for 'All nodes'", p,
		},
		{
			"6to4", "2002:c0a8:0101:0:0:0:0:1",
			"Longhand:  2002:c0a8:0101:0000:0000:0000:0000:0001\nShorthand: 2002:c0a8:101::1\n\n6to4 transition IPv6 address detected. See RFC 3056 for more details.\n6to4 prefix range: 2002::/16\n\nEncapsulated IPv4 address: 192.168.1.1\nSLA ID: 0\nInterface ID (base 16): 0001\nInterface ID (base 10): 1", p,
		},
		{
			"teredo", "2001:0000:4136:e378:8000:63bf:3fff:fdd2",
			"Longhand:  2001:0000:4136:e378:8000:63bf:3fff:fdd2\nShorthand: 2001:0:4136:e378:8000:63bf:3fff:fdd2\n\nTeredo tunneling IPv6 address detected\n\nServer IPv4 address: 65.54.227.120\nClient IPv4 address: 192.0.2.45\nClient UDP port:     40000\nFlags:\n\tCone:    1 (Client is behind a cone NAT)\n\tR:       0\n\tRandom1: 0000\n\tUG:      00\n\tRandom2: 00000000\n\nThis is a valid Teredo address which complies with RFC 4380, however it does not comply with RFC 5991 (Teredo Security Updates) as there are no randomised bits in the flag field.\n\nTeredo prefix range: 2001::/32", p,
		},
		{
			"unique-local", "fd12:3456:789a:1::1",
			"Longhand:  fd12:3456:789a:0001:0000:0000:0000:0001\nShorthand: fd12:3456:789a:1::1\n\nThis is a unique local address comparable to the IPv4 private addresses 10.0.0.0/8, 172.16.0.0/12 and 192.168.0.0/16. See RFC 4193 for more details.\nUnique local addresses range: fc00::/7", p,
		},
		{
			"ipv4-translated", "0:0:0:0:ffff:0:c0a8:0101",
			"Longhand:  0000:0000:0000:0000:ffff:0000:c0a8:0101\nShorthand: ::ffff:0:c0a8:101\n\nIPv4-translated address detected. Used by Stateless IP/ICMP Translation (SIIT). See RFCs 6145 and 6052 for more details.\nTranslated IPv4 address: 192.168.1.1\nIPv4-translated addresses range: ::ffff:0:0:0/96", p,
		},
		{
			"discard", "100::1",
			"Longhand:  0100:0000:0000:0000:0000:0000:0000:0001\nShorthand: 100::1\n\nDiscard prefix detected. This is used when forwarding traffic to a sinkhole router to mitigate the effects of a denial-of-service attack. See RFC 6666 for more details.\nDiscard range: 100::/64", p,
		},
		{
			"well-known translation", "64:ff9b::c0a8:0101",
			"Longhand:  0064:ff9b:0000:0000:0000:0000:c0a8:0101\nShorthand: 64:ff9b::c0a8:101\n\n'Well-Known' prefix for IPv4/IPv6 translation detected. See RFC 6052 for more details.\nTranslated IPv4 address: 192.168.1.1\n'Well-Known' prefix range: 64:ff9b::/96", p,
		},
		{
			"benchmarking", "2001:2::1",
			"Longhand:  2001:0002:0000:0000:0000:0000:0000:0001\nShorthand: 2001:2::1\n\nAssigned to the Benchmarking Methodology Working Group (BMWG) for benchmarking IPv6. Corresponds to 198.18.0.0/15 for benchmarking IPv4. See RFC 5180 for more details.\nBMWG range: 2001:2::/48", p,
		},
		{
			"ORCHIDv1", "2001:10::1",
			"Longhand:  2001:0010:0000:0000:0000:0000:0000:0001\nShorthand: 2001:10::1\n\nDeprecated, previously ORCHIDv1 (Overlay Routable Cryptographic Hash Identifiers).\nORCHIDv1 range: 2001:10::/28\nORCHIDv2 now uses 2001:20::/28.", p,
		},
		{
			"ORCHIDv2", "2001:20::1",
			"Longhand:  2001:0020:0000:0000:0000:0000:0000:0001\nShorthand: 2001:20::1\n\nORCHIDv2 (Overlay Routable Cryptographic Hash Identifiers).\nThese are non-routed IPv6 addresses used for Cryptographic Hash Identifiers.\nORCHIDv2 range: 2001:20::/28", p,
		},
		{
			"multicast site-local + LLMNR", "ff05::1:3",
			"Longhand:  ff05:0000:0000:0000:0000:0000:0001:0003\nShorthand: ff05::1:3\n\nThis is a reserved multicast address.\nMulticast addresses range: ff00::/8\n\nReserved Multicast Block for Site Local Scope\nReserved Multicast Address for 'All LLMNR Hosts (defined in RFC4795)'", p,
		},
		{
			"multicast global + routers", "ff0e::2",
			"Longhand:  ff0e:0000:0000:0000:0000:0000:0000:0002\nShorthand: ff0e::2\n\nThis is a reserved multicast address.\nMulticast addresses range: ff00::/8\n\nReserved Multicast Block for Global Scope\nReserved Multicast Address for 'All routers'", p,
		},
		{
			"mcast interface-local", "ff01::1",
			"Longhand:  ff01:0000:0000:0000:0000:0000:0000:0001\nShorthand: ff01::1\n\nThis is a reserved multicast address.\nMulticast addresses range: ff00::/8\n\nReserved Multicast Block for Interface Local Scope\nReserved Multicast Address for 'All nodes'", p,
		},
		{
			"mcast realm-local", "ff03::1",
			"Longhand:  ff03:0000:0000:0000:0000:0000:0000:0001\nShorthand: ff03::1\n\nThis is a reserved multicast address.\nMulticast addresses range: ff00::/8\n\nReserved Multicast Block for Realm Local Scope\nReserved Multicast Address for 'All nodes'", p,
		},
		{
			"mcast admin-local", "ff04::1",
			"Longhand:  ff04:0000:0000:0000:0000:0000:0000:0001\nShorthand: ff04::1\n\nThis is a reserved multicast address.\nMulticast addresses range: ff00::/8\n\nReserved Multicast Block for Admin Local Scope\nReserved Multicast Address for 'All nodes'", p,
		},
		{
			"mcast site-local all-routers", "ff05::2",
			"Longhand:  ff05:0000:0000:0000:0000:0000:0000:0002\nShorthand: ff05::2\n\nThis is a reserved multicast address.\nMulticast addresses range: ff00::/8\n\nReserved Multicast Block for Site Local Scope\nReserved Multicast Address for 'All routers'", p,
		},
		{
			"mcast org-local mdns", "ff08::fb",
			"Longhand:  ff08:0000:0000:0000:0000:0000:0000:00fb\nShorthand: ff08::fb\n\nThis is a reserved multicast address.\nMulticast addresses range: ff00::/8\n\nReserved Multicast Block for Organisation Local Scope\nReserved Multicast Address for 'Multicast DNS'", p,
		},
		{
			"mcast global ntp", "ff0e::101",
			"Longhand:  ff0e:0000:0000:0000:0000:0000:0000:0101\nShorthand: ff0e::101\n\nThis is a reserved multicast address.\nMulticast addresses range: ff00::/8\n\nReserved Multicast Block for Global Scope\nReserved Multicast Address for 'Network Time Protocol'", p,
		},
		{
			"mcast dhcp servers", "ff05::1:2",
			"Longhand:  ff05:0000:0000:0000:0000:0000:0001:0002\nShorthand: ff05::1:2\n\nThis is a reserved multicast address.\nMulticast addresses range: ff00::/8\n\nReserved Multicast Block for Site Local Scope\nReserved Multicast Address for 'All DHCP Servers and Relay Agents (defined in RFC3315)'", p,
		},
		{
			"mcast llmnr", "ff02::1:3",
			"Longhand:  ff02:0000:0000:0000:0000:0000:0001:0003\nShorthand: ff02::1:3\n\nThis is a reserved multicast address.\nMulticast addresses range: ff00::/8\n\nReserved Multicast Block for Link Local Scope\nReserved Multicast Address for 'All LLMNR Hosts (defined in RFC4795)'", p,
		},
		// A Teredo tunneling address (2001::/32) decodes the embedded server/client
		// IPv4, UDP port and flag bits (cone NAT + the RFC 5991 compliance note).
		{
			"teredo", "2001:0000:4136:e378:8000:63bf:3fff:fdd2",
			"Longhand:  2001:0000:4136:e378:8000:63bf:3fff:fdd2\nShorthand: 2001:0:4136:e378:8000:63bf:3fff:fdd2\n\nTeredo tunneling IPv6 address detected\n\nServer IPv4 address: 65.54.227.120\nClient IPv4 address: 192.0.2.45\nClient UDP port:     40000\nFlags:\n\tCone:    1 (Client is behind a cone NAT)\n\tR:       0\n\tRandom1: 0000\n\tUG:      00\n\tRandom2: 00000000\n\nThis is a valid Teredo address which complies with RFC 4380, however it does not comply with RFC 5991 (Teredo Security Updates) as there are no randomised bits in the flag field.\n\nTeredo prefix range: 2001::/32", p,
		},
	})
}

// TestParseIPv6AddressInvalid covers the rejection of non-IPv6 input.
func TestParseIPv6AddressInvalid(t *testing.T) {
	_, err := core.Recipe{{Op: "Parse IPv6 address", Args: []any{}}}.
		Execute(core.NewDish([]byte("not an address"), core.TypeString))
	if err == nil || !strings.Contains(err.Error(), "invalid IPv6 address") {
		t.Fatalf("got err %v, want invalid-address error", err)
	}
}

// TestParseIPv6AddressTeredoFlags exercises the Teredo flag branches not hit by
// the oracle fixture (which uses a cone/no-random address). Wording is
// oracle-verified.
func TestParseIPv6AddressTeredoFlags(t *testing.T) {
	cases := []struct {
		addr string
		want []string
	}{
		// flags 0x0440: not behind cone NAT, random1/random2 set -> fully valid.
		{"2001:0:0:0:440:0:0:0", []string{
			"Cone:    0 (Client is not behind a cone NAT)",
			"This is a valid Teredo address which complies with RFC 4380 and RFC 5991.",
		}},
		// flags 0x4000: R flag set -> error line and invalid address.
		{"2001:0:0:0:4000:0:0:0", []string{
			"R:       1 Error: This flag should be set to 0. See RFC 5991 and RFC 4380.",
			"This is an invalid Teredo address.",
		}},
		// flags 0x0100: UG flag set -> error line.
		{"2001:0:0:0:100:0:0:0", []string{
			"UG:      01 Error: This flag should be set to 00. See RFC 4380.",
		}},
	}
	for _, c := range cases {
		out, err := runOp(t, "Parse IPv6 address", c.addr)
		if err != nil {
			t.Fatalf("%s: %v", c.addr, err)
		}
		for _, w := range c.want {
			if !strings.Contains(out, w) {
				t.Errorf("%s: missing %q in:\n%s", c.addr, w, out)
			}
		}
	}
}

// TestParseIPv6AddressMulticastDefaults covers the multicast scope/address
// helpers' fall-through (empty) returns: ff00:: has an unlisted scope and an
// unlisted low hextet; ff02::1:5 has ipv6[6]==1 with an unlisted ipv6[7].
func TestParseIPv6AddressMulticastDefaults(t *testing.T) {
	for _, addr := range []string{"ff00::", "ff02::1:5"} {
		out, err := runOp(t, "Parse IPv6 address", addr)
		if err != nil {
			t.Fatalf("%s: %v", addr, err)
		}
		if !strings.Contains(out, "This is a reserved multicast address.") {
			t.Errorf("%s: missing multicast line in:\n%s", addr, out)
		}
	}
}

// --- direct tests for the helpers extracted from ParseIPv6Address.Run ---

// TestIPv6MappedIPv4 documents packing the last two hextets into an IPv4 value.
func TestIPv6MappedIPv4(t *testing.T) {
	ipv6 := [8]int{0, 0, 0, 0, 0, 0xffff, 0x0102, 0x0304}
	if got := ipv6MappedIPv4(ipv6); got != 0x01020304 {
		t.Fatalf("ipv6MappedIPv4 = %#x, want 0x01020304", got)
	}
}

// TestDescribeIPv6Type documents the classification switch for representative
// address types.
func TestDescribeIPv6Type(t *testing.T) {
	cases := []struct {
		name      string
		ipv6      [8]int
		shorthand string
		wantSub   string
	}{
		{"unspecified", [8]int{}, "::", "Unspecified address"},
		{"loopback", [8]int{0, 0, 0, 0, 0, 0, 0, 1}, "::1", "Loopback address"},
		{"mapped IPv4", [8]int{0, 0, 0, 0, 0, 0xffff, 0x0102, 0x0304}, "::ffff:102:304", "IPv4-mapped IPv6 address"},
		{"multicast link-local", [8]int{0xff02, 0, 0, 0, 0, 0, 0, 1}, "ff02::1", "reserved multicast address"},
		{"documentation", [8]int{0x2001, 0xdb8, 0, 0, 0, 0, 0, 0}, "2001:db8::", "documentation IPv6 address"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var b strings.Builder
			describeIPv6Type(&b, c.ipv6, c.shorthand)
			if !strings.Contains(b.String(), c.wantSub) {
				t.Fatalf("output %q missing %q", b.String(), c.wantSub)
			}
		})
	}
}

// TestWriteEUI64 documents the modified-EUI-64 interface/MAC extraction, keyed on
// the FF:FE marker in the middle hextets.
func TestWriteEUI64(t *testing.T) {
	// Marker: (ipv6[5] & 0xff) == 0xff and (ipv6[6] >> 8) == 0xfe.
	ipv6 := [8]int{0x2001, 0xdb8, 0, 0, 0x0211, 0x22ff, 0xfe33, 0x4455}
	var b strings.Builder
	writeEUI64(&b, ipv6)
	out := b.String()
	if !strings.Contains(out, "modified EUI-64") || !strings.Contains(out, "MAC address") {
		t.Fatalf("EUI-64 output missing expected fields: %q", out)
	}

	// A non-EUI-64 address writes nothing.
	var b2 strings.Builder
	writeEUI64(&b2, [8]int{0x2001, 0, 0, 0, 0, 0, 0, 1})
	if b2.Len() != 0 {
		t.Fatalf("expected no output for non-EUI-64 address, got %q", b2.String())
	}
}

// TestDescribeSpecialIPv6 documents the special/embedded-IPv4 group and its
// "handled" signal.
func TestDescribeSpecialIPv6(t *testing.T) {
	var b strings.Builder
	if !describeSpecialIPv6(&b, [8]int{0, 0, 0, 0, 0, 0xffff, 0x0102, 0x0304}, "::ffff:102:304") {
		t.Fatal("mapped IPv4 should be handled")
	}
	if !strings.Contains(b.String(), "Mapped IPv4 address") {
		t.Fatalf("missing mapped IPv4 detail: %q", b.String())
	}
	// A 2001:: address is not a "special" address; the group reports unhandled.
	var b2 strings.Builder
	if describeSpecialIPv6(&b2, [8]int{0x2001, 0, 0, 0, 0, 0, 0, 0}, "2001::") {
		t.Fatal("2001:: should not be handled by the special group")
	}
}

// TestDescribe2001Prefix documents the 2001:: sub-allocation classification.
func TestDescribe2001Prefix(t *testing.T) {
	var b strings.Builder
	describe2001Prefix(&b, [8]int{0x2001, 0xdb8, 0, 0, 0, 0, 0, 0})
	if !strings.Contains(b.String(), "documentation IPv6 address") {
		t.Fatalf("2001:db8:: not classified: %q", b.String())
	}
	// An unrecognised 2001 sub-prefix writes nothing.
	var b2 strings.Builder
	describe2001Prefix(&b2, [8]int{0x2001, 0x50, 0, 0, 0, 0, 0, 0})
	if b2.Len() != 0 {
		t.Fatalf("unexpected output for unknown 2001 sub-prefix: %q", b2.String())
	}
}

// TestIPv6ZeroRange documents the inclusive all-zero-hextets check.
func TestIPv6ZeroRange(t *testing.T) {
	ip := [8]int{0, 0, 0, 5, 0, 0, 0, 0}
	if !ipv6ZeroRange(ip, 0, 2) {
		t.Fatal("hextets 0-2 are zero")
	}
	if ipv6ZeroRange(ip, 0, 3) {
		t.Fatal("hextet 3 is non-zero")
	}
	if !ipv6ZeroRange(ip, 4, 7) {
		t.Fatal("hextets 4-7 are zero")
	}
}
