package ops

// Generator tests: jsGenerate's output must byte-match escodegen.generate with
// CyberChef's default options (indent "\t", quotes "auto", semicolons true).
// Golden vectors in testdata/jsbeautify.jsonl are produced from the exact
// escodegen package.

import (
	"bufio"
	"encoding/json"
	"os"
	"testing"
)

func TestJSBeautifyGolden(t *testing.T) {
	f, err := os.Open("testdata/jsbeautify.jsonl")
	if err != nil {
		t.Fatalf("open golden: %v", err)
	}
	defer func() { _ = f.Close() }()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	n := 0
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var vec struct {
			Src  string `json:"src"`
			Want string `json:"want"`
		}
		if err := json.Unmarshal(line, &vec); err != nil {
			t.Fatalf("bad golden line: %v", err)
		}
		n++
		ast, err := jsParse(vec.Src)
		if err != nil {
			t.Errorf("parse %q: unexpected error %v", vec.Src, err)
			continue
		}
		if got := jsGenerate(ast, "\t", "auto", true); got != vec.Want {
			t.Errorf("beautify %q:\n got:\n%s\nwant:\n%s", vec.Src, got, vec.Want)
		}
	}
	if n == 0 {
		t.Fatal("no golden vectors loaded")
	}
}
