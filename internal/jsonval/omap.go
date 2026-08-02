package jsonval

import (
	"bytes"
	"encoding/json"
)

// OMap is an insertion-ordered JSON object. Go's encoding/json sorts map keys,
// so the packet parsers build omaps to preserve CyberChef's field order.
type OMap struct {
	keys []string
	vals map[string]any
}

// NewOMap returns an empty ordered map.
func NewOMap() *OMap { return &OMap{vals: map[string]any{}} }

// Set stores v under key and returns o for chaining. A new key goes to the
// end; an existing one keeps its position.
func (o *OMap) Set(key string, v any) *OMap {
	if _, ok := o.vals[key]; !ok {
		o.keys = append(o.keys, key)
	}
	o.vals[key] = v
	return o
}

// MarshalNoEscape marshals v to compact JSON without escaping <, > and &, matching
// JavaScript's JSON.stringify.
func MarshalNoEscape(v any) ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(b.Bytes(), "\n"), nil
}

// Get returns the value stored under key, and whether it is present.
func (o *OMap) Get(key string) (any, bool) {
	v, ok := o.vals[key]
	return v, ok
}

// Value returns the value stored under key, or nil when it is absent, the way
// a Go map read returns the zero value.
func (o *OMap) Value(key string) any { return o.vals[key] }

// Keys returns the keys in insertion order. The slice is the map's own; callers
// only range over it.
func (o *OMap) Keys() []string { return o.keys }

// Merge appends all of src's entries into o, in src's order. A key both maps
// hold keeps its position in o and takes src's value.
func (o *OMap) Merge(src *OMap) {
	for _, k := range src.keys {
		o.Set(k, src.vals[k])
	}
}

// MarshalJSON emits the object with keys in insertion order.
func (o *OMap) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte('{')
	for i, k := range o.keys {
		if i > 0 {
			b.WriteByte(',')
		}
		kb, err := MarshalNoEscape(k)
		if err != nil {
			return nil, err
		}
		b.Write(kb)
		b.WriteByte(':')
		vb, err := MarshalNoEscape(o.vals[k])
		if err != nil {
			return nil, err
		}
		b.Write(vb)
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}

// MarshalOMap renders an omap as compact JSON text (CyberChef's minified form).
// The packet parsers only store ints, strings and nested omaps, all JSON-safe,
// so their `MarshalOMap` error propagations are unreachable in practice; the
// encoder's actual failure mode is exercised directly in parsenet_test.go.
func MarshalOMap(o *OMap) ([]byte, error) {
	return MarshalNoEscape(o)
}
