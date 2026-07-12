package ops

// Tests for the CBOR Encode / CBOR Decode operations.
//
// CyberChef's CBOR operations are thin wrappers around the `cbor` npm library
// (v10.0.12), calling Cbor.encodeCanonical / Cbor.decodeFirstSync. This is a
// from-scratch port; the transcribed fixtures below come from
// ../CyberChef/tests/operations/tests/CBOR{Encode,Decode}.mjs, and the larger
// vector tables were derived from that exact library used as an oracle (via the
// CyberChef-server /bake endpoint). They are ordinary tests — edit as needed.

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"math"
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// cborEncodeRecipe encodes JSON input to canonical CBOR and renders it as
// space-delimited hex, mirroring CyberChef's fixture recipes.
var cborEncodeRecipe = core.Recipe{{Op: "CBOR Encode"}, {Op: "To Hex", Args: []any{"Space"}}}

// cborBytes decodes a space-delimited hex string into raw bytes.
func cborBytes(t *testing.T, h string) []byte {
	t.Helper()
	b, err := hex.DecodeString(strings.ReplaceAll(h, " ", ""))
	if err != nil {
		t.Fatalf("bad hex %q: %v", h, err)
	}
	return b
}

// TestCBOREncodeFixtures transcribes CyberChef's CBOREncode.mjs cases.
func TestCBOREncodeFixtures(t *testing.T) {
	runCases(t, []opCase{
		{"CBOR Encode: integer", "15", "0f", cborEncodeRecipe},
		{"CBOR Encode: decimal", "1.5", "f9 3e 00", cborEncodeRecipe},
		{"CBOR Encode: text", `"Text"`, "64 54 65 78 74", cborEncodeRecipe},
		{"CBOR Encode: boolean true", "true", "f5", cborEncodeRecipe},
		{"CBOR Encode: boolean false", "false", "f4", cborEncodeRecipe},
		{"CBOR Encode: map", `{"a":1,"b":2,"c":3}`, "a3 61 61 01 61 62 02 61 63 03", cborEncodeRecipe},
		{"CBOR Encode: list", "[0,1,2]", "83 00 01 02", cborEncodeRecipe},
	})
}

// TestCBOREncodeVectors covers integer widths, canonical shortest floats,
// negative-zero, the MAX_SAFE_INTEGER boundary, canonical key ordering, empty
// containers, nesting and unicode — all oracle-verified against cbor 10.0.12.
func TestCBOREncodeVectors(t *testing.T) {
	vec := map[string]string{
		"0":                      "00",
		"-1":                     "20",
		"23":                     "17",
		"24":                     "18 18",
		"255":                    "18 ff",
		"256":                    "19 01 00",
		"65535":                  "19 ff ff",
		"65536":                  "1a 00 01 00 00",
		"4294967295":             "1a ff ff ff ff",
		"4294967296":             "1b 00 00 00 01 00 00 00 00",
		"-24":                    "37",
		"-256":                   "38 ff",
		"-257":                   "39 01 00",
		"1000000000000":          "1b 00 00 00 e8 d4 a5 10 00",
		"9007199254740991":       "1b 00 1f ff ff ff ff ff ff",
		"9007199254740992":       "fa 5a 00 00 00",
		"9007199254740994":       "fb 43 40 00 00 00 00 00 01",
		"-9007199254740991":      "3b 00 1f ff ff ff ff ff fe",
		"-9007199254740992":      "fa da 00 00 00",
		"-9007199254740994":      "fb c3 40 00 00 00 00 00 01",
		"18446744073709551615":   "fa 5f 80 00 00",
		"10000000000000000000":   "fb 43 e1 58 e4 60 91 3d 00",
		"100000000000000000000":  "fb 44 15 af 1d 78 b5 8c 40",
		"-9223372036854775808":   "fa df 00 00 00",
		"1.5":                    "f9 3e 00",
		"2.5":                    "f9 41 00",
		"0.0":                    "00",
		"-0.0":                   "f9 80 00",
		"100000.0":               "1a 00 01 86 a0",
		"65504.0":                "19 ff e0",
		"65505.0":                "19 ff e1",
		"3.4028234663852886e38":  "fa 7f 7f ff ff",
		"1.1":                    "fb 3f f1 99 99 99 99 99 9a",
		"1e300":                  "fb 7e 37 e4 3c 88 00 75 9c",
		"5e-324":                 "fb 00 00 00 00 00 00 00 01",
		"0.000030517578125":      "f9 02 00",
		"5.960464477539063e-8":   "f9 00 01",
		"0.00006103515625":       "f9 04 00",
		"0.00006109476089477539": "f9 04 01",
		`"hello"`:                "65 68 65 6c 6c 6f",
		`""`:                     "60",
		"[]":                     "80",
		"{}":                     "a0",
		"null":                   "f6",
		"true":                   "f5",
		"false":                  "f4",
		"[1,[2,[3]]]":            "82 01 82 02 81 03",
		`{"a":{"b":1}}`:          "a1 61 61 a1 61 62 01",
		`{"aa":1,"b":2,"c":3,"z":4,"aaa":5,"":6}`: "a6 60 06 61 62 02 61 63 03 61 7a 04 62 61 61 01 63 61 61 61 05",
		`"héllo"`: "66 68 c3 a9 6c 6c 6f",
		`"é"`:     "62 c3 a9",
	}
	cases := make([]opCase, 0, len(vec))
	for in, want := range vec {
		cases = append(cases, opCase{"encode " + in, in, want, cborEncodeRecipe})
	}
	runCases(t, cases)
}

// TestCBORDecodeFixtures transcribes CyberChef's CBORDecode.mjs cases. The
// values are compared compact (as with the fixtures' JSON Minify step) via the
// white-box decoder; TestCBORDecodePretty covers the indent-4 wrapping.
func TestCBORDecodeFixtures(t *testing.T) {
	vec := []struct{ hex, want string }{
		{"0f", "15"},
		{"f9 3e 00", "1.5"},
		{"64 54 65 78 74", `"Text"`},
		{"f5", "true"},
		{"f4", "false"},
		{"a3 61 61 01 61 62 02 61 63 03", `{"a":1,"b":2,"c":3}`},
		{"83 00 01 02", "[0,1,2]"},
	}
	for _, c := range vec {
		v, err := cborDecode(cborBytes(t, c.hex))
		if err != nil {
			t.Fatalf("decode %q: %v", c.hex, err)
		}
		if got := jsStringify(v, 0); got != c.want {
			t.Fatalf("decode %q = %q want %q", c.hex, got, c.want)
		}
	}
}

// TestCBORDecodeVectors exercises the full value model: floats (half incl.
// Infinity/NaN/subnormal), byte strings as Node Buffers, order-preserving and
// non-string-keyed maps, indefinite-length containers, the safe-integer edge,
// and tags (unknown tags render as {tag,value}).
func TestCBORDecodeVectors(t *testing.T) {
	vec := []struct{ hex, want string }{
		{"f6", "null"},
		{"f9 00 00", "0"},
		{"f9 80 00", "0"},
		{"f9 7c 00", "null"},
		{"f9 7e 00", "null"},
		{"f9 04 00", "0.00006103515625"},
		{"fb 3f f1 99 99 99 99 99 9a", "1.1"},
		{"fa 7f 7f ff ff", "3.4028234663852886e+38"},
		{"45 68 65 6c 6c 6f", `{"type":"Buffer","data":[104,101,108,108,111]}`},
		{"5f 42 01 02 43 03 04 05 ff", `{"type":"Buffer","data":[1,2,3,4,5]}`},
		{"7f 62 61 62 61 63 ff", `"abc"`},
		{"9f 01 02 03 ff", "[1,2,3]"},
		{"bf 61 61 01 ff", `{"a":1}`},
		{"a2 61 62 01 61 61 02", `{"b":1,"a":2}`},
		{"a2 61 61 01 61 61 02", `{"a":2}`},
		{"a2 61 32 09 61 31 08", `{"1":8,"2":9}`}, // JS enumerates integer keys first

		{"c0 80", "null"},
		{"a1 01 02", "{}"},
		{"a1 01 1b 00 20 00 00 00 00 00 00", "{}"},
		{"1b 00 1f ff ff ff ff ff ff", "9007199254740991"},
		{"3b 00 1f ff ff ff ff ff fe", "-9007199254740991"},
		{"d9 03 e7 01", `{"tag":999,"value":1}`},
		{"d8 7b 63 61 62 63", `{"tag":123,"value":"abc"}`},
		{"80", "[]"},
		{"a0", "{}"},
		{"60", `""`},
		// Date tags: 0 is new Date(value), 1 is new Date(value*1000).
		{"c0 74 32 30 31 33 2d 30 33 2d 32 31 54 32 30 3a 30 34 3a 30 30 5a", `"2013-03-21T20:04:00.000Z"`},
		{"c1 1a 51 4b 67 b0", `"2013-03-21T20:04:00.000Z"`},
		{"c1 fb 41 d4 52 d9 ec 00 00 00", `"2013-03-21T20:04:00.000Z"`},
		{"c0 01", `"1970-01-01T00:00:00.001Z"`},
		{"c0 63 61 62 63", "null"},
		{"c1 63 61 62 63", "null"},
		{"c1 f9 7c 00", "null"},
		{"c1 f9 7e 00", "null"},
	}
	for _, c := range vec {
		v, err := cborDecode(cborBytes(t, c.hex))
		if err != nil {
			t.Fatalf("decode %q: %v", c.hex, err)
		}
		if got := jsStringify(v, 0); got != c.want {
			t.Fatalf("decode %q = %q want %q", c.hex, got, c.want)
		}
	}
}

// TestCBORDecodePretty checks that CBOR Decode emits JSON.stringify(value, null,
// 4) exactly, matching CyberChef's JSON dish presentation.
func TestCBORDecodePretty(t *testing.T) {
	out, err := CBORDecode{}.Run(abytes(string(cborBytes(t, "a1 61 61 01"))), nil)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := out.String(), "{\n    \"a\": 1\n}"; got != want {
		t.Fatalf("pretty decode = %q want %q", got, want)
	}
}

// TestCBORDecodeBigInt covers the "Do not know how to serialize a BigInt" error
// CyberChef produces for integers beyond MAX_SAFE_INTEGER and for bignum tags,
// but only when the value is actually reachable by JSON.stringify.
func TestCBORDecodeBigInt(t *testing.T) {
	bigCases := []string{
		"1b 00 20 00 00 00 00 00 00",
		"1b ff ff ff ff ff ff ff ff",
		"3b 00 1f ff ff ff ff ff ff",
		"c2 41 05",
		"c2 42 01 00",
		"c3 41 05",
		"a1 61 61 1b 00 20 00 00 00 00 00 00",
		"81 1b 00 20 00 00 00 00 00 00",
	}
	for _, h := range bigCases {
		_, err := CBORDecode{}.Run(abytes(string(cborBytes(t, h))), nil)
		if err == nil || !strings.Contains(err.Error(), "Do not know how to serialize a BigInt") {
			t.Fatalf("decode %q: got err %v, want BigInt error", h, err)
		}
	}
}

// TestCBORDecodeMalformed covers truncation, trailing data, reserved/indefinite
// misuse and other corruption — all of which CyberChef rejects.
func TestCBORDecodeMalformed(t *testing.T) {
	cases := []string{
		"",                        // empty input
		"18",                      // one-byte arg truncated
		"19 01",                   // two-byte arg truncated
		"1c",                      // reserved additional info (28)
		"42 01",                   // byte string length exceeds data
		"a1 61 61",                // map missing value
		"00 01 02",                // trailing data after first item
		"ff",                      // lone break
		"81",                      // array element missing
		"82 01",                   // second array element missing
		"9f 01",                   // indefinite array, no break
		"bf",                      // indefinite map, no break, no pair
		"bf 61 61",                // indefinite map, value missing
		"5f 00",                   // indefinite byte string, wrong-major chunk
		"5f 42 01 02",             // indefinite byte string, no break
		"f8 ff",                   // one-byte simple value (unsupported)
		"e0",                      // simple value 0 (unsupported)
		"f9 00",                   // half float truncated
		"fa 00 00 00",             // single float truncated
		"fb 00 00 00 00",          // double float truncated
		"1a 00 00 00",             // four-byte arg truncated
		"1b 00 00 00 00 00 00 00", // eight-byte arg truncated
		"d9 03",                   // tag arg truncated
		"c0",                      // tag with missing inner value
		"39",                      // negative-int arg truncated
		"63 61 62",                // text string length exceeds data
		"78",                      // text string length byte missing
		"98",                      // array length byte missing
		"b8",                      // map length byte missing
		"a1 18",                   // map key truncated
		"9f 18",                   // indefinite array element truncated
		"5f",                      // indefinite byte string, immediate EOF
		"5f 42 01",                // indefinite byte string chunk truncated
		"9f",                      // indefinite array, immediate EOF
		"5f 5f 41 61 ff ff",       // nested indefinite string chunk (invalid)
	}
	for _, h := range cases {
		_, err := CBORDecode{}.Run(abytes(string(cborBytes(t, h))), nil)
		if err == nil {
			t.Fatalf("decode %q: expected error", h)
		}
	}
}

// TestCBORDecodeViaRecipe exercises the operation through the engine (From Hex →
// CBOR Decode), covering Args and the JSON dish presentation.
func TestCBORDecodeViaRecipe(t *testing.T) {
	runCases(t, []opCase{
		{"decode int", "0f", "15", core.Recipe{{Op: "From Hex"}, {Op: "CBOR Decode"}}},
		{"decode text", "64 54 65 78 74", `"Text"`, core.Recipe{{Op: "From Hex"}, {Op: "CBOR Decode"}}},
	})
}

// TestCBORHalfPrecisionLoss covers cborHalf's denormal precision-loss branch: a
// float32 in the half-subnormal exponent range whose mantissa cannot be
// represented in half precision is rejected (encoded as a single instead).
func TestCBORHalfPrecisionLoss(t *testing.T) {
	f := float64(math.Float32frombits(0x38002000))
	if _, ok := cborHalf(f); ok {
		t.Fatalf("cborHalf(%v) unexpectedly succeeded", f)
	}
}

// TestCBORDecodeUndefined checks that a top-level CBOR undefined (0xf7) yields
// empty output, matching JSON.stringify(undefined).
func TestCBORDecodeUndefined(t *testing.T) {
	out, err := CBORDecode{}.Run(abytes("\xf7"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if out.String() != "" {
		t.Fatalf("undefined output = %q want empty", out.String())
	}
}

// TestCBOREncodeValueDirect covers the encoder's guard branches that the JSON
// front door cannot reach: an unsupported Go type, an unparseable number, and
// error propagation out of arrays and maps.
func TestCBOREncodeValueDirect(t *testing.T) {
	var buf bytes.Buffer
	bad := []any{
		struct{}{},
		json.Number("not-a-number"),
		[]any{struct{}{}},
		map[string]any{"k": struct{}{}},
	}
	for _, v := range bad {
		if err := cborEncodeValue(&buf, v); err == nil {
			t.Fatalf("expected error for %T", v)
		}
	}
}

// TestCBOREncodeErrors covers invalid JSON input.
func TestCBOREncodeErrors(t *testing.T) {
	for _, in := range []string{"", "not json", "{bad}", "[1,2", "1 2"} {
		_, err := CBOREncode{}.Run(sdish(in), nil)
		if err == nil {
			t.Fatalf("encode %q: expected error", in)
		}
	}
}

// TestCBORRoundTrip mirrors CyberChef's round-trip fixture: JSON encoded to CBOR
// and back is unchanged.
func TestCBORRoundTrip(t *testing.T) {
	for _, in := range []string{
		`{"a":1,"b":false,"c":[1,2,3]}`,
		`{"a":{"b":{"c":[1,2.5,"x",null,true]}}}`,
		`[]`,
		`{}`,
		`"hello world"`,
		`1.5`,
	} {
		enc, err := CBOREncode{}.Run(sdish(in), nil)
		if err != nil {
			t.Fatalf("encode %q: %v", in, err)
		}
		v, err := cborDecode(enc.Bytes())
		if err != nil {
			t.Fatalf("decode round-trip %q: %v", in, err)
		}
		if got := jsStringify(v, 0); got != in {
			t.Fatalf("round-trip %q = %q", in, got)
		}
	}
}
