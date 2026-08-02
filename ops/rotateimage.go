package ops

import (
	"image"
	"math"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/jimp"
)

func init() {
	core.Register(RotateImage{})
}

// RotateImage rotates an image by a number of degrees. Ported from CyberChef's
// Rotate Image (Jimp's rotate). Multiples of 90 degrees use Jimp's exact
// matrix rotation (pixel-identical); other angles use an approximate rotation
// with the same output canvas size (Jimp upscales first, so pixels differ —
// reduced fidelity, documented).
type RotateImage struct{}

// Meta returns the operation metadata.
func (RotateImage) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Rotate Image",
		Module:      "Image",
		Description: "Rotates an image by the specified number of degrees.",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeArrayBuffer,
	}
}

// Args returns the argument definitions.
func (RotateImage) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Rotation amount (degrees)", Type: core.ArgNumber, Value: float64(90)},
	}
}

// Run rotates the image.
func (RotateImage) Run(in *core.Dish, args []any) (*core.Dish, error) {
	deg := args[0].(float64)
	out, err := jimp.Transform(in.Bytes(), "Invalid file type.", func(img *image.NRGBA) *image.NRGBA {
		return jimpRotate(img, deg)
	})
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeByteArray), nil
}

// jimpRotate dispatches to the exact matrix rotation for multiples of 90 and to
// the approximate rotation otherwise.
func jimpRotate(img *image.NRGBA, deg float64) *image.NRGBA {
	deg = math.Mod(deg, 360)
	if deg == 0 {
		return img
	}
	if math.Mod(deg, 90) == 0 {
		return matrixRotate(img, int(deg))
	}
	return advancedRotate(img, deg)
}

// matrixRotate rotates by a multiple of 90 degrees, exactly reproducing Jimp's
// matrixRotate: pixels are relocated with no resampling and the canvas is
// resized.
func matrixRotate(img *image.NRGBA, deg int) *image.NRGBA {
	w, h := img.Rect.Dx(), img.Rect.Dy()
	var angle int
	switch deg {
	case 90, -270:
		angle = 90
	case 180, -180:
		angle = 180
	case 270, -90:
		angle = -90
	}
	nW, nH := h, w
	if angle == 180 {
		nW, nH = w, h
	}
	dst := image.NewNRGBA(image.Rect(0, 0, nW, nH))
	for x := range w {
		for y := range h {
			var dx, dy int
			switch angle {
			case 90:
				dx, dy = y, w-x-1
			case -90:
				dx, dy = h-y-1, x
			case 180:
				dx, dy = w-x-1, h-y-1
			}
			si := img.PixOffset(x, y)
			di := dst.PixOffset(dx, dy)
			copy(dst.Pix[di:di+4], img.Pix[si:si+4])
		}
	}
	return dst
}

// advancedRotate rotates by an arbitrary angle into a canvas sized with Jimp's
// expansion formula, sampling the source nearest-neighbour (approximate:
// CyberChef upscales the source first, so pixel values differ).
func advancedRotate(img *image.NRGBA, deg float64) *image.NRGBA {
	rad := deg * math.Pi / 180
	cos, sin := math.Cos(rad), math.Sin(rad)
	ow, oh := img.Rect.Dx(), img.Rect.Dy()
	w := int(math.Ceil(math.Abs(float64(ow)*cos)+math.Abs(float64(oh)*sin))) + 1
	h := int(math.Ceil(math.Abs(float64(ow)*sin)+math.Abs(float64(oh)*cos))) + 1
	if w%2 != 0 {
		w++
	}
	if h%2 != 0 {
		h++
	}
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			cx := float64(x) - float64(w)/2 + 0.5
			cy := float64(y) - float64(h)/2 + 0.5
			sx := cos*cx + sin*cy + float64(ow)/2 - 0.5
			sy := -sin*cx + cos*cy + float64(oh)/2 - 0.5
			ix, iy := int(math.Round(sx)), int(math.Round(sy))
			if ix >= 0 && ix < ow && iy >= 0 && iy < oh {
				si := img.PixOffset(ix, iy)
				di := dst.PixOffset(x, y)
				copy(dst.Pix[di:di+4], img.Pix[si:si+4])
			}
		}
	}
	return dst
}
