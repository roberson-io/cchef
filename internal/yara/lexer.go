// Package yara compiles and runs YARA rules.
//
// It follows libyara, the original C implementation, rather than the newer
// yara-x: the two differ in a handful of documented ways and libyara is what
// this has to agree with.
package yara

import (
	"fmt"
	"strconv"
	"strings"
)

// tokenKind is what a token turned out to be.
type tokenKind int

const (
	tokenEOF tokenKind = iota
	tokenKeyword
	tokenIdentifier
	// The four sigils that name a string: $ for the string itself, # for how
	// many times it matched, @ for where, and ! for how long the match was.
	tokenStringIdentifier
	tokenStringCount
	tokenStringOffset
	tokenStringLength
	tokenInteger
	tokenDouble
	tokenText
	tokenRegex
	tokenHexString
	tokenPunct
)

// String names a kind, for error messages and tests.
func (k tokenKind) String() string {
	switch k {
	case tokenEOF:
		return "end of input"
	case tokenKeyword:
		return "keyword"
	case tokenIdentifier:
		return "identifier"
	case tokenStringIdentifier:
		return "string identifier"
	case tokenStringCount:
		return "string count"
	case tokenStringOffset:
		return "string offset"
	case tokenStringLength:
		return "string length"
	case tokenInteger:
		return "integer"
	case tokenDouble:
		return "double"
	case tokenText:
		return "text"
	case tokenRegex:
		return "regex"
	case tokenHexString:
		return "hex string"
	}
	return "punctuation"
}

// token is one item the lexer found.
type token struct {
	kind tokenKind
	// text is the token's value: the word for a keyword or identifier, the
	// decoded bytes for a text string, the body of a regex or hex string, and
	// the number written out again for an integer or a double.
	text string
	// flags are the letters after a regex's closing slash.
	flags string
	num   int64
	dbl   float64
	line  int
}

// keywords are the words the language reserves, taken from libyara's lexer.
var keywords = map[string]bool{
	"all": true, "and": true, "any": true, "ascii": true, "at": true,
	"base64": true, "base64wide": true, "condition": true, "contains": true,
	"defined": true, "endswith": true, "entrypoint": true, "false": true,
	"filesize": true, "for": true, "fullword": true, "global": true,
	"icontains": true, "iendswith": true, "iequals": true, "import": true,
	"in": true, "istartswith": true, "matches": true, "meta": true,
	"nocase": true, "none": true, "not": true, "of": true, "or": true,
	"private": true, "rule": true, "startswith": true, "strings": true,
	"them": true, "true": true, "wide": true, "xor": true,
}

// The multipliers an integer may be written with.
const (
	kilobyte = 1024
	megabyte = 1024 * 1024
)

// lexer walks the text of a rule set.
//
// A brace opens a hex string only where one can appear, which is directly after
// the equals sign of a string declaration, so the last token is remembered. A
// slash never divides — libyara spells integer division with a backslash — so
// one always opens a regex or a comment.
type lexer struct {
	src  string
	pos  int
	line int
	prev token
}

// newLexer starts a scan of src.
func newLexer(src string) *lexer { return &lexer{src: src, line: 1} }

// errorf reports a fault at the line the lexer has reached.
func (l *lexer) errorf(format string, args ...any) error {
	return &compileError{line: l.line, msg: fmt.Sprintf(format, args...)}
}

// next returns the following token, or one of kind tokenEOF at the end.
func (l *lexer) next() (token, error) {
	tok, err := l.scan()
	if err == nil {
		l.prev = tok
	}
	return tok, err
}

// scan does the work of next, leaving it to record what was found.
func (l *lexer) scan() (token, error) {
	if err := l.skipSpaceAndComments(); err != nil {
		return token{}, err
	}
	if l.pos >= len(l.src) {
		return token{kind: tokenEOF, line: l.line}, nil
	}

	start := l.line
	c := l.src[l.pos]
	switch {
	case c == '"':
		return l.scanText(start)
	case c == '/':
		return l.scanRegex(start)
	case c == '{' && l.prev.kind == tokenPunct && l.prev.text == "=":
		return l.scanHexString(start)
	case c >= '0' && c <= '9':
		return l.scanNumber(start)
	case isIdentifierStart(c):
		return l.scanWord(start)
	case l.opensStringIdentifier(c):
		return l.scanStringIdentifier(start)
	}
	return l.scanPunct(start)
}

// opensStringIdentifier reports whether a byte begins one of the four sigils.
// An exclamation mark only does when it is not the start of "!=".
func (l *lexer) opensStringIdentifier(c byte) bool {
	switch c {
	case '$', '#', '@':
		return true
	case '!':
		return !strings.HasPrefix(l.src[l.pos:], "!=")
	}
	return false
}

// skipSpaceAndComments moves past anything that carries no meaning.
func (l *lexer) skipSpaceAndComments() error {
	for l.pos < len(l.src) {
		switch c := l.src[l.pos]; {
		case c == '\n':
			l.line++
			l.pos++
		case c == ' ' || c == '\t' || c == '\r':
			l.pos++
		case strings.HasPrefix(l.src[l.pos:], "//"):
			for l.pos < len(l.src) && l.src[l.pos] != '\n' {
				l.pos++
			}
		case strings.HasPrefix(l.src[l.pos:], "/*"):
			end := strings.Index(l.src[l.pos+2:], "*/")
			if end < 0 {
				return l.errorf("unterminated comment")
			}
			l.line += strings.Count(l.src[l.pos:l.pos+2+end+2], "\n")
			l.pos += 2 + end + 2
		default:
			return nil
		}
	}
	return nil
}

// isIdentifierStart reports whether a byte may open an identifier.
func isIdentifierStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// isIdentifierByte reports whether a byte may continue one.
func isIdentifierByte(c byte) bool { return isIdentifierStart(c) || (c >= '0' && c <= '9') }

// scanWord reads a keyword or an identifier.
func (l *lexer) scanWord(line int) (token, error) {
	start := l.pos
	for l.pos < len(l.src) && isIdentifierByte(l.src[l.pos]) {
		l.pos++
	}
	word := l.src[start:l.pos]
	kind := tokenIdentifier
	if keywords[word] {
		kind = tokenKeyword
	}
	return token{kind: kind, text: word, line: line}, nil
}

// scanStringIdentifier reads one of the four sigils and the name after it. The
// name may be missing, which is how a rule refers to whichever string a loop is
// currently looking at.
func (l *lexer) scanStringIdentifier(line int) (token, error) {
	var kind tokenKind
	switch l.src[l.pos] {
	case '$':
		kind = tokenStringIdentifier
	case '#':
		kind = tokenStringCount
	case '@':
		kind = tokenStringOffset
	default:
		kind = tokenStringLength
	}

	start := l.pos
	l.pos++
	for l.pos < len(l.src) && isIdentifierByte(l.src[l.pos]) {
		l.pos++
	}
	return token{kind: kind, text: l.src[start:l.pos], line: line}, nil
}

// scanNumber reads an integer, in any of the bases the language writes them in,
// or a double.
func (l *lexer) scanNumber(line int) (token, error) {
	start := l.pos
	if base, digits := l.numberBase(); base != 10 {
		return l.scanBasedInteger(start, base, digits, line)
	}

	for l.pos < len(l.src) && isDigit(l.src[l.pos]) {
		l.pos++
	}
	// A dot is only part of the number when a digit follows it; two dots are
	// the range operator.
	if l.pos+1 < len(l.src) && l.src[l.pos] == '.' && isDigit(l.src[l.pos+1]) {
		return l.scanDouble(start, line)
	}
	return l.scanDecimal(start, line)
}

// scanBasedInteger reads a number written in a base other than ten, whose two
// opening characters say which.
func (l *lexer) scanBasedInteger(start, base int, digits string, line int) (token, error) {
	l.pos += 2
	from := l.pos
	for l.pos < len(l.src) && strings.IndexByte(digits, lower(l.src[l.pos])) >= 0 {
		l.pos++
	}
	n, err := strconv.ParseInt(l.src[from:l.pos], base, 64)
	if err != nil {
		return token{}, l.errorf("%q is not a number", l.src[start:l.pos])
	}
	return integerToken(n, line), nil
}

// scanDouble reads the fractional part of a number and what it comes to.
func (l *lexer) scanDouble(start, line int) (token, error) {
	l.pos++
	for l.pos < len(l.src) && isDigit(l.src[l.pos]) {
		l.pos++
	}
	d, err := strconv.ParseFloat(l.src[start:l.pos], 64)
	if err != nil {
		return token{}, l.errorf("%q is not a number", l.src[start:l.pos])
	}
	return token{kind: tokenDouble, text: strconv.FormatFloat(d, 'g', -1, 64), dbl: d, line: line}, nil
}

// scanDecimal finishes a plain number, taking in the size it may be given in.
func (l *lexer) scanDecimal(start, line int) (token, error) {
	n, err := strconv.ParseInt(l.src[start:l.pos], 10, 64)
	if err != nil {
		return token{}, l.errorf("%q is not a number", l.src[start:l.pos])
	}
	switch {
	case strings.HasPrefix(l.src[l.pos:], "KB"):
		l.pos += 2
		n *= kilobyte
	case strings.HasPrefix(l.src[l.pos:], "MB"):
		l.pos += 2
		n *= megabyte
	}
	return integerToken(n, line), nil
}

// numberBase reports the base a number is written in, and which digits it may
// use, from the two characters that open it.
func (l *lexer) numberBase() (int, string) {
	if l.pos+1 >= len(l.src) || l.src[l.pos] != '0' {
		return 10, ""
	}
	switch lower(l.src[l.pos+1]) {
	case 'x':
		return 16, "0123456789abcdef"
	case 'o':
		return 8, "01234567"
	}
	return 10, ""
}

// integerToken builds a token for a number that has been worked out.
func integerToken(n int64, line int) token {
	return token{kind: tokenInteger, text: strconv.FormatInt(n, 10), num: n, line: line}
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

// lower folds one byte to lower case.
func lower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 'a' - 'A'
	}
	return c
}

// scanText reads a quoted string, undoing the escapes it is written with.
func (l *lexer) scanText(line int) (token, error) {
	l.pos++ // the opening quote
	var b strings.Builder
	for l.pos < len(l.src) {
		switch c := l.src[l.pos]; c {
		case '"':
			l.pos++
			return token{kind: tokenText, text: b.String(), line: line}, nil
		case '\n':
			return token{}, l.errorf("unterminated string")
		case '\\':
			if err := l.readEscape(&b); err != nil {
				return token{}, err
			}
		default:
			b.WriteByte(c)
			l.pos++
		}
	}
	return token{}, l.errorf("unterminated string")
}

// readEscape reads one backslash escape into b.
func (l *lexer) readEscape(b *strings.Builder) error {
	if l.pos+1 >= len(l.src) {
		return l.errorf("unterminated string")
	}
	c := l.src[l.pos+1]
	l.pos += 2
	switch c {
	case '"', '\\':
		b.WriteByte(c)
	case 't':
		b.WriteByte('\t')
	case 'n':
		b.WriteByte('\n')
	case 'r':
		b.WriteByte('\r')
	case 'x':
		if l.pos+2 > len(l.src) {
			return l.errorf("truncated escape sequence")
		}
		n, err := strconv.ParseUint(l.src[l.pos:l.pos+2], 16, 8)
		if err != nil {
			return l.errorf("illegal escape sequence")
		}
		l.pos += 2
		b.WriteByte(byte(n))
	default:
		return l.errorf("illegal escape sequence")
	}
	return nil
}

// scanRegex reads a regular expression and the letters that follow it.
func (l *lexer) scanRegex(line int) (token, error) {
	l.pos++ // the opening slash
	var b strings.Builder
	for l.pos < len(l.src) {
		switch c := l.src[l.pos]; {
		case c == '\\' && l.pos+1 < len(l.src):
			b.WriteString(l.src[l.pos : l.pos+2])
			l.pos += 2
		case c == '\n':
			return token{}, l.errorf("unterminated regular expression")
		case c == '/':
			l.pos++
			start := l.pos
			for l.pos < len(l.src) && (l.src[l.pos] == 'i' || l.src[l.pos] == 's') {
				l.pos++
			}
			return token{kind: tokenRegex, text: b.String(), flags: l.src[start:l.pos], line: line}, nil
		default:
			b.WriteByte(c)
			l.pos++
		}
	}
	return token{}, l.errorf("unterminated regular expression")
}

// scanHexString reads the body of a braced hex pattern, leaving it to be
// picked apart later.
func (l *lexer) scanHexString(line int) (token, error) {
	l.pos++ // the opening brace
	start := l.pos
	for l.pos < len(l.src) && l.src[l.pos] != '}' {
		if l.src[l.pos] == '\n' {
			l.line++
		}
		l.pos++
	}
	if l.pos >= len(l.src) {
		return token{}, l.errorf("unterminated string")
	}
	body := strings.TrimSpace(l.src[start:l.pos])
	l.pos++ // the closing brace
	return token{kind: tokenHexString, text: body, line: line}, nil
}

// multiCharPunct are the operators written with more than one character, longest
// first so that a prefix never wins.
var multiCharPunct = []string{"==", "!=", "<=", ">=", "<<", ">>", ".."}

// scanPunct reads an operator or a piece of punctuation.
func (l *lexer) scanPunct(line int) (token, error) {
	for _, op := range multiCharPunct {
		if strings.HasPrefix(l.src[l.pos:], op) {
			l.pos += len(op)
			return token{kind: tokenPunct, text: op, line: line}, nil
		}
	}
	c := l.src[l.pos]
	if !strings.ContainsRune(`{}()[]:=,.*+-\%&|^~<>`, rune(c)) {
		return token{}, l.errorf("unexpected character %q", string(c))
	}
	l.pos++
	return token{kind: tokenPunct, text: string(c), line: line}, nil
}
