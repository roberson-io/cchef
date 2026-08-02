package ops

import (
	"image"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/jimp"
	"github.com/roberson-io/cchef/internal/jsnum"
)

func init() {
	core.Register(AddTextToImage{})
}

// AddTextToImage draws text onto an image. Ported from CyberChef's Add Text To
// Image, which renders with Jimp from the 72px Roboto bitmap-font atlases; cchef
// embeds those same atlases and reproduces Jimp's glyph blitting, bicubic
// downscale and compositing, so the result is pixel-identical for lossless
// formats.
type AddTextToImage struct{}

// Meta returns the operation metadata.
func (AddTextToImage) Meta() core.OpMeta {
	return core.OpMeta{
		Name:   "Add Text To Image",
		Module: "Image",
		Description: "Adds text onto an image.<br><br>Text can be horizontally or vertically aligned, " +
			"or the position can be manually specified.<br>Variants of the Roboto font face are " +
			"available in any size or colour.",
		InputType:  core.TypeArrayBuffer,
		OutputType: core.TypeArrayBuffer,
	}
}

var (
	addTextSizeMin             = float64(8)
	channelMin, channelMax     = float64(0), float64(255)
	addTextFontFaceOptionNames = []string{"Roboto", "Roboto Black", "Roboto Mono", "Roboto Slab"}
)

// Args returns the argument definitions.
func (AddTextToImage) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Text", Type: core.ArgString, Value: ""},
		{Name: "Horizontal align", Type: core.ArgOption, Value: []string{"None", "Left", "Center", "Right"}},
		{Name: "Vertical align", Type: core.ArgOption, Value: []string{"None", "Top", "Middle", "Bottom"}},
		{Name: "X position", Type: core.ArgNumber, Integer: true, Value: float64(0)},
		{Name: "Y position", Type: core.ArgNumber, Integer: true, Value: float64(0)},
		{Name: "Size", Type: core.ArgNumber, Integer: true, Value: float64(32), Min: &addTextSizeMin},
		{Name: "Font face", Type: core.ArgOption, Value: addTextFontFaceOptionNames},
		{Name: "Red", Type: core.ArgNumber, Integer: true, Value: float64(255), Min: &channelMin, Max: &channelMax},
		{Name: "Green", Type: core.ArgNumber, Integer: true, Value: float64(255), Min: &channelMin, Max: &channelMax},
		{Name: "Blue", Type: core.ArgNumber, Integer: true, Value: float64(255), Min: &channelMin, Max: &channelMax},
		{Name: "Alpha", Type: core.ArgNumber, Integer: true, Value: float64(255), Min: &channelMin, Max: &channelMax},
	}
}

// addTextParams holds the operation's coerced arguments.
type addTextParams struct {
	text           string
	hAlign, vAlign string
	xPos, yPos     float64
	size           float64
	face           string
	r, g, b, a     float64
}

// Run draws the text onto the image.
func (AddTextToImage) Run(in *core.Dish, args []any) (*core.Dish, error) {
	p := addTextParams{
		text: args[0].(string), hAlign: args[1].(string), vAlign: args[2].(string),
		xPos: args[3].(float64), yPos: args[4].(float64), size: args[5].(float64),
		face: args[6].(string),
		r:    args[7].(float64), g: args[8].(float64), b: args[9].(float64), a: args[10].(float64),
	}
	out, err := jimp.TransformE(in.Bytes(), "Invalid file type.", func(img *image.NRGBA) (*image.NRGBA, error) {
		return drawTextOnImage(img, p)
	})
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeByteArray), nil
}

// drawTextOnImage renders the text to its own bitmap, scales it to the requested
// size and composites it onto img.
func drawTextOnImage(img *image.NRGBA, p addTextParams) (*image.NRGBA, error) {
	font, err := loadBMFont(p.face)
	if err != nil {
		return nil, err
	}
	font = recolourBMFont(font, p.r, p.g, p.b, p.a)

	textImage := image.NewNRGBA(image.Rect(0, 0, bmMeasureText(font, p.text), bmMeasureTextHeight(font, p.text)))
	bmPrint(textImage, font, 0, 0, p.text)

	// CyberChef switches strategy on the point size, not the scale factor; with
	// a minimum size of 8 the bilinear branch is unreachable through the CLI.
	if p.size != 1 {
		mode := "bilinearInterpolation"
		if p.size > 1 {
			mode = "bicubicInterpolation"
		}
		textImage = jimp.Scale(textImage, p.size/72, mode)
	}

	x, y := alignTextPosition(img, textImage, p)
	jimp.Blit(img, textImage, x, y)
	return img, nil
}

// alignTextPosition resolves the alignment options into blit coordinates, which
// Jimp rounds. The rounding is JavaScript's (halves go to +∞), which differs
// from Go's for a negative half — reachable when the text is wider than the
// image and centred.
func alignTextPosition(img, textImage *image.NRGBA, p addTextParams) (int, int) {
	x, y := p.xPos, p.yPos
	switch p.hAlign {
	case "Left":
		x = 0
	case "Center":
		x = float64(img.Rect.Dx())/2 - float64(textImage.Rect.Dx())/2
	case "Right":
		x = float64(img.Rect.Dx() - textImage.Rect.Dx())
	}
	switch p.vAlign {
	case "Top":
		y = 0
	case "Middle":
		y = float64(img.Rect.Dy())/2 - float64(textImage.Rect.Dy())/2
	case "Bottom":
		y = float64(img.Rect.Dy() - textImage.Rect.Dy())
	}
	return jsnum.Round(x), jsnum.Round(y)
}

// recolourBMFont returns a copy of font with its atlas pages tinted, applying
// CyberChef's per-channel `value - (255 - wanted)` with a floor of 0. The cached
// font is shared between runs, so the pages must be copied rather than tinted in
// place.
func recolourBMFont(font *bmFont, r, g, b, a float64) *bmFont {
	wanted := [4]float64{r, g, b, a}
	out := *font
	out.Pages = make([]*image.NRGBA, len(font.Pages))
	for i, page := range font.Pages {
		tinted := image.NewNRGBA(page.Rect)
		for j := 0; j < len(page.Pix); j += 4 {
			for ch := range 4 {
				if v := float64(page.Pix[j+ch]) - (255 - wanted[ch]); v > 0 {
					tinted.Pix[j+ch] = byte(v)
				}
			}
		}
		out.Pages[i] = tinted
	}
	return &out
}
