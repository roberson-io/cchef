// Package protobuf reads Protobuf wire data with or without a schema.
//
// [NewParser] walks raw wire format into an ordered field-number map, tracking
// each field's wire type; [ShowRawTypes] annotates that output the way
// CyberChef's Protobuf Decode does. [CompileSchema] compiles a .proto source
// so decoding can use real field names and types, and [SchemaDecode] applies
// it. The Protobuf Decode and Protobuf Encode operations are built on these.
package protobuf

import (
	"errors"
	"fmt"
	"maps"
	"strconv"

	"github.com/roberson-io/cchef/internal/jsonval"
	"github.com/roberson-io/cchef/internal/opsutil"
)

// Parser performs a schema-less decode of protobuf wire data, mirroring
// CyberChef's lib/Protobuf.mjs raw parser. It records per-field wire types (for
// the "Show Types" option) and treats a length-delimited field as a nested
// message when it parses cleanly, otherwise as a byte string.
type Parser struct {
	data   []byte
	offset int
	// FieldTypes maps a field number to its wire type (int), or to a nested
	// map[string]any of sub-field types for submessages.
	FieldTypes map[string]any
}

var errProtobufOverrun = errors.New("exhausted buffer")

// NewParser returns a parser over raw wire data.
func NewParser(data []byte) *Parser {
	return &Parser{data: data, FieldTypes: map[string]any{}}
}

// Parse reads all fields into an ordered map keyed by field number.
func (p *Parser) Parse() (*jsonval.OMap, error) {
	obj := jsonval.NewOMap()
	for p.offset < len(p.data) {
		key, value, err := p.parseField()
		if err != nil {
			return nil, err
		}
		p.addField(obj, key, value)
	}
	if p.offset > len(p.data) {
		return nil, errProtobufOverrun
	}
	return obj, nil
}

// addField inserts value under key, collecting repeats into an array.
func (p *Parser) addField(obj *jsonval.OMap, key string, value any) {
	if existing, ok := obj.Get(key); ok {
		if arr, isArr := existing.([]any); isArr {
			obj.Set(key, append(arr, value))
		} else {
			obj.Set(key, []any{existing, value})
		}
		return
	}
	obj.Set(key, value)
}

func (p *Parser) parseField() (string, any, error) {
	if p.offset >= len(p.data) {
		p.offset = len(p.data) + 1
		return "", nil, errProtobufOverrun
	}
	wireType := int(p.data[p.offset]) & 0x07
	key := strconv.Itoa(p.fieldNumber())

	if _, isMap := p.FieldTypes[key].(map[string]any); !isMap {
		p.FieldTypes[key] = wireType
	}

	switch wireType {
	case 0:
		return key, p.varInt(), nil
	case 1:
		return key, p.uint64(), nil
	case 2:
		v, err := p.lenDelim(key)
		return key, v, err
	case 5:
		return key, p.uint32(), nil
	default:
		return "", nil, fmt.Errorf("unknown type 0x%x", wireType)
	}
}

// fieldNumber reads the varint-encoded field number from the tag, whose low 3
// bits are the wire type (already read).
func (p *Parser) fieldNumber() int {
	shift := -3
	fieldNumber := 0
	for {
		b := 0
		if p.offset < len(p.data) {
			b = int(p.data[p.offset])
		}
		switch {
		case shift < 28 && shift == -3:
			fieldNumber += (b & 0x78) >> 3
		case shift < 28:
			fieldNumber += (b & 0x7f) << shift
		default:
			fieldNumber += (b & 0x7f) * (1 << shift)
		}
		shift += 7
		msb := b & 0x80
		p.offset++
		if msb != 0x80 {
			break
		}
	}
	return fieldNumber
}

func (p *Parser) varInt() float64 {
	value := 0.0
	shift := 0
	for {
		b := 0
		if p.offset < len(p.data) {
			b = int(p.data[p.offset])
		}
		if shift < 28 {
			value += float64((b & 0x7f) << shift)
		} else {
			value += float64(b&0x7f) * float64(int64(1)<<shift)
		}
		shift += 7
		msb := b & 0x80
		p.offset++
		if msb != 0x80 {
			break
		}
	}
	return value
}

func (p *Parser) uint64() float64 {
	b := func() float64 {
		v := 0.0
		if p.offset < len(p.data) {
			v = float64(p.data[p.offset])
		}
		p.offset++
		return v
	}
	lower := b() + b()*0x100 + b()*0x10000 + b()*0x1000000
	upper := b() + b()*0x100 + b()*0x10000 + b()*0x1000000
	return upper*0x100000000 + lower
}

func (p *Parser) uint32() float64 {
	v := 0.0
	for i := range 4 {
		bv := 0.0
		if p.offset < len(p.data) {
			bv = float64(p.data[p.offset])
		}
		v += bv * float64(int64(1)<<(8*i))
		p.offset++
	}
	return v
}

// lenDelim reads a length-delimited field: a nested message when it parses
// cleanly, otherwise the raw bytes as a latin1 string.
func (p *Parser) lenDelim(fieldNum string) (any, error) {
	// The length comes off the wire and can name more bytes than exist, or more
	// than an int can hold. JavaScript keeps it as a float and simply runs off
	// the end, so any length the buffer cannot satisfy is treated as exactly
	// that: an overrun, reported when parsing finishes.
	length := p.varInt()
	end := len(p.data) + 1
	if length >= 0 && length <= float64(len(p.data)) {
		end = p.offset + int(length)
	}
	// Reading the length can already have run the offset past the data, and
	// slicing from beyond the end yields nothing rather than failing.
	start := min(p.offset, len(p.data))
	sliceEnd := min(max(end, start), len(p.data))
	fieldBytes := p.data[start:sliceEnd]

	var field any
	sub := NewParser(fieldBytes)
	if parsed, err := sub.Parse(); err == nil {
		field = parsed
		merged, _ := p.FieldTypes[fieldNum].(map[string]any)
		if merged == nil {
			merged = map[string]any{}
		}
		maps.Copy(merged, sub.FieldTypes)
		p.FieldTypes[fieldNum] = merged
	} else {
		field = opsutil.BytesAsLatin1(fieldBytes)
	}
	p.offset = end
	return field, nil
}

// typeInfo returns the wire-type description used by "Show Types".
func typeInfo(wireType int) string {
	switch wireType {
	case 0:
		return "VarInt (e.g. int32, bool)"
	case 1:
		return "64-Bit (e.g. fixed64, double)"
	case 2:
		return "L-delim (e.g. string, message)"
	case 5:
		return "32-Bit (e.g. fixed32, float)"
	}
	return ""
}

// ShowRawTypes rewrites raw-decode field-number keys to include the wire type,
// recursing into submessages.
func ShowRawTypes(raw *jsonval.OMap, fieldTypes map[string]any) *jsonval.OMap {
	out := jsonval.NewOMap()
	for _, fieldNum := range raw.Keys() {
		value := raw.Value(fieldNum)
		ft := fieldTypes[fieldNum]
		var outType int
		var outValue any

		if subTypes, isMsg := ft.(map[string]any); isMsg {
			outType = 2
			switch v := value.(type) {
			case []any:
				instances := make([]any, 0, len(v))
				for _, inst := range v {
					if sub, ok := inst.(*jsonval.OMap); ok {
						instances = append(instances, ShowRawTypes(sub, subTypes))
					} else {
						instances = append(instances, inst)
					}
				}
				outValue = instances
			case *jsonval.OMap:
				outValue = ShowRawTypes(v, subTypes)
			default:
				outValue = value
			}
		} else {
			if wt, ok := ft.(int); ok {
				outType = wt
			}
			outValue = value
		}
		out.Set(fmt.Sprintf("field #%s: %s", fieldNum, typeInfo(outType)), outValue)
	}
	return out
}
