package ops

import (
	"errors"
	"image"
	"slices"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/jimp"
)

func init() {
	core.Register(ExtractLSB{})
}

// lsbColourOptions is the channel order R, G, B, A shared by the four colour
// pattern arguments.
var lsbColourOptions = []string{"R", "G", "B", "A"}

// ExtractLSB reads a chosen bit of chosen channels from every pixel of an
// image. Ported from CyberChef ExtractLSB.mjs.
type ExtractLSB struct{}

// Meta returns the operation metadata.
func (ExtractLSB) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Extract LSB",
		Module:      "Image",
		Description: "Extracts the Least Significant Bit data from each pixel in an image. This is a common way to hide data in Steganography.",
		InfoURL:     "https://wikipedia.org/wiki/Bit_numbering#Least_significant_bit_in_digital_steganography",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns four colour patterns (the last three may be empty), the pixel
// order, and which bit to read.
func (ExtractLSB) Args() []core.ArgDef {
	withBlank := append([]string{""}, lsbColourOptions...)
	return []core.ArgDef{
		{Name: "Colour Pattern #1", Type: core.ArgOption, Value: lsbColourOptions},
		{Name: "Colour Pattern #2", Type: core.ArgOption, Value: withBlank},
		{Name: "Colour Pattern #3", Type: core.ArgOption, Value: withBlank},
		{Name: "Colour Pattern #4", Type: core.ArgOption, Value: withBlank},
		{Name: "Pixel Order", Type: core.ArgOption, Value: []string{"Row", "Column"}},
		{Name: "Bit", Type: core.ArgNumber, Integer: true, Value: 0},
	}
}

// Run extracts the chosen bits.
func (ExtractLSB) Run(in *core.Dish, args []any) (*core.Dish, error) {
	img, _, err := jimp.Decode(in.Bytes(), "Please enter a valid image file.")
	if err != nil {
		return nil, err
	}

	bit := int(args[5].(float64))
	if bit < 0 || bit > 7 {
		return nil, errors.New("Error: Bit argument must be between 0 and 7") //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}

	var colours []int
	for _, a := range args[:4] {
		if name := a.(string); name != "" {
			colours = append(colours, slices.Index(lsbColourOptions, name))
		}
	}

	bits := lsbWalk(img, colours, bit, args[4].(string) == "Row")
	return core.NewDish(packBits(bits), core.TypeByteArray), nil
}

// lsbWalk collects the chosen bit of the chosen channels from every pixel, in
// row-major or column-major order. The column walk visits whole pixels top to
// bottom; upstream's column arithmetic instead lands on unrelated bytes (and
// off the end of a narrow image), which is logged as a CyberChef bug.
func lsbWalk(img *image.NRGBA, colours []int, bit int, rowMajor bool) []byte {
	width, height := img.Rect.Dx(), img.Rect.Dy()
	out := make([]byte, 0, width*height*len(colours))
	sample := func(x, y int) {
		i := y*img.Stride + x*4
		for _, c := range colours {
			out = append(out, img.Pix[i+c]>>bit&1)
		}
	}
	if rowMajor {
		for y := range height {
			for x := range width {
				sample(x, y)
			}
		}
	} else {
		for x := range width {
			for y := range height {
				sample(x, y)
			}
		}
	}
	return out
}

// packBits packs a bit sequence into bytes, eight to a byte, first bit most
// significant. A short final group keeps its value unshifted, exactly as
// CyberChef's fromBinary parses the trailing digits.
func packBits(bits []byte) []byte {
	out := make([]byte, 0, (len(bits)+7)/8)
	for i := 0; i < len(bits); i += 8 {
		var v byte
		for _, b := range bits[i:min(i+8, len(bits))] {
			v = v<<1 | b
		}
		out = append(out, v)
	}
	return out
}
