package ops

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(ProtobufEncode{})
}

// ProtobufEncode encodes a JSON object into protobuf wire data using a schema.
type ProtobufEncode struct{}

// Meta returns the operation metadata.
func (ProtobufEncode) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Protobuf Encode",
		Module:      "Default",
		Description: "Encodes a valid JSON object into a protobuf byte array using the input .proto schema.",
		InfoURL:     "https://wikipedia.org/wiki/Protocol_Buffers",
		InputType:   core.TypeJSON,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (ProtobufEncode) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Schema (.proto text)", Type: core.ArgString, Value: ""},
	}
}

// Run encodes the input JSON. Ported from CyberChef ProtobufEncode.mjs /
// lib/Protobuf.mjs encode.
func (ProtobufEncode) Run(in *core.Dish, args []any) (*core.Dish, error) {
	schema := args[0].(string)

	md, err := protobufCompileSchema(schema)
	if err != nil {
		return nil, err
	}
	if md == nil {
		return nil, fmt.Errorf("schema error: schema not defined")
	}

	var val any
	if err := json.Unmarshal(in.Bytes(), &val); err != nil {
		return nil, fmt.Errorf("input error: %w", err)
	}
	out, err := protobufMarshalJSON(md, val)
	if err != nil {
		return nil, fmt.Errorf("input error: %w", err)
	}
	return core.NewDish(out, core.TypeByteArray), nil
}

// protobufMarshalJSON serialises a decoded-JSON object into protobuf wire bytes,
// using the schema's proto field names. Fields present in the input are emitted
// in field-number order (repeated scalars unpacked, matching protobufjs); any
// field present in the input is emitted even when its value is zero, and unknown
// input keys are ignored.
func protobufMarshalJSON(md protoreflect.MessageDescriptor, val any) ([]byte, error) {
	// protobufjs treats a non-object (or null) input as a message with no fields
	// set, producing empty output rather than an error.
	obj, ok := val.(map[string]any)
	if !ok {
		return nil, nil
	}
	fields := md.Fields()
	var out []byte
	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)
		v, present := obj[string(fd.Name())]
		if !present || v == nil {
			continue
		}
		var err error
		out, err = protobufAppendJSONField(out, fd, v)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func protobufAppendJSONField(out []byte, fd protoreflect.FieldDescriptor, v any) ([]byte, error) {
	switch {
	case fd.IsList():
		arr, ok := v.([]any)
		if !ok {
			return nil, fmt.Errorf("field %s: expected an array", fd.Name())
		}
		for _, item := range arr {
			var err error
			if out, err = protobufAppendJSONScalar(out, fd, item); err != nil {
				return nil, err
			}
		}
	case fd.IsMap():
		m, ok := v.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("field %s: expected an object", fd.Name())
		}
		for mk, mv := range m {
			entry, err := protobufAppendJSONScalar(nil, fd.MapKey(), mk)
			if err != nil {
				return nil, err
			}
			if entry, err = protobufAppendJSONScalar(entry, fd.MapValue(), mv); err != nil {
				return nil, err
			}
			out = protowire.AppendTag(out, fd.Number(), protowire.BytesType)
			out = protowire.AppendBytes(out, entry)
		}
	default:
		return protobufAppendJSONScalar(out, fd, v)
	}
	return out, nil
}

// protobufAppendJSONScalar appends a single (non-list) value, recursing into
// nested messages by their JSON representation.
func protobufAppendJSONScalar(out []byte, fd protoreflect.FieldDescriptor, v any) ([]byte, error) {
	if fd.Kind() == protoreflect.MessageKind || fd.Kind() == protoreflect.GroupKind {
		sub, err := protobufMarshalJSON(fd.Message(), v)
		if err != nil {
			return nil, err
		}
		out = protowire.AppendTag(out, fd.Number(), protowire.BytesType)
		return protowire.AppendBytes(out, sub), nil
	}
	pv, err := protobufValueFromJSON(fd, v)
	if err != nil {
		return nil, err
	}
	return protobufAppendField(out, fd, pv), nil
}

// protobufAppendField appends one tagged scalar field value to out.
func protobufAppendField(out []byte, fd protoreflect.FieldDescriptor, v protoreflect.Value) []byte {
	num := fd.Number()
	switch fd.Kind() {
	case protoreflect.BoolKind:
		b := uint64(0)
		if v.Bool() {
			b = 1
		}
		out = protowire.AppendTag(out, num, protowire.VarintType)
		return protowire.AppendVarint(out, b)
	case protoreflect.Int32Kind, protoreflect.Int64Kind:
		out = protowire.AppendTag(out, num, protowire.VarintType)
		return protowire.AppendVarint(out, uint64(v.Int())) // #nosec G115 -- narrowed to the declared protobuf field width (wire semantics)
	case protoreflect.Uint32Kind, protoreflect.Uint64Kind:
		out = protowire.AppendTag(out, num, protowire.VarintType)
		return protowire.AppendVarint(out, v.Uint())
	case protoreflect.Sint32Kind:
		n := int32(v.Int()) // #nosec G115 -- narrowed to the declared protobuf field width (wire semantics)
		out = protowire.AppendTag(out, num, protowire.VarintType)
		return protowire.AppendVarint(out, uint64(uint32((n<<1)^(n>>31)))) // #nosec G115 -- narrowed to the declared protobuf field width (wire semantics)
	case protoreflect.Sint64Kind:
		out = protowire.AppendTag(out, num, protowire.VarintType)
		return protowire.AppendVarint(out, protowire.EncodeZigZag(v.Int()))
	case protoreflect.EnumKind:
		out = protowire.AppendTag(out, num, protowire.VarintType)
		return protowire.AppendVarint(out, uint64(v.Enum())) // #nosec G115 -- narrowed to the declared protobuf field width (wire semantics)
	case protoreflect.Fixed32Kind:
		out = protowire.AppendTag(out, num, protowire.Fixed32Type)
		return protowire.AppendFixed32(out, uint32(v.Uint())) // #nosec G115 -- narrowed to the declared protobuf field width (wire semantics)
	case protoreflect.Sfixed32Kind:
		out = protowire.AppendTag(out, num, protowire.Fixed32Type)
		return protowire.AppendFixed32(out, uint32(v.Int())) // #nosec G115 -- narrowed to the declared protobuf field width (wire semantics)
	case protoreflect.FloatKind:
		out = protowire.AppendTag(out, num, protowire.Fixed32Type)
		return protowire.AppendFixed32(out, math.Float32bits(float32(v.Float())))
	case protoreflect.Fixed64Kind:
		out = protowire.AppendTag(out, num, protowire.Fixed64Type)
		return protowire.AppendFixed64(out, v.Uint())
	case protoreflect.Sfixed64Kind:
		out = protowire.AppendTag(out, num, protowire.Fixed64Type)
		return protowire.AppendFixed64(out, uint64(v.Int())) // #nosec G115 -- narrowed to the declared protobuf field width (wire semantics)
	case protoreflect.DoubleKind:
		out = protowire.AppendTag(out, num, protowire.Fixed64Type)
		return protowire.AppendFixed64(out, math.Float64bits(v.Float()))
	case protoreflect.StringKind:
		out = protowire.AppendTag(out, num, protowire.BytesType)
		return protowire.AppendString(out, v.String())
	case protoreflect.BytesKind:
		out = protowire.AppendTag(out, num, protowire.BytesType)
		return protowire.AppendBytes(out, v.Bytes())
	}
	return out
}

// coerceNumber mirrors protobufjs's lenient numeric coercion: JSON numbers pass
// through, numeric strings are parsed (JSON int64/uint64 are represented as
// strings), non-numeric strings become 0, and booleans become 0/1.
func coerceNumber(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case string:
		if f, err := strconv.ParseFloat(strings.TrimSpace(x), 64); err == nil {
			return f
		}
		return 0
	case bool:
		if x {
			return 1
		}
	}
	return 0
}

// coerceBool mirrors JavaScript's Boolean(): any non-empty string is true.
func coerceBool(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case string:
		return x != ""
	case float64:
		return x != 0
	}
	return false
}

// protobufValueFromJSON converts a scalar JSON value to a protoreflect.Value,
// applying protobufjs-style coercion.
func protobufValueFromJSON(fd protoreflect.FieldDescriptor, v any) (protoreflect.Value, error) {
	switch fd.Kind() {
	case protoreflect.BoolKind:
		return protoreflect.ValueOfBool(coerceBool(v)), nil
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return protoreflect.ValueOfInt32(int32(int64(coerceNumber(v)))), nil // #nosec G115 -- narrowed to the declared protobuf field width (wire semantics)
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return protoreflect.ValueOfInt64(int64(coerceNumber(v))), nil
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return protoreflect.ValueOfUint32(uint32(int64(coerceNumber(v)))), nil // #nosec G115 -- narrowed to the declared protobuf field width (wire semantics)
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return protoreflect.ValueOfUint64(uint64(int64(coerceNumber(v)))), nil // #nosec G115 -- narrowed to the declared protobuf field width (wire semantics)
	case protoreflect.FloatKind:
		return protoreflect.ValueOfFloat32(float32(coerceNumber(v))), nil
	case protoreflect.DoubleKind:
		return protoreflect.ValueOfFloat64(coerceNumber(v)), nil
	case protoreflect.StringKind:
		if s, ok := v.(string); ok {
			return protoreflect.ValueOfString(s), nil
		}
		return protoreflect.ValueOfString(fmt.Sprintf("%v", v)), nil
	case protoreflect.BytesKind:
		s, ok := v.(string)
		if !ok {
			return protoreflect.Value{}, fmt.Errorf("field %s: expected a base64 string", fd.Name())
		}
		b, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			return protoreflect.Value{}, fmt.Errorf("field %s: invalid base64 bytes", fd.Name())
		}
		return protoreflect.ValueOfBytes(b), nil
	case protoreflect.EnumKind:
		if s, ok := v.(string); ok {
			ev := fd.Enum().Values().ByName(protoreflect.Name(s))
			if ev == nil {
				return protoreflect.Value{}, fmt.Errorf("field %s: unknown enum value %q", fd.Name(), s)
			}
			return protoreflect.ValueOfEnum(ev.Number()), nil
		}
		return protoreflect.ValueOfEnum(protoreflect.EnumNumber(int32(coerceNumber(v)))), nil
	}
	return protoreflect.Value{}, fmt.Errorf("field %s: unsupported type %s", fd.Name(), fd.Kind())
}
