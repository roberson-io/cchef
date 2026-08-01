package ops

// Statement and expression node handlers for the JavaScript Beautify code
// generator (transliterated from escodegen's CodeGenerator.Statement /
// .Expression maps). Dispatch is via handler maps to mirror escodegen's
// `this[type]` lookup and keep any single function's complexity low.

import "strings"

func jsStr(v any) string { s, _ := v.(string); return s }

func (g *jsGen) generateExpression(expr any, precedence, flags int) string {
	h, ok := g.exprH[jsNodeType(expr)]
	if !ok {
		return ""
	}
	result := h(expr, precedence, flags)
	if g.comments != nil {
		result = g.addComments(expr, result)
	}
	return result
}

func (g *jsGen) generateStatement(stmt any, flags int) string {
	h, ok := g.stmtH[jsNodeType(stmt)]
	if !ok {
		return ""
	}
	result := h(stmt, flags)
	if g.comments != nil {
		result = g.addComments(stmt, result)
	}
	return result
}

func (g *jsGen) buildHandlers() {
	g.stmtH = map[string]func(any, int) string{
		"Program":             g.genProgram,
		"BlockStatement":      g.genBlockStatement,
		"ExpressionStatement": g.genExpressionStatement,
		"VariableDeclaration": g.genVariableDeclaration,
		"VariableDeclarator":  g.genVariableDeclarator,
		"FunctionDeclaration": g.genFunctionDeclaration,
		"ReturnStatement":     g.genReturnStatement,
		"IfStatement":         g.genIfStatement,
		"ForStatement":        g.genForStatement,
		"ForInStatement":      g.genForInStatement,
		"ForOfStatement":      g.genForOfStatement,
		"WhileStatement":      g.genWhileStatement,
		"DoWhileStatement":    g.genDoWhileStatement,
		"BreakStatement":      g.genBreakStatement,
		"ContinueStatement":   g.genContinueStatement,
		"ThrowStatement":      g.genThrowStatement,
		"TryStatement":        g.genTryStatement,
		"CatchClause":         g.genCatchClause,
		"SwitchStatement":     g.genSwitchStatement,
		"SwitchCase":          g.genSwitchCase,
		"LabeledStatement":    g.genLabeledStatement,
		"EmptyStatement":      g.genEmptyStatement,
		"DebuggerStatement":   g.genDebuggerStatement,
		"WithStatement":       g.genWithStatement,
		"ClassDeclaration":    g.genClassDeclaration,
		"ClassBody":           g.genClassBody,
	}
	g.exprH = map[string]func(any, int, int) string{
		"SequenceExpression":       g.genSequenceExpression,
		"AssignmentExpression":     g.genAssignmentExpression,
		"ArrowFunctionExpression":  g.genArrowFunctionExpression,
		"ConditionalExpression":    g.genConditionalExpression,
		"LogicalExpression":        g.genLogicalExpression,
		"BinaryExpression":         g.genBinaryExpression,
		"CallExpression":           g.genCallExpression,
		"NewExpression":            g.genNewExpression,
		"MemberExpression":         g.genMemberExpression,
		"MetaProperty":             g.genMetaProperty,
		"UnaryExpression":          g.genUnaryExpression,
		"YieldExpression":          g.genYieldExpression,
		"AwaitExpression":          g.genAwaitExpression,
		"UpdateExpression":         g.genUpdateExpression,
		"FunctionExpression":       g.genFunctionExpression,
		"ArrayExpression":          g.genArrayExpression,
		"ArrayPattern":             g.genArrayPattern,
		"RestElement":              g.genRestElement,
		"ClassExpression":          g.genClassExpression,
		"MethodDefinition":         g.genMethodDefinition,
		"Property":                 g.genProperty,
		"ObjectExpression":         g.genObjectExpression,
		"AssignmentPattern":        g.genAssignmentPattern,
		"ObjectPattern":            g.genObjectPattern,
		"ThisExpression":           g.genThisExpression,
		"Super":                    g.genSuper,
		"Identifier":               g.genIdentifierExpr,
		"Literal":                  g.genLiteral,
		"SpreadElement":            g.genSpreadElement,
		"TaggedTemplateExpression": g.genTaggedTemplateExpression,
		"TemplateElement":          g.genTemplateElement,
		"TemplateLiteral":          g.genTemplateLiteral,
	}
}

// --- statements ---

func (g *jsGen) genProgram(stmt any, flags int) string {
	body := jsFieldSlice(stmt, "body")
	var result strings.Builder
	for i, s := range body {
		bf := sTFTF
		if i == len(body)-1 {
			bf |= fSemicolonOpt
		}
		fragment := g.addIndent(g.generateStatement(s, bf))
		result.WriteString(fragment)
		if i+1 < len(body) && !endsWithLineTerminator(fragment) {
			result.WriteString(g.newline)
		}
	}
	return result.String()
}

func (g *jsGen) genBlockStatement(stmt any, flags int) string {
	var result strings.Builder
	result.WriteString("{" + g.newline)
	g.withIndent(func() {
		body := jsFieldSlice(stmt, "body")
		bodyFlags := sTFFF
		if flags&fFuncBody != 0 {
			bodyFlags |= fDirectiveCtx
		}
		for i, s := range body {
			bf := bodyFlags
			if i == len(body)-1 {
				bf |= fSemicolonOpt
			}
			fragment := g.addIndent(g.generateStatement(s, bf))
			result.WriteString(fragment)
			if !endsWithLineTerminator(fragment) {
				result.WriteString(g.newline)
			}
		}
	})
	result.WriteString(g.addIndent("}"))
	return result.String()
}

func (g *jsGen) genExpressionStatement(stmt any, flags int) string {
	result := g.generateExpression(jsField(stmt, "expression"), pSequence, eTTT)
	if len(result) > 0 && (result[0] == '{' || isClassPrefixed(result) || isFunctionPrefixed(result) || isAsyncPrefixed(result)) {
		return "(" + result + ")" + g.semicolon(flags)
	}
	return result + g.semicolon(flags)
}

func (g *jsGen) genVariableDeclaration(stmt any, flags int) string {
	kind := jsStr(jsField(stmt, "kind"))
	decls := jsFieldSlice(stmt, "declarations")
	bodyFlags := sFFFF
	if flags&fAllowIn != 0 {
		bodyFlags = sTFFF
	}
	var result strings.Builder
	result.WriteString(kind)
	block := func() {
		if len(g.jsNodeLeading(decls[0])) > 0 {
			result.WriteString("\n" + g.addIndent(g.generateStatement(decls[0], bodyFlags)))
		} else {
			result.WriteString(" " + g.generateStatement(decls[0], bodyFlags))
		}
		for i := 1; i < len(decls); i++ {
			if len(g.jsNodeLeading(decls[i])) > 0 {
				result.WriteString("," + g.newline + g.addIndent(g.generateStatement(decls[i], bodyFlags)))
			} else {
				result.WriteString("," + g.space + g.generateStatement(decls[i], bodyFlags))
			}
		}
	}
	if len(decls) > 1 {
		g.withIndent(block)
	} else {
		block()
	}
	return result.String() + g.semicolon(flags)
}

func (g *jsGen) genVariableDeclarator(stmt any, flags int) string {
	itemFlags := eFTT
	if flags&fAllowIn != 0 {
		itemFlags = eTTT
	}
	if init := jsField(stmt, "init"); init != nil {
		return g.generateExpression(jsField(stmt, "id"), pAssignment, itemFlags) +
			g.space + "=" + g.space + g.generateExpression(init, pAssignment, itemFlags)
	}
	return g.generatePattern(jsField(stmt, "id"), pAssignment, itemFlags)
}

func (g *jsGen) genFunctionDeclaration(stmt any, flags int) string {
	star := g.generateStarSuffix(stmt)
	if star == "" {
		star = " "
	}
	id := ""
	if v := jsField(stmt, "id"); v != nil {
		id = g.generateIdentifier(v)
	}
	return g.generateAsyncPrefix(stmt, true) + "function" + star + id + g.generateFunctionBody(stmt)
}

func (g *jsGen) genReturnStatement(stmt any, flags int) string {
	if arg := jsField(stmt, "argument"); arg != nil {
		return g.join("return", g.generateExpression(arg, pSequence, eTTT)) + g.semicolon(flags)
	}
	return "return" + g.semicolon(flags)
}

func (g *jsGen) genIfStatement(stmt any, flags int) string {
	var result string
	g.withIndent(func() {
		result = "if" + g.space + "(" + g.generateExpression(jsField(stmt, "test"), pSequence, eTTT) + ")"
	})
	bodyFlags := sTFFF
	if flags&fSemicolonOpt != 0 {
		bodyFlags |= fSemicolonOpt
	}
	if alt := jsField(stmt, "alternate"); alt != nil {
		result += g.maybeBlock(jsField(stmt, "consequent"), sTFFF)
		result = g.maybeBlockSuffix(jsField(stmt, "consequent"), result)
		if jsNodeType(alt) == "IfStatement" {
			result = g.join(result, "else "+g.generateStatement(alt, bodyFlags))
		} else {
			result = g.join(result, g.join("else", g.maybeBlock(alt, bodyFlags)))
		}
	} else {
		result += g.maybeBlock(jsField(stmt, "consequent"), bodyFlags)
	}
	return result
}

func (g *jsGen) genForStatement(stmt any, flags int) string {
	var result string
	g.withIndent(func() {
		result = "for" + g.space + "("
		if init := jsField(stmt, "init"); init != nil {
			if jsNodeType(init) == "VariableDeclaration" {
				result += g.generateStatement(init, sFFFF)
			} else {
				result += g.generateExpression(init, pSequence, eFTT) + ";"
			}
		} else {
			result += ";"
		}
		if test := jsField(stmt, "test"); test != nil {
			result += g.space + g.generateExpression(test, pSequence, eTTT) + ";"
		} else {
			result += ";"
		}
		if upd := jsField(stmt, "update"); upd != nil {
			result += g.space + g.generateExpression(upd, pSequence, eTTT) + ")"
		} else {
			result += ")"
		}
	})
	bf := sTFFF
	if flags&fSemicolonOpt != 0 {
		bf = sTFFT
	}
	return result + g.maybeBlock(jsField(stmt, "body"), bf)
}

func (g *jsGen) genForInStatement(stmt any, flags int) string {
	return g.generateIterationForStatement("in", stmt, forBodyFlags(flags))
}

func (g *jsGen) genForOfStatement(stmt any, flags int) string {
	return g.generateIterationForStatement("of", stmt, forBodyFlags(flags))
}

func forBodyFlags(flags int) int {
	if flags&fSemicolonOpt != 0 {
		return sTFFT
	}
	return sTFFF
}

func (g *jsGen) genWhileStatement(stmt any, flags int) string {
	var result string
	g.withIndent(func() {
		result = "while" + g.space + "(" + g.generateExpression(jsField(stmt, "test"), pSequence, eTTT) + ")"
	})
	return result + g.maybeBlock(jsField(stmt, "body"), forBodyFlags(flags))
}

func (g *jsGen) genDoWhileStatement(stmt any, flags int) string {
	result := g.join("do", g.maybeBlock(jsField(stmt, "body"), sTFFF))
	result = g.maybeBlockSuffix(jsField(stmt, "body"), result)
	return g.join(result, "while"+g.space+"("+g.generateExpression(jsField(stmt, "test"), pSequence, eTTT)+")"+g.semicolon(flags))
}

func (g *jsGen) genBreakStatement(stmt any, flags int) string {
	if l := jsField(stmt, "label"); l != nil {
		return "break " + jsIdentName(l) + g.semicolon(flags)
	}
	return "break" + g.semicolon(flags)
}

func (g *jsGen) genContinueStatement(stmt any, flags int) string {
	if l := jsField(stmt, "label"); l != nil {
		return "continue " + jsIdentName(l) + g.semicolon(flags)
	}
	return "continue" + g.semicolon(flags)
}

func (g *jsGen) genThrowStatement(stmt any, flags int) string {
	return g.join("throw", g.generateExpression(jsField(stmt, "argument"), pSequence, eTTT)) + g.semicolon(flags)
}

func (g *jsGen) genTryStatement(stmt any, flags int) string {
	result := "try" + g.maybeBlock(jsField(stmt, "block"), sTFFF)
	result = g.maybeBlockSuffix(jsField(stmt, "block"), result)
	if h := jsField(stmt, "handler"); h != nil {
		result = g.join(result, g.generateStatement(h, sTFFF))
		if jsField(stmt, "finalizer") != nil {
			result = g.maybeBlockSuffix(jsField(h, "body"), result)
		}
	}
	if f := jsField(stmt, "finalizer"); f != nil {
		result = g.join(result, "finally"+g.maybeBlock(f, sTFFF))
	}
	return result
}

func (g *jsGen) genCatchClause(stmt any, flags int) string {
	var result string
	g.withIndent(func() {
		if p := jsField(stmt, "param"); p != nil {
			result = "catch" + g.space + "(" + g.generateExpression(p, pSequence, eTTT) + ")"
		} else {
			result = "catch"
		}
	})
	return result + g.maybeBlock(jsField(stmt, "body"), sTFFF)
}

func (g *jsGen) genSwitchStatement(stmt any, flags int) string {
	var result string
	g.withIndent(func() {
		result = "switch" + g.space + "(" + g.generateExpression(jsField(stmt, "discriminant"), pSequence, eTTT) + ")" + g.space + "{" + g.newline
	})
	cases := jsFieldSlice(stmt, "cases")
	for i, c := range cases {
		bf := sTFFF
		if i == len(cases)-1 {
			bf |= fSemicolonOpt
		}
		fragment := g.addIndent(g.generateStatement(c, bf))
		result += fragment
		if !endsWithLineTerminator(fragment) {
			result += g.newline
		}
	}
	return result + g.addIndent("}")
}

func (g *jsGen) genSwitchCase(stmt any, flags int) string {
	var result string
	g.withIndent(func() {
		if t := jsField(stmt, "test"); t != nil {
			result = g.join("case", g.generateExpression(t, pSequence, eTTT)) + ":"
		} else {
			result = "default:"
		}
		consequent := jsFieldSlice(stmt, "consequent")
		i, iz := 0, len(consequent)
		if iz > 0 && jsNodeType(consequent[0]) == "BlockStatement" {
			result += g.maybeBlock(consequent[0], sTFFF)
			i = 1
		}
		if i != iz && !endsWithLineTerminator(result) {
			result += g.newline
		}
		for ; i < iz; i++ {
			bf := sTFFF
			if i == iz-1 && flags&fSemicolonOpt != 0 {
				bf |= fSemicolonOpt
			}
			fragment := g.addIndent(g.generateStatement(consequent[i], bf))
			result += fragment
			if i+1 != iz && !endsWithLineTerminator(fragment) {
				result += g.newline
			}
		}
	})
	return result
}

func (g *jsGen) genLabeledStatement(stmt any, flags int) string {
	return jsIdentName(jsField(stmt, "label")) + ":" + g.maybeBlock(jsField(stmt, "body"), forBodyFlags(flags))
}

func (g *jsGen) genEmptyStatement(stmt any, flags int) string { return ";" }

func (g *jsGen) genDebuggerStatement(stmt any, flags int) string {
	return "debugger" + g.semicolon(flags)
}

func (g *jsGen) genWithStatement(stmt any, flags int) string {
	var result string
	g.withIndent(func() {
		result = "with" + g.space + "(" + g.generateExpression(jsField(stmt, "object"), pSequence, eTTT) + ")"
	})
	return result + g.maybeBlock(jsField(stmt, "body"), forBodyFlags(flags))
}

func (g *jsGen) genClassDeclaration(stmt any, flags int) string {
	result := "class"
	if id := jsField(stmt, "id"); id != nil {
		result = g.join(result, g.generateExpression(id, pSequence, eTTT))
	}
	if sc := jsField(stmt, "superClass"); sc != nil {
		result = g.join(result, g.join("extends", g.generateExpression(sc, pUnary, eTTT)))
	}
	return result + g.space + g.generateStatement(jsField(stmt, "body"), sTFFT)
}

func (g *jsGen) genClassBody(stmt any, flags int) string {
	result := "{" + g.newline
	g.withIndent(func() {
		body := jsFieldSlice(stmt, "body")
		for i, el := range body {
			result += g.base + g.generateExpression(el, pSequence, eTTT)
			if i+1 < len(body) {
				result += g.newline
			}
		}
	})
	if !endsWithLineTerminator(result) {
		result += g.newline
	}
	return result + g.base + "}"
}

// --- expressions ---

func (g *jsGen) genSequenceExpression(expr any, precedence, flags int) string {
	if pSequence < precedence {
		flags |= fAllowIn
	}
	exprs := jsFieldSlice(expr, "expressions")
	var b strings.Builder
	for i, e := range exprs {
		b.WriteString(g.generateExpression(e, pAssignment, flags))
		if i+1 < len(exprs) {
			b.WriteString("," + g.space)
		}
	}
	return parenthesize(b.String(), pSequence, precedence)
}

func (g *jsGen) genAssignmentExpression(expr any, precedence, flags int) string {
	return g.generateAssignment(jsField(expr, "left"), jsField(expr, "right"), jsStr(jsField(expr, "operator")), precedence, flags)
}

func (g *jsGen) genArrowFunctionExpression(expr any, precedence, flags int) string {
	return parenthesize(g.generateFunctionBody(expr), pArrowFunction, precedence)
}

func (g *jsGen) genConditionalExpression(expr any, precedence, flags int) string {
	if pConditional < precedence {
		flags |= fAllowIn
	}
	s := g.generateExpression(jsField(expr, "test"), pCoalesce, flags) +
		g.space + "?" + g.space + g.generateExpression(jsField(expr, "consequent"), pAssignment, flags) +
		g.space + ":" + g.space + g.generateExpression(jsField(expr, "alternate"), pAssignment, flags)
	return parenthesize(s, pConditional, precedence)
}

func (g *jsGen) genLogicalExpression(expr any, precedence, flags int) string {
	return g.genBinaryExpression(expr, precedence, flags)
}

func (g *jsGen) genBinaryExpression(expr any, precedence, flags int) string {
	op := jsStr(jsField(expr, "operator"))
	current := jsBinaryPrecedence[op]
	leftPrec, rightPrec := current, current+1
	if op == "**" {
		leftPrec, rightPrec = pPostfix, current
	}
	if current < precedence {
		flags |= fAllowIn
	}
	fragment := g.generateExpression(jsField(expr, "left"), leftPrec, flags)
	var result string
	if lastRune(fragment) == '/' && isIdentPartES5(firstRune(op)) {
		result = fragment + " " + op
	} else {
		result = g.join(fragment, op)
	}
	right := g.generateExpression(jsField(expr, "right"), rightPrec, flags)
	if (op == "/" && firstRune(right) == '/') || (strings.HasSuffix(op, "<") && strings.HasPrefix(right, "!--")) {
		result = result + " " + right
	} else {
		result = g.join(result, right)
	}
	if op == "in" && flags&fAllowIn == 0 {
		return "(" + result + ")"
	}
	return parenthesize(result, current, precedence)
}

func (g *jsGen) genCallExpression(expr any, precedence, flags int) string {
	var result strings.Builder
	result.WriteString(g.generateExpression(jsField(expr, "callee"), pCall, eTTF) + "(")
	args := jsFieldSlice(expr, "arguments")
	for i, a := range args {
		result.WriteString(g.generateExpression(a, pAssignment, eTTT))
		if i+1 < len(args) {
			result.WriteString("," + g.space)
		}
	}
	result.WriteString(")")
	if flags&fAllowCall == 0 {
		return "(" + result.String() + ")"
	}
	return parenthesize(result.String(), pCall, precedence)
}

func (g *jsGen) genNewExpression(expr any, precedence, flags int) string {
	// parentheses is always on for CyberChef, so itemFlags is always E_TFF and
	// the argument list is always emitted.
	var result strings.Builder
	result.WriteString(g.join("new", g.generateExpression(jsField(expr, "callee"), pNew, eTFF)) + "(")
	args := jsFieldSlice(expr, "arguments")
	for i, a := range args {
		result.WriteString(g.generateExpression(a, pAssignment, eTTT))
		if i+1 < len(args) {
			result.WriteString("," + g.space)
		}
	}
	result.WriteString(")")
	return parenthesize(result.String(), pNew, precedence)
}

func (g *jsGen) genMemberExpression(expr any, precedence, flags int) string {
	objFlags := eTFF
	if flags&fAllowCall != 0 {
		objFlags = eTTF
	}
	result := g.generateExpression(jsField(expr, "object"), pCall, objFlags)
	if jsFieldBool(expr, "computed") {
		propFlags := eTFT
		if flags&fAllowCall != 0 {
			propFlags = eTTT
		}
		result += "[" + g.generateExpression(jsField(expr, "property"), pSequence, propFlags) + "]"
	} else {
		obj := jsField(expr, "object")
		if jsNodeType(obj) == "Literal" {
			if _, ok := jsField(obj, "value").(float64); ok && needsNumberDot(result) {
				result += " "
			}
		}
		result += "." + g.generateIdentifier(jsField(expr, "property"))
	}
	return parenthesize(result, pMember, precedence)
}

func (g *jsGen) genMetaProperty(expr any, precedence, flags int) string {
	return parenthesize(jsIdentName(jsField(expr, "meta"))+"."+jsIdentName(jsField(expr, "property")), pMember, precedence)
}

func (g *jsGen) genUnaryExpression(expr any, precedence, flags int) string {
	op := jsStr(jsField(expr, "operator"))
	fragment := g.generateExpression(jsField(expr, "argument"), pUnary, eTTT)
	var result string
	if len(op) > 2 {
		result = g.join(op, fragment)
	} else {
		l, r := lastRune(op), firstRune(fragment)
		if ((l == '+' || l == '-') && l == r) || (isIdentPartES5(l) && isIdentPartES5(r)) {
			result = op + " " + fragment
		} else {
			result = op + fragment
		}
	}
	return parenthesize(result, pUnary, precedence)
}

func (g *jsGen) genYieldExpression(expr any, precedence, flags int) string {
	result := "yield"
	if jsFieldBool(expr, "delegate") {
		result = "yield*"
	}
	if arg := jsField(expr, "argument"); arg != nil {
		result = g.join(result, g.generateExpression(arg, pYield, eTTT))
	}
	return parenthesize(result, pYield, precedence)
}

func (g *jsGen) genAwaitExpression(expr any, precedence, flags int) string {
	return parenthesize(g.join("await", g.generateExpression(jsField(expr, "argument"), pAwait, eTTT)), pAwait, precedence)
}

func (g *jsGen) genUpdateExpression(expr any, precedence, flags int) string {
	op := jsStr(jsField(expr, "operator"))
	if jsFieldBool(expr, "prefix") {
		return parenthesize(op+g.generateExpression(jsField(expr, "argument"), pUnary, eTTT), pUnary, precedence)
	}
	return parenthesize(g.generateExpression(jsField(expr, "argument"), pPostfix, eTTT)+op, pPostfix, precedence)
}

func (g *jsGen) genFunctionExpression(expr any, precedence, flags int) string {
	result := g.generateAsyncPrefix(expr, true) + "function"
	star := g.generateStarSuffix(expr)
	if id := jsField(expr, "id"); id != nil {
		if star == "" {
			star = " "
		}
		result += star + g.generateIdentifier(id)
	} else {
		if star == "" {
			star = g.space
		}
		result += star
	}
	return result + g.generateFunctionBody(expr)
}

func (g *jsGen) genArrayExpression(expr any, precedence, flags int) string {
	return g.arrayExpr(expr, false)
}

func (g *jsGen) genArrayPattern(expr any, precedence, flags int) string {
	return g.arrayExpr(expr, true)
}

func (g *jsGen) arrayExpr(expr any, isPattern bool) string {
	elements := jsFieldSlice(expr, "elements")
	if len(elements) == 0 {
		return "[]"
	}
	multiline := !isPattern && len(elements) > 1
	result := "["
	if multiline {
		result += g.newline
	}
	g.withIndent(func() {
		for i, el := range elements {
			if el == nil {
				if multiline {
					result += g.base
				}
				if i+1 == len(elements) {
					result += ","
				}
			} else {
				if multiline {
					result += g.base
				}
				result += g.generateExpression(el, pAssignment, eTTT)
			}
			if i+1 < len(elements) {
				if multiline {
					result += "," + g.newline
				} else {
					result += "," + g.space
				}
			}
		}
	})
	if multiline && !endsWithLineTerminator(result) {
		result += g.newline
	}
	if multiline {
		result += g.base
	}
	return result + "]"
}

func (g *jsGen) genRestElement(expr any, precedence, flags int) string {
	return "..." + g.generatePattern(jsField(expr, "argument"), 0, 0)
}

func (g *jsGen) genClassExpression(expr any, precedence, flags int) string {
	result := "class"
	if id := jsField(expr, "id"); id != nil {
		result = g.join(result, g.generateExpression(id, pSequence, eTTT))
	}
	if sc := jsField(expr, "superClass"); sc != nil {
		result = g.join(result, g.join("extends", g.generateExpression(sc, pUnary, eTTT)))
	}
	return result + g.space + g.generateStatement(jsField(expr, "body"), sTFFT)
}

func (g *jsGen) genMethodDefinition(expr any, precedence, flags int) string {
	result := ""
	if jsFieldBool(expr, "static") {
		result = "static" + g.space
	}
	key := g.generatePropertyKey(jsField(expr, "key"), jsFieldBool(expr, "computed"))
	body := g.generateFunctionBody(jsField(expr, "value"))
	var fragment string
	if kind := jsStr(jsField(expr, "kind")); kind == "get" || kind == "set" {
		fragment = g.join(kind, key) + body
	} else {
		fragment = g.generateMethodPrefix(expr) + key + body
	}
	return g.join(result, fragment)
}

func (g *jsGen) genProperty(expr any, precedence, flags int) string {
	computed := jsFieldBool(expr, "computed")
	if kind := jsStr(jsField(expr, "kind")); kind == "get" || kind == "set" {
		return kind + " " + g.generatePropertyKey(jsField(expr, "key"), computed) + g.generateFunctionBody(jsField(expr, "value"))
	}
	if jsFieldBool(expr, "shorthand") {
		val := jsField(expr, "value")
		if jsNodeType(val) == "AssignmentPattern" {
			return g.genAssignmentPattern(val, pSequence, eTTT)
		}
		return g.generatePropertyKey(jsField(expr, "key"), computed)
	}
	if jsFieldBool(expr, "method") {
		return g.generateMethodPrefix(expr) + g.generatePropertyKey(jsField(expr, "key"), computed) + g.generateFunctionBody(jsField(expr, "value"))
	}
	return g.generatePropertyKey(jsField(expr, "key"), computed) + ":" + g.space + g.generateExpression(jsField(expr, "value"), pAssignment, eTTT)
}

func (g *jsGen) genObjectExpression(expr any, precedence, flags int) string {
	props := jsFieldSlice(expr, "properties")
	if len(props) == 0 {
		return "{}"
	}
	multiline := len(props) > 1
	var fragment string
	g.withIndent(func() {
		fragment = g.generateExpression(props[0], pSequence, eTTT)
	})
	if !multiline && !strings.ContainsAny(fragment, "\r\n") {
		return "{" + g.space + fragment + g.space + "}"
	}
	var result string
	g.withIndent(func() {
		result = "{" + g.newline + g.base + fragment
		if multiline {
			result += "," + g.newline
			for i := 1; i < len(props); i++ {
				result += g.base + g.generateExpression(props[i], pSequence, eTTT)
				if i+1 < len(props) {
					result += "," + g.newline
				}
			}
		}
	})
	if !endsWithLineTerminator(result) {
		result += g.newline
	}
	return result + g.base + "}"
}

func (g *jsGen) genAssignmentPattern(expr any, precedence, flags int) string {
	return g.generateAssignment(jsField(expr, "left"), jsField(expr, "right"), "=", precedence, flags)
}

func (g *jsGen) genObjectPattern(expr any, precedence, flags int) string {
	props := jsFieldSlice(expr, "properties")
	if len(props) == 0 {
		return "{}"
	}
	multiline := objectPatternMultiline(props)
	result := "{"
	if multiline {
		result += g.newline
	}
	g.withIndent(func() {
		for i, p := range props {
			if multiline {
				result += g.base
			}
			result += g.generateExpression(p, pSequence, eTTT)
			if i+1 < len(props) {
				if multiline {
					result += "," + g.newline
				} else {
					result += "," + g.space
				}
			}
		}
	})
	if multiline && !endsWithLineTerminator(result) {
		result += g.newline
	}
	if multiline {
		result += g.base
	}
	return result + "}"
}

func objectPatternMultiline(props []any) bool {
	if len(props) == 1 {
		p := props[0]
		return jsNodeType(p) == "Property" && jsNodeType(jsField(p, "value")) != "Identifier"
	}
	for _, p := range props {
		if jsNodeType(p) == "Property" && !jsFieldBool(p, "shorthand") {
			return true
		}
	}
	return false
}

func (g *jsGen) genThisExpression(expr any, precedence, flags int) string { return "this" }
func (g *jsGen) genSuper(expr any, precedence, flags int) string          { return "super" }
func (g *jsGen) genIdentifierExpr(expr any, precedence, flags int) string {
	return g.generateIdentifier(expr)
}

func (g *jsGen) genLiteral(expr any, precedence, flags int) string {
	if r := jsField(expr, "regex"); r != nil {
		return "/" + jsStr(jsField(r, "pattern")) + "/" + jsStr(jsField(r, "flags"))
	}
	switch val := jsField(expr, "value").(type) {
	case string:
		return g.escapeString(val)
	case float64:
		return jsFormatNumber(val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	default:
		return "null"
	}
}

func (g *jsGen) genSpreadElement(expr any, precedence, flags int) string {
	return "..." + g.generateExpression(jsField(expr, "argument"), pAssignment, eTTT)
}

func (g *jsGen) genTaggedTemplateExpression(expr any, precedence, flags int) string {
	itemFlags := eTTF
	if flags&fAllowCall == 0 {
		itemFlags = eTFF
	}
	result := g.generateExpression(jsField(expr, "tag"), pCall, itemFlags) +
		g.generateExpression(jsField(expr, "quasi"), pPrimary, eFFT)
	return parenthesize(result, pTaggedTemplate, precedence)
}

func (g *jsGen) genTemplateElement(expr any, precedence, flags int) string {
	return jsStr(jsField(jsField(expr, "value"), "raw"))
}

func (g *jsGen) genTemplateLiteral(expr any, precedence, flags int) string {
	quasis := jsFieldSlice(expr, "quasis")
	exprs := jsFieldSlice(expr, "expressions")
	var result strings.Builder
	result.WriteString("`")
	for i, q := range quasis {
		result.WriteString(g.generateExpression(q, pPrimary, eTTT))
		if i+1 < len(quasis) {
			result.WriteString("${" + g.space + g.generateExpression(exprs[i], pSequence, eTTT) + g.space + "}")
		}
	}
	return result.String() + "`"
}

// --- small helpers ---

func needsNumberDot(fragment string) bool {
	if strings.ContainsAny(fragment, ".eExX") {
		return false
	}
	last := fragment[len(fragment)-1]
	if last < '0' || last > '9' {
		return false
	}
	return len(fragment) < 2 || fragment[0] != '0'
}

func isPrefixedBy(fragment, word string, ok func(byte) bool) bool {
	if len(fragment) < len(word) || fragment[:len(word)] != word {
		return false
	}
	if len(fragment) == len(word) {
		return ok(0)
	}
	return ok(fragment[len(word)])
}

func isClassPrefixed(fragment string) bool {
	return isPrefixedBy(fragment, "class", func(c byte) bool {
		return c == '{' || isJSWhiteSpace(int(c)) || jsIsLineTerm(int(c))
	})
}

func isFunctionPrefixed(fragment string) bool {
	return isPrefixedBy(fragment, "function", func(c byte) bool {
		return c == '(' || c == '*' || isJSWhiteSpace(int(c)) || jsIsLineTerm(int(c))
	})
}

func isAsyncPrefixed(fragment string) bool {
	if len(fragment) < 6 || fragment[:5] != "async" || !isJSWhiteSpace(int(fragment[5])) {
		return false
	}
	i := 6
	for i < len(fragment) && isJSWhiteSpace(int(fragment[i])) {
		i++
	}
	return i != len(fragment) && isFunctionPrefixed(fragment[i:])
}
