package ops

import (
	"image"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(InvertImage{})
}

// InvertImage inverts the colours of an image. Ported from CyberChef's Invert
// Image (Jimp's invert): each RGB channel becomes 255-value; alpha is untouched.
type InvertImage struct{}

// Meta returns the operation metadata.
func (InvertImage) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Invert Image",
		Module:      "Image",
		Description: "Invert the colours of an image.",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeArrayBuffer,
	}
}

// Args returns the argument definitions.
func (InvertImage) Args() []core.ArgDef { return nil }

// Run inverts the image colours.
func (InvertImage) Run(in *core.Dish, _ []any) (*core.Dish, error) {
	out, err := imageTransform(in.Bytes(), "Invalid input file format.", func(img *image.NRGBA) *image.NRGBA {
		jimpInvert(img)
		return img
	})
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeByteArray), nil
}

// jimpInvert replaces each RGB channel with 255-value in place.
func jimpInvert(img *image.NRGBA) {
	p := img.Pix
	for i := 0; i < len(p); i += 4 {
		p[i] = 255 - p[i]
		p[i+1] = 255 - p[i+1]
		p[i+2] = 255 - p[i+2]
	}
}
