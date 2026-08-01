package ops

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(BSONSerialise{})
	core.Register(BSONDeserialise{})
}

// BSON element type bytes (https://bsonspec.org/spec.html). Only the subset cchef
// reads or writes is named here.
const (
	bsonTypeDouble    = 0x01
	bsonTypeString    = 0x02
	bsonTypeDocument  = 0x03
	bsonTypeArray     = 0x04
	bsonTypeBinary    = 0x05
	bsonTypeObjectID  = 0x07
	bsonTypeBool      = 0x08
	bsonTypeDate      = 0x09
	bsonTypeNull      = 0x0A
	bsonTypeRegex     = 0x0B
	bsonTypeInt32     = 0x10
	bsonTypeTimestamp = 0x11
	bsonTypeInt64     = 0x12
	bsonTypeMinKey    = 0xFF
	bsonTypeMaxKey    = 0x7F
)

const (
	bsonObjectIDLen = 12
	bsonMinDocLen   = 5 // int32 length + terminator
)

// bsonSerialise encodes the ordered JSON value tree as a BSON document, matching
// js-bson's serialize(). The top-level value must be an object (a JSON null
// serialises to an empty document, as js-bson does); an array or scalar root is
// rejected with js-bson's exact error text.
func bsonSerialise(v any) ([]byte, error) {
	switch x := v.(type) {
	case jsObject:
		return bsonEncodeDoc(x), nil
	case nil:
		return bsonEncodeDoc(jsObject{}), nil
	case []any:
		//nolint:staticcheck,revive // js-bson's verbatim error text
		return nil, errors.New("BSONError: serialize does not support an array as the root input")
	default:
		//nolint:staticcheck,revive // js-bson's verbatim error text
		return nil, errors.New("BSONError: serialize does not support non-object as the root input")
	}
}

// bsonEncodeDoc encodes an object as a BSON document. Keys are emitted in
// ECMAScript enumeration order (integer-like keys first, ascending), matching how
// js-bson iterates a JS object.
func bsonEncodeDoc(obj jsObject) []byte {
	body := []byte{}
	for _, p := range jsESOrder(obj) {
		body = append(body, bsonEncodeElem(p.k, p.v)...)
	}
	out := make([]byte, 4, len(body)+bsonMinDocLen)
	binary.LittleEndian.PutUint32(out, uint32(len(body)+bsonMinDocLen)) // #nosec G115 -- document length is non-negative
	out = append(out, body...)
	return append(out, 0x00)
}

// bsonEncodeElem encodes a single element: type byte, key cstring, then value.
func bsonEncodeElem(key string, val any) []byte {
	typ, value := bsonEncodeValue(val)
	out := make([]byte, 0, 1+len(key)+1+len(value))
	out = append(out, typ)
	out = append(out, key...)
	out = append(out, 0x00)
	return append(out, value...)
}

// bsonEncodeValue returns the element type byte and encoded value bytes for a JSON
// value. JSON.parse only yields null/bool/number/string/array/object, so those are
// the cases reached from BSON serialise; int64 is handled too for completeness.
func bsonEncodeValue(val any) (byte, []byte) {
	switch x := val.(type) {
	case nil:
		return bsonTypeNull, nil
	case bool:
		if x {
			return bsonTypeBool, []byte{1}
		}
		return bsonTypeBool, []byte{0}
	case string:
		return bsonTypeString, bsonEncodeString(x)
	case float64:
		return bsonEncodeNumber(x)
	case int64:
		if x >= math.MinInt32 && x <= math.MaxInt32 {
			return bsonTypeInt32, binary.LittleEndian.AppendUint32(nil, uint32(int32(x))) // #nosec G115 -- writing the two's-complement bytes of an in-range int32
		}
		return bsonTypeInt64, binary.LittleEndian.AppendUint64(nil, uint64(x)) // #nosec G115 -- writing the two's-complement bytes of an int64
	case []any:
		return bsonTypeArray, bsonEncodeDoc(bsonArrayDoc(x))
	case jsObject:
		return bsonTypeDocument, bsonEncodeDoc(x)
	default:
		return bsonTypeNull, nil
	}
}

// bsonEncodeNumber applies js-bson's number-type rule: an integer-valued number in
// int32 range becomes an int32, everything else a double. Negative zero is kept as
// a double, since an int32 cannot represent its sign (matching js-bson).
func bsonEncodeNumber(f float64) (byte, []byte) {
	negZero := f == 0 && math.Signbit(f)
	if !negZero && f == math.Trunc(f) && !math.IsInf(f, 0) && f >= math.MinInt32 && f <= math.MaxInt32 {
		return bsonTypeInt32, binary.LittleEndian.AppendUint32(nil, uint32(int32(f))) // #nosec G115 -- writing the two's-complement bytes of an in-range int32
	}
	return bsonTypeDouble, binary.LittleEndian.AppendUint64(nil, math.Float64bits(f))
}

func bsonEncodeString(s string) []byte {
	out := binary.LittleEndian.AppendUint32(nil, uint32(len(s)+1)) // #nosec G115 -- string length is non-negative
	out = append(out, s...)
	return append(out, 0x00)
}

// bsonArrayDoc turns an array into a document with stringified integer keys
// "0","1",…, which is how BSON represents arrays.
func bsonArrayDoc(arr []any) jsObject {
	pairs := make(jsObject, len(arr))
	for i, e := range arr {
		pairs[i] = jsPair{k: strconv.Itoa(i), v: e}
	}
	return pairs
}

// bsonReader decodes a BSON document into the ordered JSON value tree.
type bsonReader struct {
	b   []byte
	pos int
}

// bsonDeserialise decodes a single top-level BSON document.
func bsonDeserialise(data []byte) (jsObject, error) {
	r := &bsonReader{b: data}
	doc, err := r.readDocument()
	if err != nil {
		return nil, err
	}
	if r.pos != len(data) {
		return nil, errors.New("trailing bytes after BSON document")
	}
	return doc, nil
}

func (r *bsonReader) readDocument() (jsObject, error) {
	start := r.pos
	length, err := r.readInt32()
	if err != nil {
		return nil, err
	}
	if length < bsonMinDocLen || start+int(length) > len(r.b) {
		return nil, errors.New("invalid BSON document length")
	}
	end := start + int(length)
	obj := jsObject{}
	for r.pos < end-1 {
		typ := r.b[r.pos]
		r.pos++
		key, err := r.readCString()
		if err != nil {
			return nil, err
		}
		val, err := r.readValue(typ)
		if err != nil {
			return nil, err
		}
		obj = append(obj, jsPair{k: key, v: val})
	}
	if r.pos >= len(r.b) || r.b[r.pos] != 0x00 {
		return nil, errors.New("missing BSON document terminator")
	}
	r.pos++
	if r.pos != end {
		return nil, errors.New("BSON document length mismatch")
	}
	return obj, nil
}

// readValue decodes the common element types and delegates the rest to
// readExtendedValue, keeping each dispatch small.
func (r *bsonReader) readValue(typ byte) (any, error) {
	switch typ {
	case bsonTypeDouble:
		bits, err := r.readUint64()
		return math.Float64frombits(bits), err
	case bsonTypeString:
		return r.readString()
	case bsonTypeDocument:
		return r.readDocument()
	case bsonTypeArray:
		return r.readArray()
	case bsonTypeBool:
		return r.readBool()
	case bsonTypeNull:
		return nil, nil
	case bsonTypeInt32:
		v, err := r.readInt32()
		return int64(v), err
	case bsonTypeInt64:
		v, err := r.readUint64()
		return int64(v), err // #nosec G115 -- reinterpreting 8 bytes as a signed int64
	default:
		return r.readExtendedValue(typ)
	}
}

// readExtendedValue decodes the richer BSON types, reproducing how js-bson's
// JSON.stringify renders each (ObjectId → hex string, UTC datetime → ISO string,
// Binary → base64, Timestamp → {"$timestamp": "..."}, and RegExp/MinKey/MaxKey →
// an empty object).
func (r *bsonReader) readExtendedValue(typ byte) (any, error) {
	switch typ {
	case bsonTypeObjectID:
		return r.readObjectID()
	case bsonTypeDate:
		return r.readDate()
	case bsonTypeBinary:
		return r.readBinary()
	case bsonTypeTimestamp:
		return r.readTimestamp()
	case bsonTypeRegex:
		return r.readRegex()
	case bsonTypeMinKey, bsonTypeMaxKey:
		return jsObject{}, nil
	default:
		return nil, fmt.Errorf("unsupported BSON element type 0x%02x", typ)
	}
}

func (r *bsonReader) readArray() (any, error) {
	doc, err := r.readDocument()
	if err != nil {
		return nil, err
	}
	arr := make([]any, len(doc))
	for i, p := range doc {
		arr[i] = p.v
	}
	return arr, nil
}

func (r *bsonReader) readBool() (any, error) {
	if r.pos >= len(r.b) {
		return nil, errors.New("unexpected end of BSON boolean")
	}
	b := r.b[r.pos]
	r.pos++
	return b != 0, nil
}

func (r *bsonReader) readObjectID() (any, error) {
	if r.pos+bsonObjectIDLen > len(r.b) {
		return nil, errors.New("unexpected end of BSON ObjectId")
	}
	s := hex.EncodeToString(r.b[r.pos : r.pos+bsonObjectIDLen])
	r.pos += bsonObjectIDLen
	return s, nil
}

func (r *bsonReader) readDate() (any, error) {
	ms, err := r.readUint64()
	if err != nil {
		return nil, err
	}
	t := time.UnixMilli(int64(ms)).UTC() // #nosec G115 -- reinterpreting 8 bytes as a signed epoch-millis int64
	return t.Format("2006-01-02T15:04:05.000Z07:00"), nil
}

func (r *bsonReader) readBinary() (any, error) {
	length, err := r.readInt32()
	if err != nil {
		return nil, err
	}
	// Skip the subtype byte, then read length data bytes.
	if length < 0 || r.pos+1+int(length) > len(r.b) {
		return nil, errors.New("invalid BSON binary length")
	}
	r.pos++
	data := r.b[r.pos : r.pos+int(length)]
	r.pos += int(length)
	return base64.StdEncoding.EncodeToString(data), nil
}

func (r *bsonReader) readTimestamp() (any, error) {
	v, err := r.readUint64()
	if err != nil {
		return nil, err
	}
	return jsObject{{k: "$timestamp", v: strconv.FormatUint(v, 10)}}, nil
}

func (r *bsonReader) readRegex() (any, error) {
	if _, err := r.readCString(); err != nil { // pattern
		return nil, err
	}
	if _, err := r.readCString(); err != nil { // flags
		return nil, err
	}
	return jsObject{}, nil
}

func (r *bsonReader) readInt32() (int32, error) {
	if r.pos+4 > len(r.b) {
		return 0, errors.New("unexpected end of BSON int32")
	}
	v := int32(binary.LittleEndian.Uint32(r.b[r.pos:])) // #nosec G115 -- reinterpreting 4 bytes as signed int32
	r.pos += 4
	return v, nil
}

func (r *bsonReader) readUint64() (uint64, error) {
	if r.pos+8 > len(r.b) {
		return 0, errors.New("unexpected end of BSON int64")
	}
	v := binary.LittleEndian.Uint64(r.b[r.pos:])
	r.pos += 8
	return v, nil
}

func (r *bsonReader) readCString() (string, error) {
	for i := r.pos; i < len(r.b); i++ {
		if r.b[i] == 0x00 {
			s := string(r.b[r.pos:i])
			r.pos = i + 1
			return s, nil
		}
	}
	return "", errors.New("unterminated BSON cstring")
}

func (r *bsonReader) readString() (any, error) {
	length, err := r.readInt32()
	if err != nil {
		return nil, err
	}
	if length < 1 || r.pos+int(length) > len(r.b) {
		return nil, errors.New("invalid BSON string length")
	}
	s := string(r.b[r.pos : r.pos+int(length)-1]) // exclude trailing null
	r.pos += int(length)
	return s, nil
}

// BSONSerialise struct.
type BSONSerialise struct{}

// Meta returns the operation metadata.
func (BSONSerialise) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "BSON serialise",
		Module:      "Serialise",
		Description: "BSON is a computer data interchange format used mainly as a data storage and network transfer format in the MongoDB database. It is a binary form for representing simple data structures, associative arrays (called objects or documents in MongoDB), and various data types of specific interest to MongoDB. The name 'BSON' is based on the term JSON and stands for 'Binary JSON'.\n\nInput data should be valid JSON.",
		InfoURL:     "https://wikipedia.org/wiki/BSON",
		InputType:   core.TypeString,
		OutputType:  core.TypeArrayBuffer,
	}
}

// Args returns the argument definitions.
func (BSONSerialise) Args() []core.ArgDef { return nil }

// Run serialises JSON input to BSON.
func (BSONSerialise) Run(in *core.Dish, _ []any) (*core.Dish, error) {
	input := in.String()
	if input == "" {
		return core.NewDish([]byte{}, core.TypeArrayBuffer), nil
	}
	val, err := jsonParseOrdered([]byte(input))
	if err != nil {
		return nil, fmt.Errorf("invalid JSON input: %w", err)
	}
	out, err := bsonSerialise(val)
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeArrayBuffer), nil
}

// BSONDeserialise struct.
type BSONDeserialise struct{}

// Meta returns the operation metadata.
func (BSONDeserialise) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "BSON deserialise",
		Module:      "Serialise",
		Description: "BSON is a computer data interchange format used mainly as a data storage and network transfer format in the MongoDB database. It is a binary form for representing simple data structures, associative arrays (called objects or documents in MongoDB), and various data types of specific interest to MongoDB. The name 'BSON' is based on the term JSON and stands for 'Binary JSON'.\n\nInput data should be in a raw bytes format.",
		InfoURL:     "https://wikipedia.org/wiki/BSON",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (BSONDeserialise) Args() []core.ArgDef { return nil }

// Run deserialises BSON input to JSON.
func (BSONDeserialise) Run(in *core.Dish, _ []any) (*core.Dish, error) {
	data := in.Bytes()
	if len(data) == 0 {
		return core.NewDish([]byte{}, core.TypeString), nil
	}
	doc, err := bsonDeserialise(data)
	if err != nil {
		return nil, err
	}
	return core.NewDish([]byte(jsStringify(doc, 2)), core.TypeString), nil
}
