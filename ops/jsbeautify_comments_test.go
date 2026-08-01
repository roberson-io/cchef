package ops

// Comment-mode generator tests: jsGenerateComments (parse-full + attachComments +
// comment emission) must byte-match escodegen.generate with attachComments and
// comment:true. Golden vectors in testdata/jsbeautify_comments.jsonl are produced
// from the exact escodegen/estraverse packages.

import (
	"bufio"
	"encoding/json"
	"os"
	"testing"
)

func TestJSBeautifyComments(t *testing.T) {
	f, err := os.Open("testdata/jsbeautify_comments.jsonl")
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
			Src  string `json:"src"`
			Want string `json:"want"`
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
		cs := jsAttachComments(ast, comments, tokens)
		if got := jsGenerateComments(ast, "\t", "auto", true, cs); got != vec.Want {
			t.Errorf("beautify+comments %q:\n got:\n%s\nwant:\n%s", vec.Src, got, vec.Want)
		}
	}
	if n == 0 {
		t.Fatal("no golden vectors loaded")
	}
}
