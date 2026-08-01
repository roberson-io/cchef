package ops

// Confirms the shared ordered-JSON helpers (jsonvalue.go) reproduce esprima's
// JSON.stringify(ast, null, 2) output — number formatting, string escaping and
// pretty-printed structure — so the JavaScript Parser can reuse them. Expected
// values captured from Node (scratchpad/jsoncases.mjs).

import (
	"math"
	"testing"
)

func TestJSParserNumberFormat(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{0, "0"},
		{1, "1"},
		{-1, "-1"},
		{1.5, "1.5"},
		{0.1, "0.1"},
		{100, "100"},
		{1e10, "10000000000"},
		{1e21, "1e+21"},
		{1e-7, "1e-7"},
		{123456789012345680, "123456789012345680"},
		{math.Copysign(0, -1), "0"},
	}
	for _, c := range cases {
		if got := jsFormatNumber(c.in); got != c.want {
			t.Errorf("jsFormatNumber(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestJSParserStringEscape(t *testing.T) {
	// Short escapes plus a raw control character (U+0001) rendered as .
	in := "a\"b\\c\td\ne\rf\bg\fh" + string(rune(1)) + "i"
	want := "\"a\\\"b\\\\c\\td\\ne\\rf\\bg\\fh\\u0001i\""
	if got := jsJSONString(in); got != want {
		t.Errorf("jsJSONString = %q, want %q", got, want)
	}
	if got := jsJSONString("café中😀"); got != `"café中😀"` {
		t.Errorf("jsJSONString(unicode) = %q", got)
	}
}

func TestJSParserStringifyShape(t *testing.T) {
	// A Program-shaped AST renders identically to JSON.stringify(x, null, 2).
	prog := jsObject{
		{k: "type", v: "Program"},
		{k: "body", v: []any{jsObject{{k: "type", v: "X"}, {k: "v", v: float64(1)}}}},
		{k: "sourceType", v: "script"},
	}
	want := "{\n  \"type\": \"Program\",\n  \"body\": [\n    {\n      \"type\": \"X\",\n      \"v\": 1\n    }\n  ],\n  \"sourceType\": \"script\"\n}"
	if got := jsStringify(prog, 2); got != want {
		t.Errorf("stringify:\n%s\nwant:\n%s", got, want)
	}
	mixed := jsObject{{k: "n", v: nil}, {k: "t", v: true}, {k: "arr", v: []any{}}}
	wantMixed := "{\n  \"n\": null,\n  \"t\": true,\n  \"arr\": []\n}"
	if got := jsStringify(mixed, 2); got != wantMixed {
		t.Errorf("mixed:\n%s\nwant:\n%s", got, wantMixed)
	}
}
