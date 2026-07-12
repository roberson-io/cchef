package ops

import (
	"bytes"
	"fmt"
	"math"
	"strings"
	"time"

	yaml "go.yaml.in/yaml/v3"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(JSONToYAML{})
	core.Register(YAMLToJSON{})
}

// CyberChef backs these two operations with different JavaScript libraries —
// JSON to YAML uses `yaml` (eemeli), YAML to JSON uses `js-yaml` — both of which
// follow the YAML 1.2 core schema. cchef uses the Go `go.yaml.in/yaml/v3`
// library for both. Output is faithful for the common cases; it differs on a few
// YAML 1.1-vs-1.2 points documented in docs/data-format.md (e.g. yes/no/on are
// quoted on output and parsed as booleans on input; quoting style may differ).

// JSONToYAML formats a JSON value as YAML.
type JSONToYAML struct{}

// Meta returns the operation metadata.
func (JSONToYAML) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "JSON to YAML",
		Module:      "Default",
		Description: "Format a JSON object into YAML.",
		InfoURL:     "https://en.wikipedia.org/wiki/YAML",
		InputType:   core.TypeJSON,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (JSONToYAML) Args() []core.ArgDef { return nil }

// Run converts JSON input to YAML.
func (JSONToYAML) Run(in *core.Dish, _ []any) (*core.Dish, error) {
	v, err := jsonParseOrdered(in.Bytes())
	if err != nil {
		return nil, fmt.Errorf("JSON to YAML: parse JSON input: %w", err)
	}
	node, err := yamlBuildNode(v)
	if err != nil {
		return nil, fmt.Errorf("JSON to YAML: %w", err)
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(node); err != nil {
		return nil, fmt.Errorf("JSON to YAML: %w", err)
	}
	_ = enc.Close()
	return core.NewDish(buf.Bytes(), core.TypeString), nil
}

// YAMLToJSON converts YAML into JSON.
type YAMLToJSON struct{}

// Meta returns the operation metadata.
func (YAMLToJSON) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "YAML to JSON",
		Module:      "Default",
		Description: "Convert YAML to JSON.",
		InfoURL:     "https://en.wikipedia.org/wiki/YAML",
		InputType:   core.TypeString,
		OutputType:  core.TypeJSON,
	}
}

// Args returns the argument definitions.
func (YAMLToJSON) Args() []core.ArgDef { return nil }

// Run converts YAML input to JSON.
func (YAMLToJSON) Run(in *core.Dish, _ []any) (*core.Dish, error) {
	// js-yaml keeps a literal block scalar's trailing newline even when the
	// document has no final newline; go-yaml only does so when one is present.
	// Appending a newline (harmless for YAML) reconciles the two.
	data := in.Bytes()
	if n := len(data); n > 0 && data[n-1] != '\n' {
		data = append(append([]byte(nil), data...), '\n')
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("Unable to parse YAML: %w", err) //nolint:staticcheck // mirrors CyberChef's "Unable to parse YAML" message
	}
	v, err := yamlNodeToValue(&doc)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse YAML: %w", err) //nolint:staticcheck // mirrors CyberChef's "Unable to parse YAML" message
	}
	return core.NewDish([]byte(jsStringify(v, 4)), core.TypeJSON), nil
}

// --- JSON -> YAML node building ---

// yamlBuildNode converts an order-preserving JSON value into a yaml.Node tree.
func yamlBuildNode(v any) (*yaml.Node, error) {
	switch x := v.(type) {
	case nil:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "null"}, nil
	case bool:
		var n yaml.Node
		if err := n.Encode(x); err != nil {
			return nil, err
		}
		return &n, nil
	case float64:
		return yamlNumberNode(x), nil
	case string:
		var n yaml.Node
		if err := n.Encode(x); err != nil {
			return nil, err
		}
		return &n, nil
	case []any:
		n := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		for _, e := range x {
			child, err := yamlBuildNode(e)
			if err != nil {
				return nil, err
			}
			n.Content = append(n.Content, child)
		}
		return n, nil
	case jsObject:
		n := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		for _, p := range x {
			key, err := yamlBuildNode(p.k)
			if err != nil {
				return nil, err
			}
			val, err := yamlBuildNode(p.v)
			if err != nil {
				return nil, err
			}
			n.Content = append(n.Content, key, val)
		}
		return n, nil
	default:
		return nil, fmt.Errorf("unsupported value type %T", v)
	}
}

// yamlNumberNode renders a JSON number the way the JS libraries do: an
// integer-valued number becomes a plain integer (not scientific notation),
// everything else a float, both formatted like JSON.stringify.
func yamlNumberNode(f float64) *yaml.Node {
	s := jsFormatNumber(f)
	tag := "!!float"
	if f == math.Trunc(f) && !math.IsInf(f, 0) && !math.IsNaN(f) && !strings.ContainsAny(s, ".eE") {
		tag = "!!int"
	}
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: tag, Value: s}
}

// --- YAML -> JSON value conversion ---

// yamlNodeToValue walks a yaml.Node tree into an order-preserving JSON value.
func yamlNodeToValue(n *yaml.Node) (any, error) {
	switch n.Kind {
	case 0:
		// An empty, blank or comment-only document is a null node.
		return nil, nil
	case yaml.DocumentNode:
		if len(n.Content) == 0 {
			return nil, nil
		}
		return yamlNodeToValue(n.Content[0])
	case yaml.MappingNode:
		obj := jsObject{}
		for i := 0; i+1 < len(n.Content); i += 2 {
			key, err := yamlMapKey(n.Content[i])
			if err != nil {
				return nil, err
			}
			val, err := yamlNodeToValue(n.Content[i+1])
			if err != nil {
				return nil, err
			}
			if idx := jsIndex(obj, key); idx >= 0 {
				// js-yaml rejects duplicate mapping keys; match that.
				return nil, fmt.Errorf("duplicated mapping key %q", key)
			}
			obj = append(obj, jsPair{k: key, v: val})
		}
		return obj, nil
	case yaml.SequenceNode:
		arr := []any{}
		for _, c := range n.Content {
			val, err := yamlNodeToValue(c)
			if err != nil {
				return nil, err
			}
			arr = append(arr, val)
		}
		return arr, nil
	case yaml.AliasNode:
		return yamlNodeToValue(n.Alias)
	default: // ScalarNode
		return yamlScalar(n)
	}
}

// yamlMapKey converts a mapping key node to its JSON string form (JSON object
// keys are always strings, matching JSON.stringify of a parsed YAML object).
func yamlMapKey(n *yaml.Node) (string, error) {
	v, err := yamlNodeToValue(n)
	if err != nil {
		return "", err
	}
	return jsToString(v), nil
}

// yamlScalar resolves a scalar node to a JSON value using the library's tag
// resolution: integers, floats, booleans, null and strings map directly, and a
// timestamp renders as an ISO-8601 string (matching js-yaml's Date output).
func yamlScalar(n *yaml.Node) (any, error) {
	if n.Tag == "!!timestamp" {
		var t time.Time
		if err := n.Decode(&t); err == nil {
			return t.UTC().Format("2006-01-02T15:04:05.000Z"), nil
		}
	}
	var v any
	if err := n.Decode(&v); err != nil {
		return nil, err
	}
	// js-yaml resolves numbers to JavaScript numbers (float64), so integers
	// beyond 2^53 lose precision; mirror that rather than keeping Go's exact
	// int64.
	switch x := v.(type) {
	case int:
		return float64(x), nil
	case int64:
		return float64(x), nil
	case uint64:
		return float64(x), nil
	case float64:
		return x, nil
	case time.Time:
		return x.UTC().Format("2006-01-02T15:04:05.000Z"), nil
	default: // bool, string, nil
		return v, nil
	}
}
