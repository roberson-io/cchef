package ops

import (
	"strings"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/jsonval"
)

func init() {
	core.Register(ProtobufDecode{})
}

// ProtobufDecode decodes protobuf wire data to JSON, with or without a schema.
type ProtobufDecode struct{}

// Meta returns the operation metadata.
func (ProtobufDecode) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Protobuf Decode",
		Module:      "Default",
		Description: "Decodes any Protobuf encoded data to a JSON representation of the data using the field number as the field key. If a .proto schema is defined, the encoded data is decoded with reference to the schema; only one message instance is decoded.",
		InfoURL:     "https://wikipedia.org/wiki/Protocol_Buffers",
		InputType:   core.TypeByteArray,
		OutputType:  core.TypeJSON,
	}
}

// Args returns the argument definitions.
func (ProtobufDecode) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Schema (.proto text)", Type: core.ArgString, Value: ""},
		{Name: "Show Unknown Fields", Type: core.ArgBoolean, Value: false},
		{Name: "Show Types", Type: core.ArgBoolean, Value: false},
	}
}

// Run decodes the protobuf data. Ported from CyberChef ProtobufDecode.mjs /
// lib/Protobuf.mjs.
func (ProtobufDecode) Run(in *core.Dish, args []any) (*core.Dish, error) {
	schema := args[0].(string)
	showUnknown := args[1].(bool)
	showTypes := args[2].(bool)
	data := in.Bytes()

	parser := newProtobufParser(data)
	raw, err := parser.parse()
	if err != nil {
		return nil, err
	}

	rawForOutput := raw
	if showTypes {
		rawForOutput = showRawTypes(raw, parser.fieldTypes)
	}

	if strings.TrimSpace(schema) == "" {
		out, err := jsonval.MarshalNoEscape(rawForOutput)
		if err != nil {
			return nil, err
		}
		return core.NewDish(out, core.TypeJSON), nil
	}

	out, err := protobufSchemaDecode(data, rawForOutput, schema, showUnknown, showTypes)
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeJSON), nil
}
