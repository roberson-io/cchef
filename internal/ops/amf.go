package ops

import (
	"bytes"
	"encoding/json"
	"fmt"

	amf "github.com/elobuff/goamf"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(AMFDecode{})
	core.Register(AMFEncode{})
}

// amfFormats are the supported AMF versions. AMF3 is the default (index 1),
// matching CyberChef.
var amfFormats = []string{"AMF0", "AMF3"}

// amfVersion maps a format name to the library's version constant.
func amfVersion(format string) amf.Version {
	if format == "AMF0" {
		return amf.AMF0
	}
	return amf.AMF3
}

// toAMFValue recursively converts a value decoded by encoding/json into the
// concrete amf.Object / amf.Array types the encoder requires (it rejects plain
// map[string]interface{} and []interface{}).
func toAMFValue(v any) any {
	switch x := v.(type) {
	case map[string]any:
		o := amf.Object{}
		for k, val := range x {
			o[k] = toAMFValue(val)
		}
		return o
	case []any:
		a := make(amf.Array, len(x))
		for i, val := range x {
			a[i] = toAMFValue(val)
		}
		return a
	default:
		return v
	}
}

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
	dec := &amf.Decoder{}
	val, err := dec.Decode(bytes.NewReader(in.Bytes()), amfVersion(args[0].(string)))
	if err != nil {
		return nil, fmt.Errorf("AMF decode: %w", err)
	}
	out, err := json.Marshal(val)
	if err != nil {
		return nil, fmt.Errorf("AMF decode: marshal JSON: %w", err)
	}
	return core.NewDish(out, core.TypeJSON), nil
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
	var val any
	if err := json.Unmarshal(in.Bytes(), &val); err != nil {
		return nil, fmt.Errorf("AMF encode: parse JSON input: %w", err)
	}
	var buf bytes.Buffer
	enc := &amf.Encoder{}
	if _, err := enc.Encode(&buf, toAMFValue(val), amfVersion(args[0].(string))); err != nil {
		return nil, fmt.Errorf("AMF encode: %w", err)
	}
	return core.NewDish(buf.Bytes(), core.TypeArrayBuffer), nil
}
