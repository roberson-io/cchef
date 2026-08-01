package jsparse

// Scanner tests: token (type, raw) sequences verified against esprima.tokenize
// (see scratchpad/esptok.mjs).

import "testing"

var jsTokenName = map[int]string{
	tkBooleanLiteral: "Boolean", tkEOF: "<end>", tkIdentifier: "Identifier",
	tkKeyword: "Keyword", tkNullLiteral: "Null", tkNumericLiteral: "Numeric",
	tkPunctuator: "Punctuator", tkStringLiteral: "String", tkTemplate: "Template",
}

// lexAll tokenizes src into [type, raw] pairs, recovering scanner panics.
func lexAll(t *testing.T, src string) (pairs [][2]string, err error) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			if se, ok := r.(*jsSyntaxError); ok {
				err = se
				return
			}
			panic(r)
		}
	}()
	sc := newJSScanner(src)
	for {
		sc.scanComments()
		tok := sc.lex()
		if tok.typ == tkEOF {
			return pairs, nil
		}
		pairs = append(pairs, [2]string{jsTokenName[tok.typ], sc.rawToken(tok)})
	}
}

func TestJSScanner(t *testing.T) {
	cases := []struct {
		src  string
		want [][2]string
	}{
		{"1+2", [][2]string{{"Numeric", "1"}, {"Punctuator", "+"}, {"Numeric", "2"}}},
		{"x = 3;", [][2]string{{"Identifier", "x"}, {"Punctuator", "="}, {"Numeric", "3"}, {"Punctuator", ";"}}},
		{"foo.bar", [][2]string{{"Identifier", "foo"}, {"Punctuator", "."}, {"Identifier", "bar"}}},
		{"0xFF 0b101 0o17 .5 1e3 1.5e-2 010", [][2]string{
			{"Numeric", "0xFF"},
			{"Numeric", "0b101"},
			{"Numeric", "0o17"},
			{"Numeric", ".5"},
			{"Numeric", "1e3"},
			{"Numeric", "1.5e-2"},
			{"Numeric", "010"},
		}},
		{"'a\\n b' \"c\\td\"", [][2]string{{"String", "'a\\n b'"}, {"String", "\"c\\td\""}}},
		{"true false null undefined", [][2]string{
			{"Boolean", "true"}, {"Boolean", "false"}, {"Null", "null"}, {"Identifier", "undefined"},
		}},
		{"a && b || c", [][2]string{
			{"Identifier", "a"}, {"Punctuator", "&&"}, {"Identifier", "b"}, {"Punctuator", "||"}, {"Identifier", "c"},
		}},
		{"!x", [][2]string{{"Punctuator", "!"}, {"Identifier", "x"}}},
		{"a === b !== c", [][2]string{
			{"Identifier", "a"}, {"Punctuator", "==="}, {"Identifier", "b"}, {"Punctuator", "!=="}, {"Identifier", "c"},
		}},
		{"( ) { } [ ] ; , : ? ~", [][2]string{
			{"Punctuator", "("},
			{"Punctuator", ")"},
			{"Punctuator", "{"},
			{"Punctuator", "}"},
			{"Punctuator", "["},
			{"Punctuator", "]"},
			{"Punctuator", ";"},
			{"Punctuator", ","},
			{"Punctuator", ":"},
			{"Punctuator", "?"},
			{"Punctuator", "~"},
		}},
		{"return function while for if else", [][2]string{
			{"Keyword", "return"},
			{"Keyword", "function"},
			{"Keyword", "while"},
			{"Keyword", "for"},
			{"Keyword", "if"},
			{"Keyword", "else"},
		}},
		{"// line\n42 /* block */ 7", [][2]string{{"Numeric", "42"}, {"Numeric", "7"}}},
	}
	for _, c := range cases {
		got, err := lexAll(t, c.src)
		if err != nil {
			t.Errorf("%q: unexpected error %v", c.src, err)
			continue
		}
		if len(got) != len(c.want) {
			t.Errorf("%q: got %v tokens, want %v", c.src, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("%q token %d: got %v, want %v", c.src, i, got[i], c.want[i])
			}
		}
	}
}
