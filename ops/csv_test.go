package ops

// Tests for the CSV to JSON and JSON to CSV operations.
//
// Fixtures are transcribed from CyberChef's tests/operations/tests/CSV.mjs and
// JSONtoCSV.mjs; the edge-case tables were derived from CyberChef (via the
// CyberChef-server oracle). CSV to JSON is a port of Utils.parseCSV plus a
// dictionary/array shaping step; JSON to CSV reproduces CyberChef's direct path
// and its flatten-based fallback (the `flat` npm library). These are ordinary
// tests — edit as needed.

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/jsonval"
)

// exampleCSV is the EXAMPLE_CSV fixture from CSV.mjs: a header row, a plain row,
// a row of quoted delimiters, and a row exercising doubled quotes and an
// embedded CRLF inside a quoted cell.
const exampleCSV = "A,B,C,D,E,F\r\n" +
	"1,2,3,4,5,6\r\n" +
	"\",\",;,',\"\"\"\",,\r\n" +
	"\"\"\"hello\"\"\",\"a\"\"1\",\"multi\r\nline\",,,end\r\n"

// csvValue parses CSV with the default delimiters and shapes it, returning the
// compact JSON of the result.
func csvValue(t *testing.T, input string, dict bool) string {
	t.Helper()
	rows := parseCSV(input, []rune{','}, []rune{'\r', '\n'})
	return jsonval.Stringify(csvRowsToValue(rows, dict), 0)
}

// TestCSVToJSONExample checks the EXAMPLE_CSV fixture in both output shapes.
func TestCSVToJSONExample(t *testing.T) {
	wantDict := `[{"A":"1","B":"2","C":"3","D":"4","E":"5","F":"6"},` +
		`{"A":",","B":";","C":"'","D":"\"","E":"","F":""},` +
		`{"A":"\"hello\"","B":"a\"1","C":"multi\r\nline","D":"","E":"","F":"end"}]`
	if got := csvValue(t, exampleCSV, true); got != wantDict {
		t.Fatalf("dict:\n got %s\nwant %s", got, wantDict)
	}
	wantArrays := `[["A","B","C","D","E","F"],["1","2","3","4","5","6"],` +
		`[",",";","'","\"","",""],["\"hello\"","a\"1","multi\r\nline","","","end"]]`
	if got := csvValue(t, exampleCSV, false); got != wantArrays {
		t.Fatalf("arrays:\n got %s\nwant %s", got, wantArrays)
	}
}

// TestCSVToJSONEdges covers ragged rows, single-cell input, header-only input,
// BOM stripping and empty input — all oracle-verified.
func TestCSVToJSONEdges(t *testing.T) {
	cases := []struct {
		name  string
		input string
		dict  bool
		want  string
	}{
		{"ragged fewer dict (missing key omitted)", "A,B,C\r\n1,2\r\n", true, `[{"A":"1","B":"2"}]`},
		{"ragged more dict (extra cell ignored)", "A,B\r\n1,2,3\r\n", true, `[{"A":"1","B":"2"}]`},
		{"multi rows, mixed widths", "A,B,C\r\n1,2\r\n7,8,9\r\n", true, `[{"A":"1","B":"2"},{"A":"7","B":"8","C":"9"}]`},
		{"ragged arrays keep lengths", "A,B,C\r\n1,2\r\n", false, `[["A","B","C"],["1","2"]]`},
		{"single cell no delimiter", "a", false, `[]`},
		{"header only", "A,B,C\r\n", true, `[]`},
		{"empty input arrays", "", false, `[]`},
		{"empty input dict", "", true, `[]`},
		{"BOM stripped", "\uFEFFA,B\r\n1,2\r\n", false, `[["A","B"],["1","2"]]`},
		{"LF-only rows", "A,B\n1,2\n", false, `[["A","B"],["1","2"]]`},
		{"last line no newline", "A,B\r\n1,2", false, `[["A","B"],["1","2"]]`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := csvValue(t, c.input, c.dict); got != c.want {
				t.Fatalf("got %s want %s", got, c.want)
			}
		})
	}
}

// TestCSVToJSONCustomDelims checks non-default cell/row delimiters (each
// character in the argument is its own delimiter).
func TestCSVToJSONCustomDelims(t *testing.T) {
	rows := parseCSV("a;b\r\n1;2\r\n", []rune{';'}, []rune{'\r', '\n'})
	if got := jsonval.Stringify(csvRowsToValue(rows, false), 0); got != `[["a","b"],["1","2"]]` {
		t.Fatalf("custom delims: %s", got)
	}
}

// TestCSVToJSONViaRecipe exercises the operation through the engine, covering
// Args, the binaryShortString escape parsing of "\r\n", and the JSON dish's
// pretty (indent-4) presentation.
func TestCSVToJSONViaRecipe(t *testing.T) {
	out, err := core.Recipe{{Op: "CSV to JSON", Args: []any{",", "\\r\\n", "Array of dictionaries"}}}.
		Execute(core.NewDish([]byte("a,b\r\n1,2\r\n"), core.TypeString))
	if err != nil {
		t.Fatal(err)
	}
	want := "[\n    {\n        \"a\": \"1\",\n        \"b\": \"2\"\n    }\n]"
	if out.String() != want {
		t.Fatalf("got %q want %q", out.String(), want)
	}
}

// jsonCSV parses JSON (order-preserving) and converts it to CSV with the given
// delimiters, failing the test on error.
func jsonCSV(t *testing.T, jsonText, cell, row string) string {
	t.Helper()
	v, err := jsonval.ParseOrdered([]byte(jsonText))
	if err != nil {
		t.Fatalf("parse %q: %v", jsonText, err)
	}
	out, err := jsonToCSV(v, cell, row)
	if err != nil {
		t.Fatalf("jsonToCSV %q: %v", jsonText, err)
	}
	return out
}

// TestJSONToCSVFixtures transcribes the fixtures from JSONtoCSV.mjs and the two
// JSON-to-CSV cases in CSV.mjs, run through the engine.
func TestJSONToCSVFixtures(t *testing.T) {
	r := core.Recipe{{Op: "JSON to CSV", Args: []any{",", "\\r\\n"}}}
	dictInput := `[{"A":"1","B":"2","C":"3","D":"4","E":"5","F":"6"},{"A":",","B":";","C":"'","D":"\"","E":"","F":""},{"A":"\"hello\"","B":"a\"1","C":"multi\r\nline","D":"","E":"","F":"end"}]`
	arraysInput := `[["A","B","C","D","E","F"],["1","2","3","4","5","6"],[",",";","'","\"","",""],["\"hello\"","a\"1","multi\r\nline","","","end"]]`
	runCases(t, []opCase{
		{"strings as values", `{"a":"1","b":"2","c":"3"}`, "a,b,c\r\n1,2,3\r\n", r},
		{"numbers as values", `{"a":1,"b":2,"c":3}`, "a,b,c\r\n1,2,3\r\n", r},
		{"numbers and strings", `{"a":1,"b":"2","c":3}`, "a,b,c\r\n1,2,3\r\n", r},
		{"boolean and null", `{"a":false,"b":null,"c":3}`, "a,b,c\r\nfalse,null,3\r\n", r},
		{"JSON as an array", `[{"a":1,"b":"2","c":3}]`, "a,b,c\r\n1,2,3\r\n", r},
		{"multiple in array", `[{"a":1,"b":"2","c":3},{"a":1,"b":"2","c":3}]`, "a,b,c\r\n1,2,3\r\n1,2,3\r\n", r},
		{"empty JSON", `{}`, "\r\n\r\n", r},
		{"empty JSON in array", `[{}]`, "\r\n\r\n", r},
		{"nested JSON", `{"a":1,"b":{"c":2,"d":3}}`, "a,b.c,b.d\r\n1,2,3\r\n", r},
		{"nested array", `{"a":1,"b":[2,3]}`, "a,b.0,b.1\r\n1,2,3\r\n", r},
		{"nested JSON, nested array", `{"a":1,"b":{"c":[2,3],"d":4}}`, "a,b.c.0,b.c.1,b.d\r\n1,2,3,4\r\n", r},
		{"nested array, nested JSON", `{"a":1,"b":[{"c":3,"d":4}]}`, "a,b.0.c,b.0.d\r\n1,3,4\r\n", r},
		{"nested array, nested array", `{"a":1,"b":[[2,3]]}`, "a,b.0.0,b.0.1\r\n1,2,3\r\n", r},
		{"nested JSON, nested JSON", `{"a":1,"b":{"c":{"d":2,"e":3}}}`, "a,b.c.d,b.c.e\r\n1,2,3\r\n", r},
		{"array of dictionaries (round trip)", dictInput, exampleCSV, r},
		{"array of arrays (round trip)", arraysInput, exampleCSV, r},
	})
}

// TestJSONToCSVEdges covers the JavaScript-quirk behaviours (a top-level string
// becomes indexed columns; primitives yield empty headers) and the flatten
// fallback, all oracle-verified.
func TestJSONToCSVEdges(t *testing.T) {
	cases := []struct{ name, input, want string }{
		{"top-level string indexed", `"hello"`, "0,1,2,3,4\r\nh,e,l,l,o\r\n"},
		{"top-level number", `5`, "\r\n\r\n"},
		{"top-level bool", `true`, "\r\n\r\n"},
		{"top-level array of numbers", `[1,2,3]`, "\r\n\r\n\r\n\r\n"},
		{"big number formatting", `{"a":100000000000000000000}`, "a\r\n100000000000000000000\r\n"},
		{"float formatting", `{"a":1.5}`, "a\r\n1.5\r\n"},
		{"empty object value", `{"a":{}}`, "a\r\n{}\r\n"},
		{"empty array value", `{"a":[]}`, "a\r\n[]\r\n"},
		{"deeply nested empty", `{"a":{"b":{}}}`, "a.b\r\n{}\r\n"},
		{"array of arrays ragged", `[["A","B"],["1"]]`, "A,B\r\n1\r\n"},
		{"array of arrays nested cell (fallback)", `[["A","B"],[{"x":1},"2"]]`, "0.0,0.1,1.0.x,1.1\r\nA,B,1,2\r\n"},
		{"dict nested value force-quoted", `{"a":{"x":"p,q"}}`, "a.x\r\n\"p,q\"\r\n"},
		{"mixed dict and scalar row", `[{"a":1},5]`, "a\r\n1\r\nundefined\r\n"},
		{"string element in array", `["hi"]`, "0,1\r\nh,i\r\n"},
		{"array with empty array element", `[[]]`, "\r\n"},
		{"empty top-level array", `[]`, "\r\n\r\n"},
		{"array path with non-array row (fallback)", `[["a","b"],5]`, "1,0.0,0.1\r\n5,a,b\r\n"},
		{"key order preserved", `{"z":1,"a":2}`, "z,a\r\n1,2\r\n"},
		{"quoting cell and row delimiters", `{"a":"x,y","b":"li\nne"}`, "a,b\r\n\"x,y\",\"li\nne\"\r\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := jsonCSV(t, c.input, ",", "\r\n"); got != c.want {
				t.Fatalf("got %q want %q", got, c.want)
			}
		})
	}
}

// TestJSONToCSVCustomDelims checks non-default cell/row delimiters.
func TestJSONToCSVCustomDelims(t *testing.T) {
	if got := jsonCSV(t, `[["a","b"],["1","2"]]`, "\t", ";"); got != "a\tb;1\t2;" {
		t.Fatalf("custom delims: %q", got)
	}
}

// TestJSONToCSVErrors covers input that cannot be represented as CSV (a bare
// null, whose keys cannot be read) and invalid JSON.
func TestJSONToCSVErrors(t *testing.T) {
	if v, err := jsonval.ParseOrdered([]byte(`null`)); err != nil {
		t.Fatal(err)
	} else if _, err := jsonToCSV(v, ",", "\r\n"); err == nil {
		t.Fatal("expected error for null input")
	}
	_, err := JSONToCSV{}.Run(sdish("not json"), []any{",", "\\r\\n"})
	if err == nil || !strings.Contains(err.Error(), "JSON") {
		t.Fatalf("invalid JSON: got %v", err)
	}
	// Valid JSON that cannot become CSV (bare null) errors from Run's jsonToCSV.
	if _, err := (JSONToCSV{}).Run(sdish("null"), []any{",", "\\r\\n"}); err == nil {
		t.Fatal("expected error for null via Run")
	}
}

// TestJSONToCSVKeyOrdering checks that object keys enumerate in ECMAScript order
// (integer indices ascending, then the rest in insertion order), matching
// JavaScript's Object.keys / JSON.stringify.
func TestJSONToCSVKeyOrdering(t *testing.T) {
	cases := []struct{ name, input, want string }{
		{"integer keys ascending", `{"2":"a","10":"b","1":"c","x":"d"}`, "1,2,10,x\r\nc,a,b,d\r\n"},
		{"integers before insertion strings", `{"b":1,"0":2,"a":3,"2":4}`, "0,2,b,a\r\n2,4,1,3\r\n"},
		{"leading zero is not an index", `{"01":1,"1":2}`, "1,01\r\n2,1\r\n"},
		{"negative is not an index", `{"-1":1,"0":2}`, "0,-1\r\n2,1\r\n"},
		{"2^32-1 is not an index", `{"4294967295":1,"4294967296":2,"5":3}`, "5,4294967295,4294967296\r\n3,1,2\r\n"},
		{"flatten integer key ordering", `[{"a":1},null]`, "1,0.a\r\nnull,1\r\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := jsonCSV(t, c.input, ",", "\r\n"); got != c.want {
				t.Fatalf("got %q want %q", got, c.want)
			}
		})
	}
}

// TestJSONToCSVRowShapes covers dictionary rows that are arrays or other
// primitives (JS property access on them), exercising csvGet.
func TestJSONToCSVRowShapes(t *testing.T) {
	cases := []struct{ name, input, want string }{
		{"array row, non-numeric key", `[{"a":1},[9,8]]`, "a\r\n1\r\nundefined\r\n"},
		{"array row, numeric key", `[{"0":1},[9,8]]`, "0\r\n1\r\n9\r\n"},
		{"dict row missing key", `[{"a":1},{"b":2}]`, "a\r\n1\r\nundefined\r\n"},
		{"string row out of range", `[{"a":1},"xy"]`, "a\r\n1\r\nundefined\r\n"},
		{"bool value", `{"a":true}`, "a\r\ntrue\r\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := jsonCSV(t, c.input, ",", "\r\n"); got != c.want {
				t.Fatalf("got %q want %q", got, c.want)
			}
		})
	}
}

// TestCSVToJSONDuplicateHeader checks that a repeated header name keeps its first
// column position with the last row value (JS object assignment semantics).
func TestCSVToJSONDuplicateHeader(t *testing.T) {
	if got := csvValue(t, "A,A\r\n1,2\r\n", true); got != `[{"A":"2"}]` {
		t.Fatalf("duplicate header: %s", got)
	}
}

// TestCSVToJSONIntegerHeaders checks integer header reordering in the JSON dish.
func TestCSVToJSONIntegerHeaders(t *testing.T) {
	if got := csvValue(t, "2,1\r\na,b\r\n", true); got != `[{"1":"b","2":"a"}]` {
		t.Fatalf("integer headers: %s", got)
	}
}

// TestJSONParseOrderedErrors covers the order-preserving parser's error paths:
// trailing data, malformed containers and truncation.
func TestJSONParseOrderedErrors(t *testing.T) {
	for _, in := range []string{"1 2", "not json", `{"a":}`, `{"a":1`, "[1,", "[1,2", "[1 2]", `{"a" 1}`, `{1:2}`, ""} {
		if _, err := jsonval.ParseOrdered([]byte(in)); err == nil {
			t.Fatalf("parse %q: expected error", in)
		}
	}
	// Duplicate keys keep the last value at the first position.
	v, err := jsonval.ParseOrdered([]byte(`{"a":1,"b":2,"a":3}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := jsonval.Stringify(v, 0); got != `{"a":3,"b":2}` {
		t.Fatalf("dup key: %s", got)
	}
}
