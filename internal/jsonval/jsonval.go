// Package jsonval holds an insertion-ordered JSON value model.
//
// Go's encoding/json stores an object in a map, which loses the order its keys
// arrived in. Several operations have to preserve that order — a formatter must
// not reshuffle the document it was given, and AMF encodes keys in the order
// they appear — so [Object] keeps its pairs in a slice instead.
//
// [Stringify] writes a value back out the way JavaScript's JSON.stringify does,
// including its rule that integer-like keys come first in ascending order and
// everything else follows in insertion order. [ESOrder] applies that rule on
// its own.
package jsonval

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
)

// Pair is one ordered key/value entry of a Object.
type Pair struct {
	K string
	V any
}

// Object is an insertion-ordered JSON object, used where key order must be
// preserved (Avro records/maps, CBOR maps) rather than sorted as Go's map
// marshalling would do.
type Object []Pair

// Undefined represents a JavaScript `undefined`. JSON.stringify omits it from
// objects and renders it as null inside arrays; at the top level the whole
// output is empty. Only CBOR decoding produces it.
type Undefined struct{}

// jsArrayIndex reports whether k is a canonical ECMAScript array index (a
// non-negative integer string with no leading zeros, value < 2^32-1) and, if
// so, its numeric value.
func jsArrayIndex(k string) (uint64, bool) {
	if k == "0" {
		return 0, true
	}
	if k == "" || k[0] < '1' || k[0] > '9' {
		return 0, false
	}
	for i := 0; i < len(k); i++ {
		if k[i] < '0' || k[i] > '9' {
			return 0, false
		}
	}
	n, err := strconv.ParseUint(k, 10, 64)
	if err != nil || n >= 4294967295 {
		return 0, false
	}
	return n, true
}

// ESOrder reorders an object's entries the way JavaScript enumerates own
// string keys (OrdinaryOwnPropertyKeys): integer-index keys first in ascending
// numeric order, then the remaining keys in insertion order. JSON.stringify and
// Object.keys both use this order.
func ESOrder(obj Object) Object {
	type indexed struct {
		n   uint64
		pos int
	}
	var ints []indexed
	var rest []int
	for i, p := range obj {
		if n, ok := jsArrayIndex(p.K); ok {
			ints = append(ints, indexed{n, i})
		} else {
			rest = append(rest, i)
		}
	}
	if len(ints) == 0 {
		return obj
	}
	sort.SliceStable(ints, func(a, b int) bool { return ints[a].n < ints[b].n })
	out := make(Object, 0, len(obj))
	for _, it := range ints {
		out = append(out, obj[it.pos])
	}
	for _, i := range rest {
		out = append(out, obj[i])
	}
	return out
}

// Index returns the position of key k in obj, or -1.
func Index(obj Object, k string) int {
	for i, p := range obj {
		if p.K == k {
			return i
		}
	}
	return -1
}

// ParseOrdered parses JSON preserving object key order (as Object), which
// Go's map-based json.Unmarshal cannot. Numbers become float64 (matching a
// JavaScript JSON.parse), objects Object, arrays []any, and duplicate object
// keys keep their first position with the last value (JS object semantics). It
// rejects trailing data after the single top-level value.
func ParseOrdered(data []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	v, err := jsonParseValue(dec)
	if err != nil {
		return nil, err
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, errors.New("trailing data after JSON value")
	}
	return v, nil
}

func jsonParseValue(dec *json.Decoder) (any, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	delim, ok := tok.(json.Delim)
	if !ok {
		if n, ok := tok.(json.Number); ok {
			f, err := n.Float64()
			return f, err
		}
		return tok, nil // string, bool, or nil
	}
	switch delim {
	case '{':
		obj := Object{}
		for dec.More() {
			keyTok, err := dec.Token()
			if err != nil {
				return nil, err
			}
			key := keyTok.(string)
			val, err := jsonParseValue(dec)
			if err != nil {
				return nil, err
			}
			if i := Index(obj, key); i >= 0 {
				obj[i].V = val
			} else {
				obj = append(obj, Pair{K: key, V: val})
			}
		}
		if _, err := dec.Token(); err != nil { // consume '}'
			return nil, err
		}
		return obj, nil
	default: // '['
		arr := []any{}
		for dec.More() {
			val, err := jsonParseValue(dec)
			if err != nil {
				return nil, err
			}
			arr = append(arr, val)
		}
		if _, err := dec.Token(); err != nil { // consume ']'
			return nil, err
		}
		return arr, nil
	}
}

// Buffer renders a byte slice the way JSON.stringify renders a Node Buffer:
// {"type":"Buffer","data":[...]}.
func Buffer(b []byte) Object {
	data := make([]any, len(b))
	for i, c := range b {
		data[i] = int64(c)
	}
	return Object{{K: "type", V: "Buffer"}, {K: "data", V: data}}
}

// Stringify reproduces JavaScript's JSON.stringify(value, null, indent):
// indent 0 is compact, indent > 0 pretty-prints with that many spaces. It
// understands nil, bool, int64, float64, string, []any, Object and
// Undefined values.
// Stringify serialises v like JSON.stringify(value, null, indent) using indent
// spaces (0 = compact).
func Stringify(v any, indent int) string {
	return StringifyIndent(v, strings.Repeat(" ", indent))
}

// StringifyIndent serialises v like JSON.stringify(value, null, unit), where
// unit is the literal indentation string (a tab, N spaces, or any string; ""
// produces compact output). This backs JSON Beautify's binaryShortString indent.
func StringifyIndent(v any, unit string) string {
	var sb strings.Builder
	jsWrite(&sb, v, unit, "")
	return sb.String()
}

func jsWrite(sb *strings.Builder, v any, unit, cur string) {
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
		sb.WriteString(FormatNumber(x))
	case string:
		sb.WriteString(jsJSONString(x))
	case []any:
		jsWriteArray(sb, x, unit, cur)
	case Object:
		jsWriteObject(sb, x, unit, cur)
	case Undefined:
		// Only reached for an array element; JSON.stringify renders it as null.
		sb.WriteString("null")
	}
}

func jsWriteArray(sb *strings.Builder, arr []any, unit, cur string) {
	if len(arr) == 0 {
		sb.WriteString("[]")
		return
	}
	if unit == "" {
		sb.WriteByte('[')
		for i, e := range arr {
			if i > 0 {
				sb.WriteByte(',')
			}
			jsWrite(sb, e, "", "")
		}
		sb.WriteByte(']')
		return
	}
	inner := cur + unit
	sb.WriteString("[\n")
	for i, e := range arr {
		sb.WriteString(inner)
		jsWrite(sb, e, unit, inner)
		if i < len(arr)-1 {
			sb.WriteByte(',')
		}
		sb.WriteByte('\n')
	}
	sb.WriteString(cur)
	sb.WriteByte(']')
}

func jsWriteObject(sb *strings.Builder, obj Object, unit, cur string) {
	// JSON.stringify enumerates in ECMAScript key order and omits properties
	// whose value is undefined.
	pairs := make(Object, 0, len(obj))
	for _, p := range ESOrder(obj) {
		if _, ok := p.V.(Undefined); ok {
			continue
		}
		pairs = append(pairs, p)
	}
	if len(pairs) == 0 {
		sb.WriteString("{}")
		return
	}
	if unit == "" {
		sb.WriteByte('{')
		for i, p := range pairs {
			if i > 0 {
				sb.WriteByte(',')
			}
			sb.WriteString(jsJSONString(p.K))
			sb.WriteByte(':')
			jsWrite(sb, p.V, "", "")
		}
		sb.WriteByte('}')
		return
	}
	inner := cur + unit
	sb.WriteString("{\n")
	for i, p := range pairs {
		sb.WriteString(inner)
		sb.WriteString(jsJSONString(p.K))
		sb.WriteString(": ")
		jsWrite(sb, p.V, unit, inner)
		if i < len(pairs)-1 {
			sb.WriteByte(',')
		}
		sb.WriteByte('\n')
	}
	sb.WriteString(cur)
	sb.WriteByte('}')
}

// FormatNumber matches JSON.stringify's number output: NaN/Infinity become
// null, negative zero becomes "0", and everything else uses Go's ECMAScript-
// compatible float formatting.
func FormatNumber(f float64) string {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return "null"
	}
	if f == 0 {
		return "0"
	}
	b, _ := json.Marshal(f)
	return string(b)
}

// jsJSONString escapes a string the way JSON.stringify does (no HTML escaping of
// <, >, &).
func jsJSONString(s string) string {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s)
	return strings.TrimRight(b.String(), "\n")
}
