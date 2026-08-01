package ops

import (
	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(TCPIPChecksum{})
}

// TCPIPChecksum computes the 16-bit one's-complement Internet checksum (RFC 1071)
// over the raw input bytes. Ported from CyberChef's TCPIPChecksum; the core sum
// is the shared tcpipChecksum helper (also used by the IPv4 header parser).
type TCPIPChecksum struct{}

// Meta returns the operation metadata.
func (TCPIPChecksum) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "TCP/IP Checksum",
		Module:      "Default",
		Description: "Calculates the checksum of a TCP (Transport Control Protocol) or IP (Internet Protocol) header from an input of raw bytes.",
		InfoURL:     "https://wikipedia.org/wiki/IPv4_header_checksum",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (TCPIPChecksum) Args() []core.ArgDef { return nil }

// Run computes the checksum.
func (TCPIPChecksum) Run(in *core.Dish, _ []any) (*core.Dish, error) {
	return core.NewDish([]byte(tcpipChecksum(in.Bytes())), core.TypeString), nil
}
