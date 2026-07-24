package ops

import (
	"image"
	"math"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(ImageBrightnessContrast{})
}

// ImageBrightnessContrast adjusts image brightness and contrast. Ported from
// CyberChef's Image Brightness / Contrast (Jimp's brightness/contrast).
type ImageBrightnessContrast struct{}

// Meta returns the operation metadata.
func (ImageBrightnessContrast) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Image Brightness / Contrast",
		Module:      "Image",
		Description: "Adjust the brightness or contrast of an image.",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeArrayBuffer,
	}
}

var bcMin, bcMax = float64(-100), float64(100)

// Args returns the argument definitions.
func (ImageBrightnessContrast) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Brightness", Type: core.ArgNumber, Value: float64(0), Min: &bcMin, Max: &bcMax},
		{Name: "Contrast", Type: core.ArgNumber, Value: float64(0), Min: &bcMin, Max: &bcMax},
	}
}

// Run adjusts brightness then contrast.
func (ImageBrightnessContrast) Run(in *core.Dish, args []any) (*core.Dish, error) {
	brightness := args[0].(float64)
	contrast := args[1].(float64)
	out, err := imageTransform(in.Bytes(), "Invalid file type.", func(img *image.NRGBA) *image.NRGBA {
		if brightness != 0 {
			jimpBrightness(img, brightness/100)
		}
		if contrast != 0 {
			jimpContrast(img, contrast/100)
		}
		return img
	})
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeByteArray), nil
}

// jimpBrightness multiplies each RGB channel by val, clamping to 0-255.
func jimpBrightness(img *image.NRGBA, val float64) {
	p := img.Pix
	for i := 0; i < len(p); i += 4 {
		p[i] = byte(math.Max(0, math.Min(255, float64(p[i])*val)))
		p[i+1] = byte(math.Max(0, math.Min(255, float64(p[i+1])*val)))
		p[i+2] = byte(math.Max(0, math.Min(255, float64(p[i+2])*val)))
	}
}

// jimpContrast applies Jimp's contrast factor around the 127 midpoint.
func jimpContrast(img *image.NRGBA, val float64) {
	factor := (val + 1) / (1 - val)
	adjust := func(v byte) byte {
		x := int(math.Floor(factor*(float64(v)-127) + 127))
		if x < 0 {
			return 0
		}
		if x > 255 {
			return 255
		}
		return byte(x)
	}
	p := img.Pix
	for i := 0; i < len(p); i += 4 {
		p[i] = adjust(p[i])
		p[i+1] = adjust(p[i+1])
		p[i+2] = adjust(p[i+2])
	}
}
