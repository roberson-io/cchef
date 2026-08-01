package ops

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
)

// Action Message Format, both versions, written against the Adobe
// specifications (AMF0 of 2006 and AMF3 of 2013).
//
// Values are the ones JSON carries — null, boolean, number, string, array,
// object — with objects held as jsObject so the order the input gave survives
// into the encoding. A Go map would lose it, and the order is part of the
// output: the same object has to encode to the same bytes every time.
//
// The reader checks the length before every read. A decoder that indexes into
// its buffer and trusts what it finds turns a truncated file into a panic.

// errAMFMalformed reports input that ran out or does not parse.
var errAMFMalformed = errors.New("malformed AMF data")

// AMF0 type markers.
const (
	amf0Number      = 0x00
	amf0Boolean     = 0x01
	amf0String      = 0x02
	amf0Object      = 0x03
	amf0Null        = 0x05
	amf0Undefined   = 0x06
	amf0Reference   = 0x07
	amf0ECMAArray   = 0x08
	amf0ObjectEnd   = 0x09
	amf0StrictArray = 0x0a
	amf0Date        = 0x0b
	amf0LongString  = 0x0c
	amf0Unsupported = 0x0d
	amf0XMLDoc      = 0x0f
	amf0TypedObject = 0x10
	amf0AVMPlus     = 0x11
)

// AMF3 type markers.
const (
	amf3Undefined  = 0x00
	amf3Null       = 0x01
	amf3False      = 0x02
	amf3True       = 0x03
	amf3Integer    = 0x04
	amf3Double     = 0x05
	amf3String     = 0x06
	amf3XMLDoc     = 0x07
	amf3Date       = 0x08
	amf3Array      = 0x09
	amf3Object     = 0x0a
	amf3XML        = 0x0b
	amf3ByteArray  = 0x0c
	amf3VectorInt  = 0x0d
	amf3VectorUint = 0x0e
	amf3VectorDbl  = 0x10
	amf3Dictionary = 0x11
)

// amf0MaxShortString is the longest string an AMF0 string marker can carry;
// anything longer takes the long-string marker instead.
const amf0MaxShortString = 0xffff

// amfMaxU29 is the largest value AMF3's variable-length integer can hold.
const amfMaxU29 = 0x1fffffff

// amfReader reads AMF from a byte slice, refusing to read past the end.
type amfReader struct {
	data []byte
	pos  int
}

// next returns the next n bytes, or an error if they are not all there.
func (r *amfReader) next(n int) ([]byte, error) {
	if n < 0 || n > len(r.data)-r.pos {
		return nil, errAMFMalformed
	}
	b := r.data[r.pos : r.pos+n]
	r.pos += n
	return b, nil
}

func (r *amfReader) byte() (byte, error) {
	b, err := r.next(1)
	if err != nil {
		return 0, err
	}
	return b[0], nil
}

func (r *amfReader) uint16() (int, error) {
	b, err := r.next(2)
	if err != nil {
		return 0, err
	}
	return int(binary.BigEndian.Uint16(b)), nil
}

func (r *amfReader) uint32() (int, error) {
	b, err := r.next(4)
	if err != nil {
		return 0, err
	}
	n := binary.BigEndian.Uint32(b)
	// The value indexes or sizes something, so it has to fit an int on every
	// platform before it is used as one. Whether it fits the data that remains
	// is the caller's business: a length is bounded by next, and a count is
	// bounded by the elements actually being there to read.
	if n > math.MaxInt32 {
		return 0, errAMFMalformed
	}
	return int(n), nil
}

func (r *amfReader) double() (float64, error) {
	b, err := r.next(8)
	if err != nil {
		return 0, err
	}
	return math.Float64frombits(binary.BigEndian.Uint64(b)), nil
}

// u29 reads AMF3's variable-length unsigned integer. The first three bytes
// carry seven bits each and set the high bit to say another follows; a fourth
// carries all eight.
func (r *amfReader) u29() (int, error) {
	var n int
	for range 3 {
		b, err := r.byte()
		if err != nil {
			return 0, err
		}
		n = n<<7 | int(b&0x7f)
		if b&0x80 == 0 {
			return n, nil
		}
	}
	b, err := r.byte()
	if err != nil {
		return 0, err
	}
	return n<<8 | int(b), nil
}

// amfWriter builds AMF output. A value AMF3's 29-bit length cannot express is
// recorded here rather than returned from every call: the limit belongs to the
// writer, and the caller checks once when the value is complete.
type amfWriter struct {
	buf []byte
	err error
}

func (w *amfWriter) byte(b byte)    { w.buf = append(w.buf, b) }
func (w *amfWriter) bytes(b []byte) { w.buf = append(w.buf, b...) }
func (w *amfWriter) uint16(n int)   { w.buf = binary.BigEndian.AppendUint16(w.buf, uint16(n)) } // #nosec G115 -- callers bound n to amf0MaxShortString
func (w *amfWriter) uint32(n int)   { w.buf = binary.BigEndian.AppendUint32(w.buf, uint32(n)) } // #nosec G115 -- callers bound n to a slice length
func (w *amfWriter) double(f float64) {
	w.buf = binary.BigEndian.AppendUint64(w.buf, math.Float64bits(f))
}

// u29 writes AMF3's variable-length unsigned integer. Every byte is masked to
// the seven or eight bits it carries, so the conversions cannot lose anything
// the format would have kept. A value past what 29 bits hold is refused: it
// would otherwise be written truncated and read back as a different number.
func (w *amfWriter) u29(n int) {
	if w.err != nil {
		return
	}
	if n < 0 || n > amfMaxU29 {
		w.err = fmt.Errorf("value %d is too large for AMF3's 29-bit length", n)
		return
	}
	switch {
	case n < 0x80:
		w.byte(u29Byte(n))
	case n < 0x4000:
		w.byte(u29Byte(n>>7 | 0x80))
		w.byte(u29Byte(n & 0x7f))
	case n < 0x200000:
		w.byte(u29Byte(n>>14 | 0x80))
		w.byte(u29Byte(n>>7&0x7f | 0x80))
		w.byte(u29Byte(n & 0x7f))
	default:
		w.byte(u29Byte(n>>22 | 0x80))
		w.byte(u29Byte(n>>15&0x7f | 0x80))
		w.byte(u29Byte(n>>8&0x7f | 0x80))
		w.byte(u29Byte(n))
	}
}

// u29Byte narrows one piece of a u29 to the byte it already is. Every caller
// has masked to eight bits or fewer, and u29 has bounded the whole value to
// amfMaxU29, so nothing is lost here.
func u29Byte(n int) byte { return byte(n) } // #nosec G115 -- callers mask to eight bits or fewer

// amfParseJSON reads JSON into the value kinds the encoders handle, keeping
// object keys in the order they were written.
func amfParseJSON(data []byte) (any, error) {
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.UseNumber()
	v, err := amfParseValue(dec)
	if err != nil {
		return nil, err
	}
	// Anything after the first value means the input was not one JSON value.
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, errors.New("trailing data after the JSON value")
	}
	return v, nil
}

// amfParseValue reads one value from an open decoder.
func amfParseValue(dec *json.Decoder) (any, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	return amfParseToken(dec, tok)
}

// amfParseToken turns a token into a value, reading the rest of an array or
// object from the decoder when the token opens one.
func amfParseToken(dec *json.Decoder, tok json.Token) (any, error) {
	switch t := tok.(type) {
	case json.Delim:
		if t == '[' {
			return amfParseArray(dec)
		}
		// The only other delimiter that opens a value is '{'; the closing two
		// are consumed by the readers that opened them.
		return amfParseObject(dec)
	case json.Number:
		f, err := t.Float64()
		if err != nil {
			return nil, err
		}
		return f, nil
	default: // string, bool, nil
		return tok, nil
	}
}

func amfParseArray(dec *json.Decoder) (any, error) {
	arr := []any{}
	for dec.More() {
		v, err := amfParseValue(dec)
		if err != nil {
			return nil, err
		}
		arr = append(arr, v)
	}
	_, err := dec.Token() // ']'
	return arr, err
}

func amfParseObject(dec *json.Decoder) (any, error) {
	obj := jsObject{}
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		// Token only ever yields a string here: a key of any other kind is a
		// syntax error the decoder has already refused.
		key, _ := keyTok.(string)
		v, err := amfParseValue(dec)
		if err != nil {
			return nil, err
		}
		obj = append(obj, jsPair{k: key, v: v})
	}
	_, err := dec.Token() // '}'
	return obj, err
}

// amf0Encode writes one value in AMF0.
func amf0Encode(w *amfWriter, v any) error {
	switch x := v.(type) {
	case nil:
		w.byte(amf0Null)
	case bool:
		w.byte(amf0Boolean)
		if x {
			w.byte(1)
		} else {
			w.byte(0)
		}
	case float64:
		w.byte(amf0Number)
		w.double(x)
	case string:
		amf0EncodeString(w, x, true)
	case []any:
		w.byte(amf0StrictArray)
		w.uint32(len(x))
		for _, e := range x {
			if err := amf0Encode(w, e); err != nil {
				return err
			}
		}
	case jsObject:
		w.byte(amf0Object)
		for _, p := range x {
			if len(p.k) > amf0MaxShortString {
				return fmt.Errorf("object key is longer than %d bytes", amf0MaxShortString)
			}
			w.uint16(len(p.k))
			w.bytes([]byte(p.k))
			if err := amf0Encode(w, p.v); err != nil {
				return err
			}
		}
		// An empty key followed by the end marker closes the object.
		w.uint16(0)
		w.byte(amf0ObjectEnd)
	default:
		return fmt.Errorf("cannot encode %T in AMF0", v)
	}
	return nil
}

// amf0EncodeString writes a string, choosing the long form when it does not
// fit the 16-bit length. withMarker is false for the class name of a typed
// object and for object keys, which carry a bare length.
func amf0EncodeString(w *amfWriter, s string, withMarker bool) {
	if len(s) > amf0MaxShortString {
		if withMarker {
			w.byte(amf0LongString)
		}
		w.uint32(len(s))
		w.bytes([]byte(s))
		return
	}
	if withMarker {
		w.byte(amf0String)
	}
	w.uint16(len(s))
	w.bytes([]byte(s))
}

// amf0Decode reads one AMF0 value. refs holds the objects seen so far, which
// the reference marker points back into.
func amf0Decode(r *amfReader, refs *[]any) (any, error) {
	marker, err := r.byte()
	if err != nil {
		return nil, err
	}
	if v, handled, err := amf0DecodeScalar(r, marker); handled {
		return v, err
	}
	return amf0DecodeComposite(r, refs, marker)
}

// amf0DecodeScalar reads the markers that carry a single value. handled is
// false for the markers that need the reference table, which the caller takes.
func amf0DecodeScalar(r *amfReader, marker byte) (v any, handled bool, err error) {
	switch marker {
	case amf0Number:
		d, err := r.double()
		return d, true, err
	case amf0Boolean:
		b, err := r.byte()
		return b != 0, true, err
	case amf0String:
		str, err := amf0ReadString(r, false)
		return str, true, err
	case amf0LongString, amf0XMLDoc:
		str, err := amf0ReadString(r, true)
		return str, true, err
	case amf0Null, amf0Undefined, amf0Unsupported:
		return nil, true, nil
	case amf0Date:
		millis, err := r.double()
		if err != nil {
			return nil, true, err
		}
		// The time zone is specified as unused, but occupies two bytes.
		_, err = r.next(2)
		return millis, true, err
	}
	return nil, false, nil
}

// amf0DecodeComposite reads the markers that build or refer to a container.
func amf0DecodeComposite(r *amfReader, refs *[]any, marker byte) (any, error) {
	switch marker {
	case amf0Object:
		return amf0ReadObject(r, refs)
	case amf0TypedObject:
		// The class name is carried but the value it describes is the object.
		if _, err := amf0ReadString(r, false); err != nil {
			return nil, err
		}
		return amf0ReadObject(r, refs)
	case amf0ECMAArray:
		// The count is advisory; the object-end marker terminates it.
		if _, err := r.uint32(); err != nil {
			return nil, err
		}
		return amf0ReadObject(r, refs)
	case amf0StrictArray:
		return amf0ReadStrictArray(r, refs)
	case amf0Reference:
		i, err := r.uint16()
		if err != nil {
			return nil, err
		}
		if i >= len(*refs) {
			return nil, errAMFMalformed
		}
		return (*refs)[i], nil
	case amf0AVMPlus:
		return amf3Decode(r, newAMF3Tables())
	default:
		return nil, fmt.Errorf("unknown AMF0 marker 0x%02x", marker)
	}
}

// amf0ReadStrictArray reads a counted array of values.
func amf0ReadStrictArray(r *amfReader, refs *[]any) (any, error) {
	n, err := r.uint32()
	if err != nil {
		return nil, err
	}
	arr := make([]any, 0, min(n, len(r.data)-r.pos))
	for range n {
		e, err := amf0Decode(r, refs)
		if err != nil {
			return nil, err
		}
		arr = append(arr, e)
	}
	*refs = append(*refs, arr)
	return arr, nil
}

// amf0ReadString reads a length-prefixed string.
func amf0ReadString(r *amfReader, long bool) (string, error) {
	var n int
	var err error
	if long {
		n, err = r.uint32()
	} else {
		n, err = r.uint16()
	}
	if err != nil {
		return "", err
	}
	b, err := r.next(n)
	return string(b), err
}

// amf0ReadObject reads key/value pairs until the empty key and end marker.
func amf0ReadObject(r *amfReader, refs *[]any) (any, error) {
	obj := jsObject{}
	*refs = append(*refs, obj)
	at := len(*refs) - 1
	for {
		key, err := amf0ReadString(r, false)
		if err != nil {
			return nil, err
		}
		if key == "" {
			// An empty key is the terminator only when the end marker follows;
			// otherwise it is a genuine key with an empty name.
			save := r.pos
			b, err := r.byte()
			if err != nil {
				return nil, err
			}
			if b == amf0ObjectEnd {
				(*refs)[at] = obj
				return obj, nil
			}
			r.pos = save
		}
		v, err := amf0Decode(r, refs)
		if err != nil {
			return nil, err
		}
		obj = append(obj, jsPair{k: key, v: v})
		(*refs)[at] = obj
	}
}

// amf3Tables holds the three back-reference tables an AMF3 stream indexes
// into. They are per-message, so a fresh set starts every decode.
type amf3Tables struct {
	strings []string
	objects []any
	traits  [][]string
}

func newAMF3Tables() *amf3Tables { return &amf3Tables{} }

// amf3Encode writes one value in AMF3. Nothing is written by reference: the
// output is a little larger than it need be, and reads identically.
func amf3Encode(w *amfWriter, v any) error {
	switch x := v.(type) {
	case nil:
		w.byte(amf3Null)
	case bool:
		if x {
			w.byte(amf3True)
		} else {
			w.byte(amf3False)
		}
	case float64:
		// Every number goes out as a double, including whole ones, which is
		// what CyberChef does and keeps the round trip exact.
		w.byte(amf3Double)
		w.double(x)
	case string:
		w.byte(amf3String)
		amf3EncodeString(w, x)
	case []any:
		w.byte(amf3Array)
		w.u29(len(x)<<1 | 1)
		w.byte(0x01) // no associative entries
		for _, e := range x {
			if err := amf3Encode(w, e); err != nil {
				return err
			}
		}
	case jsObject:
		return amf3EncodeObject(w, x)
	default:
		return fmt.Errorf("cannot encode %T in AMF3", v)
	}
	return nil
}

// amf3EncodeString writes a string body: the length shifted left with the low
// bit set to mark it inline, then the UTF-8 bytes.
func amf3EncodeString(w *amfWriter, s string) {
	w.u29(len(s)<<1 | 1)
	w.bytes([]byte(s))
}

// amf3EncodeObject writes an object as sealed traits: the keys become the
// member names and the values follow in the same order.
func amf3EncodeObject(w *amfWriter, obj jsObject) error {
	w.byte(amf3Object)
	// Inline object, inline traits, not externalizable, not dynamic, with one
	// sealed member per key.
	w.u29(len(obj)<<4 | 0x03)
	w.byte(0x01) // anonymous class
	for _, p := range obj {
		amf3EncodeString(w, p.k)
	}
	for _, p := range obj {
		if err := amf3Encode(w, p.v); err != nil {
			return err
		}
	}
	return nil
}

// amf3Decode reads one AMF3 value.
func amf3Decode(r *amfReader, t *amf3Tables) (any, error) {
	marker, err := r.byte()
	if err != nil {
		return nil, err
	}
	if v, handled, err := amf3DecodeScalar(r, t, marker); handled {
		return v, err
	}
	switch marker {
	case amf3Date:
		return amf3ReadDate(r, t)
	case amf3ByteArray:
		return amf3ReadByteArray(r, t)
	case amf3Array:
		return amf3ReadArray(r, t)
	case amf3Object:
		return amf3ReadObject(r, t)
	default:
		return nil, fmt.Errorf("unknown AMF3 marker 0x%02x", marker)
	}
}

// amf3DecodeScalar reads the markers that carry a single value.
func amf3DecodeScalar(r *amfReader, t *amf3Tables, marker byte) (v any, handled bool, err error) {
	switch marker {
	case amf3Undefined, amf3Null:
		return nil, true, nil
	case amf3False:
		return false, true, nil
	case amf3True:
		return true, true, nil
	case amf3Integer:
		n, err := r.u29()
		if err != nil {
			return nil, true, err
		}
		// The 29-bit integer is signed; the top bit is the sign.
		if n > amfMaxU29>>1 {
			n -= amfMaxU29 + 1
		}
		return float64(n), true, nil
	case amf3Double:
		d, err := r.double()
		return d, true, err
	case amf3String, amf3XMLDoc, amf3XML:
		str, err := amf3ReadString(r, t)
		return str, true, err
	}
	return nil, false, nil
}

// amf3ReadDate reads a date, which is milliseconds since the epoch or a
// reference to one already read.
func amf3ReadDate(r *amfReader, t *amf3Tables) (any, error) {
	ref, err := r.u29()
	if err != nil {
		return nil, err
	}
	if ref&1 == 0 {
		return amf3Ref(t, ref>>1)
	}
	d, err := r.double()
	if err != nil {
		return nil, err
	}
	t.objects = append(t.objects, d)
	return d, nil
}

// amf3ReadByteArray reads raw bytes, returned as a string so they survive into
// JSON the same way a string does.
func amf3ReadByteArray(r *amfReader, t *amf3Tables) (any, error) {
	ref, err := r.u29()
	if err != nil {
		return nil, err
	}
	if ref&1 == 0 {
		return amf3Ref(t, ref>>1)
	}
	b, err := r.next(ref >> 1)
	if err != nil {
		return nil, err
	}
	s := string(b)
	t.objects = append(t.objects, s)
	return s, nil
}

// amf3Ref returns the object at index i, or an error if the stream points
// somewhere that was never written.
func amf3Ref(t *amf3Tables, i int) (any, error) {
	if i < 0 || i >= len(t.objects) {
		return nil, errAMFMalformed
	}
	return t.objects[i], nil
}

// amf3ReadString reads a string, which is either inline or an index into the
// string table. The empty string is always inline and never recorded.
func amf3ReadString(r *amfReader, t *amf3Tables) (string, error) {
	n, err := r.u29()
	if err != nil {
		return "", err
	}
	if n&1 == 0 {
		i := n >> 1
		if i < 0 || i >= len(t.strings) {
			return "", errAMFMalformed
		}
		return t.strings[i], nil
	}
	b, err := r.next(n >> 1)
	if err != nil {
		return "", err
	}
	s := string(b)
	if s != "" {
		t.strings = append(t.strings, s)
	}
	return s, nil
}

// amf3ReadArray reads the associative entries, then the dense ones. An
// associative entry makes the whole thing an object, since JSON arrays have no
// keys.
func amf3ReadArray(r *amfReader, t *amf3Tables) (any, error) {
	n, err := r.u29()
	if err != nil {
		return nil, err
	}
	if n&1 == 0 {
		return amf3Ref(t, n>>1)
	}
	count := n >> 1
	at := len(t.objects)
	t.objects = append(t.objects, nil)

	assoc := jsObject{}
	for {
		key, err := amf3ReadString(r, t)
		if err != nil {
			return nil, err
		}
		if key == "" {
			break
		}
		v, err := amf3Decode(r, t)
		if err != nil {
			return nil, err
		}
		assoc = append(assoc, jsPair{k: key, v: v})
	}

	if count > len(r.data)-r.pos {
		return nil, errAMFMalformed
	}
	dense := make([]any, 0, count)
	for range count {
		e, err := amf3Decode(r, t)
		if err != nil {
			return nil, err
		}
		dense = append(dense, e)
	}
	if len(assoc) > 0 {
		// Mixed arrays cannot be a JSON array, so the dense part joins the
		// associative part under its index.
		for i, e := range dense {
			assoc = append(assoc, jsPair{k: fmt.Sprint(i), v: e})
		}
		t.objects[at] = assoc
		return assoc, nil
	}
	t.objects[at] = dense
	return dense, nil
}

// amf3ReadObject reads an object: traits (inline or referenced), the sealed
// member values in order, then any dynamic members.
func amf3ReadObject(r *amfReader, t *amf3Tables) (any, error) {
	n, err := r.u29()
	if err != nil {
		return nil, err
	}
	if n&1 == 0 {
		return amf3Ref(t, n>>1)
	}
	at := len(t.objects)
	t.objects = append(t.objects, nil)

	members, dynamic, err := amf3ReadTraits(r, t, n)
	if err != nil {
		return nil, err
	}

	obj := jsObject{}
	for _, m := range members {
		v, err := amf3Decode(r, t)
		if err != nil {
			return nil, err
		}
		obj = append(obj, jsPair{k: m, v: v})
	}
	if dynamic {
		for {
			key, err := amf3ReadString(r, t)
			if err != nil {
				return nil, err
			}
			if key == "" {
				break
			}
			v, err := amf3Decode(r, t)
			if err != nil {
				return nil, err
			}
			obj = append(obj, jsPair{k: key, v: v})
		}
	}
	t.objects[at] = obj
	return obj, nil
}

// amf3ReadTraits reads an object's member list, which is either written out
// here or referenced from an earlier object of the same class. It reports the
// member names and whether further members follow dynamically.
func amf3ReadTraits(r *amfReader, t *amf3Tables, n int) (members []string, dynamic bool, err error) {
	if n&2 == 0 { // traits by reference
		i := n >> 2
		if i < 0 || i >= len(t.traits) {
			return nil, false, errAMFMalformed
		}
		return t.traits[i], false, nil
	}
	if n&4 != 0 {
		// Externalizable: the class decides its own layout, which cannot be
		// read without knowing the class.
		return nil, false, errors.New("AMF3 externalizable objects are not supported")
	}
	count := n >> 4
	if _, err := amf3ReadString(r, t); err != nil { // class name
		return nil, false, err
	}
	if count > len(r.data)-r.pos {
		return nil, false, errAMFMalformed
	}
	members = make([]string, 0, count)
	for range count {
		m, err := amf3ReadString(r, t)
		if err != nil {
			return nil, false, err
		}
		members = append(members, m)
	}
	t.traits = append(t.traits, members)
	return members, n&8 != 0, nil
}
