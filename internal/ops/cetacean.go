package ops

import (
	"strings"
	"unicode/utf16"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(CetaceanCipherEncode{})
	core.Register(CetaceanCipherDecode{})
}

// CetaceanCipherEncode encodes each UTF-16 code unit as its 16-bit binary
// representation, writing 'e' for a 1 bit and 'E' for a 0 bit.
type CetaceanCipherEncode struct{}

// Meta returns the operation metadata.
func (CetaceanCipherEncode) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Cetacean Cipher Encode",
		Module:      "Ciphers",
		Description: "Converts any input into Cetacean Cipher. <br/><br/>e.g. <code>hi</code> becomes <code>EEEEEEEEEeeEeEEEEEEEEEEEEeeEeEEe</code>",
		InfoURL:     "https://hitchhikers.fandom.com/wiki/Dolphins",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (CetaceanCipherEncode) Args() []core.ArgDef { return nil }

// Run encodes the input. Ported from CyberChef CetaceanCipherEncode.mjs, which
// maps over UTF-16 code units: a space passes through literally, otherwise the
// code unit's 16-bit binary becomes 'e' (1) / 'E' (0).
func (CetaceanCipherEncode) Run(in *core.Dish, args []any) (*core.Dish, error) {
	var b strings.Builder
	for _, u := range utf16.Encode([]rune(in.String())) {
		if u == ' ' {
			b.WriteByte(' ')
			continue
		}
		for bit := 15; bit >= 0; bit-- {
			if (u>>bit)&1 == 1 {
				b.WriteByte('e')
			} else {
				b.WriteByte('E')
			}
		}
	}
	return core.NewDish([]byte(b.String()), core.TypeString), nil
}

// CetaceanCipherDecode reverses CetaceanCipherEncode: each 16-'e'/'E' group is
// read back as a 16-bit code unit ('e' is 1, anything else is 0).
type CetaceanCipherDecode struct{}

// Meta returns the operation metadata.
func (CetaceanCipherDecode) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Cetacean Cipher Decode",
		Module:      "Ciphers",
		Description: "Decode Cetacean Cipher input. <br/><br/>e.g. <code>EEEEEEEEEeeEeEEEEEEEEEEEEeeEeEEe</code> becomes <code>hi</code>",
		InfoURL:     "https://hitchhikers.fandom.com/wiki/Dolphins",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (CetaceanCipherDecode) Args() []core.ArgDef { return nil }

// Run decodes the input. Ported from CyberChef CetaceanCipherDecode.mjs, which
// iterates code points: a space expands to the 16 bits of 0x20 (a space
// character), 'e' is a 1 bit and anything else is a 0 bit; every 16 bits form a
// code unit.
func (CetaceanCipherDecode) Run(in *core.Dish, args []any) (*core.Dish, error) {
	var bits []int
	for _, r := range in.String() {
		switch r {
		case ' ':
			bits = append(bits, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0)
		case 'e':
			bits = append(bits, 1)
		default:
			bits = append(bits, 0)
		}
	}

	var units []uint16
	for i := 0; i < len(bits); i += 16 {
		end := min(i+16, len(bits))
		v := 0
		for _, bit := range bits[i:end] {
			v = v<<1 | bit
		}
		units = append(units, uint16(v)) // #nosec G115 -- at most 16 bits, so v is in [0,65535]
	}
	return core.NewDish([]byte(string(utf16.Decode(units))), core.TypeString), nil
}
