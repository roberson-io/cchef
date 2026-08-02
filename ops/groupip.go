package ops

import (
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/dlclark/regexp2"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(GroupIPAddresses{})
}

// ipDelimOptions mirrors CyberChef's IP_DELIM_OPTIONS.
var ipDelimOptions = []string{"Line feed", "CRLF", "Space", "Comma", "Semi-colon"}

// groupIPv4Re matches a single IPv4 address (capture 1); groupIPv6Re matches a
// single IPv6 address (capture 1) and needs regexp2 for its lookahead/backrefs.
var (
	groupIPv4Re = regexp.MustCompile(`^\s*((?:\d{1,3}\.){3}\d{1,3})\s*$`)
	groupIPv6Re = regexp2.MustCompile(`^\s*(((?=.*::)(?!.*::.+::)(::)?([\dA-F]{1,4}:(:|\b)|){5}|([\dA-F]{1,4}:){6})((([\dA-F]{1,4}((?!\4)::|:\b|(?![\dA-F])))|(?!\3\4)){2}|(((2[0-4]|1\d|[1-9])?\d|25[0-5])\.?\b){4}))\s*$`, regexp2.IgnoreCase)
)

// strToIpv4 parses a dotted-decimal IPv4 address to a 32-bit value. Every caller
// passes a string already matched by a `(\d{1,3}\.){3}\d{1,3}` regex, so the
// block-count check never fails; the out-of-range check does fire (the regex
// admits octets up to 999).
func strToIpv4(ipStr string) (uint32, error) {
	blocks := strings.Split(ipStr, ".")
	if len(blocks) != 4 {
		return 0, fmt.Errorf("more than 4 blocks")
	}
	var v uint32
	for i := range 4 {
		n, err := strconv.Atoi(blocks[i])
		if err != nil || n < 0 || n > 255 {
			return 0, fmt.Errorf("block out of range")
		}
		v |= uint32(n) << (24 - 8*i)
	}
	return v, nil
}

// ipv4ToStr renders a 32-bit IPv4 value as a dotted-decimal string.
func ipv4ToStr(ip uint32) string {
	return fmt.Sprintf("%d.%d.%d.%d", (ip>>24)&255, (ip>>16)&255, (ip>>8)&255, ip&255)
}

// strToIpv6 parses an IPv6 address string into eight 16-bit blocks, expanding a
// "::" shorthand. Callers that feed it a string pre-validated by groupIPv6Re
// (Parse IPv6 address, Parse IP range) cannot trigger its error returns: the
// regex guarantees 3–8 blocks of at most 4 hex digits, so neither the
// block-count nor the out-of-range check can fire.
func strToIpv6(ipStr string) ([8]int, error) {
	var ipv6 [8]int
	blocks := strings.Split(ipStr, ":")
	if len(blocks) < 3 || len(blocks) > 8 {
		return ipv6, fmt.Errorf("badly formatted IPv6 address")
	}
	const nan = -1
	numBlocks := make([]int, len(blocks))
	for i, blk := range blocks {
		if blk == "" {
			numBlocks[i] = nan
			continue
		}
		n, err := strconv.ParseInt(blk, 16, 64)
		if err != nil {
			numBlocks[i] = nan
		} else if n < 0 || n > 65535 {
			return ipv6, fmt.Errorf("block out of range")
		} else {
			numBlocks[i] = int(n)
		}
	}

	j := 0
	for i := range 8 {
		if numBlocks[j] == nan {
			ipv6[i] = 0
			if i == 8-len(numBlocks[j:]) {
				j++
			}
		} else {
			ipv6[i] = numBlocks[j]
			j++
		}
	}
	return ipv6, nil
}

// ipv6ToStr renders eight 16-bit blocks as an IPv6 string, using "::" shorthand
// for the longest zero run when compact.
func ipv6ToStr(ipv6 [8]int, compact bool) string {
	var output strings.Builder
	if compact {
		start, end, s, e := -1, -1, 0, -1
		for i := range 8 {
			if ipv6[i] == 0 && e == i-1 {
				e = i
			} else if ipv6[i] == 0 {
				s, e = i, i
			}
			if e >= 0 && (e-s) > (end-start) {
				start, end = s, e
			}
		}
		for i := 0; i < 8; i++ {
			if i != start {
				fmt.Fprintf(&output, "%x:", ipv6[i])
			} else {
				output.WriteString(":")
				i = end
				if end == 7 {
					output.WriteString(":")
				}
			}
		}
		out := output.String()
		if strings.HasPrefix(out, ":") {
			out = ":" + out
		}
		return out[:len(out)-1]
	}
	for i := range 8 {
		fmt.Fprintf(&output, "%04x:", ipv6[i])
	}
	out := output.String()
	return out[:len(out)-1]
}

// genIpv6Mask builds an 8-block network mask for the given CIDR.
func genIpv6Mask(cidr int) [8]uint32 {
	var mask [8]uint32
	for i := range 8 {
		if cidr > (i+1)*16 {
			mask[i] = 0x0000FFFF
		} else {
			shift := max(cidr-i*16, 0)
			mask[i] = ^((uint32(0x0000FFFF) >> uint(shift)) | 0xFFFF0000)
		}
	}
	return mask
}

// GroupIPAddresses groups a list of IP addresses into their subnets.
type GroupIPAddresses struct{}

// Meta returns the operation metadata.
func (GroupIPAddresses) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Group IP addresses",
		Module:      "Default",
		Description: "Groups a list of IP addresses into subnets. Supports both IPv4 and IPv6 addresses.",
		InfoURL:     "https://wikipedia.org/wiki/Subnetwork",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (GroupIPAddresses) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Delimiter", Type: core.ArgOption, Value: ipDelimOptions},
		{Name: "Subnet (CIDR)", Type: core.ArgNumber, Integer: true, Value: 24},
		{Name: "Only show the subnets", Type: core.ArgBoolean, Value: false},
	}
}

// Run groups the addresses.
func (GroupIPAddresses) Run(in *core.Dish, args []any) (*core.Dish, error) {
	delim := charRep(args[0].(string))
	cidr := int(args[1].(float64))
	onlySubnets := args[2].(bool)

	if cidr < 0 || cidr >= ipv6Bits {
		return nil, fmt.Errorf("cidr must be less than %d for IPv4 or %d for IPv6", ipv4Bits, ipv6Bits)
	}

	var ipv4Mask uint32 = 0xFFFFFFFF
	if cidr < ipv4Bits {
		ipv4Mask = ^(uint32(0xFFFFFFFF) >> uint(cidr))
	}
	ipv6Mask := genIpv6Mask(cidr)

	groups, err := groupIPTokens(in.String(), delim, ipv4Mask, ipv6Mask)
	if err != nil {
		return nil, err
	}

	var out strings.Builder
	writeIPv4Networks(&out, groups.ipv4, cidr, onlySubnets)
	writeIPv6Networks(&out, groups.ipv6, groups.ipv6Order, cidr, onlySubnets)
	return core.NewDish([]byte(out.String()), core.TypeString), nil
}

// ipv4Bits and ipv6Bits are the address widths, used to validate the CIDR and
// decide the IPv4 mask.
const (
	ipv4Bits = 32
	ipv6Bits = 128
)

// ipGroups holds addresses grouped by network: IPv4 keyed by network integer,
// IPv6 keyed by network string (with first-seen order preserved).
type ipGroups struct {
	ipv4      map[uint32][]uint32
	ipv6      map[string][][8]int
	ipv6Order []string
}

// groupIPTokens splits the input and buckets each IPv4/IPv6 address under its
// masked network.
func groupIPTokens(input, delim string, ipv4Mask uint32, ipv6Mask [8]uint32) (ipGroups, error) {
	g := ipGroups{ipv4: map[uint32][]uint32{}, ipv6: map[string][][8]int{}}
	for token := range strings.SplitSeq(input, delim) {
		if m := groupIPv4Re.FindStringSubmatch(token); m != nil {
			ip, err := strToIpv4(m[1])
			if err != nil {
				return ipGroups{}, err
			}
			network := ip & ipv4Mask
			g.ipv4[network] = append(g.ipv4[network], ip)
		} else if m6, _ := groupIPv6Re.FindStringMatch(token); m6 != nil {
			ip, err := strToIpv6(m6.GroupByNumber(1).String())
			if err != nil {
				return ipGroups{}, err
			}
			var network [8]int
			for j := range 8 {
				network[j] = ip[j] & int(ipv6Mask[j])
			}
			networkStr := ipv6ToStr(network, true)
			if _, ok := g.ipv6[networkStr]; !ok {
				g.ipv6Order = append(g.ipv6Order, networkStr)
			}
			g.ipv6[networkStr] = append(g.ipv6[networkStr], ip)
		}
	}
	return g, nil
}

// writeIPv4Networks renders the IPv4 networks in ascending network order (JS
// integer-key iteration); each network's members sort lexicographically by
// decimal string (JS default Array.prototype.sort).
func writeIPv4Networks(out *strings.Builder, networks map[uint32][]uint32, cidr int, onlySubnets bool) {
	v4keys := make([]uint32, 0, len(networks))
	for k := range networks {
		v4keys = append(v4keys, k)
	}
	slices.Sort(v4keys)
	for _, network := range v4keys {
		ips := networks[network]
		sort.SliceStable(ips, func(i, j int) bool {
			return strconv.FormatUint(uint64(ips[i]), 10) < strconv.FormatUint(uint64(ips[j]), 10)
		})
		out.WriteString(ipv4ToStr(network) + "/" + strconv.Itoa(cidr) + "\n")
		if !onlySubnets {
			for _, ip := range ips {
				out.WriteString("  " + ipv4ToStr(ip) + "\n")
			}
			out.WriteString("\n")
		}
	}
}

// writeIPv6Networks renders the IPv6 networks in first-seen order; members stay
// unsorted (CyberChef leaves this as a TODO).
func writeIPv6Networks(out *strings.Builder, networks map[string][][8]int, order []string, cidr int, onlySubnets bool) {
	for _, networkStr := range order {
		out.WriteString(networkStr + "/" + strconv.Itoa(cidr) + "\n")
		if !onlySubnets {
			for _, ip := range networks[networkStr] {
				out.WriteString("  " + ipv6ToStr(ip, true) + "\n")
			}
			out.WriteString("\n")
		}
	}
}
