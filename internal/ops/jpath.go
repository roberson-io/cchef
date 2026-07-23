package ops

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(JPathExpression{})
}

// JPathExpression extracts information from a JSON object with a JSONPath query.
// Ported from CyberChef JPathExpression.mjs, which wraps the jsonpath-plus npm
// library. cchef reimplements the JSONPath evaluator over the order-preserving
// jsonvalue.go representation (jsObject / []any) so matched values serialize
// byte-for-byte like jsonpath-plus's results.map(JSON.stringify).join(delimiter),
// including ECMAScript object key ordering.
//
// Known minor divergences (all on degenerate inputs, verified against the
// oracle): a query with a trailing unterminated '[' errors here but is ignored by
// jsonpath-plus; a bare null document returns null here whereas jsonpath-plus
// throws an uncaught TypeError; and string-coercing an array/object inside a
// filter yields "" rather than JavaScript's comma-joined form.
type JPathExpression struct{}

// Meta returns the operation metadata.
func (JPathExpression) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "JPath expression",
		Module:      "Code",
		Description: "Extract information from a JSON object with a JPath query.",
		InfoURL:     "http://goessner.net/articles/JsonPath/",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (JPathExpression) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Query", Type: core.ArgString, Value: ""},
		{Name: "Result delimiter", Type: core.ArgString, Value: `\n`},
	}
}

// Run evaluates the JSONPath query and joins the serialized matches.
func (JPathExpression) Run(in *core.Dish, args []any) (*core.Dish, error) {
	query := args[0].(string)
	delim := parseEscapedChars(args[1].(string))

	root, err := jsonParseOrdered(in.Bytes())
	if err != nil {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError prefix
		return nil, errors.New("Invalid input JSON: " + err.Error())
	}
	segs, err := parseJPath(query)
	if err != nil {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError prefix
		return nil, errors.New("Invalid JPath expression: " + err.Error())
	}
	matches := evalJPath(root, segs)
	parts := make([]string, len(matches))
	for i, m := range matches {
		parts[i] = jsStringify(m, 0)
	}
	return core.NewDish([]byte(strings.Join(parts, delim)), core.TypeString), nil
}

// evalJPath applies the parsed segments left to right, starting from the root.
func evalJPath(root any, segs []jpSeg) []any {
	cur := []any{root}
	for _, seg := range segs {
		var next []any
		for _, node := range cur {
			next = append(next, seg.apply(node)...)
		}
		cur = next
	}
	return cur
}

// jsChildren returns a node's child values in the order jsonpath-plus iterates
// them: object values in ECMAScript key order, array elements in order. Scalars
// have no children.
func jsChildren(node any) []any {
	switch x := node.(type) {
	case jsObject:
		ordered := jsESOrder(x)
		out := make([]any, len(ordered))
		for i, p := range ordered {
			out[i] = p.v
		}
		return out
	case []any:
		return x
	default:
		return nil
	}
}

// jsChild resolves a named child. On objects it is a key lookup; on arrays and
// strings the special "length" property yields the length; nothing else matches.
func jsChild(node any, name string) (any, bool) {
	switch x := node.(type) {
	case jsObject:
		if i := jsIndex(x, name); i >= 0 {
			return x[i].v, true
		}
	case []any:
		if name == "length" {
			return float64(len(x)), true
		}
	case string:
		if name == "length" {
			return float64(len(utf16.Encode([]rune(x)))), true
		}
	}
	return nil, false
}

// jsIndexInto resolves an integer index. On arrays a non-negative in-range index
// yields the element (negative or out-of-range matches nothing, mirroring
// jsonpath-plus); on objects it is a lookup of the decimal key.
func jsIndexInto(node any, i int) (any, bool) {
	switch x := node.(type) {
	case []any:
		if i >= 0 && i < len(x) {
			return x[i], true
		}
	case jsObject:
		if j := jsIndex(x, strconv.Itoa(i)); j >= 0 {
			return x[j].v, true
		}
	}
	return nil, false
}

// descendantOrSelf returns node and all its descendants in document order.
func descendantOrSelf(node any) []any {
	out := []any{node}
	for _, c := range jsChildren(node) {
		out = append(out, descendantOrSelf(c)...)
	}
	return out
}

// jpSegKind enumerates the JSONPath step types.
type jpSegKind int

const (
	segChild jpSegKind = iota
	segWildcard
	segIndexUnion
	segNameUnion
	segSlice
	segFilter
	segScript
	segRecursive
)

// jpSlice holds a parsed [start:end:step] slice. step defaults to 1.
type jpSlice struct {
	start, end, step int
	hasStart, hasEnd bool
}

// jpSeg is one step of a compiled JSONPath.
type jpSeg struct {
	kind    jpSegKind
	name    string
	indices []int
	names   []string
	slice   jpSlice
	expr    *jpExpr
}

// apply expands a single node into the nodes selected by this segment.
func (s jpSeg) apply(node any) []any {
	switch s.kind {
	case segChild:
		if v, ok := jsChild(node, s.name); ok {
			return []any{v}
		}
	case segWildcard:
		return jsChildren(node)
	case segIndexUnion:
		return s.applyIndexUnion(node)
	case segNameUnion:
		// jsonpath-plus resolves a single bracketed name but yields nothing for a
		// comma-separated list of names.
		if len(s.names) == 1 {
			if v, ok := jsChild(node, s.names[0]); ok {
				return []any{v}
			}
		}
	case segSlice:
		return s.applySlice(node)
	case segFilter:
		return s.applyFilter(node)
	case segScript:
		return s.applyScript(node)
	case segRecursive:
		return descendantOrSelf(node)
	}
	return nil
}

// applyIndexUnion resolves each index in the union.
func (s jpSeg) applyIndexUnion(node any) []any {
	var out []any
	for _, i := range s.indices {
		if v, ok := jsIndexInto(node, i); ok {
			out = append(out, v)
		}
	}
	return out
}

// applyFilter keeps each child for which the filter expression is truthy.
func (s jpSeg) applyFilter(node any) []any {
	var out []any
	for _, c := range jsChildren(node) {
		if evalTruthy(s.expr, c) {
			out = append(out, c)
		}
	}
	return out
}

// applySlice implements Python-style array slicing. A step of 0 or less selects
// nothing (jsonpath-plus does not support reverse slices).
func (s jpSeg) applySlice(node any) []any {
	arr, ok := node.([]any)
	if !ok {
		return nil
	}
	n := len(arr)
	step := s.slice.step
	if step <= 0 {
		return nil
	}
	start, end := 0, n
	if s.slice.hasStart {
		start = clampIndex(s.slice.start, n)
	}
	if s.slice.hasEnd {
		end = clampIndex(s.slice.end, n)
	}
	var out []any
	for i := start; i < end; i += step {
		out = append(out, arr[i])
	}
	return out
}

// applyScript evaluates a [(expr)] script and uses its result as an index (for a
// number) or a key (for a string).
func (s jpSeg) applyScript(node any) []any {
	v, ok := evalExpr(s.expr, node)
	if !ok {
		return nil
	}
	switch r := v.(type) {
	case float64:
		if res, ok := jsIndexInto(node, int(r)); ok {
			return []any{res}
		}
	case string:
		if res, ok := jsChild(node, r); ok {
			return []any{res}
		}
	}
	return nil
}

// clampIndex normalizes a (possibly negative) slice bound into [0, n].
func clampIndex(i, n int) int {
	if i < 0 {
		i += n
	}
	if i < 0 {
		return 0
	}
	if i > n {
		return n
	}
	return i
}

// parseJPath compiles a JSONPath query into a sequence of segments. It supports
// the jsonpath-plus subset: child (.name / ['name']), wildcard (* / [*]),
// recursive descent (..), index and index-union, slices, filters ([?(expr)]) and
// script expressions ([(expr)]).
func parseJPath(q string) ([]jpSeg, error) {
	p := &jpParser{s: strings.TrimSpace(q)}
	if p.pos < len(p.s) && p.s[p.pos] == '$' {
		p.pos++
	}
	var segs []jpSeg
	for p.pos < len(p.s) {
		switch c := p.s[p.pos]; c {
		case '.':
			more, err := p.parseDot()
			if err != nil {
				return nil, err
			}
			segs = append(segs, more...)
		case '[':
			seg, err := p.parseBracket()
			if err != nil {
				return nil, err
			}
			segs = append(segs, seg)
		default:
			return nil, fmt.Errorf("unexpected %q at %d", string(c), p.pos)
		}
	}
	if len(segs) == 0 {
		// A bare "$" selects the root value.
		return nil, nil
	}
	return segs, nil
}

type jpParser struct {
	s   string
	pos int
}

// parseDot handles a "." or ".." at the current position, returning the one or
// two segments it introduces.
func (p *jpParser) parseDot() ([]jpSeg, error) {
	if p.pos+1 < len(p.s) && p.s[p.pos+1] == '.' {
		p.pos += 2
		segs := []jpSeg{{kind: segRecursive}}
		if p.pos < len(p.s) && p.s[p.pos] != '[' && p.s[p.pos] != '.' {
			seg, err := p.parseNameStep()
			if err != nil {
				return nil, err
			}
			segs = append(segs, seg)
		}
		return segs, nil
	}
	p.pos++
	seg, err := p.parseNameStep()
	if err != nil {
		return nil, err
	}
	return []jpSeg{seg}, nil
}

// parseNameStep parses a dot step whose leading dot has been consumed: "*" for a
// wildcard or a bare property name.
func (p *jpParser) parseNameStep() (jpSeg, error) {
	if p.pos < len(p.s) && p.s[p.pos] == '*' {
		p.pos++
		return jpSeg{kind: segWildcard}, nil
	}
	start := p.pos
	for p.pos < len(p.s) && p.s[p.pos] != '.' && p.s[p.pos] != '[' {
		p.pos++
	}
	name := p.s[start:p.pos]
	if name == "" {
		return jpSeg{}, fmt.Errorf("empty property name at %d", start)
	}
	return jpSeg{kind: segChild, name: name}, nil
}

// parseBracket parses a [...] step. p.s[p.pos] is '['.
func (p *jpParser) parseBracket() (jpSeg, error) {
	end, err := matchBracket(p.s, p.pos)
	if err != nil {
		return jpSeg{}, err
	}
	inner := strings.TrimSpace(p.s[p.pos+1 : end])
	p.pos = end + 1

	switch {
	case inner == "*":
		return jpSeg{kind: segWildcard}, nil
	case strings.HasPrefix(inner, "?"):
		return parseFilterSeg(inner)
	case strings.HasPrefix(inner, "("):
		return parseScriptSeg(inner)
	case len(splitTopLevel(inner, ':')) > 1:
		return parseSliceSeg(inner)
	default:
		return parseUnionSeg(inner)
	}
}

func parseFilterSeg(inner string) (jpSeg, error) {
	body := strings.TrimSpace(strings.TrimPrefix(inner, "?"))
	if !strings.HasPrefix(body, "(") || !strings.HasSuffix(body, ")") {
		return jpSeg{}, fmt.Errorf("malformed filter %q", inner)
	}
	expr, err := parseJPExpr(body[1 : len(body)-1])
	if err != nil {
		return jpSeg{}, err
	}
	return jpSeg{kind: segFilter, expr: expr}, nil
}

func parseScriptSeg(inner string) (jpSeg, error) {
	if !strings.HasSuffix(inner, ")") {
		return jpSeg{}, fmt.Errorf("malformed script %q", inner)
	}
	expr, err := parseJPExpr(inner[1 : len(inner)-1])
	if err != nil {
		return jpSeg{}, err
	}
	return jpSeg{kind: segScript, expr: expr}, nil
}

func parseSliceSeg(inner string) (jpSeg, error) {
	parts := splitTopLevel(inner, ':')
	sl := jpSlice{step: 1}
	set := func(s string) (int, bool, error) {
		s = strings.TrimSpace(s)
		if s == "" {
			return 0, false, nil
		}
		n, err := strconv.Atoi(s)
		return n, true, err
	}
	var err error
	if sl.start, sl.hasStart, err = set(parts[0]); err != nil {
		return jpSeg{}, err
	}
	if sl.end, sl.hasEnd, err = set(parts[1]); err != nil {
		return jpSeg{}, err
	}
	if len(parts) > 2 {
		if step, ok, err := set(parts[2]); err != nil {
			return jpSeg{}, err
		} else if ok {
			sl.step = step
		}
	}
	return jpSeg{kind: segSlice, slice: sl}, nil
}

func parseUnionSeg(inner string) (jpSeg, error) {
	members := splitTopLevel(inner, ',')
	var indices []int
	var names []string
	for _, m := range members {
		m = strings.TrimSpace(m)
		if q := unquoteJP(m); q != m {
			names = append(names, q)
			continue
		}
		n, err := strconv.Atoi(m)
		if err != nil {
			// A bare unquoted word is treated as a property name.
			names = append(names, m)
			continue
		}
		indices = append(indices, n)
	}
	if len(names) == 0 {
		return jpSeg{kind: segIndexUnion, indices: indices}, nil
	}
	return jpSeg{kind: segNameUnion, names: names}, nil
}

// unquoteJP strips matching single or double quotes; returns s unchanged if not
// quoted.
func unquoteJP(s string) string {
	if len(s) >= 2 && (s[0] == '\'' || s[0] == '"') && s[len(s)-1] == s[0] {
		return s[1 : len(s)-1]
	}
	return s
}

// matchBracket returns the index of the ']' that closes the '[' at open, honoring
// quotes and nested brackets/parens.
func matchBracket(s string, open int) (int, error) {
	depth := 0
	var quote byte
	for i := open; i < len(s); i++ {
		c := s[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '\'' || c == '"':
			quote = c
		case c == '[' || c == '(':
			depth++
		case c == ')':
			depth--
		case c == ']':
			depth--
			if depth == 0 {
				return i, nil
			}
		}
	}
	return 0, fmt.Errorf("unterminated '[' at %d", open)
}

// jpExpr is a filter/script expression AST node.
type jpExpr struct {
	kind        jpExprKind
	op          string
	left, right *jpExpr
	num         float64
	str         string
	b           bool
	path        []jpAccess // for exAt
}

type jpExprKind int

const (
	exBinary jpExprKind = iota
	exUnary
	exAt
	exNumber
	exString
	exBool
	exNull
)

// jpAccess is one @ path accessor: either a named key or an integer index.
type jpAccess struct {
	name    string
	index   int
	isIndex bool
}

// --- tokenizer ---

type jpTok struct {
	kind string // "at" "." "[" "]" "(" ")" num str ident op
	val  string
	num  float64
}

// jpPunct maps single-character tokens to their token kind.
var jpPunct = map[byte]string{'@': "at", '.': ".", '[': "[", ']': "]", '(': "(", ')': ")"}

func lexJPExpr(s string) ([]jpTok, error) {
	var toks []jpTok
	i := 0
	for i < len(s) {
		c := s[i]
		switch {
		case isJPSpace(c):
			i++
		case jpPunct[c] != "":
			toks = append(toks, jpTok{kind: jpPunct[c]})
			i++
		case c == '\'' || c == '"':
			str, ni, err := lexString(s, i)
			if err != nil {
				return nil, err
			}
			toks = append(toks, jpTok{kind: "str", val: str})
			i = ni
		case c >= '0' && c <= '9':
			num, ni := lexNumber(s, i)
			f, _ := strconv.ParseFloat(num, 64)
			toks = append(toks, jpTok{kind: "num", num: f})
			i = ni
		case isJPIdentStart(c):
			start := i
			for i < len(s) && isJPIdentChar(s[i]) {
				i++
			}
			toks = append(toks, jpTok{kind: "ident", val: s[start:i]})
		default:
			op, ni, err := lexOp(s, i)
			if err != nil {
				return nil, err
			}
			toks = append(toks, jpTok{kind: "op", val: op})
			i = ni
		}
	}
	return toks, nil
}

func lexString(s string, i int) (string, int, error) {
	q := s[i]
	i++
	start := i
	for i < len(s) && s[i] != q {
		i++
	}
	if i >= len(s) {
		return "", 0, fmt.Errorf("unterminated string literal")
	}
	return s[start:i], i + 1, nil
}

func lexNumber(s string, i int) (string, int) {
	start := i
	for i < len(s) && ((s[i] >= '0' && s[i] <= '9') || s[i] == '.') {
		i++
	}
	return s[start:i], i
}

func lexOp(s string, i int) (string, int, error) {
	for _, op := range []string{"==", "!=", "<=", ">=", "&&", "||"} {
		if strings.HasPrefix(s[i:], op) {
			return op, i + 2, nil
		}
	}
	switch s[i] {
	case '<', '>', '!', '+', '-', '*', '/':
		return string(s[i]), i + 1, nil
	}
	return "", 0, fmt.Errorf("unexpected %q in expression", string(s[i]))
}

func isJPSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

func isJPIdentStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isJPIdentChar(c byte) bool {
	return isJPIdentStart(c) || (c >= '0' && c <= '9')
}

// parseJPExpr parses a filter/script expression string into an AST.
func parseJPExpr(s string) (*jpExpr, error) {
	toks, err := lexJPExpr(s)
	if err != nil {
		return nil, err
	}
	p := &jpExprParser{toks: toks}
	e, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	if p.pos != len(p.toks) {
		return nil, fmt.Errorf("unexpected token in expression")
	}
	return e, nil
}

type jpExprParser struct {
	toks []jpTok
	pos  int
}

func (p *jpExprParser) peekOp() string {
	if p.pos < len(p.toks) && p.toks[p.pos].kind == "op" {
		return p.toks[p.pos].val
	}
	return ""
}

func (p *jpExprParser) parseOr() (*jpExpr, error) {
	return p.parseBinary(p.parseAnd, "||")
}

func (p *jpExprParser) parseAnd() (*jpExpr, error) {
	return p.parseBinary(p.parseCmp, "&&")
}

// parseBinary folds left-associative operators from ops using the next-tighter
// sub-parser.
func (p *jpExprParser) parseBinary(sub func() (*jpExpr, error), ops ...string) (*jpExpr, error) {
	left, err := sub()
	if err != nil {
		return nil, err
	}
	for {
		op := p.peekOp()
		if !jpContains(ops, op) {
			return left, nil
		}
		p.pos++
		right, err := sub()
		if err != nil {
			return nil, err
		}
		left = &jpExpr{kind: exBinary, op: op, left: left, right: right}
	}
}

func (p *jpExprParser) parseCmp() (*jpExpr, error) {
	left, err := p.parseAdd()
	if err != nil {
		return nil, err
	}
	if op := p.peekOp(); jpContains([]string{"==", "!=", "<", "<=", ">", ">="}, op) {
		p.pos++
		right, err := p.parseAdd()
		if err != nil {
			return nil, err
		}
		return &jpExpr{kind: exBinary, op: op, left: left, right: right}, nil
	}
	return left, nil
}

func (p *jpExprParser) parseAdd() (*jpExpr, error) {
	return p.parseBinary(p.parseMul, "+", "-")
}

func (p *jpExprParser) parseMul() (*jpExpr, error) {
	return p.parseBinary(p.parseUnary, "*", "/")
}

func (p *jpExprParser) parseUnary() (*jpExpr, error) {
	if op := p.peekOp(); op == "!" || op == "-" {
		p.pos++
		operand, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &jpExpr{kind: exUnary, op: op, left: operand}, nil
	}
	return p.parsePrimary()
}

func (p *jpExprParser) parsePrimary() (*jpExpr, error) {
	if p.pos >= len(p.toks) {
		return nil, fmt.Errorf("unexpected end of expression")
	}
	t := p.toks[p.pos]
	switch t.kind {
	case "(":
		p.pos++
		e, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if p.pos >= len(p.toks) || p.toks[p.pos].kind != ")" {
			return nil, fmt.Errorf("missing ')' in expression")
		}
		p.pos++
		return e, nil
	case "num":
		p.pos++
		return &jpExpr{kind: exNumber, num: t.num}, nil
	case "str":
		p.pos++
		return &jpExpr{kind: exString, str: t.val}, nil
	case "ident":
		p.pos++
		switch t.val {
		case "true":
			return &jpExpr{kind: exBool, b: true}, nil
		case "false":
			return &jpExpr{kind: exBool, b: false}, nil
		case "null":
			return &jpExpr{kind: exNull}, nil
		}
		return nil, fmt.Errorf("unexpected identifier %q", t.val)
	case "at":
		return p.parseAt()
	}
	return nil, fmt.Errorf("unexpected %q in expression", t.kind)
}

// parseAt parses "@" followed by one or more accessors. A bare "@" is an error,
// matching jsonpath-plus.
func (p *jpExprParser) parseAt() (*jpExpr, error) {
	p.pos++ // '@'
	var path []jpAccess
	for p.pos < len(p.toks) {
		switch p.toks[p.pos].kind {
		case ".":
			p.pos++
			if p.pos >= len(p.toks) || p.toks[p.pos].kind != "ident" {
				return nil, fmt.Errorf("expected name after '.'")
			}
			path = append(path, jpAccess{name: p.toks[p.pos].val})
			p.pos++
		case "[":
			acc, err := p.parseAtBracket()
			if err != nil {
				return nil, err
			}
			path = append(path, acc)
		default:
			goto done
		}
	}
done:
	if len(path) == 0 {
		return nil, fmt.Errorf(`unexpected bare "@"`)
	}
	return &jpExpr{kind: exAt, path: path}, nil
}

func (p *jpExprParser) parseAtBracket() (jpAccess, error) {
	p.pos++ // '['
	if p.pos >= len(p.toks) {
		return jpAccess{}, fmt.Errorf("unterminated '[' in @ path")
	}
	t := p.toks[p.pos]
	var acc jpAccess
	switch t.kind {
	case "num":
		acc = jpAccess{index: int(t.num), isIndex: true}
	case "str":
		acc = jpAccess{name: t.val}
	default:
		return jpAccess{}, fmt.Errorf("invalid @ path index")
	}
	p.pos++
	if p.pos >= len(p.toks) || p.toks[p.pos].kind != "]" {
		return jpAccess{}, fmt.Errorf("missing ']' in @ path")
	}
	p.pos++
	return acc, nil
}

func jpContains(ss []string, s string) bool {
	if s == "" {
		return false
	}
	return slices.Contains(ss, s)
}

// evalExpr evaluates a filter/script expression against the context value @. The
// bool result is false when the expression references a missing path (undefined).
func evalExpr(e *jpExpr, ctx any) (any, bool) {
	switch e.kind {
	case exNumber:
		return e.num, true
	case exString:
		return e.str, true
	case exBool:
		return e.b, true
	case exNull:
		return nil, true
	case exAt:
		return evalAt(e.path, ctx)
	case exUnary:
		return evalUnary(e, ctx)
	default: // exBinary
		return evalBinary(e, ctx)
	}
}

// evalTruthy reports the JavaScript truthiness of an expression's value.
func evalTruthy(e *jpExpr, ctx any) bool {
	v, ok := evalExpr(e, ctx)
	return ok && jsTruthyVal(v)
}

func evalAt(path []jpAccess, ctx any) (any, bool) {
	v := ctx
	for _, acc := range path {
		var ok bool
		if acc.isIndex {
			v, ok = jsIndexInto(v, acc.index)
		} else {
			v, ok = jsChild(v, acc.name)
		}
		if !ok {
			return nil, false
		}
	}
	return v, true
}

func evalUnary(e *jpExpr, ctx any) (any, bool) {
	if e.op == "!" {
		return !evalTruthy(e.left, ctx), true
	}
	// unary minus
	v, ok := evalExpr(e.left, ctx)
	if !ok {
		return nil, false
	}
	if f, ok := v.(float64); ok {
		return -f, true
	}
	return nil, false
}

func evalBinary(e *jpExpr, ctx any) (any, bool) {
	switch e.op {
	case "&&":
		return evalTruthy(e.left, ctx) && evalTruthy(e.right, ctx), true
	case "||":
		return evalTruthy(e.left, ctx) || evalTruthy(e.right, ctx), true
	case "+", "-", "*", "/":
		return evalArith(e, ctx)
	default:
		return evalCompare(e, ctx)
	}
}

func evalArith(e *jpExpr, ctx any) (any, bool) {
	lv, lok := evalExpr(e.left, ctx)
	rv, rok := evalExpr(e.right, ctx)
	if !lok || !rok {
		return nil, false
	}
	if e.op == "+" {
		if ls, lok := lv.(string); lok {
			return ls + jsCoerceString(rv), true
		}
		if rs, rok := rv.(string); rok {
			return jsCoerceString(lv) + rs, true
		}
	}
	lf, lok := lv.(float64)
	rf, rok := rv.(float64)
	if !lok || !rok {
		return nil, false
	}
	switch e.op {
	case "+":
		return lf + rf, true
	case "-":
		return lf - rf, true
	case "*":
		return lf * rf, true
	default: // "/"
		return lf / rf, true
	}
}

func evalCompare(e *jpExpr, ctx any) (any, bool) {
	lv, lok := evalExpr(e.left, ctx)
	rv, rok := evalExpr(e.right, ctx)
	if !lok || !rok {
		// undefined operand: only (in)equality is defined and yields not-equal.
		switch e.op {
		case "==":
			return !lok && !rok, true
		case "!=":
			return lok != rok, true
		}
		return false, true
	}
	switch e.op {
	case "==":
		return jsEqual(lv, rv), true
	case "!=":
		return !jsEqual(lv, rv), true
	default:
		return jsOrder(e.op, lv, rv), true
	}
}

// jsEqual implements JavaScript's loose == for the value types JSONPath produces.
func jsEqual(a, b any) bool {
	switch av := a.(type) {
	case float64:
		switch bv := b.(type) {
		case float64:
			return av == bv
		case string:
			if f, err := strconv.ParseFloat(bv, 64); err == nil {
				return av == f
			}
		}
	case string:
		switch bv := b.(type) {
		case string:
			return av == bv
		case float64:
			if f, err := strconv.ParseFloat(av, 64); err == nil {
				return f == bv
			}
		}
	case bool:
		bv, ok := b.(bool)
		return ok && av == bv
	case nil:
		return b == nil
	}
	return false
}

// jsOrder implements <, <=, >, >= for numbers and strings.
func jsOrder(op string, a, b any) bool {
	af, aok := a.(float64)
	bf, bok := b.(float64)
	if aok && bok {
		return orderResult(op, af > bf, af == bf)
	}
	as, aok := a.(string)
	bs, bok := b.(string)
	if aok && bok {
		return orderResult(op, as > bs, as == bs)
	}
	return false
}

func orderResult(op string, gt, eq bool) bool {
	switch op {
	case "<":
		return !gt && !eq
	case "<=":
		return !gt
	case ">":
		return gt
	default: // ">="
		return gt || eq
	}
}

func jsTruthyVal(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case float64:
		return x != 0 && !math.IsNaN(x)
	case string:
		return x != ""
	default: // []any, jsObject
		return true
	}
}

// jsCoerceString renders a value the way JavaScript's String()/+ coercion does
// for the primitives that appear in filters.
func jsCoerceString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return jsFormatNumber(x)
	case bool:
		return strconv.FormatBool(x)
	case nil:
		return "null"
	}
	return ""
}
