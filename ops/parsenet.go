package ops

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
)

// toHexFast renders bytes as lowercase hex with no delimiter (CyberChef toHexFast).
func toHexFast(b []byte) string { return hex.EncodeToString(b) }

// omap is an insertion-ordered JSON object. Go's encoding/json sorts map keys,
// so the packet parsers build omaps to preserve CyberChef's field order.
type omap struct {
	keys []string
	vals map[string]any
}

func newOMap() *omap { return &omap{vals: map[string]any{}} }

func (o *omap) set(key string, v any) *omap {
	if _, ok := o.vals[key]; !ok {
		o.keys = append(o.keys, key)
	}
	o.vals[key] = v
	return o
}

// jsonNoEscape marshals v to compact JSON without escaping <, > and &, matching
// JavaScript's JSON.stringify.
func jsonNoEscape(v any) ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(b.Bytes(), "\n"), nil
}

// MarshalJSON emits the object with keys in insertion order.
func (o *omap) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte('{')
	for i, k := range o.keys {
		if i > 0 {
			b.WriteByte(',')
		}
		kb, err := jsonNoEscape(k)
		if err != nil {
			return nil, err
		}
		b.Write(kb)
		b.WriteByte(':')
		vb, err := jsonNoEscape(o.vals[k])
		if err != nil {
			return nil, err
		}
		b.Write(vb)
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}

// marshalOMap renders an omap as compact JSON text (CyberChef's minified form).
// The packet parsers only store ints, strings and nested omaps, all JSON-safe,
// so their `marshalOMap` error propagations are unreachable in practice; the
// encoder's actual failure mode is exercised directly in parsenet_test.go.
func marshalOMap(o *omap) ([]byte, error) {
	return jsonNoEscape(o)
}
