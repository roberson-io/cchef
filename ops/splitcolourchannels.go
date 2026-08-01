package ops

import (
	"bytes"
	"errors"
	"image"
	"image/png"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(SplitColourChannels{})
}

// SplitColourChannels separates an image into its red, green and blue channels.
// Ported from CyberChef's Split Colour Channels, which zeroes the other two
// channels with Jimp's colour({apply: "…", params: [-255]}). Alpha is kept and
// every channel is written as PNG regardless of the input format, as CyberChef
// does. The three images are pixel-identical to CyberChef.
type SplitColourChannels struct{}

// Meta returns the operation metadata.
func (SplitColourChannels) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Split Colour Channels",
		Module:      "Image",
		Description: "Splits the given image into its red, green and blue colour channels.",
		InfoURL:     "https://wikipedia.org/wiki/Channel_(digital_image)",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeFileList,
	}
}

// Args returns the argument definitions.
func (SplitColourChannels) Args() []core.ArgDef { return nil }

// channelNames are the output file names, indexed by channel offset.
var channelNames = [3]string{"red.png", "green.png", "blue.png"}

// Run splits the image into one file per colour channel.
func (SplitColourChannels) Run(in *core.Dish, _ []any) (*core.Dish, error) {
	src, _, err := decodeImageNRGBA(in.Bytes(), "Invalid file type.")
	if err != nil {
		return nil, err
	}
	files, err := splitChannelFiles(src)
	if err != nil {
		return nil, err
	}
	return core.NewFileListDish(files), nil
}

// splitChannelFiles renders one PNG per colour channel of src.
func splitChannelFiles(src *image.NRGBA) ([]core.NamedFile, error) {
	files := make([]core.NamedFile, 0, len(channelNames))
	for ch, name := range channelNames {
		data, err := encodeChannel(src, ch)
		if err != nil {
			//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
			return nil, errors.New("Could not split " + name[:len(name)-4] + " channel: " + err.Error())
		}
		files = append(files, core.NamedFile{Name: name, Data: data})
	}
	return files, nil
}

// encodeChannel renders src keeping only channel ch (0=R, 1=G, 2=B) and its
// alpha, as PNG.
func encodeChannel(src *image.NRGBA, ch int) ([]byte, error) {
	out := image.NewNRGBA(src.Rect)
	for i := 0; i < len(src.Pix); i += 4 {
		out.Pix[i+ch] = src.Pix[i+ch]
		out.Pix[i+3] = src.Pix[i+3]
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, out); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
