package ops

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(ChangeIPFormat{})
}

// ipFromNumber splits a whole 32-bit IP (parsed in the given radix) into its four
// bytes, big-endian. Ported from ChangeIPFormat.mjs fromNumber.
func ipFromNumber(value string, radix int) []byte {
	dec, _ := strconv.ParseUint(strings.TrimSpace(value), radix, 64)
	d := uint32(dec)
	return []byte{byte(d >> 24), byte(d >> 16), byte(d >> 8), byte(d)}
}

// hexToBytes decodes a hex string (ignoring non-hex characters) into bytes,
// matching CyberChef's fromHex for the delimiter-free case.
func hexToBytes(s string) []byte {
	clean := nonHex.ReplaceAllString(s, "")
	out := make([]byte, 0, len(clean)/2)
	for i := 0; i+2 <= len(clean); i += 2 {
		v, _ := strconv.ParseUint(clean[i:i+2], 16, 8)
		out = append(out, byte(v))
	}
	return out
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

// Run converts the IP format. Ported from CyberChef ChangeIPFormat.mjs.
func (ChangeIPFormat) Run(in *core.Dish, args []any) (*core.Dish, error) {
	inFormat := args[0].(string)
	outFormat := args[1].(string)

	var out strings.Builder
	for _, line := range strings.Split(in.String(), "\n") {
		if line == "" {
			continue
		}
		if inFormat == outFormat {
			out.WriteString(line + "\n")
			continue
		}

		var baIP []byte
		switch inFormat {
		case "Dotted Decimal":
			for _, oct := range strings.Split(line, ".") {
				v, _ := strconv.Atoi(oct)
				baIP = append(baIP, byte(v))
			}
		case "Decimal":
			baIP = ipFromNumber(line, 10)
		case "Octal":
			baIP = ipFromNumber(line, 8)
		case "Hex":
			baIP = hexToBytes(line)
		default:
			return nil, fmt.Errorf("unsupported input IP format")
		}

		switch outFormat {
		case "Dotted Decimal":
			parts := make([]string, len(baIP))
			for i, b := range baIP {
				parts[i] = strconv.Itoa(int(b))
			}
			out.WriteString(strings.Join(parts, ".") + "\n")
		case "Decimal":
			out.WriteString(strconv.FormatUint(uint64(ipToUint32(baIP)), 10) + "\n")
		case "Octal":
			out.WriteString("0" + strconv.FormatUint(uint64(ipToUint32(baIP)), 8) + "\n")
		case "Hex":
			for _, b := range baIP {
				fmt.Fprintf(&out, "%02x", b)
			}
			out.WriteString("\n")
		default:
			return nil, fmt.Errorf("unsupported output IP format")
		}
	}

	s := out.String()
	if len(s) > 0 {
		s = s[:len(s)-1] // drop the trailing newline
	}
	return core.NewDish([]byte(s), core.TypeString), nil
}

// ipToUint32 packs the first four bytes of an IP byte array into a 32-bit value.
func ipToUint32(ba []byte) uint32 {
	var v uint32
	for i := 0; i < 4 && i < len(ba); i++ {
		v |= uint32(ba[i]) << (24 - 8*i)
	}
	return v
}
