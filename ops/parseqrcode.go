package ops

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(ParseQRCode{})
}

// ParseQRCode reads a QR code out of an image. Ported from CyberChef's Parse QR
// Code, which preprocesses with Jimp and reads with jsQR.
type ParseQRCode struct{}

// Meta returns the operation metadata.
func (ParseQRCode) Meta() core.OpMeta {
	return core.OpMeta{
		Name:   "Parse QR Code",
		Module: "Image",
		Description: "Reads an image file and attempts to detect and read a Quick Response (QR) " +
			"code from the image. Normalise Image attempts to normalise the image before " +
			"parsing it to improve detection of a QR code.",
		InputType:  core.TypeArrayBuffer,
		OutputType: core.TypeString,
	}
}

// Args returns the argument definitions.
func (ParseQRCode) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Normalise image", Type: core.ArgBoolean, Value: false},
	}
}

// The quality the image is re-encoded at before it is read, which matches the
// default of the library CyberChef preprocesses with.
const qrReadJPEGQuality = 100

// Run reads the code.
func (ParseQRCode) Run(in *core.Dish, args []any) (*core.Dish, error) {
	normalise := args[0].(bool)

	img, _, err := decodeImageNRGBA(in.Bytes(), "Invalid file type.")
	if err != nil {
		return nil, err
	}
	if normalise {
		jimpGreyscale(img)
		jimpNormalize(img)
	}

	// Transparent pixels become opaque white, and everything else opaque at the
	// colour it already has, since the reader cannot see through them.
	for i := 0; i < len(img.Pix); i += 4 {
		if img.Pix[i+3] == 0 {
			img.Pix[i], img.Pix[i+1], img.Pix[i+2] = 0xFF, 0xFF, 0xFF
		}
		img.Pix[i+3] = 0xFF
	}

	// CyberChef re-encodes the image before reading it, so the reader sees the
	// pixels a lossy round trip leaves behind.
	flattened := qrFlattenThroughJPEG(img)

	text, ok := qrRead(flattened.Pix, flattened.Bounds().Dx(), flattened.Bounds().Dy())
	if !ok {
		return nil, errors.New("could not read a QR code from the image")
	}
	return core.NewDish([]byte(text), core.TypeString), nil
}

// qrFlattenThroughJPEG puts the image through the same lossy round trip the
// preprocessing does. Neither step can fail: the image is already decoded, and
// the buffer it is written to never errors.
func qrFlattenThroughJPEG(img *image.NRGBA) *image.NRGBA {
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, &jpeg.Options{Quality: qrReadJPEGQuality})
	decoded, _ := jpeg.Decode(&buf)
	return toNRGBA(decoded)
}
