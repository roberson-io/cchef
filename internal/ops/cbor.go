package ops

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"slices"
	"sort"
	"time"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(CBOREncode{})
	core.Register(CBORDecode{})
}

// maxSafeInt is JavaScript's Number.MAX_SAFE_INTEGER (2^53 - 1). The cbor
// library encodes integer-valued numbers within this magnitude as CBOR
// integers and everything else as floats, and decodes integers beyond it to
// BigInt (which JSON.stringify then refuses to serialise).
const maxSafeInt = 9007199254740991

// cborDesc is the shared description of both CBOR operations.
const cborDesc = "Concise Binary Object Representation (CBOR) is a binary data serialization format loosely based on JSON. Like JSON it allows the transmission of data objects that contain name–value pairs, but in a more concise manner. This increases processing and transfer speeds at the cost of human readability. It is defined in IETF RFC 8949."

// CBOREncode serializes JSON into canonical CBOR (RFC 8949).
//
// CyberChef wraps the `cbor` npm library's Cbor.encodeCanonical; this is a
// from-scratch port of its canonical rules: shortest integer/float forms
// (including half-precision and the negative-zero special case), the
// MAX_SAFE_INTEGER-to-float fallback, and length-first map key ordering.
type CBOREncode struct{}

// Meta returns the operation metadata.
func (CBOREncode) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "CBOR Encode",
		Module:      "Serialise",
		Description: cborDesc,
		InfoURL:     "https://wikipedia.org/wiki/CBOR",
		InputType:   core.TypeJSON,
		OutputType:  core.TypeArrayBuffer,
	}
}

// Args returns the argument definitions.
func (CBOREncode) Args() []core.ArgDef { return nil }

// Run encodes JSON input into canonical CBOR bytes.
func (CBOREncode) Run(in *core.Dish, _ []any) (*core.Dish, error) {
	dec := json.NewDecoder(bytes.NewReader(in.Bytes()))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, fmt.Errorf("CBOR encode: parse JSON input: %w", err)
	}
	if dec.More() {
		return nil, errors.New("CBOR encode: trailing data after JSON value")
	}
	var buf bytes.Buffer
	if err := cborEncodeValue(&buf, v); err != nil {
		return nil, fmt.Errorf("CBOR encode: %w", err)
	}
	return core.NewDish(buf.Bytes(), core.TypeArrayBuffer), nil
}

// CBORDecode deserializes CBOR (RFC 8949) into JSON.
//
// CyberChef wraps the `cbor` library's Cbor.decodeFirstSync; this reproduces its
// value model: byte strings render as Node Buffers, string-keyed maps keep their
// order (non-string keys collapse to {}), integers beyond MAX_SAFE_INTEGER and
// bignum tags become BigInts that fail to serialise, and date tags 0/1 render as
// ISO strings.
type CBORDecode struct{}

// Meta returns the operation metadata.
func (CBORDecode) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "CBOR Decode",
		Module:      "Serialise",
		Description: cborDesc,
		InfoURL:     "https://wikipedia.org/wiki/CBOR",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeJSON,
	}
}

// Args returns the argument definitions.
func (CBORDecode) Args() []core.ArgDef { return nil }

// Run decodes CBOR bytes into JSON.
func (CBORDecode) Run(in *core.Dish, _ []any) (*core.Dish, error) {
	v, err := cborDecode(in.Bytes())
	if err != nil {
		return nil, fmt.Errorf("CBOR decode: %w", err)
	}
	if cborHasBigInt(v) {
		return nil, errors.New("Do not know how to serialize a BigInt") //nolint:staticcheck,revive // verbatim JSON.stringify(BigInt) TypeError text CyberChef surfaces
	}
	if _, ok := v.(jsUndefined); ok {
		// JSON.stringify(undefined) is undefined; CyberChef emits empty output.
		return core.NewDish([]byte{}, core.TypeJSON), nil
	}
	return core.NewDish([]byte(jsStringify(v, 4)), core.TypeJSON), nil
}

// --- encoder ---

func cborEncodeValue(buf *bytes.Buffer, v any) error {
	switch x := v.(type) {
	case nil:
		buf.WriteByte(0xf6)
	case bool:
		if x {
			buf.WriteByte(0xf5)
		} else {
			buf.WriteByte(0xf4)
		}
	case string:
		cborPutHead(buf, 0x60, uint64(len(x)))
		buf.WriteString(x)
	case json.Number:
		f, err := x.Float64()
		if err != nil {
			return err
		}
		cborEncodeNumber(buf, f)
	case []any:
		cborPutHead(buf, 0x80, uint64(len(x)))
		for _, e := range x {
			if err := cborEncodeValue(buf, e); err != nil {
				return err
			}
		}
	case map[string]any:
		cborPutHead(buf, 0xa0, uint64(len(x)))
		for _, k := range cborSortedKeys(x) {
			cborPutHead(buf, 0x60, uint64(len(k)))
			buf.WriteString(k)
			if err := cborEncodeValue(buf, x[k]); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unsupported value type %T", v)
	}
	return nil
}

// cborEncodeNumber encodes a JS number, matching cbor's _pushNumber: an
// integer-valued number within MAX_SAFE_INTEGER becomes a CBOR integer (with -0
// as a special half-float), everything else a shortest-form float.
func cborEncodeNumber(buf *bytes.Buffer, f float64) {
	if f == math.Trunc(f) && !math.IsInf(f, 0) {
		if f == 0 && math.Signbit(f) {
			buf.Write([]byte{0xf9, 0x80, 0x00})
			return
		}
		if f < 0 {
			cborEncodeInt(buf, -f-1, 0x20, f)
		} else {
			cborEncodeInt(buf, f, 0x00, f)
		}
		return
	}
	cborEncodeFloat(buf, f)
}

// cborEncodeInt encodes a non-negative integer-valued n under the given major
// type. Beyond MAX_SAFE_INTEGER it falls back to a float of the original signed
// value, exactly as cbor's _pushInt does.
func cborEncodeInt(buf *bytes.Buffer, n float64, major byte, orig float64) {
	limit := float64(maxSafeInt)
	if major == 0x20 { // negative integers stop one short of the safe max
		limit--
	}
	if n > limit {
		cborEncodeFloat(buf, orig)
		return
	}
	cborPutHead(buf, major, uint64(n))
}

// cborEncodeFloat writes f as the shortest of half/single/double that preserves
// its value, matching cbor's canonical _pushFloat.
func cborEncodeFloat(buf *bytes.Buffer, f float64) {
	if h, ok := cborHalf(f); ok {
		buf.WriteByte(0xf9)
		writeBE(buf, h)
		return
	}
	if float64(float32(f)) == f {
		buf.WriteByte(0xfa)
		writeBE(buf, math.Float32bits(float32(f)))
		return
	}
	buf.WriteByte(0xfb)
	writeBE(buf, math.Float64bits(f))
}

// cborHalf ports cbor's utils.writeHalf: it returns the IEEE-754 half-precision
// bits of f when f is exactly representable as one (f is assumed non-zero,
// finite and non-NaN, which the callers guarantee).
func cborHalf(f float64) (uint16, bool) {
	u := math.Float32bits(float32(f))
	if u&0x1fff != 0 {
		return 0, false
	}
	s16 := (u >> 16) & 0x8000
	exp := (u >> 23) & 0xff
	mant := u & 0x7fffff
	switch {
	case exp >= 113 && exp <= 142:
		s16 += ((exp - 112) << 10) + (mant >> 13)
	case exp >= 103 && exp < 113:
		if mant&((uint32(1)<<(126-exp))-1) != 0 {
			return 0, false
		}
		s16 += (mant + 0x800000) >> (126 - exp)
	default:
		return 0, false
	}
	return uint16(s16), true // #nosec G115 -- half-float bits fit in 16 by construction (sign | 15-bit field)
}

// cborPutHead writes a CBOR type byte for major with the shortest encoding of n
// (a length or a small integer value).
func cborPutHead(buf *bytes.Buffer, major byte, n uint64) {
	switch {
	case n < 24:
		buf.WriteByte(major | byte(n))
	case n <= 0xff:
		buf.WriteByte(major | 24)
		buf.WriteByte(byte(n))
	case n <= 0xffff:
		buf.WriteByte(major | 25)
		writeBE(buf, uint16(n))
	case n <= 0xffffffff:
		buf.WriteByte(major | 26)
		writeBE(buf, uint32(n))
	default:
		buf.WriteByte(major | 27)
		writeBE(buf, n)
	}
}

// cborSortedKeys returns the map's keys in canonical (length-first) order: keys
// are compared by their CBOR encodings bytewise, exactly as cbor sorts them.
func cborSortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	enc := make(map[string][]byte, len(m))
	for k := range m {
		var b bytes.Buffer
		cborPutHead(&b, 0x60, uint64(len(k)))
		b.WriteString(k)
		enc[k] = b.Bytes()
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return bytes.Compare(enc[keys[i]], enc[keys[j]]) < 0
	})
	return keys
}

// writeBE writes v to buf in big-endian order.
func writeBE[T uint16 | uint32 | uint64](buf *bytes.Buffer, v T) {
	switch x := any(v).(type) {
	case uint16:
		var b [2]byte
		binary.BigEndian.PutUint16(b[:], x)
		buf.Write(b[:])
	case uint32:
		var b [4]byte
		binary.BigEndian.PutUint32(b[:], x)
		buf.Write(b[:])
	case uint64:
		var b [8]byte
		binary.BigEndian.PutUint64(b[:], x)
		buf.Write(b[:])
	}
}

// --- decoder ---

// cborBigInt marks an integer decoded outside the JavaScript safe range (or a
// bignum tag): the cbor library yields a BigInt for these, which JSON.stringify
// cannot serialise.
type cborBigInt struct{}

type creader struct {
	data []byte
	pos  int
}

func (r *creader) eof() bool { return r.pos >= len(r.data) }

func (r *creader) readByte() (byte, error) {
	if r.pos >= len(r.data) {
		return 0, io.ErrUnexpectedEOF
	}
	b := r.data[r.pos]
	r.pos++
	return b, nil
}

func (r *creader) peek() (byte, error) {
	if r.pos >= len(r.data) {
		return 0, io.ErrUnexpectedEOF
	}
	return r.data[r.pos], nil
}

func (r *creader) take(n uint64) ([]byte, error) {
	if n > uint64(len(r.data)-r.pos) { // #nosec G115 -- pos <= len is a reader invariant, so the difference is non-negative
		return nil, io.ErrUnexpectedEOF
	}
	b := r.data[r.pos : r.pos+int(n)] // #nosec G115 -- n bounded by the length check above
	r.pos += int(n)                   // #nosec G115 -- n bounded by the length check above
	return b, nil
}

// cborDecode decodes a single CBOR data item, requiring that it consumes all of
// data (trailing bytes are an error, matching decodeFirstSync).
func cborDecode(data []byte) (any, error) {
	r := &creader{data: data}
	v, err := cborReadValue(r)
	if err != nil {
		return nil, err
	}
	if !r.eof() {
		return nil, errors.New("trailing data after CBOR item")
	}
	return v, nil
}

func cborReadValue(r *creader) (any, error) {
	ib, err := r.readByte()
	if err != nil {
		return nil, err
	}
	major := ib >> cborMajorShift
	ai := ib & cborAIMask
	switch major {
	case 0, 1:
		return cborReadInt(r, ai, major)
	case 2, 3:
		return cborReadStringValue(r, ai, major)
	case 4:
		return cborReadArray(r, ai)
	case 5:
		return cborReadMap(r, ai)
	case 6:
		return cborReadTag(r, ai)
	default: // 7
		return cborReadSimple(r, ai)
	}
}

// A CBOR initial byte packs the major type in its high 3 bits and the
// additional-info in its low 5 bits.
const (
	cborMajorShift = 5
	cborAIMask     = 0x1f
)

// cborReadInt reads an unsigned (major 0) or negative (major 1) integer,
// promoting values beyond JS's safe range to a bigint.
func cborReadInt(r *creader, ai, major byte) (any, error) {
	n, err := cborReadArg(r, ai)
	if err != nil {
		return nil, err
	}
	if major == 0 {
		if n > maxSafeInt {
			return cborBigInt{}, nil
		}
		return int64(n), nil
	}
	if n >= maxSafeInt {
		return cborBigInt{}, nil
	}
	return -1 - int64(n), nil
}

// cborReadStringValue reads a byte string (major 2, as a Buffer) or text string
// (major 3, as a Go string).
func cborReadStringValue(r *creader, ai, major byte) (any, error) {
	b, err := cborReadString(r, ai, major)
	if err != nil {
		return nil, err
	}
	if major == 2 {
		return jsBuffer(b), nil
	}
	return string(b), nil
}

// cborReadTag reads a tag (major 6) and its tagged value, then applies the tag's
// semantics.
func cborReadTag(r *creader, ai byte) (any, error) {
	tag, err := cborReadArg(r, ai)
	if err != nil {
		return nil, err
	}
	inner, err := cborReadValue(r)
	if err != nil {
		return nil, err
	}
	return cborApplyTag(tag, inner), nil
}

// cborReadArg reads the argument (length or value) that follows an initial byte
// with additional-info ai. Indefinite (31) and reserved (28-30) are rejected.
func cborReadArg(r *creader, ai byte) (uint64, error) {
	switch {
	case ai < 24:
		return uint64(ai), nil
	case ai == 24:
		b, err := r.readByte()
		return uint64(b), err
	case ai == 25:
		b, err := r.take(2)
		if err != nil {
			return 0, err
		}
		return uint64(binary.BigEndian.Uint16(b)), nil
	case ai == 26:
		b, err := r.take(4)
		if err != nil {
			return 0, err
		}
		return uint64(binary.BigEndian.Uint32(b)), nil
	case ai == 27:
		b, err := r.take(8)
		if err != nil {
			return 0, err
		}
		return binary.BigEndian.Uint64(b), nil
	default:
		return 0, fmt.Errorf("invalid additional information %d", ai)
	}
}

// cborReadString reads a definite- or indefinite-length byte/text string; the
// chunks of an indefinite string must share its major type.
func cborReadString(r *creader, ai, major byte) ([]byte, error) {
	if ai == 31 {
		var out []byte
		for {
			b, err := r.readByte()
			if err != nil {
				return nil, err
			}
			if b == 0xff {
				return out, nil
			}
			if b>>5 != major || b&0x1f == 31 {
				return nil, errors.New("invalid chunk in indefinite string")
			}
			chunk, err := cborReadString(r, b&0x1f, major)
			if err != nil {
				return nil, err
			}
			out = append(out, chunk...)
		}
	}
	n, err := cborReadArg(r, ai)
	if err != nil {
		return nil, err
	}
	return r.take(n)
}

func cborReadArray(r *creader, ai byte) (any, error) {
	arr := []any{}
	if ai == 31 {
		for {
			b, err := r.peek()
			if err != nil {
				return nil, err
			}
			if b == 0xff {
				r.pos++
				return arr, nil
			}
			v, err := cborReadValue(r)
			if err != nil {
				return nil, err
			}
			arr = append(arr, v)
		}
	}
	n, err := cborReadArg(r, ai)
	if err != nil {
		return nil, err
	}
	for range n {
		v, err := cborReadValue(r)
		if err != nil {
			return nil, err
		}
		arr = append(arr, v)
	}
	return arr, nil
}

func cborReadMap(r *creader, ai byte) (any, error) {
	type kv struct{ k, v any }
	var pairs []kv
	readPair := func() error {
		k, err := cborReadValue(r)
		if err != nil {
			return err
		}
		v, err := cborReadValue(r)
		if err != nil {
			return err
		}
		pairs = append(pairs, kv{k, v})
		return nil
	}
	if ai == 31 {
		for {
			b, err := r.peek()
			if err != nil {
				return nil, err
			}
			if b == 0xff {
				r.pos++
				break
			}
			if err := readPair(); err != nil {
				return nil, err
			}
		}
	} else {
		n, err := cborReadArg(r, ai)
		if err != nil {
			return nil, err
		}
		for range n {
			if err := readPair(); err != nil {
				return nil, err
			}
		}
	}
	// cbor builds a plain object only when every key is a (non-__proto__)
	// string; otherwise it builds a Map, which JSON.stringify renders as {}.
	obj := jsObject{}
	for _, p := range pairs {
		s, ok := p.k.(string)
		if !ok || s == "__proto__" {
			return jsObject{}, nil
		}
		if i := jsIndex(obj, s); i >= 0 {
			obj[i].v = p.v
		} else {
			obj = append(obj, jsPair{k: s, v: p.v})
		}
	}
	return obj, nil
}

func cborReadSimple(r *creader, ai byte) (any, error) {
	switch ai {
	case 20:
		return false, nil
	case 21:
		return true, nil
	case 22:
		return nil, nil
	case 23:
		return jsUndefined{}, nil
	case 25:
		b, err := r.take(2)
		if err != nil {
			return nil, err
		}
		return cborParseHalf(b), nil
	case 26:
		b, err := r.take(4)
		if err != nil {
			return nil, err
		}
		return float64(math.Float32frombits(binary.BigEndian.Uint32(b))), nil
	case 27:
		b, err := r.take(8)
		if err != nil {
			return nil, err
		}
		return math.Float64frombits(binary.BigEndian.Uint64(b)), nil
	default:
		return nil, fmt.Errorf("unsupported simple value %d", ai)
	}
}

// cborParseHalf ports cbor's utils.parseHalf: it decodes IEEE-754 half-precision
// bytes to a float64.
func cborParseHalf(b []byte) float64 {
	sign := 1.0
	if b[0]&0x80 != 0 {
		sign = -1
	}
	exp := (b[0] & 0x7c) >> 2
	mant := int(b[0]&0x03)<<8 | int(b[1])
	switch exp {
	case 0:
		return sign * 5.9604644775390625e-8 * float64(mant)
	case 0x1f:
		if mant != 0 {
			return math.NaN()
		}
		return sign * math.Inf(1)
	default:
		return sign * math.Pow(2, float64(exp)-25) * float64(1024+mant)
	}
}

// cborApplyTag mirrors cbor's default tag handlers for the tags that yield clean
// JSON: tag 0 is new Date(value) and tag 1 is new Date(value*1000), both
// rendered via toISOString(); 2/3 become (unserialisable) BigInts; any other tag
// is left as a {tag, value} object.
func cborApplyTag(tag uint64, inner any) any {
	switch tag {
	case 0:
		if s, ok := inner.(string); ok {
			return cborDateString(s)
		}
		if ms, ok := cborNumber(inner); ok {
			return cborMillisToISO(ms)
		}
		return nil
	case 1:
		if secs, ok := cborNumber(inner); ok {
			return cborMillisToISO(secs * 1000)
		}
		return nil
	case 2, 3:
		return cborBigInt{}
	default:
		return jsObject{{k: "tag", v: int64(tag)}, {k: "value", v: inner}} // #nosec G115 -- CBOR tag numbers are small in practice; mirrors the JS Number tag
	}
}

// cborNumber returns inner as a float64 if it is a decoded CBOR number.
func cborNumber(inner any) (float64, bool) {
	switch x := inner.(type) {
	case int64:
		return float64(x), true
	case float64:
		return x, true
	default:
		return 0, false
	}
}

// cborDateString renders an RFC 3339 datetime string the way
// new Date(s).toISOString() does; an unparseable string becomes null
// (JSON.stringify of an Invalid Date).
func cborDateString(s string) any {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC().Format("2006-01-02T15:04:05.000Z")
		}
	}
	return nil
}

// cborMillisToISO renders an epoch-milliseconds value as an ISO date string,
// matching new Date(ms).toISOString(); non-finite input becomes null.
func cborMillisToISO(ms float64) any {
	if math.IsNaN(ms) || math.IsInf(ms, 0) {
		return nil
	}
	return time.UnixMilli(int64(ms)).UTC().Format("2006-01-02T15:04:05.000Z")
}

// cborHasBigInt reports whether any BigInt is reachable by JSON.stringify in the
// decoded tree (non-string-keyed maps already dropped their contents).
func cborHasBigInt(v any) bool {
	switch x := v.(type) {
	case cborBigInt:
		return true
	case []any:
		if slices.ContainsFunc(x, cborHasBigInt) {
			return true
		}
	case jsObject:
		for _, p := range x {
			if cborHasBigInt(p.v) {
				return true
			}
		}
	}
	return false
}
