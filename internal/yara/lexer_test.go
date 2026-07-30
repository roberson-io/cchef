package yara

import (
	"strings"
	"testing"
)

// kinds runs the lexer over src and returns what it found, as "kind:text" for
// each token, so a whole scan can be written on one line.
func kinds(t *testing.T, src string) []string {
	t.Helper()
	lx := newLexer(src)
	var out []string
	for {
		tok, err := lx.next()
		if err != nil {
			t.Fatalf("lexing %q: %v", src, err)
		}
		if tok.kind == tokenEOF {
			return out
		}
		out = append(out, tok.kind.String()+":"+tok.text)
	}
}

func TestLexerRuleShape(t *testing.T) {
	got := kinds(t, `rule R : tag { meta: a = "x" strings: $a = "hi" condition: $a }`)
	want := []string{
		"keyword:rule", "identifier:R", "punctuation::", "identifier:tag", "punctuation:{",
		"keyword:meta", "punctuation::", "identifier:a", "punctuation:=", "text:x",
		"keyword:strings", "punctuation::", "string identifier:$a", "punctuation:=", "text:hi",
		"keyword:condition", "punctuation::", "string identifier:$a", "punctuation:}",
	}
	assertTokens(t, got, want)
}

func TestLexerStringIdentifiers(t *testing.T) {
	got := kinds(t, `$a #a @a !a $ # @ ! $a*`)
	want := []string{
		"string identifier:$a", "string count:#a", "string offset:@a", "string length:!a",
		"string identifier:$", "string count:#", "string offset:@", "string length:!",
		"string identifier:$a", "punctuation:*",
	}
	assertTokens(t, got, want)
}

func TestLexerNumbers(t *testing.T) {
	got := kinds(t, `0 42 0x10 0o17 1KB 2MB 3.5 -7`)
	want := []string{
		"integer:0", "integer:42", "integer:16", "integer:15",
		"integer:1024", "integer:2097152", "double:3.5", "punctuation:-", "integer:7",
	}
	assertTokens(t, got, want)
}

func TestLexerTextStringEscapes(t *testing.T) {
	cases := []struct{ src, want string }{
		{`"plain"`, "plain"},
		{`"a\tb"`, "a\tb"},
		{`"a\nb"`, "a\nb"},
		{`"a\rb"`, "a\rb"},
		{`"a\\b"`, `a\b`},
		{`"a\"b"`, `a"b`},
		{`"a\x41b"`, "aAb"},
		{`"\x00"`, "\x00"},
	}
	for _, c := range cases {
		t.Run(c.src, func(t *testing.T) {
			lx := newLexer(c.src)
			tok, err := lx.next()
			if err != nil {
				t.Fatalf("lex: %v", err)
			}
			if tok.kind != tokenText || tok.text != c.want {
				t.Errorf("got %v %q, want a text string %q", tok.kind, tok.text, c.want)
			}
		})
	}
}

func TestLexerHexAndRegex(t *testing.T) {
	got := kinds(t, `$a = { 68 ?? [1-3] ( 61 | 62 ) } /ab+c/is`)
	want := []string{
		"string identifier:$a", "punctuation:=",
		"hex string:68 ?? [1-3] ( 61 | 62 )", "regex:ab+c",
	}
	assertTokens(t, got, want)
}

// TestLexerRegexFlags covers the letters that may follow a regex, which say
// whether it ignores case and whether a dot reaches across a newline.
func TestLexerRegexFlags(t *testing.T) {
	for _, c := range []struct{ src, body, flags string }{
		{`/ab+c/is`, "ab+c", "is"},
		{`/ab/i`, "ab", "i"},
		{`/ab/`, "ab", ""},
		{`/a\/b/`, `a\/b`, ""},
		{`/a[/]b/`, `a[`, ""},
	} {
		t.Run(c.src, func(t *testing.T) {
			tok, err := newLexer(c.src).next()
			if err != nil {
				t.Fatalf("lex: %v", err)
			}
			if tok.kind != tokenRegex || tok.text != c.body || tok.flags != c.flags {
				t.Errorf("got %v %q flags %q, want a regex %q flags %q",
					tok.kind, tok.text, tok.flags, c.body, c.flags)
			}
		})
	}
}

func TestLexerComments(t *testing.T) {
	got := kinds(t, "rule /* block\ncomment */ R // line comment\n{ }")
	want := []string{"keyword:rule", "identifier:R", "punctuation:{", "punctuation:}"}
	assertTokens(t, got, want)
}

func TestLexerOperators(t *testing.T) {
	got := kinds(t, `== != < <= > >= << >> .. and or not + - * \ % & | ^ ~`)
	want := []string{
		"punctuation:==", "punctuation:!=", "punctuation:<", "punctuation:<=", "punctuation:>", "punctuation:>=",
		"punctuation:<<", "punctuation:>>", "punctuation:..",
		"keyword:and", "keyword:or", "keyword:not",
		"punctuation:+", "punctuation:-", "punctuation:*", `punctuation:\`, "punctuation:%",
		"punctuation:&", "punctuation:|", "punctuation:^", "punctuation:~",
	}
	assertTokens(t, got, want)
}

// TestLexerNumbersInOtherCases covers the letters a base may be written with in
// either case, and the sizes a number may be given in.
func TestLexerNumbersInOtherCases(t *testing.T) {
	got := kinds(t, `0X1F 0O7 0xff`)
	assertTokens(t, got, []string{"integer:31", "integer:7", "integer:255"})
}

// TestLexerRejectsBadNumbers covers numbers that open as one base and then hold
// nothing, or that will not fit.
func TestLexerRejectsBadNumbers(t *testing.T) {
	// The last is a double too large to hold; YARA writes no exponents, so the
	// only way to overflow one is to write the digits out.
	for _, src := range []string{`0x `, `0o `, `99999999999999999999`, strings.Repeat("9", 400) + ".5"} {
		t.Run(src, func(t *testing.T) {
			lx := newLexer(src)
			for {
				tok, err := lx.next()
				if err != nil {
					return
				}
				if tok.kind == tokenEOF {
					t.Fatal("lexed a number that should have been refused")
				}
			}
		})
	}
}

// TestLexerRejectsATrailingBackslash covers a string whose last character is the
// start of an escape that never arrives.
func TestLexerRejectsATrailingBackslash(t *testing.T) {
	if _, err := newLexer(`"a\`).next(); err == nil {
		t.Fatal("lexed a string ending in a backslash")
	}
}

// TestLexerEdges covers the corners the cases above leave: the name for the end
// of input, an escape and a regex that stop dead, and a hex string written over
// several lines.
func TestLexerEdges(t *testing.T) {
	if got := tokenEOF.String(); got != "end of input" {
		t.Errorf("end of input is named %q", got)
	}
	for _, src := range []string{`"a\x`, "/ab\nc/"} {
		if _, err := newLexer(src).next(); err == nil {
			t.Errorf("lexed %q, which stops part way through", src)
		}
	}

	lx := newLexer("$a = {\n68 65\n6c\n} rule")
	hex, err := lx.next()
	if err != nil {
		t.Fatalf("lex: %v", err)
	}
	for hex.kind != tokenHexString {
		if hex, err = lx.next(); err != nil {
			t.Fatalf("lex: %v", err)
		}
	}
	if hex.text != "68 65\n6c" {
		t.Errorf("hex body is %q", hex.text)
	}
	after, err := lx.next()
	if err != nil {
		t.Fatalf("lex: %v", err)
	}
	if after.line != 4 {
		t.Errorf("the token after a three-line hex string is on line %d, want 4", after.line)
	}
}

func TestLexerKeywordsAreWholeWords(t *testing.T) {
	// "android" opens with "and" but is an identifier, not the operator.
	got := kinds(t, `android and not_a_keyword not`)
	want := []string{
		"identifier:android", "keyword:and", "identifier:not_a_keyword", "keyword:not",
	}
	assertTokens(t, got, want)
}

func TestLexerTracksLines(t *testing.T) {
	lx := newLexer("rule\n\nR\n{")
	for _, want := range []int{1, 3, 4} {
		tok, err := lx.next()
		if err != nil {
			t.Fatalf("lex: %v", err)
		}
		if tok.line != want {
			t.Errorf("token %q is on line %d, want %d", tok.text, tok.line, want)
		}
	}
}

func TestLexerRejectsBadInput(t *testing.T) {
	cases := []struct{ name, src string }{
		{"an unterminated text string", `"abc`},
		{"an unterminated regex", `/abc`},
		{"an unterminated hex string", `$a = { 68 65`},
		{"an unterminated block comment", `/* nope`},
		{"a stray character", "rule R \x01"},
		{"a bad escape", `"a\qb"`},
		{"a short hex escape", `"a\x4"`},
		{"a text string with a newline", "\"a\nb\""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			lx := newLexer(c.src)
			for {
				tok, err := lx.next()
				if err != nil {
					return // refused, which is what this checks
				}
				if tok.kind == tokenEOF {
					t.Fatal("lexed something that should have been refused")
				}
			}
		})
	}
}

// assertTokens compares two token lists and says where they first differ.
func assertTokens(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) == len(want) {
		same := true
		for i := range got {
			if got[i] != want[i] {
				same = false
				break
			}
		}
		if same {
			return
		}
	}
	t.Errorf("got  [%s]\nwant [%s]", strings.Join(got, " "), strings.Join(want, " "))
}
