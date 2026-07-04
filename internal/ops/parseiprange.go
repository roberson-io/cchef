package ops

import (
	"fmt"
	"math"
	"math/big"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/dlclark/regexp2"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(ParseIPRange{})
}

const largeRangeError = "The specified range contains more than 65,536 addresses. Running this query could crash your browser. If you want to run it, select the \"Allow large queries\" option. You are advised to turn off \"Auto Bake\" whilst editing large ranges."

var (
	ipRangeV4CidrRe  = regexp.MustCompile(`^\s*((?:\d{1,3}\.){3}\d{1,3})/(\d\d?)\s*$`)
	ipRangeV4RangeRe = regexp.MustCompile(`^\s*((?:\d{1,3}\.){3}\d{1,3})\s*-\s*((?:\d{1,3}\.){3}\d{1,3})\s*$`)
	ipRangeV4ListRe  = regexp.MustCompile(`^\s*(((?:\d{1,3}\.){3}\d{1,3})(/(\d\d?))?(\n|$)(\n*))+\s*$`)

	ipRangeV6CidrRe  = regexp2.MustCompile(`^\s*(((?=.*::)(?!.*::.+::)(::)?([\dA-F]{1,4}:(:|\b)|){5}|([\dA-F]{1,4}:){6})((([\dA-F]{1,4}((?!\4)::|:\b|(?![\dA-F])))|(?!\3\4)){2}|(((2[0-4]|1\d|[1-9])?\d|25[0-5])\.?\b){4}))/(\d\d?\d?)\s*$`, regexp2.IgnoreCase)
	ipRangeV6RangeRe = regexp2.MustCompile(`^\s*(((?=.*::)(?!.*::[^-]+::)(::)?([\dA-F]{1,4}:(:|\b)|){5}|([\dA-F]{1,4}:){6})((([\dA-F]{1,4}((?!\4)::|:\b|(?![\dA-F])))|(?!\3\4)){2}|(((2[0-4]|1\d|[1-9])?\d|25[0-5])\.?\b){4}))\s*-\s*(((?=.*::)(?!.*::.+::)(::)?([\dA-F]{1,4}:(:|\b)|){5}|([\dA-F]{1,4}:){6})((([\dA-F]{1,4}((?!\17)::|:\b|(?![\dA-F])))|(?!\16\17)){2}|(((2[0-4]|1\d|[1-9])?\d|25[0-5])\.?\b){4}))\s*$`, regexp2.IgnoreCase)
	ipRangeV6ListRe  = regexp2.MustCompile(`^\s*((((?=.*::)(?!.*::.+::)(::)?([\dA-F]{1,4}:(:|\b)|){5}|([\dA-F]{1,4}:){6})((([\dA-F]{1,4}((?!\4)::|:\b|(?![\dA-F])))|(?!\3\4)){2}|(((2[0-4]|1\d|[1-9])?\d|25[0-5])\.?\b){4}))(/(\d\d?\d?))?(\n|$)(\n*))+\s*$`, regexp2.IgnoreCase)
)

// generateIpv4Range lists every IPv4 address from ip1 to ip2 inclusive.
func generateIpv4Range(ip1, ip2 uint32) []string {
	if ip2 < ip1 {
		return []string{"Second IP address smaller than first."}
	}
	var r []string
	for ip := ip1; ; ip++ {
		r = append(r, ipv4ToStr(ip))
		if ip == ip2 {
			break
		}
	}
	return r
}

// jsFloatString renders f the way JavaScript's Number.prototype.toString does:
// plain digits for integers below 1e21, exponential notation otherwise.
func jsFloatString(f float64) string {
	if f == math.Trunc(f) && math.Abs(f) < 1e21 {
		return strconv.FormatFloat(f, 'f', -1, 64)
	}
	return strconv.FormatFloat(f, 'g', -1, 64)
}

// ipv6TotalAddresses reproduces CyberChef's (float-precision) count of addresses
// between two IPv6 endpoints.
func ipv6TotalAddresses(ip1, ip2 [8]int) string {
	var sb strings.Builder
	for i := 0; i < 8; i++ {
		if diff := ip2[i] - ip1[i]; diff != 0 {
			sb.WriteString(strconv.FormatInt(int64(diff), 2))
		}
	}
	bin := sb.String()
	if bin == "" {
		return "1"
	}
	bi, _ := new(big.Int).SetString(bin, 2)
	fv, _ := new(big.Float).SetInt(bi).Float64()
	return jsFloatString(fv + 1)
}

// ipv4CidrRange describes and (optionally) enumerates an IPv4 CIDR block.
func ipv4CidrRange(ipStr string, cidr int, netInfo, enumerate, allowLarge bool) (string, error) {
	network, err := strToIpv4(ipStr)
	if err != nil {
		return "", err
	}
	if cidr < 0 || cidr > 31 {
		return "", fmt.Errorf("the IPv4 CIDR must be less than 32")
	}
	mask := ^(uint32(0xFFFFFFFF) >> uint(cidr))
	ip1 := network & mask
	ip2 := ip1 | ^mask

	var out strings.Builder
	if netInfo {
		fmt.Fprintf(&out, "Network: %s\nCIDR: %d\nMask: %s\nRange: %s - %s\nTotal addresses in range: %d\n\n",
			ipv4ToStr(network), cidr, ipv4ToStr(mask), ipv4ToStr(ip1), ipv4ToStr(ip2), (ip2-ip1)+1)
	}
	if enumerate {
		if cidr >= 16 || allowLarge {
			out.WriteString(strings.Join(generateIpv4Range(ip1, ip2), "\n"))
		} else {
			out.WriteString(largeRangeError)
		}
	}
	return out.String(), nil
}

// ipv4HyphenatedRange describes the minimum subnet covering an IPv4 range and
// enumerates it.
func ipv4HyphenatedRange(rangeStr string, netInfo, enumerate, allowLarge bool) (string, error) {
	parts := strings.SplitN(rangeStr, "-", 2)
	ip1, err := strToIpv4(strings.TrimSpace(parts[0]))
	if err != nil {
		return "", err
	}
	ip2, err := strToIpv4(strings.TrimSpace(parts[1]))
	if err != nil {
		return "", err
	}

	diff := ip1 ^ ip2
	cidr := 32
	var mask uint32
	for diff != 0 {
		diff >>= 1
		cidr--
		mask = (mask << 1) | 1
	}
	mask = ^mask
	network := ip1 & mask
	subIP1 := network & mask
	subIP2 := subIP1 | ^mask

	var out strings.Builder
	if netInfo {
		fmt.Fprintf(&out, "Minimum subnet required to hold this range:\n\tNetwork: %s\n\tCIDR: %d\n\tMask: %s\n\tSubnet range: %s - %s\n\tTotal addresses in subnet: %d\n\nRange: %s - %s\nTotal addresses in range: %d\n\n",
			ipv4ToStr(network), cidr, ipv4ToStr(mask), ipv4ToStr(subIP1), ipv4ToStr(subIP2), (subIP2-subIP1)+1,
			ipv4ToStr(ip1), ipv4ToStr(ip2), (ip2-ip1)+1)
	}
	if enumerate {
		if (ip2-ip1) <= 65536 || allowLarge {
			out.WriteString(strings.Join(generateIpv4Range(ip1, ip2), "\n"))
		} else {
			out.WriteString(largeRangeError)
		}
	}
	return out.String(), nil
}

// ipv4ListedRange collapses a list of IPs/CIDRs to their min–max range.
func ipv4ListedRange(listStr string, netInfo, enumerate, allowLarge bool) (string, error) {
	var vals []uint32
	for _, line := range strings.Split(listStr, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.Contains(line, "/") {
			p := strings.SplitN(line, "/", 2)
			network, err := strToIpv4(p[0])
			if err != nil {
				return "", err
			}
			cidr, _ := strconv.Atoi(p[1])
			if cidr < 0 || cidr > 31 {
				return "", fmt.Errorf("the IPv4 CIDR must be less than 32")
			}
			mask := ^(uint32(0xFFFFFFFF) >> uint(cidr))
			vals = append(vals, network&mask, (network&mask)|^mask)
		} else {
			v, err := strToIpv4(line)
			if err != nil {
				return "", err
			}
			vals = append(vals, v)
		}
	}
	sort.Slice(vals, func(i, j int) bool { return vals[i] < vals[j] })
	return ipv4HyphenatedRange(ipv4ToStr(vals[0])+" - "+ipv4ToStr(vals[len(vals)-1]), netInfo, enumerate, allowLarge)
}

// ipv6CidrRange describes an IPv6 CIDR block.
func ipv6CidrRange(ipStr string, cidr int, netInfo bool) (string, error) {
	network, err := strToIpv6(ipStr)
	if err != nil {
		return "", err
	}
	if cidr < 0 || cidr > 127 {
		return "", fmt.Errorf("the IPv6 CIDR must be less than 128")
	}
	mask := genIpv6Mask(cidr)
	var maskArr, ip1, ip2 [8]int
	for i := 0; i < 8; i++ {
		maskArr[i] = int(mask[i])
		ip1[i] = network[i] & int(mask[i])
		ip2[i] = ip1[i] | (int(^mask[i]) & 0x0000FFFF)
	}
	var out strings.Builder
	if netInfo {
		fmt.Fprintf(&out, "Network: %s\nShorthand: %s\nCIDR: %d\nMask: %s\nRange: %s - %s\nTotal addresses in range: %s\n\n",
			ipv6ToStr(network, false), ipv6ToStr(network, true), cidr, ipv6ToStr(maskArr, false),
			ipv6ToStr(ip1, false), ipv6ToStr(ip2, false), ipv6TotalAddresses(ip1, ip2))
	}
	return out.String(), nil
}

// ipv6HyphenatedRange describes an IPv6 range.
func ipv6HyphenatedRange(rangeStr string, netInfo bool) (string, error) {
	parts := strings.SplitN(rangeStr, "-", 2)
	ip1, err := strToIpv6(strings.TrimSpace(parts[0]))
	if err != nil {
		return "", err
	}
	ip2, err := strToIpv6(strings.TrimSpace(parts[1]))
	if err != nil {
		return "", err
	}
	var out strings.Builder
	if netInfo {
		fmt.Fprintf(&out, "Range: %s - %s\nShorthand range: %s - %s\nTotal addresses in range: %s\n\n",
			ipv6ToStr(ip1, false), ipv6ToStr(ip2, false), ipv6ToStr(ip1, true), ipv6ToStr(ip2, true),
			ipv6TotalAddresses(ip1, ip2))
	}
	return out.String(), nil
}

// ipv6ListedRange collapses a list of IPv6 addresses/CIDRs to their min–max range.
func ipv6ListedRange(listStr string, netInfo bool) (string, error) {
	type v6 struct {
		s string
		a [8]int
	}
	var list []v6
	for _, line := range strings.Split(listStr, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.Contains(line, "/") {
			p := strings.SplitN(line, "/", 2)
			network, err := strToIpv6(p[0])
			if err != nil {
				return "", err
			}
			cidr, _ := strconv.Atoi(p[1])
			if cidr < 0 || cidr > 127 {
				return "", fmt.Errorf("the IPv6 CIDR must be less than 128")
			}
			mask := genIpv6Mask(cidr)
			var lo, hi [8]int
			for j := 0; j < 8; j++ {
				lo[j] = network[j] & int(mask[j])
				hi[j] = lo[j] | (int(^mask[j]) & 0x0000FFFF)
			}
			list = append(list, v6{ipv6ToStr(lo, true), lo}, v6{ipv6ToStr(hi, true), hi})
		} else {
			a, err := strToIpv6(line)
			if err != nil {
				return "", err
			}
			list = append(list, v6{line, a})
		}
	}
	sort.SliceStable(list, func(i, j int) bool {
		for k := 0; k < 8; k++ {
			if list[i].a[k] != list[j].a[k] {
				return list[i].a[k] < list[j].a[k]
			}
		}
		return false
	})
	return ipv6HyphenatedRange(list[0].s+" - "+list[len(list)-1].s, netInfo)
}

// regexp2Matches reports whether re matches s.
func regexp2Matches(re *regexp2.Regexp, s string) bool {
	m, _ := re.FindStringMatch(s)
	return m != nil
}

// ParseIPRange enumerates and describes an IPv4/IPv6 CIDR, hyphenated range, or
// list of addresses.
type ParseIPRange struct{}

// Meta returns the operation metadata.
func (ParseIPRange) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Parse IP range",
		Module:      "Default",
		Description: "Given a CIDR range (e.g. 10.0.0.0/24), a hyphenated range (e.g. 10.0.0.0 - 10.0.1.0), or a list of IPs and CIDR ranges, this operation provides network information and enumerates all IP addresses in the range.<br><br>IPv6 is supported but will not be enumerated.",
		InfoURL:     "https://wikipedia.org/wiki/Subnetwork",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ParseIPRange) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Include network info", Type: core.ArgBoolean, Value: true},
		{Name: "Enumerate IP addresses", Type: core.ArgBoolean, Value: true},
		{Name: "Allow large queries", Type: core.ArgBoolean, Value: false},
	}
}

// Run parses the range. Ported from CyberChef ParseIPRange.mjs.
func (ParseIPRange) Run(in *core.Dish, args []any) (*core.Dish, error) {
	netInfo := args[0].(bool)
	enumerate := args[1].(bool)
	allowLarge := args[2].(bool)
	input := in.String()

	var out string
	var err error
	switch {
	case ipRangeV4CidrRe.MatchString(input):
		m := ipRangeV4CidrRe.FindStringSubmatch(input)
		cidr, _ := strconv.Atoi(m[2])
		out, err = ipv4CidrRange(m[1], cidr, netInfo, enumerate, allowLarge)
	case ipRangeV4RangeRe.MatchString(input):
		m := ipRangeV4RangeRe.FindStringSubmatch(input)
		out, err = ipv4HyphenatedRange(m[1]+"-"+m[2], netInfo, enumerate, allowLarge)
	case ipRangeV4ListRe.MatchString(input):
		out, err = ipv4ListedRange(strings.TrimSpace(input), netInfo, enumerate, allowLarge)
	case regexp2Matches(ipRangeV6CidrRe, input):
		trimmed := strings.TrimSpace(input)
		idx := strings.LastIndex(trimmed, "/")
		cidr, _ := strconv.Atoi(trimmed[idx+1:])
		out, err = ipv6CidrRange(trimmed[:idx], cidr, netInfo)
	case regexp2Matches(ipRangeV6RangeRe, input):
		out, err = ipv6HyphenatedRange(strings.TrimSpace(input), netInfo)
	case regexp2Matches(ipRangeV6ListRe, input):
		out, err = ipv6ListedRange(strings.TrimSpace(input), netInfo)
	default:
		return nil, fmt.Errorf("invalid input: enter either a CIDR range (e.g. 10.0.0.0/24) or a hyphenated range (e.g. 10.0.0.0 - 10.0.1.0); IPv6 also supported")
	}
	if err != nil {
		return nil, err
	}
	return core.NewDish([]byte(out), core.TypeString), nil
}
