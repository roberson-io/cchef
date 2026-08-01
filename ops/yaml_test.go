package ops

// Tests for the JSON to YAML / YAML to JSON operations.
//
// CyberChef backs these with two different YAML 1.2 JS libraries (`yaml` for
// JSON to YAML, `js-yaml` for YAML to JSON). cchef uses the Go
// `go.yaml.in/yaml/v3` library for both. The fixture cases come from
// ../CyberChef/tests/operations/tests/JSONtoYAML.mjs; the larger tables were
// derived from the CyberChef-server oracle and cover cases where the Go library
// agrees with CyberChef. TestYAMLDivergences documents the known YAML 1.1-vs-1.2
// differences where cchef intentionally follows the Go library instead.

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

var (
	j2yRecipe = core.Recipe{{Op: "JSON to YAML"}}
	y2jRecipe = core.Recipe{{Op: "YAML to JSON"}}
)

// TestJSONToYAMLFixture transcribes CyberChef's JSONtoYAML.mjs fixture.
func TestJSONToYAMLFixture(t *testing.T) {
	runCases(t, []opCase{
		{"JSON to YAML", `{ "number": 3, "plain": "string" }`, "number: 3\nplain: string\n", j2yRecipe},
	})
}

// TestYAMLToJSONFixture transcribes CyberChef's JSONtoYAML.mjs YAML-to-JSON case.
func TestYAMLToJSONFixture(t *testing.T) {
	runCases(t, []opCase{
		{
			"YAML to JSON", "number: 3\nplain: string\nblock: |\n  two\n  lines",
			"{\n    \"number\": 3,\n    \"plain\": \"string\",\n    \"block\": \"two\\nlines\\n\"\n}", y2jRecipe,
		},
	})
}

// TestJSONToYAMLVectors covers scalars, containers, key-order preservation,
// nesting, the integer-vs-scientific number fix, block scalars and unicode —
// all oracle-verified cases where the Go library matches CyberChef.
func TestJSONToYAMLVectors(t *testing.T) {
	runCases(t, []opCase{
		{"single pair", `{"a":1}`, "a: 1\n", j2yRecipe},
		{"sequence", `[1,2,3]`, "- 1\n- 2\n- 3\n", j2yRecipe},
		{"top string", `"hello"`, "hello\n", j2yRecipe},
		{"top int", `42`, "42\n", j2yRecipe},
		{"top bool", `true`, "true\n", j2yRecipe},
		{"top null", `null`, "null\n", j2yRecipe},
		{"empty map", `{}`, "{}\n", j2yRecipe},
		{"empty seq", `[]`, "[]\n", j2yRecipe},
		{"nested empty map", `{"a":{}}`, "a: {}\n", j2yRecipe},
		{"nested empty seq", `{"a":[]}`, "a: []\n", j2yRecipe},
		{"key order preserved", `{"b":1,"a":2}`, "b: 1\na: 2\n", j2yRecipe},
		{
			"nested", `{"nested":{"a":[1,2,3],"b":true},"nul":null}`,
			"nested:\n  a:\n    - 1\n    - 2\n    - 3\n  b: true\nnul: null\n", j2yRecipe,
		},
		{
			"numbers not scientific", `{"f":3.14,"big":10000000000,"neg":-5,"z":0}`,
			"f: 3.14\nbig: 10000000000\nneg: -5\nz: 0\n", j2yRecipe,
		},
		{
			"block scalars", `{"multi":"two\nlines\n","nl":"a\nb"}`,
			"multi: |\n  two\n  lines\nnl: |-\n  a\n  b\n", j2yRecipe,
		},
		{"seq of maps", `{"list":[{"x":1},{"z":2}]}`, "list:\n  - x: 1\n  - z: 2\n", j2yRecipe},
		{"unicode BMP", `{"greeting":"héllo"}`, "greeting: héllo\n", j2yRecipe},
		{
			"quoted keyword strings", `{"a":"true","b":"123","c":"null","d":""}`,
			"a: \"true\"\nb: \"123\"\nc: \"null\"\nd: \"\"\n", j2yRecipe,
		},
	})
}

// TestYAMLToJSONVectors covers flow/block collections, scalar types, anchors and
// aliases, quoted scalars and nesting — oracle-verified agreeing cases.
func TestYAMLToJSONVectors(t *testing.T) {
	runCases(t, []opCase{
		{"flow map", `{a: 1, b: two}`, "{\n    \"a\": 1,\n    \"b\": \"two\"\n}", y2jRecipe},
		{"flow seq", `[1, 2, 3]`, "[\n    1,\n    2,\n    3\n]", y2jRecipe},
		{"top scalar", `42`, "42", y2jRecipe},
		{
			"types", "i: 42\nf: 3.14\ns: hello\nb: true\nn: null\nneg: -5",
			"{\n    \"i\": 42,\n    \"f\": 3.14,\n    \"s\": \"hello\",\n    \"b\": true,\n    \"n\": null,\n    \"neg\": -5\n}", y2jRecipe,
		},
		{
			"nested", "a:\n  b:\n    - 1\n    - 2\n  c: true",
			"{\n    \"a\": {\n        \"b\": [\n            1,\n            2\n        ],\n        \"c\": true\n    }\n}", y2jRecipe,
		},
		{"anchors and aliases", "a: &x 1\nb: *x", "{\n    \"a\": 1,\n    \"b\": 1\n}", y2jRecipe},
		{"quoted scalars", "s: \"123\"\nt: \"true\"", "{\n    \"s\": \"123\",\n    \"t\": \"true\"\n}", y2jRecipe},
		{"empty string value", `a: ""`, "{\n    \"a\": \"\"\n}", y2jRecipe},
		{"block strip", "x: |-\n  a\n  b", "{\n    \"x\": \"a\\nb\"\n}", y2jRecipe},
		// yes/no/on parse as strings (YAML 1.2), matching js-yaml.
		{
			"yes/no/on stay strings", "a: yes\nb: no\nc: on",
			"{\n    \"a\": \"yes\",\n    \"b\": \"no\",\n    \"c\": \"on\"\n}", y2jRecipe,
		},
		// Timestamps resolve to ISO-8601 strings, matching js-yaml's Date output.
		{"date", "d: 2020-01-01", "{\n    \"d\": \"2020-01-01T00:00:00.000Z\"\n}", y2jRecipe},
		{"datetime", "d: 2020-01-01T12:30:00Z", "{\n    \"d\": \"2020-01-01T12:30:00.000Z\"\n}", y2jRecipe},
		// Large integers lose precision to JS float64, matching js-yaml.
		{"large int (float64)", "n: 9223372036854775807", "{\n    \"n\": 9223372036854776000\n}", y2jRecipe},
		{"huge int (uint64)", "n: 18446744073709551615", "{\n    \"n\": 18446744073709552000\n}", y2jRecipe},
		// An empty/comment-only document is a null node.
		{"comment only", "# just a comment", "null", y2jRecipe},
		{"integer map key coerced", "1: a\n2: b", "{\n    \"1\": \"a\",\n    \"2\": \"b\"\n}", y2jRecipe},
	})
}

// TestYAMLDivergences pins the intentional YAML 1.1-vs-1.2 differences between
// cchef's Go library and CyberChef's JS libraries. These assert cchef's actual
// behaviour; the comments record what CyberChef emits instead.
func TestYAMLDivergences(t *testing.T) {
	runCases(t, []opCase{
		// CyberChef (YAML 1.2) leaves "yes" a plain string; the Go library quotes
		// it defensively because YAML 1.1 treats yes/no/on as booleans.
		{"yes quoted on output", `{"s":"yes"}`, "s: \"yes\"\n", j2yRecipe},
		// Same for a bool-family map key: CyberChef emits `y:`, the Go library `"y":`.
		{"bool-family key quoted", `{"y":1}`, "\"y\": 1\n", j2yRecipe},
		// CyberChef emits double quotes ("has: colon"); the Go library uses single
		// quotes for the same string.
		{"single-quote style", `{"q":"has: colon"}`, "q: 'has: colon'\n", j2yRecipe},
		// CyberChef emits astral characters raw (emoji: 😀); the Go library escapes
		// them in a double-quoted scalar.
		{"astral char escaped", `{"emoji":"😀"}`, "emoji: \"\\U0001F600\"\n", j2yRecipe},
	})
}

// TestYAMLToJSONErrors covers input that is not valid YAML, plus duplicate
// mapping keys (which js-yaml also rejects).
func TestYAMLToJSONErrors(t *testing.T) {
	for _, in := range []string{"a: b: c", "[unclosed", "{a: 1", "\t- x", "a: 1\na: 2"} {
		if _, err := (YAMLToJSON{}).Run(sdish(in), nil); err == nil {
			t.Fatalf("parse %q: expected error", in)
		}
	}
}

// TestJSONToYAMLErrors covers invalid JSON input and the encoder's guard against
// an unsupported Go value type (unreachable through the JSON front door).
func TestJSONToYAMLErrors(t *testing.T) {
	for _, in := range []string{"", "not json", "{bad}", "[1,2", "1 2"} {
		if _, err := (JSONToYAML{}).Run(sdish(in), nil); err == nil {
			t.Fatalf("encode %q: expected error", in)
		}
	}
	if _, err := yamlBuildNode(struct{}{}); err == nil {
		t.Fatal("yamlBuildNode(struct{}{}): expected error")
	}
	if _, err := yamlBuildNode([]any{struct{}{}}); err == nil {
		t.Fatal("yamlBuildNode of a bad sequence element: expected error")
	}
	if _, err := yamlBuildNode(jsObject{{k: "k", v: struct{}{}}}); err == nil {
		t.Fatal("yamlBuildNode of a bad map value: expected error")
	}
}

// TestYAMLRoundTrip encodes JSON to YAML and back, checking the value survives.
func TestYAMLRoundTrip(t *testing.T) {
	for _, in := range []string{
		`{"a":1,"b":"two","c":[1,2,3],"d":{"e":true,"f":null}}`,
		`[1,"two",3.5,false,null]`,
		`{"nested":{"deep":{"x":[{"y":1}]}}}`,
	} {
		y, err := (JSONToYAML{}).Run(sdish(in), nil)
		if err != nil {
			t.Fatalf("to yaml %q: %v", in, err)
		}
		j, err := (YAMLToJSON{}).Run(core.NewDish(y.Bytes(), core.TypeString), nil)
		if err != nil {
			t.Fatalf("to json %q: %v", in, err)
		}
		// YAML to JSON pretty-prints (indent 4); compare against the compact input
		// via a re-parse.
		got, err := jsonParseOrdered(j.Bytes())
		if err != nil {
			t.Fatalf("reparse %q: %v", in, err)
		}
		want, _ := jsonParseOrdered([]byte(in))
		if jsStringify(got, 0) != jsStringify(want, 0) {
			t.Fatalf("round-trip %q = %q", in, jsStringify(got, 0))
		}
	}
}
