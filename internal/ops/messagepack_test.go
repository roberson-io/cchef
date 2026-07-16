package ops

// Tests for the To MessagePack / From MessagePack operations.
//
// CyberChef's MessagePack operations are thin wrappers around the `notepack.io`
// npm library (v3.0.1), calling notepack.encode / notepack.decode. This is a
// from-scratch port; there are no upstream fixture files, so every vector below
// was derived from that exact library used as an oracle (the CyberChef-server
// /bake endpoint running notepack in Node). They are ordinary tests — edit as
// needed.

import (
	"bytes"
	"encoding/hex"
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// msgpackEncodeRecipe encodes JSON input to MessagePack and renders it as
// contiguous lowercase hex for comparison.
var msgpackEncodeRecipe = core.Recipe{{Op: "To MessagePack"}, {Op: "To Hex", Args: []any{"None"}}}

// msgpackBytes decodes a hex string into raw bytes.
func msgpackBytes(t *testing.T, h string) []byte {
	t.Helper()
	b, err := hex.DecodeString(strings.ReplaceAll(h, " ", ""))
	if err != nil {
		t.Fatalf("bad hex %q: %v", h, err)
	}
	return b
}

// TestToMessagePackBasics covers the headline JSON types.
func TestToMessagePackBasics(t *testing.T) {
	runCases(t, []opCase{
		{"To MessagePack: integer", "15", "0f", msgpackEncodeRecipe},
		{"To MessagePack: float", "1.5", "cb3ff8000000000000", msgpackEncodeRecipe},
		{"To MessagePack: text", `"Text"`, "a454657874", msgpackEncodeRecipe},
		{"To MessagePack: true", "true", "c3", msgpackEncodeRecipe},
		{"To MessagePack: false", "false", "c2", msgpackEncodeRecipe},
		{"To MessagePack: null", "null", "c0", msgpackEncodeRecipe},
		{"To MessagePack: list", "[0,1,2]", "93000102", msgpackEncodeRecipe},
		{"To MessagePack: map", `{"a":1,"b":2,"c":3}`, "83a16101a16202a16303", msgpackEncodeRecipe},
	})
}

// TestToMessagePackVectors exercises every integer width (fixnum/uint/int
// 8/16/32/64), the float-64 fallback, JS ECMAScript key ordering (integer keys
// first, ascending, then insertion order), the large-integer bit-operation
// quirks notepack inherits from JavaScript, empty containers, nesting and
// unicode — all oracle-verified against notepack 3.0.1.
func TestToMessagePackVectors(t *testing.T) {
	vec := map[string]string{
		// positive integer widths
		"0":          "00",
		"127":        "7f",
		"128":        "cc80",
		"255":        "ccff",
		"256":        "cd0100",
		"65535":      "cdffff",
		"65536":      "ce00010000",
		"4294967295": "ceffffffff",
		"4294967296": "cf0000000100000000",
		// negative integer widths
		"-1":          "ff",
		"-32":         "e0",
		"-33":         "d0df",
		"-128":        "d080",
		"-129":        "d1ff7f",
		"-32768":      "d18000",
		"-32769":      "d2ffff7fff",
		"-2147483648": "d280000000",
		"-2147483649": "d3ffffffff7fffffff",
		// safe-integer edge and large integers (JS bit-op quirks preserved)
		"9007199254740991":     "cf001fffffffffffff",
		"9007199254740992":     "cf0020000000000000",
		"9007199254740994":     "cf0020000000000002",
		"-9007199254740991":    "d3ffe0000000000001",
		"1000000000000":        "cf000000e8d4a51000",
		"1234567890123456789":  "cf112210f47de98100",
		"1e19":                 "cf8ac7230489e80000",
		"1e20":                 "cf6bc75e2d63100000",
		"1e21":                 "cf35c9adc5dea00000",
		"-1e19":                "d37538dcfb76180000",
		"1e300":                "cf0000000000000000",
		"18446744073709551615": "cf0000000000000000",
		// floats (notepack always encodes non-integers as float 64)
		"1.1":     "cb3ff199999999999a",
		"2.5":     "cb4004000000000000",
		"0.5":     "cb3fe0000000000000",
		"3.14159": "cb400921f9f01b866e",
		"-0.0":    "00",
		"0.0":     "00",
		// strings, containers, nesting, unicode
		`""`:                "a0",
		`"héllo"`:           "a668c3a96c6c6f",
		`"😀"`:               "a4f09f9880",
		"[]":                "90",
		"{}":                "80",
		"[[1],[2,3]]":       "929101920203",
		`{"a":{"b":[1,2]}}`: "81a16181a162920102",
		// ECMAScript key ordering
		`{"b":1,"a":2}`:              "82a16201a16102",
		`{"2":1,"1":2,"b":3}`:        "83a13102a13201a16203",
		`{"10":1,"9":2,"1":3}`:       "83a13103a13902a2313001",
		`{"":1,"a":2}`:               "82a001a16102",
		`{"z":1,"2":2,"aa":3,"1":4}`: "84a13104a13202a17a01a2616103",
	}
	cases := make([]opCase, 0, len(vec))
	for in, want := range vec {
		cases = append(cases, opCase{"encode " + in, in, want, msgpackEncodeRecipe})
	}
	runCases(t, cases)
}

// TestToMessagePackWidths checks the fixstr/str8/str16/str32, fixarray/array16/
// array32 and fixmap/map16/map32 length-header widths at their boundaries. The
// headers are spec-deterministic; the small-length forms are oracle-verified
// elsewhere, so here we assert only the header prefix that the width selects.
func TestToMessagePackWidths(t *testing.T) {
	enc := func(v any) []byte {
		var buf bytes.Buffer
		if err := msgpackEncode(&buf, v); err != nil {
			t.Fatalf("encode: %v", err)
		}
		return buf.Bytes()
	}
	strOf := func(n int) string { return strings.Repeat("a", n) }
	arrOf := func(n int) []any {
		a := make([]any, n)
		for i := range a {
			a[i] = float64(0)
		}
		return a
	}
	mapOf := func(n int) jsObject {
		o := make(jsObject, n)
		for i := range o {
			o[i] = jsPair{k: strconv.Itoa(i), v: float64(0)}
		}
		return o
	}
	check := func(name string, got, wantPrefix []byte) {
		if !bytes.HasPrefix(got, wantPrefix) {
			t.Fatalf("%s: got prefix %x want %x", name, got[:min(len(got), len(wantPrefix))], wantPrefix)
		}
	}
	check("fixstr", enc(strOf(31)), []byte{0xbf})
	check("str8", enc(strOf(32)), []byte{0xd9, 0x20})
	check("str16", enc(strOf(256)), []byte{0xda, 0x01, 0x00})
	check("str32", enc(strOf(65536)), []byte{0xdb, 0x00, 0x01, 0x00, 0x00})
	check("fixarray", enc(arrOf(15)), []byte{0x9f})
	check("array16", enc(arrOf(16)), []byte{0xdc, 0x00, 0x10})
	check("array32", enc(arrOf(65536)), []byte{0xdd, 0x00, 0x01, 0x00, 0x00})
	check("fixmap", enc(mapOf(15)), []byte{0x8f})
	check("map16", enc(mapOf(16)), []byte{0xde, 0x00, 0x10})
	check("map32", enc(mapOf(65536)), []byte{0xdf, 0x00, 0x01, 0x00, 0x00})
}

// TestMessagePackTooLong covers the (otherwise unreachable) guard that rejects
// strings/arrays/maps whose length overflows MessagePack's 32-bit length field.
func TestMessagePackTooLong(t *testing.T) {
	if !msgpackTooLong(1 << 32) {
		t.Fatal("msgpackTooLong(2^32) = false, want true")
	}
	if msgpackTooLong(0xffffffff) {
		t.Fatal("msgpackTooLong(2^32-1) = true, want false")
	}
}

// TestFromMessagePackVectors exercises the full decode value model: integer and
// float widths, the uint64/int64 float widening, byte strings as Node Buffers,
// order-preserving and String()-coerced map keys, undefined omission, ext types
// and timestamps — all oracle-verified. Values are compared compact via the
// white-box decoder.
func TestFromMessagePackVectors(t *testing.T) {
	vec := []struct{ hex, want string }{
		// basics
		{"0f", "15"},
		{"cb3ff8000000000000", "1.5"},
		{"a454657874", `"Text"`},
		{"c3", "true"},
		{"c2", "false"},
		{"c0", "null"},
		{"93000102", "[0,1,2]"},
		{"83a16101a16202a16303", `{"a":1,"b":2,"c":3}`},
		// integer widths
		{"7f", "127"},
		{"cd0100", "256"},
		{"ce00010000", "65536"},
		{"d18000", "-32768"},
		{"d280000000", "-2147483648"},
		{"e0", "-32"},
		{"ff", "-1"},
		{"d080", "-128"},
		// float 32 / float 64
		{"ca3fc00000", "1.5"},
		{"ca3dcccccd", "0.10000000149011612"},
		// uint64 / int64 (JS float widening, lossy beyond 2^53)
		{"cf0020000000000000", "9007199254740992"},
		{"cf0020000000000002", "9007199254740994"},
		{"cfffffffffffffffff", "18446744073709552000"},
		{"cfffffffff00000000", "18446744069414584000"},
		{"cf000000e8d4a51000", "1000000000000"},
		{"d3ffe0000000000001", "-9007199254740991"},
		// empty containers, empty string
		{"90", "[]"},
		{"80", "{}"},
		{"a0", `""`},
		// map key ordering and String()-coerced non-string keys
		{"82a13102a13201", `{"1":2,"2":1}`},
		{"82a16101a16102", `{"a":2}`}, // duplicate key: last value wins
		{"810102", `{"1":2}`},
		{"81c201", `{"false":1}`},
		{"81cb3ff8000000000000a178", `{"1.5":"x"}`},
		{"81c001", `{"null":1}`},
		{"81c301", `{"true":1}`},
		{"8192010203", `{"1,2":3}`},
		{"8192c00105", `{",1":5}`},        // array key with a null element
		{"8193c0d400010105", `{",,1":5}`}, // array key with null + undefined elements
		// bin -> Node Buffer
		{"c40105", `{"type":"Buffer","data":[5]}`},
		{"c403010203", `{"type":"Buffer","data":[1,2,3]}`},
		{"c500020102", `{"type":"Buffer","data":[1,2]}`},
		{"c6000000020102", `{"type":"Buffer","data":[1,2]}`},
		// ext (type != special) -> [type, Buffer]; ext type 0 -> ArrayBuffer ({})
		{"d5050102", `[5,{"type":"Buffer","data":[1,2]}]`},
		{"d60501020304", `[5,{"type":"Buffer","data":[1,2,3,4]}]`},
		{"d7050706050403020100", `[5,{"type":"Buffer","data":[7,6,5,4,3,2,1,0]}]`},
		{"d8050102030405060708090a0b0c0d0e0f10", `[5,{"type":"Buffer","data":[1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16]}]`},
		{"c702070102", `[7,{"type":"Buffer","data":[1,2]}]`},
		{"c80002070102", `[7,{"type":"Buffer","data":[1,2]}]`},
		{"c900000002070102", `[7,{"type":"Buffer","data":[1,2]}]`},
		{"d40507", `[5,{"type":"Buffer","data":[7]}]`},
		{"c70300010203", "{}"},
		{"c8000300010203", "{}"},
		// String()-coerced exotic keys (undefined, object, NaN/Infinity floats)
		{"81d4000105", `{"undefined":5}`},
		{"818005", `{"[object Object]":5}`},
		{"81cb7ff800000000000005", `{"NaN":5}`},
		{"81cb7ff000000000000005", `{"Infinity":5}`},
		{"81cbfff000000000000005", `{"-Infinity":5}`},
		// undefined: omitted from objects, rendered null in arrays
		{"91d40001", "[null]"},
		{"82a161d40001a16205", `{"b":5}`},
		// wide-header (non-canonical) length forms
		{"dc00020102", "[1,2]"},
		{"de0001a16101", `{"a":1}`},
		{"df00000001a16101", `{"a":1}`},
		{"da0003616263", `"abc"`},
		{"db00000003616263", `"abc"`},
		// timestamps -> ISO strings
		{"d6ff00000000", `"1970-01-01T00:00:00.000Z"`},
		{"d6ffffffffff", `"2106-02-07T06:28:15.000Z"`},
		{"d7ff0000000051e29bf0", `"2013-07-14T12:39:12.000Z"`},
		{"d70000000000000003e8", `"1970-01-01T00:00:01.000Z"`}, // fixext8 type 0: custom date
		{"c70cff000000000000000051e29bf0", `"2013-07-14T12:39:12.000Z"`},
		// timestamp 96 with a huge/negative second field -> JS extended-year form
		{"c70cff3b9ac9ff000000ff51e29bf0", `"+036719-07-28T06:47:13.000Z"`},
		{"c70cff00000000fffffff000000000", `"-000208-05-13T16:27:44.000Z"`},
		{"c70cff000000000000200000000000", "null"}, // out-of-range date -> null
	}
	for _, c := range vec {
		v, err := msgpackDecode(msgpackBytes(t, c.hex))
		if err != nil {
			t.Fatalf("decode %q: %v", c.hex, err)
		}
		if got := jsStringify(v, 0); got != c.want {
			t.Fatalf("decode %q = %q want %q", c.hex, got, c.want)
		}
	}
}

// TestFromMessagePackPretty checks that From MessagePack emits
// JSON.stringify(value, null, 4), matching CyberChef's JSON dish presentation.
func TestFromMessagePackPretty(t *testing.T) {
	out, err := FromMessagePack{}.Run(abytes(string(msgpackBytes(t, "81a16101"))), nil)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := out.String(), "{\n    \"a\": 1\n}"; got != want {
		t.Fatalf("pretty decode = %q want %q", got, want)
	}
}

// TestFromMessagePackUndefined checks that a top-level undefined (fixext1 type 0)
// yields empty output, matching JSON.stringify(undefined).
func TestFromMessagePackUndefined(t *testing.T) {
	out, err := FromMessagePack{}.Run(abytes(string(msgpackBytes(t, "d40001"))), nil)
	if err != nil {
		t.Fatal(err)
	}
	if out.String() != "" {
		t.Fatalf("undefined output = %q want empty", out.String())
	}
}

// TestFromMessagePackViaRecipe exercises the operation through the engine
// (From Hex -> From MessagePack), covering the JSON dish presentation.
func TestFromMessagePackViaRecipe(t *testing.T) {
	runCases(t, []opCase{
		{"decode int", "0f", "15", core.Recipe{{Op: "From Hex"}, {Op: "From MessagePack"}}},
		{"decode text", "a454657874", `"Text"`, core.Recipe{{Op: "From Hex"}, {Op: "From MessagePack"}}},
	})
}

// TestFromMessagePackMalformed covers truncation, the never-used prefix 0xc1,
// trailing data and other corruption — all of which notepack rejects.
func TestFromMessagePackMalformed(t *testing.T) {
	cases := []string{
		"",                       // empty input
		"c1",                     // never-used prefix
		"cc",                     // uint8 missing its byte
		"ce0102",                 // uint32 truncated
		"cf00",                   // uint64 first word truncated
		"cf00000000",             // uint64 second word missing
		"d300",                   // int64 first word truncated
		"d300000000",             // int64 second word missing
		"d0",                     // int8 missing
		"d100",                   // int16 truncated
		"d2000000",               // int32 truncated
		"cb00000000",             // double truncated
		"ca000000",               // float truncated
		"a5 6162",                // fixstr length exceeds data
		"c4",                     // bin8 length byte missing
		"c405",                   // bin8 data missing
		"c500",                   // bin16 length truncated
		"c600",                   // bin32 length truncated
		"c40105 00",              // trailing byte after bin
		"0f 0f",                  // trailing data after first item
		"81 01",                  // map missing its value
		"91",                     // array element missing
		"dc0005 01",              // array16 short of its length
		"dc00",                   // array16 length truncated
		"dd000000",               // array32 length truncated
		"de0001",                 // map16 missing pair
		"de00",                   // map16 length truncated
		"df000000",               // map32 length truncated
		"d9",                     // str8 length byte missing
		"da00",                   // str16 length truncated
		"db000000",               // str32 length truncated
		"d4",                     // fixext1 type byte missing
		"d605",                   // fixext4 non-timestamp, data missing
		"c7",                     // ext8 length byte missing
		"c8",                     // ext16 length byte missing
		"c900",                   // ext32 length truncated
		"c702",                   // ext8 type byte missing
		"c70207",                 // ext8 data missing
		"c70300",                 // ext8 type-0 ArrayBuffer data missing
		"d400",                   // fixext1 type-0 undefined value byte missing
		"c70cff",                 // timestamp96 ns missing
		"c70cff0000",             // timestamp96 truncated
		"c70cff00000000",         // timestamp96 hi word missing
		"c70cff0000000000000000", // timestamp96 lo word missing
		"d6ff 000000",            // timestamp32 truncated
		"d7ff",                   // timestamp64 hi word missing
		"d7ff 00000000",          // timestamp64 second word missing
		"d700",                   // custom-date hi word missing
		"d700 00000000",          // custom-date second word missing
	}
	for _, h := range cases {
		_, err := FromMessagePack{}.Run(abytes(string(msgpackBytes(t, h))), nil)
		if err == nil {
			t.Fatalf("decode %q: expected error", h)
		}
	}
}

// TestToMessagePackErrors covers invalid JSON input.
func TestToMessagePackErrors(t *testing.T) {
	for _, in := range []string{"", "not json", "{bad}", "[1,2", "1 2"} {
		_, err := ToMessagePack{}.Run(sdish(in), nil)
		if err == nil {
			t.Fatalf("encode %q: expected error", in)
		}
	}
}

// TestToMessagePackEncodeDirect covers the encoder's guard branches that the
// JSON front door cannot reach: an unsupported Go type, and error propagation
// out of arrays and maps.
func TestToMessagePackEncodeDirect(t *testing.T) {
	var buf bytes.Buffer
	bad := []any{
		struct{}{},
		[]any{struct{}{}},
		jsObject{{k: "k", v: struct{}{}}},
	}
	for _, v := range bad {
		if err := msgpackEncode(&buf, v); err == nil {
			t.Fatalf("expected error for %T", v)
		}
	}
}

// TestToMessagePackUndefinedValue checks that a map value of undefined is
// dropped from the encoding (notepack's Object.keys filter), which JSON input
// cannot produce but the encoder guards for. A single undefined-valued key
// therefore encodes as an empty fixmap.
func TestToMessagePackUndefinedValue(t *testing.T) {
	var buf bytes.Buffer
	if err := msgpackEncode(&buf, jsObject{{k: "a", v: jsUndefined{}}}); err != nil {
		t.Fatal(err)
	}
	if got := buf.Bytes(); len(got) != 1 || got[0] != 0x80 {
		t.Fatalf("undefined-valued map = %x want 80", got)
	}
}

// TestJSToUint32Defensive covers jsToUint32's non-finite guard, which the
// integer encode path never reaches (non-finite numbers take the float branch).
func TestJSToUint32Defensive(t *testing.T) {
	for _, f := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		if got := jsToUint32(f); got != 0 {
			t.Fatalf("jsToUint32(%v) = %d want 0", f, got)
		}
	}
}

// TestMessagePackRoundTrip mirrors CyberChef's round-trip behaviour: JSON
// encoded to MessagePack and back is unchanged.
func TestMessagePackRoundTrip(t *testing.T) {
	for _, in := range []string{
		`{"a":1,"b":false,"c":[1,2,3]}`,
		`{"a":{"b":{"c":[1,2.5,"x",null,true]}}}`,
		`[]`,
		`{}`,
		`"hello world"`,
		`1.5`,
		`{"1":10,"2":20,"z":30}`,
	} {
		enc, err := ToMessagePack{}.Run(sdish(in), nil)
		if err != nil {
			t.Fatalf("encode %q: %v", in, err)
		}
		v, err := msgpackDecode(enc.Bytes())
		if err != nil {
			t.Fatalf("decode round-trip %q: %v", in, err)
		}
		if got := jsStringify(v, 0); got != in {
			t.Fatalf("round-trip %q = %q", in, got)
		}
	}
}

// --- direct tests for the type-family helpers extracted from msgpackParsePrefixed ---

// TestMsgpackFloat documents the float32 (0xca) and float64 (0xcb) decoders.
func TestMsgpackFloat(t *testing.T) {
	// 1.5 as big-endian float32 (0x3FC00000) and float64 (0x3FF8...).
	if v, err := msgpackFloat(&mreader{data: []byte{0x3F, 0xC0, 0x00, 0x00}}, 0xca); err != nil || v != 1.5 {
		t.Fatalf("float32: %v, %v", v, err)
	}
	if v, err := msgpackFloat(&mreader{data: []byte{0x3F, 0xF8, 0, 0, 0, 0, 0, 0}}, 0xcb); err != nil || v != 1.5 {
		t.Fatalf("float64: %v, %v", v, err)
	}
}

// TestMsgpackParseSized documents dispatching the length-prefixed str/array/map
// family: str8 (0xd9) with an 8-bit length.
func TestMsgpackParseSized(t *testing.T) {
	v, err := msgpackParseSized(&mreader{data: []byte{0x02, 'h', 'i'}}, 0xd9)
	if err != nil || v != "hi" {
		t.Fatalf("str8: %v, %v", v, err)
	}
}
