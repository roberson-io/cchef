package ops

import (
	"image"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(ImageFilter{})
}

// ImageFilter applies a greyscale or sepia filter to an image. Ported from
// CyberChef's Image Filter (Jimp's greyscale/sepia).
type ImageFilter struct{}

// Meta returns the operation metadata.
func (ImageFilter) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Image Filter",
		Module:      "Image",
		Description: "Applies a greyscale or sepia filter to an image.",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeArrayBuffer,
	}
}

// Args returns the argument definitions.
func (ImageFilter) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Filter type", Type: core.ArgOption, Value: []string{"Greyscale", "Sepia"}},
	}
}

// Run applies the selected filter.
func (ImageFilter) Run(in *core.Dish, args []any) (*core.Dish, error) {
	greyscale := args[0].(string) == "Greyscale"
	out, err := imageTransform(in.Bytes(), "Invalid file type.", func(img *image.NRGBA) *image.NRGBA {
		if greyscale {
			jimpGreyscale(img)
		} else {
			jimpSepia(img)
		}
		return img
	})
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeByteArray), nil
}

// jimpGreyscale replaces each pixel with its luminance (Rec. 709 weights).
func jimpGreyscale(img *image.NRGBA) {
	p := img.Pix
	for i := 0; i < len(p); i += 4 {
		grey := uint8(0.2126*float64(p[i]) + 0.7152*float64(p[i+1]) + 0.0722*float64(p[i+2]))
		p[i], p[i+1], p[i+2] = grey, grey, grey
	}
}

// jimpSepia applies Jimp's sepia matrix in place. It reproduces Jimp's quirk of
// reusing the freshly-computed red (then green) channel in later channels.
func jimpSepia(img *image.NRGBA) {
	p := img.Pix
	for i := 0; i < len(p); i += 4 {
		r, g, b := float64(p[i]), float64(p[i+1]), float64(p[i+2])
		red := r*0.393 + g*0.769 + b*0.189
		green := red*0.349 + g*0.686 + b*0.168
		blue := red*0.272 + green*0.534 + b*0.131
		p[i] = sepiaClamp(red)
		p[i+1] = sepiaClamp(green)
		p[i+2] = sepiaClamp(blue)
	}
}

// sepiaClamp stores a sepia channel value, capping at 255 and truncating toward
// zero (Jimp's `v < 255 ? v : 255` into a Uint8Array).
func sepiaClamp(v float64) uint8 {
	if v < 255 {
		return uint8(v)
	}
	return 255
}
