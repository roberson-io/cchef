package ops

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(ParseIPv6Address{})
}

// utilsBin renders n as binary, zero-padded to length (CyberChef Utils.bin).
func utilsBin(n, length int) string { return fmt.Sprintf("%0*b", length, n) }

// ipv6MulticastScope returns the scope description for a multicast prefix.
func ipv6MulticastScope(hextet int) string {
	switch hextet {
	case 0xff01:
		return "\n\nReserved Multicast Block for Interface Local Scope"
	case 0xff02:
		return "\n\nReserved Multicast Block for Link Local Scope"
	case 0xff03:
		return "\n\nReserved Multicast Block for Realm Local Scope"
	case 0xff04:
		return "\n\nReserved Multicast Block for Admin Local Scope"
	case 0xff05:
		return "\n\nReserved Multicast Block for Site Local Scope"
	case 0xff08:
		return "\n\nReserved Multicast Block for Organisation Local Scope"
	case 0xff0e:
		return "\n\nReserved Multicast Block for Global Scope"
	}
	return ""
}

// ipv6MulticastAddress returns the well-known address description for a multicast
// address's low hextets.
func ipv6MulticastAddress(ipv6 [8]int) string {
	if ipv6[6] == 1 {
		switch ipv6[7] {
		case 2:
			return "\nReserved Multicast Address for 'All DHCP Servers and Relay Agents (defined in RFC3315)'"
		case 3:
			return "\nReserved Multicast Address for 'All LLMNR Hosts (defined in RFC4795)'"
		}
		return ""
	}
	m := map[int]string{
		1: "All nodes", 2: "All routers", 5: "OSPFv3 - All OSPF routers",
		6: "OSPFv3 - All Designated Routers", 8: "IS-IS for IPv6 Routers",
		9: "RIP Routers", 0xa: "EIGRP Routers", 0xc: "Simple Service Discovery Protocol",
		0xd: "PIM Routers", 0x16: "MLDv2 Reports (defined in RFC3810)",
		0x6b: "Precision Time Protocol v2 Peer Delay Measurement Messages",
		0xfb: "Multicast DNS", 0x101: "Network Time Protocol",
		0x108: "Network Information Service", 0x114: "Experiments",
		0x181: "Precision Time Protocol v2 Messages (exc. Peer Delay)",
	}
	if s, ok := m[ipv6[7]]; ok {
		return "\nReserved Multicast Address for '" + s + "'"
	}
	return ""
}

// ParseIPv6Address parses an IPv6 address and reports its type and details.
type ParseIPv6Address struct{}

// Meta returns the operation metadata.
func (ParseIPv6Address) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Parse IPv6 address",
		Module:      "Default",
		Description: "Displays the longhand and shorthand versions of a valid IPv6 address. Also displays the type of address and any embedded IPv4 or MAC addresses where relevant.",
		InfoURL:     "https://wikipedia.org/wiki/IPv6_address",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ParseIPv6Address) Args() []core.ArgDef { return nil }

// Run parses the address. Ported from CyberChef ParseIPv6Address.mjs.
func (ParseIPv6Address) Run(in *core.Dish, args []any) (*core.Dish, error) {
	m, _ := groupIPv6Re.FindStringMatch(in.String())
	if m == nil {
		return nil, fmt.Errorf("invalid IPv6 address")
	}
	ipv6, err := strToIpv6(m.GroupByNumber(1).String())
	if err != nil {
		return nil, err
	}
	shorthand := ipv6ToStr(ipv6, true)

	var b strings.Builder
	fmt.Fprintf(&b, "Longhand:  %s\nShorthand: %s\n", ipv6ToStr(ipv6, false), shorthand)

	// mappedIPv4 packs the last two hextets into a 32-bit IPv4 value.
	mappedIPv4 := func() uint32 { return uint32(ipv6[6]<<16 + ipv6[7]) }

	switch {
	case shorthand == "::":
		b.WriteString("\nUnspecified address corresponding to 0.0.0.0/32 in IPv4.")
		b.WriteString("\nUnspecified address range: ::/128")
	case shorthand == "::1":
		b.WriteString("\nLoopback address to the local host corresponding to 127.0.0.1/8 in IPv4.")
		b.WriteString("\nLoopback addresses range: ::1/128")
	case ipv6[0] == 0 && ipv6[1] == 0 && ipv6[2] == 0 && ipv6[3] == 0 && ipv6[4] == 0 && ipv6[5] == 0xffff:
		b.WriteString("\nIPv4-mapped IPv6 address detected. IPv6 clients will be handled natively by default, and IPv4 clients appear as IPv6 clients at their IPv4-mapped IPv6 address.")
		fmt.Fprintf(&b, "\nMapped IPv4 address: %s", ipv4ToStr(mappedIPv4()))
		b.WriteString("\nIPv4-mapped IPv6 addresses range: ::ffff:0:0/96")
	case ipv6[0] == 0 && ipv6[1] == 0 && ipv6[2] == 0 && ipv6[3] == 0 && ipv6[4] == 0xffff && ipv6[5] == 0:
		b.WriteString("\nIPv4-translated address detected. Used by Stateless IP/ICMP Translation (SIIT). See RFCs 6145 and 6052 for more details.")
		fmt.Fprintf(&b, "\nTranslated IPv4 address: %s", ipv4ToStr(mappedIPv4()))
		b.WriteString("\nIPv4-translated addresses range: ::ffff:0:0:0/96")
	case ipv6[0] == 0x100:
		b.WriteString("\nDiscard prefix detected. This is used when forwarding traffic to a sinkhole router to mitigate the effects of a denial-of-service attack. See RFC 6666 for more details.")
		b.WriteString("\nDiscard range: 100::/64")
	case ipv6[0] == 0x64 && ipv6[1] == 0xff9b && ipv6[2] == 0 && ipv6[3] == 0 && ipv6[4] == 0 && ipv6[5] == 0:
		b.WriteString("\n'Well-Known' prefix for IPv4/IPv6 translation detected. See RFC 6052 for more details.")
		fmt.Fprintf(&b, "\nTranslated IPv4 address: %s", ipv4ToStr(mappedIPv4()))
		b.WriteString("\n'Well-Known' prefix range: 64:ff9b::/96")
	case ipv6[0] == 0x2001 && ipv6[1] == 0:
		writeTeredo(&b, ipv6)
	case ipv6[0] == 0x2001 && ipv6[1] == 0x2 && ipv6[2] == 0:
		b.WriteString("\nAssigned to the Benchmarking Methodology Working Group (BMWG) for benchmarking IPv6. Corresponds to 198.18.0.0/15 for benchmarking IPv4. See RFC 5180 for more details.")
		b.WriteString("\nBMWG range: 2001:2::/48")
	case ipv6[0] == 0x2001 && ipv6[1] >= 0x10 && ipv6[1] <= 0x1f:
		b.WriteString("\nDeprecated, previously ORCHIDv1 (Overlay Routable Cryptographic Hash Identifiers).\nORCHIDv1 range: 2001:10::/28\nORCHIDv2 now uses 2001:20::/28.")
	case ipv6[0] == 0x2001 && ipv6[1] >= 0x20 && ipv6[1] <= 0x2f:
		b.WriteString("\nORCHIDv2 (Overlay Routable Cryptographic Hash Identifiers).\nThese are non-routed IPv6 addresses used for Cryptographic Hash Identifiers.")
		b.WriteString("\nORCHIDv2 range: 2001:20::/28")
	case ipv6[0] == 0x2001 && ipv6[1] == 0xdb8:
		b.WriteString("\nThis is a documentation IPv6 address. This range should be used whenever an example IPv6 address is given or to model networking scenarios. Corresponds to 192.0.2.0/24, 198.51.100.0/24, and 203.0.113.0/24 in IPv4.")
		b.WriteString("\nDocumentation range: 2001:db8::/32")
	case ipv6[0] == 0x2002:
		b.WriteString("\n6to4 transition IPv6 address detected. See RFC 3056 for more details.\n6to4 prefix range: 2002::/16")
		interfaceIDStr := strconv.FormatInt(int64(ipv6[4]), 16) + strconv.FormatInt(int64(ipv6[5]), 16) +
			strconv.FormatInt(int64(ipv6[6]), 16) + strconv.FormatInt(int64(ipv6[7]), 16)
		interfaceID, _ := new(big.Int).SetString(interfaceIDStr, 16)
		fmt.Fprintf(&b, "\n\nEncapsulated IPv4 address: %s\nSLA ID: %d\nInterface ID (base 16): %s\nInterface ID (base 10): %s",
			ipv4ToStr(uint32(ipv6[1]<<16+ipv6[2])), ipv6[3], interfaceIDStr, interfaceID.String())
	case ipv6[0] >= 0xfc00 && ipv6[0] <= 0xfdff:
		b.WriteString("\nThis is a unique local address comparable to the IPv4 private addresses 10.0.0.0/8, 172.16.0.0/12 and 192.168.0.0/16. See RFC 4193 for more details.")
		b.WriteString("\nUnique local addresses range: fc00::/7")
	case ipv6[0] >= 0xfe80 && ipv6[0] <= 0xfebf:
		b.WriteString("\nThis is a link-local address comparable to the auto-configuration addresses 169.254.0.0/16 in IPv4.")
		b.WriteString("\nLink-local addresses range: fe80::/10")
	case ipv6[0] >= 0xff00:
		b.WriteString("\nThis is a reserved multicast address.")
		b.WriteString("\nMulticast addresses range: ff00::/8")
		b.WriteString(ipv6MulticastScope(ipv6[0]))
		b.WriteString(ipv6MulticastAddress(ipv6))
	}

	// Modified EUI-64 (FF:FE in the 12th and 13th octets).
	if (ipv6[5]&0xff) == 0xff && (ipv6[6]>>8) == 0xfe {
		b.WriteString("\n\nThis IPv6 address contains a modified EUI-64 address, identified by the presence of FF:FE in the 12th and 13th octets.")
		intIdent := utilsHex(ipv6[4]>>8, 2) + ":" + utilsHex(ipv6[4]&0xff, 2) + ":" +
			utilsHex(ipv6[5]>>8, 2) + ":" + utilsHex(ipv6[5]&0xff, 2) + ":" +
			utilsHex(ipv6[6]>>8, 2) + ":" + utilsHex(ipv6[6]&0xff, 2) + ":" +
			utilsHex(ipv6[7]>>8, 2) + ":" + utilsHex(ipv6[7]&0xff, 2)
		mac := utilsHex((ipv6[4]>>8)^2, 2) + ":" + utilsHex(ipv6[4]&0xff, 2) + ":" +
			utilsHex(ipv6[5]>>8, 2) + ":" + utilsHex(ipv6[6]&0xff, 2) + ":" +
			utilsHex(ipv6[7]>>8, 2) + ":" + utilsHex(ipv6[7]&0xff, 2)
		fmt.Fprintf(&b, "\nInterface identifier: %s\nMAC address:          %s", intIdent, mac)
	}
	return core.NewDish([]byte(b.String()), core.TypeString), nil
}

// writeTeredo appends the Teredo tunnelling analysis for a 2001:0::/32 address.
func writeTeredo(b *strings.Builder, ipv6 [8]int) {
	b.WriteString("\nTeredo tunneling IPv6 address detected\n")
	serverIPv4 := uint32(ipv6[2]<<16 + ipv6[3])
	udpPort := (^ipv6[5]) & 0xffff
	clientIPv4 := ^uint32(ipv6[6]<<16 + ipv6[7])
	flagCone := (ipv6[4] >> 15) & 1
	flagR := (ipv6[4] >> 14) & 1
	flagRandom1 := (ipv6[4] >> 10) & 15
	flagUg := (ipv6[4] >> 8) & 3
	flagRandom2 := ipv6[4] & 255

	fmt.Fprintf(b, "\nServer IPv4 address: %s\nClient IPv4 address: %s\nClient UDP port:     %d\nFlags:\n\tCone:    %d",
		ipv4ToStr(serverIPv4), ipv4ToStr(clientIPv4), udpPort, flagCone)
	if flagCone != 0 {
		b.WriteString(" (Client is behind a cone NAT)")
	} else {
		b.WriteString(" (Client is not behind a cone NAT)")
	}
	fmt.Fprintf(b, "\n\tR:       %d", flagR)
	if flagR != 0 {
		b.WriteString(" Error: This flag should be set to 0. See RFC 5991 and RFC 4380.")
	}
	fmt.Fprintf(b, "\n\tRandom1: %s\n\tUG:      %s", utilsBin(flagRandom1, 4), utilsBin(flagUg, 2))
	if flagUg != 0 {
		b.WriteString(" Error: This flag should be set to 00. See RFC 4380.")
	}
	fmt.Fprintf(b, "\n\tRandom2: %s", utilsBin(flagRandom2, 8))

	switch {
	case flagR == 0 && flagUg == 0 && flagRandom1 != 0 && flagRandom2 != 0:
		b.WriteString("\n\nThis is a valid Teredo address which complies with RFC 4380 and RFC 5991.")
	case flagR == 0 && flagUg == 0:
		b.WriteString("\n\nThis is a valid Teredo address which complies with RFC 4380, however it does not comply with RFC 5991 (Teredo Security Updates) as there are no randomised bits in the flag field.")
	default:
		b.WriteString("\n\nThis is an invalid Teredo address.")
	}
	b.WriteString("\n\nTeredo prefix range: 2001::/32")
}
