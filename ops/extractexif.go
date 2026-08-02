package ops

import (
	"errors"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/exif"
)

func init() {
	core.Register(ExtractEXIF{})
}

// ExtractEXIF extracts EXIF metadata from an image.
type ExtractEXIF struct{}

// Meta returns the operation metadata.
func (ExtractEXIF) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Extract EXIF",
		Module:      "Image",
		Description: "Extracts EXIF data from an image. EXIF data is metadata embedded in images (JPEG, JPG, TIFF) and audio files.",
		InfoURL:     "https://wikipedia.org/wiki/Exif",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ExtractEXIF) Args() []core.ArgDef { return nil }

// Run extracts and formats the EXIF tags.
func (ExtractEXIF) Run(in *core.Dish, _ []any) (*core.Dish, error) {
	store, err := exif.Parse(in.Bytes())
	if err != nil {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return nil, errors.New("Could not extract EXIF data from image: Error: " + err.Error())
	}

	n := len(store.Order)
	result := "Found " + strconv.Itoa(n) + " tags.\n"
	if n > 0 {
		lines := make([]string, n)
		for i, name := range store.Order {
			v, _ := store.Get(name)
			lines[i] = name + ": " + exif.ValueString(v)
		}
		result += "\n" + strings.Join(lines, "\n")
	}
	return core.NewDish([]byte(result), core.TypeString), nil
}
