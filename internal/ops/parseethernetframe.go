package ops

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(ParseEthernetFrame{})
}

// toHexDelim renders bytes as two-digit lowercase hex joined by delim.
func toHexDelim(b []byte, delim string) string {
	parts := make([]string, len(b))
	for i, by := range b {
		parts[i] = fmt.Sprintf("%02x", by)
	}
	return strings.Join(parts, delim)
}

// ParseEthernetFrame parses an Ethernet II frame, including VLAN tags.
type ParseEthernetFrame struct{}

// Meta returns the operation metadata.
func (ParseEthernetFrame) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Parse Ethernet frame",
		Module:      "Default",
		Description: "Parses an Ethernet frame, displaying the source and destination MAC addresses, any VLAN tags, and the payload.",
		InfoURL:     "https://wikipedia.org/wiki/Ethernet_frame",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ParseEthernetFrame) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Input type", Type: core.ArgOption, Value: []string{"Raw", "Hex"}},
		{Name: "Return type", Type: core.ArgOption, Value: []string{"Text output", "Packet data", "Packet data (hex)"}},
	}
}

// Run parses the frame. Ported from CyberChef ParseEthernetFrame.mjs.
func (ParseEthernetFrame) Run(in *core.Dish, args []any) (*core.Dish, error) {
	format := args[0].(string)
	outputFormat := args[1].(string)

	var input []byte
	if format == "Hex" {
		input = hexToBytes(in.String())
	} else {
		input = []byte(in.String()) // Raw
	}

	destinationMac := byteSliceRange(input, 0, 6)
	sourceMac := byteSliceRange(input, 6, 12)

	offset := 12
	var vlans []int
	for offset < len(input) {
		et := byteSliceRange(input, offset, offset+2)
		offset += 2
		// 802.1q (0x8100) or 802.1ad (0x88a8) tagged frame.
		if len(et) == 2 && ((et[0] == 0x81 && et[1] == 0x00) || (et[0] == 0x88 && et[1] == 0xa8)) {
			tag := byteSliceRange(input, offset, offset+2)
			var t0, t1 int
			if len(tag) > 0 {
				t0 = int(tag[0])
			}
			if len(tag) > 1 {
				t1 = int(tag[1])
			}
			vlans = append(vlans, (t0&0x0f)<<8|t1)
			offset += 2
		} else {
			break
		}
	}
	packetData := byteSliceFrom(input, offset)

	switch outputFormat {
	case "Packet data":
		return core.NewDish([]byte(escapeHTMLChars.Replace(byteArrayToChars(packetData))), core.TypeString), nil
	case "Packet data (hex)":
		return core.NewDish([]byte(toHexSpace(packetData)), core.TypeString), nil
	default: // Text output
		var b strings.Builder
		fmt.Fprintf(&b, "Source MAC: %s\nDestination MAC: %s\n", toHexDelim(sourceMac, ":"), toHexDelim(destinationMac, ":"))
		if len(vlans) > 0 {
			strs := make([]string, len(vlans))
			for i, v := range vlans {
				strs[i] = strconv.Itoa(v)
			}
			fmt.Fprintf(&b, "VLAN: %s\n", strings.Join(strs, ", "))
		}
		fmt.Fprintf(&b, "Data:\n%s", toHexSpace(packetData))
		return core.NewDish([]byte(b.String()), core.TypeString), nil
	}
}
