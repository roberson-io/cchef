package ops

import (
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/jimp"
)

func init() {
	core.Register(ExtractRGBA{})
}

// ExtractRGBA lists every pixel's channel values. Ported from CyberChef
// ExtractRGBA.mjs.
type ExtractRGBA struct{}

// Meta returns the operation metadata.
func (ExtractRGBA) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Extract RGBA",
		Module:      "Image",
		Description: "Extracts each pixel's RGBA value in an image. These are sometimes used in Steganography to hide text or data.",
		InfoURL:     "https://wikipedia.org/wiki/RGBA_color_space",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeString,
	}
}

// Args returns the delimiter (any string, comma by default) and whether the
// alpha channel is included.
func (ExtractRGBA) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Delimiter", Type: core.ArgEditableOption, Value: ","},
		{Name: "Include Alpha", Type: core.ArgBoolean, Value: true},
	}
}

// Run lists the channel values.
func (ExtractRGBA) Run(in *core.Dish, args []any) (*core.Dish, error) {
	img, _, err := jimp.Decode(in.Bytes(), "Please enter a valid image file.")
	if err != nil {
		return nil, err
	}
	delimiter := args[0].(string)
	includeAlpha := args[1].(bool)

	var out strings.Builder
	first := true
	for i, v := range img.Pix {
		if !includeAlpha && i%4 == 3 {
			continue
		}
		if !first {
			out.WriteString(delimiter)
		}
		first = false
		out.WriteString(strconv.Itoa(int(v)))
	}
	return core.NewDish([]byte(out.String()), core.TypeString), nil
}
