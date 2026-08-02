package ops

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// jsonataFixture is one of CyberChef's own cases, read from its test file
// (CyberChef's tests/operations/tests/Jsonata.mjs).
type jsonataFixture struct {
	Name     string `json:"name"`
	Input    string `json:"input"`
	Expected string `json:"expected"`
	Query    string `json:"query"`
}

// readJsonataFixtures loads the transcribed cases.
func readJsonataFixtures(t *testing.T) []jsonataFixture {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "jsonata.json"))
	if err != nil {
		t.Fatalf("reading the fixtures: %v", err)
	}
	var out []jsonataFixture
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("reading the fixtures: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("no fixtures were read")
	}
	return out
}

// jsonataKeyOrderOnly names the cases whose result is right but whose object
// keys come back in a different order.
//
// The engine cchef queries with keeps the order a document's keys were written
// in, but only inside itself: the type that holds it is not part of its public
// interface, so a result reaches the operation as an ordinary map and the order
// is lost. The values, the structure and the types are all correct, which is
// what the check below confirms for these cases.
var jsonataKeyOrderOnly = map[string]bool{
	"Jsonata: Returns the first item (an object)":                                 true,
	"Jsonata: Returns the second item":                                            true,
	"Jsonata: Returns the last item":                                              true,
	"Jsonata: Negative indexed count from the end":                                true,
	"Jsonata: Returns a range of items by creating an array of indexes":           true,
	"Jsonata: Select the Phone items that have a type field that equals 'mobile'": true,
}

// jsonataOutstanding names the cases that do not yet come out right, with why.
var jsonataOutstanding = map[string]string{
	"Jsonata: Division": "the engine reads the / after a ] as the start of a " +
		"regular expression, so an expression dividing one indexed value by " +
		"another does not compile",
	"Jsonata: Select the values of all the fields of 'Address'": "the values of " +
		"an object come back in a different order from the one they were " +
		"written in",
}

// TestJsonataQueryFixtures covers CyberChef's own cases. Each is checked exactly
// where it can be, and by shape where only the order of an object's keys
// differs; the two that do not yet come out right are named above with a reason,
// and this fails if one of them starts working, so the list cannot go stale.
func TestJsonataQueryFixtures(t *testing.T) {
	var exact, orderOnly, outstanding int

	for _, f := range readJsonataFixtures(t) {
		t.Run(f.Name, func(t *testing.T) {
			got, err := runOp(t, "Jsonata Query", f.Input, f.Query)

			if reason, known := jsonataOutstanding[f.Name]; known {
				if err == nil && got == f.Expected {
					t.Fatalf("this case now comes out right; remove it from the outstanding list (%s)", reason)
				}
				outstanding++
				return
			}
			if err != nil {
				t.Fatalf("Run: %v", err)
			}

			if got == f.Expected {
				exact++
				return
			}
			if jsonataKeyOrderOnly[f.Name] {
				if !sameJSONValue(t, got, f.Expected) {
					t.Fatalf("this case differs by more than the order of its keys:\ngot  %s\nwant %s", got, f.Expected)
				}
				orderOnly++
				return
			}
			t.Errorf("got  %s\nwant %s", got, f.Expected)
		})
	}

	t.Logf("%d exact, %d differ only in key order, %d outstanding", exact, orderOnly, outstanding)
}

// sameJSONValue reports whether two JSON texts describe the same value, ignoring
// the order an object's keys are written in.
func sameJSONValue(t *testing.T, a, b string) bool {
	t.Helper()
	var x, y any
	if err := json.Unmarshal([]byte(a), &x); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(b), &y); err != nil {
		return false
	}
	return reflect.DeepEqual(x, y)
}

// TestJsonataQueryRejectsBadInput covers input that is not a JSON document.
func TestJsonataQueryRejectsBadInput(t *testing.T) {
	for _, input := range []string{"", "not json", "{", `{"a":}`} {
		_, err := runOp(t, "Jsonata Query", input, "a")
		if err == nil {
			t.Errorf("%q was read as JSON", input)
			continue
		}
		if !strings.HasPrefix(err.Error(), "Invalid input JSON: ") {
			t.Errorf("got %q, want CyberChef's wording", err.Error())
		}
	}
}

// TestJsonataQueryRejectsBadQuery covers an expression that cannot be read.
func TestJsonataQueryRejectsBadQuery(t *testing.T) {
	for _, query := range []string{"(", "a[", "$foo(", "1 +"} {
		_, err := runOp(t, "Jsonata Query", `{"a":1}`, query)
		if err == nil {
			t.Errorf("%q was read as an expression", query)
			continue
		}
		if !strings.HasPrefix(err.Error(), "Invalid Jsonata Expression: ") {
			t.Errorf("got %q, want CyberChef's wording", err.Error())
		}
	}
}

// TestJsonataQuerySelectsNothing covers an expression that matches nothing,
// which is reported as the empty string rather than as no result.
func TestJsonataQuerySelectsNothing(t *testing.T) {
	got, err := runOp(t, "Jsonata Query", `{"a":1}`, "b")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got != `""` {
		t.Errorf("got %s, want an empty string", got)
	}
}

// TestJsonataQueryDefaults covers the argument the operation starts from.
func TestJsonataQueryDefaults(t *testing.T) {
	op, ok := core.Default.Get("Jsonata Query")
	if !ok {
		t.Fatal("Jsonata Query is not registered")
	}
	args := op.Args()
	if len(args) != 1 || args[0].Name != "Query" || args[0].Value != "string" {
		t.Errorf("arguments are %+v", args)
	}
}

// TestJsonataQueryListsNameRealCases covers the two lists above naming cases
// that exist. A fixture renamed upstream would otherwise silently stop being
// checked.
func TestJsonataQueryListsNameRealCases(t *testing.T) {
	names := map[string]bool{}
	for _, f := range readJsonataFixtures(t) {
		names[f.Name] = true
	}
	for name := range jsonataKeyOrderOnly {
		if !names[name] {
			t.Errorf("no fixture is called %q", name)
		}
	}
	for name := range jsonataOutstanding {
		if !names[name] {
			t.Errorf("no fixture is called %q", name)
		}
	}
}

// TestJsonataQueryHandlesAFailingExpression covers an expression that compiles
// but cannot be evaluated, which is reported rather than unwinding.
func TestJsonataQueryHandlesAFailingExpression(t *testing.T) {
	for _, query := range []string{`$notAFunction()`, `"text" - 1`, `$number("nope")`} {
		if _, err := runOp(t, "Jsonata Query", `{"a":1}`, query); err == nil {
			t.Errorf("%q was evaluated without complaint", query)
		}
	}
}
