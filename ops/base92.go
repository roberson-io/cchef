package ops

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(ToBase92{})
	core.Register(FromBase92{})
}

// base92Chr returns the byte for a Base92 value (0-90). Ported from lib/Base92.mjs.
func base92Chr(val int) byte {
	switch {
	case val == 0:
		return '!'
	case val <= 61:
		return byte('#' + val - 1) // #nosec G115 -- value bounded by the Base92 alphabet
	default:
		return byte('a' + val - 62) // #nosec G115 -- value bounded by the Base92 alphabet
	}
}

// base92Ord returns the Base92 value of a byte, or an error if invalid.
func base92Ord(c byte) (int, error) {
	switch {
	case c == '!':
		return 0, nil
	case c >= '#' && c <= '_':
		return int(c) - '#' + 1, nil
	case c >= 'a' && c <= '}':
		return int(c) - 'a' + 62, nil
	default:
		return 0, fmt.Errorf("%q is not a base92 character", string(c))
	}
}

// ToBase92 encodes raw bytes as a Base92 string.
type ToBase92 struct{}

// Meta returns the operation metadata.
func (ToBase92) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "To Base92",
		Module:      "Default",
		Description: "Base92 encodes arbitrary byte data using 91 printable ASCII characters for a compact representation.",
		InfoURL:     "https://wikipedia.org/wiki/List_of_numeral_systems",
		InputType:   core.TypeString,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (ToBase92) Args() []core.ArgDef { return nil }

// Run encodes the input. Ported from CyberChef ToBase92.mjs.
func (ToBase92) Run(in *core.Dish, args []any) (*core.Dish, error) {
	input := in.Bytes()
	var res []byte
	var bits strings.Builder
	pos := 0

	for pos < len(input) {
		for bits.Len() < 13 && pos < len(input) {
			fmt.Fprintf(&bits, "%08b", input[pos])
			pos++
		}
		if bits.Len() < 13 {
			break
		}
		s := bits.String()
		i, _ := strconv.ParseInt(s[:13], 2, 64)
		res = append(res, base92Chr(int(i)/91), base92Chr(int(i)%91))
		rest := s[13:]
		bits.Reset()
		bits.WriteString(rest)
	}

	if bits.Len() > 0 {
		s := bits.String()
		if len(s) < 7 {
			s = padEnd(s, 6)
			v, _ := strconv.ParseInt(s, 2, 64)
			res = append(res, base92Chr(int(v)))
		} else {
			s = padEnd(s, 13)
			i, _ := strconv.ParseInt(s[:13], 2, 64)
			res = append(res, base92Chr(int(i)/91), base92Chr(int(i)%91))
		}
	}
	return core.NewDish(res, core.TypeByteArray), nil
}

// FromBase92 decodes a Base92 string back into raw bytes.
type FromBase92 struct{}

// Meta returns the operation metadata.
func (FromBase92) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "From Base92",
		Module:      "Default",
		Description: "Decodes a Base92 string back into its raw byte value.",
		InfoURL:     "https://wikipedia.org/wiki/List_of_numeral_systems",
		InputType:   core.TypeString,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (FromBase92) Args() []core.ArgDef { return nil }

// Run decodes the input. Ported from CyberChef FromBase92.mjs.
func (FromBase92) Run(in *core.Dish, args []any) (*core.Dish, error) {
	input := in.Bytes()
	var res []byte
	var bits strings.Builder

	for i := 0; i < len(input); i += 2 {
		if i+1 != len(input) {
			a, err := base92Ord(input[i])
			if err != nil {
				return nil, err
			}
			b, err := base92Ord(input[i+1])
			if err != nil {
				return nil, err
			}
			fmt.Fprintf(&bits, "%013b", a*91+b)
		} else {
			a, err := base92Ord(input[i])
			if err != nil {
				return nil, err
			}
			fmt.Fprintf(&bits, "%06b", a)
		}
		for bits.Len() >= 8 {
			s := bits.String()
			v, _ := strconv.ParseInt(s[:8], 2, 64)
			res = append(res, byte(v)) // #nosec G115 -- value bounded by the Base92 alphabet
			rest := s[8:]
			bits.Reset()
			bits.WriteString(rest)
		}
	}
	return core.NewDish(res, core.TypeByteArray), nil
}

// padEnd right-pads a binary string with '0' to at least width characters.
func padEnd(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat("0", width-len(s))
}
