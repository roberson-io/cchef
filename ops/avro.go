package ops

import (
	"bytes"
	"compress/flate"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"math"
	"strings"

	"github.com/roberson-io/cchef/core"
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
			out = jsStringify(results[0], 4)
		} else {
			out = jsStringify(results, 4)
		}
	} else {
		var sb strings.Builder
		for _, r := range results {
			sb.WriteString(jsStringify(r, 0))
			sb.WriteByte('\n')
		}
		out = sb.String()
	}
	return core.NewDish([]byte(out), core.TypeString), nil
}

// --- OCF container ---

var avroMagic = []byte{0x4f, 0x62, 0x6a, 0x01}

// avroSyncLen is the length of an OCF sync marker, in bytes.
const avroSyncLen = 16

// avroDecodeOCF mirrors avsc's BlockDecoder, including its leniency: running out
// of bytes (a truncated header or a partially-present data block) stops cleanly
// and yields the records from complete blocks only, while genuine corruption of
// a fully-present block (bad magic, sync mismatch, undecodable data) is an
// error that discards everything — matching CyberChef's reject-on-"error".
func avroDecodeOCF(data []byte) ([]any, error) {
	r := &areader{data: data}
	results := []any{}

	schema, codec, syncMarker, done, err := avroReadHeader(r)
	if err != nil {
		return nil, err
	}
	if done {
		return results, nil
	}

	for !r.eof() {
		records, done, err := avroReadBlock(r, schema, codec, syncMarker)
		if err != nil {
			return nil, err
		}
		if done {
			return results, nil
		}
		results = append(results, records...)
	}
	return results, nil
}

// avroReadHeader reads the OCF header (magic, metadata, schema, sync marker).
// done is true (with no error) for the lenient cases avsc tolerates by yielding
// no records: a stream shorter than the magic, bare magic, or a truncated header.
func avroReadHeader(r *areader) (schema *avroSchema, codec string, syncMarker []byte, done bool, err error) {
	magic, err := r.take(len(avroMagic))
	if err != nil {
		return nil, "", nil, true, nil //nolint:nilerr // avsc is lenient: fewer than 4 bytes yields no records
	}
	if !bytes.Equal(magic, avroMagic) {
		return nil, "", nil, false, errors.New("invalid Avro magic")
	}
	if r.eof() {
		return nil, "", nil, true, nil // bare magic
	}

	meta, err := r.readMetaMap()
	if err != nil {
		if errors.Is(err, io.ErrUnexpectedEOF) {
			return nil, "", nil, true, nil //nolint:nilerr // avsc is lenient: a truncated header yields no records
		}
		return nil, "", nil, false, err
	}
	schemaJSON, ok := meta["avro.schema"]
	if !ok {
		return nil, "", nil, false, errors.New("missing avro.schema")
	}
	codec = "null"
	if c, ok := meta["avro.codec"]; ok && len(c) > 0 {
		codec = string(c)
	}

	var schemaAny any
	if err := json.Unmarshal(schemaJSON, &schemaAny); err != nil {
		return nil, "", nil, false, err
	}
	reg := map[string]*avroSchema{}
	schema, err = parseAvroSchema(schemaAny, reg, "")
	if err != nil {
		return nil, "", nil, false, err
	}

	sync, err := r.take(avroSyncLen)
	if err != nil {
		return nil, "", nil, true, nil //nolint:nilerr // avsc is lenient: truncated before the header sync
	}
	return schema, codec, append([]byte(nil), sync...), false, nil
}

// avroReadBlock reads and decodes one OCF data block. done is true (no error)
// for a truncated block, which avsc tolerates by yielding the records read so
// far; a complete block with a bad sync marker is corruption (an error).
func avroReadBlock(r *areader, schema *avroSchema, codec string, syncMarker []byte) (records []any, done bool, err error) {
	count, err := r.readLong()
	if err != nil {
		if errors.Is(err, io.ErrUnexpectedEOF) {
			return nil, true, nil //nolint:nilerr // avsc is lenient: a truncated block yields the prior records
		}
		return nil, false, err
	}
	size, err := r.readLong()
	if err != nil {
		if errors.Is(err, io.ErrUnexpectedEOF) {
			return nil, true, nil //nolint:nilerr // avsc is lenient: a truncated block yields the prior records
		}
		return nil, false, err
	}
	if size < 0 || count < 0 {
		return nil, false, errors.New("invalid block header")
	}
	block, err := r.take(int(size))
	if err != nil {
		return nil, true, nil //nolint:nilerr // avsc is lenient: block data not fully present
	}
	blockSync, err := r.take(avroSyncLen)
	if err != nil {
		return nil, true, nil //nolint:nilerr // avsc is lenient: block sync not fully present
	}
	// A complete block with a bad sync or undecodable data is corruption.
	if !bytes.Equal(blockSync, syncMarker) {
		return nil, false, errors.New("sync marker mismatch")
	}
	decoded, err := avroDecompress(codec, block)
	if err != nil {
		return nil, false, err
	}
	br := &areader{data: decoded}
	records = make([]any, 0, count)
	for range count {
		v, err := decodeValue(schema, br)
		if err != nil {
			return nil, false, err
		}
		records = append(records, v)
	}
	if !br.eof() {
		return nil, false, errors.New("block not fully consumed")
	}
	return records, false, nil
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
		return parseAvroString(v, reg, ns)
	case []any:
		return parseAvroUnion(v, reg, ns)
	case map[string]any:
		return parseAvroObject(v, reg, ns)
	default:
		return nil, errors.New("invalid schema node")
	}
}

// parseAvroString resolves a bare type name: a primitive, or a named type looked
// up in the registry (optionally namespace-qualified).
func parseAvroString(v string, reg map[string]*avroSchema, ns string) (*avroSchema, error) {
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
}

// parseAvroUnion parses a JSON array of branch schemas into a union.
func parseAvroUnion(v []any, reg map[string]*avroSchema, ns string) (*avroSchema, error) {
	branches := make([]*avroSchema, 0, len(v))
	for _, b := range v {
		bs, err := parseAvroSchema(b, reg, ns)
		if err != nil {
			return nil, err
		}
		branches = append(branches, bs)
	}
	return &avroSchema{kind: "union", branches: branches, wrapped: avroUnionWraps(branches)}, nil
}

// parseAvroObject parses a JSON object schema, dispatching on its "type" field to
// named types (record/enum/fixed), array, map, or a primitive/nested reference.
func parseAvroObject(v map[string]any, reg map[string]*avroSchema, ns string) (*avroSchema, error) {
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
		// A primitive wrapped in an object, optionally with a logicalType (which
		// avsc ignores without a logicalTypes option), or a nested type/reference
		// under "type".
		if avroPrimitives[t] {
			return &avroSchema{kind: t}, nil
		}
		return parseAvroSchema(v["type"], reg, ns)
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
	if v, handled, err := decodePrimitive(s, r); handled {
		return v, err
	}
	switch s.kind {
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
		return decodeRecord(s, r)
	case "array":
		return decodeBlocks(r, func() (any, error) { return decodeValue(s.items, r) })
	case "map":
		return decodeMap(s, r)
	case "union":
		return decodeUnion(s, r)
	default:
		return nil, errors.New("unknown schema kind: " + s.kind)
	}
}

// decodePrimitive decodes the Avro primitive types (null, boolean, int, long,
// float, double, bytes, string). The bool return reports whether the kind was a
// primitive handled here; complex types return (nil, false, nil).
func decodePrimitive(s *avroSchema, r *areader) (any, bool, error) {
	switch s.kind {
	case "null":
		return nil, true, nil
	case "boolean":
		b, err := r.byte()
		if err != nil {
			return nil, true, err
		}
		return b != 0, true, nil
	case "int", "long":
		v, err := r.readLong()
		return v, true, err
	case "float":
		b, err := r.take(4)
		if err != nil {
			return nil, true, err
		}
		return float64(math.Float32frombits(binary.LittleEndian.Uint32(b))), true, nil
	case "double":
		b, err := r.take(8)
		if err != nil {
			return nil, true, err
		}
		return math.Float64frombits(binary.LittleEndian.Uint64(b)), true, nil
	case "bytes":
		b, err := r.readBytes()
		if err != nil {
			return nil, true, err
		}
		return avroBuffer(b), true, nil
	case "string":
		b, err := r.readBytes()
		if err != nil {
			return nil, true, err
		}
		return string(b), true, nil
	default:
		return nil, false, nil
	}
}

// decodeRecord decodes an Avro record: each field in order, into an
// insertion-ordered object.
func decodeRecord(s *avroSchema, r *areader) (any, error) {
	obj := make(jsObject, 0, len(s.fields))
	for _, f := range s.fields {
		v, err := decodeValue(f.schema, r)
		if err != nil {
			return nil, err
		}
		obj = append(obj, jsPair{k: f.name, v: v})
	}
	return obj, nil
}

// decodeMap decodes an Avro block-encoded map into an insertion-ordered object.
// Blocks repeat until a zero count; a negative count carries an (ignored) block
// byte-size prefix.
func decodeMap(s *avroSchema, r *areader) (any, error) {
	obj := jsObject{}
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
			obj = append(obj, jsPair{k: string(key), v: v})
		}
	}
}

// decodeUnion decodes an Avro union: a branch index followed by that branch's
// value. When the schema is wrapped, a non-null branch is tagged by name.
func decodeUnion(s *avroSchema, r *areader) (any, error) {
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
		return jsObject{{k: avroBranchName(branch), v: v}}, nil
	}
	return v, nil
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

// avroBuffer renders a byte slice as a Node Buffer, matching avsc's
// representation of Avro bytes/fixed values.
func avroBuffer(b []byte) jsObject {
	return jsBuffer(b)
}

func init() {
	core.Register(AvroToJSON{})
}
