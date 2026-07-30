package yara

import (
	"slices"
	"strings"
)

// Parse reads a set of rules.
//
// A fault in the shape of the rules is reported as "syntax error", which is
// what libyara says, with the line it was found on.
func Parse(src string) (*RuleSet, error) {
	toks, err := lexAll(src)
	if err != nil {
		return nil, err
	}
	p := &parser{toks: toks}
	return p.ruleSet()
}

// lexAll runs the lexer to the end and collects what it found.
func lexAll(src string) ([]token, error) {
	lx := newLexer(src)
	var out []token
	for {
		tok, err := lx.next()
		if err != nil {
			return nil, err
		}
		out = append(out, tok)
		if tok.kind == tokenEOF {
			return out, nil
		}
	}
}

// parser walks the tokens of a rule set.
type parser struct {
	toks []token
	pos  int
}

// cur is the token being looked at.
func (p *parser) cur() token { return p.toks[p.pos] }

// at reports whether the current token is of a kind, and one of the given texts
// when any are named.
func (p *parser) at(kind tokenKind, texts ...string) bool {
	tok := p.cur()
	if tok.kind != kind {
		return false
	}
	if len(texts) == 0 {
		return true
	}
	return slices.Contains(texts, tok.text)
}

// accept takes the current token if it matches, and says whether it did.
func (p *parser) accept(kind tokenKind, texts ...string) bool {
	if p.at(kind, texts...) {
		p.pos++
		return true
	}
	return false
}

// expect takes the current token, or fails.
func (p *parser) expect(kind tokenKind, texts ...string) (token, error) {
	if !p.at(kind, texts...) {
		return token{}, p.syntaxError()
	}
	tok := p.cur()
	p.pos++
	return tok, nil
}

// syntaxError is what any fault in the shape of the rules gets, matching the
// single message libyara reports for all of them.
func (p *parser) syntaxError() error {
	// libyara reports a fault at the very end of the rules against line zero,
	// having run out of input before it could count one.
	line := p.cur().line
	if p.cur().kind == tokenEOF {
		line = 0
	}
	return &compileError{line: line, msg: "syntax error"}
}

// ruleSet reads the imports and the rules.
func (p *parser) ruleSet() (*RuleSet, error) {
	set := &RuleSet{}
	for p.accept(tokenKeyword, "import") {
		name, err := p.expect(tokenText)
		if err != nil {
			return nil, err
		}
		set.Imports = append(set.Imports, name.text)
	}
	for !p.at(tokenEOF) {
		rule, err := p.rule()
		if err != nil {
			return nil, err
		}
		set.Rules = append(set.Rules, rule)
	}
	return set, nil
}

// rule reads one rule, from its modifiers to its closing brace.
func (p *parser) rule() (*Rule, error) {
	rule := &Rule{Line: p.cur().line}
	for {
		if p.accept(tokenKeyword, "global") {
			rule.Global = true
			continue
		}
		if p.accept(tokenKeyword, "private") {
			rule.Private = true
			continue
		}
		break
	}
	if _, err := p.expect(tokenKeyword, "rule"); err != nil {
		return nil, err
	}
	name, err := p.expect(tokenIdentifier)
	if err != nil {
		return nil, err
	}
	rule.Name = name.text

	if p.accept(tokenPunct, ":") {
		for p.at(tokenIdentifier) {
			rule.Tags = append(rule.Tags, p.cur().text)
			p.pos++
		}
		if len(rule.Tags) == 0 {
			return nil, p.syntaxError()
		}
	}
	if _, err := p.expect(tokenPunct, "{"); err != nil {
		return nil, err
	}
	if err := p.ruleBody(rule); err != nil {
		return nil, err
	}
	closing, err := p.expect(tokenPunct, "}")
	if err != nil {
		return nil, err
	}
	rule.EndLine = closing.line
	return rule, nil
}

// ruleBody reads the three sections a rule may hold. Only the condition is
// required.
func (p *parser) ruleBody(rule *Rule) error {
	if p.accept(tokenKeyword, "meta") {
		if _, err := p.expect(tokenPunct, ":"); err != nil {
			return err
		}
		if err := p.metaSection(rule); err != nil {
			return err
		}
	}
	if p.accept(tokenKeyword, "strings") {
		if _, err := p.expect(tokenPunct, ":"); err != nil {
			return err
		}
		if err := p.stringsSection(rule); err != nil {
			return err
		}
	}
	if _, err := p.expect(tokenKeyword, "condition"); err != nil {
		return err
	}
	if _, err := p.expect(tokenPunct, ":"); err != nil {
		return err
	}
	cond, err := p.expression(0)
	if err != nil {
		return err
	}
	rule.Condition = cond
	return nil
}

// metaSection reads the entries before the strings or the condition.
func (p *parser) metaSection(rule *Rule) error {
	for p.at(tokenIdentifier) {
		key := p.cur().text
		p.pos++
		if _, err := p.expect(tokenPunct, "="); err != nil {
			return err
		}
		value, err := p.metaValue()
		if err != nil {
			return err
		}
		rule.Meta = append(rule.Meta, Meta{Key: key, Value: value})
	}
	if len(rule.Meta) == 0 {
		return p.syntaxError()
	}
	return nil
}

// metaValue reads what one metadata entry holds: text, a number, or a truth.
func (p *parser) metaValue() (any, error) {
	tok := p.cur()
	switch {
	case tok.kind == tokenText:
		p.pos++
		return tok.text, nil
	case tok.kind == tokenInteger:
		p.pos++
		return tok.num, nil
	case tok.kind == tokenPunct && tok.text == "-":
		p.pos++
		n, err := p.expect(tokenInteger)
		if err != nil {
			return nil, err
		}
		return -n.num, nil
	case tok.kind == tokenKeyword && (tok.text == "true" || tok.text == "false"):
		p.pos++
		return tok.text == "true", nil
	}
	return nil, p.syntaxError()
}

// stringsSection reads the strings a rule looks for.
func (p *parser) stringsSection(rule *Rule) error {
	for p.at(tokenStringIdentifier) {
		str, err := p.stringDeclaration()
		if err != nil {
			return err
		}
		rule.Strings = append(rule.Strings, str)
	}
	if len(rule.Strings) == 0 {
		return p.syntaxError()
	}
	return nil
}

// stringDeclaration reads one string and the modifiers after it.
func (p *parser) stringDeclaration() (*String, error) {
	id := p.cur()
	p.pos++
	if _, err := p.expect(tokenPunct, "="); err != nil {
		return nil, err
	}

	str := &String{ID: id.text, Line: id.line}
	switch tok := p.cur(); tok.kind {
	case tokenText:
		str.Kind, str.Text = stringText, tok.text
	case tokenHexString:
		str.Kind, str.Text = stringHex, tok.text
		// Checked here rather than later, so that a hex pattern that makes no
		// sense is reported against the line it was written on, as libyara
		// reports it, instead of wherever the rule later falls apart.
		built, err := hexPattern(str)
		if err != nil {
			return nil, err
		}
		str.pattern = built
	case tokenRegex:
		str.Kind, str.Text, str.Flags = stringRegex, tok.text, tok.flags
	default:
		return nil, p.syntaxError()
	}
	p.pos++

	if err := p.stringModifiers(str); err != nil {
		return nil, err
	}
	return str, nil
}

// The widest a byte can be shifted by, which is what a bare xor covers.
const xorMaxKey = 255

// modifiersAllowed says which words may follow each kind of string, taken from
// libyara's grammar. A hex pattern takes only one; a regular expression takes
// everything but the two that rewrite the bytes being looked for.
var modifiersAllowed = map[stringKind]map[string]bool{
	stringText: {
		"nocase": true, "wide": true, "ascii": true, "fullword": true,
		"private": true, "xor": true, "base64": true, "base64wide": true,
	},
	stringRegex: {
		"nocase": true, "wide": true, "ascii": true, "fullword": true, "private": true,
	},
	stringHex: {"private": true},
}

// modifierWords is every word that may follow a string, whichever kind it is.
var modifierWords = map[string]bool{
	"nocase": true, "wide": true, "ascii": true, "fullword": true,
	"private": true, "xor": true, "base64": true, "base64wide": true,
}

// stringModifiers reads the words that follow a string.
func (p *parser) stringModifiers(str *String) error {
	for p.at(tokenKeyword) {
		if !modifierWords[p.cur().text] {
			// Something else entirely, such as the next string or the
			// condition, so the modifiers are done.
			return nil
		}
		if !modifiersAllowed[str.Kind][p.cur().text] {
			return p.syntaxError()
		}
		switch p.cur().text {
		case "nocase":
			str.Mods.Nocase = true
		case "wide":
			str.Mods.Wide = true
		case "ascii":
			str.Mods.ASCII = true
		case "fullword":
			str.Mods.Fullword = true
		case "private":
			str.Mods.Private = true
		case "xor":
			p.pos++
			return p.xorModifier(str)
		case "base64", "base64wide":
			wide := p.cur().text == "base64wide"
			p.pos++
			return p.base64Modifier(str, wide)
		}
		p.pos++
	}
	return nil
}

// xorModifier reads an xor and the range of keys it may use.
func (p *parser) xorModifier(str *String) error {
	str.Mods.XOR = true
	str.Mods.XORMin, str.Mods.XORMax = 0, xorMaxKey
	if p.accept(tokenPunct, "(") {
		low, err := p.expect(tokenInteger)
		if err != nil {
			return err
		}
		str.Mods.XORMin, str.Mods.XORMax = int(low.num), int(low.num)
		if p.accept(tokenPunct, "-") {
			high, err := p.expect(tokenInteger)
			if err != nil {
				return err
			}
			str.Mods.XORMax = int(high.num)
		}
		if _, err := p.expect(tokenPunct, ")"); err != nil {
			return err
		}
	}
	return p.stringModifiers(str)
}

// base64Modifier reads a base64 modifier and the alphabet it was given.
func (p *parser) base64Modifier(str *String, wide bool) error {
	if wide {
		str.Mods.Base64Wide = true
	} else {
		str.Mods.Base64 = true
	}
	if p.accept(tokenPunct, "(") {
		alphabet, err := p.expect(tokenText)
		if err != nil {
			return err
		}
		str.Mods.Base64Alphabet = alphabet.text
		if _, err := p.expect(tokenPunct, ")"); err != nil {
			return err
		}
	}
	return p.stringModifiers(str)
}

// binaryOps gives each operator its binding power, loosest first. The order is
// libyara's own, from the precedence declarations in its grammar.
var binaryOps = map[string]int{
	"or": 1, "and": 2,
	"==": 4, "!=": 4, "contains": 4, "icontains": 4, "startswith": 4,
	"endswith": 4, "istartswith": 4, "iendswith": 4, "iequals": 4, "matches": 4,
	"<": 5, "<=": 5, ">": 5, ">=": 5,
	"|": 6, "^": 7, "&": 8,
	"<<": 9, ">>": 9,
	"+": 10, "-": 10,
	"*": 11, `\`: 11, "%": 11,
}

// notPrecedence is where "not" and "defined" bind: tighter than "and", looser
// than any comparison.
const notPrecedence = 3

// primaryPrecedence is where a number begins: tighter than any comparison, but
// loose enough to take in every way of working one out. It is what follows
// "at", which libyara reads as a whole sum rather than a single number.
const primaryPrecedence = 6

// expression reads a condition, taking operators only while they bind at least
// as tightly as minPrec.
func (p *parser) expression(minPrec int) (Expr, error) {
	left, err := p.prefix(minPrec)
	if err != nil {
		return nil, err
	}
	for {
		tok := p.cur()
		if tok.kind != tokenPunct && tok.kind != tokenKeyword {
			return left, nil
		}
		prec, ok := binaryOps[tok.text]
		if !ok || prec < minPrec {
			return left, nil
		}
		p.pos++
		right, err := p.expression(prec + 1)
		if err != nil {
			return nil, err
		}
		left = Binary{Op: tok.text, L: left, R: right}
	}
}

// prefix reads whatever comes before a binary operator: the words and signs
// that take one operand, and then a primary.
func (p *parser) prefix(minPrec int) (Expr, error) {
	tok := p.cur()
	switch {
	case tok.kind == tokenKeyword && tok.text == "not" && minPrec <= notPrecedence:
		p.pos++
		x, err := p.expression(notPrecedence)
		if err != nil {
			return nil, err
		}
		return Not{X: x}, nil
	case tok.kind == tokenKeyword && tok.text == "defined" && minPrec <= notPrecedence:
		p.pos++
		x, err := p.expression(notPrecedence)
		if err != nil {
			return nil, err
		}
		return Defined{X: x}, nil
	case tok.kind == tokenPunct && (tok.text == "-" || tok.text == "~"):
		p.pos++
		x, err := p.prefix(unaryPrecedence)
		if err != nil {
			return nil, err
		}
		return Unary{Op: tok.text, X: x}, nil
	}
	return p.primary()
}

// unaryPrecedence is where a sign or a complement binds, tighter than anything
// written between two operands.
const unaryPrecedence = 12

// intFuncs are the ways a condition reads a number out of the data.
var intFuncs = map[string]bool{
	"uint8": true, "uint16": true, "uint32": true,
	"int8": true, "int16": true, "int32": true,
	"uint8be": true, "uint16be": true, "uint32be": true,
	"int8be": true, "int16be": true, "int32be": true,
}

// primary reads the smallest complete piece of a condition.
func (p *parser) primary() (Expr, error) {
	tok := p.cur()
	switch tok.kind {
	case tokenPunct:
		if tok.text == "(" {
			p.pos++
			inner, err := p.expression(0)
			if err != nil {
				return nil, err
			}
			if _, err := p.expect(tokenPunct, ")"); err != nil {
				return nil, err
			}
			return inner, nil
		}
	case tokenInteger:
		return p.integerOrCount()
	case tokenDouble:
		p.pos++
		return DoubleLit(tok.dbl), nil
	case tokenText:
		p.pos++
		return StringLit(tok.text), nil
	case tokenRegex:
		p.pos++
		return RegexLit{Body: tok.text, Flags: tok.flags}, nil
	case tokenStringIdentifier:
		return p.stringExpr()
	case tokenStringCount:
		p.pos++
		return StringCount{ID: "$" + strings.TrimPrefix(tok.text, "#")}, nil
	case tokenStringOffset, tokenStringLength:
		return p.stringIndexed()
	case tokenKeyword:
		return p.keywordExpr()
	case tokenIdentifier:
		return p.identifierExpr()
	}
	return nil, p.syntaxError()
}

// integerOrCount reads a number, which may turn out to be how many of a set
// have to hold rather than a value in its own right.
func (p *parser) integerOrCount() (Expr, error) {
	tok := p.cur()
	if p.toks[p.pos+1].kind == tokenKeyword && p.toks[p.pos+1].text == "of" {
		p.pos++
		return p.ofExpr(Quantifier{Kind: "count", Count: IntLit(tok.num)})
	}
	p.pos++
	return IntLit(tok.num), nil
}

// stringExpr reads a reference to a string, and whatever narrows it.
func (p *parser) stringExpr() (Expr, error) {
	id := p.cur().text
	p.pos++
	switch {
	case p.accept(tokenKeyword, "at"):
		offset, err := p.expression(primaryPrecedence)
		if err != nil {
			return nil, err
		}
		return StringAt{ID: id, Offset: offset}, nil
	case p.accept(tokenKeyword, "in"):
		from, to, err := p.rangeExpr()
		if err != nil {
			return nil, err
		}
		return StringIn{ID: id, From: from, To: to}, nil
	}
	return StringRef{ID: id}, nil
}

// stringIndexed reads an offset or a length, which count from one and may say
// which match they mean.
func (p *parser) stringIndexed() (Expr, error) {
	tok := p.cur()
	p.pos++
	id := "$" + tok.text[1:]

	var index Expr = IntLit(1)
	if p.accept(tokenPunct, "[") {
		inner, err := p.expression(0)
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(tokenPunct, "]"); err != nil {
			return nil, err
		}
		index = inner
	}
	if tok.kind == tokenStringOffset {
		return StringOffset{ID: id, Index: index}, nil
	}
	return StringLengthOf{ID: id, Index: index}, nil
}

// rangeExpr reads a stretch written as "(from .. to)".
func (p *parser) rangeExpr() (Expr, Expr, error) {
	if _, err := p.expect(tokenPunct, "("); err != nil {
		return nil, nil, err
	}
	from, err := p.expression(0)
	if err != nil {
		return nil, nil, err
	}
	if _, err := p.expect(tokenPunct, ".."); err != nil {
		return nil, nil, err
	}
	to, err := p.expression(0)
	if err != nil {
		return nil, nil, err
	}
	if _, err := p.expect(tokenPunct, ")"); err != nil {
		return nil, nil, err
	}
	return from, to, nil
}

// keywordExpr reads the conditions that open with a word of the language.
func (p *parser) keywordExpr() (Expr, error) {
	tok := p.cur()
	switch tok.text {
	case "true", "false":
		p.pos++
		return BoolLit(tok.text == "true"), nil
	case "filesize":
		p.pos++
		return FileSize{}, nil
	case "entrypoint":
		p.pos++
		return EntryPoint{}, nil
	case "any", "all", "none":
		p.pos++
		return p.ofExpr(Quantifier{Kind: tok.text})
	case "for":
		p.pos++
		return p.forExpr()
	}
	return nil, p.syntaxError()
}

// ofExpr reads the set an of-expression runs over, the quantifier already read.
func (p *parser) ofExpr(q Quantifier) (Expr, error) {
	if _, err := p.expect(tokenKeyword, "of"); err != nil {
		return nil, err
	}
	set, err := p.stringSet()
	if err != nil {
		return nil, err
	}
	return Of{Quantifier: q, Set: set}, nil
}

// stringSet reads either "them" or a list of string names, which may hold
// wildcards.
func (p *parser) stringSet() (StringSet, error) {
	if p.accept(tokenKeyword, "them") {
		return StringSet{Them: true}, nil
	}
	if _, err := p.expect(tokenPunct, "("); err != nil {
		return StringSet{}, err
	}
	var set StringSet
	for {
		id, err := p.expect(tokenStringIdentifier)
		if err != nil {
			return StringSet{}, err
		}
		name := id.text
		if p.accept(tokenPunct, "*") {
			name += "*"
		}
		set.Items = append(set.Items, name)
		if !p.accept(tokenPunct, ",") {
			break
		}
	}
	if _, err := p.expect(tokenPunct, ")"); err != nil {
		return StringSet{}, err
	}
	return set, nil
}

// forExpr reads a loop, over either a stretch of numbers or a set of strings.
func (p *parser) forExpr() (Expr, error) {
	q, err := p.forQuantifier()
	if err != nil {
		return nil, err
	}

	if p.accept(tokenKeyword, "of") {
		set, err := p.stringSet()
		if err != nil {
			return nil, err
		}
		body, err := p.forBody()
		if err != nil {
			return nil, err
		}
		return ForOf{Quantifier: q, Set: set, Body: body}, nil
	}

	name, err := p.expect(tokenIdentifier)
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(tokenKeyword, "in"); err != nil {
		return nil, err
	}
	from, to, err := p.rangeExpr()
	if err != nil {
		return nil, err
	}
	body, err := p.forBody()
	if err != nil {
		return nil, err
	}
	return ForRange{Quantifier: q, Var: name.text, From: from, To: to, Body: body}, nil
}

// forQuantifier reads how many times round the loop has to hold.
func (p *parser) forQuantifier() (Quantifier, error) {
	tok := p.cur()
	if tok.kind == tokenKeyword && (tok.text == "any" || tok.text == "all" || tok.text == "none") {
		p.pos++
		return Quantifier{Kind: tok.text}, nil
	}
	if tok.kind == tokenInteger {
		p.pos++
		return Quantifier{Kind: "count", Count: IntLit(tok.num)}, nil
	}
	return Quantifier{}, p.syntaxError()
}

// forBody reads the condition a loop applies each time round.
func (p *parser) forBody() (Expr, error) {
	if _, err := p.expect(tokenPunct, ":"); err != nil {
		return nil, err
	}
	if _, err := p.expect(tokenPunct, "("); err != nil {
		return nil, err
	}
	body, err := p.expression(0)
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(tokenPunct, ")"); err != nil {
		return nil, err
	}
	return body, nil
}

// identifierExpr reads a name: one of the functions that read the data, a value
// a module offers, or another rule.
func (p *parser) identifierExpr() (Expr, error) {
	name := p.cur().text
	p.pos++

	if intFuncs[name] {
		if _, err := p.expect(tokenPunct, "("); err != nil {
			return nil, err
		}
		arg, err := p.expression(0)
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(tokenPunct, ")"); err != nil {
			return nil, err
		}
		return IntFunc{Name: name, X: arg}, nil
	}

	if !p.at(tokenPunct, ".") {
		return Ident(name), nil
	}
	return p.moduleRef(name)
}

// moduleRef reads the way to a value a module offers: names separated by dots,
// any of which may pick one out of a list or a table, or be a call.
func (p *parser) moduleRef(module string) (Expr, error) {
	ref := ModuleRef{Module: module}
	for p.accept(tokenPunct, ".") {
		member, err := p.expect(tokenIdentifier)
		if err != nil {
			return nil, err
		}
		step, err := p.moduleStep(member.text)
		if err != nil {
			return nil, err
		}
		ref.Steps = append(ref.Steps, step)
	}
	return ref, nil
}

// moduleStep reads what follows a name: where in a list or table to look, or
// what to call it with.
func (p *parser) moduleStep(name string) (ModuleStep, error) {
	step := ModuleStep{Name: name}
	if p.accept(tokenPunct, "[") {
		index, err := p.expression(0)
		if err != nil {
			return step, err
		}
		if _, err := p.expect(tokenPunct, "]"); err != nil {
			return step, err
		}
		step.Index = index
		return step, nil
	}
	if !p.accept(tokenPunct, "(") {
		return step, nil
	}

	step.Call = true
	if !p.at(tokenPunct, ")") {
		for {
			arg, err := p.expression(0)
			if err != nil {
				return step, err
			}
			step.Args = append(step.Args, arg)
			if !p.accept(tokenPunct, ",") {
				break
			}
		}
	}
	_, err := p.expect(tokenPunct, ")")
	return step, err
}
