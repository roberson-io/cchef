package ops

import (
	"math"
	"unicode/utf16"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(CaesarBoxCipher{})
}

// CaesarBoxCipher is a transposition cipher: the message is written row-by-row
// into a box of a given height and read back column-by-column.
type CaesarBoxCipher struct{}

// Meta returns the operation metadata.
func (CaesarBoxCipher) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Caesar Box Cipher",
		Module:      "Ciphers",
		Description: "Caesar Box is a transposition cipher used in the Roman Empire, in which letters of the message are written in rows in a square (or a rectangle) and then, read by column.",
		InfoURL:     "https://www.dcode.fr/caesar-box-cipher",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (CaesarBoxCipher) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Box Height", Type: core.ArgNumber, Integer: true, Value: 1},
	}
}

// Run applies the cipher. Ported from CyberChef CaesarBoxCipher.mjs, which
// iterates UTF-16 code units (charAt/length), strips spaces, pads the box with
// NUL code units, then reads column-by-column skipping the padding.
func (CaesarBoxCipher) Run(in *core.Dish, args []any) (*core.Dish, error) {
	height := int(args[0].(float64))

	units := utf16.Encode([]rune(in.String()))
	// A non-positive height makes the box loops never run in CyberChef (and the
	// width would be non-finite), so the result is empty.
	if height <= 0 {
		return core.NewDish([]byte{}, core.TypeString), nil
	}

	// tableWidth is derived from the original length, before spaces are stripped.
	width := int(math.Ceil(float64(len(units)) / float64(height)))

	box := make([]uint16, 0, len(units))
	for _, u := range units {
		if u != ' ' {
			box = append(box, u)
		}
	}
	for len(box) < height*width {
		box = append(box, 0)
	}

	var result []uint16
	for i := range height {
		for j := i; j < len(box); j += height {
			if box[j] != 0 {
				result = append(result, box[j])
			}
		}
	}
	return core.NewDish([]byte(string(utf16.Decode(result))), core.TypeString), nil
}
