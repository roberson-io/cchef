package ops

// Verifies that jsParseFull produces node ranges, a comments array, and a tokens
// array byte-identical to esprima.parseScript(src, {range:true, tokens:true,
// comment:true}). Golden data in testdata/jsparser_ranges.jsonl is generated
// from esprima. Node ranges are compared as a sorted multiset of
// (type, start, end); comments and tokens are compared in source order.

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"testing"
)

// jsCollectRanges walks an AST node collecting "type@start:end" for every node
// that carries a range.
func jsCollectRanges(n any, out *[]string) {
	switch v := n.(type) {
	case jsObject:
		typ := jsNodeType(v)
		if r, ok := jsField(v, "range").([]any); ok && typ != "" && len(r) == 2 {
			*out = append(*out, fmt.Sprintf("%s@%v:%v", typ, jsRangeInt(r[0]), jsRangeInt(r[1])))
		}
		for _, pr := range v {
			if pr.k == "range" {
				continue
			}
			jsCollectRanges(pr.v, out)
		}
	case []any:
		for _, e := range v {
			jsCollectRanges(e, out)
		}
	}
}

func jsRangeInt(v any) int {
	if f, ok := v.(float64); ok {
		return int(f)
	}
	return -1
}

func TestJSParserRanges(t *testing.T) {
	f, err := os.Open("testdata/jsparser_ranges.jsonl")
	if err != nil {
		t.Fatalf("open golden: %v", err)
	}
	defer func() { _ = f.Close() }()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	n := 0
	for sc.Scan() {
		if len(sc.Bytes()) == 0 {
			continue
		}
		var vec struct {
			Src      string  `json:"src"`
			Nodes    [][]any `json:"nodes"`    // [type, start, end]
			Comments [][]any `json:"comments"` // [type, value, start, end]
			Tokens   [][]any `json:"tokens"`   // [type, value, start, end]
		}
		if err := json.Unmarshal(sc.Bytes(), &vec); err != nil {
			t.Fatalf("bad golden line: %v", err)
		}
		n++
		ast, comments, tokens, err := jsParseFull(vec.Src)
		if err != nil {
			t.Errorf("parse %q: unexpected error %v", vec.Src, err)
			continue
		}
		checkRanges(t, vec.Src, ast, vec.Nodes)
		checkComments(t, vec.Src, comments, vec.Comments)
		checkTokens(t, vec.Src, tokens, vec.Tokens)
	}
	if n == 0 {
		t.Fatal("no golden vectors loaded")
	}
}

func checkRanges(t *testing.T, src string, ast jsObject, want [][]any) {
	t.Helper()
	var got []string
	jsCollectRanges(ast, &got)
	wantList := make([]string, len(want))
	for i, w := range want {
		wantList[i] = fmt.Sprintf("%s@%v:%v", w[0], jsRangeInt2(w[1]), jsRangeInt2(w[2]))
	}
	sort.Strings(got)
	sort.Strings(wantList)
	if fmt.Sprint(got) != fmt.Sprint(wantList) {
		t.Errorf("ranges %q:\n got:  %v\n want: %v", src, got, wantList)
	}
}

func checkComments(t *testing.T, src string, got []jsObject, want [][]any) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("comments %q: got %d, want %d", src, len(got), len(want))
		return
	}
	for i, w := range want {
		typ := jsStr(jsField(got[i], "type"))
		val := jsStr(jsField(got[i], "value"))
		r, _ := jsField(got[i], "range").([]any)
		if typ != w[0] || val != w[1] || len(r) != 2 || jsRangeInt(r[0]) != jsRangeInt2(w[2]) || jsRangeInt(r[1]) != jsRangeInt2(w[3]) {
			t.Errorf("comment %q #%d: got {%q %q %v}, want %v", src, i, typ, val, r, w)
		}
	}
}

func checkTokens(t *testing.T, src string, got []jsObject, want [][]any) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("tokens %q: got %d, want %d", src, len(got), len(want))
		return
	}
	for i, w := range want {
		typ := jsStr(jsField(got[i], "type"))
		val := jsStr(jsField(got[i], "value"))
		r, _ := jsField(got[i], "range").([]any)
		if typ != w[0] || val != w[1] || len(r) != 2 || jsRangeInt(r[0]) != jsRangeInt2(w[2]) || jsRangeInt(r[1]) != jsRangeInt2(w[3]) {
			t.Errorf("token %q #%d: got {%q %q %v}, want %v", src, i, typ, val, r, w)
		}
	}
}

func jsRangeInt2(v any) int {
	if f, ok := v.(float64); ok {
		return int(f)
	}
	return -1
}
