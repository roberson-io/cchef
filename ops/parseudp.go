package ops

import (
	"encoding/hex"
	"fmt"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/bytestream"
	"github.com/roberson-io/cchef/internal/jsonval"
)

func init() {
	core.Register(ParseUDP{})
}

// parseNetInput decodes the input string per the "Hex"/"Raw" format option shared
// by the packet-parsing operations. The option list guarantees one of these two
// values, so no error path is needed.
func parseNetInput(input, format string) []byte {
	if format == "Hex" {
		return hexToBytes(input)
	}
	return []byte(input) // "Raw"
}

// ParseUDP parses a UDP datagram header into a JSON object.
type ParseUDP struct{}

// Meta returns the operation metadata.
func (ParseUDP) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Parse UDP",
		Module:      "Default",
		Description: "Parses a UDP header and payload (if present).",
		InfoURL:     "https://wikipedia.org/wiki/User_Datagram_Protocol",
		InputType:   core.TypeString,
		OutputType:  core.TypeJSON,
	}
}

// Args returns the argument definitions.
func (ParseUDP) Args() []core.ArgDef {
	return []core.ArgDef{{Name: "Input format", Type: core.ArgOption, Value: []string{"Hex", "Raw"}}}
}

// Run parses the datagram.
func (ParseUDP) Run(in *core.Dish, args []any) (*core.Dish, error) {
	s := bytestream.New(parseNetInput(in.String(), args[0].(string)))
	if s.Length() < 8 {
		return nil, fmt.Errorf("need 8 bytes for a UDP header")
	}

	udp := jsonval.NewOMap()
	udp.Set("Source port", s.ReadInt(2))
	udp.Set("Destination port", s.ReadInt(2))
	length := s.ReadInt(2)
	udp.Set("Length", length)
	udp.Set("Checksum", "0x"+hex.EncodeToString(s.GetBytes(2)))
	if s.HasMore() {
		udp.Set("Data", "0x"+hex.EncodeToString(s.GetBytes(length-8)))
	}

	out, err := jsonval.MarshalOMap(udp)
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeJSON), nil
}
