package jsparse

// Parser tests: cchef's AST output must byte-match esprima's
// JSON.stringify(parseScript(src), null, 2). Golden vectors in
// testdata/jsparser_ast.jsonl are generated from the exact esprima package
// (scratchpad/espast.mjs / genast.mjs).

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/jsonval"
)

// TestJSParseErrors exercises the parser's error and not-yet-supported branches:
// each source must fail with an error whose message contains the substring.
func TestJSParseErrors(t *testing.T) {
	cases := []struct {
		src, want string
	}{
		{"const x;", "Missing initializer in const declaration"},
		{"continue;", "Illegal continue statement"},
		{"break;", "Illegal break statement"},
		{"x: x: ;", "Label 'x' has already been declared"},
		{"a: for (;;) continue b;", "Undefined label 'b'"},
		{"a: for (;;) break b;", "Undefined label 'b'"},
		{"switch (x) { default: ; default: ; }", "More than one default clause in switch statement"},
		{"throw\nx;", "Illegal newline after throw"},
		{"try { a; }", "Missing catch or finally after try"},
		{"return;", "Illegal return statement"},
		{"for (1 in b);", "Invalid left-hand side in for-in"},
		{"for (1 of b);", "Invalid left-hand side in for-loop"},
		{"function f(...a = 1) {}", "Unexpected token ="},
		{"function f(...a, b) {}", "Rest parameter must be last formal parameter"},
		{"1 = 2;", "Invalid left-hand side in assignment"},
		{"1++;", "Invalid left-hand side in assignment"},
		{"++1;", "Invalid left-hand side in assignment"},
		{"new.target", "Unexpected identifier"},
		{"function f() { new.foo; }", "Unexpected identifier"},
		{"await x", "Unexpected identifier"},                     // await outside async is an identifier
		{"for await (const x of y) {}", "Unexpected identifier"}, // for-await not in script mode
		{"async function* g() {}", "Unexpected token *"},         // async generators unsupported by esprima
		{"({async *m() {}})", "Unexpected token *"},              // async generator method unsupported
		{"class X { static prototype() {} }", "Classes may not have static property named prototype"},
		{"class X { constructor() {} constructor() {} }", "A class may only have one constructor"},
		{"class X { async constructor() {} }", "Class constructor may not be an async method"},
		{"class X { get constructor() {} }", "Class constructor may not be an accessor"},
		// Scanner error paths.
		{"'abc", "Unexpected token ILLEGAL"},                               // unterminated string
		{"'a\nb'", "Unexpected token ILLEGAL"},                             // line terminator in string
		{"'\\x'", "Invalid hexadecimal escape sequence"},                   // bad \x escape
		{"'\\8'", "Unexpected token ILLEGAL"},                              // \8 escape
		{"'\\u{}'", "Unexpected token ILLEGAL"},                            // empty \u{}
		{"'\\u{110000}'", "Unexpected token ILLEGAL"},                      // code point out of range
		{"'\\uZZ'", "Unexpected token ILLEGAL"},                            // bad \u escape
		{"/* abc", "Unexpected token ILLEGAL"},                             // unterminated block comment
		{"/abc", "Invalid regular expression: missing /"},                  // unterminated regex
		{"/a\\\n/", "Invalid regular expression: missing /"},               // line terminator after regex backslash
		{"/a\nb/", "Invalid regular expression: missing /"},                // line terminator in regex
		{"0x", "Unexpected token ILLEGAL"},                                 // empty hex literal
		{"0b", "Unexpected token ILLEGAL"},                                 // empty binary literal
		{"0b12", "Unexpected token ILLEGAL"},                               // digit after binary literal
		{"0o", "Unexpected token ILLEGAL"},                                 // empty octal literal
		{"0o18", "Unexpected token ILLEGAL"},                               // digit after octal literal
		{"1e+", "Unexpected token ILLEGAL"},                                // exponent without digits
		{"3in", "Unexpected token ILLEGAL"},                                // identifier immediately after number
		{"@", "Unexpected token ILLEGAL"},                                  // invalid punctuator
		{"\\u0069f (x) y;", "Keyword must not contain escaped characters"}, // escaped keyword
		{"var \\u{110000} = 1;", "Unexpected token ILLEGAL"},               // code point out of range in identifier
		{"`abc", "Unexpected token ILLEGAL"},                               // unterminated template
		{"`\\01`", "Octal literals are not allowed in template strings."},  // \0 + digit in template
		{"`\\1`", "Octal literals are not allowed in template strings."},   // octal escape in template
		{"`\\x`", "Invalid hexadecimal escape sequence"},                   // bad \x in template
		// Parser error paths.
		{"(1", "Unexpected end of input"},                          // expect ) hits EOF
		{"do x;", "Unexpected end of input"},                       // expectKeyword while hits EOF
		{"x y", "Unexpected identifier"},                           // consumeSemicolon
		{"({a = 1})", "Unexpected token ="},                        // cover-initialized-name error
		{"[{a = 1}, b]", "Unexpected token ="},                     // cover-initialized-name restored across siblings
		{"]", "Unexpected token ]"},                                // punctuator in primary position
		{"typeof if", "Unexpected token if"},                       // keyword in primary position
		{"1 +", "Unexpected end of input"},                         // EOF in primary position
		{"()", "Unexpected end of input"},                          // empty parens without arrow
		{"(...a)", "Unexpected end of input"},                      // rest params without arrow
		{"(a, ...b)", "Unexpected end of input"},                   // rest in group without arrow
		{"(a.b, ...c) => d", "Unexpected token ..."},               // rest after non-binding element
		{"(a.b) => c", "Unexpected token =>"},                      // non-binding arrow head
		{"a.", "Unexpected end of input"},                          // non-identifier-name after member dot
		{"1 => 2", "Unexpected token =>"},                          // literal cannot be arrow params
		{"x\n=> y", "Unexpected token =>"},                         // newline before =>
		{"{", "Unexpected end of input"},                           // EOF inside block
		{"import x", "Line 1: Unexpected token"},                   // modules invalid in script mode
		{"export x", "Line 1: Unexpected token"},                   // modules invalid in script mode
		{"super.x", "Unexpected reserved word"},                    // super outside a method
		{"enum", "Unexpected reserved word"},                       // future reserved word
		{"let 5", "Unexpected number"},                             // let identifier then a number
		{"let[0]", "Unexpected number"},                            // `let [` parses as a lexical declaration
		{"({: 1})", "Unexpected token :"},                          // invalid object key
		{"({1})", "Unexpected token }"},                            // numeric key without value
		{"var {a};", "Unexpected token ;"},                         // destructuring var without initializer
		{"try {} catch () {}", "Unexpected token )"},               // empty catch binding
		{"function* g() { var yield; }", "Unexpected token yield"}, // yield binding in generator
		{"({get x(v){}})", "Getter must not have any formal parameters"},
		{"({set x(){}})", "Setter must have exactly one formal parameter"},
		{"({set x(a, b){}})", "Setter must have exactly one formal parameter"},
		{"({set x(...a){}})", "Setter function argument must not be a rest parameter"},
		{"({*})", "Unexpected token }"},                            // generator prop with no key
		{"class X { x }", "Unexpected token }"},                    // class field without body
		{"class X { m() { super; } }", "Unexpected token ;"},       // super not followed by ./[/(
		{"class X { m() { new super(); } }", "Unexpected token ("}, // super as new callee, not member
		{"class X { static 'prototype'(){} }", "Classes may not have static property named prototype"},
		{"\\x", "Unexpected token ILLEGAL"},                   // identifier escape not \u
		{"var \\u0020;", "Unexpected token ILLEGAL"},          // \u escape decodes to non-identifier char
		{"function* g(){ yield if; }", "Unexpected token if"}, // non-expression keyword after yield
	}
	for _, c := range cases {
		_, err := Parse(c.src)
		if err == nil {
			t.Errorf("parse %q: expected error containing %q, got nil", c.src, c.want)
			continue
		}
		if !strings.Contains(err.Error(), c.want) {
			t.Errorf("parse %q: got %q, want substring %q", c.src, err.Error(), c.want)
		}
	}
}

// TestJSParserHelpers exercises pure node/token helpers directly, including the
// not-found / wrong-type guard branches that are awkward to reach via Parse.
func TestJSParserHelpers(t *testing.T) {
	noType := jsonval.Object{{K: "name", V: "x"}}
	if got := NodeType(noType); got != "" {
		t.Errorf("NodeType(no type field) = %q, want empty", got)
	}
	if got := NodeType("not-an-object"); got != "" {
		t.Errorf("NodeType(non-object) = %q, want empty", got)
	}

	if jsIsIdentifierName(jsToken{typ: tkNumericLiteral}) {
		t.Error("jsIsIdentifierName(numeric) = true, want false")
	}
	if !jsIsIdentifierName(jsToken{typ: tkKeyword}) {
		t.Error("jsIsIdentifierName(keyword) = false, want true")
	}

	descCases := []struct {
		tok  jsToken
		want string
	}{
		{jsToken{typ: tkEOF}, jsMsgUnexpectedEOS},
		{jsToken{typ: tkNumericLiteral}, "Unexpected number"},
		{jsToken{typ: tkStringLiteral}, "Unexpected string"},
		{jsToken{typ: tkIdentifier}, "Unexpected identifier"},
		{jsToken{typ: tkPunctuator, value: "+"}, "Unexpected token +"},
		{jsToken{typ: tkPunctuator, value: ""}, "Unexpected token ILLEGAL"},
	}
	for _, c := range descCases {
		if got := jsUnexpectedDesc(c.tok); got != c.want {
			t.Errorf("jsUnexpectedDesc(%v) = %q, want %q", c.tok, got, c.want)
		}
	}

	obj := jsonval.Object{{K: "type", V: "X"}}
	if got := FieldSlice(obj, "missing"); got != nil {
		t.Errorf("FieldSlice(missing) = %v, want nil", got)
	}
	if got := Field(obj, "missing"); got != nil {
		t.Errorf("Field(missing) = %v, want nil", got)
	}
	if got := Field("not-an-object", "type"); got != nil {
		t.Errorf("Field(non-object) = %v, want nil", got)
	}
	if jsIsPropertyKey("not-a-node", "x") {
		t.Error("jsIsPropertyKey(non-node) = true, want false")
	}
	litKey := jsonval.Object{{K: "type", V: "Literal"}, {K: "value", V: 1.0}}
	if jsIsPropertyKey(litKey, "constructor") {
		t.Error("jsIsPropertyKey(numeric Literal) = true, want false")
	}
	if got := IdentName(obj); got != "" {
		t.Errorf("IdentName(no name) = %q, want empty", got)
	}
	if jsFieldIsNil(obj, "missing") {
		t.Error("jsFieldIsNil(missing) = true, want false")
	}
	if FieldBool(obj, "missing") {
		t.Error("FieldBool(missing) = true, want false")
	}
	if jsHasDirective("not-an-object") {
		t.Error("jsHasDirective(non-object) = true, want false")
	}

	p := newJSParser("x")
	if p.qualifiedPropertyName(jsToken{typ: tkPunctuator, value: "("}) {
		t.Error("qualifiedPropertyName(punctuator '(') = true, want false")
	}
	if p.qualifiedPropertyName(jsToken{typ: tkEOF}) {
		t.Error("qualifiedPropertyName(EOF) = true, want false")
	}
	if !p.qualifiedPropertyName(jsToken{typ: tkPunctuator, value: "["}) {
		t.Error("qualifiedPropertyName(punctuator '[') = false, want true")
	}

	ranges := []jsUniRange{{lo: 0x100, hi: 0x200}}
	if jsUniIn(ranges, 0x300) {
		t.Error("jsUniIn(miss above) = true, want false")
	}
	if jsUniIn(ranges, 0x50) {
		t.Error("jsUniIn(miss below) = true, want false")
	}
	if !jsUniIn(ranges, 0x150) {
		t.Error("jsUniIn(hit) = false, want true")
	}
	if got := jsCodePointUnits(0x1D400); got != 2 {
		t.Errorf("jsCodePointUnits(astral) = %d, want 2", got)
	}
	if got := jsCodePointUnits(0x41); got != 1 {
		t.Errorf("jsCodePointUnits(bmp) = %d, want 1", got)
	}
}

// TestJSParseTemplateElementGuard covers parseTemplateElement's guard for a
// non-template lookahead, which panics with a *jsSyntaxError.
func TestJSParseTemplateElementGuard(t *testing.T) {
	p := newJSParser("x") // lookahead is an identifier, not a template
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("parseTemplateElement(non-template) did not panic")
		}
		if _, ok := r.(*jsSyntaxError); !ok {
			t.Fatalf("panic = %T, want *jsSyntaxError", r)
		}
	}()
	p.parseTemplateElement()
}

// TestJSFinishGroupArrowYield covers finishGroupArrow's `(yield)` special case.
// yield cannot be parsed as a primary expression in this port, so the branch is
// unreachable through Parse; it is exercised directly with a crafted node.
func TestJSFinishGroupArrowYield(t *testing.T) {
	p := newJSParser("x")
	res := p.finishGroupArrow(jsNodeIdentifier("yield"))
	ph, ok := res.(jsArrowParams)
	if !ok {
		t.Fatalf("finishGroupArrow(yield) = %T, want jsArrowParams", res)
	}
	if len(ph.params) != 1 || IdentName(ph.params[0]) != "yield" {
		t.Errorf("finishGroupArrow(yield) params = %v, want [yield]", ph.params)
	}
}

func TestJSParserAST(t *testing.T) {
	f, err := os.Open("testdata/jsparser_ast.jsonl")
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
		ast, err := Parse(vec.Src)
		if err != nil {
			t.Errorf("parse %q: unexpected error %v", vec.Src, err)
			continue
		}
		if got := jsonval.Stringify(ast, 2); got != vec.Want {
			t.Errorf("parse %q:\n got:\n%s\nwant:\n%s", vec.Src, got, vec.Want)
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan golden: %v", err)
	}
	if n == 0 {
		t.Fatal("no golden vectors loaded")
	}
}
