package opsutil

import (
	"slices"
	"testing"
)

func TestSplitTopLevel(t *testing.T) {
	cases := []struct {
		name string
		in   string
		sep  byte
		want []string
	}{
		{"no separator", "abc", ',', []string{"abc"}},
		{"empty input", "", ',', []string{""}},
		{"plain split", "a,b,c", ',', []string{"a", "b", "c"}},
		{"empty fields", "a,,b", ',', []string{"a", "", "b"}},
		{"trailing separator", "a,b,", ',', []string{"a", "b", ""}},
		{"leading separator", ",a", ',', []string{"", "a"}},

		{"brackets protect", "a[1,2],b", ',', []string{"a[1,2]", "b"}},
		{"parens protect", "f(x,y),b", ',', []string{"f(x,y)", "b"}},
		{"nested brackets", "a[b[1,2],3],c", ',', []string{"a[b[1,2],3]", "c"}},
		{"mixed nesting", "a[f(1,2)],b", ',', []string{"a[f(1,2)]", "b"}},

		{"double quotes protect", `a["x,y"],b`, ',', []string{`a["x,y"]`, "b"}},
		{"single quotes protect", `a['x,y'],b`, ',', []string{`a['x,y']`, "b"}},
		{"quote hides a bracket", `a["]"],b`, ',', []string{`a["]"]`, "b"}},
		{"quote of the other kind is literal", `a["it's"],b`, ',', []string{`a["it's"]`, "b"}},

		{"colon separator", "a:b", ':', []string{"a", "b"}},
		{"separator inside quotes only", `":"`, ':', []string{`":"`}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := SplitTopLevel(c.in, c.sep)
			if !slices.Equal(got, c.want) {
				t.Errorf("SplitTopLevel(%q, %q) = %q, want %q", c.in, c.sep, got, c.want)
			}
		})
	}
}

// An unbalanced closing bracket drives the depth counter negative, so a
// separator after it is no longer at depth 0 and the rest of the string stays
// in one field. Callers pass syntactically valid input; this pins down what
// happens when they do not.
func TestSplitTopLevelUnbalanced(t *testing.T) {
	if got := SplitTopLevel("a],b", ','); !slices.Equal(got, []string{"a],b"}) {
		t.Errorf("got %q, want %q", got, []string{"a],b"})
	}
	if got := SplitTopLevel("a[,b", ','); !slices.Equal(got, []string{"a[,b"}) {
		t.Errorf("got %q, want %q", got, []string{"a[,b"})
	}
}

// TestEscapeHTML covers CyberChef's Utils.escapeHtml, which is not Go's
// html.EscapeString: it also escapes backticks, leaves forward slashes alone,
// and maps a null byte to U+E000 so it stays visible in rendered output.
func TestEscapeHTML(t *testing.T) {
	cases := []struct{ in, want string }{
		{"a&b", "a&amp;b"},
		{"<i>", "&lt;i&gt;"},
		{`say "hi"`, "say &quot;hi&quot;"},
		{"it's", "it&#x27;s"},
		{"`cmd`", "&#x60;cmd&#x60;"},
		{"nul\x00here", "nul\ue000here"},
		{"plain / text", "plain / text"},
		{"", ""},
	}
	for _, c := range cases {
		if got := EscapeHTML(c.in); got != c.want {
			t.Errorf("EscapeHTML(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
