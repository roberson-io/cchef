package ops

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(ParseIPv4Header{})
}

// utilsHex renders c as hex, left-padded with zeros to at least length digits
// (CyberChef Utils.hex).
func utilsHex(c, length int) string {
	s := strconv.FormatInt(int64(c), 16)
	for len(s) < length {
		s = "0" + s
	}
	return s
}

// tcpipChecksum computes the 16-bit ones-complement header checksum, rendered as
// hex. Ported from CyberChef TCPIPChecksum.
func tcpipChecksum(data []byte) string {
	csum := 0
	for i, b := range data {
		if i%2 == 0 {
			csum += int(b) << 8
		} else {
			csum += int(b)
		}
	}
	csum = (csum >> 16) + (csum & 0xffff)
	return utilsHex(0xffff-csum, 2)
}

// escapeHTMLChars escapes HTML-significant characters (CyberChef Utils.escapeHtml).
var escapeHTMLChars = strings.NewReplacer(
	"&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;",
	"'", "&#x27;", "`", "&#x60;", "\x00", "")

// byteArrayToChars maps each byte to a code point (Latin1), matching CyberChef's
// Utils.byteArrayToChars.
func byteArrayToChars(b []byte) string {
	var sb strings.Builder
	for _, by := range b {
		sb.WriteRune(rune(by))
	}
	return sb.String()
}

// byteSliceFrom / byteSliceRange clamp to bounds, mirroring JS Array.slice.
func byteSliceFrom(b []byte, start int) []byte {
	if start < 0 {
		start = 0
	}
	if start > len(b) {
		start = len(b)
	}
	return b[start:]
}

func byteSliceRange(b []byte, start, end int) []byte {
	if start < 0 {
		start = 0
	}
	if start > len(b) {
		start = len(b)
	}
	if end > len(b) {
		end = len(b)
	}
	if end < start {
		end = start
	}
	return b[start:end]
}

// ParseIPv4Header parses an IPv4 header, rendering it as a table or extracting
// the payload.
type ParseIPv4Header struct{}

// Meta returns the operation metadata.
func (ParseIPv4Header) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Parse IPv4 header",
		Module:      "Default",
		Description: "Given an IPv4 header, this operations parses and displays each field in an easily readable format.",
		InfoURL:     "https://wikipedia.org/wiki/IPv4#Header",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ParseIPv4Header) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Input format", Type: core.ArgOption, Value: []string{"Hex", "Raw"}},
		{Name: "Output format", Type: core.ArgOption, Value: []string{"Table", "Data (hex)", "Data (raw)"}},
	}
}

// Run parses the header. Ported from CyberChef ParseIPv4Header.mjs.
func (ParseIPv4Header) Run(in *core.Dish, args []any) (*core.Dish, error) {
	format := args[0].(string)
	outputFormat := args[1].(string)

	var input []byte
	if format == "Hex" {
		input = hexToBytes(in.String())
	} else {
		input = []byte(in.String()) // Raw
	}

	// A safe byte accessor: out-of-bounds reads are 0, matching JS's undefined
	// coercing to 0 in bitwise expressions.
	b := func(i int) int {
		if i >= 0 && i < len(input) {
			return int(input[i])
		}
		return 0
	}

	ihl := b(0) & 0x0f
	data := byteSliceFrom(input, ihl*4)

	switch outputFormat {
	case "Data (hex)":
		return core.NewDish([]byte(toHexSpace(data)), core.TypeString), nil
	case "Data (raw)":
		return core.NewDish([]byte(escapeHTMLChars.Replace(byteArrayToChars(data))), core.TypeString), nil
	}

	// Table.
	dscp := (b(1) >> 2) & 0x3f
	ecn := b(1) & 0x03
	length := b(2)<<8 | b(3)
	identification := b(4)<<8 | b(5)
	flags := (b(6) >> 5) & 0x07
	fragOffset := (b(6)&0x1f)<<8 | b(7)
	ttl := b(8)
	protocol := b(9)
	checksum := b(10)<<8 | b(11)
	srcIP := uint32(b(12))<<24 | uint32(b(13))<<16 | uint32(b(14))<<8 | uint32(b(15))
	dstIP := uint32(b(16))<<24 | uint32(b(17))<<16 | uint32(b(18))<<8 | uint32(b(19))
	version := (b(0) >> 4) & 0x0f

	versionStr := strconv.Itoa(version)
	if version != 4 {
		versionStr += " (Error: for IPv4 headers, this should always be set to 4)"
	}
	ihlStr := strconv.Itoa(ihl)
	if ihl < 5 {
		ihlStr += " (Error: this should always be at least 5)"
	}

	checksumHeader := make([]byte, 0, 20)
	checksumHeader = append(checksumHeader, byteSliceRange(input, 0, 10)...)
	checksumHeader = append(checksumHeader, 0, 0)
	checksumHeader = append(checksumHeader, byteSliceRange(input, 12, 20)...)
	correct := tcpipChecksum(checksumHeader)
	given := utilsHex(checksum, 2)
	checksumResult := given + " (correct)"
	if correct != given {
		checksumResult = given + " (incorrect, should be " + correct + ")"
	}

	protoInfo := protocolLookup[protocol]

	var out strings.Builder
	out.WriteString("<table class='table table-hover table-sm table-bordered table-nonfluid'><tr><th>Field</th><th>Value</th></tr>\n")
	fmt.Fprintf(&out, "<tr><td>Version</td><td>%s</td></tr>\n", versionStr)
	fmt.Fprintf(&out, "<tr><td>Internet Header Length (IHL)</td><td>%s (%d bytes)</td></tr>\n", ihlStr, ihl*4)
	fmt.Fprintf(&out, "<tr><td>Differentiated Services Code Point (DSCP)</td><td>%d</td></tr>\n", dscp)
	fmt.Fprintf(&out, "<tr><td>Explicit Congestion Notification (ECN)</td><td>%d</td></tr>\n", ecn)
	fmt.Fprintf(&out, "<tr><td>Total length</td><td>%d bytes\n  IP header: %d bytes\n  Data: %d bytes</td></tr>\n", length, ihl*4, length-ihl*4)
	fmt.Fprintf(&out, "<tr><td>Identification</td><td>0x%s (%d)</td></tr>\n", utilsHex(identification, 2), identification)
	fmt.Fprintf(&out, "<tr><td>Flags</td><td>0x%s\n  Reserved bit:%d (must be 0)\n  Don't fragment:%d\n  More fragments:%d</td></tr>\n", utilsHex(flags, 2), flags>>2, flags>>1&1, flags&1)
	fmt.Fprintf(&out, "<tr><td>Fragment offset</td><td>%d</td></tr>\n", fragOffset)
	fmt.Fprintf(&out, "<tr><td>Time-To-Live</td><td>%d</td></tr>\n", ttl)
	fmt.Fprintf(&out, "<tr><td>Protocol</td><td>%d, %s (%s)</td></tr>\n", protocol, protoInfo.protocol, protoInfo.keyword)
	fmt.Fprintf(&out, "<tr><td>Header checksum</td><td>%s</td></tr>\n", checksumResult)
	fmt.Fprintf(&out, "<tr><td>Source IP address</td><td>%s</td></tr>\n", ipv4ToStr(srcIP))
	fmt.Fprintf(&out, "<tr><td>Destination IP address</td><td>%s</td></tr>\n", ipv4ToStr(dstIP))
	fmt.Fprintf(&out, "<tr><td>Data (hex)</td><td>%s</td></tr>", toHexSpace(data))
	if ihl > 5 {
		options := byteSliceRange(input, 20, ihl*4)
		fmt.Fprintf(&out, "<tr><td>Options</td><td>%s</td></tr>", toHexSpace(options))
	}
	out.WriteString("</table>")
	return core.NewDish([]byte(out.String()), core.TypeString), nil
}
