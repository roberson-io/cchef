package ops

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/jsonval"
)

func init() {
	core.Register(RisonEncode{})
	core.Register(RisonDecode{})
}

const risonDesc = "Rison, a data serialization format optimized for compactness in URIs. Rison is a slight variation of JSON that looks vastly superior after URI encoding. Rison still expresses exactly the same set of data structures as JSON, so data can be translated back and forth without loss or guesswork."

// RisonEncode serialises a JSON value into Rison, a compact URI-friendly format.
//
// This is an in-repo port of the `rison` npm library CyberChef wraps (a pure
// JS encoder/parser), reusing cchef's JSON dish model. It reproduces the
// library's behaviour, including object-key sorting on encode and the
// replace-first-only quirk in its URI quoting.
type RisonEncode struct{}

// Meta returns the operation metadata.
func (RisonEncode) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Rison Encode",
		Module:      "Encodings",
		Description: risonDesc,
		InfoURL:     "https://github.com/Nanonid/rison",
		InputType:   core.TypeJSON,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (RisonEncode) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Encode Option", Type: core.ArgOption, Value: []string{"Encode", "Encode Object", "Encode Array", "Encode URI"}},
	}
}

// Run encodes the JSON input as Rison.
func (RisonEncode) Run(in *core.Dish, args []any) (*core.Dish, error) {
	v, err := jsonval.ParseOrdered(in.Bytes())
	if err != nil {
		return nil, fmt.Errorf("encode to Rison: parse JSON input: %w", err)
	}
	var out string
	// The Encode Option arg is a strict ArgOption, so CoerceArgs guarantees one
	// of these four values before Run; "Encode URI" is the default case.
	switch args[0].(string) {
	case "Encode":
		out, err = risonEncodeValue(v)
	case "Encode Object":
		out, err = risonEncodeObjectEntry(v)
	case "Encode Array":
		out, err = risonEncodeArrayEntry(v)
	default: // "Encode URI"
		out, err = risonEncodeURI(v)
	}
	if err != nil {
		return nil, err
	}
	return core.NewDish([]byte(out), core.TypeString), nil
}

// RisonDecode parses a Rison string into a JSON value.
type RisonDecode struct{}

// Meta returns the operation metadata.
func (RisonDecode) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Rison Decode",
		Module:      "Encodings",
		Description: risonDesc,
		InfoURL:     "https://github.com/Nanonid/rison",
		InputType:   core.TypeString,
		OutputType:  core.TypeJSON,
	}
}

// Args returns the argument definitions.
func (RisonDecode) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Decode Option", Type: core.ArgEditableOption, Value: "Decode"},
	}
}

// Run decodes the Rison input into a JSON value (rendered as 4-space JSON).
func (RisonDecode) Run(in *core.Dish, args []any) (*core.Dish, error) {
	input := in.String()
	switch args[0].(string) {
	case "Decode":
		// as-is
	case "Decode Object":
		input = "(" + input + ")"
	case "Decode Array":
		input = "!(" + input + ")"
	default:
		return nil, errors.New("Invalid Decode option") //nolint:staticcheck // verbatim CyberChef OperationError text
	}
	v, err := risonDecodeString(input)
	if err != nil {
		return nil, err
	}
	return core.NewDish([]byte(jsonval.Stringify(v, 4)), core.TypeJSON), nil
}

// --- encoder (port of rison.encode / encode_object / encode_array / encode_uri) ---

// risonNotIdchar are characters illegal inside a Rison identifier.
const risonNotIdchar = " '!:(),*@$"

// risonNotIdstart are characters illegal as the first character of an identifier
// (so identifiers can't be mistaken for numbers).
const risonNotIdstart = "-0123456789"

// risonIDOk reports whether x can be emitted as a bare identifier.
func risonIDOk(x string) bool {
	if x == "" {
		return false
	}
	if strings.IndexByte(risonNotIdstart, x[0]) >= 0 || strings.IndexByte(risonNotIdchar, x[0]) >= 0 {
		return false
	}
	for i := 1; i < len(x); i++ {
		if strings.IndexByte(risonNotIdchar, x[i]) >= 0 {
			return false
		}
	}
	return true
}

// risonEncodeString encodes a string, quoting and escaping only when necessary.
func risonEncodeString(x string) string {
	if x == "" {
		return "''"
	}
	if risonIDOk(x) {
		return x
	}
	var b strings.Builder
	b.WriteByte('\'')
	for i := 0; i < len(x); i++ {
		if c := x[i]; c == '\'' || c == '!' {
			b.WriteByte('!')
		}
		b.WriteByte(x[i])
	}
	b.WriteByte('\'')
	return b.String()
}

// risonEncodeNumber ports rison's number encoding: String(x) with the first '+'
// (from an exponent) stripped; non-finite becomes !n.
func risonEncodeNumber(f float64) string {
	if math.IsInf(f, 0) || math.IsNaN(f) {
		return "!n"
	}
	return strings.Replace(jsonval.FormatNumber(f), "+", "", 1)
}

// risonEncodeValue rison-encodes a JSON value.
func risonEncodeValue(v any) (string, error) {
	switch x := v.(type) {
	case nil:
		return "!n", nil
	case bool:
		if x {
			return "!t", nil
		}
		return "!f", nil
	case float64:
		return risonEncodeNumber(x), nil
	case string:
		return risonEncodeString(x), nil
	case []any:
		return risonEncodeArrayVal(x)
	case jsonval.Object:
		return risonEncodeObjectVal(x)
	default:
		return "", fmt.Errorf("rison can't encode value of type %T", v)
	}
}

func risonEncodeArrayVal(arr []any) (string, error) {
	var b strings.Builder
	b.WriteString("!(")
	for i, e := range arr {
		s, err := risonEncodeValue(e)
		if err != nil {
			return "", err
		}
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(s)
	}
	b.WriteByte(')')
	return b.String(), nil
}

func risonEncodeObjectVal(obj jsonval.Object) (string, error) {
	pairs := make(jsonval.Object, len(obj))
	copy(pairs, obj)
	sort.SliceStable(pairs, func(i, j int) bool { return pairs[i].K < pairs[j].K })
	var b strings.Builder
	b.WriteByte('(')
	for i, p := range pairs {
		s, err := risonEncodeValue(p.V)
		if err != nil {
			return "", err
		}
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(risonEncodeString(p.K))
		b.WriteByte(':')
		b.WriteString(s)
	}
	b.WriteByte(')')
	return b.String(), nil
}

func risonEncodeObjectEntry(v any) (string, error) {
	obj, ok := v.(jsonval.Object)
	if !ok {
		return "", errors.New("rison.encode_object expects an object argument")
	}
	r, err := risonEncodeValue(obj)
	if err != nil {
		return "", err
	}
	return r[1 : len(r)-1], nil // strip surrounding parens
}

func risonEncodeArrayEntry(v any) (string, error) {
	arr, ok := v.([]any)
	if !ok {
		return "", errors.New("rison.encode_array expects an array argument")
	}
	r, err := risonEncodeValue(arr)
	if err != nil {
		return "", err
	}
	return r[2 : len(r)-1], nil // strip "!(" prefix and ")"
}

func risonEncodeURI(v any) (string, error) {
	r, err := risonEncodeValue(v)
	if err != nil {
		return "", err
	}
	return risonQuote(r), nil
}

// risonQuote is rison's tolerant URI encoder. It reproduces the library's bug of
// replacing only the FIRST occurrence of each escaped sequence.
func risonQuote(x string) string {
	if risonURISafe(x) {
		return x
	}
	e := jsEncodeURIComponent(x)
	for _, r := range []struct{ from, to string }{
		{"%2C", ","}, {"%3A", ":"}, {"%40", "@"}, {"%24", "$"}, {"%2F", "/"}, {"%20", "+"},
	} {
		e = strings.Replace(e, r.from, r.to, 1)
	}
	return e
}

// risonURISafe reports whether x needs no URI encoding at all.
func risonURISafe(x string) bool {
	for i := 0; i < len(x); i++ {
		c := x[i]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') ||
			strings.IndexByte("-~!*()_.',:@$/", c) >= 0 {
			continue
		}
		return false
	}
	return true
}

// --- parser (port of rison.parser) --------------------------------------------

type risonParser struct {
	s   string
	idx int
}

// risonDecodeString parses a Rison string into a JSON value.
func risonDecodeString(str string) (any, error) {
	p := &risonParser{s: str}
	v, err := p.readValue()
	if err != nil {
		return nil, err
	}
	if p.next() != -1 {
		enc, _ := risonEncodeValue(str)
		return nil, p.errorf("unable to parse string as rison: '%s'", enc)
	}
	return v, nil
}

func (p *risonParser) errorf(format string, a ...any) error {
	return fmt.Errorf("rison decoder error: "+format, a...)
}

// next returns the next byte (rison uses no whitespace skipping), or -1 at end.
func (p *risonParser) next() int {
	if p.idx >= len(p.s) {
		return -1
	}
	c := p.s[p.idx]
	p.idx++
	return int(c)
}

func (p *risonParser) readValue() (any, error) {
	c := p.next()
	switch c {
	case '!':
		return p.parseBang()
	case '(':
		return p.parseObject()
	case '\'':
		return p.parseQuoted()
	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return p.parseNumber()
	}
	// fall through: parse as a bare identifier starting at the current char
	i := p.idx - 1
	if id, ok := p.matchID(i); ok {
		p.idx = i + len(id)
		return id, nil
	}
	if c != -1 {
		return nil, p.errorf("invalid character: '%c'", c)
	}
	return nil, p.errorf("empty expression")
}

// matchID matches an identifier anchored at position i, or reports false.
func (p *risonParser) matchID(i int) (string, bool) {
	if i < 0 || i >= len(p.s) {
		return "", false
	}
	c := p.s[i]
	if strings.IndexByte(risonNotIdstart, c) >= 0 || strings.IndexByte(risonNotIdchar, c) >= 0 {
		return "", false
	}
	j := i + 1
	for j < len(p.s) && strings.IndexByte(risonNotIdchar, p.s[j]) < 0 {
		j++
	}
	return p.s[i:j], true
}

func (p *risonParser) parseBang() (any, error) {
	c := p.next()
	switch c {
	case -1:
		return nil, p.errorf(`"!" at end of input`)
	case 't':
		return true, nil
	case 'f':
		return false, nil
	case 'n':
		return nil, nil
	case '(':
		return p.parseArray()
	default:
		return nil, p.errorf(`unknown literal: "!%c"`, c)
	}
}

func (p *risonParser) parseArray() (any, error) {
	arr := []any{}
	for {
		c := p.next()
		if c == ')' {
			break
		}
		if c == -1 {
			return nil, p.errorf("unmatched '!('")
		}
		if len(arr) > 0 {
			if c != ',' {
				return nil, p.errorf("missing ','")
			}
		} else if c == ',' {
			return nil, p.errorf("extra ','")
		} else {
			p.idx--
		}
		v, err := p.readValue()
		if err != nil {
			return nil, err
		}
		arr = append(arr, v)
	}
	return arr, nil
}

func (p *risonParser) parseObject() (any, error) {
	obj := jsonval.Object{}
	count := 0
	for {
		c := p.next()
		if c == ')' {
			break
		}
		// rison's object parser lacks an end-of-input guard and stack-overflows
		// on unterminated objects; cchef returns a clean error instead.
		if c == -1 {
			return nil, p.errorf("unmatched '('")
		}
		if count > 0 {
			if c != ',' {
				return nil, p.errorf("missing ','")
			}
		} else if c == ',' {
			return nil, p.errorf("extra ','")
		} else {
			p.idx--
		}
		k, err := p.readValue()
		if err != nil {
			return nil, err
		}
		if p.next() != ':' {
			return nil, p.errorf("missing ':'")
		}
		v, err := p.readValue()
		if err != nil {
			return nil, err
		}
		key := risonKeyString(k)
		if i := jsonval.Index(obj, key); i >= 0 {
			obj[i].V = v
		} else {
			obj = append(obj, jsonval.Pair{K: key, V: v})
		}
		count++
	}
	return obj, nil
}

func (p *risonParser) parseQuoted() (any, error) {
	i := p.idx
	start := i
	var seg strings.Builder
	for {
		if i >= len(p.s) {
			return nil, p.errorf(`unmatched "'"`)
		}
		c := p.s[i]
		i++
		if c == '\'' {
			break
		}
		if c == '!' {
			if start < i-1 {
				seg.WriteString(p.s[start : i-1])
			}
			if i >= len(p.s) {
				return nil, p.errorf(`unmatched "'"`)
			}
			c2 := p.s[i]
			i++
			if c2 == '!' || c2 == '\'' {
				seg.WriteByte(c2)
			} else {
				return nil, p.errorf(`invalid string escape: "!%c"`, c2)
			}
			start = i
		}
	}
	if start < i-1 {
		seg.WriteString(p.s[start : i-1])
	}
	p.idx = i
	return seg.String(), nil
}

func (p *risonParser) parseNumber() (any, error) {
	i := p.idx
	start := i - 1
	state := "int"
	permitted := "-"
	transitions := map[string]string{"int+.": "frac", "int+e": "exp", "frac+e": "exp"}
	for {
		atEnd := i >= len(p.s)
		var c byte
		if !atEnd {
			c = p.s[i]
		}
		i++
		if atEnd {
			break
		}
		if c >= '0' && c <= '9' {
			continue
		}
		if strings.IndexByte(permitted, c) >= 0 {
			permitted = ""
			continue
		}
		state = transitions[state+"+"+strings.ToLower(string(c))]
		if state == "exp" {
			permitted = "-"
		}
		if state == "" {
			break
		}
	}
	i--
	p.idx = i
	numStr := p.s[start:i]
	if numStr == "-" {
		return nil, p.errorf("invalid number")
	}
	f, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		// JavaScript Number() yields NaN for a malformed numeric literal, which
		// the parser returns as-is (JSON.stringify then renders it as null).
		return math.NaN(), nil //nolint:nilerr // matches JS Number()→NaN, not an error
	}
	return f, nil
}

// risonKeyString coerces a parsed object key to a string, as JavaScript does
// when assigning object[key].
func risonKeyString(k any) string {
	switch x := k.(type) {
	case string:
		return x
	case float64:
		return jsonval.FormatNumber(x)
	case bool:
		if x {
			return "true"
		}
		return "false"
	case nil:
		return "null"
	case []any:
		parts := make([]string, len(x))
		for i, e := range x {
			parts[i] = risonKeyString(e)
		}
		return strings.Join(parts, ",")
	default:
		return "[object Object]"
	}
}
