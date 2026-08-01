package ops

import (
	"errors"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/codepage"
	"github.com/roberson-io/cchef/internal/jsonval"
)

func init() {
	core.Register(DecodeText{})
	core.Register(EncodeText{})
	core.Register(TextEncodingBruteForce{})
}

// cpNames lists all 152 CyberChef charset option values, in CHR_ENC_CODE_PAGES
// order.
var cpNames = func() []string {
	names := make([]string, len(codepage.Charsets))
	for i, c := range codepage.Charsets {
		names[i] = c.Name
	}
	return names
}()

// cpByName maps a charset display name to its codepage number.
var cpByName = func() map[string]int {
	m := make(map[string]int, len(codepage.Charsets))
	for _, c := range codepage.Charsets {
		m[c.Name] = c.CP
	}
	return m
}()

// DecodeText decodes bytes from a chosen character encoding into text.
type DecodeText struct{}

// Meta returns the operation metadata.
func (DecodeText) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Decode text",
		Module:      "Encodings",
		Description: "Decodes text from the chosen character encoding.",
		InfoURL:     "https://wikipedia.org/wiki/Character_encoding",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (DecodeText) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Encoding", Type: core.ArgOption, Value: cpNames},
	}
}

// Run decodes the input bytes using the chosen encoding.
func (DecodeText) Run(in *core.Dish, args []any) (*core.Dish, error) {
	cp, ok := cpByName[args[0].(string)]
	if !ok {
		return nil, errors.New("Invalid encoding") //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}
	out, err := codepage.Decode(cp, in.Bytes())
	if err != nil {
		return nil, err
	}
	return core.NewDish([]byte(out), core.TypeString), nil
}

// EncodeText encodes text into a chosen character encoding.
type EncodeText struct{}

// Meta returns the operation metadata.
func (EncodeText) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Encode text",
		Module:      "Encodings",
		Description: "Encodes text into the chosen character encoding.",
		InfoURL:     "https://wikipedia.org/wiki/Character_encoding",
		InputType:   core.TypeString,
		OutputType:  core.TypeArrayBuffer,
	}
}

// Args returns the argument definitions.
func (EncodeText) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Encoding", Type: core.ArgOption, Value: cpNames},
	}
}

// Run encodes the input text using the chosen encoding.
func (EncodeText) Run(in *core.Dish, args []any) (*core.Dish, error) {
	cp, ok := cpByName[args[0].(string)]
	if !ok {
		return nil, errors.New("Invalid encoding") //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}
	out, err := codepage.Encode(cp, in.String())
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeArrayBuffer), nil
}

// TextEncodingBruteForce enumerates every supported text encoding for the input.
type TextEncodingBruteForce struct{}

// Meta returns the operation metadata.
func (TextEncodingBruteForce) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Text Encoding Brute Force",
		Module:      "Encodings",
		Description: "Enumerates all supported text encodings for the input, allowing you to quickly spot the correct one.",
		InfoURL:     "https://wikipedia.org/wiki/Character_encoding",
		InputType:   core.TypeString,
		OutputType:  core.TypeJSON,
	}
}

// Args returns the argument definitions.
func (TextEncodingBruteForce) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Mode", Type: core.ArgOption, Value: []string{"Encode", "Decode"}},
	}
}

// Run encodes or decodes the input in every charset, returning a JSON object
// mapping each charset name to its result (or "Could not decode." on error).
func (TextEncodingBruteForce) Run(in *core.Dish, args []any) (*core.Dish, error) {
	decode := args[0].(string) == "Decode"
	obj := make(jsonval.Object, 0, len(codepage.Charsets))
	for _, c := range codepage.Charsets {
		var val string
		if decode {
			if s, err := codepage.Decode(c.CP, in.Bytes()); err != nil {
				val = "Could not decode."
			} else {
				val = s
			}
		} else {
			if b, err := codepage.Encode(c.CP, in.String()); err != nil {
				val = "Could not decode."
			} else {
				val = mimeByteArrayToUtf8(b)
			}
		}
		obj = append(obj, jsonval.Pair{K: c.Name, V: val})
	}
	return core.NewDish([]byte(jsonval.Stringify(obj, 4)), core.TypeJSON), nil
}
