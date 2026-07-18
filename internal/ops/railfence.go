package ops

import (
	"errors"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(RailFenceEncode{})
	core.Register(RailFenceDecode{})
}

// railFenceErrOffset is CyberChef's verbatim offset-validation message.
//
//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
var railFenceErrOffset = errors.New("Offset has to be a positive integer")

// railFenceValidate checks the key and offset against the input length, using
// CyberChef's verbatim error text (lenName is "plain text" or "cipher").
func railFenceValidate(key, offset, length int, lenName string) error {
	if key < 2 {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return errors.New("Key has to be bigger than 2")
	}
	if key > length {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return errors.New("Key should be smaller than the " + lenName + "'s length")
	}
	if offset < 0 {
		return railFenceErrOffset
	}
	return nil
}

// railFenceArgs are the shared Key/Offset argument definitions.
func railFenceArgs() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Key", Type: core.ArgNumber, Value: float64(2)},
		{Name: "Offset", Type: core.ArgNumber, Value: float64(0)},
	}
}

// RailFenceEncode encodes with the Rail Fence cipher.
type RailFenceEncode struct{}

// Meta returns the operation metadata.
func (RailFenceEncode) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Rail Fence Cipher Encode",
		Module:      "Ciphers",
		Description: "Encodes Strings using the Rail fence Cipher provided a key and an offset",
		InfoURL:     "https://wikipedia.org/wiki/Rail_fence_cipher",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (RailFenceEncode) Args() []core.ArgDef { return railFenceArgs() }

// Run performs the encoding. Ported from CyberChef RailFenceCipherEncode.mjs.
func (RailFenceEncode) Run(in *core.Dish, args []any) (*core.Dish, error) {
	key, offset := int(args[0].(float64)), int(args[1].(float64))
	text := []rune(in.String())
	if err := railFenceValidate(key, offset, len(text), "plain text"); err != nil {
		return nil, err
	}
	cycle := (key - 1) * 2
	rows := make([][]rune, key)
	for pos, ch := range text {
		d := cycle/2 - (pos+offset)%cycle
		if d < 0 {
			d = -d
		}
		rowIdx := key - 1 - d
		rows[rowIdx] = append(rows[rowIdx], ch)
	}
	out := make([]rune, 0, len(text))
	for _, row := range rows {
		out = append(out, row...)
	}
	return core.NewDish([]byte(string(out)), core.TypeString), nil
}

// RailFenceDecode decodes Rail Fence cipher text.
type RailFenceDecode struct{}

// Meta returns the operation metadata.
func (RailFenceDecode) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Rail Fence Cipher Decode",
		Module:      "Ciphers",
		Description: "Decodes Strings that were created using the Rail fence Cipher provided a key and an offset",
		InfoURL:     "https://wikipedia.org/wiki/Rail_fence_cipher",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (RailFenceDecode) Args() []core.ArgDef { return railFenceArgs() }

// Run performs the decoding. Ported from CyberChef RailFenceCipherDecode.mjs.
func (RailFenceDecode) Run(in *core.Dish, args []any) (*core.Dish, error) {
	key, offset := int(args[0].(float64)), int(args[1].(float64))
	cipher := []rune(in.String())
	if err := railFenceValidate(key, offset, len(cipher), "cipher"); err != nil {
		return nil, err
	}
	cycle := (key - 1) * 2
	plain := make([]rune, len(cipher))
	j := 0
	for y := range key {
		for x := range cipher {
			if (y+x+offset)%cycle == 0 || (y-x-offset)%cycle == 0 {
				plain[x] = cipher[j]
				j++
			}
		}
	}
	return core.NewDish([]byte(string(plain)), core.TypeString), nil
}
