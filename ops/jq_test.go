package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// jqRecipe builds a single-step Jq recipe with the query and raw flag.
func jqRecipe(query string, raw bool) core.Recipe {
	return core.Recipe{{Op: "Jq", Args: []any{query, raw}}}
}

// The first two cases are transcribed from CyberChef's tests/operations/tests/Jq.mjs;
// the rest are authoritative outputs captured from the CyberChef-server oracle.
// CyberChef wraps jq-web (jq compiled to WASM); cchef reimplements the operation
// over gojq (a pure-Go jq), reproducing jq-web's .json() output collapse: zero
// results error, one result is returned directly, and multiple results become a
// JSON array; the value is then raw-printed (raw + string) or JSON.stringify'd.
func TestJqFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"Jq: raw JSON property", "{\"data\": \"testString\\u0000\"}",
			"testString\x00", jqRecipe(".data", true),
		},
		{
			"Jq: JSON property", "{\"data\": \"testString\\u0000\"}",
			"\"testString\\u0000\"", jqRecipe(".data", false),
		},

		{
			"Jq: stream collapses to array", `[1,2,3]`, `[1,2,3]`,
			jqRecipe(".[]", false),
		},
		{
			"Jq: mapped stream to array", `[1,2,3]`, `[2,3,4]`,
			jqRecipe(".[]|.+1", false),
		},
		{"Jq: map", `[1,2,3]`, `[2,4,6]`, jqRecipe("map(.*2)", false)},
		{
			"Jq: keys are sorted", `{"b":2,"a":1}`, `["a","b"]`,
			jqRecipe("keys", false),
		},
		{
			"Jq: object construction", `{"a":1}`, `{"x":1,"y":2}`,
			jqRecipe("{x:.a,y:2}", false),
		},
		{"Jq: raw string", `{"a":"hi"}`, `hi`, jqRecipe(".a", true)},
		{"Jq: non-raw string", `{"a":"hi"}`, `"hi"`, jqRecipe(".a", false)},
		{"Jq: raw on number stringifies", `{"a":5}`, `5`, jqRecipe(".a", true)},
		{"Jq: float formatting", `{}`, `0.3333333333333333`, jqRecipe("1/3", false)},
		{"Jq: null", `{}`, `null`, jqRecipe(".foo", false)},
		{"Jq: unicode preserved", `{"a":"café"}`, `"café"`, jqRecipe(".a", false)},
		{
			"Jq: to_entries", `{"a":1,"b":2}`,
			`[{"key":"a","value":1},{"key":"b","value":2}]`,
			jqRecipe("to_entries", false),
		},
		{"Jq: sort", `[3,1,2]`, `[1,2,3]`, jqRecipe("sort", false)},
		{"Jq: length", `{"x":[1,2,3]}`, `3`, jqRecipe(".x|length", false)},
		{
			"Jq: nested access", `{"a":{"b":{"c":42}}}`, `42`,
			jqRecipe(".a.b.c", false),
		},
		{
			"Jq: ascii_upcase", `"hello"`, `"HELLO"`,
			jqRecipe("ascii_upcase", false),
		},
		// NaN serializes as null, matching both jq's own output and JavaScript's
		// JSON.stringify (Go's encoding/json would otherwise error on NaN/Inf).
		// `infinite` is handled the same way to avoid an encode error, but its
		// exact jq-web output is unverified (jq clamps to DBL_MAX on dump), so it
		// is not asserted here pending oracle confirmation.
		{"Jq: nan is null", `{}`, `null`, jqRecipe("nan", false)},
	})
}

func TestJqErrors(t *testing.T) {
	// Zero results reproduce jq-web's exact message.
	if _, err := runOp(t, "Jq", `[1,2,3]`, ".[]|select(.>5)", false); err == nil ||
		err.Error() != "Invalid jq expression: Unexpected end of JSON input" {
		t.Fatalf("empty result: got %v", err)
	}
	// A syntax error is reported (gojq's message text differs from jq-web's, so
	// only the prefix is asserted).
	if _, err := runOp(t, "Jq", `{"a":1}`, ".a+", false); err == nil ||
		!strings.HasPrefix(err.Error(), "Invalid jq expression: ") {
		t.Fatalf("syntax error: got %v", err)
	}
	// Invalid JSON input.
	if _, err := runOp(t, "Jq", `{not json`, ".", false); err == nil {
		t.Fatal("expected error for invalid JSON input")
	}
	// A runtime error (as opposed to a parse error) is also reported.
	for _, q := range []string{".foo", "1/0"} {
		if _, err := runOp(t, "Jq", `123`, q, false); err == nil ||
			!strings.HasPrefix(err.Error(), "Invalid jq expression: ") {
			t.Fatalf("runtime error for %q: got %v", q, err)
		}
	}
}
