package ops

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/jsonval"
)

func init() {
	core.Register(ToMessagePack{})
	core.Register(FromMessagePack{})
}

// two32 is 2^32 as a float64, used to reproduce notepack's JavaScript
// 32-bit-word splitting of 64-bit integers.
const two32 = 4294967296.0

const (
	toMessagePackDesc   = "Converts JSON to MessagePack encoded byte buffer. MessagePack is a computer data interchange format. It is a binary form for representing simple data structures like arrays and associative arrays."
	fromMessagePackDesc = "Converts MessagePack encoded data to JSON. MessagePack is a computer data interchange format. It is a binary form for representing simple data structures like arrays and associative arrays."
)

// ToMessagePack serialises JSON into MessagePack.
//
// CyberChef wraps the `notepack.io` library's encode; this is an in-repo
// port of its rules: shortest integer forms, the always-float-64 fallback for
// non-integers, JavaScript ECMAScript key ordering, and the (lossy) bit
// operations notepack uses to split integers beyond 2^32.
type ToMessagePack struct{}

// Meta returns the operation metadata.
func (ToMessagePack) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "To MessagePack",
		Module:      "Code",
		Description: toMessagePackDesc,
		InfoURL:     "https://wikipedia.org/wiki/MessagePack",
		InputType:   core.TypeJSON,
		OutputType:  core.TypeArrayBuffer,
	}
}

// Args returns the argument definitions.
func (ToMessagePack) Args() []core.ArgDef { return nil }

// Run encodes JSON input into MessagePack bytes.
func (ToMessagePack) Run(in *core.Dish, _ []any) (*core.Dish, error) {
	v, err := jsonval.ParseOrdered(in.Bytes())
	if err != nil {
		return nil, fmt.Errorf("encode to MessagePack: parse JSON input: %w", err)
	}
	var buf bytes.Buffer
	if err := msgpackEncode(&buf, v); err != nil {
		return nil, fmt.Errorf("encode to MessagePack: %w", err)
	}
	return core.NewDish(buf.Bytes(), core.TypeArrayBuffer), nil
}

// FromMessagePack deserialises MessagePack into JSON.
//
// CyberChef wraps the `notepack.io` library's decode; this reproduces its value
// model: byte strings render as Node Buffers, map keys are String()-coerced,
// integers beyond 2^53 lose precision to floats, undefined markers are omitted
// from objects, ext types render as [type, Buffer] and timestamp extensions
// become ISO date strings.
type FromMessagePack struct{}

// Meta returns the operation metadata.
func (FromMessagePack) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "From MessagePack",
		Module:      "Code",
		Description: fromMessagePackDesc,
		InfoURL:     "https://wikipedia.org/wiki/MessagePack",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeJSON,
	}
}

// Args returns the argument definitions.
func (FromMessagePack) Args() []core.ArgDef { return nil }

// Run decodes MessagePack bytes into JSON.
func (FromMessagePack) Run(in *core.Dish, _ []any) (*core.Dish, error) {
	v, err := msgpackDecode(in.Bytes())
	if err != nil {
		return nil, fmt.Errorf("decode from MessagePack: %w", err)
	}
	if _, ok := v.(jsonval.Undefined); ok {
		// JSON.stringify(undefined) is undefined; CyberChef emits empty output.
		return core.NewDish([]byte{}, core.TypeJSON), nil
	}
	return core.NewDish([]byte(jsonval.Stringify(v, 4)), core.TypeJSON), nil
}

// --- encoder ---

// msgpackTooLong reports whether a container/string length overflows
// MessagePack's 32-bit length field, which notepack rejects.
func msgpackTooLong(n int) bool { return uint64(n) >= 1<<32 } // #nosec G115 -- len is non-negative

func msgpackEncode(buf *bytes.Buffer, v any) error {
	switch x := v.(type) {
	case nil:
		buf.WriteByte(0xc0)
	case bool:
		if x {
			buf.WriteByte(0xc3)
		} else {
			buf.WriteByte(0xc2)
		}
	case string:
		return msgpackEncodeString(buf, x)
	case float64:
		msgpackEncodeNumber(buf, x)
	case []any:
		return msgpackEncodeArray(buf, x)
	case jsonval.Object:
		return msgpackEncodeMap(buf, x)
	default:
		return fmt.Errorf("unsupported value type %T", v)
	}
	return nil
}

func msgpackEncodeString(buf *bytes.Buffer, s string) error {
	n := len(s)
	switch {
	case n < 0x20:
		buf.WriteByte(0xa0 | byte(n))
	case n < 0x100:
		buf.WriteByte(0xd9)
		buf.WriteByte(byte(n))
	case n < 0x10000:
		buf.WriteByte(0xda)
		writeBE(buf, uint16(n)) // #nosec G115 -- bounded by the case above
	case !msgpackTooLong(n):
		buf.WriteByte(0xdb)
		writeBE(buf, uint32(n)) // #nosec G115 -- bounded by msgpackTooLong
	default:
		return errors.New("String too long") //nolint:staticcheck // verbatim notepack.io error text
	}
	buf.WriteString(s)
	return nil
}

// msgpackEncodeNumber ports notepack's number encoding: any non-integer (or
// non-finite) value becomes a float 64; integers take the shortest fixnum/uint/
// int form, splitting values beyond 2^32 with JavaScript's 32-bit word
// operations (which lose precision, and thus fidelity, exactly as notepack
// does).
func msgpackEncodeNumber(buf *bytes.Buffer, f float64) {
	if f != math.Floor(f) || math.IsInf(f, 0) {
		buf.WriteByte(0xcb)
		writeBE(buf, math.Float64bits(f))
		return
	}
	if f >= 0 {
		switch {
		case f < 0x80:
			buf.WriteByte(byte(uint64(f))) // #nosec G115 -- value < 0x80 by the case guard
		case f < 0x100:
			buf.WriteByte(0xcc)
			buf.WriteByte(byte(uint64(f))) // #nosec G115 -- value < 0x100 by the case guard
		case f < 0x10000:
			buf.WriteByte(0xcd)
			writeBE(buf, uint16(uint64(f))) // #nosec G115 -- value < 0x10000 by the case guard
		case f < two32:
			buf.WriteByte(0xce)
			writeBE(buf, uint32(uint64(f))) // #nosec G115 -- value < 2^32 by the case guard
		default:
			buf.WriteByte(0xcf)
			writeBE(buf, jsToUint32(f/two32))
			writeBE(buf, jsToUint32(f))
		}
		return
	}
	iv := int64(f)
	switch {
	case f >= -0x20:
		buf.WriteByte(byte(iv)) // #nosec G115 -- low 8 bits (negative fixint), matching notepack
	case f >= -0x80:
		buf.WriteByte(0xd0)
		buf.WriteByte(byte(iv)) // #nosec G115 -- low 8 bits (int8), matching notepack
	case f >= -0x8000:
		buf.WriteByte(0xd1)
		writeBE(buf, uint16(iv)) // #nosec G115 -- low 16 bits, matching notepack
	case f >= -two32/2:
		buf.WriteByte(0xd2)
		writeBE(buf, uint32(iv)) // #nosec G115 -- low 32 bits, matching notepack
	default:
		buf.WriteByte(0xd3)
		writeBE(buf, jsToUint32(math.Floor(f/two32)))
		writeBE(buf, jsToUint32(f))
	}
}

func msgpackEncodeArray(buf *bytes.Buffer, arr []any) error {
	n := len(arr)
	switch {
	case n < 0x10:
		buf.WriteByte(0x90 | byte(n))
	case n < 0x10000:
		buf.WriteByte(0xdc)
		writeBE(buf, uint16(n)) // #nosec G115 -- bounded by the case above
	case !msgpackTooLong(n):
		buf.WriteByte(0xdd)
		writeBE(buf, uint32(n)) // #nosec G115 -- bounded by msgpackTooLong
	default:
		return errors.New("Array too large") //nolint:staticcheck // verbatim notepack.io error text
	}
	for _, e := range arr {
		if err := msgpackEncode(buf, e); err != nil {
			return err
		}
	}
	return nil
}

func msgpackEncodeMap(buf *bytes.Buffer, obj jsonval.Object) error {
	// notepack enumerates Object.keys (ECMAScript order) and drops keys whose
	// value is undefined; JSON input never yields undefined, but the filter is
	// kept for fidelity.
	pairs := make(jsonval.Object, 0, len(obj))
	for _, p := range jsonval.ESOrder(obj) {
		if _, ok := p.V.(jsonval.Undefined); ok {
			continue
		}
		pairs = append(pairs, p)
	}
	n := len(pairs)
	switch {
	case n < 0x10:
		buf.WriteByte(0x80 | byte(n))
	case n < 0x10000:
		buf.WriteByte(0xde)
		writeBE(buf, uint16(n)) // #nosec G115 -- bounded by the case above
	case !msgpackTooLong(n):
		buf.WriteByte(0xdf)
		writeBE(buf, uint32(n)) // #nosec G115 -- bounded by msgpackTooLong
	default:
		return errors.New("Object too large") //nolint:staticcheck // verbatim notepack.io error text
	}
	for _, p := range pairs {
		if err := msgpackEncodeString(buf, p.K); err != nil {
			return err
		}
		if err := msgpackEncode(buf, p.V); err != nil {
			return err
		}
	}
	return nil
}

// --- decoder ---

type mreader struct {
	data []byte
	pos  int
}

func (r *mreader) take(n int) ([]byte, error) {
	if n < 0 || n > len(r.data)-r.pos {
		return nil, errors.New("unexpected end of MessagePack data")
	}
	b := r.data[r.pos : r.pos+n]
	r.pos += n
	return b, nil
}

// canHold rejects a container header that declares more elements than the rest
// of the input could supply. Every element is at least one byte, so a count
// beyond that is malformed — and sizing the result from it first would let a
// few bytes of header ask for an unbounded allocation.
func (r *mreader) canHold(n int) error {
	if n < 0 || n > len(r.data)-r.pos {
		return errors.New("unexpected end of MessagePack data")
	}
	return nil
}

func (r *mreader) u8() (uint64, error) {
	b, err := r.take(1)
	if err != nil {
		return 0, err
	}
	return uint64(b[0]), nil
}

func (r *mreader) u16() (uint64, error) {
	b, err := r.take(2)
	if err != nil {
		return 0, err
	}
	return uint64(binary.BigEndian.Uint16(b)), nil
}

func (r *mreader) u32() (uint64, error) {
	b, err := r.take(4)
	if err != nil {
		return 0, err
	}
	return uint64(binary.BigEndian.Uint32(b)), nil
}

// msgpackDecode decodes a single MessagePack item, requiring that it consumes
// all of data (trailing bytes are an error, matching notepack).
func msgpackDecode(data []byte) (any, error) {
	r := &mreader{data: data}
	v, err := msgpackParse(r)
	if err != nil {
		return nil, err
	}
	if r.pos != len(data) {
		return nil, errors.New("trailing bytes after MessagePack item")
	}
	return v, nil
}

func msgpackParse(r *mreader) (any, error) {
	p, err := r.u8()
	if err != nil {
		return nil, err
	}
	prefix := byte(p) // #nosec G115 -- u8 returns a single byte value
	switch {
	case prefix < 0x80: // positive fixint
		return int64(prefix), nil
	case prefix < 0x90: // fixmap
		return msgpackMap(r, int(prefix&0x0f))
	case prefix < 0xa0: // fixarray
		return msgpackArray(r, int(prefix&0x0f))
	case prefix < 0xc0: // fixstr
		return msgpackStr(r, int(prefix&0x1f))
	case prefix > 0xdf: // negative fixint
		return int64(int8(prefix)), nil // #nosec G115 -- reinterpret the fixint byte as signed
	}
	return msgpackParsePrefixed(r, prefix)
}

// msgpackParsePrefixed handles the 0xc0..0xdf leading bytes.
func msgpackParsePrefixed(r *mreader, prefix byte) (any, error) {
	switch prefix {
	case 0xc0:
		return nil, nil
	case 0xc2:
		return false, nil
	case 0xc3:
		return true, nil
	case 0xc4, 0xc5, 0xc6:
		return msgpackBin(r, prefix)
	case 0xc7, 0xc8, 0xc9:
		return msgpackExt(r, prefix)
	case 0xca, 0xcb:
		return msgpackFloat(r, prefix)
	case 0xcc, 0xcd, 0xce, 0xcf, 0xd0, 0xd1, 0xd2, 0xd3:
		return msgpackNumber(r, prefix)
	case 0xd4, 0xd5, 0xd6, 0xd7, 0xd8:
		return msgpackFixExt(r, prefix)
	case 0xd9, 0xda, 0xdb, 0xdc, 0xdd, 0xde, 0xdf:
		return msgpackParseSized(r, prefix)
	default: // 0xc1 and any gap
		return nil, errors.New("could not parse MessagePack byte")
	}
}

// msgpackFloat decodes a 32-bit (0xca) or 64-bit (0xcb) IEEE-754 float.
func msgpackFloat(r *mreader, prefix byte) (any, error) {
	if prefix == 0xca {
		b, err := r.take(4)
		if err != nil {
			return nil, err
		}
		return float64(math.Float32frombits(binary.BigEndian.Uint32(b))), nil
	}
	b, err := r.take(8)
	if err != nil {
		return nil, err
	}
	return math.Float64frombits(binary.BigEndian.Uint64(b)), nil
}

// msgpackParseSized decodes the length-prefixed str/array/map families, whose
// length is an 8-, 16- or 32-bit big-endian count.
func msgpackParseSized(r *mreader, prefix byte) (any, error) {
	switch prefix {
	case 0xd9:
		return msgpackStrN(r, r.u8)
	case 0xda:
		return msgpackStrN(r, r.u16)
	case 0xdb:
		return msgpackStrN(r, r.u32)
	case 0xdc:
		return msgpackArrayN(r, r.u16)
	case 0xdd:
		return msgpackArrayN(r, r.u32)
	case 0xde:
		return msgpackMapN(r, r.u16)
	default: // 0xdf
		return msgpackMapN(r, r.u32)
	}
}

// msgpackNumber decodes the fixed-width integer families. uint64/int64 are
// widened through float64 exactly as notepack does, so values beyond 2^53 lose
// precision.
func msgpackNumber(r *mreader, prefix byte) (any, error) {
	switch prefix {
	case 0xcc:
		v, err := r.u8()
		return int64(v), err // #nosec G115 -- 8-bit value fits int64
	case 0xcd:
		v, err := r.u16()
		return int64(v), err // #nosec G115 -- 16-bit value fits int64
	case 0xce:
		v, err := r.u32()
		return int64(v), err // #nosec G115 -- 32-bit value fits int64
	case 0xcf:
		hi, err := r.u32()
		if err != nil {
			return nil, err
		}
		lo, err := r.u32()
		if err != nil {
			return nil, err
		}
		return float64(hi)*two32 + float64(lo), nil
	case 0xd0:
		b, err := r.take(1)
		if err != nil {
			return nil, err
		}
		return int64(int8(b[0])), nil // #nosec G115 -- reinterpret the int8 byte as signed
	case 0xd1:
		v, err := r.u16()
		if err != nil {
			return nil, err
		}
		return int64(int16(v)), nil // #nosec G115 -- reinterpret 16 bits as signed
	case 0xd2:
		v, err := r.u32()
		if err != nil {
			return nil, err
		}
		return int64(int32(v)), nil // #nosec G115 -- reinterpret 32 bits as signed
	default: // 0xd3
		hi, err := r.u32()
		if err != nil {
			return nil, err
		}
		lo, err := r.u32()
		if err != nil {
			return nil, err
		}
		return float64(int32(hi))*two32 + float64(lo), nil // #nosec G115 -- reinterpret high word as signed
	}
}

func msgpackStr(r *mreader, n int) (any, error) {
	b, err := r.take(n)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func msgpackStrN(r *mreader, readLen func() (uint64, error)) (any, error) {
	n, err := readLen()
	if err != nil {
		return nil, err
	}
	return msgpackStr(r, int(n)) // #nosec G115 -- length validated by take
}

func msgpackBin(r *mreader, prefix byte) (any, error) {
	var n uint64
	var err error
	switch prefix {
	case 0xc4:
		n, err = r.u8()
	case 0xc5:
		n, err = r.u16()
	default: // 0xc6
		n, err = r.u32()
	}
	if err != nil {
		return nil, err
	}
	b, err := r.take(int(n)) // #nosec G115 -- length validated by take
	if err != nil {
		return nil, err
	}
	return jsonval.Buffer(b), nil
}

func msgpackArray(r *mreader, n int) (any, error) {
	if err := r.canHold(n); err != nil {
		return nil, err
	}
	arr := make([]any, n)
	for i := range arr {
		v, err := msgpackParse(r)
		if err != nil {
			return nil, err
		}
		arr[i] = v
	}
	return arr, nil
}

func msgpackArrayN(r *mreader, readLen func() (uint64, error)) (any, error) {
	n, err := readLen()
	if err != nil {
		return nil, err
	}
	return msgpackArray(r, int(n)) // #nosec G115 -- element reads are bounds-checked
}

func msgpackMap(r *mreader, n int) (any, error) {
	// A pair is two values, so it needs at least two bytes.
	if err := r.canHold(2 * n); err != nil {
		return nil, err
	}
	obj := jsonval.Object{}
	for range n {
		k, err := msgpackParse(r)
		if err != nil {
			return nil, err
		}
		v, err := msgpackParse(r)
		if err != nil {
			return nil, err
		}
		key := jsToString(k)
		if i := jsonval.Index(obj, key); i >= 0 {
			obj[i].V = v
		} else {
			obj = append(obj, jsonval.Pair{K: key, V: v})
		}
	}
	return obj, nil
}

func msgpackMapN(r *mreader, readLen func() (uint64, error)) (any, error) {
	n, err := readLen()
	if err != nil {
		return nil, err
	}
	return msgpackMap(r, int(n)) // #nosec G115 -- pair reads are bounds-checked
}

// msgpackExt decodes the ext family (0xc7/0xc8/0xc9). Type 0 is notepack's
// backward-compatible ArrayBuffer, type -1 is a timestamp; any other type
// yields [type, Buffer].
func msgpackExt(r *mreader, prefix byte) (any, error) {
	var n uint64
	var err error
	switch prefix {
	case 0xc7:
		n, err = r.u8()
	case 0xc8:
		n, err = r.u16()
	default: // 0xc9
		n, err = r.u32()
	}
	if err != nil {
		return nil, err
	}
	tb, err := r.take(1)
	if err != nil {
		return nil, err
	}
	typ := int8(tb[0]) // #nosec G115 -- reinterpret the ext type byte as signed
	if typ == 0 {
		// ArrayBuffer renders as {} under JSON.stringify.
		if _, err := r.take(int(n)); err != nil { // #nosec G115 -- length validated by take
			return nil, err
		}
		return jsonval.Object{}, nil
	}
	if typ == -1 && prefix == 0xc7 {
		return msgpackTimestamp96(r)
	}
	b, err := r.take(int(n)) // #nosec G115 -- length validated by take
	if err != nil {
		return nil, err
	}
	return []any{int64(typ), jsonval.Buffer(b)}, nil
}

// msgpackFixExt decodes the fixext family (0xd4..0xd8). Type 0 on fixext1 is
// notepack's undefined marker; type 0 on fixext8 is a backward-compatible date;
// type -1 on fixext4/fixext8 are timestamps; any other yields [type, Buffer].
func msgpackFixExt(r *mreader, prefix byte) (any, error) {
	sizes := map[byte]int{0xd4: 1, 0xd5: 2, 0xd6: 4, 0xd7: 8, 0xd8: 16}
	size := sizes[prefix]
	tb, err := r.take(1)
	if err != nil {
		return nil, err
	}
	typ := int8(tb[0]) // #nosec G115 -- reinterpret the ext type byte as signed
	switch {
	case prefix == 0xd4 && typ == 0:
		if _, err := r.take(1); err != nil {
			return nil, err
		}
		return jsonval.Undefined{}, nil
	case prefix == 0xd6 && typ == -1:
		return msgpackTimestamp32(r)
	case prefix == 0xd7 && typ == 0:
		return msgpackCustomDate(r)
	case prefix == 0xd7 && typ == -1:
		return msgpackTimestamp64(r)
	}
	b, err := r.take(size)
	if err != nil {
		return nil, err
	}
	return []any{int64(typ), jsonval.Buffer(b)}, nil
}

// --- timestamps ---

func msgpackTimestamp32(r *mreader) (any, error) {
	s, err := r.u32()
	if err != nil {
		return nil, err
	}
	return jsDateToISO(float64(s) * 1e3), nil
}

func msgpackTimestamp64(r *mreader) (any, error) {
	hi, err := r.u32()
	if err != nil {
		return nil, err
	}
	lo, err := r.u32()
	if err != nil {
		return nil, err
	}
	s := float64(hi&0x3)*two32 + float64(lo)
	return jsDateToISO(s*1e3 + float64(hi>>2)/1e6), nil
}

func msgpackTimestamp96(r *mreader) (any, error) {
	ns, err := r.u32()
	if err != nil {
		return nil, err
	}
	hi, err := r.u32()
	if err != nil {
		return nil, err
	}
	lo, err := r.u32()
	if err != nil {
		return nil, err
	}
	return jsDateToISO((float64(int32(hi))*two32+float64(lo))*1e3 + float64(ns)/1e6), nil // #nosec G115 -- reinterpret high word as signed
}

func msgpackCustomDate(r *mreader) (any, error) {
	hi, err := r.u32()
	if err != nil {
		return nil, err
	}
	lo, err := r.u32()
	if err != nil {
		return nil, err
	}
	return jsDateToISO(float64(int32(hi))*two32 + float64(lo)), nil // #nosec G115 -- reinterpret high word as signed
}

// jsDateToISO renders an epoch-milliseconds value the way
// new Date(ms).toISOString() does, including the extended ±YYYYYY year form for
// dates outside 0000-9999. A value outside the valid Date range (±8.64e15 ms) or
// non-finite becomes null, matching JSON.stringify of an Invalid Date.
func jsDateToISO(ms float64) any {
	if math.IsNaN(ms) || math.IsInf(ms, 0) || math.Abs(ms) > 8.64e15 {
		return nil
	}
	t := time.UnixMilli(int64(ms)).UTC()
	y := t.Year()
	var year string
	switch {
	case y >= 0 && y <= 9999:
		year = fmt.Sprintf("%04d", y)
	case y < 0:
		year = fmt.Sprintf("-%06d", -y)
	default:
		year = fmt.Sprintf("+%06d", y)
	}
	return year + fmt.Sprintf("-%02d-%02dT%02d:%02d:%02d.%03dZ",
		int(t.Month()), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond()/1e6)
}

// --- JavaScript String() coercion for map keys ---

// jsArrayJoin reproduces Array.prototype.toString: elements joined by commas,
// with null and undefined rendered as the empty string.
func jsArrayJoin(a []any) string {
	parts := make([]string, len(a))
	for i, e := range a {
		if e == nil {
			continue
		}
		if _, ok := e.(jsonval.Undefined); ok {
			continue
		}
		parts[i] = jsToString(e)
	}
	return strings.Join(parts, ",")
}
