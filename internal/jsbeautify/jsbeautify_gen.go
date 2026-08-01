// Package jsbeautify prints a JavaScript syntax tree back out as formatted
// source, reproducing escodegen, which CyberChef uses.
//
// It takes the tree from
// [github.com/roberson-io/cchef/internal/jsparse]. Comments do not appear in
// that tree, so [AttachComments] puts them back against the nodes they belong
// to before [Generate] runs; [GenerateComments] does both together.
package jsbeautify

// Code generator for the JavaScript Beautify operation — a transliteration of
// escodegen's CodeGenerator (escodegen/escodegen.js) restricted to CyberChef's
// options (sourceMap/json/compact/renumber/preserveBlankLines all off,
// parentheses on). It walks the ESTree AST produced by jsparse.Parse and emits
// formatted source. Arrays-of-fragments in escodegen become plain string
// concatenation here since source maps are not supported.

import (
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/roberson-io/cchef/internal/jsonval"
	"github.com/roberson-io/cchef/internal/jsparse"
)

// Operator/expression precedence (escodegen's Precedence).
const (
	pSequence       = 0
	pYield          = 1
	pAssignment     = 1
	pConditional    = 2
	pArrowFunction  = 2
	pCoalesce       = 3
	pLogicalOR      = 4
	pLogicalAND     = 5
	pBitwiseOR      = 6
	pBitwiseXOR     = 7
	pBitwiseAND     = 8
	pEquality       = 9
	pRelational     = 10
	pBitwiseSHIFT   = 11
	pAdditive       = 12
	pMultiplicative = 13
	pExponentiation = 14
	pAwait          = 15
	pUnary          = 15
	pPostfix        = 16
	pCall           = 18
	pNew            = 19
	pTaggedTemplate = 20
	pMember         = 21
	pPrimary        = 22
)

// Generation flags (escodegen's F_*).
const (
	fAllowIn          = 1 << iota // F_ALLOW_IN
	fAllowCall                    // F_ALLOW_CALL
	fAllowUnparathNew             // F_ALLOW_UNPARATH_NEW
	fFuncBody                     // F_FUNC_BODY
	fDirectiveCtx                 // F_DIRECTIVE_CTX
	fSemicolonOpt                 // F_SEMICOLON_OPT
)

// Expression and statement flag sets (escodegen's E_* / S_*).
const (
	eFTT = fAllowCall | fAllowUnparathNew
	eTTF = fAllowIn | fAllowCall
	eTTT = fAllowIn | fAllowCall | fAllowUnparathNew
	eTFF = fAllowIn
	eFFT = fAllowUnparathNew
	eTFT = fAllowIn | fAllowUnparathNew

	sTFFF = fAllowIn
	sTFFT = fAllowIn | fSemicolonOpt
	sFFFF = 0
	sTFTF = fAllowIn | fDirectiveCtx
	sTTFF = fAllowIn | fFuncBody
)

var jsBinaryPrecedence = map[string]int{
	"??": pCoalesce, "||": pLogicalOR, "&&": pLogicalAND,
	"|": pBitwiseOR, "^": pBitwiseXOR, "&": pBitwiseAND,
	"==": pEquality, "!=": pEquality, "===": pEquality, "!==": pEquality,
	"<": pRelational, ">": pRelational, "<=": pRelational, ">=": pRelational,
	"in": pRelational, "instanceof": pRelational,
	"<<": pBitwiseSHIFT, ">>": pBitwiseSHIFT, ">>>": pBitwiseSHIFT,
	"+": pAdditive, "-": pAdditive,
	"*": pMultiplicative, "%": pMultiplicative, "/": pMultiplicative,
	"**": pExponentiation,
}

// jsGen holds the generator's formatting options and current indentation.
type jsGen struct {
	indent     string // one indentation unit
	base       string // current cumulative indentation
	quotes     string // "single" | "double" | "auto"
	newline    string // "\n"
	space      string // " "
	semicolons bool
	comments   *jsCommentSet // non-nil enables comment emission
	stmtH      map[string]func(any, int) string
	exprH      map[string]func(any, int, int) string
}

// Generate renders an ESTree AST to formatted source using CyberChef's options.
func Generate(ast jsonval.Object, indent, quotes string, semicolons bool) string {
	return GenerateComments(ast, indent, quotes, semicolons, nil)
}

// GenerateComments is Generate with comment emission driven by the attached
// comment set (nil to disable).
func GenerateComments(ast jsonval.Object, indent, quotes string, semicolons bool, cs *jsCommentSet) string {
	g := &jsGen{indent: indent, quotes: quotes, newline: "\n", space: " ", semicolons: semicolons, comments: cs}
	g.buildHandlers()
	return g.generateStatement(ast, sTFFF)
}

// --- helpers ---

func jsIsLineTerm(cp int) bool {
	return cp == 0x0A || cp == 0x0D || cp == 0x2028 || cp == 0x2029
}

func isIdentPartES5(c rune) bool {
	return c == '$' || c == '_' ||
		(c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
		(c >= 0x80 && jsparse.IsIdentifierPart(int(c)))
}

func lastRune(s string) rune {
	rs := []rune(s)
	if len(rs) == 0 {
		return 0
	}
	return rs[len(rs)-1]
}

func firstRune(s string) rune {
	for _, r := range s {
		return r
	}
	return 0
}

func endsWithLineTerminator(s string) bool {
	r := lastRune(s)
	return r != 0 && jsIsLineTerm(int(r))
}

func parenthesize(s string, current, precedence int) string {
	if current < precedence {
		return "(" + s + ")"
	}
	return s
}

func (g *jsGen) addIndent(s string) string { return g.base + s }

// withIndent runs fn with the base indentation increased by one unit.
func (g *jsGen) withIndent(fn func()) {
	save := g.base
	g.base += g.indent
	fn()
	g.base = save
}

func (g *jsGen) semicolon(flags int) string {
	if !g.semicolons && flags&fSemicolonOpt != 0 {
		return ""
	}
	return ";"
}

func (g *jsGen) join(left, right string) string {
	if len(left) == 0 {
		return right
	}
	if len(right) == 0 {
		return left
	}
	l := lastRune(left)
	r := firstRune(right)
	if ((l == '+' || l == '-') && l == r) ||
		(isIdentPartES5(l) && isIdentPartES5(r)) ||
		(l == '/' && r == 'i') {
		return left + " " + right
	}
	if isJSWhiteSpace(int(l)) || jsIsLineTerm(int(l)) || isJSWhiteSpace(int(r)) || jsIsLineTerm(int(r)) {
		return left + right
	}
	return left + g.space + right
}

func isJSWhiteSpace(cp int) bool { return jsparse.IsWhiteSpace(cp) }

// --- string / number literal generation ---

func jsEscapeDisallowed(code int) string {
	switch code {
	case 0x5C:
		return "\\\\"
	case 0x0A:
		return "\\n"
	case 0x0D:
		return "\\r"
	case 0x2028:
		return "\\u2028"
	case 0x2029:
		return "\\u2029"
	}
	return ""
}

func jsEscapeAllowed(code, next int) string {
	switch code {
	case 0x08:
		return "\\b"
	case 0x0C:
		return "\\f"
	case 0x09:
		return "\\t"
	}
	hex := strings.ToUpper(strconv.FormatInt(int64(code), 16))
	switch {
	case code > 0xFF:
		return "\\u" + "0000"[len(hex):] + hex
	case code == 0 && (next < '0' || next > '9'):
		return "\\0"
	case code == 0x0B:
		return "\\x0B"
	default:
		return "\\x" + "00"[len(hex):] + hex
	}
}

// escapeStringBody performs escodegen's first pass over a string: escaping
// disallowed and non-printable characters and counting bare quote occurrences.
func escapeStringBody(units []uint16) (body []uint16, singleQuotes, doubleQuotes int) {
	for i := range units {
		unit := units[i]
		code := int(unit)
		switch {
		case code == 0x27:
			singleQuotes++
		case code == 0x22:
			doubleQuotes++
		case jsIsLineTerm(code) || code == 0x5C:
			body = append(body, utf16.Encode([]rune(jsEscapeDisallowed(code)))...)
			continue
		case !isIdentPartES5(rune(unit)) && (code < 0x20 || code > 0x7E):
			next := 0
			if i+1 < len(units) {
				next = int(units[i+1])
			}
			body = append(body, utf16.Encode([]rune(jsEscapeAllowed(code, next)))...)
			continue
		}
		body = append(body, unit)
	}
	return body, singleQuotes, doubleQuotes
}

func (g *jsGen) escapeString(str string) string {
	body, singleQuotes, doubleQuotes := escapeStringBody(utf16.Encode([]rune(str)))
	single := g.quotes != "double" && (g.quotes != "auto" || doubleQuotes >= singleQuotes)
	quote := uint16('\'')
	if !single {
		quote = '"'
	}
	need := singleQuotes
	if !single {
		need = doubleQuotes
	}
	if need == 0 {
		return string(utf16.Decode(append(append([]uint16{quote}, body...), quote)))
	}
	out := []uint16{quote}
	for _, code := range body {
		if (code == 0x27 && single) || (code == 0x22 && !single) {
			out = append(out, 0x5C)
		}
		out = append(out, code)
	}
	out = append(out, quote)
	return string(utf16.Decode(out))
}

// --- shared node helpers ---

func (g *jsGen) generateIdentifier(node any) string { return jsparse.IdentName(node) }

func (g *jsGen) generateAsyncPrefix(node any, spaceRequired bool) string {
	if jsparse.FieldBool(node, "async") {
		if spaceRequired {
			return "async "
		}
		return "async" + g.space
	}
	return ""
}

func (g *jsGen) generateStarSuffix(node any) string {
	if jsparse.FieldBool(node, "generator") {
		return "*" + g.space
	}
	return ""
}

func (g *jsGen) generateMethodPrefix(prop any) string {
	fn := jsparse.Field(prop, "value")
	prefix := ""
	if jsparse.FieldBool(fn, "async") {
		prefix += g.generateAsyncPrefix(fn, !jsparse.FieldBool(prop, "computed"))
	}
	if jsparse.FieldBool(fn, "generator") && g.generateStarSuffix(fn) != "" {
		prefix += "*"
	}
	return prefix
}

func (g *jsGen) generatePattern(node any, precedence, flags int) string {
	if jsparse.NodeType(node) == "Identifier" {
		return g.generateIdentifier(node)
	}
	return g.generateExpression(node, precedence, flags)
}

func (g *jsGen) generateFunctionParams(node any) string {
	params := jsparse.FieldSlice(node, "params")
	typ := jsparse.NodeType(node)
	if typ == "ArrowFunctionExpression" && len(params) == 1 && jsparse.NodeType(params[0]) == "Identifier" {
		return g.generateAsyncPrefix(node, true) + g.generateIdentifier(params[0])
	}
	var b strings.Builder
	if typ == "ArrowFunctionExpression" {
		b.WriteString(g.generateAsyncPrefix(node, false))
	}
	b.WriteString("(")
	for i, p := range params {
		b.WriteString(g.generatePattern(p, pAssignment, eTTT))
		if i+1 < len(params) {
			b.WriteString("," + g.space)
		}
	}
	b.WriteString(")")
	return b.String()
}

func (g *jsGen) generateFunctionBody(node any) string {
	result := g.generateFunctionParams(node)
	if jsparse.NodeType(node) == "ArrowFunctionExpression" {
		result += g.space + "=>"
	}
	if jsparse.FieldBool(node, "expression") {
		result += g.space
		expr := g.generateExpression(jsparse.Field(node, "body"), pAssignment, eTTT)
		if len(expr) > 0 && expr[0] == '{' {
			expr = "(" + expr + ")"
		}
		result += expr
	} else {
		result += g.maybeBlock(jsparse.Field(node, "body"), sTTFF)
	}
	return result
}

func (g *jsGen) generatePropertyKey(key any, computed bool) string {
	if computed {
		return "[" + g.generateExpression(key, pAssignment, eTTT) + "]"
	}
	return g.generateExpression(key, pAssignment, eTTT)
}

func (g *jsGen) generateAssignment(left, right any, operator string, precedence, flags int) string {
	if pAssignment < precedence {
		flags |= fAllowIn
	}
	s := g.generateExpression(left, pCall, flags) +
		g.space + operator + g.space +
		g.generateExpression(right, pAssignment, flags)
	return parenthesize(s, pAssignment, precedence)
}

func (g *jsGen) maybeBlock(stmt any, flags int) string {
	noLeadingComment := len(g.jsNodeLeading(stmt)) == 0
	switch jsparse.NodeType(stmt) {
	case "BlockStatement":
		if noLeadingComment {
			return g.space + g.generateStatement(stmt, flags)
		}
	case "EmptyStatement":
		if noLeadingComment {
			return ";"
		}
	}
	var result string
	g.withIndent(func() {
		result = g.newline + g.addIndent(g.generateStatement(stmt, flags))
	})
	return result
}

func (g *jsGen) maybeBlockSuffix(stmt any, result string) string {
	ends := endsWithLineTerminator(result)
	if jsparse.NodeType(stmt) == "BlockStatement" && len(g.jsNodeLeading(stmt)) == 0 && !ends {
		return result + g.space
	}
	if ends {
		return result + g.base
	}
	return result + g.newline + g.base
}

func (g *jsGen) generateIterationForStatement(operator string, stmt any, flags int) string {
	result := "for" + g.space + "("
	g.withIndent(func() {
		left := jsparse.Field(stmt, "left")
		if jsparse.NodeType(left) == "VariableDeclaration" {
			g.withIndent(func() {
				kind, _ := jsparse.Field(left, "kind").(string)
				decls := jsparse.FieldSlice(left, "declarations")
				result += kind + " " + g.generateStatement(decls[0], sFFFF)
			})
		} else {
			result += g.generateExpression(left, pCall, eTTT)
		}
		result = g.join(result, operator)
		result = g.join(result, g.generateExpression(jsparse.Field(stmt, "right"), pAssignment, eTTT)) + ")"
	})
	result += g.maybeBlock(jsparse.Field(stmt, "body"), flags)
	return result
}
