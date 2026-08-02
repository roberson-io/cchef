package ops

import (
	"errors"
	"image"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/jimp"
)

func init() {
	core.Register(BlurImage{})
}

// BlurImage blurs an image with a fast box blur or a Gaussian blur. Ported from
// CyberChef's Blur Image (Jimp's blur/gaussian).
type BlurImage struct{}

// Meta returns the operation metadata.
func (BlurImage) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Blur Image",
		Module:      "Image",
		Description: "Applies a blur effect to the image.",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeArrayBuffer,
	}
}

var blurAmountMin = float64(1)

// Args returns the argument definitions.
func (BlurImage) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Amount", Type: core.ArgNumber, Value: float64(5), Min: &blurAmountMin},
		{Name: "Type", Type: core.ArgOption, Value: []string{"Fast", "Gaussian"}},
	}
}

// Run blurs the image.
func (BlurImage) Run(in *core.Dish, args []any) (*core.Dish, error) {
	amount := args[0].(float64)
	gaussian := args[1].(string) == "Gaussian"
	out, err := jimp.TransformE(in.Bytes(), "Invalid file type.", func(img *image.NRGBA) (*image.NRGBA, error) {
		if gaussian {
			jimp.Gaussian(img, amount)
			return img, nil
		}
		r := int(amount)
		if r < 1 || r >= jimp.BlurMaxRadius {
			//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
			return nil, errors.New("Error blurring image. (blur amount out of range)")
		}
		jimp.BlurFast(img, r)
		return img, nil
	})
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeByteArray), nil
}
