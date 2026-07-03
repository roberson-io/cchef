package ops

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(IPv6TransitionAddresses{})
}

// ipv6XOR flips the universal/local bit nibble for EUI-64 conversion (CyberChef's
// XOR lookup table).
var ipv6XOR = map[byte]byte{
	'0': '2', '1': '3', '2': '0', '3': '1', '4': '6', '5': '7', '6': '4', '7': '5',
	'8': 'a', '9': 'b', 'a': '8', 'b': '9', 'c': 'e', 'd': 'f', 'e': 'c', 'f': 'd',
}

var (
	ipv6TransIPv4Re = regexp.MustCompile(`^[0-9]{1,3}(?:\.[0-9]{1,3}){3}$`)
	ipv6TransRangeRe = regexp.MustCompile(`/24$`)
	ipv6TransMACRe   = regexp.MustCompile(`^([0-9a-fA-F]{2}:){5}[0-9a-fA-F]{2}$`)
	ipv6TransV6Re    = regexp.MustCompile(`^((?:[0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|(?:[0-9a-fA-F]{1,4}:){1,7}:|(?:[0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|(?:[0-9a-fA-F]{1,4}:){1,5}(?::[0-9a-fA-F]{1,4}){1,2}|(?:[0-9a-fA-F]{1,4}:){1,4}(?::[0-9a-fA-F]{1,4}){1,3}|(?:[0-9a-fA-F]{1,4}:){1,3}(?::[0-9a-fA-F]{1,4}){1,4}|(?:[0-9a-fA-F]{1,4}:){1,2}(?::[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:(?:(?::[0-9a-fA-F]{1,4}){1,6})|:(?:(?::[0-9a-fA-F]{1,4}){1,7}|:)|fe80:(?::[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|::(?:ffff(?::0{1,4}){0,1}:){0,1}(?:(?:25[0-5]|(?:2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(?:25[0-5]|(?:2[0-4]|1{0,1}[0-9]){0,1}[0-9])|(?:[0-9a-fA-F]{1,4}:){1,4}:(?:(?:25[0-5]|(?:2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(?:25[0-5]|(?:2[0-4]|1{0,1}[0-9]){0,1}[0-9]))$`)
	ipv6TransHextet1 = regexp.MustCompile(`:([0-9a-z]{1,4}):[0-9a-z]{1,4}$`)
	ipv6TransHextet2 = regexp.MustCompile(`:([0-9a-z]{1,4})$`)
)

// ipv6Hexify converts a decimal octet string to a 2-digit hex string.
func ipv6Hexify(octet string) string {
	n, _ := strconv.Atoi(octet)
	return fmt.Sprintf("%02x", n)
}

// ipv6Intify parses a hex string to an int.
func ipv6Intify(hex string) int {
	v, _ := strconv.ParseInt(hex, 16, 64)
	return int(v)
}

// IPv6TransitionAddresses converts between IPv4/MAC and IPv6 transition addresses.
type IPv6TransitionAddresses struct{}

// Meta returns the operation metadata.
func (IPv6TransitionAddresses) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "IPv6 Transition Addresses",
		Module:      "Default",
		Description: "Converts IPv4 addresses to their IPv6 transition addresses. IPv6 transition addresses can also be converted back into their original IPv4 address. MAC addresses can also be converted into the EUI-64 format, this can them be appended to your IPv6 /64 range to obtain a full /128 address.<br><br>Transition technologies enable translation between IPv4 and IPv6 addresses or tunneling to allow traffic to pass through the incompatible network, allowing the two standards to coexist.<br><br>Only /24 ranges and prefixes are currently supported.",
		InfoURL:     "https://wikipedia.org/wiki/IPv6_transition_mechanism",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (IPv6TransitionAddresses) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Ignore ranges", Type: core.ArgBoolean, Value: true},
		{Name: "Remove headers", Type: core.ArgBoolean, Value: false},
	}
}

// jsSliceNeg mimics JavaScript String.slice(start, end), including negative
// indices (counted from the end).
func jsSliceNeg(s string, start, end int) string {
	n := len(s)
	if start < 0 {
		if start += n; start < 0 {
			start = 0
		}
	} else if start > n {
		start = n
	}
	if end < 0 {
		if end += n; end < 0 {
			end = 0
		}
	} else if end > n {
		end = n
	}
	if end < start {
		end = start
	}
	return s[start:end]
}

// padStartZero left-pads s with '0' to at least length characters.
func padStartZero(s string, length int) string {
	if len(s) >= length {
		return s
	}
	return strings.Repeat("0", length-len(s)) + s
}

// ipv6IPTransition builds the four transition addresses for an IPv4 address or
// /24 range. Ported from IPv6TransitionAddresses.mjs ipTransition.
func ipv6IPTransition(input string, isRange, removeHeaders bool) string {
	hexip := strings.Split(input, ".")
	var b strings.Builder
	head := ipv6Hexify(hexip[0]) + ipv6Hexify(hexip[1]) + ":" + ipv6Hexify(hexip[2])

	// writeLine emits one transition line: prefix + head, then either the range
	// suffix or the final octet followed by the per-line host suffix.
	writeLine := func(label, mid, rangeSuffix, hostSuffix string) {
		if !removeHeaders {
			b.WriteString(label)
		}
		b.WriteString(mid + head)
		if isRange {
			b.WriteString(rangeSuffix)
		} else {
			b.WriteString(ipv6Hexify(hexip[3]) + hostSuffix)
		}
	}
	writeLine("6to4: ", "2002:", "00::/40\n", "::/48\n")
	writeLine("IPv4 Mapped: ", "::ffff:", "00/120\n", "\n")
	writeLine("IPv4 Translated: ", "::ffff:0:", "00/120\n", "\n")
	writeLine("Nat 64: ", "64:ff9b::", "00/120\n", "\n")
	return b.String()
}

// ipv6MACTransition converts a MAC address to its EUI-64 interface ID. Ported
// from IPv6TransitionAddresses.mjs macTransition.
func ipv6MACTransition(input string, removeHeaders bool) string {
	p := strings.Split(input, ":")
	mac := p[0] + p[1] + ":" + p[2] + "ff:fe" + p[3] + ":" + p[4] + p[5]
	var b strings.Builder
	if !removeHeaders {
		b.WriteString("EUI-64 Interface ID: ")
	}
	b.WriteByte(mac[0])
	b.WriteByte(ipv6XOR[mac[1]])
	b.WriteString(mac[2:])
	return b.String()
}

// ipv6UnTransition converts an IPv6 transition address back to its original IPv4
// or MAC address. Ported from IPv6TransitionAddresses.mjs unTransition.
func ipv6UnTransition(input string, removeHeaders bool) string {
	var b strings.Builder
	header := func(h string) {
		if !removeHeaders {
			b.WriteString(h)
		}
	}
	mappedPrefixes := []string{"::ffff:", "0000:0000:0000:0000:0000:ffff:", "::ffff:0000:",
		"0000:0000:0000:0000:ffff:0000:", "64:ff9b::", "0064:ff9b:0000:0000:0000:0000:"}
	hasMappedPrefix := false
	for _, p := range mappedPrefixes {
		if strings.HasPrefix(input, p) {
			hasMappedPrefix = true
			break
		}
	}

	switch {
	case strings.HasPrefix(input, "2002:"):
		header("IPv4: ")
		fmt.Fprintf(&b, "%d.%d.%d.%d\n",
			ipv6Intify(jsSliceNeg(input, 5, 7)), ipv6Intify(jsSliceNeg(input, 7, 9)),
			ipv6Intify(jsSliceNeg(input, 10, 12)), ipv6Intify(jsSliceNeg(input, 12, 14)))
	case hasMappedPrefix:
		m1 := ipv6TransHextet1.FindStringSubmatch(input)
		m2 := ipv6TransHextet2.FindStringSubmatch(input)
		if m1 == nil || m2 == nil {
			return b.String()
		}
		h := padStartZero(m1[1], 4) + padStartZero(m2[1], 4)
		header("IPv4: ")
		fmt.Fprintf(&b, "%d.%d.%d.%d\n",
			ipv6Intify(jsSliceNeg(h, -8, -7)+jsSliceNeg(h, -7, -6)),
			ipv6Intify(jsSliceNeg(h, -6, -5)+jsSliceNeg(h, -5, -4)),
			ipv6Intify(jsSliceNeg(h, -4, -3)+jsSliceNeg(h, -3, -2)),
			ipv6Intify(jsSliceNeg(h, -2, -1)+jsSliceNeg(h, len(h)-1, len(h))))
	case strings.ToUpper(jsSliceNeg(input, -12, -7)) == "FF:FE":
		header("Mac Address: ")
		mac := strings.ToUpper(jsSliceNeg(input, -19, -17) + ":" + jsSliceNeg(input, -17, -15) + ":" +
			jsSliceNeg(input, -14, -12) + ":" + jsSliceNeg(input, -7, -5) + ":" +
			jsSliceNeg(input, -4, -2) + ":" + jsSliceNeg(input, len(input)-2, len(input)))
		key := mac[1]
		if key >= 'A' && key <= 'F' { // the XOR table is keyed by lowercase hex
			key += 'a' - 'A'
		}
		b.WriteByte(mac[0])
		b.WriteByte(ipv6XOR[key])
		b.WriteString(mac[2:])
		b.WriteString("\n")
	}
	return b.String()
}

// Run performs the transition. Ported from CyberChef IPv6TransitionAddresses.mjs.
func (IPv6TransitionAddresses) Run(in *core.Dish, args []any) (*core.Dish, error) {
	ignoreRanges := args[0].(bool)
	removeHeaders := args[1].(bool)

	output := ""
	for _, line := range strings.Split(in.String(), "\n") {
		if line == "" {
			continue
		}
		if ignoreRanges && strings.Contains(line, "/") {
			continue
		}
		switch {
		case ipv6TransIPv4Re.MatchString(line):
			output += ipv6IPTransition(line, false, removeHeaders)
		case ipv6TransRangeRe.MatchString(line):
			output += ipv6IPTransition(line, true, removeHeaders)
		case ipv6TransMACRe.MatchString(line):
			output += ipv6MACTransition(line, removeHeaders)
		case ipv6TransV6Re.MatchString(line):
			output += ipv6UnTransition(line, removeHeaders)
		default:
			output = "Enter compressed or expanded IPv6 address, IPv4 address or MAC Address."
		}
	}
	return core.NewDish([]byte(output), core.TypeString), nil
}
