package jsparse

// Scanner for the JavaScript Parser operation — a faithful transliteration of
// esprima's Scanner (esprima/dist/esprima.js module 12). esprima operates on
// UTF-16 code units, so the source is held as []uint16 and all indexing is by
// code unit. esprima throws on malformed input; this port mirrors that control
// flow with panic/recover of *jsSyntaxError, caught at the parser's entry point.
//
// The scanner handles the full script grammar: whitespace and comments,
// identifiers/keywords (ASCII, non-ASCII, and `\u`/`\u{}`-escaped, via the
// Unicode v8.0.0 tables in jsparser_unicode.go), punctuators, numeric literals
// (decimal/hex/binary/octal/float/exponent), string literals with escapes,
// template literals, regular expressions, and boolean/null.

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/roberson-io/cchef/internal/jsonval"
)

// Token types (esprima's numeric enum).
const (
	tkBooleanLiteral    = 1
	tkEOF               = 2
	tkIdentifier        = 3
	tkKeyword           = 4
	tkNullLiteral       = 5
	tkNumericLiteral    = 6
	tkPunctuator        = 7
	tkStringLiteral     = 8
	tkRegularExpression = 9
	tkTemplate          = 10
)

// jsSyntaxError mirrors esprima's thrown error, formatted like V8: "Line N: msg".
type jsSyntaxError struct {
	line, column, index int
	description         string
}

func (e *jsSyntaxError) Error() string {
	return fmt.Sprintf("Line %d: %s", e.line, e.description)
}

// jsToken is a lexed token. value holds a string for most types and a float64
// for numeric literals (as esprima's token.value does); start/end are code-unit
// offsets used to recover the raw source slice.
type jsToken struct {
	typ        int
	value      any
	octal      bool
	cooked     string // template: the cooked (escape-processed) text
	head       bool   // template: starts with backtick
	tail       bool   // template: ends with backtick
	pattern    string // regex: the pattern body
	flags      string // regex: the flags
	lineNumber int
	lineStart  int
	start      int
	end        int
}

// --- character classification (esprima Character, module 4) ---

// IsWhiteSpace reports whether a code point is whitespace to JavaScript,
// which counts more characters than Go does.
func IsWhiteSpace(cp int) bool {
	switch cp {
	case 0x20, 0x09, 0x0B, 0x0C, 0xA0, 0x1680, 0x2000, 0x2001, 0x2002, 0x2003,
		0x2004, 0x2005, 0x2006, 0x2007, 0x2008, 0x2009, 0x200A, 0x202F, 0x205F,
		0x3000, 0xFEFF:
		return true
	}
	return false
}

func jsIsLineTerminator(cp int) bool {
	return cp == 0x0A || cp == 0x0D || cp == 0x2028 || cp == 0x2029
}

// jsIsIdentifierStart / IsIdentifierPart mirror esprima's Character helpers:
// the ASCII range exactly, plus the Unicode v8.0.0 ID_Start/ID_Continue sets
// (jsparser_unicode.go) for cp >= 0x80.
func jsIsIdentifierStart(cp int) bool {
	return cp == 0x24 || cp == 0x5F ||
		(cp >= 0x41 && cp <= 0x5A) || (cp >= 0x61 && cp <= 0x7A) || cp == 0x5C ||
		(cp >= 0x80 && jsUniIn(jsIdentifierStartRanges, cp))
}

// IsIdentifierPart reports whether a code point may appear after the first
// character of a JavaScript identifier.
func IsIdentifierPart(cp int) bool {
	return cp == 0x24 || cp == 0x5F ||
		(cp >= 0x41 && cp <= 0x5A) || (cp >= 0x61 && cp <= 0x7A) ||
		(cp >= 0x30 && cp <= 0x39) || cp == 0x5C ||
		(cp >= 0x80 && jsUniIn(jsIdentifierPartRanges, cp))
}

// jsUniIn reports whether cp falls in one of the sorted, non-overlapping ranges.
func jsUniIn(ranges []jsUniRange, cp int) bool {
	lo, hi := 0, len(ranges)-1
	c := rune(cp) // #nosec G115 -- cp is a Unicode code point (0-0x10FFFF)
	for lo <= hi {
		mid := (lo + hi) / 2
		switch r := ranges[mid]; {
		case c < r.lo:
			hi = mid - 1
		case c > r.hi:
			lo = mid + 1
		default:
			return true
		}
	}
	return false
}

func jsIsDecimalDigit(cp int) bool { return cp >= 0x30 && cp <= 0x39 }

func jsIsHexDigit(cp int) bool {
	return (cp >= 0x30 && cp <= 0x39) || (cp >= 0x41 && cp <= 0x46) || (cp >= 0x61 && cp <= 0x66)
}

func jsIsOctalDigit(cp int) bool { return cp >= 0x30 && cp <= 0x37 }

func jsHexValue(cp int) int {
	switch {
	case cp >= 0x30 && cp <= 0x39:
		return cp - 0x30
	case cp >= 0x41 && cp <= 0x46:
		return cp - 0x41 + 10
	default:
		return cp - 0x61 + 10
	}
}

// --- scanner ---

type jsScanner struct {
	source     []uint16
	length     int
	index      int
	lineNumber int
	lineStart  int
	curlyStack []string // tracks '{' and '${' nesting for template scanning

	trackComment bool             // when set, comments are collected (ParseFull)
	comments     []jsonval.Object // collected comments (type/value/range)
	seenComment  map[int]bool     // dedup by start offset (peeks re-scan comments)
}

// recordComment appends a collected comment, deduplicating by start offset since
// lookahead peeks re-scan the same comments. multiLine selects Block vs Line;
// [vStart,vEnd) is the comment body slice, [rStart,rEnd) the full range.
func (s *jsScanner) recordComment(multiLine bool, vStart, vEnd, rStart, rEnd int) {
	if s.seenComment == nil {
		s.seenComment = map[int]bool{}
	}
	if s.seenComment[rStart] {
		return
	}
	s.seenComment[rStart] = true
	typ := "Line"
	if multiLine {
		typ = "Block"
	}
	s.comments = append(s.comments, jsonval.Object{
		{K: "type", V: typ},
		{K: "value", V: s.slice(vStart, vEnd)},
		{K: "range", V: []any{float64(rStart), float64(rEnd)}},
	})
}

func newJSScanner(code string) *jsScanner {
	src := utf16.Encode([]rune(code))
	ln := 0
	if len(src) > 0 {
		ln = 1
	}
	return &jsScanner{source: src, length: len(src), lineNumber: ln}
}

func (s *jsScanner) eof() bool { return s.index >= s.length }

// jsScannerState snapshots the scanner position for lookahead peeking.
type jsScannerState struct{ index, lineNumber, lineStart int }

func (s *jsScanner) saveState() jsScannerState {
	return jsScannerState{s.index, s.lineNumber, s.lineStart}
}

func (s *jsScanner) restoreState(st jsScannerState) {
	s.index, s.lineNumber, s.lineStart = st.index, st.lineNumber, st.lineStart
}

// at returns the code unit at i, or -1 past the end (mirroring charCodeAt→NaN
// comparisons that are always false in the JS).
func (s *jsScanner) at(i int) int {
	if i < 0 || i >= s.length {
		return -1
	}
	return int(s.source[i])
}

// slice returns the decoded source between code-unit offsets [a, b).
func (s *jsScanner) slice(a, b int) string {
	return string(utf16.Decode(s.source[a:b]))
}

// throwUnexpectedToken panics with a *jsSyntaxError, mirroring esprima's throw.
func (s *jsScanner) throwUnexpectedToken(desc string) {
	if desc == "" {
		desc = "Unexpected token ILLEGAL"
	}
	panic(&jsSyntaxError{
		line:        s.lineNumber,
		column:      s.index - s.lineStart + 1,
		index:       s.index,
		description: desc,
	})
}

// scanComments skips whitespace, line terminators and comments.
func (s *jsScanner) scanComments() {
	start := s.index == 0
	for !s.eof() {
		ch := s.at(s.index)
		switch {
		case IsWhiteSpace(ch):
			s.index++
		case jsIsLineTerminator(ch):
			s.index++
			if ch == 0x0D && s.at(s.index) == 0x0A {
				s.index++
			}
			s.lineNumber++
			s.lineStart = s.index
			start = true
		default:
			cont, newStart := s.skipCommentMarker(ch, start)
			if !cont {
				return
			}
			if newStart {
				start = true
			}
		}
	}
}

// skipCommentMarker skips a comment (line `//`, block `/* */`, or the HTML-style
// `<!--` / `-->`). cont is false when ch does not begin a comment (so scanning
// stops); newStart marks that a line-comment set the line-start flag.
func (s *jsScanner) skipCommentMarker(ch int, atStart bool) (cont, newStart bool) {
	switch {
	case ch == 0x2F: // /
		switch s.at(s.index + 1) {
		case 0x2F: // //
			s.index += 2
			s.skipSingleLineComment(2)
			return true, true
		case 0x2A: // /*
			s.index += 2
			s.skipMultiLineComment()
			return true, false
		}
	case atStart && ch == 0x2D: // --> line comment at line start
		if s.at(s.index+1) == 0x2D && s.at(s.index+2) == 0x3E {
			s.index += 3
			s.skipSingleLineComment(3)
			return true, false
		}
	case ch == 0x3C: // <!-- line comment
		if s.index+4 <= s.length && s.slice(s.index+1, s.index+4) == "!--" {
			s.index += 4
			s.skipSingleLineComment(4)
			return true, false
		}
	}
	return false, false
}

func (s *jsScanner) skipSingleLineComment(offset int) {
	start := s.index - offset
	for !s.eof() {
		ch := s.at(s.index)
		s.index++
		if jsIsLineTerminator(ch) {
			if s.trackComment {
				s.recordComment(false, start+offset, s.index-1, start, s.index-1)
			}
			if ch == 13 && s.at(s.index) == 10 {
				s.index++
			}
			s.lineNumber++
			s.lineStart = s.index
			return
		}
	}
	if s.trackComment {
		s.recordComment(false, start+offset, s.index, start, s.index)
	}
}

func (s *jsScanner) skipMultiLineComment() {
	start := s.index - 2
	for !s.eof() {
		ch := s.at(s.index)
		switch {
		case jsIsLineTerminator(ch):
			if ch == 0x0D && s.at(s.index+1) == 0x0A {
				s.index++
			}
			s.lineNumber++
			s.index++
			s.lineStart = s.index
		case ch == 0x2A: // *
			if s.at(s.index+1) == 0x2F {
				s.index += 2
				if s.trackComment {
					s.recordComment(true, start+2, s.index-2, start, s.index)
				}
				return
			}
			s.index++
		default:
			s.index++
		}
	}
	s.throwUnexpectedToken("")
}

// --- identifiers ---

// jsKeywords is the reserved-word set esprima classifies as Keyword tokens.
var jsKeywords = map[string]bool{
	"if": true, "in": true, "do": true, "var": true, "for": true, "new": true,
	"try": true, "let": true, "this": true, "else": true, "case": true,
	"void": true, "with": true, "enum": true, "while": true, "break": true,
	"catch": true, "throw": true, "const": true, "yield": true, "class": true,
	"super": true, "return": true, "typeof": true, "delete": true, "switch": true,
	"export": true, "import": true, "default": true, "finally": true,
	"extends": true, "function": true, "continue": true, "debugger": true,
	"instanceof": true,
}

func (s *jsScanner) isKeyword(id string) bool { return jsKeywords[id] }

// codePointAt returns the Unicode code point at code-unit offset i, combining a
// surrogate pair when present.
func (s *jsScanner) codePointAt(i int) int {
	cp := s.at(i)
	if cp >= 0xD800 && cp <= 0xDBFF && i+1 < s.length {
		if second := s.at(i + 1); second >= 0xDC00 && second <= 0xDFFF {
			return (cp-0xD800)*0x400 + (second - 0xDC00) + 0x10000
		}
	}
	return cp
}

// jsCodePointUnits reports the number of UTF-16 code units a code point occupies.
func jsCodePointUnits(cp int) int {
	if cp > 0xFFFF {
		return 2
	}
	return 1
}

// jsUTF16Len reports the number of UTF-16 code units in s (esprima's id.length).
func jsUTF16Len(s string) int { return len(utf16.Encode([]rune(s))) }

func (s *jsScanner) getIdentifier() string {
	start := s.index
	s.index++
	for !s.eof() {
		ch := s.at(s.index)
		if ch == 0x5C || (ch >= 0xD800 && ch < 0xDFFF) {
			// Unicode-escaped or astral identifier: rescan from the start.
			s.index = start
			return s.getComplexIdentifier()
		}
		if IsIdentifierPart(ch) {
			s.index++
		} else {
			break
		}
	}
	return s.slice(start, s.index)
}

// scanIdentifierEscape parses a `\uHHHH` or `\u{...}` escape inside an
// identifier, validating the decoded code point as an identifier start/part.
func (s *jsScanner) scanIdentifierEscape(isStart bool) rune {
	if s.at(s.index) != 0x75 { // u
		s.throwUnexpectedToken("")
	}
	s.index++
	if s.at(s.index) == 0x7B { // {
		s.index++
		return s.scanUnicodeCodePointEscape()
	}
	ch, ok := s.scanHexEscape('u')
	valid := IsIdentifierPart(int(ch))
	if isStart {
		valid = jsIsIdentifierStart(int(ch))
	}
	if !ok || ch == '\\' || !valid {
		s.throwUnexpectedToken("")
	}
	return ch
}

func (s *jsScanner) getComplexIdentifier() string {
	cp := s.codePointAt(s.index)
	id := []rune{rune(cp)} // #nosec G115 -- cp is a Unicode code point (0-0x10FFFF)
	s.index += jsCodePointUnits(cp)
	if cp == 0x5C {
		id = []rune{s.scanIdentifierEscape(true)}
	}
	for !s.eof() {
		cp = s.codePointAt(s.index)
		if !IsIdentifierPart(cp) {
			break
		}
		id = append(id, rune(cp)) // #nosec G115 -- cp is a Unicode code point (0-0x10FFFF)
		s.index += jsCodePointUnits(cp)
		if cp == 0x5C {
			id = id[:len(id)-1]
			id = append(id, s.scanIdentifierEscape(false))
		}
	}
	return string(id)
}

func (s *jsScanner) scanIdentifier() jsToken {
	start := s.index
	var id string
	if s.at(start) == 0x5C {
		id = s.getComplexIdentifier()
	} else {
		id = s.getIdentifier()
	}
	typ := tkIdentifier
	switch {
	case jsUTF16Len(id) == 1:
		typ = tkIdentifier
	case s.isKeyword(id):
		typ = tkKeyword
	case id == "null":
		typ = tkNullLiteral
	case id == "true" || id == "false":
		typ = tkBooleanLiteral
	}
	if typ != tkIdentifier && start+jsUTF16Len(id) != s.index {
		s.throwUnexpectedToken("Keyword must not contain escaped characters")
	}
	return jsToken{typ: typ, value: id, lineNumber: s.lineNumber, lineStart: s.lineStart, start: start, end: s.index}
}

// --- punctuators ---

var (
	jsPunct3 = map[string]bool{"===": true, "!==": true, ">>>": true, "<<=": true, ">>=": true, "**=": true}
	jsPunct2 = map[string]bool{
		"&&": true, "||": true, "==": true, "!=": true, "+=": true, "-=": true,
		"*=": true, "/=": true, "++": true, "--": true, "<<": true, ">>": true,
		"&=": true, "|=": true, "^=": true, "%=": true, "<=": true, ">=": true,
		"=>": true, "**": true,
	}
)

func (s *jsScanner) scanPunctuator() jsToken {
	start := s.index
	str := string(rune(s.source[s.index]))
	switch str {
	case "(":
		s.index++
	case "{":
		s.curlyStack = append(s.curlyStack, "{")
		s.index++
	case "}":
		s.index++
		if len(s.curlyStack) > 0 {
			s.curlyStack = s.curlyStack[:len(s.curlyStack)-1]
		}
	case ".":
		s.index++
		if s.at(s.index) == 0x2E && s.at(s.index+1) == 0x2E {
			s.index += 2
			str = "..."
		}
	case ")", ";", ",", "[", "]", ":", "?", "~":
		s.index++
	default:
		four := s.substr(s.index, 4)
		switch four {
		case ">>>=":
			s.index += 4
			str = four
		default:
			three := s.substr(s.index, 3)
			if jsPunct3[three] {
				s.index += 3
				str = three
				break
			}
			two := s.substr(s.index, 2)
			if jsPunct2[two] {
				s.index += 2
				str = two
				break
			}
			one := string(rune(s.source[s.index]))
			if strings.IndexByte("<>=!+-*%&|^/", one[0]) >= 0 {
				s.index++
				str = one
			}
		}
	}
	if s.index == start {
		s.throwUnexpectedToken("")
	}
	return jsToken{typ: tkPunctuator, value: str, lineNumber: s.lineNumber, lineStart: s.lineStart, start: start, end: s.index}
}

// substr returns up to n code units from i as a decoded string.
func (s *jsScanner) substr(i, n int) string {
	end := min(i+n, s.length)
	return s.slice(i, end)
}

// --- numeric literals ---

func (s *jsScanner) scanHexLiteral(start int) jsToken {
	var num strings.Builder
	for !s.eof() && jsIsHexDigit(s.at(s.index)) {
		num.WriteString(string(rune(s.source[s.index])))
		s.index++
	}
	if len(num.String()) == 0 || jsIsIdentifierStart(s.at(s.index)) {
		s.throwUnexpectedToken("")
	}
	v, _ := strconv.ParseUint(num.String(), 16, 64)
	return s.numToken(float64(v), start)
}

func (s *jsScanner) scanBinaryLiteral(start int) jsToken {
	var num strings.Builder
	for !s.eof() {
		ch := s.at(s.index)
		if ch != 0x30 && ch != 0x31 {
			break
		}
		num.WriteString(string(rune(s.source[s.index])))
		s.index++
	}
	if len(num.String()) == 0 {
		s.throwUnexpectedToken("")
	}
	if !s.eof() {
		ch := s.at(s.index)
		if jsIsIdentifierStart(ch) || jsIsDecimalDigit(ch) {
			s.throwUnexpectedToken("")
		}
	}
	v, _ := strconv.ParseUint(num.String(), 2, 64)
	return s.numToken(float64(v), start)
}

func (s *jsScanner) scanOctalLiteral(prefix int, start int) jsToken {
	num := ""
	octal := false
	if jsIsOctalDigit(prefix) {
		octal = true
		num = "0" + string(rune(s.source[s.index]))
		s.index++
	} else {
		s.index++
	}
	for !s.eof() && jsIsOctalDigit(s.at(s.index)) {
		num += string(rune(s.source[s.index]))
		s.index++
	}
	if !octal && len(num) == 0 {
		s.throwUnexpectedToken("")
	}
	if jsIsIdentifierStart(s.at(s.index)) || jsIsDecimalDigit(s.at(s.index)) {
		s.throwUnexpectedToken("")
	}
	v, _ := strconv.ParseUint(num, 8, 64)
	t := s.numToken(float64(v), start)
	t.octal = octal
	return t
}

func (s *jsScanner) isImplicitOctalLiteral() bool {
	for i := s.index + 1; i < s.length; i++ {
		ch := s.at(i)
		if ch == 0x38 || ch == 0x39 { // 8 or 9
			return false
		}
		if !jsIsOctalDigit(ch) {
			return true
		}
	}
	return true
}

func (s *jsScanner) numToken(v float64, start int) jsToken {
	return jsToken{typ: tkNumericLiteral, value: v, lineNumber: s.lineNumber, lineStart: s.lineStart, start: start, end: s.index}
}

// scanRadixLiteral handles a numeric literal with a leading '0': hex (0x),
// binary (0b), explicit octal (0o) or legacy implicit octal. ch is the character
// after the '0'; ok is false when it is an ordinary decimal.
func (s *jsScanner) scanRadixLiteral(ch, start int) (jsToken, bool) {
	switch {
	case ch == 0x78 || ch == 0x58: // x X
		s.index++
		return s.scanHexLiteral(start), true
	case ch == 0x62 || ch == 0x42: // b B
		s.index++
		return s.scanBinaryLiteral(start), true
	case ch == 0x6F || ch == 0x4F: // o O
		return s.scanOctalLiteral(ch, start), true
	case ch >= 0 && jsIsOctalDigit(ch) && s.isImplicitOctalLiteral():
		return s.scanOctalLiteral(ch, start), true
	}
	return jsToken{}, false
}

func (s *jsScanner) scanNumericLiteral() jsToken {
	start := s.index
	ch := s.at(start)
	num := ""
	if ch != 0x2E { // not '.'
		num = string(rune(s.source[s.index]))
		s.index++
		ch = s.at(s.index)
		if num == "0" {
			if tok, ok := s.scanRadixLiteral(ch, start); ok {
				return tok
			}
		}
		for jsIsDecimalDigit(s.at(s.index)) {
			num += string(rune(s.source[s.index]))
			s.index++
		}
		ch = s.at(s.index)
	}
	if ch == 0x2E { // .
		num += string(rune(s.source[s.index]))
		s.index++
		for jsIsDecimalDigit(s.at(s.index)) {
			num += string(rune(s.source[s.index]))
			s.index++
		}
		ch = s.at(s.index)
	}
	if ch == 0x65 || ch == 0x45 { // e E
		num += string(rune(s.source[s.index]))
		s.index++
		ch = s.at(s.index)
		if ch == 0x2B || ch == 0x2D { // + -
			num += string(rune(s.source[s.index]))
			s.index++
		}
		if jsIsDecimalDigit(s.at(s.index)) {
			for jsIsDecimalDigit(s.at(s.index)) {
				num += string(rune(s.source[s.index]))
				s.index++
			}
		} else {
			s.throwUnexpectedToken("")
		}
	}
	if jsIsIdentifierStart(s.at(s.index)) {
		s.throwUnexpectedToken("")
	}
	v, _ := strconv.ParseFloat(num, 64)
	return s.numToken(v, start)
}

// --- string literals ---

func (s *jsScanner) scanHexEscape(prefix int) (rune, bool) {
	length := 2
	if prefix == 'u' {
		length = 4
	}
	code := 0
	for range length {
		if !s.eof() && jsIsHexDigit(s.at(s.index)) {
			code = code*16 + jsHexValue(s.at(s.index))
			s.index++
		} else {
			return 0, false
		}
	}
	return rune(code), true
}

func (s *jsScanner) scanUnicodeCodePointEscape() rune {
	ch := s.at(s.index)
	code := 0
	if ch == 0x7D { // }
		s.throwUnexpectedToken("")
	}
	for !s.eof() {
		ch = s.at(s.index)
		s.index++
		if !jsIsHexDigit(ch) {
			break
		}
		code = code*16 + jsHexValue(ch)
	}
	if code > 0x10FFFF || ch != 0x7D {
		s.throwUnexpectedToken("")
	}
	return rune(code)
}

func (s *jsScanner) octalToDecimal(ch int) (int, bool) {
	octal := ch != 0x30
	code := ch - 0x30
	if !s.eof() && jsIsOctalDigit(s.at(s.index)) {
		octal = true
		code = code*8 + (s.at(s.index) - 0x30)
		s.index++
		if (ch == 0x30 || ch == 0x31 || ch == 0x32 || ch == 0x33) && !s.eof() && jsIsOctalDigit(s.at(s.index)) {
			code = code*8 + (s.at(s.index) - 0x30)
			s.index++
		}
	}
	return code, octal
}

func (s *jsScanner) scanStringLiteral() jsToken {
	start := s.index
	quote := s.at(start)
	s.index++
	octal := false
	var str strings.Builder
	closed := false
	for !s.eof() {
		ch := s.at(s.index)
		s.index++
		if ch == quote {
			closed = true
			break
		}
		if ch == 0x5C { // backslash
			octal = s.scanStringEscape(&str) || octal
			continue
		}
		if jsIsLineTerminator(ch) {
			break // unterminated string
		}
		str.WriteRune(jsRuneFromUnit(ch))
	}
	if !closed {
		s.index = start
		s.throwUnexpectedToken("")
	}
	return jsToken{typ: tkStringLiteral, value: str.String(), octal: octal, lineNumber: s.lineNumber, lineStart: s.lineStart, start: start, end: s.index}
}

// jsSimpleEscape maps the single-character string/template escapes to their
// values.
var jsSimpleEscape = map[int]byte{'n': '\n', 'r': '\r', 't': '\t', 'b': '\b', 'f': '\f', 'v': 0x0B}

// scanStringEscape processes a backslash escape inside a string literal,
// appending to str and returning whether it was a (legacy) octal escape.
func (s *jsScanner) scanStringEscape(str *strings.Builder) bool {
	ch := s.at(s.index)
	s.index++
	if ch >= 0 && jsIsLineTerminator(ch) {
		s.consumeLineTerminator(ch)
		return false
	}
	if b, ok := jsSimpleEscape[ch]; ok {
		str.WriteByte(b)
		return false
	}
	switch ch {
	case 'u':
		s.scanUnicodeEscapeInto(str)
	case 'x':
		r, ok := s.scanHexEscape('x')
		if !ok {
			s.throwUnexpectedToken("Invalid hexadecimal escape sequence")
		}
		str.WriteRune(r)
	case '8', '9':
		str.WriteByte(byte(ch))
		s.throwUnexpectedToken("")
	default:
		if ch >= 0 && jsIsOctalDigit(ch) {
			code, oct := s.octalToDecimal(ch)
			str.WriteRune(rune(code)) // #nosec G115 -- octal escape value is 0-255
			return oct
		}
		if ch >= 0 {
			str.WriteRune(jsRuneFromUnit(ch))
		}
	}
	return false
}

// consumeLineTerminator advances over a line terminator (handling CRLF).
func (s *jsScanner) consumeLineTerminator(ch int) {
	s.lineNumber++
	if ch == '\r' && s.at(s.index) == '\n' {
		s.index++
	}
	s.lineStart = s.index
}

// scanUnicodeEscapeInto handles a `\u` escape (either `\u{...}` or `\uHHHH`).
func (s *jsScanner) scanUnicodeEscapeInto(str *strings.Builder) {
	if s.at(s.index) == 0x7B { // {
		s.index++
		str.WriteRune(s.scanUnicodeCodePointEscape())
		return
	}
	r, ok := s.scanHexEscape('u')
	if !ok {
		s.throwUnexpectedToken("")
	}
	str.WriteRune(r)
}

// jsRuneFromUnit converts a single UTF-16 code unit to a rune (astral pairs are
// handled by callers; lone units map directly).
func jsRuneFromUnit(u int) rune { return rune(u) } // #nosec G115 -- u is a UTF-16 code unit (0-0xFFFF)

// scanTemplate scans a template literal component (head/middle/tail).
func (s *jsScanner) scanTemplate() jsToken {
	var cooked strings.Builder
	terminated := false
	start := s.index
	head := s.at(start) == 0x60 // `
	tail := false
	rawOffset := 2
	s.index++
	for !s.eof() {
		ch := s.at(s.index)
		s.index++
		switch {
		case ch == 0x60: // `
			rawOffset = 1
			tail = true
			terminated = true
		case ch == 0x24: // $
			if s.at(s.index) == 0x7B { // {
				s.curlyStack = append(s.curlyStack, "${")
				s.index++
				terminated = true
			} else {
				cooked.WriteByte('$')
			}
		case ch == 0x5C: // backslash
			s.scanTemplateEscape(&cooked)
		case jsIsLineTerminator(ch):
			s.lineNumber++
			if ch == '\r' && s.at(s.index) == '\n' {
				s.index++
			}
			s.lineStart = s.index
			cooked.WriteByte('\n')
		default:
			cooked.WriteRune(jsRuneFromUnit(ch))
		}
		if terminated {
			break
		}
	}
	if !terminated {
		s.throwUnexpectedToken("")
	}
	if !head {
		s.curlyStack = s.curlyStack[:len(s.curlyStack)-1]
	}
	return jsToken{
		typ: tkTemplate, value: s.slice(start+1, s.index-rawOffset), cooked: cooked.String(),
		head: head, tail: tail, lineNumber: s.lineNumber, lineStart: s.lineStart, start: start, end: s.index,
	}
}

// scanTemplateEscape processes a backslash escape inside a template literal.
func (s *jsScanner) scanTemplateEscape(cooked *strings.Builder) {
	ch := s.at(s.index)
	s.index++
	if jsIsLineTerminator(ch) {
		s.consumeLineTerminator(ch)
		return
	}
	if b, ok := jsSimpleEscape[ch]; ok {
		cooked.WriteByte(b)
		return
	}
	switch ch {
	case 'u':
		if s.at(s.index) == 0x7B {
			s.index++
			cooked.WriteRune(s.scanUnicodeCodePointEscape())
		} else {
			restore := s.index
			if r, ok := s.scanHexEscape('u'); ok {
				cooked.WriteRune(r)
			} else {
				s.index = restore
				cooked.WriteByte('u')
			}
		}
	case 'x':
		r, ok := s.scanHexEscape('x')
		if !ok {
			s.throwUnexpectedToken("Invalid hexadecimal escape sequence")
		}
		cooked.WriteRune(r)
	case '0':
		if jsIsDecimalDigit(s.at(s.index)) {
			s.throwUnexpectedToken("Octal literals are not allowed in template strings.")
		}
		cooked.WriteByte(0)
	default:
		if jsIsOctalDigit(ch) {
			s.throwUnexpectedToken("Octal literals are not allowed in template strings.")
		}
		cooked.WriteRune(jsRuneFromUnit(ch))
	}
}

// --- regular expressions ---

const jsMsgUnterminatedRegExp = "Invalid regular expression: missing /"

// scanRegExpBody scans the pattern between the slashes (without the slashes).
func (s *jsScanner) scanRegExpBody() string {
	s.index++ // leading '/'
	var pat strings.Builder
	classMarker := false
	terminated := false
	for !s.eof() {
		ch := s.at(s.index)
		s.index++
		switch {
		case ch == 0x5C: // backslash escape
			pat.WriteRune(jsRuneFromUnit(ch))
			ch = s.at(s.index)
			s.index++
			if jsIsLineTerminator(ch) {
				s.throwUnexpectedToken(jsMsgUnterminatedRegExp)
			}
			pat.WriteRune(jsRuneFromUnit(ch))
		case jsIsLineTerminator(ch):
			s.throwUnexpectedToken(jsMsgUnterminatedRegExp)
		case classMarker:
			pat.WriteRune(jsRuneFromUnit(ch))
			if ch == ']' {
				classMarker = false
			}
		default:
			if ch == '/' {
				terminated = true
			} else {
				pat.WriteRune(jsRuneFromUnit(ch))
				if ch == '[' {
					classMarker = true
				}
			}
		}
		if terminated {
			break
		}
	}
	if !terminated {
		s.throwUnexpectedToken(jsMsgUnterminatedRegExp)
	}
	return pat.String()
}

// scanRegExpFlags scans the trailing flag characters.
func (s *jsScanner) scanRegExpFlags() string {
	var flags strings.Builder
	for !s.eof() {
		ch := s.at(s.index)
		if !IsIdentifierPart(ch) {
			break
		}
		s.index++
		flags.WriteRune(jsRuneFromUnit(ch))
	}
	return flags.String()
}

// scanRegExp scans a regular-expression literal. Pattern validity is not checked
// (Go's RE2 engine differs from JS, so validating here would false-reject valid
// JS patterns); esprima's "Invalid regular expression" error is thus not raised.
func (s *jsScanner) scanRegExp() jsToken {
	start := s.index
	pattern := s.scanRegExpBody()
	flags := s.scanRegExpFlags()
	return jsToken{
		typ: tkRegularExpression, value: "", pattern: pattern, flags: flags,
		lineNumber: s.lineNumber, lineStart: s.lineStart, start: start, end: s.index,
	}
}

// lex returns the next token, dispatching on the current code unit.
func (s *jsScanner) lex() jsToken {
	if s.eof() {
		return jsToken{typ: tkEOF, value: "", lineNumber: s.lineNumber, lineStart: s.lineStart, start: s.index, end: s.index}
	}
	cp := s.at(s.index)
	if jsIsIdentifierStart(cp) {
		return s.scanIdentifier()
	}
	if cp == 0x28 || cp == 0x29 || cp == 0x3B { // ( ) ;
		return s.scanPunctuator()
	}
	if cp == 0x27 || cp == 0x22 { // ' "
		return s.scanStringLiteral()
	}
	if cp == 0x2E { // .
		if jsIsDecimalDigit(s.at(s.index + 1)) {
			return s.scanNumericLiteral()
		}
		return s.scanPunctuator()
	}
	if jsIsDecimalDigit(cp) {
		return s.scanNumericLiteral()
	}
	if cp == 0x60 || (cp == 0x7D && len(s.curlyStack) > 0 && s.curlyStack[len(s.curlyStack)-1] == "${") {
		return s.scanTemplate()
	}
	return s.scanPunctuator()
}

// jsRawToken decodes the raw source text of a token (getTokenRaw).
func (s *jsScanner) rawToken(t jsToken) string { return s.slice(t.start, t.end) }
