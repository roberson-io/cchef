package ops

import (
	"image"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(FlipImage{})
}

// FlipImage flips an image along its X or Y axis. Ported from CyberChef's Flip
// Image (Jimp's flip): each pixel is moved to the mirrored position.
type FlipImage struct{}

// Meta returns the operation metadata.
func (FlipImage) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Flip Image",
		Module:      "Image",
		Description: "Flips an image along its X or Y axis.",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeArrayBuffer,
	}
}

// Args returns the argument definitions.
func (FlipImage) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Axis", Type: core.ArgOption, Value: []string{"Horizontal", "Vertical"}},
	}
}

// Run flips the image.
func (FlipImage) Run(in *core.Dish, args []any) (*core.Dish, error) {
	axis := args[0].(string)
	out, err := imageTransform(in.Bytes(), "Invalid input file type.", func(img *image.NRGBA) *image.NRGBA {
		return jimpFlip(img, axis == "Horizontal", axis == "Vertical")
	})
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeByteArray), nil
}

// jimpFlip returns a copy of img mirrored on the requested axes.
func jimpFlip(img *image.NRGBA, horizontal, vertical bool) *image.NRGBA {
	w, h := img.Rect.Dx(), img.Rect.Dy()
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			dx, dy := x, y
			if horizontal {
				dx = w - 1 - x
			}
			if vertical {
				dy = h - 1 - y
			}
			si := img.PixOffset(x, y)
			di := dst.PixOffset(dx, dy)
			copy(dst.Pix[di:di+4], img.Pix[si:si+4])
		}
	}
	return dst
}
