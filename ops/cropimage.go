package ops

import (
	"errors"
	"image"
	"math"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/jimp"
)

func init() {
	core.Register(CropImage{})
}

// CropImage crops an image to a region, or auto-crops a uniform border. Ported
// from CyberChef's Crop Image (Jimp's crop/autocrop).
type CropImage struct{}

// Meta returns the operation metadata.
func (CropImage) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Crop Image",
		Module:      "Image",
		Description: "Crops an image to the specified region, or automatically crops edges.",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeArrayBuffer,
	}
}

var (
	cropZeroMin = float64(0)
	cropOneMin  = float64(1)
	cropTolMax  = float64(100)
)

// Args returns the argument definitions.
func (CropImage) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "X Position", Type: core.ArgNumber, Integer: true, Value: float64(0), Min: &cropZeroMin},
		{Name: "Y Position", Type: core.ArgNumber, Integer: true, Value: float64(0), Min: &cropZeroMin},
		{Name: "Width", Type: core.ArgNumber, Integer: true, Value: float64(10), Min: &cropOneMin},
		{Name: "Height", Type: core.ArgNumber, Integer: true, Value: float64(10), Min: &cropOneMin},
		{Name: "Autocrop", Type: core.ArgBoolean, Value: false},
		{Name: "Autocrop tolerance (%)", Type: core.ArgNumber, Value: float64(0.02), Min: &cropZeroMin, Max: &cropTolMax},
		{Name: "Only autocrop frames", Type: core.ArgBoolean, Value: true},
		{Name: "Symmetric autocrop", Type: core.ArgBoolean, Value: false},
		{Name: "Autocrop keep border (px)", Type: core.ArgNumber, Integer: true, Value: float64(0), Min: &cropZeroMin},
	}
}

// Run crops the image.
func (CropImage) Run(in *core.Dish, args []any) (*core.Dish, error) {
	out, err := jimp.TransformE(in.Bytes(), "Invalid file type.", func(img *image.NRGBA) (*image.NRGBA, error) {
		var cropped *image.NRGBA
		var cErr error
		if args[4].(bool) {
			cropped, cErr = jimpAutocrop(img, args[5].(float64)/100, args[6].(bool), args[7].(bool), args[8].(float64))
		} else {
			cropped, cErr = jimp.CropExact(img, round(args[0].(float64)), round(args[1].(float64)), round(args[2].(float64)), round(args[3].(float64)))
		}
		if cErr != nil {
			//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
			return nil, errors.New("Error cropping image. (" + cErr.Error() + ")")
		}
		return cropped, nil
	})
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeByteArray), nil
}

// round rounds a float to the nearest integer (JS Math.round semantics for the
// non-negative values used here).
func round(f float64) int { return int(math.Round(f)) }

// rgba holds one pixel's channels as ints.
type rgba struct{ r, g, b, a int }

// pixelRGBA reads the pixel at (x, y) of a dense NRGBA.
func pixelRGBA(img *image.NRGBA, x, y int) rgba {
	i := img.PixOffset(x, y)
	return rgba{int(img.Pix[i]), int(img.Pix[i+1]), int(img.Pix[i+2]), int(img.Pix[i+3])}
}

// colorDiff is Jimp's normalised colour distance (0..1).
func colorDiff(c1, c2 rgba) float64 {
	sq := func(n int) float64 { return float64(n) * float64(n) }
	da := c1.a - c2.a
	const maxVal = 255.0 * 255.0 * 3.0
	return (math.Max(sq(c1.r-c2.r), sq(c1.r-c2.r-da)) +
		math.Max(sq(c1.g-c2.g), sq(c1.g-c2.g-da)) +
		math.Max(sq(c1.b-c2.b), sq(c1.b-c2.b-da))) / maxVal
}

// autocropMinPixels is the minimum pixels kept per side to avoid a 0-size image.
const autocropMinPixels = 1

// rowUniform reports whether every pixel in row y (columns [x0,x1)) matches
// target within tolerance.
func rowUniform(img *image.NRGBA, target rgba, tol float64, y, x0, x1 int) bool {
	for x := x0; x < x1; x++ {
		if colorDiff(target, pixelRGBA(img, x, y)) > tol {
			return false
		}
	}
	return true
}

// colUniform reports whether every pixel in column x (rows [y0,y1)) matches
// target within tolerance.
func colUniform(img *image.NRGBA, target rgba, tol float64, x, y0, y1 int) bool {
	for y := y0; y < y1; y++ {
		if colorDiff(target, pixelRGBA(img, x, y)) > tol {
			return false
		}
	}
	return true
}

// autocropSides counts how many rows/columns of border to remove from each side.
func autocropSides(img *image.NRGBA, target rgba, tol float64) (north, west, south, east int) {
	w, h := img.Rect.Dx(), img.Rect.Dy()
	for y := 0; y < h-autocropMinPixels && rowUniform(img, target, tol, y, 0, w); y++ {
		north++
	}
	for x := 0; x < w-autocropMinPixels && colUniform(img, target, tol, x, north, h); x++ {
		west++
	}
	for y := h - 1; y >= north+autocropMinPixels && rowUniform(img, target, tol, y, 0, w); y-- {
		south++
	}
	for x := w - 1; x >= west+autocropMinPixels && colUniform(img, target, tol, x, north, h); x-- {
		east++
	}
	return north, west, south, east
}

// jimpAutocrop trims a uniform border matching the top-left pixel colour.
// Ported from @jimp/plugin-crop's autocrop.
func jimpAutocrop(img *image.NRGBA, tolerance float64, cropOnlyFrames, cropSymmetric bool, leaveBorder float64) (*image.NRGBA, error) {
	w, h := img.Rect.Dx(), img.Rect.Dy()
	target := pixelRGBA(img, 0, 0)
	north, west, south, east := autocropSides(img, target, tolerance)

	wc := float64(west) - leaveBorder
	ec := float64(east) - leaveBorder
	nc := float64(north) - leaveBorder
	sc := float64(south) - leaveBorder
	if cropSymmetric {
		horizontal := math.Min(ec, wc)
		vertical := math.Min(nc, sc)
		wc, ec, nc, sc = horizontal, horizontal, vertical, vertical
	}
	wc, ec, nc, sc = math.Max(wc, 0), math.Max(ec, 0), math.Max(nc, 0), math.Max(sc, 0)
	remainingW := float64(w) - (wc + ec)
	remainingH := float64(h) - (sc + nc)

	var doCrop bool
	if cropOnlyFrames {
		doCrop = ec != 0 && nc != 0 && wc != 0 && sc != 0
	} else {
		doCrop = ec != 0 || nc != 0 || wc != 0 || sc != 0
	}
	if doCrop {
		return jimp.CropExact(img, round(wc), round(nc), round(remainingW), round(remainingH))
	}
	return img, nil
}
