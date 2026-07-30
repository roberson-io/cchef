package yara

import (
	"fmt"
	"strings"
)

// Rules is a checked set of rules, ready to be run over some data.
type Rules struct {
	Rules    []*Rule
	Imports  []string
	Warnings []Warning
	// patterns holds, for each rule, what to look for for each of its strings.
	// A string may come to more than one pattern, so each is a list.
	patterns []map[string][]*pattern
}

// modules are the ones this package can run.
var modules = map[string]bool{
	"hash": true, "math": true, "console": true, "time": true,
	"elf": true, "pe": true, "dotnet": true,
}

// Compile checks a parsed rule set over: that every string a condition names
// was declared, that every string declared is named, that no two rules share a
// name, and that the modules asked for are ones that can be run.
func Compile(set *RuleSet) (*Rules, error) {
	if err := checkImports(set); err != nil {
		return nil, err
	}

	rules := &Rules{Rules: set.Rules, Imports: set.Imports}
	imported := make(map[string]bool, len(set.Imports))
	for _, name := range set.Imports {
		imported[name] = true
	}

	seen := make(map[string]bool, len(set.Rules))
	for _, rule := range set.Rules {
		if seen[rule.Name] {
			return nil, nameError(rule, "duplicated identifier %q", rule.Name)
		}
		seen[rule.Name] = true

		if err := checkRule(rule, seen, imported); err != nil {
			return nil, err
		}
		compiled, err := compileRuleStrings(rule, rules)
		if err != nil {
			return nil, err
		}
		rules.patterns = append(rules.patterns, compiled)
	}
	return rules, nil
}

// compileRuleStrings turns a rule's strings into what to look for, noting any
// that would slow a scan down.
func compileRuleStrings(rule *Rule, rules *Rules) (map[string][]*pattern, error) {
	out := make(map[string][]*pattern, len(rule.Strings))
	for _, str := range rule.Strings {
		patterns, err := compileStrings(str)
		if err != nil {
			return nil, err
		}
		out[str.ID] = patterns

		if str.Kind == stringRegex && unboundedDot(str.Text) {
			rules.Warnings = append(rules.Warnings, Warning{
				Line: str.Line,
				Message: str.ID + " contains .*, .+ or .{x,} consider using " +
					".{,N}, .{1,N} or {x,N} with a reasonable value for N",
			})
		}
		if slowsScanning(str) {
			rules.Warnings = append(rules.Warnings, Warning{
				Line:    str.Line,
				Message: fmt.Sprintf("string %q may slow down scanning", str.ID),
			})
		}
	}
	return out, nil
}

// checkImports refuses a module this package does not have. Every module in
// CyberChef's own build is here, so the rest of YARA's — macho, dex, lnk, magic
// and cuckoo among them — are refused as naming nothing at all, which is what
// CyberChef does with them too.
func checkImports(set *RuleSet) error {
	for _, name := range set.Imports {
		if !modules[name] {
			return &compileError{line: 1, msg: fmt.Sprintf("unknown module %q", name)}
		}
	}
	return nil
}

// ruleError reports a fault against the line a rule closes on, which is where
// libyara reports whatever it finds while finishing one.
func ruleError(rule *Rule, format string, args ...any) error {
	return &compileError{line: rule.EndLine, msg: fmt.Sprintf(format, args...)}
}

// nameError reports a fault against the line a rule opens on, which is where
// libyara reports a name it has already seen.
func nameError(rule *Rule, format string, args ...any) error {
	return &compileError{line: rule.Line, msg: fmt.Sprintf(format, args...)}
}

// checkRule looks over one rule's condition. seen holds the rules declared so
// far, which is what a condition may refer to.
func checkRule(rule *Rule, seen, imported map[string]bool) error {
	declared := make(map[string]bool, len(rule.Strings))
	for _, str := range rule.Strings {
		declared[str.ID] = true
	}

	referenced := make(map[string]bool, len(rule.Strings))
	var bad error
	walk(rule.Condition, func(e Expr) {
		if bad != nil {
			return
		}
		bad = noteReference(rule, e, declared, referenced, seen, imported)
	})
	if bad != nil {
		return bad
	}

	for _, str := range rule.Strings {
		if !referenced[str.ID] {
			return ruleError(rule, "unreferenced string %q", str.ID)
		}
	}
	return nil
}

// noteReference records what one piece of a condition refers to, and reports
// anything it names that is not there.
func noteReference(rule *Rule, e Expr, declared, referenced, seen, imported map[string]bool) error {
	switch n := e.(type) {
	case StringRef:
		return useString(rule, n.ID, declared, referenced)
	case StringCount:
		return useString(rule, n.ID, declared, referenced)
	case StringOffset:
		return useString(rule, n.ID, declared, referenced)
	case StringLengthOf:
		return useString(rule, n.ID, declared, referenced)
	case StringAt:
		return useString(rule, n.ID, declared, referenced)
	case StringIn:
		return useString(rule, n.ID, declared, referenced)
	case Of:
		return useSet(rule, n.Set, declared, referenced)
	case ForOf:
		return useSet(rule, n.Set, declared, referenced)
	case Ident:
		return useIdent(rule, string(n), seen)
	case ModuleRef:
		return useModule(rule, n, imported)
	}
	return nil
}

// useString marks a string as referred to, and reports one that was never
// declared. A bare sigil stands for whichever string a loop is looking at, so
// it names nothing in particular.
func useString(rule *Rule, id string, declared, referenced map[string]bool) error {
	if id == "$" {
		return nil
	}
	if !declared[id] {
		return ruleError(rule, "undefined string %q", id)
	}
	referenced[id] = true
	return nil
}

// useSet marks every string a set covers. A set never names a string that is
// not there: "them" takes whatever the rule has, and a wildcard takes whatever
// it happens to match.
func useSet(rule *Rule, set StringSet, declared, referenced map[string]bool) error {
	if set.Them {
		for id := range declared {
			referenced[id] = true
		}
		return nil
	}
	for _, item := range set.Items {
		if prefix, ok := strings.CutSuffix(item, "*"); ok {
			for id := range declared {
				if strings.HasPrefix(id, prefix) {
					referenced[id] = true
				}
			}
			continue
		}
		if err := useString(rule, item, declared, referenced); err != nil {
			return err
		}
	}
	return nil
}

// useIdent reports a name that is neither a rule already declared nor the
// variable of a loop.
func useIdent(rule *Rule, name string, seen map[string]bool) error {
	if seen[name] || loopVariables(rule)[name] {
		return nil
	}
	return ruleError(rule, "undefined identifier %q", name)
}

// useModule reports a module used without being imported.
func useModule(rule *Rule, ref ModuleRef, imported map[string]bool) error {
	if !imported[ref.Module] {
		return ruleError(rule, "undefined identifier %q", ref.Module)
	}
	return checkModuleRef(rule, ref)
}

// loopVariables collects the names a rule's loops introduce, which its
// conditions may then use.
func loopVariables(rule *Rule) map[string]bool {
	out := map[string]bool{}
	walk(rule.Condition, func(e Expr) {
		if loop, ok := e.(ForRange); ok {
			out[loop.Var] = true
		}
	})
	return out
}

// walk calls fn for a condition and everything inside it.
func walk(e Expr, fn func(Expr)) {
	if e == nil {
		return
	}
	fn(e)
	for _, child := range children(e) {
		walk(child, fn)
	}
}

// children returns the conditions held inside one.
func children(e Expr) []Expr {
	switch n := e.(type) {
	case Not:
		return []Expr{n.X}
	case Defined:
		return []Expr{n.X}
	case Unary:
		return []Expr{n.X}
	case IntFunc:
		return []Expr{n.X}
	case Binary:
		return []Expr{n.L, n.R}
	case StringOffset:
		return []Expr{n.Index}
	case StringLengthOf:
		return []Expr{n.Index}
	case StringAt:
		return []Expr{n.Offset}
	case StringIn:
		return []Expr{n.From, n.To}
	case Of:
		return quantifierChildren(n.Quantifier)
	case ForOf:
		return append(quantifierChildren(n.Quantifier), n.Body)
	case ForRange:
		return append(quantifierChildren(n.Quantifier), n.From, n.To, n.Body)
	case ModuleRef:
		return moduleChildren(n)
	}
	return nil
}

// quantifierChildren returns the count a quantifier was given, if it was one.
func quantifierChildren(q Quantifier) []Expr {
	if q.Kind == "count" {
		return []Expr{q.Count}
	}
	return nil
}
