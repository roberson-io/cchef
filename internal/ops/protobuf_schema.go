package ops

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/bufbuild/protocompile"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

// protobufCompileSchema compiles .proto text and returns the first top-level
// message descriptor (CyberChef's mainMessageName is the first message defined).
func protobufCompileSchema(schema string) (protoreflect.MessageDescriptor, error) {
	compiler := protocompile.Compiler{
		Resolver: protocompile.WithStandardImports(&protocompile.SourceResolver{
			Accessor: protocompile.SourceAccessorFromMap(map[string]string{"schema.proto": schema}),
		}),
	}
	files, err := compiler.Compile(context.Background(), "schema.proto")
	if err != nil {
		return nil, fmt.Errorf("schema: %w", err)
	}
	msgs := files[0].Messages()
	if msgs.Len() == 0 {
		// A valid schema that defines no message: the caller falls back to the
		// raw decode (mirrors mergeDecodes returning rawDecode when there is no
		// main message).
		return nil, nil
	}
	return msgs.Get(0), nil
}

// protobufSchemaDecode decodes protobuf data against a .proto schema, mirroring
// protobufjs toObject conventions (bytes as strings, longs as numbers, enums as
// names, defaults included). Ported from lib/Protobuf.mjs mergeDecodes.
func protobufSchemaDecode(data []byte, raw *omap, schema string, showUnknown, showTypes bool) ([]byte, error) {
	md, err := protobufCompileSchema(schema)
	if err != nil {
		return nil, err
	}
	if md == nil {
		// Valid schema with no message defined: fall back to the raw decode.
		return jsonNoEscape(raw)
	}
	msg := dynamicpb.NewMessage(md)
	if err := proto.Unmarshal(data, msg); err != nil {
		return nil, fmt.Errorf("input: %w", err)
	}
	decoded := protobufMessageToObject(msg, showTypes)

	if !showUnknown {
		return jsonNoEscape(decoded)
	}
	out := newOMap()
	out.set(string(md.Name()), decoded)
	out.set("Unknown Fields", protobufCompareFields(raw, md))
	return jsonNoEscape(out)
}

// protobufMessageToObject converts a decoded message to an ordered map, listing
// repeated/map fields first then singular fields (protobufjs toObject order).
func protobufMessageToObject(msg protoreflect.Message, showTypes bool) *omap {
	fields := msg.Descriptor().Fields()
	var listFields, singularFields []protoreflect.FieldDescriptor
	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)
		if fd.IsList() || fd.IsMap() {
			listFields = append(listFields, fd)
		} else {
			singularFields = append(singularFields, fd)
		}
	}

	out := newOMap()
	for _, fd := range append(listFields, singularFields...) {
		name := string(fd.Name())
		if showTypes {
			name = fmt.Sprintf("%s (%s)", name, protobufFieldTypeName(fd))
		}
		out.set(name, protobufFieldValue(msg, fd, showTypes))
	}
	return out
}

// protobufFieldValue returns the toObject-style value for a single field.
func protobufFieldValue(msg protoreflect.Message, fd protoreflect.FieldDescriptor, showTypes bool) any {
	switch {
	case fd.IsList():
		list := msg.Get(fd).List()
		arr := []any{}
		for i := 0; i < list.Len(); i++ {
			arr = append(arr, protobufScalar(fd, list.Get(i), showTypes))
		}
		return arr
	case fd.IsMap():
		m := msg.Get(fd).Map()
		obj := newOMap()
		m.Range(func(k protoreflect.MapKey, v protoreflect.Value) bool {
			obj.set(k.Value().String(), protobufScalar(fd.MapValue(), v, showTypes))
			return true
		})
		return obj
	case fd.Kind() == protoreflect.MessageKind || fd.Kind() == protoreflect.GroupKind:
		if msg.Has(fd) {
			return protobufMessageToObject(msg.Get(fd).Message(), showTypes)
		}
		return nil
	default:
		return protobufScalar(fd, msg.Get(fd), showTypes)
	}
}

// protobufScalar converts a scalar protoreflect value using toObject conventions.
func protobufScalar(fd protoreflect.FieldDescriptor, v protoreflect.Value, showTypes bool) any {
	switch fd.Kind() {
	case protoreflect.BoolKind:
		return v.Bool()
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind,
		protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return float64(v.Int())
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind,
		protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return float64(v.Uint())
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		return v.Float()
	case protoreflect.StringKind:
		return v.String()
	case protoreflect.BytesKind:
		// protobufjs toObject with bytes:String base64-encodes byte fields.
		return base64.StdEncoding.EncodeToString(v.Bytes())
	case protoreflect.EnumKind:
		if ev := fd.Enum().Values().ByNumber(v.Enum()); ev != nil {
			return string(ev.Name())
		}
		return float64(v.Enum())
	case protoreflect.MessageKind, protoreflect.GroupKind:
		return protobufMessageToObject(v.Message(), showTypes)
	}
	return nil
}

// protobufExtractFieldID pulls the field number out of a raw-decode key, which
// is either a plain number ("3") or a Show-Types key ("field #3: ...").
func protobufExtractFieldID(key string) (int, bool) {
	s := key
	if strings.HasPrefix(key, "field #") {
		s = key[len("field #"):]
		if i := strings.IndexByte(s, ':'); i >= 0 {
			s = s[:i]
		}
	}
	n, err := strconv.Atoi(s)
	return n, err == nil
}

// protobufCompareFields returns the raw-decode fields not represented in the
// schema, plus annotations for repeated/submessage mismatches. Ported from
// Protobuf.compareFields.
func protobufCompareFields(raw *omap, md protoreflect.MessageDescriptor) *omap {
	schemaByID := map[int]protoreflect.FieldDescriptor{}
	fields := md.Fields()
	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)
		schemaByID[int(fd.Number())] = fd
	}

	out := newOMap()
	for _, key := range raw.keys {
		value := raw.vals[key]
		id, ok := protobufExtractFieldID(key)
		fd, known := schemaByID[id]
		if !ok || !known {
			out.set(key, value)
			continue
		}

		arr, isArr := value.([]any)
		if isArr && !fd.IsList() {
			out.set(fmt.Sprintf("(%s) %s is a repeated field", md.Name(), fd.Name()), value)
		}
		if fd.Kind() == protoreflect.MessageKind || fd.Kind() == protoreflect.GroupKind {
			sub := newOMap()
			if isArr {
				for _, inst := range arr {
					if instOmap, ok := inst.(*omap); ok {
						for _, k := range instOmap.keys {
							sub.set(k, instOmap.vals[k])
						}
					}
				}
			} else if v, ok := value.(*omap); ok {
				sub = v
			}
			if subCompared := protobufCompareFields(sub, fd.Message()); len(subCompared.keys) != 0 {
				out.set(fmt.Sprintf("%s (%s) has missing fields", fd.Name(), fd.Message().Name()), subCompared)
			}
		}
	}
	return out
}

// protobufFieldTypeName returns the .proto type name for a field (used by
// "Show Types"): the scalar keyword, or the message/enum type name.
func protobufFieldTypeName(fd protoreflect.FieldDescriptor) string {
	switch fd.Kind() {
	case protoreflect.MessageKind, protoreflect.GroupKind:
		return string(fd.Message().Name())
	case protoreflect.EnumKind:
		return string(fd.Enum().Name())
	default:
		return fd.Kind().String()
	}
}
