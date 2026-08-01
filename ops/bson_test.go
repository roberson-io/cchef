package ops

import (
	"math"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// bsonSerRecipe serialises then hex-encodes the bytes for readable comparison.
func bsonSerRecipe() core.Recipe {
	return core.Recipe{
		{Op: "BSON serialise", Args: []any{}},
		{Op: "To Hex", Args: []any{"None"}},
	}
}

// bsonDeserRecipe hex-decodes the input bytes then deserialises them.
func bsonDeserRecipe() core.Recipe {
	return core.Recipe{
		{Op: "From Hex", Args: []any{"None"}},
		{Op: "BSON deserialise", Args: []any{}},
	}
}

// Fixtures transcribed from ../CyberChef/tests/operations/tests/BSON.mjs (basic
// case) plus oracle-verified vectors covering the number-type rule (int32 vs
// double), booleans, null, arrays, nested documents and ECMAScript key ordering.
// CyberChef wraps js-bson's serialize(); cchef reimplements the codec from
// scratch (ops/bson.go), so these are byte-for-byte.
func TestBSONSerialiseFixtures(t *testing.T) {
	runCases(t, []opCase{
		{"BSON serialise: nothing", "", "", bsonSerRecipe()},
		{
			"BSON serialise: basic", `{"hello":"world"}`,
			"160000000268656c6c6f0006000000776f726c640000", bsonSerRecipe(),
		},
		{"int32", `{"n":5}`, "0c000000106e000500000000", bsonSerRecipe()},
		{"negative int32", `{"n":-1}`, "0c000000106e00ffffffff00", bsonSerRecipe()},
		{"integer beyond int32 -> double", `{"n":2147483648}`, "10000000016e00000000000000e04100", bsonSerRecipe()},
		{"int32 max stays int32", `{"n":2147483647}`, "0c000000106e00ffffff7f00", bsonSerRecipe()},
		{"double", `{"n":3.14}`, "10000000016e001f85eb51b81e094000", bsonSerRecipe()},
		{"negative zero stays double", `{"g":-0.0}`, "10000000016700000000000000008000", bsonSerRecipe()},
		{"boolean true", `{"n":true}`, "09000000086e000100", bsonSerRecipe()},
		{"boolean false", `{"n":false}`, "09000000086e000000", bsonSerRecipe()},
		{"null value", `{"n":null}`, "080000000a6e0000", bsonSerRecipe()},
		{"array", `{"a":[1,2]}`, "1b0000000461001300000010300001000000103100020000000000", bsonSerRecipe()},
		{"nested document", `{"o":{"x":1}}`, "14000000036f000c000000107800010000000000", bsonSerRecipe()},
		{"ES key ordering (integer keys first)", `{"b":1,"1":2,"a":3}`, "1a00000010310002000000106200010000001061000300000000", bsonSerRecipe()},
		{"null root -> empty document", "null", "0500000000", bsonSerRecipe()},
	})
}

// TestBSONSerialiseErrors covers js-bson's root-input restrictions (verbatim
// error text) and invalid JSON.
func TestBSONSerialiseErrors(t *testing.T) {
	cases := map[string]string{
		"[10,20]": "BSONError: serialize does not support an array as the root input",
		"5":       "BSONError: serialize does not support non-object as the root input",
		`"hi"`:    "BSONError: serialize does not support non-object as the root input",
		"true":    "BSONError: serialize does not support non-object as the root input",
	}
	for in, want := range cases {
		_, err := runOp(t, "BSON serialise", in)
		if err == nil {
			t.Errorf("expected error for %q", in)
		} else if err.Error() != want {
			t.Errorf("input %q: got error %q, want %q", in, err.Error(), want)
		}
	}
	if _, err := runOp(t, "BSON serialise", "{bad}"); err == nil {
		t.Errorf("expected error for invalid JSON")
	}
}

// Fixtures transcribed from BSON.mjs (basic case) plus oracle-verified vectors for
// the richer BSON element types, each rendered as js-bson's
// JSON.stringify(_, null, 2) does: ObjectId as a hex string, UTC datetime as an
// ISO string, Binary as base64, Timestamp as {"$timestamp": "..."}, and
// RegExp/MinKey/MaxKey as an empty object.
func TestBSONDeserialiseFixtures(t *testing.T) {
	runCases(t, []opCase{
		{"BSON deserialise: nothing", "", "", bsonDeserRecipe()},
		{
			"BSON deserialise: basic", "160000000268656c6c6f0006000000776f726c640000",
			"{\n  \"hello\": \"world\"\n}", bsonDeserRecipe(),
		},
		{"int32", "0c000000106e000500000000", "{\n  \"n\": 5\n}", bsonDeserRecipe()},
		{"double", "10000000016e001f85eb51b81e094000", "{\n  \"n\": 3.14\n}", bsonDeserRecipe()},
		{"boolean", "09000000086e000100", "{\n  \"n\": true\n}", bsonDeserRecipe()},
		{"null", "080000000a6e0000", "{\n  \"n\": null\n}", bsonDeserRecipe()},
		{"array", "1b0000000461001300000010300001000000103100020000000000", "{\n  \"a\": [\n    1,\n    2\n  ]\n}", bsonDeserRecipe()},
		{"int64", "1000000012610000f2052a0100000000", "{\n  \"a\": 5000000000\n}", bsonDeserRecipe()},
		{"ObjectId -> hex string", "1500000007696400507f1f77bcf86cd79943901100", "{\n  \"id\": \"507f1f77bcf86cd799439011\"\n}", bsonDeserRecipe()},
		{"UTC datetime -> ISO string", "1000000009640000e8665e6f01000000", "{\n  \"d\": \"2020-01-01T00:00:00.000Z\"\n}", bsonDeserRecipe()},
		{"embedded document", "14000000036f000c000000107800010000000000", "{\n  \"o\": {\n    \"x\": 1\n  }\n}", bsonDeserRecipe()},
		{"Binary -> base64", "10000000056200030000000061626300", "{\n  \"b\": \"YWJj\"\n}", bsonDeserRecipe()},
		{"Timestamp", "1000000011740015cd5b070000000000", "{\n  \"t\": {\n    \"$timestamp\": \"123456789\"\n  }\n}", bsonDeserRecipe()},
		{"RegExp -> empty object", "0d0000000b7200616200690000", "{\n  \"r\": {}\n}", bsonDeserRecipe()},
		{"MinKey -> empty object", "08000000ff6d0000", "{\n  \"m\": {}\n}", bsonDeserRecipe()},
		{"MaxKey -> empty object", "080000007f6d0000", "{\n  \"m\": {}\n}", bsonDeserRecipe()},
	})
}

// TestBSONDeserialiseErrors covers unsupported element types (Decimal128) and
// malformed input (bad lengths, truncation, missing terminator).
func TestBSONDeserialiseErrors(t *testing.T) {
	bad := []string{
		"180000001364000000000000000000000000000000000000", // Decimal128 (0x13) unsupported
		"05000000",           // truncated (missing terminator)
		"04000000",           // length below minimum
		"ff000000",           // length beyond buffer
		"0c000000106e00",     // element value truncated
		"0c000000106e0005",   // int32 value truncated
		"09000000026e00",     // string length truncated
		"08000000086e00",     // boolean value truncated
		"0a0000000b6e006100", // regex flags cstring unterminated
	}
	for _, hexIn := range bad {
		out, err := core.Recipe{
			{Op: "From Hex", Args: []any{"None"}},
			{Op: "BSON deserialise", Args: []any{}},
		}.Execute(sdish(hexIn))
		if err == nil {
			t.Errorf("expected error for %q, got %q", hexIn, out.String())
		}
	}
	// A document with trailing bytes after the terminator is rejected.
	if _, err := bsonDeserialise([]byte{0x05, 0x00, 0x00, 0x00, 0x00, 0x99}); err == nil {
		t.Errorf("expected error for trailing bytes")
	}
}

// TestBSONCodecDirect exercises the encoder/decoder helpers directly for the
// int64 encode branch (unreachable from JSON input, which is always float64) and
// a NaN double round-trip.
func TestBSONCodecDirect(t *testing.T) {
	// int64 in int32 range encodes as int32; out of range as int64.
	typ, _ := bsonEncodeValue(int64(7))
	if typ != bsonTypeInt32 {
		t.Errorf("int64(7) -> type 0x%02x, want int32", typ)
	}
	typ, _ = bsonEncodeValue(int64(5000000000))
	if typ != bsonTypeInt64 {
		t.Errorf("int64(5e9) -> type 0x%02x, want int64", typ)
	}
	// NaN encodes as a double and round-trips (js-bson stores NaN as a double).
	doc := bsonEncodeDoc(jsObject{{k: "x", v: math.NaN()}})
	back, err := bsonDeserialise(doc)
	if err != nil {
		t.Fatalf("deserialise NaN: %v", err)
	}
	if _, ok := back[0].v.(float64); !ok {
		t.Errorf("NaN did not round-trip to a float64: %T", back[0].v)
	}
	// A value type the JSON path never produces encodes as null.
	if typ, _ := bsonEncodeValue(jsUndefined{}); typ != bsonTypeNull {
		t.Errorf("jsUndefined -> type 0x%02x, want null", typ)
	}
}

// TestBSONReaderBounds drives each decoder helper's bounds/format error branch
// directly with a too-short or malformed buffer.
func TestBSONReaderBounds(t *testing.T) {
	short := func(name string, err error) {
		if err == nil {
			t.Errorf("%s: expected error on short buffer", name)
		}
	}
	_, e := (&bsonReader{b: []byte{1, 2, 3}}).readInt32()
	short("readInt32", e)
	_, e = (&bsonReader{b: []byte{1, 2, 3}}).readUint64()
	short("readUint64", e)
	_, e = (&bsonReader{b: []byte{}}).readBool()
	short("readBool", e)
	_, e = (&bsonReader{b: []byte{1, 2, 3}}).readObjectID()
	short("readObjectID", e)
	_, e = (&bsonReader{b: []byte{1, 2, 3}}).readDate()
	short("readDate", e)
	_, e = (&bsonReader{b: []byte{1, 2, 3}}).readTimestamp()
	short("readTimestamp", e)
	_, e = (&bsonReader{b: []byte{1, 2}}).readBinary() // too short for length prefix
	short("readBinary length prefix", e)
	_, e = (&bsonReader{b: []byte{0x05, 0, 0, 0}}).readBinary() // length ok, data overruns
	short("readBinary overrun", e)
	_, e = (&bsonReader{b: []byte{0xff, 0xff, 0xff, 0xff}}).readBinary() // negative length
	short("readBinary negative", e)
	_, e = (&bsonReader{b: []byte{1, 2}}).readString() // too short for length prefix
	short("readString length prefix", e)
	_, e = (&bsonReader{b: []byte{0x10, 0, 0, 0}}).readString() // length ok, body overruns
	short("readString overrun", e)
	_, e = (&bsonReader{b: []byte{0, 0, 0, 0}}).readString() // length < 1
	short("readString len<1", e)
	_, e = (&bsonReader{b: []byte{'a', 'b'}}).readCString()
	short("readCString", e)
	_, e = (&bsonReader{b: []byte{'a', 'b'}}).readRegex() // unterminated pattern
	short("readRegex pattern", e)
	_, e = (&bsonReader{b: []byte{'a', 0, 'i'}}).readRegex() // unterminated flags
	short("readRegex flags", e)
	_, e = (&bsonReader{b: []byte{0, 0}}).readArray() // bad inner document
	short("readArray", e)
	// readDocument: too short for the length prefix.
	_, e = bsonDeserialise([]byte{1, 2})
	short("readDocument length", e)
	// readDocument: valid length, element key cstring unterminated.
	_, e = bsonDeserialise([]byte{0x07, 0, 0, 0, 0x10, 'a', 'b'})
	short("readDocument key", e)
	// readDocument: missing terminator (non-zero byte where the terminator belongs).
	_, e = bsonDeserialise([]byte{0x05, 0, 0, 0, 0x01})
	short("readDocument terminator", e)
	// readDocument: an element value overruns the declared document length, so the
	// terminator is found past `end` (length mismatch).
	_, e = bsonDeserialise([]byte{0x08, 0, 0, 0, 0x10, 0x00, 0xAA, 0xBB, 0xCC, 0xDD, 0x00})
	short("readDocument length mismatch", e)
}
