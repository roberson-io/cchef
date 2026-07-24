package ops

import (
	"image"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(NormaliseImage{})
}

// NormaliseImage stretches each colour channel to the full 0-255 range. Ported
// from CyberChef's Normalise Image (Jimp's normalize).
type NormaliseImage struct{}

// Meta returns the operation metadata.
func (NormaliseImage) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Normalise Image",
		Module:      "Image",
		Description: "Normalise the image colours.",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeArrayBuffer,
	}
}

// Args returns the argument definitions.
func (NormaliseImage) Args() []core.ArgDef { return nil }

// Run normalises the image.
func (NormaliseImage) Run(in *core.Dish, _ []any) (*core.Dish, error) {
	out, err := imageTransform(in.Bytes(), "Invalid file type.", func(img *image.NRGBA) *image.NRGBA {
		jimpNormalize(img)
		return img
	})
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeByteArray), nil
}

// histBounds returns the first and last values with a non-zero count.
func histBounds(h *[256]int) (int, int) {
	lo := 0
	for lo < 256 && h[lo] == 0 {
		lo++
	}
	hi := 255
	for hi >= 0 && h[hi] == 0 {
		hi--
	}
	return lo, hi
}

// normValue maps v from [min,max] onto 0-255 (truncated); a flat channel maps to 0.
func normValue(v, minV, maxV int) byte {
	if maxV == minV {
		return 0
	}
	return byte((float64(v-minV) * 255) / float64(maxV-minV))
}

// jimpNormalize stretches each RGB channel independently to full range.
func jimpNormalize(img *image.NRGBA) {
	var hr, hg, hb [256]int
	p := img.Pix
	for i := 0; i < len(p); i += 4 {
		hr[p[i]]++
		hg[p[i+1]]++
		hb[p[i+2]]++
	}
	rlo, rhi := histBounds(&hr)
	glo, ghi := histBounds(&hg)
	blo, bhi := histBounds(&hb)
	for i := 0; i < len(p); i += 4 {
		p[i] = normValue(int(p[i]), rlo, rhi)
		p[i+1] = normValue(int(p[i+1]), glo, ghi)
		p[i+2] = normValue(int(p[i+2]), blo, bhi)
	}
}
