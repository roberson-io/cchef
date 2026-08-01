package ops

import (
	"strings"
	"testing"
)

// TestHTMLToTextPassesInputThrough covers the whole of what the operation does.
// In CyberChef it changes how the result is shown rather than what it holds:
// the type of the value is what the interface reads to decide whether to render
// markup, and this operation hands the value on as plain text so the markup is
// shown as it stands. Nothing renders anything at a command line, so here it
// passes its input through unchanged.
func TestHTMLToTextPassesInputThrough(t *testing.T) {
	for _, tc := range []struct{ name, input string }{
		{"nothing at all", ""},
		{"plain text", "hello"},
		{"a tag", "<b>hello</b>"},
		{"a whole document", "<html><body><p>one</p><p>two</p></body></html>"},
		{"an entity, which is left as it is", "&amp; &lt; &#65;"},
		{"a script, which is not run or removed", "<script>alert(1)</script>"},
		{"a comment", "<!-- nothing to see -->"},
		{"markup that does not close", "<div><span>text"},
		{"characters outside ASCII", "<p>héllo · 日本語 · 😀</p>"},
		{"lines and tabs", "<p>one</p>\n\t<p>two</p>\r\n"},
		{"something that is not markup at all", "1 < 2 && 3 > 2"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := runOp(t, "HTML To Text", tc.input)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if out != tc.input {
				t.Errorf("got %q, want %q", out, tc.input)
			}
		})
	}
}

// TestHTMLToTextLeavesLongInputAlone covers input past any buffer a naive
// implementation might reach for.
func TestHTMLToTextLeavesLongInputAlone(t *testing.T) {
	input := strings.Repeat("<p>a paragraph of text</p>\n", 5000)
	out, err := runOp(t, "HTML To Text", input)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out != input {
		t.Errorf("the output is %d characters, want %d", len(out), len(input))
	}
}
