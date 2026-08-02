package ops

import (
	"image"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/jimp"
)

func init() {
	core.Register(CoverImage{})
}

// CoverImage scales an image to fill a box and crops the overflow.
type CoverImage struct{}

// Meta returns the operation metadata.
func (CoverImage) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Cover Image",
		Module:      "Image",
		Description: "Scales an image to the specified width and height, maintaining aspect ratio and cropping the overflow.",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeArrayBuffer,
	}
}

var coverDimMin = float64(1)

// Args returns the argument definitions.
func (CoverImage) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Width", Type: core.ArgNumber, Integer: true, Value: float64(100), Min: &coverDimMin},
		{Name: "Height", Type: core.ArgNumber, Integer: true, Value: float64(100), Min: &coverDimMin},
		{Name: "Horizontal align", Type: core.ArgOption, Value: []string{"Left", "Center", "Right"}, DefaultIndex: 1},
		{Name: "Vertical align", Type: core.ArgOption, Value: []string{"Top", "Middle", "Bottom"}, DefaultIndex: 1},
		{Name: "Resizing algorithm", Type: core.ArgOption, Value: []string{
			"Nearest Neighbour", "Bilinear", "Bicubic", "Hermite", "Bezier",
		}, DefaultIndex: 1},
	}
}

// Run covers the image.
func (CoverImage) Run(in *core.Dish, args []any) (*core.Dish, error) {
	w := round(args[0].(float64))
	h := round(args[1].(float64))
	alignH := jimp.HAlignIndex[args[2].(string)]
	alignV := jimp.VAlignIndex[args[3].(string)]
	mode := jimp.StrategyNames[args[4].(string)]

	out, err := jimp.TransformE(in.Bytes(), "Invalid file type.", func(img *image.NRGBA) (*image.NRGBA, error) {
		return jimp.Cover(img, w, h, alignH, alignV, mode)
	})
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeByteArray), nil
}
