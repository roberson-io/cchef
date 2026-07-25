package ops

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"

	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(ConvertImageFormat{})
}

// ConvertImageFormat re-encodes an image to a different format. Ported from
// CyberChef's Convert Image Format.
//
// Reduced fidelity, by design: for lossless targets (PNG/BMP/TIFF) the pixels
// are identical to CyberChef but the encoded bytes differ (different encoders),
// and the PNG filter-type/deflate-level options do not affect the pixels (Go's
// PNG encoder doesn't expose the filter type; the deflate level only changes
// compression). JPEG output is lossy and re-encoded with Go's encoder, so it is
// an approximation.
type ConvertImageFormat struct{}

// Meta returns the operation metadata.
func (ConvertImageFormat) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Convert Image Format",
		Module:      "Image",
		Description: "Converts an image between different formats. Supported formats: JPEG, PNG, BMP and TIFF.",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeArrayBuffer,
	}
}

var (
	jpegQualityMin, jpegQualityMax = float64(1), float64(100)
	deflateMin, deflateMax         = float64(0), float64(9)
)

// Args returns the argument definitions.
func (ConvertImageFormat) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Output Format", Type: core.ArgOption, Value: []string{"JPEG", "PNG", "BMP", "TIFF"}},
		{Name: "JPEG Quality", Type: core.ArgNumber, Value: float64(80), Min: &jpegQualityMin, Max: &jpegQualityMax},
		{Name: "PNG Filter Type", Type: core.ArgOption, Value: []string{"Auto", "None", "Sub", "Up", "Average", "Paeth"}},
		{Name: "PNG Deflate Level", Type: core.ArgNumber, Value: float64(9), Min: &deflateMin, Max: &deflateMax},
	}
}

// Run converts the image to the requested format.
func (ConvertImageFormat) Run(in *core.Dish, args []any) (*core.Dish, error) {
	out, err := convertImage(in.Bytes(), args[0].(string), int(args[1].(float64)), int(args[3].(float64)))
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeByteArray), nil
}

// convertImage decodes data and re-encodes it in the named format.
func convertImage(data []byte, format string, quality, deflate int) ([]byte, error) {
	img, _, err := decodeImageNRGBA(data, "Invalid file format.")
	if err != nil {
		return nil, err
	}
	return encodeConverted(img, format, quality, deflate)
}

// pngDeflateLevel maps a 0-9 deflate level onto Go's png.CompressionLevel.
func pngDeflateLevel(level int) png.CompressionLevel {
	switch {
	case level == 0:
		return png.NoCompression
	case level <= 3:
		return png.BestSpeed
	case level >= 8:
		return png.BestCompression
	default:
		return png.DefaultCompression
	}
}

// encodeConverted encodes img in the named format.
func encodeConverted(img *image.NRGBA, format string, quality, deflate int) ([]byte, error) {
	var buf bytes.Buffer
	var err error
	switch format {
	case "JPEG":
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality})
	case "BMP":
		err = bmp.Encode(&buf, img)
	case "TIFF":
		err = tiff.Encode(&buf, img, nil)
	default: // PNG
		enc := png.Encoder{CompressionLevel: pngDeflateLevel(deflate)}
		err = enc.Encode(&buf, img)
	}
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
