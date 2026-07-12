package ops

import (
	"bytes"
	"encoding/json"
	"math"
	"strconv"
	"strings"
)

// jsPair is one ordered key/value entry of a jsObject.
type jsPair struct {
	k string
	v any
}

// jsObject is an insertion-ordered JSON object, used where key order must be
// preserved (Avro records/maps, CBOR maps) rather than sorted as Go's map
// marshalling would do.
type jsObject []jsPair

// jsUndefined represents a JavaScript `undefined`. JSON.stringify omits it from
// objects and renders it as null inside arrays; at the top level the whole
// output is empty. Only CBOR decoding produces it.
type jsUndefined struct{}

// jsBuffer renders a byte slice the way JSON.stringify renders a Node Buffer:
// {"type":"Buffer","data":[...]}.
func jsBuffer(b []byte) jsObject {
	data := make([]any, len(b))
	for i, c := range b {
		data[i] = int64(c)
	}
	return jsObject{{k: "type", v: "Buffer"}, {k: "data", v: data}}
}

// jsStringify reproduces JavaScript's JSON.stringify(value, null, indent):
// indent 0 is compact, indent > 0 pretty-prints with that many spaces. It
// understands nil, bool, int64, float64, string, []any, jsObject and
// jsUndefined values.
func jsStringify(v any, indent int) string {
	var sb strings.Builder
	jsWrite(&sb, v, indent, "")
	return sb.String()
}

func jsWrite(sb *strings.Builder, v any, indent int, cur string) {
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
		sb.WriteString(jsFormatNumber(x))
	case string:
		sb.WriteString(jsJSONString(x))
	case []any:
		jsWriteArray(sb, x, indent, cur)
	case jsObject:
		jsWriteObject(sb, x, indent, cur)
	case jsUndefined:
		// Only reached for an array element; JSON.stringify renders it as null.
		sb.WriteString("null")
	}
}

func jsWriteArray(sb *strings.Builder, arr []any, indent int, cur string) {
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
			jsWrite(sb, e, 0, "")
		}
		sb.WriteByte(']')
		return
	}
	inner := cur + strings.Repeat(" ", indent)
	sb.WriteString("[\n")
	for i, e := range arr {
		sb.WriteString(inner)
		jsWrite(sb, e, indent, inner)
		if i < len(arr)-1 {
			sb.WriteByte(',')
		}
		sb.WriteByte('\n')
	}
	sb.WriteString(cur)
	sb.WriteByte(']')
}

func jsWriteObject(sb *strings.Builder, obj jsObject, indent int, cur string) {
	// JSON.stringify omits properties whose value is undefined.
	pairs := make(jsObject, 0, len(obj))
	for _, p := range obj {
		if _, ok := p.v.(jsUndefined); ok {
			continue
		}
		pairs = append(pairs, p)
	}
	if len(pairs) == 0 {
		sb.WriteString("{}")
		return
	}
	if indent == 0 {
		sb.WriteByte('{')
		for i, p := range pairs {
			if i > 0 {
				sb.WriteByte(',')
			}
			sb.WriteString(jsJSONString(p.k))
			sb.WriteByte(':')
			jsWrite(sb, p.v, 0, "")
		}
		sb.WriteByte('}')
		return
	}
	inner := cur + strings.Repeat(" ", indent)
	sb.WriteString("{\n")
	for i, p := range pairs {
		sb.WriteString(inner)
		sb.WriteString(jsJSONString(p.k))
		sb.WriteString(": ")
		jsWrite(sb, p.v, indent, inner)
		if i < len(pairs)-1 {
			sb.WriteByte(',')
		}
		sb.WriteByte('\n')
	}
	sb.WriteString(cur)
	sb.WriteByte('}')
}

// jsFormatNumber matches JSON.stringify's number output: NaN/Infinity become
// null, negative zero becomes "0", and everything else uses Go's ECMAScript-
// compatible float formatting.
func jsFormatNumber(f float64) string {
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
