package ops

import (
	"image"
	"math"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/jimp"
)

func init() {
	core.Register(ImageHueSaturationLightness{})
}

// ImageHueSaturationLightness adjusts an image's hue, saturation and lightness.
type ImageHueSaturationLightness struct{}

// Meta returns the operation metadata.
func (ImageHueSaturationLightness) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Image Hue/Saturation/Lightness",
		Module:      "Image",
		Description: "Adjusts the hue / saturation / lightness (HSL) values of an image.",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeArrayBuffer,
	}
}

var (
	hueMin, hueMax = float64(-360), float64(360)
	slMin, slMax   = float64(-100), float64(100)
)

// Args returns the argument definitions.
func (ImageHueSaturationLightness) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Hue", Type: core.ArgNumber, Value: float64(0), Min: &hueMin, Max: &hueMax},
		{Name: "Saturation", Type: core.ArgNumber, Value: float64(0), Min: &slMin, Max: &slMax},
		{Name: "Lightness", Type: core.ArgNumber, Value: float64(0), Min: &slMin, Max: &slMax},
	}
}

// Run applies the hue, saturation and lightness adjustments in order.
func (ImageHueSaturationLightness) Run(in *core.Dish, args []any) (*core.Dish, error) {
	hue := args[0].(float64)
	saturation := args[1].(float64)
	lightness := args[2].(float64)
	out, err := jimp.Transform(in.Bytes(), "Invalid file type.", func(img *image.NRGBA) *image.NRGBA {
		if hue != 0 {
			hslAdjust(img, "spin", hue)
		}
		if saturation != 0 {
			hslAdjust(img, "saturate", saturation)
		}
		if lightness != 0 {
			hslAdjust(img, "lighten", lightness)
		}
		return img
	})
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeByteArray), nil
}

// hslAdjust applies one tinycolor HSL modification to every pixel's RGB.
func hslAdjust(img *image.NRGBA, action string, amount float64) {
	p := img.Pix
	for i := 0; i < len(p); i += 4 {
		h, s, l := rgbToHsl(int(p[i]), int(p[i+1]), int(p[i+2]))
		hDeg := h * 360
		switch action {
		case "spin":
			hue := math.Mod(hDeg+amount, 360)
			if hue < 0 {
				hue += 360
			}
			hDeg = hue
		case "saturate":
			s = tcClamp01(s + amount/100)
		case "lighten":
			l = tcClamp01(l + amount/100)
		}
		r, g, b := hslToRgb(hDeg, s, l)
		p[i], p[i+1], p[i+2] = tcRound(r), tcRound(g), tcRound(b)
	}
}
