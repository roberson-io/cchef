package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/jsonval"
)

func jpathRecipe(query, delim string) core.Recipe {
	return core.Recipe{{Op: "JPath expression", Args: []any{query, delim}}}
}

// JPath expression has no upstream fixture file (CyberChef wraps jsonpath-plus);
// these are authoritative outputs captured from the CyberChef-server oracle. cchef
// reimplements JSONPath over the order-preserving jsonvalue.go representation so
// each matched value serializes byte-for-byte like jsonpath-plus's
// results.map(JSON.stringify).join(delimiter).
func TestJPathFixtures(t *testing.T) {
	const store = `{"store":{"book":[{"title":"A","price":8.95,"author":"x"},` +
		`{"title":"B","price":12.99,"author":"y"},{"title":"C","price":5.0}],` +
		`"bicycle":{"color":"red","price":19.95}}}`
	runCases(t, []opCase{
		{
			"JPath: wildcard titles", store, `"A"|"B"|"C"`,
			jpathRecipe("$.store.book[*].title", "|"),
		},
		{
			"JPath: recursive descent price", store, `8.95|12.99|5|19.95`,
			jpathRecipe("$..price", "|"),
		},
		{
			"JPath: index returns object in insertion order", store,
			`{"title":"A","price":8.95,"author":"x"}`,
			jpathRecipe("$.store.book[0]", "|"),
		},
		{
			"JPath: slice", store, `"A"|"B"`,
			jpathRecipe("$.store.book[0:2].title", "|"),
		},
		{
			"JPath: filter", store, `"A"|"C"`,
			jpathRecipe("$.store.book[?(@.price<10)].title", "|"),
		},
		{
			"JPath: script expression", store, `"C"`,
			jpathRecipe("$.store.book[(@.length-1)].title", "|"),
		},
		{
			"JPath: index union", store, `"A"|"B"`,
			jpathRecipe("$.store.book[0,1].title", "|"),
		},
		{
			"JPath: wildcard over object is insertion order", store,
			`[{"title":"A","price":8.95,"author":"x"},` +
				`{"title":"B","price":12.99,"author":"y"},{"title":"C","price":5}]` +
				`|{"color":"red","price":19.95}`,
			jpathRecipe("$.store.*", "|"),
		},

		{"JPath: root", `{"a":1,"b":2}`, `{"a":1,"b":2}`, jpathRecipe("$", "|")},
		{
			"JPath: wildcard uses ES key order", `{"2":"a","1":"b","x":"c"}`,
			`"b"|"a"|"c"`, jpathRecipe("$.*", "|"),
		},
		{
			"JPath: numeric index on object", `{"0":"x"}`, `"x"`,
			jpathRecipe("$[0]", "|"),
		},
		{
			"JPath: negative plain index yields empty", `[1,2,3]`, "",
			jpathRecipe("$[-1]", "|"),
		},
		{
			"JPath: step slice", `[0,1,2,3,4,5]`, `0|2|4`,
			jpathRecipe("$[::2]", "|"),
		},
		{
			"JPath: negative slice bounds", `[0,1,2,3,4,5]`, `2|3|4`,
			jpathRecipe("$[-4:-1]", "|"),
		},
		{
			"JPath: length magic on array", `{"a":[1,2,3]}`, `3`,
			jpathRecipe("$.a.length", "|"),
		},
		{
			"JPath: length magic on string", `{"s":"hello"}`, `5`,
			jpathRecipe("$.s.length", "|"),
		},
		{
			"JPath: length not applied to object", `{"o":{"a":1}}`, "",
			jpathRecipe("$.o.length", "|"),
		},
		{
			"JPath: recursive descent document order",
			`{"a":{"b":{"a":1}},"c":[{"a":2}]}`, `{"b":{"a":1}}|1|2`,
			jpathRecipe("$..a", "|"),
		},
		{
			"JPath: existence filter", `[{"a":1},{"b":2}]`, `{"a":1}`,
			jpathRecipe("$[?(@.a)]", "|"),
		},
		{
			"JPath: null value matched", `{"a":null}`, `null`,
			jpathRecipe("$.a", "|"),
		},
		{
			"JPath: child of array yields empty", `[{"x":1}]`, "",
			jpathRecipe("$.x", "|"),
		},
		{
			"JPath: bracket quoted key", `{"a b":1}`, `1`,
			jpathRecipe(`$["a b"]`, "|"),
		},
		{
			"JPath: filter arithmetic", `[{"a":1,"b":2},{"a":3,"b":1}]`, `3`,
			jpathRecipe("$[?(@.a+@.b==4)].a", "|"),
		},
		{
			"JPath: filter logical and", `[{"a":1,"b":2},{"a":1,"b":3}]`, `3`,
			jpathRecipe("$[?(@.a==1 && @.b>2)].b", "|"),
		},
		{
			"JPath: newline delimiter", `{"a":[1,2]}`, "1\n2",
			jpathRecipe("$.a[*]", `\n`),
		},
		{
			"JPath: no match yields empty", `{"a":1}`, "",
			jpathRecipe("$.missing", "|"),
		},
		{
			"JPath: filter string equality", `[{"n":"a"},{"n":"b"}]`, `"b"`,
			jpathRecipe(`$[?(@.n=="b")].n`, "|"),
		},
		{
			"JPath: filter bracket access", `[{"k":1},{"k":2}]`, `2`,
			jpathRecipe(`$[?(@["k"]==2)].k`, "|"),
		},
		{
			"JPath: filter unary minus", `[{"v":-5},{"v":3}]`, `-5`,
			jpathRecipe("$[?(@.v==-5)].v", "|"),
		},
		{
			"JPath: filter division", `[{"v":10},{"v":20}]`, `20`,
			jpathRecipe("$[?(@.v/2==10)].v", "|"),
		},
		{
			"JPath: filter string comparison", `[{"s":"apple"},{"s":"banana"}]`,
			`"banana"`, jpathRecipe(`$[?(@.s>"b")].s`, "|"),
		},
		{
			"JPath: filter null equality", `[{"v":null},{"v":1}]`, `null`,
			jpathRecipe("$[?(@.v==null)].v", "|"),
		},
		{
			"JPath: filter string concat", `[{"a":"x","b":"y"}]`, `"x"`,
			jpathRecipe(`$[?(@.a+@.b=="xy")].a`, "|"),
		},
		{
			"JPath: filter length property", `[{"arr":[1,2]},{"arr":[1,2,3]}]`,
			`[1,2,3]`, jpathRecipe("$[?(@.arr.length==3)].arr", "|"),
		},
		{
			"JPath: filter boolean equality", `[{"ok":false},{"ok":true}]`, `false`,
			jpathRecipe("$[?(@.ok==false)].ok", "|"),
		},
		{
			"JPath: script arithmetic index", `{"x":[1,2,3]}`, `3`,
			jpathRecipe("$.x[(1+1)]", "|"),
		},
		{
			"JPath: negative step yields empty", `[1,2,3]`, "",
			jpathRecipe("$[::-1]", "|"),
		},
		{
			"JPath: multi-name union yields empty", `{"a":1,"b":2,"c":3}`, "",
			jpathRecipe(`$["a","c"]`, "|"),
		},
	})
}

func TestJPathErrors(t *testing.T) {
	if _, err := runOp(t, "JPath expression", `not json`, "$.a", "|"); err == nil ||
		!strings.HasPrefix(err.Error(), "Invalid input JSON: ") {
		t.Fatalf("invalid JSON: got %v", err)
	}
	if _, err := runOp(t, "JPath expression", `{"a":1}`, "$[?(@>2)]", "|"); err == nil ||
		!strings.HasPrefix(err.Error(), "Invalid JPath expression: ") {
		t.Fatalf("invalid query: got %v", err)
	}
}

func jpCtx(t *testing.T, j string) any {
	t.Helper()
	v, err := jsonval.ParseOrdered([]byte(j))
	if err != nil {
		t.Fatalf("bad ctx JSON %q: %v", j, err)
	}
	return v
}

func jpTruthy(t *testing.T, expr, ctx string) bool {
	t.Helper()
	e, err := parseJPExpr(expr)
	if err != nil {
		t.Fatalf("parse %q: %v", expr, err)
	}
	return evalTruthy(e, jpCtx(t, ctx))
}

func TestJPExprTruthiness(t *testing.T) {
	ctx := `{"n":5,"s":"hi","t":true,"f":false,"z":0,"e":"","nul":null,"arr":[1,2],"neg":-3}`
	cases := []struct {
		expr string
		want bool
	}{
		{"@.n==5", true},
		{"@.n!=5", false},
		{"@.n>3", true},
		{"@.n>=5", true},
		{"@.n<3", false},
		{"@.n<=5", true},
		{"@.s==\"hi\"", true},
		{"@.s>\"a\"", true},
		{"@.s<\"a\"", false},
		{"@.t==true", true},
		{"@.f==false", true},
		{"@.nul==null", true},
		{"@.n==\"5\"", true},
		{"@.s==5", false},
		{"@.n+1==6", true},
		{"@.n-5==0", true},
		{"@.n*2==10", true},
		{"@.n/5==1", true},
		{"-@.n==-5", true},
		{"@.n>3 && @.s==\"hi\"", true},
		{"@.n<3 || @.t", true},
		{"!@.f", true},
		{"!@.t", false},
		{"@.arr.length==2", true},
		{"@.n", true},
		{"@.z", false},
		{"@.e", false},
		{"@.nul", false},
		{"@.arr", true},
		{"@.missing", false},
		{"@.missing==1", false},
		{"@.missing!=1", true},
		{"@.s+\"!\"==\"hi!\"", true},
		{"@.neg<0", true},
		{"(@.n+@.neg)==2", true},
		{"@[\"n\"]==5", true},
	}
	for _, c := range cases {
		if got := jpTruthy(t, c.expr, ctx); got != c.want {
			t.Errorf("truthy(%q) = %v, want %v", c.expr, got, c.want)
		}
	}
}

func TestJPExprCoercion(t *testing.T) {
	// String concatenation coerces the other operand (number, bool, null).
	cases := []struct{ expr, ctx string }{
		{`@.s+@.n=="x5"`, `{"s":"x","n":5}`},
		{`@.n+@.s=="5x"`, `{"n":5,"s":"x"}`},
		{`@.s+@.b=="xtrue"`, `{"s":"x","b":true}`},
		{`@.s+@.z=="xnull"`, `{"s":"x","z":null}`},
	}
	for _, c := range cases {
		if !jpTruthy(t, c.expr, c.ctx) {
			t.Errorf("expected true: %q on %s", c.expr, c.ctx)
		}
	}
}

func TestJPExprParseErrors(t *testing.T) {
	for _, expr := range []string{
		"@",          // bare @ is an error
		"@.a =~ /x/", // regex operator unsupported
		"@.a ==",     // trailing operator
		"(@.a",       // unclosed paren
		"@.a == 'x",  // unterminated string
		"@.",         // missing name after dot
		"@[",         // unterminated bracket in @ path
		"1 2",        // unexpected trailing token
		"foo",        // unknown identifier
		"@.a[true]",  // invalid @ path index
		"1+@",        // bare @ in a binary right operand
		"!@",         // bare @ in a unary operand
		"(@)",        // bare @ inside parentheses
		"@.a==)",     // unexpected ')' as a primary
		"@[0",        // missing ']' after @ index
	} {
		if _, err := parseJPExpr(expr); err == nil {
			t.Errorf("expected parse error for %q", expr)
		}
	}
}

// jpEval compiles and runs a query directly, returning the serialized matches.
func jpEval(t *testing.T, jsonStr, query string) []string {
	t.Helper()
	root, err := jsonval.ParseOrdered([]byte(jsonStr))
	if err != nil {
		t.Fatalf("bad JSON %q: %v", jsonStr, err)
	}
	segs, err := parseJPath(query)
	if err != nil {
		t.Fatalf("parse %q: %v", query, err)
	}
	matches := evalJPath(root, segs)
	out := make([]string, len(matches))
	for i, m := range matches {
		out[i] = jsonval.Stringify(m, 0)
	}
	return out
}

func TestParseJPathErrors(t *testing.T) {
	for _, q := range []string{
		"$@",        // unexpected character
		"$.",        // empty property name
		"$[",        // unterminated bracket
		"$[1:z]",    // invalid slice bound
		"$[?(@)]",   // bare @ propagates
		"$[(@.x",    // unterminated script bracket
		"$..[",      // recursive then unterminated bracket
		"$.a[?(x)]", // unknown identifier in filter
		"$[?x]",     // filter without parentheses
		"$[(1)x]",   // trailing text after script parens
		"$[(@)]",    // bare @ inside script
		"$[z:1]",    // invalid slice start
		"$[1:2:z]",  // invalid slice step
	} {
		if _, err := parseJPath(q); err == nil {
			t.Errorf("expected error for %q", q)
		}
	}
}

func TestJPathSegmentBehaviors(t *testing.T) {
	eq := func(got []string, want ...string) bool {
		if len(got) != len(want) {
			return false
		}
		for i := range got {
			if got[i] != want[i] {
				return false
			}
		}
		return true
	}
	cases := []struct {
		name, json, query string
		want              []string
	}{
		{"script string key", `{"a":1,"b":2}`, `$[('a')]`, []string{"1"}},
		{"multi-name union empty", `{"a":1,"b":2}`, `$['a','b']`, nil},
		{"single quoted name", `{"a b":1}`, `$['a b']`, []string{"1"}},
		{"filter index access", `[[1,2],[3,4]]`, `$[?(@[0]==3)]`, []string{"[3,4]"}},
		{"slice over upper bound", `[1,2,3]`, `$[10:20]`, nil},
		{"slice under lower bound", `[1,2,3]`, `$[-10:2]`, []string{"1", "2"}},
		{"negative step empty", `[1,2,3]`, `$[::-1]`, nil},
		{"recursive then bracket", `{"a":[10,20]}`, `$..[0]`, []string{"10"}},
		{"script out of range", `[1,2]`, `$[(5)]`, nil},
		{"index union with missing", `[1,2,3]`, `$[0,9]`, []string{"1"}},
		{"length on missing in filter", `[{"v":1}]`, `$[?(@.missing<5)]`, nil},
		{"slice on object empty", `{"a":1}`, `$.a[0:2]`, nil},
		{"script referencing missing", `[1,2]`, `$[(@.missing)]`, nil},
		{"unquoted bracket word is a name", `{"foo":1}`, `$[foo]`, []string{"1"}},
	}
	for _, c := range cases {
		if got := jpEval(t, c.json, c.query); !eq(got, c.want...) {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}

func TestJPExprMoreEval(t *testing.T) {
	// String-that-parses compared to a number (JS loose ==), @-index access,
	// undefined ordering, and incomparable ordering.
	cases := []struct {
		expr, ctx string
		want      bool
	}{
		{`@.s==5`, `{"s":"5"}`, true},
		{`@[0]==1`, `[1,2]`, true},
		{`@.missing<5`, `{}`, false},
		{`@.t<@.f`, `{"t":true,"f":false}`, false},
		{`-@.missing==0`, `{}`, false},    // unary minus on undefined
		{`-@.s==0`, `{"s":"x"}`, false},   // unary minus on non-number
		{`@.missing+1==1`, `{}`, false},   // arithmetic with undefined
		{`@.t-1==0`, `{"t":true}`, false}, // arithmetic with non-number
	}
	for _, c := range cases {
		e, err := parseJPExpr(c.expr)
		if err != nil {
			t.Fatalf("parse %q: %v", c.expr, err)
		}
		ctx, _ := jsonval.ParseOrdered([]byte(c.ctx))
		if got := evalTruthy(e, ctx); got != c.want {
			t.Errorf("%q on %s = %v want %v", c.expr, c.ctx, got, c.want)
		}
	}
}

func TestJPathInvalidJSONMessage(t *testing.T) {
	_, err := runOp(t, "JPath expression", `{bad`, "$", "|")
	if err == nil || !strings.HasPrefix(err.Error(), "Invalid input JSON: ") {
		t.Fatalf("got %v", err)
	}
}
