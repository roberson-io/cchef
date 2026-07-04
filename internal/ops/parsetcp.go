package ops

import (
	"fmt"
	"math/big"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(ParseTCP{})
}

// tcpOpt describes a TCP option kind: its name, whether it carries a length byte,
// and an optional value parser.
type tcpOpt struct {
	name   string
	length bool
	parser func([]byte) any
}

// bytesToLargeNumber renders a big-endian byte array as a decimal string
// (CyberChef bytesToLargeNumber via BigNumber).
func bytesToLargeNumber(bs []byte) string { return new(big.Int).SetBytes(bs).String() }

func windowScaleParser(data []byte) any {
	if len(data) != 1 {
		return fmt.Sprintf("Error: Window Scale should be one byte long (received 0x%s)", toHexFast(data))
	}
	return newOMap().set("Shift count", int(data[0])).set("Multiplier", 1<<data[0])
}

func tcpTimestampParser(data []byte) any {
	if len(data) != 8 {
		return fmt.Sprintf("Error: Timestamp field should be 8 bytes long (received 0x%s)", toHexFast(data))
	}
	return newOMap().set("Current Timestamp", bytesToLargeNumber(data[0:4])).set("Echo Reply", bytesToLargeNumber(data[4:8]))
}

func tcpAlternateChecksumParser(data []byte) any {
	lookup := map[byte]string{
		0: "TCP Checksum", 1: "8-bit Fletchers's algorithm",
		2: "16-bit Fletchers's algorithm", 3: "Redundant Checksum Avoidance",
	}[data[0]]
	return fmt.Sprintf("%s (0x%s)", lookup, toHexFast(data))
}

// tcpOptionKindLookup mirrors CyberChef's TCP_OPTION_KIND_LOOKUP (IANA registry).
var tcpOptionKindLookup = map[int]tcpOpt{
	0:   {"End of Option List", false, nil},
	1:   {"No-Operation", false, nil},
	2:   {"Maximum Segment Size", true, nil},
	3:   {"Window Scale", true, windowScaleParser},
	4:   {"SACK Permitted", true, nil},
	5:   {"SACK", true, nil},
	6:   {"Echo (obsoleted by option 8)", true, nil},
	7:   {"Echo Reply (obsoleted by option 8)", true, nil},
	8:   {"Timestamps", true, tcpTimestampParser},
	9:   {"Partial Order Connection Permitted (obsolete)", true, nil},
	10:  {"Partial Order Service Profile (obsolete)", true, nil},
	11:  {"CC (obsolete)", true, nil},
	12:  {"CC.NEW (obsolete)", true, nil},
	13:  {"CC.ECHO (obsolete)", true, nil},
	14:  {"TCP Alternate Checksum Request (obsolete)", true, tcpAlternateChecksumParser},
	15:  {"TCP Alternate Checksum Data (obsolete)", true, nil},
	16:  {"Skeeter", true, nil},
	17:  {"Bubba", true, nil},
	18:  {"Trailer Checksum Option", true, nil},
	19:  {"MD5 Signature Option (obsoleted by option 29)", true, nil},
	20:  {"SCPS Capabilities", true, nil},
	21:  {"Selective Negative Acknowledgements", true, nil},
	22:  {"Record Boundaries", true, nil},
	23:  {"Corruption experienced", true, nil},
	24:  {"SNAP", true, nil},
	25:  {"Unassigned (released 2000-12-18)", true, nil},
	26:  {"TCP Compression Filter", true, nil},
	27:  {"Quick-Start Response", true, nil},
	28:  {"User Timeout Option (also, other known unauthorized use)", true, nil},
	29:  {"TCP Authentication Option (TCP-AO)", true, nil},
	30:  {"Multipath TCP (MPTCP)", true, nil},
	69:  {"Encryption Negotiation (TCP-ENO)", true, nil},
	70:  {"Reserved (known unauthorized use without proper IANA assignment)", true, nil},
	76:  {"Reserved (known unauthorized use without proper IANA assignment)", true, nil},
	77:  {"Reserved (known unauthorized use without proper IANA assignment)", true, nil},
	78:  {"Reserved (known unauthorized use without proper IANA assignment)", true, nil},
	253: {"RFC3692-style Experiment 1 (also improperly used for shipping products) ", true, nil},
	254: {"RFC3692-style Experiment 2 (also improperly used for shipping products) ", true, nil},
}

// ParseTCP parses a TCP segment header (and options) into a JSON object.
type ParseTCP struct{}

// Meta returns the operation metadata.
func (ParseTCP) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Parse TCP",
		Module:      "Default",
		Description: "Parses a TCP header and payload (if present).",
		InfoURL:     "https://wikipedia.org/wiki/Transmission_Control_Protocol",
		InputType:   core.TypeString,
		OutputType:  core.TypeJSON,
	}
}

// Args returns the argument definitions.
func (ParseTCP) Args() []core.ArgDef {
	return []core.ArgDef{{Name: "Input format", Type: core.ArgOption, Value: []string{"Hex", "Raw"}}}
}

// Run parses the segment. Ported from CyberChef ParseTCP.mjs.
func (ParseTCP) Run(in *core.Dish, args []any) (*core.Dish, error) {
	s := newByteStream(parseNetInput(in.String(), args[0].(string)))
	if s.length() < 20 {
		return nil, fmt.Errorf("need at least 20 bytes for a TCP header")
	}

	tcp := newOMap()
	tcp.set("Source port", s.readInt(2))
	tcp.set("Destination port", s.readInt(2))
	tcp.set("Sequence number", bytesToLargeNumber(s.getBytes(4)))
	tcp.set("Acknowledgement number", s.readInt(4))
	dataOffset := s.readBits(4)
	tcp.set("Data offset", dataOffset)

	flags := newOMap()
	flags.set("Reserved", fmt.Sprintf("%03b", s.readBits(3)))
	for _, name := range []string{"NS", "CWR", "ECE", "URG", "ACK", "PSH", "RST", "SYN", "FIN"} {
		flags.set(name, s.readBits(1))
	}
	tcp.set("Flags", flags)

	windowSize := s.readInt(2)
	tcp.set("Window size", windowSize)
	tcp.set("Checksum", "0x"+toHexFast(s.getBytes(2)))
	tcp.set("Urgent pointer", "0x"+toHexFast(s.getBytes(2)))

	windowScaleShift := 0
	if dataOffset > 5 {
		remaining := dataOffset*4 - 20
		options := newOMap()
		for remaining > 0 {
			kind := s.readInt(1)
			option := newOMap()
			option.set("Kind", kind)
			opt, ok := tcpOptionKindLookup[kind]
			if !ok {
				opt = tcpOpt{name: "Reserved", length: true}
			}
			optLength := 0
			if opt.length {
				optLength = s.readInt(1)
				option.set("Length", optLength)
				if optLength > 2 {
					var value any
					switch {
					case opt.parser != nil:
						value = opt.parser(s.getBytes(optLength - 2))
					case optLength <= 6:
						value = s.readInt(optLength - 2)
					default:
						value = "0x" + toHexFast(s.getBytes(optLength-2))
					}
					option.set("Value", value)
					if kind == 3 {
						if om, ok := value.(*omap); ok {
							if sc, ok := om.vals["Shift count"].(int); ok {
								windowScaleShift = sc
							}
						}
					}
				}
			}
			options.set(opt.name, option)
			length := 1
			if optLength != 0 {
				length = optLength
			}
			remaining -= length
		}
		tcp.set("Options", options)
	}

	if s.hasMore() {
		tcp.set("Data", "0x"+toHexFast(s.getBytes(-1)))
	}

	// Improve display values (updates in place, preserving key order).
	tcp.set("Data offset", fmt.Sprintf("%d (%d bytes)", dataOffset, dataOffset*4))
	scaled := new(big.Int).Lsh(big.NewInt(int64(windowSize)), uint(windowScaleShift))
	tcp.set("Window size", fmt.Sprintf("%d (Scaled: %s)", windowSize, scaled.String()))

	out, err := marshalOMap(tcp)
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeJSON), nil
}
