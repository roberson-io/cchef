package ops

import (
	"fmt"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/amfcodec"
	"github.com/roberson-io/cchef/internal/jsonval"
)

func init() {
	core.Register(AMFDecode{})
	core.Register(AMFEncode{})
}

// amfFormats are the supported AMF versions. AMF3 is the default (index 1),
// matching CyberChef.
var amfFormats = []string{"AMF0", "AMF3"}

// AMFDecode deserializes Action Message Format binary data into JSON.
type AMFDecode struct{}

// Meta returns the operation metadata.
func (AMFDecode) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "AMF Decode",
		Module:      "Encodings",
		Description: "Action Message Format (AMF) is a binary format used to serialize object graphs. This operation deserializes AMF data into JSON.",
		InfoURL:     "https://wikipedia.org/wiki/Action_Message_Format",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeJSON,
	}
}

// Args returns the argument definitions.
func (AMFDecode) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Format", Type: core.ArgOption, Value: amfFormats, DefaultIndex: 1},
	}
}

// Run decodes AMF bytes into a JSON string.
func (AMFDecode) Run(in *core.Dish, args []any) (*core.Dish, error) {
	r := amfcodec.NewReader(in.Bytes())
	var val any
	var err error
	if args[0].(string) == "AMF0" {
		var refs []any
		val, err = amfcodec.Decode0(r, &refs)
	} else {
		val, err = amfcodec.Decode3(r, amfcodec.NewTables3())
	}
	if err != nil {
		return nil, fmt.Errorf("AMF decode: %w", err)
	}
	return core.NewDish([]byte(jsonval.Stringify(val, 0)), core.TypeJSON), nil
}

// AMFEncode serializes JSON into Action Message Format binary data.
type AMFEncode struct{}

// Meta returns the operation metadata.
func (AMFEncode) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "AMF Encode",
		Module:      "Encodings",
		Description: "Action Message Format (AMF) is a binary format used to serialize object graphs. This operation serializes JSON into AMF data.",
		InfoURL:     "https://wikipedia.org/wiki/Action_Message_Format",
		InputType:   core.TypeJSON,
		OutputType:  core.TypeArrayBuffer,
	}
}

// Args returns the argument definitions.
func (AMFEncode) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Format", Type: core.ArgOption, Value: amfFormats, DefaultIndex: 1},
	}
}

// Run encodes a JSON string into AMF bytes.
func (AMFEncode) Run(in *core.Dish, args []any) (*core.Dish, error) {
	val, err := amfcodec.ParseJSON(in.Bytes())
	if err != nil {
		return nil, fmt.Errorf("AMF encode: parse JSON input: %w", err)
	}
	w := &amfcodec.Writer{}
	if args[0].(string) == "AMF0" {
		err = amfcodec.Encode0(w, val)
	} else {
		err = amfcodec.Encode3(w, val)
	}
	if err == nil {
		// A length the format cannot express is recorded on the writer rather
		// than returned from every call that writes one.
		err = w.Err()
	}
	if err != nil {
		return nil, fmt.Errorf("AMF encode: %w", err)
	}
	return core.NewDish(w.Bytes(), core.TypeArrayBuffer), nil
}
