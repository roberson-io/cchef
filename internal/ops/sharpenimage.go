package ops

import (
	"image"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(SharpenImage{})
}

// SharpenImage sharpens an image with an unsharp mask. Ported from CyberChef's
// Sharpen Image, which builds the mask from a Gaussian blur (Jimp's gaussian)
// and merges it back weighted by a luminance threshold.
type SharpenImage struct{}

// Meta returns the operation metadata.
func (SharpenImage) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Sharpen Image",
		Module:      "Image",
		Description: "Sharpens an image (Unsharp mask).",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeArrayBuffer,
	}
}

var (
	sharpenRadiusMin                   = float64(1)
	sharpenAmountMin                   = float64(0)
	sharpenThreshMin, sharpenThreshMax = float64(0), float64(100)
)

// Args returns the argument definitions.
func (SharpenImage) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Radius", Type: core.ArgNumber, Value: float64(2), Min: &sharpenRadiusMin},
		{Name: "Amount", Type: core.ArgNumber, Value: float64(1), Min: &sharpenAmountMin},
		{Name: "Threshold", Type: core.ArgNumber, Value: float64(10), Min: &sharpenThreshMin, Max: &sharpenThreshMax},
	}
}

// Run sharpens the image.
func (SharpenImage) Run(in *core.Dish, args []any) (*core.Dish, error) {
	radius := args[0].(float64)
	amount := args[1].(float64)
	threshold := args[2].(float64)
	out, err := imageTransform(in.Bytes(), "Invalid file type.", func(img *image.NRGBA) *image.NRGBA {
		return jimpSharpen(img, radius, amount, threshold)
	})
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeByteArray), nil
}

// jimpSharpen applies the unsharp mask in place and returns img.
func jimpSharpen(img *image.NRGBA, radius, amount, threshold float64) *image.NRGBA {
	blur := cloneNRGBA(img)
	jimpGaussian(blur, radius)

	// mask = max(0, original - blur) per RGB channel.
	mask := cloneNRGBA(img)
	mp, bp := mask.Pix, blur.Pix
	for i := 0; i < len(mp); i += 4 {
		for c := range 3 {
			if mp[i+c] > bp[i+c] {
				mp[i+c] -= bp[i+c]
			} else {
				mp[i+c] = 0
			}
		}
	}

	p := img.Pix
	for i := 0; i < len(p); i += 4 {
		maskR, maskG, maskB := float64(mp[i]), float64(mp[i+1]), float64(mp[i+2])
		nR, nG, nB := float64(p[i]), float64(p[i+1]), float64(p[i+2])
		maskLum := 0.2126*maskR + 0.7152*maskG + 0.0722*maskB
		normLum := 0.2126*nR + 0.7152*nG + 0.0722*nB
		lumDiff := maskLum - normLum
		if lumDiff < 0 {
			lumDiff = -lumDiff
		}
		maskR, maskG, maskB = maskR*amount, maskG*amount, maskB*amount
		if (lumDiff/255)*100 >= threshold {
			p[i] = sharpenMerge(nR, maskR)
			p[i+1] = sharpenMerge(nG, maskG)
			p[i+2] = sharpenMerge(nB, maskB)
		}
	}
	return img
}

// sharpenMerge adds the weighted mask to the base channel, capping at 255.
func sharpenMerge(base, mask float64) byte {
	if base+mask <= 255 {
		return byte(base + mask)
	}
	return 255
}
