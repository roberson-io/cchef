package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// Protobuf Encode verified against the CyberChef-server oracle. The result is
// bracketed with To Hex for a readable comparison.
func TestProtobufEncode(t *testing.T) {
	toHex := core.RecipeOp{Op: "To Hex", Args: []any{"None"}}
	schema := `syntax="proto3"; message Test { int32 a = 1; string b = 2; repeated int32 c = 3; Sub sub = 5; enum E {X=0;Y=1;} E e = 6; } message Sub { int32 x = 1; }`
	schemaC := `syntax="proto3"; message M { int32 a=1; int64 big=2; bool flag=3; }`
	runCases(t, []opCase{
		{
			"encode: scalars + repeated", `{"a":150,"b":"hi","c":[1,2]}`, "0896011202686918011802",
			core.Recipe{{Op: "Protobuf Encode", Args: []any{schema}}, toHex},
		},
		{
			"encode: nested + enum name", `{"a":150,"sub":{"x":7},"e":"Y"}`, "0896012a0208073001",
			core.Recipe{{Op: "Protobuf Encode", Args: []any{schema}}, toHex},
		},
		{
			"encode: empty object", `{}`, "",
			core.Recipe{{Op: "Protobuf Encode", Args: []any{schema}}, toHex},
		},
		{
			// Every scalar wire kind, including negative sint/sfixed and bytes.
			"encode: all scalar kinds",
			`{"bo":true,"u32":5,"u64":6,"f32":7,"f64":8,"fl":1.5,"d":2.5,"by":"AAE=","s32":-1,"sf32":-2}`,
			"0801100518062507000000290800000000000000350000c03f39000000000000044042020001480155feffffff",
			core.Recipe{{Op: "Protobuf Encode", Args: []any{`syntax="proto3"; message All { bool bo=1; uint32 u32=2; uint64 u64=3; fixed32 f32=4; fixed64 f64=5; float fl=6; double d=7; bytes by=8; sint32 s32=9; sfixed32 sf32=10; }`}}, toHex},
		},
		{
			"encode: map field", `{"m":{"key":5}}`, "0a070a036b65791005",
			core.Recipe{{Op: "Protobuf Encode", Args: []any{`syntax="proto3"; message M { map<string,int32> m = 1; }`}}, toHex},
		},
		{
			"encode: repeated messages", `{"subs":[{"x":5},{"x":7}]}`, "0a0208050a020807",
			core.Recipe{{Op: "Protobuf Encode", Args: []any{`syntax="proto3"; message M { repeated Sub subs = 1; } message Sub { int32 x = 1; }`}}, toHex},
		},
		{
			"encode: sint64 and sfixed64", `{"a":-5,"b":-3}`, "080911fdffffffffffffff",
			core.Recipe{{Op: "Protobuf Encode", Args: []any{`syntax="proto3"; message M { sint64 a=1; sfixed64 b=2; }`}}, toHex},
		},
		// protobufjs coercion: numeric strings (incl. int64/uint64 JSON strings),
		// non-numeric strings -> 0, floats truncate, string -> bool.
		{
			"encode: int32 from string", `{"a":"150"}`, "089601",
			core.Recipe{{Op: "Protobuf Encode", Args: []any{schemaC}}, toHex},
		},
		{
			"encode: int64 from string", `{"big":"300"}`, "10ac02",
			core.Recipe{{Op: "Protobuf Encode", Args: []any{schemaC}}, toHex},
		},
		{
			"encode: bool from string", `{"flag":"true"}`, "1801",
			core.Recipe{{Op: "Protobuf Encode", Args: []any{schemaC}}, toHex},
		},
		{
			"encode: non-numeric string is zero", `{"a":"xyz"}`, "0800",
			core.Recipe{{Op: "Protobuf Encode", Args: []any{schemaC}}, toHex},
		},
		{
			"encode: float truncates", `{"a":150.7}`, "089601",
			core.Recipe{{Op: "Protobuf Encode", Args: []any{schemaC}}, toHex},
		},
		{
			"encode: unknown input field ignored", `{"a":5,"zzz":9}`, "0805",
			core.Recipe{{Op: "Protobuf Encode", Args: []any{`syntax="proto3"; message M { int32 a=1; }`}}, toHex},
		},
		{
			"encode: bool coerced to number", `{"a":true}`, "0801",
			core.Recipe{{Op: "Protobuf Encode", Args: []any{`syntax="proto3"; message M { int32 a=1; }`}}, toHex},
		},
		{
			"encode: number coerced to bool", `{"flag":1}`, "0801",
			core.Recipe{{Op: "Protobuf Encode", Args: []any{`syntax="proto3"; message M { bool flag=1; }`}}, toHex},
		},
		{
			"encode: enum by number", `{"e":1}`, "0801",
			core.Recipe{{Op: "Protobuf Encode", Args: []any{`syntax="proto3"; message M { enum E{X=0;Y=1;} E e=1; }`}}, toHex},
		},
		{
			// A non-object input yields empty output (protobufjs leniency).
			"encode: non-object input is empty", `[1,2]`, "",
			core.Recipe{{Op: "Protobuf Encode", Args: []any{`syntax="proto3"; message M { int32 a=1; }`}}, toHex},
		},
		{
			// A number for a string field is stringified.
			"encode: string from number", `{"s":5}`, "0a0135",
			core.Recipe{{Op: "Protobuf Encode", Args: []any{`syntax="proto3"; message M { string s=1; }`}}, toHex},
		},
	})

	// A schema with no message defined errors.
	if _, err := runOp(t, "Protobuf Encode", `{"a":1}`, `syntax="proto3";`); err == nil {
		t.Error("no message schema: expected error")
	}
	// A syntactically invalid schema errors.
	if _, err := runOp(t, "Protobuf Encode", `{"a":1}`, "not a valid proto"); err == nil {
		t.Error("invalid schema: expected error")
	}
	// Malformed JSON input errors.
	if _, err := runOp(t, "Protobuf Encode", `{not valid json`, `syntax="proto3"; message M { int32 a=1; }`); err == nil {
		t.Error("invalid JSON: expected error")
	}
	// Invalid base64 for a bytes field errors.
	if _, err := runOp(t, "Protobuf Encode", `{"b":"!!!"}`, `syntax="proto3"; message M { bytes b=1; }`); err == nil {
		t.Error("invalid base64: expected error")
	}
	// A non-array value for a repeated field errors (as protobufjs does).
	if _, err := runOp(t, "Protobuf Encode", `{"c":5}`, `syntax="proto3"; message M { repeated int32 c=1; }`); err == nil {
		t.Error("repeated as scalar: expected error")
	}
	// A non-object value for a map field errors.
	if _, err := runOp(t, "Protobuf Encode", `{"m":5}`, `syntax="proto3"; message M { map<string,int32> m=1; }`); err == nil {
		t.Error("map as scalar: expected error")
	}
}
