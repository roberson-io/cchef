package ops

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/itchyny/gojq"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(Jq{})
}

// Jq processes the JSON input with a jq query. Ported from CyberChef Jq.mjs, which
// wraps jq-web (jq compiled to WASM); cchef reimplements it over gojq, a pure-Go
// jq. It reproduces jq-web's jq.json() collapse of jq's output stream: zero
// results is an error, a single result is returned directly, and multiple results
// become a JSON array. The value is then printed raw (when Raw is set and the
// result is a string) or serialized like JSON.stringify.
type Jq struct{}

// Meta returns the operation metadata.
func (Jq) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Jq",
		Module:      "Jq",
		Description: "jq is a lightweight and flexible command-line JSON processor.",
		InfoURL:     "https://github.com/jqlang/jq",
		InputType:   core.TypeJSON,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (Jq) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Query", Type: core.ArgString, Value: ""},
		{Name: "Raw", Type: core.ArgBoolean, Value: false},
	}
}

// Run evaluates the jq query against the JSON input.
func (Jq) Run(in *core.Dish, args []any) (*core.Dish, error) {
	query := args[0].(string)
	raw := args[1].(bool)

	var data any
	if err := json.Unmarshal(in.Bytes(), &data); err != nil {
		return nil, fmt.Errorf("invalid JSON input: %w", err)
	}
	q, err := gojq.Parse(query)
	if err != nil {
		return nil, jqError(err)
	}

	var outs []any
	iter := q.Run(data)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if e, ok := v.(error); ok {
			return nil, jqError(e)
		}
		outs = append(outs, v)
	}

	// jq-web's jq.json() collapses the output stream: 0 -> error, 1 -> value,
	// N -> array.
	var result any
	switch len(outs) {
	case 0:
		//nolint:staticcheck,revive // jq-web's verbatim error text
		return nil, errors.New("Invalid jq expression: Unexpected end of JSON input")
	case 1:
		result = outs[0]
	default:
		result = outs
	}

	if raw {
		if s, ok := result.(string); ok {
			return core.NewDish([]byte(s), core.TypeString), nil
		}
	}
	out, err := jqStringify(normalizeNonFinite(result))
	if err != nil {
		return nil, err
	}
	return core.NewDish([]byte(out), core.TypeString), nil
}

// normalizeNonFinite replaces NaN and ±Inf floats with nil throughout the value,
// so serialization matches JavaScript's JSON.stringify (which renders them as
// null) rather than failing as Go's encoding/json does.
func normalizeNonFinite(v any) any {
	switch t := v.(type) {
	case float64:
		if math.IsNaN(t) || math.IsInf(t, 0) {
			return nil
		}
	case []any:
		for i, e := range t {
			t[i] = normalizeNonFinite(e)
		}
	case map[string]any:
		for k, e := range t {
			t[k] = normalizeNonFinite(e)
		}
	}
	return v
}

// jqError wraps a gojq parse/runtime error the way CyberChef does.
func jqError(err error) error {
	//nolint:staticcheck,revive // CyberChef's verbatim OperationError prefix
	return errors.New("Invalid jq expression: " + err.Error())
}

// jqStringify serializes v the way JavaScript's JSON.stringify does: compact,
// without escaping <, > and &, and preserving UTF-8.
func jqStringify(v any) (string, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return "", err
	}
	return strings.TrimSuffix(b.String(), "\n"), nil
}
