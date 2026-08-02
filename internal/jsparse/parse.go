// Package jsparse parses JavaScript into an ESTree-shaped syntax tree.
//
// It is a port of esprima, the parser CyberChef uses, so the tree it returns
// matches esprima's node for node — the JavaScript Parser operation prints it
// directly, and JavaScript Beautify walks it. The tree is built out of
// [github.com/roberson-io/cchef/internal/jsonval.Object] values rather than Go
// structs, because a node's shape varies by type and the output has to
// serialise in esprima's key order.
//
// [Parse] returns the tree alone; [ParseFull] also returns the comments and
// tokens a beautifier needs to put things back where they were. The Field
// accessors read a node's properties, which are typed as any.
package jsparse

// Recursive-descent parser for the JavaScript Parser operation — a
// transliteration of esprima's Parser (esprima/dist/esprima.js module 8). AST
// nodes are built as ordered jsonval.Object values mirroring esprima's Node
// constructors (module 7), so JSON.stringify(ast, null, 2) via the shared
// serializer reproduces esprima byte-for-byte. Errors are raised as panics of
// *jsSyntaxError and recovered at the entry point.
//
// In-progress: this session implements the expression grammar (literals,
// identifiers, unary/update, binary precedence incl. **, logical, conditional,
// assignment, sequence, member/computed/call) and the statement spine (program
// with directive prologue, expression/empty/block statements). Declarations
// (var/let/const/function/class), control-flow statements, arrow functions,
// templates, regex, destructuring, yield/async, and the loc/range/tokens/comment
// options are not yet ported and raise an error rather than diverging silently.

import (
	"sort"

	"github.com/roberson-io/cchef/internal/jsonval"
)

// --- AST node constructors (field order per esprima Node, module 7) ---

func jsNodeScript(body []any) jsonval.Object {
	return jsonval.Object{{K: "type", V: "Program"}, {K: "body", V: body}, {K: "sourceType", V: "script"}}
}

func jsNodeExpressionStatement(expr any) jsonval.Object {
	return jsonval.Object{{K: "type", V: "ExpressionStatement"}, {K: "expression", V: expr}}
}

func jsNodeDirective(expr any, directive string) jsonval.Object {
	return jsonval.Object{{K: "type", V: "ExpressionStatement"}, {K: "expression", V: expr}, {K: "directive", V: directive}}
}

func jsNodeEmptyStatement() jsonval.Object {
	return jsonval.Object{{K: "type", V: "EmptyStatement"}}
}

func jsNodeBlockStatement(body []any) jsonval.Object {
	return jsonval.Object{{K: "type", V: "BlockStatement"}, {K: "body", V: body}}
}

func jsNodeLiteral(value any, raw string) jsonval.Object {
	return jsonval.Object{{K: "type", V: "Literal"}, {K: "value", V: value}, {K: "raw", V: raw}}
}

func jsNodeIdentifier(name string) jsonval.Object {
	return jsonval.Object{{K: "type", V: "Identifier"}, {K: "name", V: name}}
}

func jsNodeBinary(op string, left, right any) jsonval.Object {
	typ := "BinaryExpression"
	if op == "||" || op == "&&" {
		typ = "LogicalExpression"
	}
	return jsonval.Object{{K: "type", V: typ}, {K: "operator", V: op}, {K: "left", V: left}, {K: "right", V: right}}
}

func jsNodeUnary(op string, arg any) jsonval.Object {
	return jsonval.Object{{K: "type", V: "UnaryExpression"}, {K: "operator", V: op}, {K: "argument", V: arg}, {K: "prefix", V: true}}
}

func jsNodeUpdate(op string, arg any, prefix bool) jsonval.Object {
	return jsonval.Object{{K: "type", V: "UpdateExpression"}, {K: "operator", V: op}, {K: "argument", V: arg}, {K: "prefix", V: prefix}}
}

func jsNodeAssignment(op string, left, right any) jsonval.Object {
	return jsonval.Object{{K: "type", V: "AssignmentExpression"}, {K: "operator", V: op}, {K: "left", V: left}, {K: "right", V: right}}
}

func jsNodeConditional(test, cons, alt any) jsonval.Object {
	return jsonval.Object{{K: "type", V: "ConditionalExpression"}, {K: "test", V: test}, {K: "consequent", V: cons}, {K: "alternate", V: alt}}
}

func jsNodeSequence(exprs []any) jsonval.Object {
	return jsonval.Object{{K: "type", V: "SequenceExpression"}, {K: "expressions", V: exprs}}
}

func jsNodeStaticMember(obj, prop any) jsonval.Object {
	return jsonval.Object{{K: "type", V: "MemberExpression"}, {K: "computed", V: false}, {K: "object", V: obj}, {K: "property", V: prop}}
}

func jsNodeComputedMember(obj, prop any) jsonval.Object {
	return jsonval.Object{{K: "type", V: "MemberExpression"}, {K: "computed", V: true}, {K: "object", V: obj}, {K: "property", V: prop}}
}

func jsNodeCall(callee any, args []any) jsonval.Object {
	return jsonval.Object{{K: "type", V: "CallExpression"}, {K: "callee", V: callee}, {K: "arguments", V: args}}
}

func jsNodeThis() jsonval.Object { return jsonval.Object{{K: "type", V: "ThisExpression"}} }

func jsNodeRegexLiteral(raw, pattern, flags string) jsonval.Object {
	regex := jsonval.Object{{K: "pattern", V: pattern}, {K: "flags", V: flags}}
	return jsonval.Object{{K: "type", V: "Literal"}, {K: "value", V: jsonval.Object{}}, {K: "raw", V: raw}, {K: "regex", V: regex}}
}

func jsNodeArray(elements []any) jsonval.Object {
	return jsonval.Object{{K: "type", V: "ArrayExpression"}, {K: "elements", V: elements}}
}

func jsNodeObject(properties []any) jsonval.Object {
	return jsonval.Object{{K: "type", V: "ObjectExpression"}, {K: "properties", V: properties}}
}

func jsNodeProperty(kind string, key, value any, computed, method, shorthand bool) jsonval.Object {
	return jsonval.Object{
		{K: "type", V: "Property"},
		{K: "key", V: key},
		{K: "computed", V: computed},
		{K: "value", V: value},
		{K: "kind", V: kind},
		{K: "method", V: method},
		{K: "shorthand", V: shorthand},
	}
}

func jsNodeSpreadElement(arg any) jsonval.Object {
	return jsonval.Object{{K: "type", V: "SpreadElement"}, {K: "argument", V: arg}}
}

func jsNodeRestElement(arg any) jsonval.Object {
	return jsonval.Object{{K: "type", V: "RestElement"}, {K: "argument", V: arg}}
}

func jsNodeNew(callee any, args []any) jsonval.Object {
	return jsonval.Object{{K: "type", V: "NewExpression"}, {K: "callee", V: callee}, {K: "arguments", V: args}}
}

func jsNodeMetaProperty(meta, property any) jsonval.Object {
	return jsonval.Object{{K: "type", V: "MetaProperty"}, {K: "meta", V: meta}, {K: "property", V: property}}
}

func jsNodeYield(argument any, delegate bool) jsonval.Object {
	return jsonval.Object{{K: "type", V: "YieldExpression"}, {K: "argument", V: argument}, {K: "delegate", V: delegate}}
}

func jsNodeArrayPattern(elements []any) jsonval.Object {
	return jsonval.Object{{K: "type", V: "ArrayPattern"}, {K: "elements", V: elements}}
}

func jsNodeObjectPattern(properties []any) jsonval.Object {
	return jsonval.Object{{K: "type", V: "ObjectPattern"}, {K: "properties", V: properties}}
}

func jsNodeAssignmentPattern(left, right any) jsonval.Object {
	return jsonval.Object{{K: "type", V: "AssignmentPattern"}, {K: "left", V: left}, {K: "right", V: right}}
}

func jsNodeVariableDeclaration(decls []any, kind string) jsonval.Object {
	return jsonval.Object{{K: "type", V: "VariableDeclaration"}, {K: "declarations", V: decls}, {K: "kind", V: kind}}
}

func jsNodeVariableDeclarator(id, init any) jsonval.Object {
	return jsonval.Object{{K: "type", V: "VariableDeclarator"}, {K: "id", V: id}, {K: "init", V: init}}
}

func jsNodeFunction(kind string, id, params, body any, generator, async bool) jsonval.Object {
	return jsonval.Object{
		{K: "type", V: kind},
		{K: "id", V: id},
		{K: "params", V: params},
		{K: "body", V: body},
		{K: "generator", V: generator},
		{K: "expression", V: false},
		{K: "async", V: async},
	}
}

func jsNodeArrowFunction(params []any, body any, expression, async bool) jsonval.Object {
	return jsonval.Object{
		{K: "type", V: "ArrowFunctionExpression"},
		{K: "id", V: nil},
		{K: "params", V: params},
		{K: "body", V: body},
		{K: "generator", V: false},
		{K: "expression", V: expression},
		{K: "async", V: async},
	}
}

func jsNodeAwait(argument any) jsonval.Object {
	return jsonval.Object{{K: "type", V: "AwaitExpression"}, {K: "argument", V: argument}}
}

func jsNodeSuper() jsonval.Object { return jsonval.Object{{K: "type", V: "Super"}} }

func jsNodeClass(kind string, id, superClass, body any) jsonval.Object {
	return jsonval.Object{
		{K: "type", V: kind},
		{K: "id", V: id},
		{K: "superClass", V: superClass},
		{K: "body", V: body},
	}
}

func jsNodeClassBody(body []any) jsonval.Object {
	return jsonval.Object{{K: "type", V: "ClassBody"}, {K: "body", V: body}}
}

func jsNodeMethodDefinition(key any, computed bool, value any, kind string, static bool) jsonval.Object {
	return jsonval.Object{
		{K: "type", V: "MethodDefinition"},
		{K: "key", V: key},
		{K: "computed", V: computed},
		{K: "value", V: value},
		{K: "kind", V: kind},
		{K: "static", V: static},
	}
}

// jsArrowParams is the internal cover-grammar placeholder (esprima's
// ArrowParameterPlaceHolder): a parenthesised list that turned out to be arrow
// parameters. It is never serialized. async marks `async (...) =>` heads.
type jsArrowParams struct {
	params []any
	async  bool
}

func jsNodeIf(test, cons, alt any) jsonval.Object {
	return jsonval.Object{{K: "type", V: "IfStatement"}, {K: "test", V: test}, {K: "consequent", V: cons}, {K: "alternate", V: alt}}
}

func jsNodeWhile(test, body any) jsonval.Object {
	return jsonval.Object{{K: "type", V: "WhileStatement"}, {K: "test", V: test}, {K: "body", V: body}}
}

func jsNodeDoWhile(body, test any) jsonval.Object {
	return jsonval.Object{{K: "type", V: "DoWhileStatement"}, {K: "body", V: body}, {K: "test", V: test}}
}

func jsNodeFor(init, test, update, body any) jsonval.Object {
	return jsonval.Object{{K: "type", V: "ForStatement"}, {K: "init", V: init}, {K: "test", V: test}, {K: "update", V: update}, {K: "body", V: body}}
}

func jsNodeForIn(left, right, body any) jsonval.Object {
	return jsonval.Object{{K: "type", V: "ForInStatement"}, {K: "left", V: left}, {K: "right", V: right}, {K: "body", V: body}, {K: "each", V: false}}
}

func jsNodeForOf(left, right, body any) jsonval.Object {
	return jsonval.Object{{K: "type", V: "ForOfStatement"}, {K: "left", V: left}, {K: "right", V: right}, {K: "body", V: body}}
}

func jsNodeReturn(arg any) jsonval.Object {
	return jsonval.Object{{K: "type", V: "ReturnStatement"}, {K: "argument", V: arg}}
}

func jsNodeBreak(label any) jsonval.Object {
	return jsonval.Object{{K: "type", V: "BreakStatement"}, {K: "label", V: label}}
}

func jsNodeContinue(label any) jsonval.Object {
	return jsonval.Object{{K: "type", V: "ContinueStatement"}, {K: "label", V: label}}
}

func jsNodeThrow(arg any) jsonval.Object {
	return jsonval.Object{{K: "type", V: "ThrowStatement"}, {K: "argument", V: arg}}
}

func jsNodeSwitch(disc any, cases []any) jsonval.Object {
	return jsonval.Object{{K: "type", V: "SwitchStatement"}, {K: "discriminant", V: disc}, {K: "cases", V: cases}}
}

func jsNodeSwitchCase(test any, consequent []any) jsonval.Object {
	return jsonval.Object{{K: "type", V: "SwitchCase"}, {K: "test", V: test}, {K: "consequent", V: consequent}}
}

func jsNodeTry(block, handler, finalizer any) jsonval.Object {
	return jsonval.Object{{K: "type", V: "TryStatement"}, {K: "block", V: block}, {K: "handler", V: handler}, {K: "finalizer", V: finalizer}}
}

func jsNodeCatch(param, body any) jsonval.Object {
	return jsonval.Object{{K: "type", V: "CatchClause"}, {K: "param", V: param}, {K: "body", V: body}}
}

func jsNodeWith(obj, body any) jsonval.Object {
	return jsonval.Object{{K: "type", V: "WithStatement"}, {K: "object", V: obj}, {K: "body", V: body}}
}

func jsNodeLabeled(label, body any) jsonval.Object {
	return jsonval.Object{{K: "type", V: "LabeledStatement"}, {K: "label", V: label}, {K: "body", V: body}}
}

func jsNodeDebugger() jsonval.Object { return jsonval.Object{{K: "type", V: "DebuggerStatement"}} }

func jsNodeTemplateElement(raw, cooked string, tail bool) jsonval.Object {
	value := jsonval.Object{{K: "raw", V: raw}, {K: "cooked", V: cooked}}
	return jsonval.Object{{K: "type", V: "TemplateElement"}, {K: "value", V: value}, {K: "tail", V: tail}}
}

func jsNodeTemplateLiteral(quasis, expressions []any) jsonval.Object {
	return jsonval.Object{{K: "type", V: "TemplateLiteral"}, {K: "quasis", V: quasis}, {K: "expressions", V: expressions}}
}

func jsNodeTaggedTemplate(tag, quasi any) jsonval.Object {
	return jsonval.Object{{K: "type", V: "TaggedTemplateExpression"}, {K: "tag", V: tag}, {K: "quasi", V: quasi}}
}

// NodeType returns the "type" field of an AST node.
func NodeType(n any) string {
	obj, ok := n.(jsonval.Object)
	if !ok {
		return ""
	}
	for _, p := range obj {
		if p.K == "type" {
			s, _ := p.V.(string)
			return s
		}
	}
	return ""
}

// --- parser messages ---

const (
	jsMsgInvalidLHSInAssignment = "Invalid left-hand side in assignment"
	jsMsgUnexpectedEOS          = "Unexpected end of input"
)

// --- parser ---

type jsParserContext struct {
	allowIn                        bool
	allowYield                     bool
	await                          bool
	isAssignmentTarget             bool
	isBindingElement               bool
	strict                         bool
	inFunctionBody                 bool
	inIteration                    bool
	inSwitch                       bool
	labelSet                       map[string]bool
	firstCoverInitializedNameError *jsToken
}

type jsParser struct {
	scanner           *jsScanner
	lookahead         jsToken
	hasLineTerminator bool
	context           jsParserContext

	operatorPrecedence map[string]int

	track   bool             // when set, node ranges and tokens are collected (ParseFull)
	tokens  []jsonval.Object // collected tokens (type/value/range), source order
	lastEnd int              // end offset of the last consumed token (esprima lastMarker.index)
}

// newJSParserBare builds the parser without priming the first token, so callers
// can set tracking flags before the initial scan (which may skip leading
// comments).
func newJSParserBare(code string) *jsParser {
	sc := newJSScanner(code)
	return &jsParser{
		scanner: sc,
		lookahead: jsToken{
			typ: tkEOF, value: "", lineNumber: sc.lineNumber,
		},
		context: jsParserContext{allowIn: true, allowYield: true, isAssignmentTarget: true, isBindingElement: true, labelSet: map[string]bool{}},
		operatorPrecedence: map[string]int{
			")": 0, ";": 0, ",": 0, "=": 0, "]": 0,
			"||": 1, "&&": 2, "|": 3, "^": 4, "&": 5,
			"==": 6, "!=": 6, "===": 6, "!==": 6,
			"<": 7, ">": 7, "<=": 7, ">=": 7,
			"<<": 8, ">>": 8, ">>>": 8,
			"+": 9, "-": 9, "*": 11, "/": 11, "%": 11,
		},
	}
}

func newJSParser(code string) *jsParser {
	p := newJSParserBare(code)
	p.nextToken()
	return p
}

// nextToken advances to the next token, returning the previous lookahead. When
// tracking, it records the consumed (non-EOF) token and updates lastEnd. Peeks
// via saveState/lex/restoreState bypass nextToken, so they are not recorded.
func (p *jsParser) nextToken() jsToken {
	token := p.lookahead
	if p.track {
		p.lastEnd = token.end
		if token.typ != tkEOF {
			p.tokens = append(p.tokens, p.convertToken(token))
		}
	}
	p.scanner.scanComments()
	next := p.scanner.lex()
	if p.track && p.context.strict && next.typ == tkIdentifier {
		if v, ok := next.value.(string); ok && jsIsStrictModeReservedWord(v) {
			next.typ = tkKeyword
		}
	}
	p.hasLineTerminator = token.lineNumber != next.lineNumber
	p.lookahead = next
	return token
}

// jsIsStrictModeReservedWord reports whether id is a strict-mode reserved word
// (esprima's Scanner.isStrictModeReservedWord); such an identifier is promoted
// to a Keyword token in strict mode.
func jsIsStrictModeReservedWord(id string) bool {
	switch id {
	case "implements", "interface", "package", "private", "protected", "public", "static", "yield", "let":
		return true
	}
	return false
}

// convertToken builds a token record (esprima's convertToken): the esprima token
// name, the raw source text, and the code-unit range.
func (p *jsParser) convertToken(t jsToken) jsonval.Object {
	return jsonval.Object{
		{K: "type", V: jsTokenTypeName(t.typ)},
		{K: "value", V: p.scanner.rawToken(t)},
		{K: "range", V: []any{float64(t.start), float64(t.end)}},
	}
}

// jsTokenName maps a token type to esprima's TokenName string.
func jsTokenTypeName(typ int) string {
	switch typ {
	case tkBooleanLiteral:
		return "Boolean"
	case tkIdentifier:
		return "Identifier"
	case tkKeyword:
		return "Keyword"
	case tkNullLiteral:
		return "Null"
	case tkNumericLiteral:
		return "Numeric"
	case tkPunctuator:
		return "Punctuator"
	case tkStringLiteral:
		return "String"
	case tkRegularExpression:
		return "RegularExpression"
	case tkTemplate:
		return "Template"
	}
	return ""
}

// finalize attaches a range to a freshly-created node (esprima's finalize).
// start is the marker index (start of the node's first token); the end is the
// last consumed token's end. Non-jsonval.Object values (arrow placeholders) and the
// untracked path pass through unchanged.
func (p *jsParser) finalize(start int, node any) any {
	if !p.track {
		return node
	}
	obj, ok := node.(jsonval.Object)
	if !ok {
		return node
	}
	return append(obj, jsonval.Pair{K: "range", V: []any{float64(start), float64(p.lastEnd)}})
}

// carryRange copies src's range onto dst (esprima mutates patterns in place,
// preserving the original expression's range). No-op when untracked.
func (p *jsParser) carryRange(src, dst any) any {
	if !p.track {
		return dst
	}
	r := Field(src, "range")
	obj, ok := dst.(jsonval.Object)
	if r == nil || !ok {
		return dst
	}
	return append(obj, jsonval.Pair{K: "range", V: r})
}

func (p *jsParser) getTokenRaw(t jsToken) string { return p.scanner.rawToken(t) }

// nextRegexToken re-scans the current position as a regular-expression literal
// (the normal lexer scanned the leading '/' as a division punctuator).
func (p *jsParser) nextRegexToken() jsToken {
	token := p.scanner.scanRegExp()
	p.lookahead = token
	p.nextToken()
	return token
}

func (p *jsParser) match(value string) bool {
	return p.lookahead.typ == tkPunctuator && p.lookahead.value == value
}

func (p *jsParser) matchKeyword(kw string) bool {
	return p.lookahead.typ == tkKeyword && p.lookahead.value == kw
}

func (p *jsParser) matchContextualKeyword(kw string) bool {
	return p.lookahead.typ == tkIdentifier && p.lookahead.value == kw
}

// matchAsyncFunction reports whether the lookahead is `async` immediately
// followed (on the same line) by the `function` keyword.
func (p *jsParser) matchAsyncFunction() bool {
	if !p.matchContextualKeyword("async") {
		return false
	}
	state := p.scanner.saveState()
	p.scanner.scanComments()
	next := p.scanner.lex()
	p.scanner.restoreState(state)
	return state.lineNumber == next.lineNumber && next.typ == tkKeyword && next.value == "function"
}

// throwError raises a parser error (esprima's throwError) with the given text.
func (p *jsParser) throwError(msg string) {
	panic(&jsSyntaxError{line: p.scanner.lineNumber, description: msg})
}

func (p *jsParser) matchAssign() bool {
	if p.lookahead.typ != tkPunctuator {
		return false
	}
	switch p.lookahead.value.(string) {
	case "=", "*=", "**=", "/=", "%=", "+=", "-=", "<<=", ">>=", ">>>=", "&=", "^=", "|=":
		return true
	}
	return false
}

func jsIsIdentifierName(t jsToken) bool {
	switch t.typ {
	case tkIdentifier, tkKeyword, tkBooleanLiteral, tkNullLiteral:
		return true
	}
	return false
}

func (p *jsParser) throwUnexpectedToken(t jsToken) {
	panic(&jsSyntaxError{line: t.lineNumber, column: t.start - t.lineStart + 1, index: t.start, description: jsUnexpectedDesc(t)})
}

func jsUnexpectedDesc(t jsToken) string {
	switch t.typ {
	case tkEOF:
		return jsMsgUnexpectedEOS
	case tkNumericLiteral:
		return "Unexpected number"
	case tkStringLiteral:
		return "Unexpected string"
	case tkIdentifier:
		return "Unexpected identifier"
	}
	if v, ok := t.value.(string); ok && v != "" {
		if t.typ == tkKeyword && jsFutureReservedWord(v) {
			return "Unexpected reserved word"
		}
		return "Unexpected token " + v
	}
	return "Unexpected token ILLEGAL"
}

// jsFutureReservedWord reports whether id is a future reserved word (esprima's
// Scanner.isFutureReservedWord), which surfaces as "Unexpected reserved word".
func jsFutureReservedWord(id string) bool {
	switch id {
	case "enum", "export", "import", "super":
		return true
	}
	return false
}

func (p *jsParser) tolerateError(msg string) {
	panic(&jsSyntaxError{line: p.scanner.lineNumber, description: msg})
}

func (p *jsParser) expect(value string) {
	token := p.nextToken()
	if token.typ != tkPunctuator || token.value != value {
		p.throwUnexpectedToken(token)
	}
}

func (p *jsParser) expectKeyword(keyword string) {
	token := p.nextToken()
	if token.typ != tkKeyword || token.value != keyword {
		p.throwUnexpectedToken(token)
	}
}

func (p *jsParser) expectCommaSeparator() { p.expect(",") }

func (p *jsParser) consumeSemicolon() {
	if p.match(";") {
		p.nextToken()
	} else if !p.hasLineTerminator {
		if p.lookahead.typ != tkEOF && !p.match("}") {
			p.throwUnexpectedToken(p.lookahead)
		}
	}
}

// --- cover grammar helpers ---

func (p *jsParser) isolateCoverGrammar(fn func() any) any {
	prevBind, prevAssign, prevErr := p.context.isBindingElement, p.context.isAssignmentTarget, p.context.firstCoverInitializedNameError
	p.context.isBindingElement, p.context.isAssignmentTarget, p.context.firstCoverInitializedNameError = true, true, nil
	result := fn()
	if p.context.firstCoverInitializedNameError != nil {
		p.throwUnexpectedToken(*p.context.firstCoverInitializedNameError)
	}
	p.context.isBindingElement, p.context.isAssignmentTarget, p.context.firstCoverInitializedNameError = prevBind, prevAssign, prevErr
	return result
}

func (p *jsParser) inheritCoverGrammar(fn func() any) any {
	prevBind, prevAssign, prevErr := p.context.isBindingElement, p.context.isAssignmentTarget, p.context.firstCoverInitializedNameError
	p.context.isBindingElement, p.context.isAssignmentTarget, p.context.firstCoverInitializedNameError = true, true, nil
	result := fn()
	p.context.isBindingElement = p.context.isBindingElement && prevBind
	p.context.isAssignmentTarget = p.context.isAssignmentTarget && prevAssign
	if prevErr != nil {
		p.context.firstCoverInitializedNameError = prevErr
	}
	return result
}

// --- expressions ---

func (p *jsParser) parsePrimaryExpression() any {
	start := p.lookahead.start
	switch p.lookahead.typ {
	case tkIdentifier:
		if p.matchAsyncFunction() {
			return p.parseFunctionExpression()
		}
		return p.finalize(start, jsNodeIdentifier(p.nextToken().value.(string)))
	case tkNumericLiteral, tkStringLiteral:
		p.context.isAssignmentTarget = false
		p.context.isBindingElement = false
		token := p.nextToken()
		return p.finalize(start, jsNodeLiteral(token.value, p.getTokenRaw(token)))
	case tkBooleanLiteral:
		p.context.isAssignmentTarget = false
		p.context.isBindingElement = false
		token := p.nextToken()
		return p.finalize(start, jsNodeLiteral(token.value.(string) == "true", p.getTokenRaw(token)))
	case tkNullLiteral:
		p.context.isAssignmentTarget = false
		p.context.isBindingElement = false
		token := p.nextToken()
		return p.finalize(start, jsNodeLiteral(nil, p.getTokenRaw(token)))
	case tkTemplate:
		return p.parseTemplateLiteral()
	case tkPunctuator:
		switch p.lookahead.value {
		case "(":
			p.context.isBindingElement = false
			return p.inheritCoverGrammar(p.parseGroupExpression)
		case "[":
			return p.inheritCoverGrammar(p.parseArrayInitializer)
		case "{":
			return p.inheritCoverGrammar(p.parseObjectInitializer)
		case "/", "/=":
			p.context.isAssignmentTarget = false
			p.context.isBindingElement = false
			p.scanner.index = p.lookahead.start
			token := p.nextRegexToken()
			return p.finalize(start, jsNodeRegexLiteral(p.getTokenRaw(token), token.pattern, token.flags))
		}
		p.throwUnexpectedToken(p.nextToken())
	case tkKeyword:
		return p.parsePrimaryKeyword()
	}
	p.throwUnexpectedToken(p.nextToken())
	return nil
}

// parsePrimaryKeyword handles a keyword in primary-expression position.
func (p *jsParser) parsePrimaryKeyword() any {
	start := p.lookahead.start
	switch {
	case !p.context.strict && p.context.allowYield && p.matchKeyword("yield"):
		return p.parseIdentifierName()
	case !p.context.strict && p.matchKeyword("let"):
		return p.finalize(start, jsNodeIdentifier(p.nextToken().value.(string)))
	case p.matchKeyword("this"):
		p.context.isAssignmentTarget = false
		p.context.isBindingElement = false
		p.nextToken()
		return p.finalize(start, jsNodeThis())
	case p.matchKeyword("function"):
		p.context.isAssignmentTarget = false
		p.context.isBindingElement = false
		return p.parseFunctionExpression()
	case p.matchKeyword("class"):
		p.context.isAssignmentTarget = false
		p.context.isBindingElement = false
		return p.parseClassExpression()
	}
	p.throwUnexpectedToken(p.nextToken())
	return nil
}

// parseGroupExpression handles a parenthesised expression, returning a
// jsArrowParams placeholder when the parentheses turn out to be arrow-function
// parameters. Destructured/defaulted arrow params rely on
// reinterpretExpressionAsPattern, which currently errors on non-simple targets.
func (p *jsParser) parseGroupExpression() any {
	p.expect("(")
	if p.match(")") { // () =>
		p.nextToken()
		if !p.match("=>") {
			p.expect("=>")
		}
		return jsArrowParams{params: []any{}}
	}
	groupStart := p.lookahead.start
	var params []jsToken
	if p.match("...") { // (...rest) =>
		rest := p.parseRestElement(&params)
		p.expect(")")
		if !p.match("=>") {
			p.expect("=>")
		}
		return jsArrowParams{params: []any{rest}}
	}
	p.context.isBindingElement = true
	expr := p.inheritCoverGrammar(p.parseAssignmentExpression)
	if p.match(",") {
		var arrow bool
		expr, arrow = p.parseGroupCommaTail(groupStart, expr, &params)
		if arrow {
			return expr
		}
	}
	p.expect(")")
	if p.match("=>") {
		expr = p.finishGroupArrow(expr)
	}
	p.context.isBindingElement = false
	return expr
}

// reinterpretAll reinterprets each expression as an assignment/binding pattern,
// replacing entries in place.
func (p *jsParser) reinterpretAll(exprs []any) {
	for i, e := range exprs {
		exprs[i] = p.reinterpretExpressionAsPattern(e)
	}
}

// parseGroupCommaTail parses the remaining comma-separated items after the first
// expression in a parenthesised group, returning either a SequenceExpression or
// (when an arrow follows) a jsArrowParams placeholder.
func (p *jsParser) parseGroupCommaTail(start int, first any, params *[]jsToken) (any, bool) {
	expressions := []any{first}
	p.context.isAssignmentTarget = false
	for p.lookahead.typ != tkEOF {
		if !p.match(",") {
			break
		}
		p.nextToken()
		if p.match(")") {
			p.nextToken()
			p.reinterpretAll(expressions)
			return jsArrowParams{params: expressions}, true
		}
		if p.match("...") {
			if !p.context.isBindingElement {
				p.throwUnexpectedToken(p.lookahead)
			}
			expressions = append(expressions, p.parseRestElement(params))
			p.expect(")")
			if !p.match("=>") {
				p.expect("=>")
			}
			p.context.isBindingElement = false
			p.reinterpretAll(expressions)
			return jsArrowParams{params: expressions}, true
		}
		expressions = append(expressions, p.inheritCoverGrammar(p.parseAssignmentExpression))
	}
	return p.finalize(start, jsNodeSequence(expressions)), false
}

// finishGroupArrow converts a parenthesised expression into arrow parameters when
// a '=>' follows.
func (p *jsParser) finishGroupArrow(expr any) any {
	if NodeType(expr) == "Identifier" && IdentName(expr) == "yield" {
		return jsArrowParams{params: []any{expr}}
	}
	if !p.context.isBindingElement {
		p.throwUnexpectedToken(p.lookahead)
	}
	if NodeType(expr) == "SequenceExpression" {
		seq := FieldSlice(expr, "expressions")
		p.reinterpretAll(seq)
		return jsArrowParams{params: seq}
	}
	expr = p.reinterpretExpressionAsPattern(expr)
	return jsArrowParams{params: []any{expr}}
}

// FieldSlice returns the []any value of the named node field.
func FieldSlice(n any, field string) []any {
	obj, _ := n.(jsonval.Object)
	for _, pr := range obj {
		if pr.K == field {
			s, _ := pr.V.([]any)
			return s
		}
	}
	return nil
}

func (p *jsParser) parseArguments() []any {
	p.expect("(")
	args := []any{}
	if !p.match(")") {
		for {
			var expr any
			if p.match("...") {
				expr = p.parseSpreadElement()
			} else {
				expr = p.isolateCoverGrammar(p.parseAssignmentExpression)
			}
			args = append(args, expr)
			if p.match(")") {
				break
			}
			p.expectCommaSeparator()
			if p.match(")") {
				break
			}
		}
	}
	p.expect(")")
	return args
}

// parseAsyncArgument parses one argument of a possible async arrow, clearing the
// cover-initialized-name error so a `{x = 1}` parameter default is allowed.
func (p *jsParser) parseAsyncArgument() any {
	arg := p.parseAssignmentExpression()
	p.context.firstCoverInitializedNameError = nil
	return arg
}

func (p *jsParser) parseAsyncArguments() []any {
	p.expect("(")
	args := []any{}
	if !p.match(")") {
		for {
			var expr any
			if p.match("...") {
				expr = p.parseSpreadElement()
			} else {
				expr = p.isolateCoverGrammar(p.parseAsyncArgument)
			}
			args = append(args, expr)
			if p.match(")") {
				break
			}
			p.expectCommaSeparator()
			if p.match(")") {
				break
			}
		}
	}
	p.expect(")")
	return args
}

func (p *jsParser) parseIdentifierName() any {
	start := p.lookahead.start
	token := p.nextToken()
	if !jsIsIdentifierName(token) {
		p.throwUnexpectedToken(token)
	}
	return p.finalize(start, jsNodeIdentifier(token.value.(string)))
}

// parseCallStart parses the leading expression of a call/member chain: a
// `super` reference, a `new` expression, or a primary expression.
func (p *jsParser) parseCallStart() any {
	switch {
	case p.matchKeyword("super") && p.context.inFunctionBody:
		start := p.lookahead.start
		p.nextToken()
		expr := p.finalize(start, jsNodeSuper())
		if !p.match("(") && !p.match(".") && !p.match("[") {
			p.throwUnexpectedToken(p.lookahead)
		}
		return expr
	case p.matchKeyword("new"):
		return p.inheritCoverGrammar(p.parseNewExpression)
	default:
		return p.inheritCoverGrammar(p.parsePrimaryExpression)
	}
}

func (p *jsParser) parseLeftHandSideExpressionAllowCall() any {
	startToken := p.lookahead
	start := startToken.start
	maybeAsync := p.matchContextualKeyword("async")
	prevAllowIn := p.context.allowIn
	p.context.allowIn = true
	expr := p.parseCallStart()
	for {
		switch {
		case p.match("."):
			p.context.isBindingElement = false
			p.context.isAssignmentTarget = true
			p.expect(".")
			expr = p.finalize(start, jsNodeStaticMember(expr, p.parseIdentifierName()))
		case p.match("("):
			asyncArrow := maybeAsync && startToken.lineNumber == p.lookahead.lineNumber
			p.context.isBindingElement = false
			p.context.isAssignmentTarget = false
			var args []any
			if asyncArrow {
				args = p.parseAsyncArguments()
			} else {
				args = p.parseArguments()
			}
			expr = p.finalize(start, jsNodeCall(expr, args))
			if asyncArrow && p.match("=>") {
				for i := range args {
					args[i] = p.reinterpretExpressionAsPattern(args[i])
				}
				p.context.allowIn = prevAllowIn
				return jsArrowParams{params: args, async: true}
			}
		case p.match("["):
			p.context.isBindingElement = false
			p.context.isAssignmentTarget = true
			p.expect("[")
			property := p.isolateCoverGrammar(p.parseExpression)
			p.expect("]")
			expr = p.finalize(start, jsNodeComputedMember(expr, property))
		case p.lookahead.typ == tkTemplate && p.lookahead.head:
			quasi := p.parseTemplateLiteral()
			expr = p.finalize(start, jsNodeTaggedTemplate(expr, quasi))
		default:
			p.context.allowIn = prevAllowIn
			return expr
		}
	}
}

func (p *jsParser) parseUpdateExpression() any {
	start := p.lookahead.start
	if p.match("++") || p.match("--") {
		token := p.nextToken()
		expr := p.inheritCoverGrammar(p.parseUnaryExpression)
		if !p.context.isAssignmentTarget {
			p.tolerateError(jsMsgInvalidLHSInAssignment)
		}
		expr = p.finalize(start, jsNodeUpdate(token.value.(string), expr, true))
		p.context.isAssignmentTarget = false
		p.context.isBindingElement = false
		return expr
	}
	expr := p.inheritCoverGrammar(p.parseLeftHandSideExpressionAllowCall)
	if !p.hasLineTerminator && p.lookahead.typ == tkPunctuator && (p.match("++") || p.match("--")) {
		if !p.context.isAssignmentTarget {
			p.tolerateError(jsMsgInvalidLHSInAssignment)
		}
		p.context.isAssignmentTarget = false
		p.context.isBindingElement = false
		operator := p.nextToken().value.(string)
		expr = p.finalize(start, jsNodeUpdate(operator, expr, false))
	}
	return expr
}

func (p *jsParser) parseUnaryExpression() any {
	start := p.lookahead.start
	if p.match("+") || p.match("-") || p.match("~") || p.match("!") ||
		p.matchKeyword("delete") || p.matchKeyword("void") || p.matchKeyword("typeof") {
		token := p.nextToken()
		expr := p.inheritCoverGrammar(p.parseUnaryExpression)
		expr = p.finalize(start, jsNodeUnary(token.value.(string), expr))
		p.context.isAssignmentTarget = false
		p.context.isBindingElement = false
		return expr
	}
	if p.context.await && p.matchContextualKeyword("await") {
		p.nextToken()
		return p.finalize(start, jsNodeAwait(p.parseUnaryExpression()))
	}
	return p.parseUpdateExpression()
}

func (p *jsParser) parseExponentiationExpression() any {
	start := p.lookahead.start
	expr := p.inheritCoverGrammar(p.parseUnaryExpression)
	if NodeType(expr) != "UnaryExpression" && p.match("**") {
		p.nextToken()
		p.context.isAssignmentTarget = false
		p.context.isBindingElement = false
		right := p.isolateCoverGrammar(p.parseExponentiationExpression)
		expr = p.finalize(start, jsNodeBinary("**", expr, right))
	}
	return expr
}

func (p *jsParser) binaryPrecedence(t jsToken) int {
	switch t.typ {
	case tkPunctuator:
		return p.operatorPrecedence[t.value.(string)]
	case tkKeyword:
		op := t.value.(string)
		if op == "instanceof" || (p.context.allowIn && op == "in") {
			return 7
		}
	}
	return 0
}

func (p *jsParser) parseBinaryExpression() any {
	startToken := p.lookahead
	expr := p.inheritCoverGrammar(p.parseExponentiationExpression)
	token := p.lookahead
	prec := p.binaryPrecedence(token)
	if prec <= 0 {
		return expr
	}
	p.nextToken()
	p.context.isAssignmentTarget = false
	p.context.isBindingElement = false
	// markers parallels the operands on stack: each entry is the start offset of
	// that operand, used as the range start of the binary node it becomes part of.
	markers := []int{startToken.start, p.lookahead.start}
	left := expr
	right := p.isolateCoverGrammar(p.parseExponentiationExpression)
	stack := []any{left, token.value.(string), right}
	precedences := []int{prec}
	for {
		prec = p.binaryPrecedence(p.lookahead)
		if prec <= 0 {
			break
		}
		for len(stack) > 2 && prec <= precedences[len(precedences)-1] {
			right = stack[len(stack)-1]
			operator := stack[len(stack)-2].(string)
			left = stack[len(stack)-3]
			stack = stack[:len(stack)-3]
			precedences = precedences[:len(precedences)-1]
			markers = markers[:len(markers)-1]
			stack = append(stack, p.finalize(markers[len(markers)-1], jsNodeBinary(operator, left, right)))
		}
		stack = append(stack, p.nextToken().value.(string))
		precedences = append(precedences, prec)
		markers = append(markers, p.lookahead.start)
		stack = append(stack, p.isolateCoverGrammar(p.parseExponentiationExpression))
	}
	i := len(stack) - 1
	expr = stack[i]
	markers = markers[:len(markers)-1]
	for i > 1 {
		marker := markers[len(markers)-1]
		markers = markers[:len(markers)-1]
		operator := stack[i-1].(string)
		expr = p.finalize(marker, jsNodeBinary(operator, stack[i-2], expr))
		i -= 2
	}
	return expr
}

func (p *jsParser) parseConditionalExpression() any {
	start := p.lookahead.start
	expr := p.inheritCoverGrammar(p.parseBinaryExpression)
	if p.match("?") {
		p.nextToken()
		prevAllowIn := p.context.allowIn
		p.context.allowIn = true
		consequent := p.isolateCoverGrammar(p.parseAssignmentExpression)
		p.context.allowIn = prevAllowIn
		p.expect(":")
		alternate := p.isolateCoverGrammar(p.parseAssignmentExpression)
		expr = p.finalize(start, jsNodeConditional(expr, consequent, alternate))
		p.context.isAssignmentTarget = false
		p.context.isBindingElement = false
	}
	return expr
}

// reinterpretAsCoverFormalsList returns the parameter list for an arrow
// function, or nil if the expression cannot be arrow parameters.
func (p *jsParser) reinterpretAsCoverFormalsList(expr any) []any {
	if ph, ok := expr.(jsArrowParams); ok {
		return ph.params
	}
	if NodeType(expr) == "Identifier" {
		return []any{expr}
	}
	return nil
}

// isStartOfExpression reports whether the lookahead can begin an expression
// (esprima's isStartOfExpression), used to detect an optional yield argument.
func (p *jsParser) isStartOfExpression() bool {
	value, _ := p.lookahead.value.(string)
	switch p.lookahead.typ {
	case tkPunctuator:
		switch value {
		case "[", "(", "{", "+", "-", "!", "~", "++", "--", "/", "/=":
			return true
		}
		return false
	case tkKeyword:
		switch value {
		case "class", "delete", "function", "let", "new", "super", "this", "typeof", "void", "yield":
			return true
		}
		return false
	}
	return true
}

func (p *jsParser) parseYieldExpression() any {
	start := p.lookahead.start
	p.expectKeyword("yield")
	var argument any
	delegate := false
	if !p.hasLineTerminator {
		prevAllowYield := p.context.allowYield
		p.context.allowYield = false
		delegate = p.match("*")
		if delegate {
			p.nextToken()
			argument = p.parseAssignmentExpression()
		} else if p.isStartOfExpression() {
			argument = p.parseAssignmentExpression()
		}
		p.context.allowYield = prevAllowYield
	}
	return p.finalize(start, jsNodeYield(argument, delegate))
}

// maybeAsyncArrowArg rewrites `async x` (a bare `async` followed on the same
// line by an identifier or yield) into a single-parameter async-arrow
// placeholder; otherwise returns expr unchanged.
func (p *jsParser) maybeAsyncArrowArg(startToken jsToken, expr any) any {
	if startToken.typ == tkIdentifier && startToken.lineNumber == p.lookahead.lineNumber && startToken.value == "async" &&
		(p.lookahead.typ == tkIdentifier || p.matchKeyword("yield")) {
		arg := p.reinterpretExpressionAsPattern(p.parsePrimaryExpression())
		return jsArrowParams{params: []any{arg}, async: true}
	}
	return expr
}

// finishArrowFunction parses the `=> body` of an arrow function given its
// parameter list.
func (p *jsParser) finishArrowFunction(start int, list []any, isAsync bool) any {
	if p.hasLineTerminator {
		p.throwUnexpectedToken(p.lookahead)
	}
	p.context.firstCoverInitializedNameError = nil
	prevStrict := p.context.strict
	prevYield := p.context.allowYield
	prevAwait := p.context.await
	p.context.allowYield = true
	p.context.await = isAsync
	p.expect("=>")
	var body any
	if p.match("{") {
		prevAllowIn := p.context.allowIn
		p.context.allowIn = true
		body = p.parseFunctionSourceElements()
		p.context.allowIn = prevAllowIn
	} else {
		body = p.isolateCoverGrammar(p.parseAssignmentExpression)
	}
	expression := NodeType(body) != "BlockStatement"
	arrow := p.finalize(start, jsNodeArrowFunction(list, body, expression, isAsync))
	p.context.strict = prevStrict
	p.context.allowYield = prevYield
	p.context.await = prevAwait
	return arrow
}

func (p *jsParser) parseAssignmentExpression() any {
	if !p.context.allowYield && p.matchKeyword("yield") {
		return p.parseYieldExpression()
	}
	startToken := p.lookahead
	start := startToken.start
	expr := p.maybeAsyncArrowArg(startToken, p.parseConditionalExpression())
	if ph, isArrow := expr.(jsArrowParams); isArrow || p.match("=>") {
		p.context.isAssignmentTarget = false
		p.context.isBindingElement = false
		isAsync := isArrow && ph.async
		list := p.reinterpretAsCoverFormalsList(expr)
		if list != nil {
			expr = p.finishArrowFunction(start, list, isAsync)
		}
		return expr
	}
	if p.matchAssign() {
		if !p.context.isAssignmentTarget {
			p.tolerateError(jsMsgInvalidLHSInAssignment)
		}
		if !p.match("=") {
			p.context.isAssignmentTarget = false
			p.context.isBindingElement = false
		} else {
			expr = p.reinterpretExpressionAsPattern(expr)
		}
		token := p.nextToken()
		operator := token.value.(string)
		right := p.isolateCoverGrammar(p.parseAssignmentExpression)
		expr = p.finalize(start, jsNodeAssignment(operator, expr, right))
		p.context.firstCoverInitializedNameError = nil
	}
	return expr
}

func (p *jsParser) parseExpression() any {
	start := p.lookahead.start
	expr := p.isolateCoverGrammar(p.parseAssignmentExpression)
	if p.match(",") {
		expressions := []any{expr}
		for p.lookahead.typ != tkEOF {
			if !p.match(",") {
				break
			}
			p.nextToken()
			expressions = append(expressions, p.isolateCoverGrammar(p.parseAssignmentExpression))
		}
		expr = p.finalize(start, jsNodeSequence(expressions))
	}
	return expr
}

// --- statements ---

func (p *jsParser) parseExpressionStatement() any {
	start := p.lookahead.start
	expr := p.parseExpression()
	p.consumeSemicolon()
	return p.finalize(start, jsNodeExpressionStatement(expr))
}

func (p *jsParser) parseEmptyStatement() any {
	start := p.lookahead.start
	p.expect(";")
	return p.finalize(start, jsNodeEmptyStatement())
}

func (p *jsParser) parseBlock() any {
	start := p.lookahead.start
	p.expect("{")
	block := []any{}
	for !p.match("}") {
		block = append(block, p.parseStatementListItem())
	}
	p.expect("}")
	return p.finalize(start, jsNodeBlockStatement(block))
}

func (p *jsParser) parseStatement() any {
	switch p.lookahead.typ {
	case tkBooleanLiteral, tkNullLiteral, tkNumericLiteral, tkStringLiteral, tkTemplate, tkRegularExpression:
		return p.parseExpressionStatement()
	case tkPunctuator:
		switch p.lookahead.value {
		case "{":
			return p.parseBlock()
		case "(":
			return p.parseExpressionStatement()
		case ";":
			return p.parseEmptyStatement()
		default:
			return p.parseExpressionStatement()
		}
	case tkIdentifier:
		return p.parseLabelledStatement()
	case tkKeyword:
		return p.parseKeywordStatement()
	}
	p.throwUnexpectedToken(p.lookahead)
	return nil
}

// parseKeywordStatement dispatches a statement that begins with a keyword.
func (p *jsParser) parseKeywordStatement() any {
	switch p.lookahead.value {
	case "break":
		return p.parseBreakStatement()
	case "continue":
		return p.parseContinueStatement()
	case "debugger":
		return p.parseDebuggerStatement()
	case "do":
		return p.parseDoWhileStatement()
	case "for":
		return p.parseForStatement()
	case "function":
		return p.parseFunctionDeclaration(false)
	case "if":
		return p.parseIfStatement()
	case "return":
		return p.parseReturnStatement()
	case "switch":
		return p.parseSwitchStatement()
	case "throw":
		return p.parseThrowStatement()
	case "try":
		return p.parseTryStatement()
	case "var":
		return p.parseVariableStatement()
	case "while":
		return p.parseWhileStatement()
	case "with":
		return p.parseWithStatement()
	default:
		return p.parseExpressionStatement()
	}
}

func (p *jsParser) parseStatementListItem() any {
	p.context.isAssignmentTarget = true
	p.context.isBindingElement = true
	if p.lookahead.typ == tkKeyword {
		switch p.lookahead.value {
		case "export", "import":
			// ES modules are not valid in script mode (esprima.parseScript);
			// esprima reports the bare "Unexpected token" here.
			p.throwUnexpectedTokenMsg(p.lookahead, "Unexpected token")
		case "class":
			return p.parseClassDeclaration(false)
		case "const":
			return p.parseLexicalDeclaration(false)
		case "function":
			return p.parseFunctionDeclaration(false)
		case "let":
			if p.isLexicalDeclaration() {
				return p.parseLexicalDeclaration(false)
			}
			return p.parseStatement()
		}
	}
	if p.matchAsyncFunction() {
		return p.parseFunctionDeclaration(false)
	}
	return p.parseStatement()
}

func (p *jsParser) parseDirective() any {
	token := p.lookahead
	start := token.start
	expr := p.parseExpression()
	directive := ""
	isDirective := false
	if NodeType(expr) == "Literal" {
		raw := p.getTokenRaw(token)
		directive = raw[1 : len(raw)-1]
		isDirective = true
	}
	p.consumeSemicolon()
	if isDirective {
		return p.finalize(start, jsNodeDirective(expr, directive))
	}
	return p.finalize(start, jsNodeExpressionStatement(expr))
}

func (p *jsParser) parseDirectivePrologues() []any {
	body := []any{}
	for p.lookahead.typ == tkStringLiteral {
		stmt := p.parseDirective()
		body = append(body, stmt)
		// A Directive has a "directive" field; a plain string ExpressionStatement
		// ends the prologue.
		if !jsHasDirective(stmt) {
			break
		}
	}
	return body
}

// jsHasDirective reports whether a statement node carries a "directive" field.
func jsHasDirective(n any) bool {
	obj, ok := n.(jsonval.Object)
	if !ok {
		return false
	}
	for _, pr := range obj {
		if pr.K == "directive" {
			return true
		}
	}
	return false
}

func (p *jsParser) parseScript() jsonval.Object {
	start := p.lookahead.start
	body := p.parseDirectivePrologues()
	for p.lookahead.typ != tkEOF {
		body = append(body, p.parseStatementListItem())
	}
	return p.finalize(start, jsNodeScript(body)).(jsonval.Object)
}

// IdentName returns the "name" field of an Identifier node.
func IdentName(n any) string {
	obj, _ := n.(jsonval.Object)
	for _, pr := range obj {
		if pr.K == "name" {
			s, _ := pr.V.(string)
			return s
		}
	}
	return ""
}

// jsFieldIsNil reports whether the named field of a node exists and is nil.
func jsFieldIsNil(n any, field string) bool {
	obj, _ := n.(jsonval.Object)
	for _, pr := range obj {
		if pr.K == field {
			return pr.V == nil
		}
	}
	return false
}

// Field returns the value of the named node field, or nil.
func Field(n any, field string) any {
	obj, _ := n.(jsonval.Object)
	for _, pr := range obj {
		if pr.K == field {
			return pr.V
		}
	}
	return nil
}

// reinterpretExpressionAsPattern converts an expression into the equivalent
// assignment/binding pattern (esprima's reinterpretExpressionAsPattern), turning
// array/object literals into patterns and spread into rest. The result is
// returned (nodes are immutable here rather than mutated in place).
func (p *jsParser) reinterpretExpressionAsPattern(expr any) any {
	switch NodeType(expr) {
	case "SpreadElement":
		arg := p.reinterpretExpressionAsPattern(Field(expr, "argument"))
		return p.carryRange(expr, jsonval.Object{{K: "type", V: "RestElement"}, {K: "argument", V: arg}})
	case "ArrayExpression":
		elements := FieldSlice(expr, "elements")
		out := make([]any, len(elements))
		for i, e := range elements {
			if e != nil {
				out[i] = p.reinterpretExpressionAsPattern(e)
			}
		}
		return p.carryRange(expr, jsonval.Object{{K: "type", V: "ArrayPattern"}, {K: "elements", V: out}})
	case "ObjectExpression":
		props := FieldSlice(expr, "properties")
		out := make([]any, len(props))
		for i, pr := range props {
			out[i] = p.reinterpretPropertyValue(pr)
		}
		return p.carryRange(expr, jsonval.Object{{K: "type", V: "ObjectPattern"}, {K: "properties", V: out}})
	case "AssignmentExpression":
		left := p.reinterpretExpressionAsPattern(Field(expr, "left"))
		return p.carryRange(expr, jsonval.Object{{K: "type", V: "AssignmentPattern"}, {K: "left", V: left}, {K: "right", V: Field(expr, "right")}})
	}
	// Identifier, MemberExpression, RestElement, AssignmentPattern and anything
	// else are left as-is (esprima tolerates other node types).
	return expr
}

// reinterpretPropertyValue rebuilds an object Property node with its value field
// reinterpreted as a pattern, preserving field order.
func (p *jsParser) reinterpretPropertyValue(prop any) any {
	obj, _ := prop.(jsonval.Object)
	out := make(jsonval.Object, len(obj))
	for i, pr := range obj {
		if pr.K == "value" {
			out[i] = jsonval.Pair{K: "value", V: p.reinterpretExpressionAsPattern(pr.V)}
		} else {
			out[i] = pr
		}
	}
	return out
}

// --- template literals ---

func (p *jsParser) parseTemplateElement() any {
	start := p.lookahead.start
	if p.lookahead.typ != tkTemplate {
		p.throwUnexpectedToken(p.lookahead)
	}
	token := p.nextToken()
	return p.finalize(start, jsNodeTemplateElement(token.value.(string), token.cooked, token.tail))
}

func (p *jsParser) parseTemplateLiteral() any {
	start := p.lookahead.start
	expressions := []any{}
	quasi := p.parseTemplateElement()
	quasis := []any{quasi}
	for !FieldBool(quasi, "tail") {
		expressions = append(expressions, p.parseExpression())
		quasi = p.parseTemplateElement()
		quasis = append(quasis, quasi)
	}
	return p.finalize(start, jsNodeTemplateLiteral(quasis, expressions))
}

// FieldBool returns the bool value of the named node field.
func FieldBool(n any, field string) bool {
	obj, _ := n.(jsonval.Object)
	for _, pr := range obj {
		if pr.K == field {
			b, _ := pr.V.(bool)
			return b
		}
	}
	return false
}

// --- array / object initializers, new, spread ---

func (p *jsParser) parseSpreadElement() any {
	start := p.lookahead.start
	p.expect("...")
	return p.finalize(start, jsNodeSpreadElement(p.inheritCoverGrammar(p.parseAssignmentExpression)))
}

func (p *jsParser) parseArrayInitializer() any {
	start := p.lookahead.start
	elements := []any{}
	p.expect("[")
	for !p.match("]") {
		switch {
		case p.match(","):
			p.nextToken()
			elements = append(elements, nil)
		case p.match("..."):
			element := p.parseSpreadElement()
			if !p.match("]") {
				p.context.isAssignmentTarget = false
				p.context.isBindingElement = false
				p.expect(",")
			}
			elements = append(elements, element)
		default:
			elements = append(elements, p.inheritCoverGrammar(p.parseAssignmentExpression))
			if !p.match("]") {
				p.expect(",")
			}
		}
	}
	p.expect("]")
	return p.finalize(start, jsNodeArray(elements))
}

func (p *jsParser) parseObjectPropertyKey() any {
	start := p.lookahead.start
	token := p.nextToken()
	switch token.typ {
	case tkStringLiteral, tkNumericLiteral:
		return p.finalize(start, jsNodeLiteral(token.value, p.getTokenRaw(token)))
	case tkIdentifier, tkBooleanLiteral, tkNullLiteral, tkKeyword:
		return p.finalize(start, jsNodeIdentifier(token.value.(string)))
	case tkPunctuator:
		if token.value == "[" {
			key := p.isolateCoverGrammar(p.parseAssignmentExpression)
			p.expect("]")
			return key
		}
	}
	p.throwUnexpectedToken(token)
	return nil
}

func (p *jsParser) qualifiedPropertyName(t jsToken) bool {
	switch t.typ {
	case tkIdentifier, tkStringLiteral, tkBooleanLiteral, tkNullLiteral, tkNumericLiteral, tkKeyword:
		return true
	case tkPunctuator:
		return t.value == "["
	}
	return false
}

func (p *jsParser) parsePropertyMethodFunction() any {
	start := p.lookahead.start
	prevYield := p.context.allowYield
	p.context.allowYield = true
	params := p.parseFormalParameters()
	body := p.parseFunctionSourceElements()
	p.context.allowYield = prevYield
	return p.finalize(start, jsNodeFunction("FunctionExpression", nil, params, body, false, false))
}

func (p *jsParser) parseGetterMethod() any {
	start := p.lookahead.start
	prevYield := p.context.allowYield
	p.context.allowYield = true
	params := p.parseFormalParameters()
	if len(params) > 0 {
		p.tolerateError("Getter must not have any formal parameters")
	}
	body := p.parseFunctionSourceElements()
	p.context.allowYield = prevYield
	return p.finalize(start, jsNodeFunction("FunctionExpression", nil, params, body, false, false))
}

func (p *jsParser) parseSetterMethod() any {
	start := p.lookahead.start
	prevYield := p.context.allowYield
	p.context.allowYield = true
	params := p.parseFormalParameters()
	if len(params) != 1 {
		p.tolerateError("Setter must have exactly one formal parameter")
	} else if NodeType(params[0]) == "RestElement" {
		p.tolerateError("Setter function argument must not be a rest parameter")
	}
	body := p.parseFunctionSourceElements()
	p.context.allowYield = prevYield
	return p.finalize(start, jsNodeFunction("FunctionExpression", nil, params, body, false, false))
}

func (p *jsParser) parsePropertyMethodAsyncFunction() any {
	start := p.lookahead.start
	prevYield, prevAwait := p.context.allowYield, p.context.await
	p.context.allowYield = false
	p.context.await = true
	params := p.parseFormalParameters()
	body := p.parseFunctionSourceElements()
	p.context.allowYield, p.context.await = prevYield, prevAwait
	return p.finalize(start, jsNodeFunction("FunctionExpression", nil, params, body, false, true))
}

func (p *jsParser) parseGeneratorMethod() any {
	start := p.lookahead.start
	prevYield := p.context.allowYield
	p.context.allowYield = true
	params := p.parseFormalParameters()
	p.context.allowYield = false
	body := p.parseFunctionSourceElements()
	p.context.allowYield = prevYield
	return p.finalize(start, jsNodeFunction("FunctionExpression", nil, params, body, true, false))
}

// parseObjectPropertyHead parses the leading key of an object property,
// reporting whether the key is computed and whether it is an async method.
func (p *jsParser) parseObjectPropertyHead(start int, token jsToken) (key any, computed, isAsync bool) {
	switch {
	case token.typ == tkIdentifier:
		id := token.value.(string)
		p.nextToken()
		computed = p.match("[")
		isAsync = !p.hasLineTerminator && id == "async" && !p.match(":") && !p.match("(") && !p.match("*") && !p.match(",")
		if isAsync {
			key = p.parseObjectPropertyKey()
		} else {
			key = p.finalize(start, jsNodeIdentifier(id))
		}
	case p.match("*"):
		p.nextToken()
	default:
		computed = p.match("[")
		key = p.parseObjectPropertyKey()
	}
	return
}

func (p *jsParser) parseObjectProperty() any {
	start := p.lookahead.start
	token := p.lookahead
	var value any
	method, shorthand := false, false

	key, computed, isAsync := p.parseObjectPropertyHead(start, token)

	lookaheadKey := p.qualifiedPropertyName(p.lookahead)
	kind := "init"
	switch {
	case !isAsync && token.typ == tkIdentifier && token.value == "get" && lookaheadKey:
		kind = "get"
		computed = p.match("[")
		key = p.parseObjectPropertyKey()
		value = p.parseGetterMethod()
	case !isAsync && token.typ == tkIdentifier && token.value == "set" && lookaheadKey:
		kind = "set"
		computed = p.match("[")
		key = p.parseObjectPropertyKey()
		value = p.parseSetterMethod()
	case token.typ == tkPunctuator && token.value == "*" && lookaheadKey:
		computed = p.match("[")
		key = p.parseObjectPropertyKey()
		value = p.parseGeneratorMethod()
		method = true
	default:
		value, method, shorthand = p.parseObjectPropertyValue(start, token, key, isAsync)
	}
	return p.finalize(start, jsNodeProperty(kind, key, value, computed, method, shorthand))
}

// parseObjectPropertyValue parses a non-accessor property's value (`key: v`,
// method `key()`, or shorthand `key`/`key = default`).
func (p *jsParser) parseObjectPropertyValue(start int, token jsToken, key any, isAsync bool) (value any, method, shorthand bool) {
	if key == nil {
		p.throwUnexpectedToken(p.lookahead)
	}
	switch {
	case p.match(":"):
		p.nextToken()
		value = p.inheritCoverGrammar(p.parseAssignmentExpression)
	case p.match("("):
		if isAsync {
			value = p.parsePropertyMethodAsyncFunction()
		} else {
			value = p.parsePropertyMethodFunction()
		}
		method = true
	case token.typ == tkIdentifier:
		id := p.finalize(start, jsNodeIdentifier(token.value.(string)))
		if p.match("=") {
			tok := p.lookahead
			p.context.firstCoverInitializedNameError = &tok
			p.nextToken()
			shorthand = true
			value = p.finalize(start, jsNodeAssignmentPattern(id, p.isolateCoverGrammar(p.parseAssignmentExpression)))
		} else {
			shorthand = true
			value = id
		}
	default:
		p.throwUnexpectedToken(p.nextToken())
	}
	return value, method, shorthand
}

func (p *jsParser) parseObjectInitializer() any {
	start := p.lookahead.start
	p.expect("{")
	properties := []any{}
	for !p.match("}") {
		properties = append(properties, p.parseObjectProperty())
		if !p.match("}") {
			p.expectCommaSeparator()
		}
	}
	p.expect("}")
	return p.finalize(start, jsNodeObject(properties))
}

func (p *jsParser) parseLeftHandSideExpression() any {
	start := p.lookahead.start
	var expr any
	switch {
	case p.matchKeyword("super") && p.context.inFunctionBody:
		expr = p.parseSuper()
	case p.matchKeyword("new"):
		expr = p.inheritCoverGrammar(p.parseNewExpression)
	default:
		expr = p.inheritCoverGrammar(p.parsePrimaryExpression)
	}
	for {
		switch {
		case p.match("["):
			p.context.isBindingElement = false
			p.context.isAssignmentTarget = true
			p.expect("[")
			property := p.isolateCoverGrammar(p.parseExpression)
			p.expect("]")
			expr = p.finalize(start, jsNodeComputedMember(expr, property))
		case p.match("."):
			p.context.isBindingElement = false
			p.context.isAssignmentTarget = true
			p.expect(".")
			expr = p.finalize(start, jsNodeStaticMember(expr, p.parseIdentifierName()))
		case p.lookahead.typ == tkTemplate && p.lookahead.head:
			quasi := p.parseTemplateLiteral()
			expr = p.finalize(start, jsNodeTaggedTemplate(expr, quasi))
		default:
			return expr
		}
	}
}

func (p *jsParser) parseNewExpression() any {
	start := p.lookahead.start
	id := p.parseIdentifierName() // 'new'
	if p.match(".") {
		p.nextToken()
		if p.lookahead.typ == tkIdentifier && p.context.inFunctionBody && p.lookahead.value == "target" {
			property := p.parseIdentifierName()
			return p.finalize(start, jsNodeMetaProperty(id, property))
		}
		p.throwUnexpectedToken(p.lookahead)
	}
	callee := p.isolateCoverGrammar(p.parseLeftHandSideExpression)
	args := []any{}
	if p.match("(") {
		args = p.parseArguments()
	}
	p.context.isAssignmentTarget = false
	p.context.isBindingElement = false
	return p.finalize(start, jsNodeNew(callee, args))
}

// --- function definitions ---

func (p *jsParser) parseFunctionSourceElements() any {
	start := p.lookahead.start
	p.expect("{")
	body := p.parseDirectivePrologues()
	prevLabel, prevIter, prevSwitch, prevFunc := p.context.labelSet, p.context.inIteration, p.context.inSwitch, p.context.inFunctionBody
	p.context.labelSet = map[string]bool{}
	p.context.inIteration = false
	p.context.inSwitch = false
	p.context.inFunctionBody = true
	for p.lookahead.typ != tkEOF && !p.match("}") {
		body = append(body, p.parseStatementListItem())
	}
	p.expect("}")
	p.context.labelSet, p.context.inIteration, p.context.inSwitch, p.context.inFunctionBody = prevLabel, prevIter, prevSwitch, prevFunc
	return p.finalize(start, jsNodeBlockStatement(body))
}

func (p *jsParser) parseRestElement(params *[]jsToken) any {
	start := p.lookahead.start
	p.expect("...")
	arg := p.parsePattern(params, "")
	if p.match("=") {
		p.throwError("Unexpected token =")
	}
	if !p.match(")") {
		p.throwError("Rest parameter must be last formal parameter")
	}
	return p.finalize(start, jsNodeRestElement(arg))
}

func (p *jsParser) parseFormalParameter(params *[]any) {
	var patParams []jsToken
	var param any
	if p.match("...") {
		param = p.parseRestElement(&patParams)
	} else {
		param = p.parsePatternWithDefault(&patParams, "")
	}
	*params = append(*params, param)
}

func (p *jsParser) parseFormalParameters() []any {
	params := []any{}
	p.expect("(")
	if !p.match(")") {
		for p.lookahead.typ != tkEOF {
			p.parseFormalParameter(&params)
			if p.match(")") {
				break
			}
			p.expect(",")
			if p.match(")") {
				break
			}
		}
	}
	p.expect(")")
	return params
}

func (p *jsParser) parseFunctionDeclaration(identifierIsOptional bool) any {
	start := p.lookahead.start
	isAsync := p.matchContextualKeyword("async")
	if isAsync {
		p.nextToken()
	}
	p.expectKeyword("function")
	isGenerator := !isAsync && p.match("*")
	if isGenerator {
		p.nextToken()
	}
	var id any
	if !identifierIsOptional || !p.match("(") {
		id = p.parseVariableIdentifier("")
	}
	prevYield, prevAwait := p.context.allowYield, p.context.await
	p.context.allowYield = !isGenerator
	p.context.await = isAsync
	params := p.parseFormalParameters()
	body := p.parseFunctionSourceElements()
	p.context.allowYield, p.context.await = prevYield, prevAwait
	return p.finalize(start, jsNodeFunction("FunctionDeclaration", id, params, body, isGenerator, isAsync))
}

func (p *jsParser) parseFunctionExpression() any {
	start := p.lookahead.start
	isAsync := p.matchContextualKeyword("async")
	if isAsync {
		p.nextToken()
	}
	p.expectKeyword("function")
	isGenerator := !isAsync && p.match("*")
	if isGenerator {
		p.nextToken()
	}
	prevYield, prevAwait := p.context.allowYield, p.context.await
	p.context.allowYield = !isGenerator
	p.context.await = isAsync
	var id any
	if !p.match("(") {
		id = p.parseVariableIdentifier("")
	}
	params := p.parseFormalParameters()
	body := p.parseFunctionSourceElements()
	p.context.allowYield, p.context.await = prevYield, prevAwait
	return p.finalize(start, jsNodeFunction("FunctionExpression", id, params, body, isGenerator, isAsync))
}

// --- classes ---

func (p *jsParser) throwUnexpectedTokenMsg(t jsToken, msg string) {
	panic(&jsSyntaxError{line: t.lineNumber, column: t.start - t.lineStart + 1, index: t.start, description: msg})
}

// jsIsPropertyKey reports whether key is the (non-computed) property named name.
func jsIsPropertyKey(key any, name string) bool {
	switch NodeType(key) {
	case "Identifier":
		return IdentName(key) == name
	case "Literal":
		v, _ := Field(key, "value").(string)
		return v == name
	}
	return false
}

func (p *jsParser) parseSuper() any {
	start := p.lookahead.start
	p.expectKeyword("super")
	if !p.match("[") && !p.match(".") {
		p.throwUnexpectedToken(p.lookahead)
	}
	return p.finalize(start, jsNodeSuper())
}

func (p *jsParser) parseClassDeclaration(identifierIsOptional bool) any {
	start := p.lookahead.start
	prevStrict := p.context.strict
	p.context.strict = true
	p.expectKeyword("class")
	var id any
	if !identifierIsOptional || p.lookahead.typ == tkIdentifier {
		id = p.parseVariableIdentifier("")
	}
	superClass := p.parseClassHeritage()
	body := p.parseClassBody()
	p.context.strict = prevStrict
	return p.finalize(start, jsNodeClass("ClassDeclaration", id, superClass, body))
}

func (p *jsParser) parseClassExpression() any {
	start := p.lookahead.start
	prevStrict := p.context.strict
	p.context.strict = true
	p.expectKeyword("class")
	var id any
	if p.lookahead.typ == tkIdentifier {
		id = p.parseVariableIdentifier("")
	}
	superClass := p.parseClassHeritage()
	body := p.parseClassBody()
	p.context.strict = prevStrict
	return p.finalize(start, jsNodeClass("ClassExpression", id, superClass, body))
}

func (p *jsParser) parseClassHeritage() any {
	if !p.matchKeyword("extends") {
		return nil
	}
	p.nextToken()
	return p.isolateCoverGrammar(p.parseLeftHandSideExpressionAllowCall)
}

func (p *jsParser) parseClassBody() any {
	start := p.lookahead.start
	body := []any{}
	hasConstructor := false
	p.expect("{")
	for !p.match("}") {
		if p.match(";") {
			p.nextToken()
		} else {
			body = append(body, p.parseClassElement(&hasConstructor))
		}
	}
	p.expect("}")
	return p.finalize(start, jsNodeClassBody(body))
}

// parseClassElementHead parses the `static`, `async` and `*` prefixes plus the
// leading key of a class element.
func (p *jsParser) parseClassElementHead() (token jsToken, key any, computed, isStatic, isAsync bool) {
	token = p.lookahead
	if p.match("*") {
		p.nextToken()
		return
	}
	computed = p.match("[")
	key = p.parseObjectPropertyKey()
	if IdentName(key) == "static" && (p.qualifiedPropertyName(p.lookahead) || p.match("*")) {
		token = p.lookahead
		isStatic = true
		computed = p.match("[")
		if p.match("*") {
			p.nextToken()
		} else {
			key = p.parseObjectPropertyKey()
		}
	}
	if token.typ == tkIdentifier && !p.hasLineTerminator && token.value == "async" {
		punct, _ := p.lookahead.value.(string)
		if !p.match(":") && !p.match("(") && !p.match("}") && punct != "" {
			isAsync = true
			token = p.lookahead
			key = p.parseObjectPropertyKey()
			if token.typ == tkIdentifier && token.value == "constructor" {
				p.throwUnexpectedTokenMsg(token, "Class constructor may not be an async method")
			}
		}
	}
	return
}

// parseClassElementValue determines a class element's kind and function value
// (accessor, generator, or plain method).
func (p *jsParser) parseClassElementValue(token jsToken, key *any, computed *bool, isAsync bool) (kind string, value any, method bool) {
	lookaheadKey := p.qualifiedPropertyName(p.lookahead)
	switch {
	case token.typ == tkIdentifier && token.value == "get" && lookaheadKey:
		kind = "get"
		*computed = p.match("[")
		*key = p.parseObjectPropertyKey()
		value = p.parseGetterMethod()
	case token.typ == tkIdentifier && token.value == "set" && lookaheadKey:
		kind = "set"
		*computed = p.match("[")
		*key = p.parseObjectPropertyKey()
		value = p.parseSetterMethod()
	case token.typ == tkPunctuator && token.value == "*" && lookaheadKey:
		kind = "method"
		*computed = p.match("[")
		*key = p.parseObjectPropertyKey()
		value = p.parseGeneratorMethod()
		method = true
	}
	if kind == "" && *key != nil && p.match("(") {
		kind = "method"
		if isAsync {
			value = p.parsePropertyMethodAsyncFunction()
		} else {
			value = p.parsePropertyMethodFunction()
		}
		method = true
	}
	return kind, value, method
}

func (p *jsParser) parseClassElement(hasConstructor *bool) any {
	start := p.lookahead.start
	token, key, computed, isStatic, isAsync := p.parseClassElementHead()
	kind, value, method := p.parseClassElementValue(token, &key, &computed, isAsync)
	if kind == "" {
		p.throwUnexpectedToken(p.lookahead)
	}
	if !computed {
		kind = p.checkClassSpecialKey(token, key, kind, method, value, isStatic, hasConstructor)
	}
	return p.finalize(start, jsNodeMethodDefinition(key, computed, value, kind, isStatic))
}

// checkClassSpecialKey validates the `prototype`/`constructor` restrictions and
// promotes a constructor method's kind.
func (p *jsParser) checkClassSpecialKey(token jsToken, key any, kind string, method bool, value any, isStatic bool, hasConstructor *bool) string {
	if isStatic && jsIsPropertyKey(key, "prototype") {
		p.throwUnexpectedTokenMsg(token, "Classes may not have static property named prototype")
	}
	if !isStatic && jsIsPropertyKey(key, "constructor") {
		if kind != "method" || !method || FieldBool(value, "generator") {
			p.throwUnexpectedTokenMsg(token, "Class constructor may not be an accessor")
		}
		if *hasConstructor {
			p.throwUnexpectedTokenMsg(token, "A class may only have one constructor")
		}
		*hasConstructor = true
		kind = "constructor"
	}
	return kind
}

// --- patterns ---

func (p *jsParser) parseVariableIdentifier(kind string) any {
	start := p.lookahead.start
	token := p.nextToken()
	switch {
	case token.typ == tkKeyword && token.value == "yield":
		if !p.context.allowYield {
			p.throwUnexpectedToken(token)
		}
	case token.typ != tkIdentifier:
		if token.value != "let" || kind != "var" {
			p.throwUnexpectedToken(token)
		}
	}
	return p.finalize(start, jsNodeIdentifier(token.value.(string)))
}

func (p *jsParser) parseBindingRestElement(params *[]jsToken, kind string) any {
	start := p.lookahead.start
	p.expect("...")
	return p.finalize(start, jsNodeRestElement(p.parsePattern(params, kind)))
}

func (p *jsParser) parseArrayPattern(params *[]jsToken, kind string) any {
	start := p.lookahead.start
	p.expect("[")
	elements := []any{}
	for !p.match("]") {
		if p.match(",") {
			p.nextToken()
			elements = append(elements, nil)
		} else {
			if p.match("...") {
				elements = append(elements, p.parseBindingRestElement(params, kind))
				break
			}
			elements = append(elements, p.parsePatternWithDefault(params, kind))
			if !p.match("]") {
				p.expect(",")
			}
		}
	}
	p.expect("]")
	return p.finalize(start, jsNodeArrayPattern(elements))
}

func (p *jsParser) parsePropertyPattern(params *[]jsToken, kind string) any {
	start := p.lookahead.start
	computed, shorthand := false, false
	var key, value any
	if p.lookahead.typ == tkIdentifier {
		keyToken := p.lookahead
		key = p.parseVariableIdentifier("")
		init := p.finalize(start, jsNodeIdentifier(keyToken.value.(string)))
		switch {
		case p.match("="):
			*params = append(*params, keyToken)
			shorthand = true
			p.nextToken()
			value = p.finalize(start, jsNodeAssignmentPattern(init, p.parseAssignmentExpression()))
		case !p.match(":"):
			*params = append(*params, keyToken)
			shorthand = true
			value = init
		default:
			p.expect(":")
			value = p.parsePatternWithDefault(params, kind)
		}
	} else {
		computed = p.match("[")
		key = p.parseObjectPropertyKey()
		p.expect(":")
		value = p.parsePatternWithDefault(params, kind)
	}
	return p.finalize(start, jsNodeProperty("init", key, value, computed, false, shorthand))
}

func (p *jsParser) parseObjectPattern(params *[]jsToken, kind string) any {
	start := p.lookahead.start
	properties := []any{}
	p.expect("{")
	for !p.match("}") {
		properties = append(properties, p.parsePropertyPattern(params, kind))
		if !p.match("}") {
			p.expect(",")
		}
	}
	p.expect("}")
	return p.finalize(start, jsNodeObjectPattern(properties))
}

func (p *jsParser) parsePattern(params *[]jsToken, kind string) any {
	switch {
	case p.match("["):
		return p.parseArrayPattern(params, kind)
	case p.match("{"):
		return p.parseObjectPattern(params, kind)
	}
	*params = append(*params, p.lookahead)
	return p.parseVariableIdentifier(kind)
}

func (p *jsParser) parsePatternWithDefault(params *[]jsToken, kind string) any {
	start := p.lookahead.start
	pattern := p.parsePattern(params, kind)
	if p.match("=") {
		p.nextToken()
		prevYield := p.context.allowYield
		p.context.allowYield = true
		right := p.isolateCoverGrammar(p.parseAssignmentExpression)
		p.context.allowYield = prevYield
		pattern = p.finalize(start, jsNodeAssignmentPattern(pattern, right))
	}
	return pattern
}

// --- variable / lexical declarations ---

func (p *jsParser) parseVariableDeclaration(inFor bool) any {
	start := p.lookahead.start
	var params []jsToken
	id := p.parsePattern(&params, "var")
	var init any
	if p.match("=") {
		p.nextToken()
		init = p.isolateCoverGrammar(p.parseAssignmentExpression)
	} else if NodeType(id) != "Identifier" && !inFor {
		p.expect("=")
	}
	return p.finalize(start, jsNodeVariableDeclarator(id, init))
}

func (p *jsParser) parseVariableDeclarationList(inFor bool) []any {
	list := []any{p.parseVariableDeclaration(inFor)}
	for p.match(",") {
		p.nextToken()
		list = append(list, p.parseVariableDeclaration(inFor))
	}
	return list
}

func (p *jsParser) parseVariableStatement() any {
	start := p.lookahead.start
	p.expectKeyword("var")
	declarations := p.parseVariableDeclarationList(false)
	p.consumeSemicolon()
	return p.finalize(start, jsNodeVariableDeclaration(declarations, "var"))
}

func (p *jsParser) parseLexicalBinding(kind string, inFor bool) any {
	start := p.lookahead.start
	var params []jsToken
	id := p.parsePattern(&params, kind)
	var init any
	if kind == "const" {
		if !p.matchKeyword("in") && !p.matchContextualKeyword("of") {
			if p.match("=") {
				p.nextToken()
				init = p.isolateCoverGrammar(p.parseAssignmentExpression)
			} else {
				p.throwError("Missing initializer in const declaration")
			}
		}
	} else if (!inFor && NodeType(id) != "Identifier") || p.match("=") {
		p.expect("=")
		init = p.isolateCoverGrammar(p.parseAssignmentExpression)
	}
	return p.finalize(start, jsNodeVariableDeclarator(id, init))
}

func (p *jsParser) parseBindingList(kind string, inFor bool) []any {
	list := []any{p.parseLexicalBinding(kind, inFor)}
	for p.match(",") {
		p.nextToken()
		list = append(list, p.parseLexicalBinding(kind, inFor))
	}
	return list
}

func (p *jsParser) isLexicalDeclaration() bool {
	state := p.scanner.saveState()
	p.scanner.scanComments()
	next := p.scanner.lex()
	p.scanner.restoreState(state)
	switch next.typ {
	case tkIdentifier:
		return true
	case tkPunctuator:
		return next.value == "[" || next.value == "{"
	case tkKeyword:
		return next.value == "let" || next.value == "yield"
	}
	return false
}

func (p *jsParser) parseLexicalDeclaration(inFor bool) any {
	start := p.lookahead.start
	kind := p.nextToken().value.(string)
	declarations := p.parseBindingList(kind, inFor)
	p.consumeSemicolon()
	return p.finalize(start, jsNodeVariableDeclaration(declarations, kind))
}

// --- control-flow statements ---

func (p *jsParser) parseIfClause() any { return p.parseStatement() }

func (p *jsParser) parseIfStatement() any {
	start := p.lookahead.start
	p.expectKeyword("if")
	p.expect("(")
	test := p.parseExpression()
	p.expect(")")
	consequent := p.parseIfClause()
	var alternate any
	if p.matchKeyword("else") {
		p.nextToken()
		alternate = p.parseIfClause()
	}
	return p.finalize(start, jsNodeIf(test, consequent, alternate))
}

func (p *jsParser) parseWhileStatement() any {
	start := p.lookahead.start
	p.expectKeyword("while")
	p.expect("(")
	test := p.parseExpression()
	p.expect(")")
	prevIter := p.context.inIteration
	p.context.inIteration = true
	body := p.parseStatement()
	p.context.inIteration = prevIter
	return p.finalize(start, jsNodeWhile(test, body))
}

func (p *jsParser) parseDoWhileStatement() any {
	start := p.lookahead.start
	p.expectKeyword("do")
	prevIter := p.context.inIteration
	p.context.inIteration = true
	body := p.parseStatement()
	p.context.inIteration = prevIter
	p.expectKeyword("while")
	p.expect("(")
	test := p.parseExpression()
	p.expect(")")
	if p.match(";") {
		p.nextToken()
	}
	return p.finalize(start, jsNodeDoWhile(body, test))
}

// jsForHead holds the parsed head of a for-statement: either a C-style init or a
// for-in/for-of left/right pair (leftSet distinguishes them).
type jsForHead struct {
	init, left, right any
	forIn, leftSet    bool
}

// parseForVarDeclHead handles the `var`/`let`/`const` init of a for-statement,
// detecting for-in / for-of.
func (p *jsParser) parseForVarDeclHead(kind string) jsForHead {
	declStart := p.lookahead.start
	p.nextToken()
	prevAllowIn := p.context.allowIn
	p.context.allowIn = false
	var declarations []any
	if kind == "var" {
		declarations = p.parseVariableDeclarationList(true)
	} else {
		declarations = p.parseBindingList(kind, true)
	}
	p.context.allowIn = prevAllowIn
	single := len(declarations) == 1
	noInit := single && jsFieldIsNil(declarations[0], "init")
	switch {
	case single && p.matchKeyword("in") && (kind == "var" || noInit):
		decl := p.finalize(declStart, jsNodeVariableDeclaration(declarations, kind))
		p.nextToken()
		return jsForHead{left: decl, right: p.parseExpression(), forIn: true, leftSet: true}
	case noInit && p.matchContextualKeyword("of"):
		decl := p.finalize(declStart, jsNodeVariableDeclaration(declarations, kind))
		p.nextToken()
		return jsForHead{left: decl, right: p.parseAssignmentExpression(), leftSet: true}
	default:
		// esprima finalizes a `var` declaration before consuming ';' but a
		// let/const one after consumeSemicolon, so the ranges differ.
		if kind == "var" {
			decl := p.finalize(declStart, jsNodeVariableDeclaration(declarations, kind))
			p.expect(";")
			return jsForHead{init: decl}
		}
		p.consumeSemicolon()
		return jsForHead{init: p.finalize(declStart, jsNodeVariableDeclaration(declarations, kind))}
	}
}

// parseForExprHead handles the expression init of a for-statement, detecting
// for-in / for-of.
func (p *jsParser) parseForExprHead() jsForHead {
	start := p.lookahead.start
	prevAllowIn := p.context.allowIn
	p.context.allowIn = false
	init := p.inheritCoverGrammar(p.parseAssignmentExpression)
	p.context.allowIn = prevAllowIn
	switch {
	case p.matchKeyword("in"):
		if !p.context.isAssignmentTarget || NodeType(init) == "AssignmentExpression" {
			p.tolerateError("Invalid left-hand side in for-in")
		}
		p.nextToken()
		init = p.reinterpretExpressionAsPattern(init)
		return jsForHead{left: init, right: p.parseExpression(), forIn: true, leftSet: true}
	case p.matchContextualKeyword("of"):
		if !p.context.isAssignmentTarget || NodeType(init) == "AssignmentExpression" {
			p.tolerateError("Invalid left-hand side in for-loop")
		}
		p.nextToken()
		init = p.reinterpretExpressionAsPattern(init)
		return jsForHead{left: init, right: p.parseAssignmentExpression(), leftSet: true}
	default:
		if p.match(",") {
			initSeq := []any{init}
			for p.match(",") {
				p.nextToken()
				initSeq = append(initSeq, p.isolateCoverGrammar(p.parseAssignmentExpression))
			}
			init = p.finalize(start, jsNodeSequence(initSeq))
		}
		p.expect(";")
		return jsForHead{init: init}
	}
}

func (p *jsParser) parseForStatement() any {
	start := p.lookahead.start
	p.expectKeyword("for")
	p.expect("(")
	var h jsForHead
	switch {
	case p.match(";"):
		p.nextToken()
	case p.matchKeyword("var"):
		h = p.parseForVarDeclHead("var")
	case p.matchKeyword("const") || p.matchKeyword("let"):
		h = p.parseForVarDeclHead(p.lookahead.value.(string))
	default:
		h = p.parseForExprHead()
	}
	var test, update any
	if !h.leftSet {
		if !p.match(";") {
			test = p.parseExpression()
		}
		p.expect(";")
		if !p.match(")") {
			update = p.parseExpression()
		}
	}
	p.expect(")")
	prevIter := p.context.inIteration
	p.context.inIteration = true
	body := p.isolateCoverGrammar(p.parseStatement)
	p.context.inIteration = prevIter
	switch {
	case !h.leftSet:
		return p.finalize(start, jsNodeFor(h.init, test, update, body))
	case h.forIn:
		return p.finalize(start, jsNodeForIn(h.left, h.right, body))
	default:
		return p.finalize(start, jsNodeForOf(h.left, h.right, body))
	}
}

func (p *jsParser) parseContinueStatement() any {
	start := p.lookahead.start
	p.expectKeyword("continue")
	var label any
	if p.lookahead.typ == tkIdentifier && !p.hasLineTerminator {
		id := p.parseVariableIdentifier("")
		label = id
		if !p.context.labelSet["$"+IdentName(id)] {
			p.throwError("Undefined label '" + IdentName(id) + "'")
		}
	}
	p.consumeSemicolon()
	if label == nil && !p.context.inIteration {
		p.throwError("Illegal continue statement")
	}
	return p.finalize(start, jsNodeContinue(label))
}

func (p *jsParser) parseBreakStatement() any {
	start := p.lookahead.start
	p.expectKeyword("break")
	var label any
	if p.lookahead.typ == tkIdentifier && !p.hasLineTerminator {
		id := p.parseVariableIdentifier("")
		if !p.context.labelSet["$"+IdentName(id)] {
			p.throwError("Undefined label '" + IdentName(id) + "'")
		}
		label = id
	}
	p.consumeSemicolon()
	if label == nil && !p.context.inIteration && !p.context.inSwitch {
		p.throwError("Illegal break statement")
	}
	return p.finalize(start, jsNodeBreak(label))
}

func (p *jsParser) parseReturnStatement() any {
	start := p.lookahead.start
	if !p.context.inFunctionBody {
		p.tolerateError("Illegal return statement")
	}
	p.expectKeyword("return")
	hasArgument := (!p.match(";") && !p.match("}") && !p.hasLineTerminator && p.lookahead.typ != tkEOF) ||
		p.lookahead.typ == tkStringLiteral
	var argument any
	if hasArgument {
		argument = p.parseExpression()
	}
	p.consumeSemicolon()
	return p.finalize(start, jsNodeReturn(argument))
}

func (p *jsParser) parseWithStatement() any {
	start := p.lookahead.start
	p.expectKeyword("with")
	p.expect("(")
	object := p.parseExpression()
	p.expect(")")
	body := p.parseStatement()
	return p.finalize(start, jsNodeWith(object, body))
}

func (p *jsParser) parseSwitchCase() any {
	start := p.lookahead.start
	var test any
	if p.matchKeyword("default") {
		p.nextToken()
	} else {
		p.expectKeyword("case")
		test = p.parseExpression()
	}
	p.expect(":")
	consequent := []any{}
	for !p.match("}") && !p.matchKeyword("default") && !p.matchKeyword("case") {
		consequent = append(consequent, p.parseStatementListItem())
	}
	return p.finalize(start, jsNodeSwitchCase(test, consequent))
}

func (p *jsParser) parseSwitchStatement() any {
	start := p.lookahead.start
	p.expectKeyword("switch")
	p.expect("(")
	discriminant := p.parseExpression()
	p.expect(")")
	prevSwitch := p.context.inSwitch
	p.context.inSwitch = true
	cases := []any{}
	defaultFound := false
	p.expect("{")
	for !p.match("}") {
		clause := p.parseSwitchCase()
		if jsFieldIsNil(clause, "test") {
			if defaultFound {
				p.throwError("More than one default clause in switch statement")
			}
			defaultFound = true
		}
		cases = append(cases, clause)
	}
	p.expect("}")
	p.context.inSwitch = prevSwitch
	return p.finalize(start, jsNodeSwitch(discriminant, cases))
}

func (p *jsParser) parseLabelledStatement() any {
	start := p.lookahead.start
	expr := p.parseExpression()
	if NodeType(expr) == "Identifier" && p.match(":") {
		p.nextToken()
		name := IdentName(expr)
		key := "$" + name
		if p.context.labelSet[key] {
			p.throwError("Label '" + name + "' has already been declared")
		}
		p.context.labelSet[key] = true
		var body any
		if p.matchKeyword("function") {
			body = p.parseFunctionDeclaration(false)
		} else {
			body = p.parseStatement()
		}
		delete(p.context.labelSet, key)
		return p.finalize(start, jsNodeLabeled(expr, body))
	}
	p.consumeSemicolon()
	return p.finalize(start, jsNodeExpressionStatement(expr))
}

func (p *jsParser) parseThrowStatement() any {
	start := p.lookahead.start
	p.expectKeyword("throw")
	if p.hasLineTerminator {
		p.throwError("Illegal newline after throw")
	}
	argument := p.parseExpression()
	p.consumeSemicolon()
	return p.finalize(start, jsNodeThrow(argument))
}

func (p *jsParser) parseCatchClause() any {
	start := p.lookahead.start
	p.expectKeyword("catch")
	p.expect("(")
	if p.match(")") {
		p.throwUnexpectedToken(p.lookahead)
	}
	var params []jsToken
	param := p.parsePattern(&params, "")
	p.expect(")")
	body := p.parseBlock()
	return p.finalize(start, jsNodeCatch(param, body))
}

func (p *jsParser) parseFinallyClause() any {
	p.expectKeyword("finally")
	return p.parseBlock()
}

func (p *jsParser) parseTryStatement() any {
	start := p.lookahead.start
	p.expectKeyword("try")
	block := p.parseBlock()
	var handler, finalizer any
	if p.matchKeyword("catch") {
		handler = p.parseCatchClause()
	}
	if p.matchKeyword("finally") {
		finalizer = p.parseFinallyClause()
	}
	if handler == nil && finalizer == nil {
		p.throwError("Missing catch or finally after try")
	}
	return p.finalize(start, jsNodeTry(block, handler, finalizer))
}

func (p *jsParser) parseDebuggerStatement() any {
	start := p.lookahead.start
	p.expectKeyword("debugger")
	p.consumeSemicolon()
	return p.finalize(start, jsNodeDebugger())
}

// Parse parses the source and returns the ESTree AST as a jsonval.Object, or a
// *jsSyntaxError (recovered from the panic-based control flow).
func Parse(code string) (ast jsonval.Object, err error) {
	defer func() {
		if r := recover(); r != nil {
			if se, ok := r.(*jsSyntaxError); ok {
				err = se
				return
			}
			panic(r)
		}
	}()
	p := newJSParser(code)
	return p.parseScript(), nil
}

// ParseFull parses the source with esprima's range/tokens/comment options
// enabled: every AST node carries a range, and flat comments/tokens arrays are
// returned in source order. On a syntax error it returns the error like Parse.
func ParseFull(code string) (ast jsonval.Object, comments []jsonval.Object, tokens []jsonval.Object, err error) {
	defer func() {
		if r := recover(); r != nil {
			if se, ok := r.(*jsSyntaxError); ok {
				err = se
				return
			}
			panic(r)
		}
	}()
	p := newJSParserBare(code)
	p.track = true
	p.scanner.trackComment = true
	p.nextToken()
	ast = p.parseScript()
	comments = p.scanner.comments
	sort.Slice(comments, func(i, j int) bool {
		return jsRangeStart(comments[i]) < jsRangeStart(comments[j])
	})
	return ast, comments, p.tokens, nil
}

// jsRangeStart returns the start offset of a node/comment's range field.
func jsRangeStart(n any) int {
	if r, ok := Field(n, "range").([]any); ok && len(r) == 2 {
		if f, ok := r[0].(float64); ok {
			return int(f)
		}
	}
	return -1
}
