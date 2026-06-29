package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// Diff outputs verified against the CyberChef-server oracle (sample delimiter "|").
func TestDiff(t *testing.T) {
	d := func(in, by string, sa, sr, ss, iw bool) core.Recipe {
		return core.Recipe{{Op: "Diff", Args: []any{"|", by, sa, sr, ss, iw}}}
	}
	runCases(t, []opCase{
		{"char", "the quick brown fox|the quick red fox",
			"the quick <del>b</del>r<del>own</del><ins>ed</ins> fox",
			d("the quick brown fox|the quick red fox", "Character", true, true, false, false)},
		{"char show-subtraction", "the quick brown fox|the quick red fox",
			"<del>b</del><del>own</del><ins>ed</ins>",
			d("the quick brown fox|the quick red fox", "Character", true, true, true, false)},
		{"char add-only", "abc|abXc", "ab<ins>X</ins>c",
			d("abc|abXc", "Character", true, false, false, false)},
		{"char del-only", "abc|abXc", "abc",
			d("abc|abXc", "Character", false, true, false, false)},
		{"char html-escape", "a<b>c|a<d>c", "a&lt;<del>b</del><ins>d</ins>&gt;c",
			d("a<b>c|a<d>c", "Character", true, true, false, false)},
		{"word", "the quick brown fox|the quick red fox",
			"the quick <del>brown</del><ins>red</ins> fox",
			d("the quick brown fox|the quick red fox", "Word", true, true, false, false)},
		{"sentence", "Hello there. How are you? Bye.|Hello there. How is you? Bye.",
			"Hello there. <del>How are you?</del><ins>How is you?</ins> Bye.",
			d("Hello there. How are you? Bye.|Hello there. How is you? Bye.", "Sentence", true, true, false, false)},
		{"line", "line one\nline two\nline three|line one\nline 2\nline three",
			"line one\n<del>line two\n</del><ins>line 2\n</ins>line three",
			d("line one\nline two\nline three|line one\nline 2\nline three", "Line", true, true, false, false)},
		{"css", "a{color:red}|a{color:blue}", "a{color:<del>red</del><ins>blue</ins>}",
			d("a{color:red}|a{color:blue}", "CSS", true, true, false, false)},
		// JSON mode behaves as a line diff over the raw input (no canonicalisation).
		{"json single-line", `{"a":1,"b":2}|{"b":2,"a":1}`,
			"<del>{&quot;a&quot;:1,&quot;b&quot;:2}</del><ins>{&quot;b&quot;:2,&quot;a&quot;:1}</ins>",
			d(`{"a":1,"b":2}|{"b":2,"a":1}`, "JSON", true, true, false, false)},
		{"json pretty", "{\n  \"a\": 1,\n  \"b\": 2\n}|{\n  \"a\": 1,\n  \"b\": 3\n}",
			"{\n  &quot;a&quot;: 1,\n<del>  &quot;b&quot;: 2\n</del><ins>  &quot;b&quot;: 3\n</ins>}",
			d("{\n  \"a\": 1,\n  \"b\": 2\n}|{\n  \"a\": 1,\n  \"b\": 3\n}", "JSON", true, true, false, false)},
		// Line + ignore whitespace compares trimmed lines.
		{"line ignore-ws", "a\n  b\nc|a\nb\nc", "a\nb\nc",
			d("a\n  b\nc|a\nb\nc", "Line", true, true, false, true)},
	})
}

// Diff errors when the sample count is not exactly two (matches CyberChef).
func TestDiffSampleCount(t *testing.T) {
	if _, err := runOp(t, "Diff", "only one sample", "|", "Character", true, true, false, false); err == nil {
		t.Error("expected error for incorrect number of samples")
	}
}

// Word + ignore whitespace. cchef collapses whitespace-only changes; this differs from
// CyberChef's jsdiff, which attaches the trailing space to the deleted word
// ("<del>brown </del>"). This asserts cchef's own (documented) behaviour.
func TestDiffWordIgnoreWhitespace(t *testing.T) {
	out, err := runOp(t, "Diff", "the quick brown  fox|the quick red fox", "|", "Word", true, true, false, true)
	if err != nil {
		t.Fatal(err)
	}
	if want := "the quick <del>brown</del><ins>red</ins> fox"; out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}
