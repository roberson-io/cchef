package ops

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/draw"
	_ "image/gif" // register GIF decoder
	"image/jpeg"
	"image/png"

	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
	_ "golang.org/x/image/webp" // register WEBP decoder (decode only)
)

// Shared helpers for the Multimedia image operations. CyberChef performs these
// over the Jimp library, which decodes to a straight (non-premultiplied) RGBA
// bitmap, applies a pixel transform, then re-encodes to the source format (GIF
// is re-encoded as PNG). cchef reproduces the pixel math exactly on an
// image.NRGBA (also straight RGBA), so lossless formats (PNG/BMP/TIFF) are
// pixel-identical to CyberChef; only the encoded bytes differ. JPEG output is
// re-encoded lossily and is therefore an approximation.

// jimpJPEGQuality matches Jimp's default JPEG encode quality.
const jimpJPEGQuality = 100

// imageTransform runs the common decode -> transform -> encode pipeline shared
// by the Multimedia image operations. transform receives the decoded bitmap and
// returns the (possibly new) bitmap to encode.
func imageTransform(data []byte, invalidMsg string, transform func(*image.NRGBA) *image.NRGBA) ([]byte, error) {
	img, format, err := decodeImageNRGBA(data, invalidMsg)
	if err != nil {
		return nil, err
	}
	return encodeImageNRGBA(transform(img), format)
}

// imageTransformE is like imageTransform but the transform may fail (e.g. an
// out-of-bounds crop).
func imageTransformE(data []byte, invalidMsg string, transform func(*image.NRGBA) (*image.NRGBA, error)) ([]byte, error) {
	img, format, err := decodeImageNRGBA(data, invalidMsg)
	if err != nil {
		return nil, err
	}
	out, err := transform(img)
	if err != nil {
		return nil, err
	}
	return encodeImageNRGBA(out, format)
}

// decodeImageNRGBA validates that data is an image, decodes it to a dense
// straight-RGBA bitmap, and returns the bitmap plus the source format name.
func decodeImageNRGBA(data []byte, invalidMsg string) (*image.NRGBA, string, error) {
	if isImage(data) == "" {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return nil, "", errors.New(invalidMsg)
	}
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return nil, "", fmt.Errorf("Error loading image. (%w)", err)
	}
	return toNRGBA(img), format, nil
}

// toNRGBA returns img as a dense image.NRGBA anchored at the origin.
func toNRGBA(img image.Image) *image.NRGBA {
	b := img.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), img, b.Min, draw.Src)
	return dst
}

// encodeImageNRGBA encodes img back to the source format. GIF input is encoded
// as PNG (Jimp does the same, as it has no GIF encoder).
func encodeImageNRGBA(img *image.NRGBA, format string) ([]byte, error) {
	var buf bytes.Buffer
	var err error
	switch format {
	case "jpeg":
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: jimpJPEGQuality})
	case "bmp":
		err = bmp.Encode(&buf, img)
	case "tiff":
		err = tiff.Encode(&buf, img, nil)
	default: // png, gif -> png
		err = png.Encode(&buf, img)
	}
	if err != nil {
		return nil, fmt.Errorf("error encoding image: %w", err)
	}
	return buf.Bytes(), nil
}
