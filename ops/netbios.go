package ops

import (
	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(EncodeNetBIOSName{})
	core.Register(DecodeNetBIOSName{})
}

// EncodeNetBIOSName encodes a NetBIOS name (level-1 encoding).
type EncodeNetBIOSName struct{}

// Meta returns the operation metadata.
func (EncodeNetBIOSName) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Encode NetBIOS Name",
		Module:      "Default",
		Description: "NetBIOS names as seen across the client interface to NetBIOS are exactly 16 bytes long. Within the NetBIOS-over-TCP protocols, a longer representation is used.<br><br>There are two levels of encoding. The first level maps a NetBIOS name into a domain system name.  The second level maps the domain system name into the 'compressed' representation required for interaction with the domain name system.<br><br>This operation carries out the first level of encoding. See RFC 1001 for full details.",
		InfoURL:     "https://wikipedia.org/wiki/NetBIOS",
		InputType:   core.TypeByteArray,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (EncodeNetBIOSName) Args() []core.ArgDef {
	return []core.ArgDef{{Name: "Offset", Type: core.ArgNumber, Integer: true, Value: 65}}
}

// Run encodes the name.
func (EncodeNetBIOSName) Run(in *core.Dish, args []any) (*core.Dish, error) {
	offset := byte(int(args[0].(float64))) // #nosec G115 -- offset arg coerced to a byte, matching NetBIOS byte arithmetic
	input := in.Bytes()
	if len(input) > 16 {
		return core.NewDish(nil, core.TypeByteArray), nil
	}

	// Pad to 16 bytes with spaces, then split each byte into two offset nibbles.
	buf := make([]byte, 16)
	copy(buf, input)
	for i := len(input); i < 16; i++ {
		buf[i] = 32
	}
	output := make([]byte, 0, 32)
	for _, b := range buf {
		output = append(output, (b>>4)+offset, (b&0xf)+offset)
	}
	return core.NewDish(output, core.TypeByteArray), nil
}

// DecodeNetBIOSName decodes a NetBIOS name (level-1 encoding).
type DecodeNetBIOSName struct{}

// Meta returns the operation metadata.
func (DecodeNetBIOSName) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Decode NetBIOS Name",
		Module:      "Default",
		Description: "NetBIOS names as seen across the client interface to NetBIOS are exactly 16 bytes long. Within the NetBIOS-over-TCP protocols, a longer representation is used.<br><br>There are two levels of encoding. The first level maps a NetBIOS name into a domain system name.  The second level maps the domain system name into the 'compressed' representation required for interaction with the domain name system.<br><br>This operation decodes the first level of encoding. See RFC 1001 for full details.",
		InfoURL:     "https://wikipedia.org/wiki/NetBIOS",
		InputType:   core.TypeByteArray,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (DecodeNetBIOSName) Args() []core.ArgDef {
	return []core.ArgDef{{Name: "Offset", Type: core.ArgNumber, Integer: true, Value: 65}}
}

// Run decodes the name.
func (DecodeNetBIOSName) Run(in *core.Dish, args []any) (*core.Dish, error) {
	offset := int(args[0].(float64))
	input := in.Bytes()
	var output []byte
	if len(input) <= 32 && len(input)%2 == 0 {
		for i := 0; i < len(input); i += 2 {
			hi := (int(input[i]) & 0xff) - offset
			lo := (int(input[i+1]) & 0xff) - offset
			output = append(output, byte((hi<<4)|(lo&0xf))) // #nosec G115 -- two nibbles combined into a byte
		}
		// Trim trailing padding spaces. Faithful to CyberChef's (quirky)
		// output.splice(i, i), which removes i elements starting at index i.
		for i := len(output) - 1; i > 0; i-- {
			if output[i] == 32 {
				del := i
				if i+del > len(output) {
					del = len(output) - i
				}
				output = append(output[:i], output[i+del:]...)
			} else {
				break
			}
		}
	}
	return core.NewDish(output, core.TypeByteArray), nil
}
