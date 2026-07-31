package ops

import (
	"errors"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(RenderPDF{})
}

// RenderPDF validates that the input is a PDF and passes the bytes through.
// Ported from CyberChef RenderPDF.mjs, whose browser <iframe> PDF preview is
// dropped; cchef offers Raw or base64 data-URI output.
type RenderPDF struct{}

// Meta returns the operation metadata.
func (RenderPDF) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Render PDF",
		Module:      "File",
		Description: "Validates that the input is a PDF and outputs it. Supports Raw and Base64 input formats.",
		InputType:   core.TypeString,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (RenderPDF) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Input format", Type: core.ArgOption, Value: []string{"Base64", "Raw"}},
	}
}

// pdfMagic is the "%PDF" signature checked at the start of a PDF file.
var pdfMagic = []byte{0x25, 0x50, 0x44, 0x46}

// Run validates the PDF and passes its bytes through.
func (RenderPDF) Run(in *core.Dish, args []any) (*core.Dish, error) {
	inputFormat := args[0].(string)

	if len(in.Bytes()) == 0 {
		return core.NewDish(nil, core.TypeByteArray), nil
	}

	var data []byte
	if inputFormat == "Base64" {
		data, _ = fromBase64(in.String(), stdBase64Alphabet, true, false)
	} else {
		data = in.Bytes()
	}

	if len(data) < len(pdfMagic) ||
		data[0] != pdfMagic[0] || data[1] != pdfMagic[1] ||
		data[2] != pdfMagic[2] || data[3] != pdfMagic[3] {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return nil, errors.New("Input does not appear to be a PDF file.")
	}

	return core.NewDish(data, core.TypeByteArray), nil
}
