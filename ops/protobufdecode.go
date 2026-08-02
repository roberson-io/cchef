package ops

import (
	"strings"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/jsonval"
	"github.com/roberson-io/cchef/internal/protobuf"
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

// Run decodes the protobuf data.
func (ProtobufDecode) Run(in *core.Dish, args []any) (*core.Dish, error) {
	schema := args[0].(string)
	showUnknown := args[1].(bool)
	showTypes := args[2].(bool)
	data := in.Bytes()

	parser := protobuf.NewParser(data)
	raw, err := parser.Parse()
	if err != nil {
		return nil, err
	}

	rawForOutput := raw
	if showTypes {
		rawForOutput = protobuf.ShowRawTypes(raw, parser.FieldTypes)
	}

	if strings.TrimSpace(schema) == "" {
		out, err := jsonval.MarshalNoEscape(rawForOutput)
		if err != nil {
			return nil, err
		}
		return core.NewDish(out, core.TypeJSON), nil
	}

	out, err := protobuf.SchemaDecode(data, rawForOutput, schema, showUnknown, showTypes)
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeJSON), nil
}
