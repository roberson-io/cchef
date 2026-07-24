package ops

import (
	"fmt"
	"math"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf16"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(JSONBeautify{})
}

// JSONBeautify indents and pretty-prints JSON. Ported from CyberChef
// JSONBeautify.mjs, which parses the input leniently with JSON5 and re-emits it
// with JSON.stringify(value, null, indent). cchef reproduces run()'s plain-string
// output over a from-scratch JSON5 parser feeding the shared order-preserving JSON
// serialiser (jsonvalue.go).
//
// The third argument (Formatted) is inert in cchef: it only controls CyberChef's
// browser present() view (a collapsible HTML tree), which the CLI does not
// produce. It is kept so recipes round-trip.
type JSONBeautify struct{}

// Meta returns the operation metadata.
func (JSONBeautify) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "JSON Beautify",
		Module:      "Code",
		Description: "Indents and pretty prints JavaScript Object Notation (JSON) code.",
		InfoURL:     "https://wikipedia.org/wiki/JSON",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (JSONBeautify) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Indent string", Type: core.ArgString, Value: "    "},
		{Name: "Sort Object Keys", Type: core.ArgBoolean, Value: false},
		{Name: "Formatted", Type: core.ArgBoolean, Value: true},
	}
}

// Run beautifies the JSON input.
func (JSONBeautify) Run(in *core.Dish, args []any) (*core.Dish, error) {
	input := in.String()
	if input == "" {
		return core.NewDish([]byte(""), core.TypeString), nil
	}
	indentStr := parseEscapedChars(args[0].(string))
	sortKeys := args[1].(bool)
	// args[2] (Formatted) is inert here; it only affects CyberChef's HTML view.

	val, err := parseJSON5(input)
	if err != nil {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return nil, fmt.Errorf("Unable to parse input as JSON.\n%w", err)
	}
	if sortKeys {
		val = jbSortKeys(val)
	}
	return core.NewDish([]byte(jsStringifyIndent(val, indentStr)), core.TypeString), nil
}

// jbSortKeys deep-sorts object keys, matching CyberChef's sortKeys (JS
// Object.keys(o).sort()). Arrays keep their order; scalars are returned as-is.
func jbSortKeys(v any) any {
	switch x := v.(type) {
	case []any:
		for i := range x {
			x[i] = jbSortKeys(x[i])
		}
		return x
	case jsObject:
		out := make(jsObject, len(x))
		copy(out, x)
		sort.Slice(out, func(i, j int) bool { return out[i].k < out[j].k })
		for i := range out {
			out[i].v = jbSortKeys(out[i].v)
		}
		return out
	}
	return v
}

// json5Parser is a recursive-descent JSON5 parser producing the shared ordered
// value tree (jsObject / []any / string / float64 / bool / nil). All numbers are
// float64, matching JSON5.parse's JS Number semantics (including precision loss
// on large integers and NaN/Infinity, which JSON.stringify then renders as null).
type json5Parser struct {
	s   []rune
	pos int
}

// parseJSON5 parses a complete JSON5 document, erroring on trailing content.
func parseJSON5(s string) (any, error) {
	p := &json5Parser{s: []rune(s)}
	p.skipSpace()
	v, err := p.parseValue()
	if err != nil {
		return nil, err
	}
	p.skipSpace()
	if p.pos < len(p.s) {
		return nil, fmt.Errorf("unexpected trailing character %q", string(p.s[p.pos]))
	}
	return v, nil
}

func (p *json5Parser) cur() (rune, bool) {
	if p.pos < len(p.s) {
		return p.s[p.pos], true
	}
	return 0, false
}

func (p *json5Parser) hasPrefix(w string) bool {
	r := []rune(w)
	if p.pos+len(r) > len(p.s) {
		return false
	}
	for i, c := range r {
		if p.s[p.pos+i] != c {
			return false
		}
	}
	return true
}

// skipSpace consumes JSON5 whitespace and // and /* */ comments. An unterminated
// block comment is consumed to EOF; the parse then fails on end-of-input.
func (p *json5Parser) skipSpace() {
	for p.pos < len(p.s) {
		c := p.s[p.pos]
		if c == '\uFEFF' || unicode.IsSpace(c) {
			p.pos++
			continue
		}
		if c == '/' && p.pos+1 < len(p.s) {
			switch p.s[p.pos+1] {
			case '/':
				for p.pos < len(p.s) && p.s[p.pos] != '\n' && p.s[p.pos] != '\r' {
					p.pos++
				}
				continue
			case '*':
				p.skipBlockComment()
				continue
			}
		}
		return
	}
}

func (p *json5Parser) skipBlockComment() {
	p.pos += 2 // consume /*
	for p.pos+1 < len(p.s) {
		if p.s[p.pos] == '*' && p.s[p.pos+1] == '/' {
			p.pos += 2
			return
		}
		p.pos++
	}
	p.pos = len(p.s) // unterminated
}

func (p *json5Parser) parseValue() (any, error) {
	c, ok := p.cur()
	if !ok {
		return nil, fmt.Errorf("unexpected end of input")
	}
	switch {
	case c == '{':
		return p.parseObject()
	case c == '[':
		return p.parseArray()
	case c == '"' || c == '\'':
		return p.parseString(c)
	case c == '-' || c == '+' || c == '.' || c == 'I' || c == 'N' || (c >= '0' && c <= '9'):
		return p.parseNumber()
	default:
		return p.parseLiteral()
	}
}

func (p *json5Parser) parseLiteral() (any, error) {
	switch {
	case p.hasPrefix("true"):
		p.pos += 4
		return true, nil
	case p.hasPrefix("false"):
		p.pos += 5
		return false, nil
	case p.hasPrefix("null"):
		p.pos += 4
		return nil, nil
	}
	return nil, fmt.Errorf("unexpected token %q", string(p.s[p.pos]))
}

func (p *json5Parser) parseNumber() (any, error) {
	start := p.pos
	neg := false
	if c, _ := p.cur(); c == '+' || c == '-' {
		neg = c == '-'
		p.pos++
	}
	switch {
	case p.hasPrefix("Infinity"):
		p.pos += len("Infinity")
		if neg {
			return math.Inf(-1), nil
		}
		return math.Inf(1), nil
	case p.hasPrefix("NaN"):
		p.pos += len("NaN")
		return math.NaN(), nil
	case p.hasPrefix("0x") || p.hasPrefix("0X"):
		return p.parseHex(neg)
	}
	for p.pos < len(p.s) && isDecimalRune(p.s[p.pos]) {
		p.pos++
	}
	lit := string(p.s[start:p.pos])
	f, err := strconv.ParseFloat(lit, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid number %q", lit)
	}
	return f, nil
}

func (p *json5Parser) parseHex(neg bool) (any, error) {
	p.pos += 2 // consume 0x
	hs := p.pos
	for p.pos < len(p.s) && isHexRune(p.s[p.pos]) {
		p.pos++
	}
	if p.pos == hs {
		return nil, fmt.Errorf("invalid hexadecimal number")
	}
	digits := string(p.s[hs:p.pos])
	var f float64
	if u, err := strconv.ParseUint(digits, 16, 64); err == nil {
		f = float64(u)
	} else {
		// JS parses arbitrarily large hex literals into a double; big values that
		// overflow uint64 are converted via big.Int, matching JSON5.parse.
		bi, _ := new(big.Int).SetString(digits, 16)
		f, _ = new(big.Float).SetInt(bi).Float64()
	}
	if neg {
		f = -f
	}
	return f, nil
}

func (p *json5Parser) parseString(quote rune) (any, error) {
	p.pos++ // opening quote
	var b strings.Builder
	for p.pos < len(p.s) {
		c := p.s[p.pos]
		switch c {
		case quote:
			p.pos++
			return b.String(), nil
		case '\\':
			p.pos++
			if err := p.readEscape(&b); err != nil {
				return nil, err
			}
		default:
			b.WriteRune(c)
			p.pos++
		}
	}
	return nil, fmt.Errorf("unterminated string")
}

func (p *json5Parser) readEscape(b *strings.Builder) error {
	if p.pos >= len(p.s) {
		return fmt.Errorf("unterminated escape")
	}
	c := p.s[p.pos]
	p.pos++
	switch c {
	case 'n':
		b.WriteByte('\n')
	case 't':
		b.WriteByte('\t')
	case 'r':
		b.WriteByte('\r')
	case 'b':
		b.WriteByte('\b')
	case 'f':
		b.WriteByte('\f')
	case 'v':
		b.WriteByte('\v')
	case '0':
		b.WriteByte(0)
	case 'x':
		return p.readHexEscape(b, 2)
	case 'u':
		return p.readUnicodeEscape(b)
	case '\r':
		if p.pos < len(p.s) && p.s[p.pos] == '\n' {
			p.pos++ // \r\n line continuation
		}
	case '\n', '\u2028', '\u2029':
		// line continuation: emit nothing
	default:
		b.WriteRune(c)
	}
	return nil
}

func (p *json5Parser) readHexEscape(b *strings.Builder, n int) error {
	r, err := p.readHexDigits(n)
	if err != nil {
		return err
	}
	b.WriteRune(r)
	return nil
}

func (p *json5Parser) readUnicodeEscape(b *strings.Builder) error {
	hi, err := p.readHexDigits(4)
	if err != nil {
		return err
	}
	if utf16.IsSurrogate(hi) && p.hasPrefix(`\u`) {
		p.pos += 2
		lo, err := p.readHexDigits(4)
		if err != nil {
			return err
		}
		b.WriteRune(utf16.DecodeRune(hi, lo))
		return nil
	}
	b.WriteRune(hi)
	return nil
}

func (p *json5Parser) readHexDigits(n int) (rune, error) {
	if p.pos+n > len(p.s) {
		return 0, fmt.Errorf("invalid escape sequence")
	}
	v, err := strconv.ParseUint(string(p.s[p.pos:p.pos+n]), 16, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid escape sequence")
	}
	p.pos += n
	// v holds at most n (<=4) hex digits, so it is <= 0xFFFF and always fits a rune.
	return rune(v), nil // #nosec G115 -- value bounded to <=0xFFFF by a 4-digit max
}

func (p *json5Parser) parseObject() (any, error) {
	p.pos++ // consume {
	obj := jsObject{}
	p.skipSpace()
	if c, _ := p.cur(); c == '}' {
		p.pos++
		return obj, nil
	}
	for {
		p.skipSpace()
		key, err := p.parseKey()
		if err != nil {
			return nil, err
		}
		p.skipSpace()
		if c, ok := p.cur(); !ok || c != ':' {
			return nil, fmt.Errorf("expected ':' after object key")
		}
		p.pos++
		p.skipSpace()
		val, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		obj = jbSetKey(obj, key, val)
		done, err := p.afterElement('}')
		if err != nil {
			return nil, err
		}
		if done {
			return obj, nil
		}
	}
}

func (p *json5Parser) parseArray() (any, error) {
	p.pos++ // consume [
	arr := []any{}
	p.skipSpace()
	if c, _ := p.cur(); c == ']' {
		p.pos++
		return arr, nil
	}
	for {
		p.skipSpace()
		v, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		arr = append(arr, v)
		done, err := p.afterElement(']')
		if err != nil {
			return nil, err
		}
		if done {
			return arr, nil
		}
	}
}

// afterElement consumes the separator following an array/object element. It
// returns done=true when the closing bracket is reached (handling a trailing
// comma), false to continue with the next element.
func (p *json5Parser) afterElement(closer rune) (bool, error) {
	p.skipSpace()
	c, ok := p.cur()
	if !ok {
		return false, fmt.Errorf("unterminated container")
	}
	switch c {
	case ',':
		p.pos++
		p.skipSpace()
		if c2, _ := p.cur(); c2 == closer {
			p.pos++
			return true, nil // trailing comma
		}
		return false, nil
	case closer:
		p.pos++
		return true, nil
	}
	return false, fmt.Errorf("expected ',' or %q", string(closer))
}

func (p *json5Parser) parseKey() (string, error) {
	c, ok := p.cur()
	if !ok {
		return "", fmt.Errorf("expected object key")
	}
	if c == '"' || c == '\'' {
		v, err := p.parseString(c)
		if err != nil {
			return "", err
		}
		return v.(string), nil
	}
	return p.parseIdentifier()
}

func (p *json5Parser) parseIdentifier() (string, error) {
	start := p.pos
	for p.pos < len(p.s) {
		c := p.s[p.pos]
		if c == '$' || c == '_' || unicode.IsLetter(c) || (p.pos > start && unicode.IsDigit(c)) {
			p.pos++
			continue
		}
		break
	}
	if p.pos == start {
		return "", fmt.Errorf("invalid identifier")
	}
	return string(p.s[start:p.pos]), nil
}

// jbSetKey appends key→value, or, if the key already exists, updates the value in
// place (last value wins, first position kept) — matching JS object semantics for
// duplicate keys.
func jbSetKey(obj jsObject, k string, v any) jsObject {
	if i := jsIndex(obj, k); i >= 0 {
		obj[i].v = v
		return obj
	}
	return append(obj, jsPair{k: k, v: v})
}

func isDecimalRune(c rune) bool {
	return (c >= '0' && c <= '9') || c == '.' || c == 'e' || c == 'E' || c == '+' || c == '-'
}

func isHexRune(c rune) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}
