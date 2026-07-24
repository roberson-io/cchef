package ops

import (
	"image"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(ResizeImage{})
}

// ResizeImage resizes an image. Ported from CyberChef's Resize Image (Jimp's
// resize/scaleToFit with the five ResizeStrategy algorithms).
type ResizeImage struct{}

// Meta returns the operation metadata.
func (ResizeImage) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Resize Image",
		Module:      "Image",
		Description: "Resizes an image to the specified width and height values.",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeArrayBuffer,
	}
}

var resizeDimMin = float64(1)

// Args returns the argument definitions.
func (ResizeImage) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Width", Type: core.ArgNumber, Value: float64(100), Min: &resizeDimMin},
		{Name: "Height", Type: core.ArgNumber, Value: float64(100), Min: &resizeDimMin},
		{Name: "Unit type", Type: core.ArgOption, Value: []string{"Pixels", "Percent"}},
		{Name: "Maintain aspect ratio", Type: core.ArgBoolean, Value: false},
		{Name: "Resizing algorithm", Type: core.ArgOption, Value: []string{
			"Nearest Neighbour", "Bilinear", "Bicubic", "Hermite", "Bezier",
		}, DefaultIndex: 1},
	}
}

// Run resizes the image.
func (ResizeImage) Run(in *core.Dish, args []any) (*core.Dish, error) {
	width := args[0].(float64)
	height := args[1].(float64)
	percent := args[2].(string) == "Percent"
	aspect := args[3].(bool)
	mode := resizeStrategyNames[args[4].(string)]

	out, err := imageTransform(in.Bytes(), "Invalid file type.", func(img *image.NRGBA) *image.NRGBA {
		w, h := width, height
		if percent {
			w = float64(img.Rect.Dx()) * (width / 100)
			h = float64(img.Rect.Dy()) * (height / 100)
		}
		if aspect {
			return jimpScaleToFit(img, w, h, mode)
		}
		return jimpResize(img, w, h, mode)
	})
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeByteArray), nil
}
