package ops

import (
	"image"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(ImageOpacity{})
}

// ImageOpacity adjusts the opacity of an image. Ported from CyberChef's Image
// Opacity (Jimp's opacity): each pixel's alpha is multiplied by opacity/100.
type ImageOpacity struct{}

// Meta returns the operation metadata.
func (ImageOpacity) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Image Opacity",
		Module:      "Image",
		Description: "Adjust the opacity of an image.",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeArrayBuffer,
	}
}

var opacityMin, opacityMax = float64(0), float64(100)

// Args returns the argument definitions.
func (ImageOpacity) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Opacity (%)", Type: core.ArgNumber, Value: float64(100), Min: &opacityMin, Max: &opacityMax},
	}
}

// Run adjusts the image opacity.
func (ImageOpacity) Run(in *core.Dish, args []any) (*core.Dish, error) {
	f := args[0].(float64) / 100
	out, err := imageTransform(in.Bytes(), "Invalid file type.", func(img *image.NRGBA) *image.NRGBA {
		jimpOpacity(img, f)
		return img
	})
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeByteArray), nil
}

// jimpOpacity multiplies each pixel's alpha by f in place, truncating toward
// zero as Jimp's Uint8Array store does.
func jimpOpacity(img *image.NRGBA, f float64) {
	p := img.Pix
	for i := 3; i < len(p); i += 4 {
		p[i] = uint8(float64(p[i]) * f)
	}
}
