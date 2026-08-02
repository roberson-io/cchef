package ops

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(ChangeIPFormat{})
}

// ipFromNumber splits a whole 32-bit IP (parsed in the given radix) into its four
// bytes, big-endian.
func ipFromNumber(value string, radix int) []byte {
	dec, _ := strconv.ParseUint(strings.TrimSpace(value), radix, 64)
	d := uint32(dec)                                                   // #nosec G115 -- 32-bit coercion of the parsed IP matches CyberChef
	return []byte{byte(d >> 24), byte(d >> 16), byte(d >> 8), byte(d)} // #nosec G115 -- extracting the four big-endian octets of a 32-bit IP
}

// ipFormats are the input/output radix options for Change IP format.
var ipFormats = []string{"Dotted Decimal", "Decimal", "Octal", "Hex"}

// ChangeIPFormat converts IPv4 addresses between dotted-decimal, decimal, octal
// and hex representations.
type ChangeIPFormat struct{}

// Meta returns the operation metadata.
func (ChangeIPFormat) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Change IP format",
		Module:      "Default",
		Description: "Convert an IP address from one format to another, e.g. <code>172.20.23.54</code> to <code>0xac141736</code>",
		InfoURL:     "",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ChangeIPFormat) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Input format", Type: core.ArgOption, Value: ipFormats},
		{Name: "Output format", Type: core.ArgOption, Value: ipFormats},
	}
}

// Run converts the IP format.
func (ChangeIPFormat) Run(in *core.Dish, args []any) (*core.Dish, error) {
	inFormat := args[0].(string)
	outFormat := args[1].(string)

	var out strings.Builder
	for line := range strings.SplitSeq(in.String(), "\n") {
		if line == "" {
			continue
		}
		if inFormat == outFormat {
			out.WriteString(line + "\n")
			continue
		}

		baIP, err := ipParseInput(inFormat, line)
		if err != nil {
			return nil, err
		}
		formatted, err := ipFormatOutput(outFormat, baIP)
		if err != nil {
			return nil, err
		}
		out.WriteString(formatted + "\n")
	}

	s := out.String()
	if len(s) > 0 {
		s = s[:len(s)-1] // drop the trailing newline
	}
	return core.NewDish([]byte(s), core.TypeString), nil
}

// ipParseInput decodes one input line into IP bytes according to inFormat.
func ipParseInput(inFormat, line string) ([]byte, error) {
	switch inFormat {
	case "Dotted Decimal":
		var baIP []byte
		for oct := range strings.SplitSeq(line, ".") {
			v, _ := strconv.Atoi(oct)
			baIP = append(baIP, byte(v)) // #nosec G115 -- octet from dotted-decimal, bounded to a byte (faithful truncation)
		}
		return baIP, nil
	case "Decimal":
		return ipFromNumber(line, 10), nil
	case "Octal":
		return ipFromNumber(line, 8), nil
	case "Hex":
		return hexToBytes(line), nil
	default:
		return nil, fmt.Errorf("unsupported input IP format")
	}
}

// ipFormatOutput renders IP bytes as a string according to outFormat.
func ipFormatOutput(outFormat string, baIP []byte) (string, error) {
	switch outFormat {
	case "Dotted Decimal":
		parts := make([]string, len(baIP))
		for i, b := range baIP {
			parts[i] = strconv.Itoa(int(b))
		}
		return strings.Join(parts, "."), nil
	case "Decimal":
		return strconv.FormatUint(uint64(ipToUint32(baIP)), 10), nil
	case "Octal":
		return "0" + strconv.FormatUint(uint64(ipToUint32(baIP)), 8), nil
	case "Hex":
		var hex strings.Builder
		for _, b := range baIP {
			fmt.Fprintf(&hex, "%02x", b)
		}
		return hex.String(), nil
	default:
		return "", fmt.Errorf("unsupported output IP format")
	}
}

// ipToUint32 packs the first four bytes of an IP byte array into a 32-bit value.
func ipToUint32(ba []byte) uint32 {
	var v uint32
	for i := 0; i < 4 && i < len(ba); i++ {
		v |= uint32(ba[i]) << (24 - 8*i)
	}
	return v
}
