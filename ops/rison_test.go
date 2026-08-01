package ops

import (
	"math"
	"testing"

	"github.com/roberson-io/cchef/core"
)

func risonEnc(option string) core.Recipe {
	return core.Recipe{{Op: "Rison Encode", Args: []any{option}}}
}

func risonDec(option string) core.Recipe {
	return core.Recipe{{Op: "Rison Decode", Args: []any{option}}}
}

// TestRisonEncodeFixtures covers the upstream CyberChef fixtures plus vectors
// generated from the same `rison` npm library CyberChef bundles (used as the
// oracle). Input is JSON (the Object dish); output is a Rison string.
func TestRisonEncodeFixtures(t *testing.T) {
	runCases(t, []opCase{
		// upstream fixtures
		{"encode object", `{"any":"json","yes":true}`, "(any:json,yes:!t)", risonEnc("Encode")},
		{"encode_object strips parens", `{"supportsObjects":true,"ints":435}`, "ints:435,supportsObjects:!t", risonEnc("Encode Object")},
		{"encode_array strips brackets", `["A","B",{"supportsObjects":true}]`, "A,B,(supportsObjects:!t)", risonEnc("Encode Array")},

		// key sorting (string sort) and numeric-string keys get quoted
		{"key sorting", `{"2":"y","10":"x","b":2,"a":1}`, "('10':x,'2':y,a:1,b:2)", risonEnc("Encode")},

		// scalars & strings
		{"bare string", `"hello"`, "hello", risonEnc("Encode")},
		{"empty string", `""`, "''", risonEnc("Encode")},
		{"string with space", `"with space"`, "'with space'", risonEnc("Encode")},
		{"string with quote", `"has'quote"`, "'has!'quote'", risonEnc("Encode")},
		{"string with bang", `"has!bang"`, "'has!!bang'", risonEnc("Encode")},
		{"string with punctuation", `"a:b,c(d)"`, "'a:b,c(d)'", risonEnc("Encode")},
		{"unicode bare", `"unïcödé"`, "unïcödé", risonEnc("Encode")},
		{"reserved glyphs quoted", `"*@$~/"`, "'*@$~/'", risonEnc("Encode")},

		// numbers
		{"integer", `435`, "435", risonEnc("Encode")},
		{"negative float", `-3.14`, "-3.14", risonEnc("Encode")},
		{"exponent strips plus", `1e+30`, "1e30", risonEnc("Encode")},
		{"negative exponent", `1e-30`, "1e-30", risonEnc("Encode")},
		{"zero", `0`, "0", risonEnc("Encode")},

		// booleans & null
		{"true", `true`, "!t", risonEnc("Encode")},
		{"false", `false`, "!f", risonEnc("Encode")},
		{"null", `null`, "!n", risonEnc("Encode")},

		// nesting
		{"nested array", `[1,2,[3,4],{"x":null}]`, "!(1,2,!(3,4),(x:!n))", risonEnc("Encode")},
		{"object with mixed", `{"key with space":"v","n":null,"arr":[1,"two",true]}`, "(arr:!(1,two,!t),'key with space':v,n:!n)", risonEnc("Encode")},

		// Encode URI (rison.quote, incl. its replace-first-only quirk)
		{"uri object", `{"a":"b c","d":["e,f","g:h"]}`, "(a:'b+c',d%3A!('e%2Cf'%2C'g%3Ah'))", risonEnc("Encode URI")},
		{"uri string", `"hello world & more, stuff"`, "'hello+world%20%26%20more,%20stuff'", risonEnc("Encode URI")},
		{"uri safe commas", `"a,b,c,d"`, "'a,b,c,d'", risonEnc("Encode URI")},
		{"uri safe colons", `"a:b:c:d"`, "'a:b:c:d'", risonEnc("Encode URI")},
	})
}

// TestRisonDecodeFixtures covers decoding a Rison string to the Object dish,
// rendered as 4-space-indented JSON (matching CyberChef's JSON.stringify(x,null,4)).
func TestRisonDecodeFixtures(t *testing.T) {
	runCases(t, []opCase{
		{"decode object", "(any:json,yes:!t)", "{\n    \"any\": \"json\",\n    \"yes\": true\n}", risonDec("Decode")},
		{"decode true", "!t", "true", risonDec("Decode")},
		{"decode false", "!f", "false", risonDec("Decode")},
		{"decode null", "!n", "null", risonDec("Decode")},
		{"decode integer", "435", "435", risonDec("Decode")},
		{"decode float", "-3.14", "-3.14", risonDec("Decode")},
		{"decode exponent", "1e30", "1e+30", risonDec("Decode")},
		{"decode quoted string", "'hello world'", "\"hello world\"", risonDec("Decode")},
		{"decode empty string", "''", "\"\"", risonDec("Decode")},
		{"decode array", "!(1,2,3)", "[\n    1,\n    2,\n    3\n]", risonDec("Decode")},
		{"decode escaped quote", "'has!'quote'", "\"has'quote\"", risonDec("Decode")},
		{"decode escaped bang", "'has!!bang'", "\"has!bang\"", risonDec("Decode")},
		{"decode bare id", "hello", "\"hello\"", risonDec("Decode")},
		{
			"decode nested", "(a:1,b:!(2,3),c:(d:!t))",
			"{\n    \"a\": 1,\n    \"b\": [\n        2,\n        3\n    ],\n    \"c\": {\n        \"d\": true\n    }\n}", risonDec("Decode"),
		},
		{
			"decode deep", "(nested:(deep:!(a,b,'c d')))",
			"{\n    \"nested\": {\n        \"deep\": [\n            \"a\",\n            \"b\",\n            \"c d\"\n        ]\n    }\n}", risonDec("Decode"),
		},

		// decode_object / decode_array wrap the input
		{"decode_object", "a:1,b:2", "{\n    \"a\": 1,\n    \"b\": 2\n}", risonDec("Decode Object")},
		{"decode_array", "1,2,3", "[\n    1,\n    2,\n    3\n]", risonDec("Decode Array")},

		// number edges
		{"decode negative exponent", "1e-3", "0.001", risonDec("Decode")},
		{"decode malformed number is NaN", "1e", "null", risonDec("Decode")},

		// non-string keys are String()-coerced (JS object[key] semantics)
		{"number key", "(1:a)", "{\n    \"1\": \"a\"\n}", risonDec("Decode")},
		{"bool key", "(!t:a)", "{\n    \"true\": \"a\"\n}", risonDec("Decode")},
		{"bool false key", "(!f:a)", "{\n    \"false\": \"a\"\n}", risonDec("Decode")},
		{"null key", "(!n:a)", "{\n    \"null\": \"a\"\n}", risonDec("Decode")},
		{"array key", "(!(1,2):a)", "{\n    \"1,2\": \"a\"\n}", risonDec("Decode")},
		{"object key", "((a:1):v)", "{\n    \"[object Object]\": \"v\"\n}", risonDec("Decode")},

		// duplicate keys: last value wins
		{"duplicate key", "(a:1,a:2)", "{\n    \"a\": 2\n}", risonDec("Decode")},
	})
}

// TestRisonEncodeErrors covers the rison.encode_* type checks and the invalid
// option path.
func TestRisonEncodeErrors(t *testing.T) {
	cases := []struct{ name, input, option, want string }{
		{"array on object", `{"not":"array"}`, "Encode Array", "rison.encode_array expects an array argument"},
		{"object on array", `["not","object"]`, "Encode Object", "rison.encode_object expects an object argument"},
		{"object on null", `null`, "Encode Object", "rison.encode_object expects an object argument"},
		{"object on string", `"astring"`, "Encode Object", "rison.encode_object expects an object argument"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := runOp(t, "Rison Encode", c.input, c.option)
			if err == nil {
				t.Fatalf("expected error %q", c.want)
			}
			if err.Error() != c.want {
				t.Fatalf("got %q, want %q", err.Error(), c.want)
			}
		})
	}
}

// TestRisonDecodeErrors covers the parser's clean error messages (reproduced
// verbatim) and the invalid option path.
func TestRisonDecodeErrors(t *testing.T) {
	cases := []struct{ name, input, option, want string }{
		{"missing colon", "(a", "Decode", "rison decoder error: missing ':'"},
		{"missing colon id", "(a1)", "Decode", "rison decoder error: missing ':'"},
		{"unknown literal", "!x", "Decode", "rison decoder error: unknown literal: \"!x\""},
		{"bang at end", "!", "Decode", "rison decoder error: \"!\" at end of input"},
		{"unmatched quote", "'unterminated", "Decode", "rison decoder error: unmatched \"'\""},
		{"extra comma object", "(,a:1)", "Decode", "rison decoder error: extra ','"},
		{"extra comma array", "!(,1)", "Decode", "rison decoder error: extra ','"},
		{"trailing junk", "a b", "Decode", "rison decoder error: unable to parse string as rison: ''a b''"},
		{"invalid number", "-", "Decode", "rison decoder error: invalid number"},
		{"unmatched array", "!(1", "Decode", "rison decoder error: unmatched '!('"},
		{"missing comma array", "!(1 2)", "Decode", "rison decoder error: missing ','"},
		{"missing comma object", "(a:1 b:2)", "Decode", "rison decoder error: missing ','"},
		{"unmatched quote after bang", "'a!", "Decode", "rison decoder error: unmatched \"'\""},
		{"invalid string escape", "'a!b'", "Decode", "rison decoder error: invalid string escape: \"!b\""},
		{"object value error", "(a:!x)", "Decode", "rison decoder error: unknown literal: \"!x\""},
		{"invalid option", "hello", "Bogus", "Invalid Decode option"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := runOp(t, "Rison Decode", c.input, c.option)
			if err == nil {
				t.Fatalf("expected error %q", c.want)
			}
			if err.Error() != c.want {
				t.Fatalf("got %q, want %q", err.Error(), c.want)
			}
		})
	}
}

// TestRisonDecodeMalformed covers inputs where rison.js throws a raw JS
// TypeError (or stack-overflows); cchef returns a clean parse error instead
// (documented divergence).
func TestRisonDecodeMalformed(t *testing.T) {
	for _, in := range []string{"", "(a:1,)", "(a:1", "1,,2", "!(1,,2)", "(", "!("} {
		if _, err := runOp(t, "Rison Decode", in, "Decode"); err == nil {
			t.Errorf("input %q: expected an error", in)
		}
	}
}

// TestRisonEncodeInvalidJSON covers the JSON-parse error path of Rison Encode.
func TestRisonEncodeInvalidJSON(t *testing.T) {
	if _, err := runOp(t, "Rison Encode", "{not json", "Encode"); err == nil {
		t.Fatal("expected an error for invalid JSON input")
	}
}

// TestRisonEncodeInternals directly exercises encoder branches that JSON input
// never reaches (non-finite numbers and unencodable values such as the
// undefined marker), mirroring the rison library's own guards.
func TestRisonEncodeInternals(t *testing.T) {
	if got := risonEncodeNumber(math.Inf(1)); got != "!n" {
		t.Errorf("Inf: got %q, want !n", got)
	}
	if got := risonEncodeNumber(math.NaN()); got != "!n" {
		t.Errorf("NaN: got %q, want !n", got)
	}
	if _, err := risonEncodeValue(jsUndefined{}); err == nil {
		t.Error("risonEncodeValue(undefined) should error")
	}
	if _, err := risonEncodeArrayVal([]any{jsUndefined{}}); err == nil {
		t.Error("risonEncodeArrayVal with undefined should error")
	}
	if _, err := risonEncodeObjectVal(jsObject{{k: "x", v: jsUndefined{}}}); err == nil {
		t.Error("risonEncodeObjectVal with undefined should error")
	}
	if _, err := risonEncodeObjectEntry(jsObject{{k: "x", v: jsUndefined{}}}); err == nil {
		t.Error("risonEncodeObjectEntry with undefined should error")
	}
	if _, err := risonEncodeArrayEntry([]any{jsUndefined{}}); err == nil {
		t.Error("risonEncodeArrayEntry with undefined should error")
	}
	if _, err := risonEncodeURI(jsUndefined{}); err == nil {
		t.Error("risonEncodeURI with undefined should error")
	}
	// risonEncodeString short-circuits "" before risonIDOk, so cover its empty
	// guard directly (mirrors the id_ok regex rejecting the empty string).
	if risonIDOk("") {
		t.Error("risonIDOk(\"\") should be false")
	}
}
