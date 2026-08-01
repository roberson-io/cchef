package jsonval

import (
	"math"
	"testing"
)

// TestJSFormatNumber covers the special-number branches directly.
func TestJSFormatNumber(t *testing.T) {
	cases := map[float64]string{
		0:                     "0",
		math.Copysign(0, -1):  "0",
		1.5:                   "1.5",
		math.NaN():            "null",
		math.Inf(1):           "null",
		math.Inf(-1):          "null",
		100000002004087730000: "100000002004087730000",
	}
	for f, want := range cases {
		if got := FormatNumber(f); got != want {
			t.Errorf("FormatNumber(%v) = %q want %q", f, got, want)
		}
	}
}

// TestJSStringifyScalars covers the boolean-false and compact-object branches of
// the serialiser.
func TestJSStringifyScalars(t *testing.T) {
	if got := Stringify(Object{{K: "b", V: false}, {K: "t", V: true}}, 0); got != `{"b":false,"t":true}` {
		t.Fatalf("compact bools: %q", got)
	}
	if got := Stringify(false, 0); got != "false" {
		t.Fatalf("bare false: %q", got)
	}
}

// TestJSStringifyUndefined checks that JavaScript's undefined is omitted from
// objects and rendered as null inside arrays, in both compact and pretty modes.
func TestJSStringifyUndefined(t *testing.T) {
	obj := Object{{K: "a", V: int64(1)}, {K: "b", V: Undefined{}}, {K: "c", V: int64(3)}}
	if got := Stringify(obj, 0); got != `{"a":1,"c":3}` {
		t.Fatalf("compact object with undefined: %q", got)
	}
	if got := Stringify(Object{{K: "a", V: Undefined{}}}, 0); got != "{}" {
		t.Fatalf("object of only undefined: %q", got)
	}
	if got := Stringify(Object{{K: "a", V: Undefined{}}}, 4); got != "{}" {
		t.Fatalf("pretty object of only undefined: %q", got)
	}
	arr := []any{int64(1), Undefined{}, int64(2)}
	if got := Stringify(arr, 0); got != "[1,null,2]" {
		t.Fatalf("compact array with undefined: %q", got)
	}
	if got := Stringify(obj, 4); got != "{\n    \"a\": 1,\n    \"c\": 3\n}" {
		t.Fatalf("pretty object with undefined: %q", got)
	}
}

// TestJSStringifyIndentUnit checks the arbitrary indent-unit variant used by JSON
// Beautify: a tab, an empty unit (compact), and a non-whitespace unit all nest
// correctly, matching JSON.stringify(value, null, unit).
func TestJSStringifyIndentUnit(t *testing.T) {
	nested := Object{{K: "a", V: []any{int64(1), Object{{K: "b", V: int64(2)}}}}}
	cases := []struct {
		unit, want string
	}{
		{"\t", "{\n\t\"a\": [\n\t\t1,\n\t\t{\n\t\t\t\"b\": 2\n\t\t}\n\t]\n}"},
		{"", `{"a":[1,{"b":2}]}`},
		{"--", "{\n--\"a\": [\n----1,\n----{\n------\"b\": 2\n----}\n--]\n}"},
	}
	for _, c := range cases {
		if got := StringifyIndent(nested, c.unit); got != c.want {
			t.Errorf("StringifyIndent(_, %q) = %q want %q", c.unit, got, c.want)
		}
	}
	// Stringify(v, n) must stay equivalent to an n-space unit.
	if a, b := Stringify(nested, 4), StringifyIndent(nested, "    "); a != b {
		t.Errorf("Stringify(_,4)=%q != StringifyIndent(_,\"    \")=%q", a, b)
	}
}
