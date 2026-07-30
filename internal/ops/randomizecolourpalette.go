package ops

import (
	"crypto/md5" // #nosec G501 -- the palette is defined by MD5, matching CyberChef; not a security control
	"fmt"
	"image"
	"math/rand/v2"
	"strconv"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(RandomizeColourPalette{})
}

// RandomizeColourPalette recolours an image so that every distinct colour
// becomes a new one derived from a seed. Ported from CyberChef
// RandomizeColourPalette.mjs.
type RandomizeColourPalette struct{}

// Meta returns the operation metadata.
func (RandomizeColourPalette) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Randomize Colour Palette",
		Module:      "Image",
		Description: "Randomizes each colour in an image's colour palette. This can often reveal text or symbols that were previously a very similar colour to their surroundings, a technique sometimes used in Steganography.",
		InfoURL:     "https://wikipedia.org/wiki/Indexed_color",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeArrayBuffer,
	}
}

// Args returns the seed; empty means a random one.
func (RandomizeColourPalette) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Seed", Type: core.ArgString, Value: ""},
	}
}

// Run recolours the image.
func (RandomizeColourPalette) Run(in *core.Dish, args []any) (*core.Dish, error) {
	seed := args[0].(string)
	if seed == "" {
		// An empty seed means a random palette, drawn the way CyberChef draws
		// one: the decimal digits of a random fraction.
		seed = strconv.FormatFloat(rand.Float64(), 'f', -1, 64)[2:] // #nosec G404 -- a random palette seed, not a secret
	}
	out, err := imageTransform(in.Bytes(), "Please enter a valid image file.", func(img *image.NRGBA) *image.NRGBA {
		randomizePalette(img, seed)
		return img
	})
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeArrayBuffer), nil
}

// randomizePalette replaces every pixel with the first three bytes of
// md5(seed + "R.G.B"), fully opaque. The same source colour always lands on
// the same new colour, so the image's palette is shuffled but its shapes stay.
func randomizePalette(img *image.NRGBA, seed string) {
	for i := 0; i < len(img.Pix); i += 4 {
		digest := md5.Sum(fmt.Appendf(nil, "%s%d.%d.%d", // #nosec G401 -- defined by the operation
			seed, img.Pix[i], img.Pix[i+1], img.Pix[i+2]))
		copy(img.Pix[i:i+3], digest[:3])
		img.Pix[i+3] = 0xff
	}
}
