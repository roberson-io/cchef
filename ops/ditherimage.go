package ops

import (
	"image"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/jimp"
)

func init() {
	core.Register(DitherImage{})
}

// DitherImage applies an ordered dither effect.
type DitherImage struct{}

// Meta returns the operation metadata.
func (DitherImage) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Dither Image",
		Module:      "Image",
		Description: "Apply a dither effect to an image.",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeArrayBuffer,
	}
}

// Args returns the argument definitions.
func (DitherImage) Args() []core.ArgDef { return nil }

// Run dithers the image.
func (DitherImage) Run(in *core.Dish, _ []any) (*core.Dish, error) {
	out, err := jimp.Transform(in.Bytes(), "Invalid file type.", func(img *image.NRGBA) *image.NRGBA {
		jimpDither(img)
		return img
	})
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeByteArray), nil
}

// ditherMatrix is Jimp's 4x4 RGB565 ordered-dither threshold matrix.
var ditherMatrix = [16]int{1, 9, 3, 11, 13, 5, 15, 7, 4, 12, 2, 10, 16, 8, 14, 6}

// jimpDither adds a per-position dither offset to each RGB channel.
func jimpDither(img *image.NRGBA) {
	w := img.Rect.Dx()
	for i := 0; i < len(img.Pix); i += 4 {
		p := i / 4
		d := ditherMatrix[((p/w&3)<<2)+(p%w%4)]
		img.Pix[i] = ditherChannel(img.Pix[i], d)
		img.Pix[i+1] = ditherChannel(img.Pix[i+1], d)
		img.Pix[i+2] = ditherChannel(img.Pix[i+2], d)
	}
}

// ditherChannel adds the dither offset to a channel, capping at 255.
func ditherChannel(v byte, d int) byte {
	// #nosec G115 -- min caps the sum at 255
	return byte(min(int(v)+d, 0xff))
}
