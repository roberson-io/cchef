package ops

import (
	"bytes"
	"compress/flate"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

// AvroToJSON converts Avro Object Container File data into JSON.
//
// CyberChef's operation is a thin wrapper around the avsc npm library's
// BlockDecoder. This is a from-scratch port of the Avro OCF format (BIP-less;
// see https://avro.apache.org/docs/1.11.1/specification/) that reproduces
// avsc's value representation: bytes/fixed render as Node Buffers
// ({"type":"Buffer","data":[...]}), records/maps keep field order, and unions
// are unwrapped unless ambiguous, in which case avsc's wrapUnions:"auto" wraps
// every non-null branch as {"<branchName>": value}.
type AvroToJSON struct{}

// Meta returns the operation metadata.
func (AvroToJSON) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Avro to JSON",
		Module:      "Serialise",
		Description: "Converts Avro encoded data into JSON.",
		InfoURL:     "https://wikipedia.org/wiki/Apache_Avro",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (AvroToJSON) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Force Valid JSON", Type: core.ArgBoolean, Value: true},
	}
}

// Run decodes the Avro input into JSON.
func (AvroToJSON) Run(in *core.Dish, args []any) (*core.Dish, error) {
	forceJSON := args[0].(bool)
	input := in.Bytes()
	if len(input) == 0 {
		return nil, errors.New("Please provide an input.") //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}

	results, err := avroDecodeOCF(input)
	if err != nil {
		return nil, errors.New("Error parsing Avro file.") //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}

	var out string
	if forceJSON {
		if len(results) == 1 {
			out = avroStringify(results[0], 4)
		} else {
			out = avroStringify(results, 4)
		}
	} else {
		var sb strings.Builder
		for _, r := range results {
			sb.WriteString(avroStringify(r, 0))
			sb.WriteByte('\n')
		}
		out = sb.String()
	}
	return core.NewDish([]byte(out), core.TypeString), nil
}

// --- OCF container ---

var avroMagic = []byte{0x4f, 0x62, 0x6a, 0x01}

// avroDecodeOCF mirrors avsc's BlockDecoder, including its leniency: running out
// of bytes (a truncated header or a partially-present data block) stops cleanly
// and yields the records from complete blocks only, while genuine corruption of
// a fully-present block (bad magic, sync mismatch, undecodable data) is an
// error that discards everything — matching CyberChef's reject-on-"error".
func avroDecodeOCF(data []byte) ([]any, error) {
	r := &areader{data: data}
	results := []any{}

	magic, err := r.take(4)
	if err != nil {
		return results, nil //nolint:nilerr // avsc is lenient: fewer than 4 bytes yields no records
	}
	if !bytes.Equal(magic, avroMagic) {
		return nil, errors.New("invalid Avro magic")
	}
	if r.eof() {
		return results, nil // bare magic
	}

	meta, err := r.readMetaMap()
	if err != nil {
		if errors.Is(err, io.ErrUnexpectedEOF) {
			return results, nil //nolint:nilerr // avsc is lenient: a truncated header yields no records
		}
		return nil, err
	}
	schemaJSON, ok := meta["avro.schema"]
	if !ok {
		return nil, errors.New("missing avro.schema")
	}
	codec := "null"
	if c, ok := meta["avro.codec"]; ok && len(c) > 0 {
		codec = string(c)
	}

	var schemaAny any
	if err := json.Unmarshal(schemaJSON, &schemaAny); err != nil {
		return nil, err
	}
	reg := map[string]*avroSchema{}
	schema, err := parseAvroSchema(schemaAny, reg, "")
	if err != nil {
		return nil, err
	}

	sync, err := r.take(16)
	if err != nil {
		return results, nil //nolint:nilerr // avsc is lenient: truncated before the header sync
	}
	syncMarker := append([]byte(nil), sync...)

	for !r.eof() {
		count, err := r.readLong()
		if err != nil {
			if errors.Is(err, io.ErrUnexpectedEOF) {
				return results, nil //nolint:nilerr // avsc is lenient: a truncated block yields the prior records
			}
			return nil, err
		}
		size, err := r.readLong()
		if err != nil {
			if errors.Is(err, io.ErrUnexpectedEOF) {
				return results, nil //nolint:nilerr // avsc is lenient: a truncated block yields the prior records
			}
			return nil, err
		}
		if size < 0 || count < 0 {
			return nil, errors.New("invalid block header")
		}
		block, err := r.take(int(size))
		if err != nil {
			return results, nil //nolint:nilerr // avsc is lenient: block data not fully present
		}
		blockSync, err := r.take(16)
		if err != nil {
			return results, nil //nolint:nilerr // avsc is lenient: block sync not fully present
		}
		// A complete block with a bad sync or undecodable data is corruption.
		if !bytes.Equal(blockSync, syncMarker) {
			return nil, errors.New("sync marker mismatch")
		}
		decoded, err := avroDecompress(codec, block)
		if err != nil {
			return nil, err
		}
		br := &areader{data: decoded}
		blockRecords := make([]any, 0, count)
		for range count {
			v, err := decodeValue(schema, br)
			if err != nil {
				return nil, err
			}
			blockRecords = append(blockRecords, v)
		}
		if !br.eof() {
			return nil, errors.New("block not fully consumed")
		}
		results = append(results, blockRecords...)
	}
	return results, nil
}

func avroDecompress(codec string, block []byte) ([]byte, error) {
	switch codec {
	case "null":
		return block, nil
	case "deflate":
		fr := flate.NewReader(bytes.NewReader(block))
		defer func() { _ = fr.Close() }()
		return io.ReadAll(fr)
	default:
		return nil, errors.New("unsupported codec: " + codec)
	}
}

// --- reader ---

type areader struct {
	data []byte
	pos  int
}

func (r *areader) eof() bool { return r.pos >= len(r.data) }

func (r *areader) take(n int) ([]byte, error) {
	if n < 0 || r.pos+n > len(r.data) {
		return nil, io.ErrUnexpectedEOF
	}
	b := r.data[r.pos : r.pos+n]
	r.pos += n
	return b, nil
}

func (r *areader) byte() (byte, error) {
	if r.pos >= len(r.data) {
		return 0, io.ErrUnexpectedEOF
	}
	b := r.data[r.pos]
	r.pos++
	return b, nil
}

// readLong reads an Avro variable-length zig-zag encoded long.
func (r *areader) readLong() (int64, error) {
	var u uint64
	var shift uint
	for {
		b, err := r.byte()
		if err != nil {
			return 0, err
		}
		u |= uint64(b&0x7f) << shift
		if b&0x80 == 0 {
			break
		}
		shift += 7
		if shift >= 64 {
			return 0, errors.New("varint overflow")
		}
	}
	return int64(u>>1) ^ -int64(u&1), nil
}

// readBytes reads a length-prefixed byte string (bytes / string).
func (r *areader) readBytes() ([]byte, error) {
	n, err := r.readLong()
	if err != nil {
		return nil, err
	}
	if n < 0 {
		return nil, errors.New("negative length")
	}
	return r.take(int(n))
}

func (r *areader) readMetaMap() (map[string][]byte, error) {
	m := map[string][]byte{}
	for {
		count, err := r.readLong()
		if err != nil {
			return nil, err
		}
		if count == 0 {
			return m, nil
		}
		if count < 0 {
			count = -count
			if _, err := r.readLong(); err != nil { // block byte size, unused
				return nil, err
			}
		}
		for i := int64(0); i < count; i++ {
			key, err := r.readBytes()
			if err != nil {
				return nil, err
			}
			val, err := r.readBytes()
			if err != nil {
				return nil, err
			}
			m[string(key)] = append([]byte(nil), val...)
		}
	}
}

// --- schema ---

type avroField struct {
	name   string
	schema *avroSchema
}

type avroSchema struct {
	kind     string // null,boolean,int,long,float,double,bytes,string,record,enum,array,map,fixed,union
	fields   []avroField
	symbols  []string
	items    *avroSchema
	values   *avroSchema
	size     int
	branches []*avroSchema
	wrapped  bool   // union: avsc wraps ambiguous unions
	name     string // full name for named types (record/enum/fixed)
}

var avroPrimitives = map[string]bool{
	"null": true, "boolean": true, "int": true, "long": true,
	"float": true, "double": true, "bytes": true, "string": true,
}

func parseAvroSchema(j any, reg map[string]*avroSchema, ns string) (*avroSchema, error) {
	switch v := j.(type) {
	case string:
		if avroPrimitives[v] {
			return &avroSchema{kind: v}, nil
		}
		if s, ok := reg[v]; ok {
			return s, nil
		}
		if ns != "" {
			if s, ok := reg[ns+"."+v]; ok {
				return s, nil
			}
		}
		return nil, errors.New("unknown named type: " + v)
	case []any:
		branches := make([]*avroSchema, 0, len(v))
		for _, b := range v {
			bs, err := parseAvroSchema(b, reg, ns)
			if err != nil {
				return nil, err
			}
			branches = append(branches, bs)
		}
		return &avroSchema{kind: "union", branches: branches, wrapped: avroUnionWraps(branches)}, nil
	case map[string]any:
		t, _ := v["type"].(string)
		switch t {
		case "record", "error":
			return parseNamed(v, reg, ns, "record")
		case "enum":
			return parseNamed(v, reg, ns, "enum")
		case "fixed":
			return parseNamed(v, reg, ns, "fixed")
		case "array":
			items, err := parseAvroSchema(v["items"], reg, ns)
			if err != nil {
				return nil, err
			}
			return &avroSchema{kind: "array", items: items}, nil
		case "map":
			values, err := parseAvroSchema(v["values"], reg, ns)
			if err != nil {
				return nil, err
			}
			return &avroSchema{kind: "map", values: values}, nil
		default:
			// A primitive wrapped in an object, optionally with a logicalType
			// (which avsc ignores without a logicalTypes option), or a nested
			// type/reference under "type".
			if avroPrimitives[t] {
				return &avroSchema{kind: t}, nil
			}
			return parseAvroSchema(v["type"], reg, ns)
		}
	default:
		return nil, errors.New("invalid schema node")
	}
}

func parseNamed(v map[string]any, reg map[string]*avroSchema, ns, kind string) (*avroSchema, error) {
	name, _ := v["name"].(string)
	childNS := ns
	if e, ok := v["namespace"].(string); ok {
		childNS = e
	} else if i := strings.LastIndex(name, "."); i >= 0 {
		childNS = name[:i]
	}
	full := name
	if !strings.Contains(name, ".") && childNS != "" {
		full = childNS + "." + name
	}

	s := &avroSchema{kind: kind, name: full}
	reg[full] = s // register before parsing fields to allow recursion

	switch kind {
	case "record":
		fieldsAny, _ := v["fields"].([]any)
		for _, f := range fieldsAny {
			fm, ok := f.(map[string]any)
			if !ok {
				return nil, errors.New("invalid field")
			}
			fname, _ := fm["name"].(string)
			fs, err := parseAvroSchema(fm["type"], reg, childNS)
			if err != nil {
				return nil, err
			}
			s.fields = append(s.fields, avroField{name: fname, schema: fs})
		}
	case "enum":
		symsAny, _ := v["symbols"].([]any)
		for _, sym := range symsAny {
			ss, _ := sym.(string)
			s.symbols = append(s.symbols, ss)
		}
	case "fixed":
		if sz, ok := v["size"].(float64); ok {
			s.size = int(sz)
		}
	}
	return s, nil
}

// avroBucket groups branch schemas by their JS value representation; a union is
// wrapped when two non-null branches share a bucket.
func avroBucket(s *avroSchema) string {
	switch s.kind {
	case "boolean":
		return "boolean"
	case "int", "long", "float", "double":
		return "number"
	case "string", "enum":
		return "string"
	case "bytes", "fixed":
		return "buffer"
	case "record", "map":
		return "object"
	case "array":
		return "array"
	default:
		return "null"
	}
}

func avroUnionWraps(branches []*avroSchema) bool {
	seen := map[string]bool{}
	for _, b := range branches {
		bk := avroBucket(b)
		if bk == "null" {
			continue
		}
		if seen[bk] {
			return true
		}
		seen[bk] = true
	}
	return false
}

func avroBranchName(s *avroSchema) string {
	switch s.kind {
	case "record", "enum", "fixed":
		return s.name
	default:
		return s.kind
	}
}

// --- value decoding ---

func decodeValue(s *avroSchema, r *areader) (any, error) {
	switch s.kind {
	case "null":
		return nil, nil
	case "boolean":
		b, err := r.byte()
		if err != nil {
			return nil, err
		}
		return b != 0, nil
	case "int", "long":
		return r.readLong()
	case "float":
		b, err := r.take(4)
		if err != nil {
			return nil, err
		}
		return float64(math.Float32frombits(binary.LittleEndian.Uint32(b))), nil
	case "double":
		b, err := r.take(8)
		if err != nil {
			return nil, err
		}
		return math.Float64frombits(binary.LittleEndian.Uint64(b)), nil
	case "bytes":
		b, err := r.readBytes()
		if err != nil {
			return nil, err
		}
		return avroBuffer(b), nil
	case "string":
		b, err := r.readBytes()
		if err != nil {
			return nil, err
		}
		return string(b), nil
	case "fixed":
		b, err := r.take(s.size)
		if err != nil {
			return nil, err
		}
		return avroBuffer(b), nil
	case "enum":
		i, err := r.readLong()
		if err != nil {
			return nil, err
		}
		if i < 0 || int(i) >= len(s.symbols) {
			return nil, errors.New("enum index out of range")
		}
		return s.symbols[i], nil
	case "record":
		obj := make(avroObject, 0, len(s.fields))
		for _, f := range s.fields {
			v, err := decodeValue(f.schema, r)
			if err != nil {
				return nil, err
			}
			obj = append(obj, avroPair{k: f.name, v: v})
		}
		return obj, nil
	case "array":
		return decodeBlocks(r, func() (any, error) { return decodeValue(s.items, r) })
	case "map":
		obj := avroObject{}
		for {
			count, err := r.readLong()
			if err != nil {
				return nil, err
			}
			if count == 0 {
				return obj, nil
			}
			if count < 0 {
				count = -count
				if _, err := r.readLong(); err != nil {
					return nil, err
				}
			}
			for i := int64(0); i < count; i++ {
				key, err := r.readBytes()
				if err != nil {
					return nil, err
				}
				v, err := decodeValue(s.values, r)
				if err != nil {
					return nil, err
				}
				obj = append(obj, avroPair{k: string(key), v: v})
			}
		}
	case "union":
		idx, err := r.readLong()
		if err != nil {
			return nil, err
		}
		if idx < 0 || int(idx) >= len(s.branches) {
			return nil, errors.New("union index out of range")
		}
		branch := s.branches[idx]
		v, err := decodeValue(branch, r)
		if err != nil {
			return nil, err
		}
		if s.wrapped && branch.kind != "null" {
			return avroObject{{k: avroBranchName(branch), v: v}}, nil
		}
		return v, nil
	default:
		return nil, errors.New("unknown schema kind: " + s.kind)
	}
}

// decodeBlocks reads an Avro block-encoded array, returning a non-nil slice so
// an empty array serialises as [].
func decodeBlocks(r *areader, item func() (any, error)) (any, error) {
	out := []any{}
	for {
		count, err := r.readLong()
		if err != nil {
			return nil, err
		}
		if count == 0 {
			return out, nil
		}
		if count < 0 {
			count = -count
			if _, err := r.readLong(); err != nil {
				return nil, err
			}
		}
		for i := int64(0); i < count; i++ {
			v, err := item()
			if err != nil {
				return nil, err
			}
			out = append(out, v)
		}
	}
}

// avroBuffer renders a byte slice the way JSON.stringify renders a Node Buffer.
func avroBuffer(b []byte) avroObject {
	data := make([]any, len(b))
	for i, c := range b {
		data[i] = int64(c)
	}
	return avroObject{{k: "type", v: "Buffer"}, {k: "data", v: data}}
}

// --- value model + JSON.stringify-equivalent serialiser ---

type avroPair struct {
	k string
	v any
}

type avroObject []avroPair

// avroStringify reproduces JavaScript's JSON.stringify(value, null, indent):
// indent 0 is compact, indent > 0 pretty-prints with that many spaces.
func avroStringify(v any, indent int) string {
	var sb strings.Builder
	avroWrite(&sb, v, indent, "")
	return sb.String()
}

func avroWrite(sb *strings.Builder, v any, indent int, cur string) {
	switch x := v.(type) {
	case nil:
		sb.WriteString("null")
	case bool:
		if x {
			sb.WriteString("true")
		} else {
			sb.WriteString("false")
		}
	case int64:
		sb.WriteString(strconv.FormatInt(x, 10))
	case float64:
		sb.WriteString(avroFormatNumber(x))
	case string:
		sb.WriteString(avroJSONString(x))
	case []any:
		avroWriteArray(sb, x, indent, cur)
	case avroObject:
		avroWriteObject(sb, x, indent, cur)
	}
}

func avroWriteArray(sb *strings.Builder, arr []any, indent int, cur string) {
	if len(arr) == 0 {
		sb.WriteString("[]")
		return
	}
	if indent == 0 {
		sb.WriteByte('[')
		for i, e := range arr {
			if i > 0 {
				sb.WriteByte(',')
			}
			avroWrite(sb, e, 0, "")
		}
		sb.WriteByte(']')
		return
	}
	inner := cur + strings.Repeat(" ", indent)
	sb.WriteString("[\n")
	for i, e := range arr {
		sb.WriteString(inner)
		avroWrite(sb, e, indent, inner)
		if i < len(arr)-1 {
			sb.WriteByte(',')
		}
		sb.WriteByte('\n')
	}
	sb.WriteString(cur)
	sb.WriteByte(']')
}

func avroWriteObject(sb *strings.Builder, obj avroObject, indent int, cur string) {
	if len(obj) == 0 {
		sb.WriteString("{}")
		return
	}
	if indent == 0 {
		sb.WriteByte('{')
		for i, p := range obj {
			if i > 0 {
				sb.WriteByte(',')
			}
			sb.WriteString(avroJSONString(p.k))
			sb.WriteByte(':')
			avroWrite(sb, p.v, 0, "")
		}
		sb.WriteByte('}')
		return
	}
	inner := cur + strings.Repeat(" ", indent)
	sb.WriteString("{\n")
	for i, p := range obj {
		sb.WriteString(inner)
		sb.WriteString(avroJSONString(p.k))
		sb.WriteString(": ")
		avroWrite(sb, p.v, indent, inner)
		if i < len(obj)-1 {
			sb.WriteByte(',')
		}
		sb.WriteByte('\n')
	}
	sb.WriteString(cur)
	sb.WriteByte('}')
}

// avroFormatNumber matches JSON.stringify's number output: NaN/Infinity become
// null, negative zero becomes "0", and everything else uses Go's ECMAScript-
// compatible float formatting.
func avroFormatNumber(f float64) string {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return "null"
	}
	if f == 0 {
		return "0"
	}
	b, _ := json.Marshal(f)
	return string(b)
}

// avroJSONString escapes a string the way JSON.stringify does (no HTML escaping
// of <, >, &).
func avroJSONString(s string) string {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s)
	return strings.TrimRight(b.String(), "\n")
}

func init() {
	core.Register(AvroToJSON{})
}
