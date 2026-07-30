package yara

import "fmt"

// What a module says its names are, and what a rule may do with them.
//
// YARA checks a reference into a module while the rule is compiled, not while
// it runs: a name the module does not declare, a structure used where a value
// belongs, or a function given the wrong arguments are all refused before any
// data is looked at. The declarations here are what those checks are made
// against, and each module builds its values to the same shape.

// modKind is what a name a module declares stands for.
type modKind int

const (
	modInt modKind = iota
	modString
	modFloat
	modBool
	modRegex
	modStruct
	modArray
	modDict
	modFunc
)

// scalarKinds are the kinds a condition can use outright. Anything else has to
// be gone into further before it means something.
var scalarKinds = map[modKind]bool{
	modInt: true, modString: true, modFloat: true, modBool: true,
}

// modDecl is what a module says one of its names is. A structure names its
// members; a list or a table says what it holds; a function says what it takes
// and what it gives back.
type modDecl struct {
	kind    modKind
	members map[string]*modDecl
	item    *modDecl
	// takes holds the shapes a function accepts, one letter per argument: s for
	// text, i for a whole number, f for a fraction, r for a regular expression.
	// A function may accept more than one shape.
	takes []string
	gives modKind
}

// The shorthands each module's declarations are written with.
func decInt() *modDecl    { return &modDecl{kind: modInt} }
func decString() *modDecl { return &modDecl{kind: modString} }
func decFloat() *modDecl  { return &modDecl{kind: modFloat} }

func decStruct(members map[string]*modDecl) *modDecl {
	return &modDecl{kind: modStruct, members: members}
}

func decArray(item *modDecl) *modDecl { return &modDecl{kind: modArray, item: item} }

func decFunc(gives modKind, takes ...string) *modDecl {
	return &modDecl{kind: modFunc, gives: gives, takes: takes}
}

// checkModuleRef reports a reference to something a module does not offer,
// using the words libyara uses for each way of getting it wrong.
func checkModuleRef(rule *Rule, ref ModuleRef) error {
	decl := moduleSchemas[ref.Module]
	if decl == nil {
		return nil
	}
	for _, step := range ref.Steps {
		member, declared := decl.members[step.Name]
		if !declared {
			return ruleError(rule, "invalid field name %q", step.Name)
		}
		next, err := stepDecl(rule, step, member)
		if err != nil {
			return err
		}
		decl = next
	}
	if !scalarKinds[decl.kind] {
		last := ref.Steps[len(ref.Steps)-1].Name
		return ruleError(rule, "wrong usage of identifier %q", last)
	}
	return nil
}

// stepDecl works out what one step of a reference leaves behind: what a list or
// table holds, what a function gives back, or the name itself untouched.
func stepDecl(rule *Rule, step ModuleStep, decl *modDecl) (*modDecl, error) {
	switch {
	case step.Index != nil:
		if decl.kind != modArray && decl.kind != modDict {
			return nil, ruleError(rule, "%q is not an array or dictionary", step.Name)
		}
		if decl.kind == modDict && !mayBe(step.Index, modString) {
			return nil, ruleError(rule, "dictionary keys must be of string type")
		}
		if decl.kind == modArray && !mayBe(step.Index, modInt) {
			return nil, ruleError(rule, "array indexes must be of integer type")
		}
		return decl.item, nil
	case step.Call:
		if decl.kind != modFunc {
			return nil, ruleError(rule, "%q is not a function", step.Name)
		}
		if !accepts(decl.takes, step.Args) {
			return nil, ruleError(rule, "wrong arguments for function %q", step.Name)
		}
		return &modDecl{kind: decl.gives}, nil
	}
	return decl, nil
}

// accepts reports whether what a call was given fits any of the shapes the
// function takes.
func accepts(takes []string, args []Expr) bool {
	for _, shape := range takes {
		if len(shape) != len(args) {
			continue
		}
		fits := true
		for i, letter := range []byte(shape) {
			if !mayBe(args[i], argKinds[letter]) {
				fits = false
				break
			}
		}
		if fits {
			return true
		}
	}
	return false
}

// argKinds maps the letters a function's shape is written with onto kinds.
var argKinds = map[byte]modKind{
	's': modString, 'i': modInt, 'f': modFloat, 'r': modRegex, 'b': modBool,
}

// mayBe reports whether a piece of a condition could stand for a given kind. A
// piece whose kind cannot be told without running the rule is allowed through,
// so that a check never refuses what libyara would accept.
func mayBe(e Expr, want modKind) bool {
	kind, known := exprKind(e)
	if !known {
		return true
	}
	// A whole number stands in for a fraction wherever one is wanted.
	if want == modFloat && kind == modInt {
		return true
	}
	return kind == want
}

// exprKind works out what kind of value a piece of a condition stands for, as
// far as that can be told before the rule is run.
func exprKind(e Expr) (modKind, bool) {
	switch n := e.(type) {
	case StringLit:
		return modString, true
	case IntLit:
		return modInt, true
	case DoubleLit:
		return modFloat, true
	case RegexLit:
		return modRegex, true
	case BoolLit:
		return modBool, true
	case StringCount, StringOffset, StringLengthOf, IntFunc:
		return modInt, true
	case ModuleRef:
		return moduleRefKind(n)
	}
	return modInt, false
}

// moduleRefKind follows a reference through its module's declarations to what
// it ends up standing for.
func moduleRefKind(ref ModuleRef) (modKind, bool) {
	decl := moduleSchemas[ref.Module]
	if decl == nil {
		return modInt, false
	}
	for _, step := range ref.Steps {
		member, declared := decl.members[step.Name]
		if !declared {
			return modInt, false
		}
		switch {
		case step.Index != nil:
			if member.item == nil {
				return modInt, false
			}
			decl = member.item
		case step.Call:
			decl = &modDecl{kind: member.gives}
		default:
			decl = member
		}
	}
	return decl.kind, scalarKinds[decl.kind]
}

// moduleChildren gives the pieces of a condition a reference holds, so that a
// walk over a condition reaches what a place in a list, or an argument, was
// worked out from.
func moduleChildren(ref ModuleRef) []Expr {
	var out []Expr
	for _, step := range ref.Steps {
		if step.Index != nil {
			out = append(out, step.Index)
		}
		out = append(out, step.Args...)
	}
	return out
}

// modValue is a value inside a module while a scan is running: something a
// condition can use outright, a structure, a list, a table, or a function.
type modValue struct {
	value  value
	fields map[string]modValue
	list   []modValue
	table  map[string]modValue
	call   func(e *evaluator, args []value) (value, error)
}

// valueOf wraps something a condition can use outright.
func valueOf(v value) modValue { return modValue{value: v} }

// structOf gathers named members.
func structOf(fields map[string]modValue) modValue { return modValue{fields: fields} }

// listOf gathers a list, which is read by its place in it.
func listOf(items []modValue) modValue { return modValue{list: items} }

// funcOf wraps a function a rule may call.
func funcOf(fn func(e *evaluator, args []value) (value, error)) modValue {
	return modValue{call: fn}
}

// evalModuleRef follows a reference to the value it names. A step that leads
// nowhere — past the end of a list, or a key a table does not hold — comes to
// nothing rather than being a fault, which is how a rule asks whether something
// is there at all.
func (e *evaluator) evalModuleRef(ref ModuleRef) (value, error) {
	current, err := e.moduleRoot(ref.Module)
	if err != nil {
		return undefined, err
	}
	for _, step := range ref.Steps {
		next, there := current.fields[step.Name]
		if !there {
			return undefined, nil
		}
		if current, err = e.evalModuleStep(step, next); err != nil {
			return undefined, err
		}
		if current.isNothing() {
			return undefined, nil
		}
	}
	return current.value, nil
}

// isNothing reports a value that is not there at all.
func (m modValue) isNothing() bool {
	return m.fields == nil && m.list == nil && m.table == nil && m.call == nil &&
		m.value.kind == valueUndefined
}

// evalModuleStep takes one step of a reference: into a list or a table, or
// through a call.
func (e *evaluator) evalModuleStep(step ModuleStep, current modValue) (modValue, error) {
	switch {
	case step.Index != nil:
		return e.evalModuleIndex(step, current)
	case step.Call:
		args := make([]value, 0, len(step.Args))
		for _, arg := range step.Args {
			// A pattern is handed to a module as it was written, since a
			// condition cannot work one out into anything else.
			if written, isRegex := arg.(RegexLit); isRegex {
				re, err := compileRegexLit(written)
				if err != nil {
					return modValue{}, err
				}
				args = append(args, regexValue(re))
				continue
			}
			v, err := e.eval(arg)
			if err != nil {
				return modValue{}, err
			}
			args = append(args, v)
		}
		v, err := current.call(e, args)
		return valueOf(v), err
	}
	return current, nil
}

// evalModuleIndex picks one out of a list or a table.
func (e *evaluator) evalModuleIndex(step ModuleStep, current modValue) (modValue, error) {
	at, err := e.eval(step.Index)
	if err != nil {
		return modValue{}, err
	}
	if current.table != nil {
		if at.kind != valueString {
			return modValue{}, nil
		}
		return current.table[at.s], nil
	}
	if at.kind != valueInt || at.i < 0 || at.i >= int64(len(current.list)) {
		return modValue{}, nil
	}
	return current.list[at.i], nil
}

// moduleRoot builds what a module offers over the data being scanned, once per
// scan, since working a whole file format out is not something to repeat for
// every mention of it in a condition.
func (e *evaluator) moduleRoot(name string) (modValue, error) {
	if root, built := e.modules[name]; built {
		return root, nil
	}
	build, known := moduleBuilders[name]
	if !known {
		return modValue{}, fmt.Errorf("the %s module is not supported", name)
	}
	root := build(e)
	if e.modules == nil {
		e.modules = map[string]modValue{}
	}
	e.modules[name] = root
	return root, nil
}
