package ops

import (
	"image"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(ContainImage{})
}

// ContainImage scales an image to fit within a bounding box, letterboxing it.
// Ported from CyberChef's Contain Image (Jimp's contain + optional background).
type ContainImage struct{}

// Meta returns the operation metadata.
func (ContainImage) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Contain Image",
		Module:      "Image",
		Description: "Scales an image to the specified width and height, maintaining aspect ratio and letterboxing the rest.",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeArrayBuffer,
	}
}

var containDimMin = float64(1)

// Args returns the argument definitions.
func (ContainImage) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Width", Type: core.ArgNumber, Value: float64(100), Min: &containDimMin},
		{Name: "Height", Type: core.ArgNumber, Value: float64(100), Min: &containDimMin},
		{Name: "Horizontal align", Type: core.ArgOption, Value: []string{"Left", "Center", "Right"}, DefaultIndex: 1},
		{Name: "Vertical align", Type: core.ArgOption, Value: []string{"Top", "Middle", "Bottom"}, DefaultIndex: 1},
		{Name: "Resizing algorithm", Type: core.ArgOption, Value: []string{
			"Nearest Neighbour", "Bilinear", "Bicubic", "Hermite", "Bezier",
		}, DefaultIndex: 1},
		{Name: "Opaque background", Type: core.ArgBoolean, Value: true},
	}
}

// Run contains the image.
func (ContainImage) Run(in *core.Dish, args []any) (*core.Dish, error) {
	w := round(args[0].(float64))
	h := round(args[1].(float64))
	alignH := hAlignIndex[args[2].(string)]
	alignV := vAlignIndex[args[3].(string)]
	mode := resizeStrategyNames[args[4].(string)]
	opaqueBg := args[5].(bool)

	out, err := imageTransform(in.Bytes(), "Invalid file type.", func(img *image.NRGBA) *image.NRGBA {
		contained := jimpContain(img, w, h, alignH, alignV, mode)
		if opaqueBg {
			bg := image.NewNRGBA(image.Rect(0, 0, w, h))
			for i := 3; i < len(bg.Pix); i += 4 {
				bg.Pix[i] = 255 // opaque black
			}
			jimpBlit(bg, contained, 0, 0)
			return bg
		}
		return contained
	})
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeByteArray), nil
}
