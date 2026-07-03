package ops

import (
	"fmt"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(StripIPv4Header{})
	core.Register(StripTCPHeader{})
	core.Register(StripUDPHeader{})
}

// StripIPv4Header removes the IPv4 header, leaving the payload.
type StripIPv4Header struct{}

// Meta returns the operation metadata.
func (StripIPv4Header) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Strip IPv4 header",
		Module:      "Default",
		Description: "Strips the IPv4 header from an IPv4 packet, outputting the payload.",
		InfoURL:     "https://wikipedia.org/wiki/IPv4",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeArrayBuffer,
	}
}

// Args returns the argument definitions.
func (StripIPv4Header) Args() []core.ArgDef { return nil }

// Run strips the header. Ported from CyberChef StripIPv4Header.mjs.
func (StripIPv4Header) Run(in *core.Dish, args []any) (*core.Dish, error) {
	data := in.Bytes()
	if len(data) < 20 {
		return nil, fmt.Errorf("input length is less than minimum IPv4 header length")
	}
	dataOffset := int(data[0]&0x0f) * 4
	if len(data) < dataOffset {
		return nil, fmt.Errorf("input length is less than IHL")
	}
	return core.NewDish(data[dataOffset:], core.TypeArrayBuffer), nil
}

// StripTCPHeader removes the TCP header, leaving the payload.
type StripTCPHeader struct{}

// Meta returns the operation metadata.
func (StripTCPHeader) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Strip TCP header",
		Module:      "Default",
		Description: "Strips the TCP header from a TCP segment, outputting the payload.",
		InfoURL:     "https://wikipedia.org/wiki/Transmission_Control_Protocol",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeArrayBuffer,
	}
}

// Args returns the argument definitions.
func (StripTCPHeader) Args() []core.ArgDef { return nil }

// Run strips the header. Ported from CyberChef StripTCPHeader.mjs.
func (StripTCPHeader) Run(in *core.Dish, args []any) (*core.Dish, error) {
	data := in.Bytes()
	if len(data) < 20 {
		return nil, fmt.Errorf("need at least 20 bytes for a TCP header")
	}
	// The data offset is the high nibble of byte 12, counted in 32-bit words.
	dataOffset := int(data[12]>>4) * 4
	if len(data) < dataOffset {
		return nil, fmt.Errorf("input length is less than data offset")
	}
	return core.NewDish(data[dataOffset:], core.TypeArrayBuffer), nil
}

// StripUDPHeader removes the 8-byte UDP header, leaving the payload.
type StripUDPHeader struct{}

// Meta returns the operation metadata.
func (StripUDPHeader) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Strip UDP header",
		Module:      "Default",
		Description: "Strips the UDP header from a UDP datagram, outputting the payload.",
		InfoURL:     "https://wikipedia.org/wiki/User_Datagram_Protocol",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeArrayBuffer,
	}
}

// Args returns the argument definitions.
func (StripUDPHeader) Args() []core.ArgDef { return nil }

// Run strips the header. Ported from CyberChef StripUDPHeader.mjs.
func (StripUDPHeader) Run(in *core.Dish, args []any) (*core.Dish, error) {
	data := in.Bytes()
	if len(data) < 8 {
		return nil, fmt.Errorf("need 8 bytes for a UDP header")
	}
	return core.NewDish(data[8:], core.TypeArrayBuffer), nil
}
