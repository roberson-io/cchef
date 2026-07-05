package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// Protobuf Decode (schema-less) verified against the CyberChef-server oracle.
func TestProtobufDecodeRaw(t *testing.T) {
	fromHex := core.RecipeOp{Op: "From Hex", Args: []any{"None"}}
	runCases(t, []opCase{
		{
			// field1 varint 150, field2 repeated string, field3 nested message.
			"raw: mixed", "089601120774657374696e671a02082a1205616761696e",
			`{"1":150,"2":["testing","again"],"3":{"1":42}}`,
			core.Recipe{fromHex, {Op: "Protobuf Decode", Args: []any{"", false, false}}},
		},
		{
			// field1 fixed32 30, field2 fixed64 1045220557.
			"raw: fixed widths", "0d1e00000011cdcc4c3e00000000",
			`{"1":30,"2":1045220557}`,
			core.Recipe{fromHex, {Op: "Protobuf Decode", Args: []any{"", false, false}}},
		},
		{
			// field2 length-delimited non-message bytes -> latin1 string.
			"raw: bytes to string", "1203ffeeaa", "{\"2\":\"ÿîª\"}",
			core.Recipe{fromHex, {Op: "Protobuf Decode", Args: []any{"", false, false}}},
		},
		{
			"raw: show types", "089601120774657374696e671a02082a1205616761696e",
			`{"field #1: VarInt (e.g. int32, bool)":150,"field #2: L-delim (e.g. string, message)":["testing","again"],"field #3: L-delim (e.g. string, message)":{"field #1: VarInt (e.g. int32, bool)":42}}`,
			core.Recipe{fromHex, {Op: "Protobuf Decode", Args: []any{"", false, true}}},
		},
		{
			"raw: show types fixed", "0d1e00000011cdcc4c3e00000000",
			`{"field #1: 32-Bit (e.g. fixed32, float)":30,"field #2: 64-Bit (e.g. fixed64, double)":1045220557}`,
			core.Recipe{fromHex, {Op: "Protobuf Decode", Args: []any{"", false, true}}},
		},
		{
			// Field number > 15 uses a multi-byte tag.
			"raw: multi-byte field number", "900107", `{"18":7}`,
			core.Recipe{fromHex, {Op: "Protobuf Decode", Args: []any{"", false, false}}},
		},
		{
			// A varint of 2^30 exercises the shift>=28 path.
			"raw: large varint", "088080808004", `{"1":1073741824}`,
			core.Recipe{fromHex, {Op: "Protobuf Decode", Args: []any{"", false, false}}},
		},
		{
			// Same field number repeated with submessages -> array of objects.
			"raw: repeated submessages", "0a0208050a020806", `{"1":[{"1":5},{"1":6}]}`,
			core.Recipe{fromHex, {Op: "Protobuf Decode", Args: []any{"", false, false}}},
		},
		{
			// Three repeats of a field exercise appending to an existing array.
			"raw: thrice-repeated scalar", "080108020803", `{"1":[1,2,3]}`,
			core.Recipe{fromHex, {Op: "Protobuf Decode", Args: []any{"", false, false}}},
		},
		{
			"raw: repeated submessages types", "0a0208050a020806",
			`{"field #1: L-delim (e.g. string, message)":[{"field #1: VarInt (e.g. int32, bool)":5},{"field #1: VarInt (e.g. int32, bool)":6}]}`,
			core.Recipe{fromHex, {Op: "Protobuf Decode", Args: []any{"", false, true}}},
		},
	})
}

// Protobuf Decode (schema-based) verified against the CyberChef-server oracle.
func TestProtobufDecodeSchema(t *testing.T) {
	fromHex := core.RecipeOp{Op: "From Hex", Args: []any{"None"}}
	schema := `syntax = "proto3"; message Test { int32 a = 1; string b = 2; repeated int32 c = 3; bool flag = 4; Sub sub = 5; enum E { X = 0; Y = 1; } E e = 6; } message Sub { int32 x = 1; }`
	schema2 := `syntax="proto3"; message M { bytes data = 1; int64 big = 2; uint64 ubig = 3; double d = 4; }`
	runCases(t, []opCase{
		{
			// Fields reordered (repeated first), defaults included, enum -> name.
			"schema: basic", "0896011202686918011802",
			`{"c":[1,2],"a":150,"b":"hi","flag":false,"sub":null,"e":"X"}`,
			core.Recipe{fromHex, {Op: "Protobuf Decode", Args: []any{schema, false, false}}},
		},
		{
			// Set submessage and enum value.
			"schema: set message+enum", "0896012a0208073001",
			`{"c":[],"a":150,"b":"","flag":false,"sub":{"x":7},"e":"Y"}`,
			core.Recipe{fromHex, {Op: "Protobuf Decode", Args: []any{schema, false, false}}},
		},
		{
			"schema: show types", "0896011202686918011802",
			`{"c (int32)":[1,2],"a (int32)":150,"b (string)":"hi","flag (bool)":false,"sub (Sub)":null,"e (E)":"X"}`,
			core.Recipe{fromHex, {Op: "Protobuf Decode", Args: []any{schema, false, true}}},
		},
		{
			// Field 9 is not in the schema -> Unknown Fields.
			"schema: unknown field", "0896014805",
			`{"Test":{"c":[],"a":150,"b":"","flag":false,"sub":null,"e":"X"},"Unknown Fields":{"9":5}}`,
			core.Recipe{fromHex, {Op: "Protobuf Decode", Args: []any{schema, true, false}}},
		},
		{
			// bytes -> base64 (toObject bytes:String); int64/uint64 -> number.
			"schema: bytes and longs", "0a02ffee10ac0218ac0221000000000000f83f",
			`{"data":"/+4=","big":300,"ubig":300,"d":1.5}`,
			core.Recipe{fromHex, {Op: "Protobuf Decode", Args: []any{schema2, false, false}}},
		},
		{
			"schema: map field", "0a070a036b65791005", `{"m":{"key":5}}`,
			core.Recipe{fromHex, {Op: "Protobuf Decode", Args: []any{`syntax="proto3"; message M { map<string,int32> m = 1; }`, false, false}}},
		},
		{
			"schema: repeated messages", "0a0208050a020807", `{"subs":[{"x":5},{"x":7}]}`,
			core.Recipe{fromHex, {Op: "Protobuf Decode", Args: []any{`syntax="proto3"; message M { repeated Sub subs = 1; } message Sub { int32 x = 1; }`, false, false}}},
		},
		{
			// float -> number; enum value not in the schema stays numeric.
			"schema: float and unknown enum", "0d0000c03f1005", `{"f":1.5,"e":5}`,
			core.Recipe{fromHex, {Op: "Protobuf Decode", Args: []any{`syntax="proto3"; message M { float f = 1; enum E {A=0;B=1;} E e = 2; }`, false, false}}},
		},
		{
			// A singular schema field seen repeated on the wire is flagged.
			"schema: repeated-field mismatch", "08050806",
			`{"M":{"a":6,"s":null},"Unknown Fields":{"(M) a is a repeated field":[5,6]}}`,
			core.Recipe{fromHex, {Op: "Protobuf Decode", Args: []any{`syntax="proto3"; message M { int32 a = 1; Sub s = 2; } message Sub { int32 x = 1; }`, true, false}}},
		},
		{
			// A submessage carrying a field absent from its schema is flagged.
			"schema: submessage missing fields", "120408074803",
			`{"M":{"s":{"x":7}},"Unknown Fields":{"s (Sub) has missing fields":{"9":3}}}`,
			core.Recipe{fromHex, {Op: "Protobuf Decode", Args: []any{`syntax="proto3"; message M { Sub s = 2; } message Sub { int32 x = 1; }`, true, false}}},
		},
		{
			// A valid schema with no message falls back to the raw decode.
			"schema: no message defined", "0805", `{"1":5}`,
			core.Recipe{fromHex, {Op: "Protobuf Decode", Args: []any{`syntax="proto3";`, false, false}}},
		},
	})

	// A syntactically invalid schema errors.
	if _, err := runOp(t, "Protobuf Decode", "x", "not a valid proto", false, false); err == nil {
		t.Error("invalid schema: expected error")
	}
}
