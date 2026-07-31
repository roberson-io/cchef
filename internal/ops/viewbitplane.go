package ops

import (
	"errors"
	"image"
	"slices"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(ViewBitPlane{})
}

// viewBitPlaneColours is the channel order shared by the colour argument.
var viewBitPlaneColours = []string{"Red", "Green", "Blue", "Alpha"}

// ViewBitPlane renders a single bit of a single channel as a black-and-white
// image. Ported from CyberChef ViewBitPlane.mjs.
type ViewBitPlane struct{}

// Meta returns the operation metadata.
func (ViewBitPlane) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "View Bit Plane",
		Module:      "Image",
		Description: "Extracts and displays a bit plane of any given image. These show only a single bit from each pixel, and can be used to hide messages in Steganography.",
		InfoURL:     "https://wikipedia.org/wiki/Bit_plane",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeArrayBuffer,
	}
}

// Args returns the channel and which bit to show.
func (ViewBitPlane) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Colour", Type: core.ArgOption, Value: viewBitPlaneColours},
		{Name: "Bit", Type: core.ArgNumber, Integer: true, Value: 0},
	}
}

// Run renders the bit plane. The bit argument is checked inside the transform
// so that an unreadable image is reported first, as upstream reports it.
func (ViewBitPlane) Run(in *core.Dish, args []any) (*core.Dish, error) {
	channel := slices.Index(viewBitPlaneColours, args[0].(string))
	bit := int(args[1].(float64))
	out, err := imageTransformE(in.Bytes(), "Please enter a valid image file.", func(img *image.NRGBA) (*image.NRGBA, error) {
		if bit < 0 || bit > 7 {
			return nil, errors.New("Error: Bit argument must be between 0 and 7") //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		}
		for i := 0; i < len(img.Pix); i += 4 {
			// A set bit paints the pixel black, a clear one white.
			v := byte(255)
			if img.Pix[i+channel]>>bit&1 == 1 {
				v = 0
			}
			img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = v, v, v, 255
		}
		return img, nil
	})
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeArrayBuffer), nil
}
