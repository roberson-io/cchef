package ops

import (
	"errors"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(RenderImage{})
}

// RenderImage validates that the input is an image and passes the bytes through.
// Ported from CyberChef RenderImage.mjs. CyberChef renders the bytes as an
// <img> in the browser via presentType=html; cchef drops that browser-only
// presentation and instead offers an "Output" option: Raw bytes (the default,
// save with -o or a pipe), a base64 data-URI, or an inline terminal preview.
type RenderImage struct{}

// Meta returns the operation metadata.
func (RenderImage) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Render Image",
		Module:      "Image",
		Description: "Validates that the input is an image and outputs it. Supports jpg/jpeg, png, gif, webp, bmp and ico.",
		InfoURL:     "https://wikipedia.org/wiki/Image_file_formats",
		InputType:   core.TypeString,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (RenderImage) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Input format", Type: core.ArgOption, Value: []string{"Raw", "Base64", "Hex"}},
	}
}

// Run validates the image and passes its bytes through.
func (RenderImage) Run(in *core.Dish, args []any) (*core.Dish, error) {
	inputFormat := args[0].(string)

	if len(in.Bytes()) == 0 {
		return core.NewDish(nil, core.TypeByteArray), nil
	}

	data := decodeImageInput(in, inputFormat)

	if isImage(data) == "" {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return nil, errors.New("Invalid file type")
	}

	return core.NewDish(data, core.TypeByteArray), nil
}

// decodeImageInput converts the recipe input to raw bytes per the input format.
func decodeImageInput(in *core.Dish, inputFormat string) []byte {
	switch inputFormat {
	case "Hex":
		return fromHexAuto(in.String())
	case "Base64":
		b, _ := fromBase64(in.String(), stdBase64Alphabet, true, false)
		return b
	default: // Raw
		return in.Bytes()
	}
}
