package ops

import (
	"reflect"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// Every expected output here came from the CyberChef oracle, driven through
// From Hex so the input bytes on both sides are identical.
func TestDiff(t *testing.T) {
	d := func(delim, by string, sa, sr, ss, iw bool) core.Recipe {
		return core.Recipe{{Op: "Diff", Args: []any{delim, by, sa, sr, ss, iw}}}
	}
	runCases(t, []opCase{
		{
			"char", "the quick brown fox|the quick red fox", "the quick <del>b</del>r<del>own</del><ins>ed</ins> fox",
			d("|", "Character", true, true, false, false),
		},
		{
			"char show-subtraction", "the quick brown fox|the quick red fox", "<del>b</del><del>own</del><ins>ed</ins>",
			d("|", "Character", true, true, true, false),
		},
		{
			"char add-only", "abc|abXc", "ab<ins>X</ins>c",
			d("|", "Character", true, false, false, false),
		},
		{
			"char del-only", "abc|abXc", "abc",
			d("|", "Character", false, true, false, false),
		},
		{
			"char html-escape", "a<b>c|a<d>c", "a&lt;<del>b</del><ins>d</ins>&gt;c",
			d("|", "Character", true, true, false, false),
		},
		{
			"char identical", "same|same", "same",
			d("|", "Character", true, true, false, false),
		},
		{
			"char both empty", "|", "",
			d("|", "Character", true, true, false, false),
		},
		{
			"char old empty", "|abc", "<ins>abc</ins>",
			d("|", "Character", true, true, false, false),
		},
		{
			"char new empty", "abc|", "<del>abc</del>",
			d("|", "Character", true, true, false, false),
		},
		{
			"char append", "abc|abcdef", "abc<ins>def</ins>",
			d("|", "Character", true, true, false, false),
		},
		{
			"char prepend", "def|abcdef", "<ins>abc</ins>def",
			d("|", "Character", true, true, false, false),
		},
		{
			"char astral", "a😀b|a😁b", "a<del>😀</del><ins>😁</ins>b",
			d("|", "Character", true, true, false, false),
		},
		{
			"char accents", "naïve café|naive cafe", "na<del>ï</del><ins>i</ins>ve caf<del>é</del><ins>e</ins>",
			d("|", "Character", true, true, false, false),
		},
		{
			"char transpose", "abcdef|abdcef", "ab<del>c</del>d<ins>c</ins>ef",
			d("|", "Character", true, true, false, false),
		},
		{
			"char quotes and amps", "a&\"b'c|a&\"d'c", "a&amp;&quot;<del>b</del><ins>d</ins>&#x27;c",
			d("|", "Character", true, true, false, false),
		},

		{
			"word", "the quick brown fox|the quick red fox", "the quick <del>brown</del><ins>red</ins> fox",
			d("|", "Word", true, true, false, false),
		},
		{
			"word newline token", "foo\nbar|foo bar", "foo<del>\n</del><ins> </ins>bar",
			d("|", "Word", true, true, false, false),
		},
		{
			"word crlf", "foo\r\nbar|foo\nbar", "foo<del>\r\n</del><ins>\n</ins>bar",
			d("|", "Word", true, true, false, false),
		},
		{
			"word multiple spaces", "foo   bar|foo bar", "foo<del>   </del><ins> </ins>bar",
			d("|", "Word", true, true, false, false),
		},
		{
			"word punctuation", "hello, world!|hello; world?", "hello<del>,</del><ins>;</ins> world<del>!</del><ins>?</ins>",
			d("|", "Word", true, true, false, false),
		},
		{
			"word insert word", "one two|one and two", "one <ins>and </ins>two",
			d("|", "Word", true, true, false, false),
		},
		{
			"word delete word", "one and two|one two", "one <del>and </del>two",
			d("|", "Word", true, true, false, false),
		},
		{
			"word underscore digits", "var_1 = 10|var_2 = 20", "<del>var_1</del><ins>var_2</ins> = <del>10</del><ins>20</ins>",
			d("|", "Word", true, true, false, false),
		},
		{
			"word latin extended", "élève naïve|élève simple", "élève <del>naïve</del><ins>simple</ins>",
			d("|", "Word", true, true, false, false),
		},
		{
			"word tabs", "foo\tbar|foo bar", "foo<del>\t</del><ins> </ins>bar",
			d("|", "Word", true, true, false, false),
		},

		{
			"word iw simple", "foo bar baz|foo baz", "foo <del>bar </del>baz",
			d("|", "Word", true, true, false, true),
		},
		{
			"word iw replace", "foo bar baz|foo qux baz", "foo <del>bar</del><ins>qux</ins> baz",
			d("|", "Word", true, true, false, true),
		},
		{
			"word iw newline before", "foo\nbar baz|foo baz", "foo<del>\nbar</del> baz",
			d("|", "Word", true, true, false, true),
		},
		{
			"word iw newline inserted", "foo baz|foo\nbar baz", "foo\n<ins>bar </ins>baz",
			d("|", "Word", true, true, false, true),
		},
		{
			"word iw runs of spaces", "foo   bar baz|foo  baz", "foo  <del> bar </del>baz",
			d("|", "Word", true, true, false, true),
		},
		{
			"word iw tab and newline", "foo\tbar\nbaz|foo baz", "foo <del>\tbar\n</del>baz",
			d("|", "Word", true, true, false, true),
		},
		{
			"word iw delete at start", "bar baz|baz", "<del>bar </del>baz",
			d("|", "Word", true, true, false, true),
		},
		{
			"word iw delete at end", "foo bar|foo", "foo<del> bar</del>",
			d("|", "Word", true, true, false, true),
		},
		{
			"word iw insert at start", "baz|bar baz", "<ins>bar </ins>baz",
			d("|", "Word", true, true, false, true),
		},
		{
			"word iw insert at end", "foo|foo bar", "foo <ins>bar</ins>",
			d("|", "Word", true, true, false, true),
		},
		{
			"word iw trailing space only", "the quick brown  fox|the quick red fox", "the quick <del>brown </del><ins>red</ins> fox",
			d("|", "Word", true, true, false, true),
		},
		{
			"word iw whitespace only", "   | ", " ",
			d("|", "Word", true, true, false, true),
		},
		{
			"word iw indent change", "  foo bar|foo bar", "foo bar",
			d("|", "Word", true, true, false, true),
		},
		{
			"word iw identical modulo space", "foo  bar|foo bar", "foo bar",
			d("|", "Word", true, true, false, true),
		},
		{
			"word iw punctuation", "hello , world|hello, world", "hello, world",
			d("|", "Word", true, true, false, true),
		},

		{
			"line", "line one\nline two\nline three|line one\nline 2\nline three", "line one\n<del>line two\n</del><ins>line 2\n</ins>line three",
			d("|", "Line", true, true, false, false),
		},
		{
			"line trailing newline", "a\nb\n|a\nb", "a\n<del>b\n</del><ins>b</ins>",
			d("|", "Line", true, true, false, false),
		},
		{
			"line insert", "a\nc\n|a\nb\nc\n", "a\n<ins>b\n</ins>c\n",
			d("|", "Line", true, true, false, false),
		},
		{
			"line delete", "a\nb\nc\n|a\nc\n", "a\n<del>b\n</del>c\n",
			d("|", "Line", true, true, false, false),
		},
		{
			"line crlf", "a\r\nb\r\n|a\nb\n", "<del>a\r\nb\r\n</del><ins>a\nb\n</ins>",
			d("|", "Line", true, true, false, false),
		},
		{
			"line blank lines", "a\n\nb\n|a\nb\n", "a\n<del>\n</del>b\n",
			d("|", "Line", true, true, false, false),
		},
		{
			"line leading newline", "\na\n|a\n", "<del>\n</del>a\n",
			d("|", "Line", true, true, false, false),
		},

		{
			"line iw", "a\n  b\nc|a\nb\nc", "a\nb\nc",
			d("|", "Line", true, true, false, true),
		},
		{
			"line iw indent block", "    x\n    y\n|x\ny\n", "x\ny\n",
			d("|", "Line", true, true, false, true),
		},
		{
			"line iw real change", "a\n  b\nc|a\n  B\nc", "a\n<del>  b\n</del><ins>  B\n</ins>c",
			d("|", "Line", true, true, false, true),
		},
		{
			"line iw trailing space", "a  \nb\n|a\nb\n", "a\nb\n",
			d("|", "Line", true, true, false, true),
		},

		{
			"sentence", "Hello there. How are you? Bye.|Hello there. How is you? Bye.", "Hello there. <del>How are you?</del><ins>How is you?</ins> Bye.",
			d("|", "Sentence", true, true, false, false),
		},
		{
			"sentence multi space", "One.  Two.  Three.|One.  Two point five.  Three.", "One.  <del>Two.</del><ins>Two point five.</ins>  Three.",
			d("|", "Sentence", true, true, false, false),
		},
		{
			"sentence no punctuation", "just some text|just other text", "<del>just some text</del><ins>just other text</ins>",
			d("|", "Sentence", true, true, false, false),
		},
		{
			"sentence exclaim", "Stop! Go. Wait?|Stop! Run. Wait?", "Stop! <del>Go.</del><ins>Run.</ins> Wait?",
			d("|", "Sentence", true, true, false, false),
		},
		{
			"sentence newline separated", "One.\nTwo.|One.\nThree.", "One.\n<del>Two.</del><ins>Three.</ins>",
			d("|", "Sentence", true, true, false, false),
		},
		{
			"sentence single char", "a|b", "<del>a</del><ins>b</ins>",
			d("|", "Sentence", true, true, false, false),
		},

		{
			"css", "a{color:red}|a{color:blue}", "a{color:<del>red</del><ins>blue</ins>}",
			d("|", "CSS", true, true, false, false),
		},
		{
			"css multi decl", "p { color: red; margin: 0; }|p { color: blue; margin: 0; }", "p { color: <del>red</del><ins>blue</ins>; margin: 0; }",
			d("|", "CSS", true, true, false, false),
		},
		{
			"css selector list", "h1,h2 { x: 1; }|h1,h3 { x: 1; }", "h1,<del>h2</del><ins>h3</ins> { x: 1; }",
			d("|", "CSS", true, true, false, false),
		},
		{
			"css whitespace", "a {\n  b: c;\n}|a {\n  b: d;\n}", "a {\n  b: <del>c</del><ins>d</ins>;\n}",
			d("|", "CSS", true, true, false, false),
		},

		{
			"json single-line", "{\"a\":1,\"b\":2}|{\"b\":2,\"a\":1}", "<del>{&quot;a&quot;:1,&quot;b&quot;:2}</del><ins>{&quot;b&quot;:2,&quot;a&quot;:1}</ins>",
			d("|", "JSON", true, true, false, false),
		},
		{
			"json pretty", "{\n  \"a\": 1,\n  \"b\": 2\n}|{\n  \"a\": 1,\n  \"b\": 3\n}", "{\n  &quot;a&quot;: 1,\n<del>  &quot;b&quot;: 2\n</del><ins>  &quot;b&quot;: 3\n</ins>}",
			d("|", "JSON", true, true, false, false),
		},
		{
			"json dangling comma", "{\n  \"a\": 1,\n  \"b\": 2\n}|{\n  \"a\": 1\n}", "{\n  &quot;a&quot;: 1,\n<del>  &quot;b&quot;: 2\n</del>}",
			d("|", "JSON", true, true, false, false),
		},
		{
			"json dangling comma added", "{\n  \"a\": 1\n}|{\n  \"a\": 1,\n  \"b\": 2\n}", "{\n  &quot;a&quot;: 1,\n<ins>  &quot;b&quot;: 2\n</ins>}",
			d("|", "JSON", true, true, false, false),
		},
		{
			"json array", "[\n  1,\n  2,\n  3\n]|[\n  1,\n  3\n]", "[\n  1,\n<del>  2,\n</del>  3\n]",
			d("|", "JSON", true, true, false, false),
		},
		{
			"json astral dangling comma", "{\n  \"a\": \"😀\",\n  \"b\": 2\n}|{\n  \"a\": \"😀\"\n}", "{\n  &quot;a&quot;: &quot;😀&quot;,\n<del>  &quot;b&quot;: 2\n</del>}",
			d("|", "JSON", true, true, false, false),
		},
		{
			"json nested", "{\n  \"a\": {\n    \"b\": 1\n  }\n}|{\n  \"a\": {\n    \"b\": 2\n  }\n}", "{\n  &quot;a&quot;: {\n<del>    &quot;b&quot;: 1\n</del><ins>    &quot;b&quot;: 2\n</ins>  }\n}",
			d("|", "JSON", true, true, false, false),
		},

		{
			"word show-subtraction", "alpha beta gamma|alpha delta gamma", "<del>beta</del><ins>delta</ins>",
			d("|", "Word", true, true, true, false),
		},
		{
			"line add-only", "a\nb\n|a\nx\nb\n", "a\n<ins>x\n</ins>b\n",
			d("|", "Line", true, false, false, false),
		},
		{
			"line del-only", "a\nx\nb\n|a\nb\n", "a\n<del>x\n</del>b\n",
			d("|", "Line", false, true, false, false),
		},
		{
			"sentence hide both", "One. Two.|One. Three.", "One. ",
			d("|", "Sentence", false, false, false, false),
		},

		// The default delimiter is two newlines, given escaped.
		{
			"default delimiter", "first\n\nsecond", "<del>first</del><ins>second</ins>",
			d(`\n\n`, "Word", true, true, false, false),
		},
	})
}

// Diff errors when the sample count is not exactly two (matches CyberChef).
func TestDiffSampleCount(t *testing.T) {
	if _, err := runOp(t, "Diff", "only one sample", "|", "Character", true, true, false, false); err == nil {
		t.Error("expected error for one sample")
	}
	if _, err := runOp(t, "Diff", "a|b|c", "|", "Character", true, true, false, false); err == nil {
		t.Error("expected error for three samples")
	}
}

// An unrecognised granularity is refused rather than silently defaulting.
// Argument coercion rejects it before Run sees it, so this reaches past that.
func TestDiffInvalidMode(t *testing.T) {
	if _, err := diffKindFor("Bogus", false); err == nil {
		t.Error("expected error for invalid Diff by")
	}
}

// Tokenizations transcribed from jsdiff's own tokenizers. Each granularity
// splits differently, and the splits decide what a change can be attributed to.
func TestDiffTokenizers(t *testing.T) {
	cases := []struct {
		name string
		fn   func(string) []string
		in   string
		want []string
	}{
		{"chars", diffTokenizeChars, "a😀b", []string{"a", "😀", "b"}},
		{"chars empty", diffTokenizeChars, "", nil},

		{"words", diffTokenizeWords, "foo bar baz", []string{"foo ", " bar ", " baz"}},
		{"words leading space", diffTokenizeWords, "  leading", []string{"  leading"}},
		{"words trailing space", diffTokenizeWords, "trailing  ", []string{"trailing  "}},
		{"words all space", diffTokenizeWords, "   ", []string{"   "}},
		{"words punctuation", diffTokenizeWords, "hello, world!", []string{"hello", ", ", " world", "!"}},
		{"words blank line", diffTokenizeWords, "a\n\nb", []string{"a\n\n", "\n\nb"}},
		{"words mixed space", diffTokenizeWords, "foo\tbar\nbaz", []string{"foo\t", "\tbar\n", "\nbaz"}},
		{"words accents", diffTokenizeWords, "élève naïve", []string{"élève ", " naïve"}},
		{"words single", diffTokenizeWords, "a", []string{"a"}},
		{"words empty", diffTokenizeWords, "", nil},

		{"words with space", diffTokenizeWordsWithSpace, "foo bar baz", []string{"foo", " ", "bar", " ", "baz"}},
		{"words with space leading", diffTokenizeWordsWithSpace, "  leading", []string{"  ", "leading"}},
		{"words with space trailing", diffTokenizeWordsWithSpace, "trailing  ", []string{"trailing", "  "}},
		{"words with space all space", diffTokenizeWordsWithSpace, "   ", []string{"   "}},
		{"words with space punctuation", diffTokenizeWordsWithSpace, "hello, world!", []string{"hello", ",", " ", "world", "!"}},
		{"words with space newlines", diffTokenizeWordsWithSpace, "a\n\nb", []string{"a", "\n", "\n", "b"}},
		{"words with space tab", diffTokenizeWordsWithSpace, "foo\tbar\nbaz", []string{"foo", "\t", "bar", "\n", "baz"}},
		{"words with space crlf", diffTokenizeWordsWithSpace, "a\r\nb", []string{"a", "\r\n", "b"}},
		{"words with space empty", diffTokenizeWordsWithSpace, "", nil},

		{"lines", diffTokenizeLines, "a\nb\n", []string{"a\n", "b\n"}},
		{"lines no trailing newline", diffTokenizeLines, "a\nb", []string{"a\n", "b"}},
		{"lines leading newline", diffTokenizeLines, "\na\n", []string{"\n", "a\n"}},
		{"lines crlf", diffTokenizeLines, "a\r\nb\r\n", []string{"a\r\n", "b\r\n"}},
		{"lines empty", diffTokenizeLines, "", nil},
		{"lines blank line", diffTokenizeLines, "a\n\nb\n", []string{"a\n", "\n", "b\n"}},

		{"sentences", diffTokenizeSentences, "One. Two.", []string{"One.", " ", "Two."}},
		{"sentences multi space", diffTokenizeSentences, "One.  Two.  Three.", []string{"One.", "  ", "Two.", "  ", "Three."}},
		{"sentences unpunctuated", diffTokenizeSentences, "no punctuation here", []string{"no punctuation here"}},
		{"sentences single char", diffTokenizeSentences, "a", []string{"a"}},
		{"sentences empty", diffTokenizeSentences, "", nil},
		{"sentences exclaim", diffTokenizeSentences, "Hi! Bye?", []string{"Hi!", " ", "Bye?"}},
		{"sentences newline", diffTokenizeSentences, "One.\nTwo.", []string{"One.", "\n", "Two."}},

		{"css", diffTokenizeCSS, "a{color:red}", []string{"a", "{", "color", ":", "red", "}", ""}},
		{
			"css spaced", diffTokenizeCSS, "p { color: red; }",
			[]string{"p", " ", "", "{", "", " ", "color", ":", "", " ", "red", ";", "", " ", "", "}", ""},
		},
		{"css empty", diffTokenizeCSS, "", []string{""}},
		{"css commas", diffTokenizeCSS, "h1,h2", []string{"h1", ",", "h2"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.fn(c.in); !reflect.DeepEqual(got, c.want) {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

// The whitespace helpers behind the word-mode tidy-up. They work on code
// points, so a multi-byte space is never cut in half.
func TestDiffWhitespaceHelpers(t *testing.T) {
	t.Run("leading and trailing", func(t *testing.T) {
		cases := []struct{ in, lead, trail string }{
			{"", "", ""},
			{"abc", "", ""},
			{"  abc", "  ", ""},
			{"abc  ", "", "  "},
			{" \t\nabc\n\t ", " \t\n", "\n\t "},
			{"   ", "   ", "   "},
			{"\u00a0a\u2028", "\u00a0", "\u2028"},
		}
		for _, c := range cases {
			if got := diffLeadingWS(c.in); got != c.lead {
				t.Errorf("leading %q: got %q, want %q", c.in, got, c.lead)
			}
			if got := diffTrailingWS(c.in); got != c.trail {
				t.Errorf("trailing %q: got %q, want %q", c.in, got, c.trail)
			}
		}
	})

	t.Run("common prefix", func(t *testing.T) {
		cases := []struct{ a, b, want string }{
			{"", "", ""},
			{"abc", "abd", "ab"},
			{"abc", "xyz", ""},
			{"ab", "abcd", "ab"},
			{"abcd", "ab", "ab"},
			{"\u2000x", "\u2028x", ""}, // shared leading UTF-8 bytes, different code points
		}
		for _, c := range cases {
			if got := diffLongestCommonPrefix(c.a, c.b); got != c.want {
				t.Errorf("prefix(%q, %q): got %q, want %q", c.a, c.b, got, c.want)
			}
		}
	})

	t.Run("common suffix", func(t *testing.T) {
		cases := []struct{ a, b, want string }{
			{"", "", ""},
			{"abc", "", ""},
			{"", "abc", ""},
			{"abc", "xbc", "bc"},
			{"abc", "xyz", ""},
			{"bc", "abc", "bc"},
			{"abc", "bc", "bc"},
			{"x\u2000", "y\u2028", ""}, // as above, at the other end
		}
		for _, c := range cases {
			if got := diffLongestCommonSuffix(c.a, c.b); got != c.want {
				t.Errorf("suffix(%q, %q): got %q, want %q", c.a, c.b, got, c.want)
			}
		}
	})

	t.Run("maximum overlap", func(t *testing.T) {
		cases := []struct{ a, b, want string }{
			{"", "", ""},
			{"  ", "  ", "  "},
			{"  ", " ", " "},
			{" ", "  ", " "},
			{"abc", "cde", "c"},
			{"abc", "xyz", ""},
			{"aaa", "aa", "aa"},
		}
		for _, c := range cases {
			if got := diffMaximumOverlap(c.a, c.b); got != c.want {
				t.Errorf("overlap(%q, %q): got %q, want %q", c.a, c.b, got, c.want)
			}
		}
	})
}

// A deletion with no surrounding keeps is left exactly as the algorithm
// produced it; there is no adjacent text to dedupe whitespace against.
func TestDiffWordDedupeWithoutKeeps(t *testing.T) {
	changes := []diffChange{{Text: "foo", Removed: true}}
	got := diffDedupeWordWhitespace(changes)
	if len(got) != 1 || got[0].Text != "foo" {
		t.Errorf("got %+v, want the deletion unchanged", got)
	}
}

// The render honours each show flag independently.
func TestDiffRender(t *testing.T) {
	changes := []diffChange{
		{Text: "a"},
		{Text: "b", Removed: true},
		{Text: "c", Added: true},
	}
	cases := []struct {
		sa, sr, ss bool
		want       string
	}{
		{true, true, false, "a<del>b</del><ins>c</ins>"},
		{false, true, false, "a<del>b</del>"},
		{true, false, false, "a<ins>c</ins>"},
		{true, true, true, "<del>b</del><ins>c</ins>"},
		{false, false, true, ""},
	}
	for _, c := range cases {
		if got := diffRender(changes, c.sa, c.sr, c.ss); got != c.want {
			t.Errorf("render(%v,%v,%v): got %q, want %q", c.sa, c.sr, c.ss, got, c.want)
		}
	}
}

// Token length is counted in sixteen-bit units, the way JavaScript counts it,
// so a character outside the basic plane counts twice.
func TestDiffTokenLength(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"abc", 3},
		{"é", 1},
		{"\U0001F600", 2},
		{"a\U0001F600b", 4},
	}
	for _, c := range cases {
		if got := diffTokenLength(c.in); got != c.want {
			t.Errorf("length %q: got %d, want %d", c.in, got, c.want)
		}
	}
}
