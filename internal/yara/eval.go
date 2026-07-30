package yara

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// Working out whether a rule's condition holds.
//
// Every piece of a condition comes to a value, which may be a number, a
// fraction, text, a truth, or nothing at all. Nothing at all is how YARA says a
// question had no answer — the offset of a match that was never made, say — and
// it spreads: anything worked out from it is also nothing. A condition that
// comes to nothing does not hold.

// valueKind says what sort of value a piece of a condition came to.
type valueKind int

const (
	valueUndefined valueKind = iota
	valueInt
	valueFloat
	valueString
	valueBool
	// valueRegex is a pattern handed to a module, which is the only place a
	// condition may pass one around rather than test against it.
	valueRegex
)

// value is what a piece of a condition came to.
type value struct {
	kind valueKind
	i    int64
	f    float64
	s    string
	b    bool
	re   *regexp.Regexp
}

// The values a condition can come to.
var (
	undefined = value{kind: valueUndefined}
	yes       = value{kind: valueBool, b: true}
	no        = value{kind: valueBool, b: false}
)

func intValue(i int64) value     { return value{kind: valueInt, i: i} }
func floatValue(f float64) value { return value{kind: valueFloat, f: f} }
func stringValue(s string) value { return value{kind: valueString, s: s} }
func boolValue(b bool) value {
	if b {
		return yes
	}
	return no
}

// truth reads a value as a condition would: a number is true when it is not
// zero, text when it is not empty, and nothing at all is never true.
func (v value) truth() bool {
	switch v.kind {
	case valueBool:
		return v.b
	case valueInt:
		return v.i != 0
	case valueFloat:
		return v.f != 0
	case valueString:
		return v.s != ""
	}
	return false
}

// number reads a value as a fraction, whichever sort of number it holds.
func (v value) number() (float64, bool) {
	switch v.kind {
	case valueInt:
		return float64(v.i), true
	case valueFloat:
		return v.f, true
	}
	return 0, false
}

// evaluator holds what one rule's condition is worked out against.
type evaluator struct {
	buf *buffer
	// matches holds where each of the rule's strings was found.
	matches map[string][]Match
	// matched says which rules have already held, for a condition that names
	// one.
	matched map[string]bool
	// vars holds the numbers a loop is walking.
	vars map[string]int64
	// modules holds what each module offers over this data, worked out the
	// first time a condition asks for it.
	modules map[string]modValue
	// current is the string a loop over a set is looking at.
	current string
	// declared lists the rule's strings, in the order it wrote them.
	declared []string
	// logs collects what the console module was asked to say.
	logs *[]string
}

// eval works out what a piece of a condition comes to.
func (e *evaluator) eval(expr Expr) (value, error) {
	if v, ok := e.evalLiteral(expr); ok {
		return v, nil
	}
	switch n := expr.(type) {
	case Not:
		return e.evalNot(n)
	case Defined:
		return e.evalDefined(n)
	case Unary:
		return e.evalUnary(n)
	case Binary:
		return e.evalBinary(n)
	case IntFunc:
		return e.evalIntFunc(n)
	case Of:
		return e.evalOf(n)
	case ForRange:
		return e.evalForRange(n)
	case ForOf:
		return e.evalForOf(n)
	case ModuleRef:
		return e.evalModuleRef(n)
	}
	return e.evalStringExpr(expr)
}

// evalLiteral works out the pieces of a condition that stand for themselves,
// and says whether the piece was one of them.
func (e *evaluator) evalLiteral(expr Expr) (value, bool) {
	switch n := expr.(type) {
	case BoolLit:
		return boolValue(bool(n)), true
	case IntLit:
		return intValue(int64(n)), true
	case DoubleLit:
		return floatValue(float64(n)), true
	case StringLit:
		return stringValue(string(n)), true
	case FileSize:
		return intValue(int64(len(e.buf.data))), true
	case EntryPoint:
		// Only an executable has one, and nothing here reads executables.
		return undefined, true
	case Ident:
		return e.evalIdent(string(n)), true
	}
	return undefined, false
}

// evalIdent reads a name: the variable of a loop, or another rule.
func (e *evaluator) evalIdent(name string) value {
	if n, ok := e.vars[name]; ok {
		return intValue(n)
	}
	if held, ok := e.matched[name]; ok {
		return boolValue(held)
	}
	return undefined
}

// evalNot inverts what follows it. Nothing at all stays nothing.
func (e *evaluator) evalNot(n Not) (value, error) {
	inner, err := e.eval(n.X)
	if err != nil || inner.kind == valueUndefined {
		return undefined, err
	}
	return boolValue(!inner.truth()), nil
}

// evalDefined asks only whether something has an answer.
func (e *evaluator) evalDefined(n Defined) (value, error) {
	inner, err := e.eval(n.X)
	if err != nil {
		return undefined, err
	}
	return boolValue(inner.kind != valueUndefined), nil
}

// evalUnary applies a sign or a bitwise complement.
func (e *evaluator) evalUnary(n Unary) (value, error) {
	inner, err := e.eval(n.X)
	if err != nil || inner.kind == valueUndefined {
		return undefined, err
	}
	if n.Op == "~" {
		if inner.kind != valueInt {
			return undefined, nil
		}
		return intValue(^inner.i), nil
	}
	if inner.kind == valueFloat {
		return floatValue(-inner.f), nil
	}
	if inner.kind != valueInt {
		return undefined, nil
	}
	return intValue(-inner.i), nil
}

// evalIntFunc reads a number out of the data at a given place. Reading past the
// end has no answer.
func (e *evaluator) evalIntFunc(n IntFunc) (value, error) {
	at, err := e.eval(n.X)
	if err != nil || at.kind != valueInt {
		return undefined, err
	}
	width, signed, big := intFuncShape(n.Name)
	offset := at.i
	if offset < 0 || offset+int64(width) > int64(len(e.buf.data)) {
		return undefined, nil
	}

	raw := uint64(0)
	for i := range width {
		b := uint64(e.buf.data[offset+int64(i)])
		if big {
			raw = raw<<8 | b
		} else {
			raw |= b << (8 * i)
		}
	}
	if !signed {
		return intValue(int64(raw)), nil // #nosec G115 -- at most four bytes
	}
	// Sign-extend from the width that was read.
	shift := 64 - 8*width
	return intValue(int64(raw<<shift) >> shift), nil // #nosec G115 -- at most four bytes
}

// intFuncShape reads a function's name as how many bytes it takes, whether the
// number is signed, and which end it starts at.
func intFuncShape(name string) (width int, signed, big bool) {
	signed = !strings.HasPrefix(name, "u")
	big = strings.HasSuffix(name, "be")
	digits := strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(name, "u"), "int"), "be")
	switch digits {
	case "8":
		return 1, signed, big
	case "16":
		return 2, signed, big
	}
	return 4, signed, big
}

// evalBinary applies an operator written between two operands.
func (e *evaluator) evalBinary(n Binary) (value, error) {
	// The two ways of combining conditions stop as soon as the answer is
	// settled, which is how a rule avoids asking a question it need not.
	if n.Op == "and" || n.Op == "or" {
		return e.evalLogical(n)
	}
	if n.Op == "matches" {
		return e.evalMatches(n)
	}

	left, err := e.eval(n.L)
	if err != nil {
		return undefined, err
	}
	right, err := e.eval(n.R)
	if err != nil {
		return undefined, err
	}
	if left.kind == valueUndefined || right.kind == valueUndefined {
		return undefined, nil
	}
	if left.kind == valueString || right.kind == valueString {
		return evalTextOp(n.Op, left, right), nil
	}
	return evalNumberOp(n.Op, left, right), nil
}

// evalMatches asks whether text answers to a regular expression. The right side
// is the expression itself rather than a value, so it is not worked out first.
func (e *evaluator) evalMatches(n Binary) (value, error) {
	left, err := e.eval(n.L)
	if err != nil {
		return undefined, err
	}
	re, ok := n.R.(RegexLit)
	if !ok || left.kind != valueString {
		return undefined, nil
	}
	compiled, err := compileRegexLit(re)
	if err != nil {
		return undefined, fmt.Errorf("the regular expression /%s/ cannot be read", re.Body)
	}
	return boolValue(compiled.MatchString(latin1([]byte(left.s)))), nil
}

// evalLogical combines two conditions, stopping early where it can.
func (e *evaluator) evalLogical(n Binary) (value, error) {
	left, err := e.eval(n.L)
	if err != nil {
		return undefined, err
	}
	if n.Op == "and" && !left.truth() {
		return no, nil
	}
	if n.Op == "or" && left.truth() {
		return yes, nil
	}
	right, err := e.eval(n.R)
	if err != nil {
		return undefined, err
	}
	return boolValue(right.truth()), nil
}

// evalTextOp applies an operator to text.
func evalTextOp(op string, left, right value) value {
	if left.kind != valueString || right.kind != valueString {
		return undefined
	}
	a, b := left.s, right.s
	switch op {
	case "==":
		return boolValue(a == b)
	case "!=":
		return boolValue(a != b)
	case "contains":
		return boolValue(strings.Contains(a, b))
	case "icontains":
		return boolValue(strings.Contains(strings.ToLower(a), strings.ToLower(b)))
	case "startswith":
		return boolValue(strings.HasPrefix(a, b))
	case "istartswith":
		return boolValue(strings.HasPrefix(strings.ToLower(a), strings.ToLower(b)))
	case "endswith":
		return boolValue(strings.HasSuffix(a, b))
	case "iendswith":
		return boolValue(strings.HasSuffix(strings.ToLower(a), strings.ToLower(b)))
	case "iequals":
		return boolValue(strings.EqualFold(a, b))
	}
	return undefined
}

// evalNumberOp applies an operator to numbers. Where either side is a fraction
// the sum is worked out as one; where both are whole, so is the answer.
func evalNumberOp(op string, left, right value) value {
	if left.kind == valueFloat || right.kind == valueFloat {
		return evalFloatOp(op, left, right)
	}
	if left.kind != valueInt || right.kind != valueInt {
		return evalBoolOp(op, left, right)
	}
	return evalIntOp(op, left.i, right.i)
}

// evalBoolOp compares two truths, which is all that can be done with them.
func evalBoolOp(op string, left, right value) value {
	switch op {
	case "==":
		return boolValue(left.truth() == right.truth())
	case "!=":
		return boolValue(left.truth() != right.truth())
	}
	return undefined
}

// evalFloatOp works an operator out over fractions.
func evalFloatOp(op string, left, right value) value {
	a, aok := left.number()
	b, bok := right.number()
	if !aok || !bok {
		return undefined
	}
	switch op {
	case "+":
		return floatValue(a + b)
	case "-":
		return floatValue(a - b)
	case "*":
		return floatValue(a * b)
	case `\`:
		if b == 0 {
			return undefined
		}
		return floatValue(a / b)
	}
	return compareOrdered(op, a, b)
}

// evalIntOp works an operator out over whole numbers.
func evalIntOp(op string, a, b int64) value {
	switch op {
	case "+":
		return intValue(a + b)
	case "-":
		return intValue(a - b)
	case "*":
		return intValue(a * b)
	case `\`, "%":
		if b == 0 {
			return undefined
		}
		if op == "%" {
			return intValue(a % b)
		}
		return intValue(a / b)
	case "&":
		return intValue(a & b)
	case "|":
		return intValue(a | b)
	case "^":
		return intValue(a ^ b)
	case "<<":
		return shift(a, b, true)
	case ">>":
		return shift(a, b, false)
	}
	return compareOrdered(op, float64(a), float64(b))
}

// maxShift is as far as a number may be moved before the answer is nothing.
const maxShift = 64

// shift moves a number's bits, which has no answer if it is asked to move them
// further than there are.
func shift(a, by int64, left bool) value {
	if by < 0 || by >= maxShift {
		return undefined
	}
	if left {
		return intValue(a << uint(by))
	}
	return intValue(a >> uint(by))
}

// compareOrdered applies the operators that put two numbers in order.
func compareOrdered(op string, a, b float64) value {
	switch op {
	case "==":
		return boolValue(a == b)
	case "!=":
		return boolValue(a != b)
	case "<":
		return boolValue(a < b)
	case "<=":
		return boolValue(a <= b)
	case ">":
		return boolValue(a > b)
	case ">=":
		return boolValue(a >= b)
	}
	return undefined
}

// evalStringExpr works out what a rule is asking about one of its strings.
func (e *evaluator) evalStringExpr(expr Expr) (value, error) {
	switch n := expr.(type) {
	case StringRef:
		return boolValue(len(e.matchesOf(n.ID)) > 0), nil
	case StringCount:
		return intValue(int64(len(e.matchesOf(n.ID)))), nil
	case StringOffset:
		return e.nthMatch(n.ID, n.Index, func(m Match) int64 { return int64(m.Offset) })
	case StringLengthOf:
		return e.nthMatch(n.ID, n.Index, func(m Match) int64 { return int64(m.Length) })
	case StringAt:
		return e.evalStringAt(n)
	case StringIn:
		return e.evalStringIn(n)
	}
	return undefined, fmt.Errorf("cannot work out %s", expr)
}

// matchesOf gives where a string was found. A bare sigil means whichever string
// a loop is looking at.
func (e *evaluator) matchesOf(id string) []Match {
	if id == "$" {
		return e.matches[e.current]
	}
	return e.matches[id]
}

// nthMatch reads something out of one match, counting from one. Asking for a
// match that was never made has no answer.
func (e *evaluator) nthMatch(id string, index Expr, of func(Match) int64) (value, error) {
	at, err := e.eval(index)
	if err != nil {
		return undefined, err
	}
	matches := e.matchesOf(id)
	if at.kind != valueInt || at.i < 1 || at.i > int64(len(matches)) {
		return undefined, nil
	}
	return intValue(of(matches[at.i-1])), nil
}

// evalStringAt asks whether a string was found at one place.
func (e *evaluator) evalStringAt(n StringAt) (value, error) {
	at, err := e.eval(n.Offset)
	if err != nil || at.kind != valueInt {
		return undefined, err
	}
	for _, m := range e.matchesOf(n.ID) {
		if int64(m.Offset) == at.i {
			return yes, nil
		}
	}
	return no, nil
}

// evalStringIn asks whether a string was found anywhere in a stretch.
func (e *evaluator) evalStringIn(n StringIn) (value, error) {
	from, err := e.eval(n.From)
	if err != nil {
		return undefined, err
	}
	to, err := e.eval(n.To)
	if err != nil {
		return undefined, err
	}
	if from.kind != valueInt || to.kind != valueInt {
		return undefined, nil
	}
	for _, m := range e.matchesOf(n.ID) {
		if int64(m.Offset) >= from.i && int64(m.Offset) <= to.i {
			return yes, nil
		}
	}
	return no, nil
}

// evalOf asks how many of a set of strings were found.
func (e *evaluator) evalOf(n Of) (value, error) {
	ids := e.setMembers(n.Set)
	held := 0
	for _, id := range ids {
		if len(e.matches[id]) > 0 {
			held++
		}
	}
	return e.quantifierHolds(n.Quantifier, held, len(ids))
}

// setMembers gives the strings a set covers, in the order the rule declared
// them.
func (e *evaluator) setMembers(set StringSet) []string {
	if set.Them {
		return e.declared
	}
	var out []string
	for _, item := range set.Items {
		if prefix, ok := strings.CutSuffix(item, "*"); ok {
			for _, id := range e.declared {
				if strings.HasPrefix(id, prefix) {
					out = append(out, id)
				}
			}
			continue
		}
		out = append(out, item)
	}
	return out
}

// quantifierHolds says whether enough of a set held.
func (e *evaluator) quantifierHolds(q Quantifier, held, total int) (value, error) {
	switch q.Kind {
	case "any":
		return boolValue(held > 0), nil
	case "all":
		return boolValue(held == total), nil
	case "none":
		return boolValue(held == 0), nil
	}
	// A count is only ever written as a whole number, which is all the parser
	// will take here.
	count, _ := q.Count.(IntLit)
	return boolValue(int64(held) >= int64(count)), nil
}

// evalForRange walks a stretch of numbers, holding the loop's variable at each.
func (e *evaluator) evalForRange(n ForRange) (value, error) {
	from, err := e.eval(n.From)
	if err != nil {
		return undefined, err
	}
	to, err := e.eval(n.To)
	if err != nil {
		return undefined, err
	}
	if from.kind != valueInt || to.kind != valueInt {
		return undefined, nil
	}

	// A range with nothing in it does not hold, whatever was asked of it, so
	// even asking for none of nothing comes to no.
	if to.i < from.i {
		return no, nil
	}

	held, total := 0, 0
	was, had := e.vars[n.Var], true
	if _, ok := e.vars[n.Var]; !ok {
		had = false
	}
	for i := from.i; i <= to.i; i++ {
		e.vars[n.Var] = i
		total++
		body, err := e.eval(n.Body)
		if err != nil {
			return undefined, err
		}
		if body.truth() {
			held++
		}
	}
	if had {
		e.vars[n.Var] = was
	} else {
		delete(e.vars, n.Var)
	}
	return e.quantifierHolds(n.Quantifier, held, total)
}

// evalForOf walks a set of strings, holding each in turn as the one a bare
// sigil refers to.
func (e *evaluator) evalForOf(n ForOf) (value, error) {
	ids := e.setMembers(n.Set)
	held := 0
	was := e.current
	for _, id := range ids {
		e.current = id
		body, err := e.eval(n.Body)
		if err != nil {
			e.current = was
			return undefined, err
		}
		if body.truth() {
			held++
		}
	}
	e.current = was
	return e.quantifierHolds(n.Quantifier, held, len(ids))
}

// MatchedString is one place one of a rule's strings was found.
type MatchedString struct {
	ID string
	Match
}

// Result is a rule that held, with everything its strings matched.
type Result struct {
	Rule    *Rule
	Matches []MatchedString
}

// Scan runs the rules over some data and returns those that held, in the order
// they were written. A private rule is worked out, so that another may refer to
// it, but is not reported.
func (r *Rules) Scan(data []byte) ([]Result, []string, error) {
	buf := newBuffer(data)
	matched := make(map[string]bool, len(r.Rules))
	var logs []string
	var out []Result

	for i, rule := range r.Rules {
		found := r.findStrings(rule, i, buf)
		e := &evaluator{
			buf: buf, matches: found, matched: matched,
			vars: map[string]int64{}, declared: declaredIDs(rule), logs: &logs,
		}
		held, err := e.eval(rule.Condition)
		if err != nil {
			return nil, nil, err
		}
		matched[rule.Name] = held.truth()
		if rule.Global && !held.truth() {
			// A global rule that does not hold rules the whole set out.
			return nil, logs, nil
		}
		if !held.truth() || rule.Private {
			continue
		}
		out = append(out, Result{Rule: rule, Matches: orderedMatches(rule, found)})
	}
	return out, logs, nil
}

// findStrings looks for every one of a rule's strings.
func (r *Rules) findStrings(rule *Rule, at int, buf *buffer) map[string][]Match {
	fixed := fixedOffsets(rule)
	found := make(map[string][]Match, len(rule.Strings))
	for _, str := range rule.Strings {
		for _, p := range r.patterns[at][str.ID] {
			found[str.ID] = append(found[str.ID], p.findAll(buf)...)
		}
		// A string looked for in several forms at once is found form by form,
		// but is reported in the order the matches appear in the data, and only
		// once for each place: where two forms both match somewhere, the first
		// form looked for is the one kept.
		matches := found[str.ID]
		slices.SortStableFunc(matches, func(a, b Match) int {
			return a.Offset - b.Offset
		})
		matches = onePerPlace(matches)
		found[str.ID] = matches
		if only, there := fixed[str.ID]; there && sitsInOnePlace(str) {
			found[str.ID] = matchesAt(matches, only)
		}
	}
	return found
}

// sitsInOnePlace says whether tying a string to one place narrows it down to
// the single match beginning there.
//
// YARA looks a string up by a run of bytes taken out of it, and only searches
// where that run was found. For a string that is one plain run of bytes, that
// run and the match begin together, so naming a place leaves at most one match.
// A string that can match in more than one way — a pattern that stretches or
// offers a choice, a run of bytes with anything wild in it, or one asked for as
// base64, which is looked for at three shifts — is looked for throughout even
// when a place is named, and every match it makes is kept.
func sitsInOnePlace(str *String) bool {
	if str.Mods.Base64 || str.Mods.Base64Wide {
		return false
	}
	switch str.Kind {
	case stringText:
		return true
	case stringHex:
		// Anything but plain bytes — a wild half-byte, a jump, or a choice —
		// makes the run more than one thing.
		return !strings.ContainsAny(str.Text, "?[(")
	default:
		return !strings.ContainsAny(str.Text, `\.*+?()[]{}|^$`)
	}
}

// onePerPlace keeps the first match beginning at each place and drops the rest,
// which is what YARA does when a string is looked for in more than one form and
// two of them match somewhere together.
func onePerPlace(matches []Match) []Match {
	out := matches[:0]
	taken := -1
	for _, m := range matches {
		if m.Offset == taken {
			continue
		}
		taken = m.Offset
		out = append(out, m)
	}
	return out
}

// matchesAt keeps only what was found in one place.
func matchesAt(matches []Match, at int64) []Match {
	var out []Match
	for _, m := range matches {
		if int64(m.Offset) == at {
			out = append(out, m)
		}
	}
	return out
}

// fixedOffsets works out which of a rule's strings are only ever asked about at
// one settled place. libyara looks for such a string there and nowhere else, so
// it is the only place it can be reported from. A string named any other way,
// or at more than one place, is looked for throughout.
func fixedOffsets(rule *Rule) map[string]int64 {
	only := map[string]int64{}
	elsewhere := map[string]bool{}
	walk(rule.Condition, func(e Expr) {
		at, isAt := e.(StringAt)
		if !isAt {
			noteStringUse(rule, e, elsewhere)
			return
		}
		place, settled := settledNumber(at.Offset)
		if !settled {
			elsewhere[at.ID] = true
			return
		}
		if was, seen := only[at.ID]; seen && was != place {
			elsewhere[at.ID] = true
			return
		}
		only[at.ID] = place
	})
	for id := range elsewhere {
		delete(only, id)
	}
	return only
}

// noteStringUse marks a string named in a way that says nothing about where it
// is, which means it has to be looked for throughout.
func noteStringUse(rule *Rule, e Expr, elsewhere map[string]bool) {
	switch n := e.(type) {
	case StringRef:
		elsewhere[n.ID] = true
	case StringCount:
		elsewhere[n.ID] = true
	case StringOffset:
		elsewhere[n.ID] = true
	case StringLengthOf:
		elsewhere[n.ID] = true
	case StringIn:
		elsewhere[n.ID] = true
	case Of:
		markSet(rule, n.Set, elsewhere)
	case ForOf:
		markSet(rule, n.Set, elsewhere)
	}
}

// markSet marks every string a set could cover, since a set says nothing about
// where any of them is.
func markSet(rule *Rule, set StringSet, elsewhere map[string]bool) {
	if set.Them {
		for _, str := range rule.Strings {
			elsewhere[str.ID] = true
		}
		return
	}
	for _, item := range set.Items {
		prefix, isWildcard := strings.CutSuffix(item, "*")
		if !isWildcard {
			elsewhere[item] = true
			continue
		}
		for _, str := range rule.Strings {
			if strings.HasPrefix(str.ID, prefix) {
				elsewhere[str.ID] = true
			}
		}
	}
}

// settledNumber is what a piece of a condition comes to when it can be worked
// out without running the rule, which is what libyara needs to look in one
// place only.
func settledNumber(e Expr) (int64, bool) {
	switch n := e.(type) {
	case IntLit:
		return int64(n), true
	case Unary:
		inner, ok := settledNumber(n.X)
		if !ok {
			return 0, false
		}
		switch n.Op {
		case "-":
			return -inner, true
		case "~":
			return ^inner, true
		}
	case Binary:
		return settledPair(n)
	}
	return 0, false
}

// settledPair works out a sum of two settled numbers.
func settledPair(n Binary) (int64, bool) {
	left, leftOK := settledNumber(n.L)
	right, rightOK := settledNumber(n.R)
	if !leftOK || !rightOK {
		return 0, false
	}
	switch n.Op {
	case "+":
		return left + right, true
	case "-":
		return left - right, true
	case "*":
		return left * right, true
	case "|":
		return left | right, true
	case "^":
		return left ^ right, true
	case "&":
		return left & right, true
	case "<<":
		return shifted(left, right, true)
	case ">>":
		return shifted(left, right, false)
	case "\\":
		if right == 0 {
			return 0, false
		}
		return left / right, true
	case "%":
		if right == 0 {
			return 0, false
		}
		return left % right, true
	}
	return 0, false
}

// shifted moves a number along by so many places, which cannot be worked out
// beforehand when it is asked to move further than a number goes.
func shifted(n, places int64, left bool) (int64, bool) {
	const bits = 64
	if places < 0 || places >= bits {
		return 0, false
	}
	if left {
		return n << places, true
	}
	return n >> places, true
}

// declaredIDs lists a rule's strings in the order it wrote them.
func declaredIDs(rule *Rule) []string {
	out := make([]string, 0, len(rule.Strings))
	for _, str := range rule.Strings {
		out = append(out, str.ID)
	}
	return out
}

// orderedMatches flattens what was found, keeping the strings in the order the
// rule declared them and each string's matches in the order they appear.
func orderedMatches(rule *Rule, found map[string][]Match) []MatchedString {
	var out []MatchedString
	for _, str := range rule.Strings {
		if str.Mods.Private {
			continue
		}
		for _, m := range found[str.ID] {
			out = append(out, MatchedString{ID: str.ID, Match: m})
		}
	}
	return out
}
