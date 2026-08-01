package ops

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"math"
	"strconv"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(GenerateImage{})
}

// GenerateImage builds an image whose pixels come from the input bytes. Ported
// from CyberChef's Generate Image (pixel-exact).
type GenerateImage struct{}

// Meta returns the operation metadata.
func (GenerateImage) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Generate Image",
		Module:      "Image",
		Description: "Generates an image using the input as pixel values.",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeArrayBuffer,
	}
}

const (
	maxPixelScaleFactor = 64
	maxPixelsPerRow     = 2048
)

var (
	genScaleMin, genScaleMax = float64(1), float64(maxPixelScaleFactor)
	genRowMin, genRowMax     = float64(1), float64(maxPixelsPerRow)
)

// Args returns the argument definitions.
func (GenerateImage) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Mode", Type: core.ArgOption, Value: []string{"Greyscale", "RG", "RGB", "RGBA", "Bits"}},
		{Name: "Pixel Scale Factor", Type: core.ArgNumber, Integer: true, Value: float64(8), Min: &genScaleMin, Max: &genScaleMax},
		{Name: "Pixels per row", Type: core.ArgNumber, Integer: true, Value: float64(64), Min: &genRowMin, Max: &genRowMax},
	}
}

// bytesPerPixelMode maps a mode to how many input bytes make one pixel.
var bytesPerPixelMode = map[string]float64{
	"Greyscale": 1, "RG": 2, "RGB": 3, "RGBA": 4, "Bits": 1.0 / 8,
}

// Run generates the image.
func (GenerateImage) Run(in *core.Dish, args []any) (*core.Dish, error) {
	mode := args[0].(string)
	scale := int(args[1].(float64))
	width := int(args[2].(float64))
	data := in.Bytes()

	bpp, ok := bytesPerPixelMode[mode]
	if !ok {
		return nil, errors.New("Unsupported Mode: (" + mode + ")") //nolint:staticcheck,revive // CyberChef verbatim
	}
	if bpp >= 1 && len(data)%int(bpp) != 0 {
		//nolint:staticcheck,revive // CyberChef verbatim OperationError text
		return nil, errors.New("Number of bytes is not a divisor of " + strconv.Itoa(int(bpp)))
	}

	height := int(math.Ceil(float64(len(data)) / bpp / float64(width)))
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	if mode == "Bits" {
		generateBits(img, data, width)
	} else {
		generatePixels(img, data, width, mode)
	}

	if scale != 1 {
		img = jimpScaleToFit(img, float64(width*scale), float64(height*scale), "nearestNeighbor")
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return core.NewDish(buf.Bytes(), core.TypeByteArray), nil
}

// generateBits renders each input byte as 8 black/white pixels (bit 0 -> white).
func generateBits(img *image.NRGBA, data []byte, width int) {
	index := 0
	for _, b := range data {
		for k := range 8 {
			value := byte(0xff)
			if (b>>(7-k))&1 == 1 {
				value = 0x00
			}
			setGenPixel(img, index%width, index/width, value, value, value, 0xff)
			index++
		}
	}
}

// generatePixels reads bytesPerPixel bytes per pixel for the non-Bits modes.
func generatePixels(img *image.NRGBA, data []byte, width int, mode string) {
	i := 0
	for i < len(data) {
		index := i / len(modeStride(mode))
		r, g, b, a := byte(0), byte(0), byte(0), byte(0xff)
		for _, ch := range modeStride(mode) {
			v := data[i]
			i++
			switch ch {
			case 'r':
				r = v
			case 'g':
				g = v
			case 'b':
				b = v
			case 'a':
				a = v
			}
		}
		if mode == "Greyscale" {
			g, b = r, r
		}
		setGenPixel(img, index%width, index/width, r, g, b, a)
	}
}

// modeStride returns the channel order consumed per pixel for a mode.
func modeStride(mode string) string {
	switch mode {
	case "Greyscale":
		return "r"
	case "RG":
		return "rg"
	case "RGB":
		return "rgb"
	default: // RGBA
		return "rgba"
	}
}

// setGenPixel writes one pixel. The image is sized from the input length, so
// callers never address outside it.
func setGenPixel(img *image.NRGBA, x, y int, r, g, b, a byte) {
	idx := img.PixOffset(x, y)
	img.Pix[idx], img.Pix[idx+1], img.Pix[idx+2], img.Pix[idx+3] = r, g, b, a
}
