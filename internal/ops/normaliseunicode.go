package ops

import (
	"errors"

	"golang.org/x/text/unicode/norm"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(NormaliseUnicode{})
}

// normaliseForms are the Unicode Normalisation Forms, in CyberChef's order.
var normaliseForms = []string{"NFD", "NFC", "NFKD", "NFKC"}

// NormaliseUnicode transforms text to one of the Unicode Normalisation Forms.
type NormaliseUnicode struct{}

// Meta returns the operation metadata.
func (NormaliseUnicode) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Normalise Unicode",
		Module:      "Encodings",
		Description: "Transform Unicode characters to one of the Normalisation Forms",
		InfoURL:     "https://wikipedia.org/wiki/Unicode_equivalence#Normal_forms",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (NormaliseUnicode) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Normal Form", Type: core.ArgOption, Value: normaliseForms},
	}
}

// Run normalises the input to the selected form.
func (NormaliseUnicode) Run(in *core.Dish, args []any) (*core.Dish, error) {
	var form norm.Form
	switch args[0].(string) {
	case "NFD":
		form = norm.NFD
	case "NFC":
		form = norm.NFC
	case "NFKD":
		form = norm.NFKD
	case "NFKC":
		form = norm.NFKC
	default:
		return nil, errors.New("Unknown Normalisation Form") //nolint:staticcheck // verbatim CyberChef OperationError text
	}
	return core.NewDish(form.Bytes(in.Bytes()), core.TypeString), nil
}
