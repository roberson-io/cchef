package jsbeautify

// Comment attachment for JavaScript Beautify — a transliteration of estraverse's
// attachComments (the algorithm escodegen.attachComments delegates to). It takes
// the AST (with node ranges), the flat comments array, and the flat tokens array
// produced by jsparse.ParseFull, and computes which node each comment leads or trails.
//
// esprima/estraverse mutate the AST in place (node.leadingComments/
// trailingComments). Here the AST nodes are immutable jsonval.Object values, so instead
// attachments are returned in side maps keyed by nodeKey (type@start:end) — no two
// distinct nodes share a type and an exact range, so the key is unambiguous.

import (
	"fmt"

	"github.com/roberson-io/cchef/internal/jsonval"
	"github.com/roberson-io/cchef/internal/jsparse"
)

// jsComment is a comment during attachment: its own range and the extended range
// (the gap between the surrounding tokens) used to decide attachment.
type jsComment struct {
	typ, value           string
	lo, hi, extLo, extHi int
}

// jsCommentSet holds the leading/trailing comments attached to each node.
type jsCommentSet struct {
	leading  map[string][]*jsComment
	trailing map[string][]*jsComment
}

func jsNodeKey(node any) string {
	obj, _ := node.(jsonval.Object)
	r, _ := jsparse.Field(obj, "range").([]any)
	if len(r) != 2 {
		return ""
	}
	return fmt.Sprintf("%s@%d:%d", jsparse.NodeType(obj), jsRangeVal(r[0]), jsRangeVal(r[1]))
}

func jsRangeVal(v any) int {
	if f, ok := v.(float64); ok {
		return int(f)
	}
	return -1
}

func jsNodeRange(node any) (int, int, bool) {
	obj, _ := node.(jsonval.Object)
	r, ok := jsparse.Field(obj, "range").([]any)
	if !ok || len(r) != 2 {
		return 0, 0, false
	}
	return jsRangeVal(r[0]), jsRangeVal(r[1]), true
}

// upperBound returns the index of the first element for which pred is true, or
// len(items) if none (estraverse's upperBound).
func jsUpperBound(items []jsonval.Object, pred func(jsonval.Object) bool) int {
	lo, hi := 0, len(items)
	for lo < hi {
		mid := (lo + hi) / 2
		if pred(items[mid]) {
			hi = mid
		} else {
			lo = mid + 1
		}
	}
	return lo
}

// extendCommentRange sets a comment's extended range to the gap between the
// tokens surrounding it (estraverse.extendCommentRange).
func extendCommentRange(c *jsComment, tokens []jsonval.Object) {
	c.extLo, c.extHi = c.lo, c.hi
	target := jsUpperBound(tokens, func(t jsonval.Object) bool {
		r, _ := jsparse.Field(t, "range").([]any)
		return jsRangeVal(r[0]) > c.lo
	})
	if target != len(tokens) {
		r, _ := jsparse.Field(tokens[target], "range").([]any)
		c.extHi = jsRangeVal(r[0])
	}
	if target-1 >= 0 {
		r, _ := jsparse.Field(tokens[target-1], "range").([]any)
		c.extLo = jsRangeVal(r[1])
	}
}

const (
	jsTravContinue = iota
	jsTravSkip
	jsTravBreak
)

// jsTraverse walks the AST depth-first: enter is called pre-order, leave
// post-order (either may be nil). enter returning jsTravSkip prunes children;
// either returning jsTravBreak stops the whole traversal.
type jsTraverser struct {
	enter, leave func(jsonval.Object) int
}

func (t *jsTraverser) visit(node jsonval.Object) int {
	if t.enter != nil {
		switch t.enter(node) {
		case jsTravBreak:
			return jsTravBreak
		case jsTravSkip:
			return t.leaveOnly(node)
		}
	}
	for _, key := range jsVisitorKeys[jsparse.NodeType(node)] {
		switch child := jsparse.Field(node, key).(type) {
		case jsonval.Object:
			if t.visit(child) == jsTravBreak {
				return jsTravBreak
			}
		case []any:
			for _, c := range child {
				if co, ok := c.(jsonval.Object); ok {
					if t.visit(co) == jsTravBreak {
						return jsTravBreak
					}
				}
			}
		}
	}
	return t.leaveOnly(node)
}

func (t *jsTraverser) leaveOnly(node jsonval.Object) int {
	if t.leave != nil && t.leave(node) == jsTravBreak {
		return jsTravBreak
	}
	return jsTravContinue
}

// AttachComments computes leading/trailing comment attachments for the AST,
// transliterating estraverse.attachComments (the non-preserve-blank-lines path).
func AttachComments(tree jsonval.Object, rawComments, tokens []jsonval.Object) *jsCommentSet {
	set := &jsCommentSet{leading: map[string][]*jsComment{}, trailing: map[string][]*jsComment{}}
	if len(rawComments) == 0 {
		return set
	}

	comments := make([]*jsComment, len(rawComments))
	for i, rc := range rawComments {
		r, _ := jsparse.Field(rc, "range").([]any)
		comments[i] = &jsComment{
			typ:   jsStr(jsparse.Field(rc, "type")),
			value: jsStr(jsparse.Field(rc, "value")),
			lo:    jsRangeVal(r[0]),
			hi:    jsRangeVal(r[1]),
		}
	}

	// No tokens: everything leads the tree.
	if len(tokens) == 0 {
		lo, _, _ := jsNodeRange(tree)
		for _, c := range comments {
			c.extLo, c.extHi = 0, lo
		}
		set.leading[jsNodeKey(tree)] = comments
		return set
	}

	for _, c := range comments {
		extendCommentRange(c, tokens)
	}

	a := &jsCommentAttacher{comments: comments, set: set}
	(&jsTraverser{enter: a.enterLeading}).visit(tree)
	a.cursor = 0
	(&jsTraverser{leave: a.leaveTrailing}).visit(tree)
	return set
}

// jsCommentAttacher carries the shrinking comment list and cursor across the two
// attachment passes.
type jsCommentAttacher struct {
	comments []*jsComment
	cursor   int
	set      *jsCommentSet
}

func (a *jsCommentAttacher) detach(cursor int) *jsComment {
	c := a.comments[cursor]
	a.comments = append(a.comments[:cursor], a.comments[cursor+1:]...)
	return c
}

// enterLeading attaches, pre-order, each comment whose extended range ends where
// the node begins.
func (a *jsCommentAttacher) enterLeading(node jsonval.Object) int {
	start, end, _ := jsNodeRange(node)
	for a.cursor < len(a.comments) {
		c := a.comments[a.cursor]
		if c.extHi > start {
			break
		}
		if c.extHi == start {
			key := jsNodeKey(node)
			a.set.leading[key] = append(a.set.leading[key], a.detach(a.cursor))
		} else {
			a.cursor++
		}
	}
	return a.travState(end)
}

// leaveTrailing attaches, post-order, each comment whose extended range begins
// where the node ends.
func (a *jsCommentAttacher) leaveTrailing(node jsonval.Object) int {
	_, end, _ := jsNodeRange(node)
	for a.cursor < len(a.comments) {
		c := a.comments[a.cursor]
		if end < c.extLo {
			break
		}
		if end == c.extLo {
			key := jsNodeKey(node)
			a.set.trailing[key] = append(a.set.trailing[key], a.detach(a.cursor))
		} else {
			a.cursor++
		}
	}
	return a.travState(end)
}

// travState returns Break when all comments are attached, or Skip when the next
// comment lies entirely past the current node's end.
func (a *jsCommentAttacher) travState(end int) int {
	if a.cursor == len(a.comments) {
		return jsTravBreak
	}
	if a.comments[a.cursor].extLo > end {
		return jsTravSkip
	}
	return jsTravContinue
}

// jsVisitorKeys is estraverse's VisitorKeys: the child properties of each node
// type, in traversal order.
var jsVisitorKeys = map[string][]string{
	"AssignmentExpression":     {"left", "right"},
	"AssignmentPattern":        {"left", "right"},
	"ArrayExpression":          {"elements"},
	"ArrayPattern":             {"elements"},
	"ArrowFunctionExpression":  {"params", "body"},
	"AwaitExpression":          {"argument"},
	"BlockStatement":           {"body"},
	"BinaryExpression":         {"left", "right"},
	"BreakStatement":           {"label"},
	"CallExpression":           {"callee", "arguments"},
	"CatchClause":              {"param", "body"},
	"ClassBody":                {"body"},
	"ClassDeclaration":         {"id", "superClass", "body"},
	"ClassExpression":          {"id", "superClass", "body"},
	"ConditionalExpression":    {"test", "consequent", "alternate"},
	"ContinueStatement":        {"label"},
	"DebuggerStatement":        {},
	"DoWhileStatement":         {"body", "test"},
	"EmptyStatement":           {},
	"ExpressionStatement":      {"expression"},
	"ForStatement":             {"init", "test", "update", "body"},
	"ForInStatement":           {"left", "right", "body"},
	"ForOfStatement":           {"left", "right", "body"},
	"FunctionDeclaration":      {"id", "params", "body"},
	"FunctionExpression":       {"id", "params", "body"},
	"Identifier":               {},
	"IfStatement":              {"test", "consequent", "alternate"},
	"Literal":                  {},
	"LabeledStatement":         {"label", "body"},
	"LogicalExpression":        {"left", "right"},
	"MemberExpression":         {"object", "property"},
	"MetaProperty":             {"meta", "property"},
	"MethodDefinition":         {"key", "value"},
	"NewExpression":            {"callee", "arguments"},
	"ObjectExpression":         {"properties"},
	"ObjectPattern":            {"properties"},
	"Program":                  {"body"},
	"Property":                 {"key", "value"},
	"RestElement":              {"argument"},
	"ReturnStatement":          {"argument"},
	"SequenceExpression":       {"expressions"},
	"SpreadElement":            {"argument"},
	"Super":                    {},
	"SwitchStatement":          {"discriminant", "cases"},
	"SwitchCase":               {"test", "consequent"},
	"TaggedTemplateExpression": {"tag", "quasi"},
	"TemplateElement":          {},
	"TemplateLiteral":          {"quasis", "expressions"},
	"ThisExpression":           {},
	"ThrowStatement":           {"argument"},
	"TryStatement":             {"block", "handler", "finalizer"},
	"UnaryExpression":          {"argument"},
	"UpdateExpression":         {"argument"},
	"VariableDeclaration":      {"declarations"},
	"VariableDeclarator":       {"id", "init"},
	"WhileStatement":           {"test", "body"},
	"WithStatement":            {"object", "body"},
	"YieldExpression":          {"argument"},
}
