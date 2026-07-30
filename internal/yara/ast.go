package yara

import (
	"fmt"
	"strconv"
	"strings"
)

// stringKind says how a string in a rule was written.
type stringKind int

const (
	stringText stringKind = iota
	stringHex
	stringRegex
)

// String names a kind, for error messages.
func (k stringKind) String() string {
	switch k {
	case stringHex:
		return "hex"
	case stringRegex:
		return "regular expression"
	}
	return "text"
}

// RuleSet is a whole file of rules, with whatever modules it asked for.
type RuleSet struct {
	Imports []string
	Rules   []*Rule
}

// Rule is one rule.
type Rule struct {
	Name      string
	Tags      []string
	Global    bool
	Private   bool
	Meta      []Meta
	Strings   []*String
	Condition Expr
	// Line is where the rule opens, and EndLine where it closes. libyara
	// reports a duplicated name at the first and everything it finds while
	// finishing the rule at the second.
	Line    int
	EndLine int
}

// Meta is one entry of a rule's metadata. Value is a string, an int64 or a bool.
type Meta struct {
	Key   string
	Value any
}

// Modifiers are the words that may follow a string, changing what it matches.
type Modifiers struct {
	Nocase   bool
	Wide     bool
	ASCII    bool
	Fullword bool
	Private  bool

	XOR    bool
	XORMin int
	XORMax int

	Base64         bool
	Base64Wide     bool
	Base64Alphabet string
}

// String is one string a rule looks for.
type String struct {
	ID    string
	Kind  stringKind
	Text  string
	Flags string
	Mods  Modifiers
	Line  int
	// pattern is what a hex string was built into when it was read, since it
	// is checked there so that a fault is reported against the right line.
	pattern string
}

// Expr is a piece of a condition.
type Expr interface{ String() string }

// The values a condition can be written with outright.
type (
	// BoolLit is true or false.
	BoolLit bool
	// IntLit is a whole number.
	IntLit int64
	// DoubleLit is a number with a fractional part.
	DoubleLit float64
	// StringLit is quoted text, used with the text operators.
	StringLit string
	// RegexLit is a regular expression written into a condition.
	RegexLit struct{ Body, Flags string }
	// Ident names a rule, or the variable a loop is walking.
	Ident string
	// FileSize is how many bytes are being scanned.
	FileSize struct{}
	// EntryPoint is where an executable starts.
	EntryPoint struct{}
)

func (b BoolLit) String() string   { return strconv.FormatBool(bool(b)) }
func (i IntLit) String() string    { return strconv.FormatInt(int64(i), 10) }
func (d DoubleLit) String() string { return strconv.FormatFloat(float64(d), 'g', -1, 64) }
func (s StringLit) String() string { return strconv.Quote(string(s)) }
func (r RegexLit) String() string  { return "/" + r.Body + "/" + r.Flags }
func (i Ident) String() string     { return string(i) }
func (FileSize) String() string    { return "filesize" }
func (EntryPoint) String() string  { return "entrypoint" }

// The four things a rule can ask about one of its strings.
type (
	// StringRef is whether a string matched at all.
	StringRef struct{ ID string }
	// StringCount is how many times it did.
	StringCount struct{ ID string }
	// StringOffset is where the nth match began.
	StringOffset struct {
		ID    string
		Index Expr
	}
	// StringLengthOf is how long the nth match was.
	StringLengthOf struct {
		ID    string
		Index Expr
	}
	// StringAt is whether a string matched at one place.
	StringAt struct {
		ID     string
		Offset Expr
	}
	// StringIn is whether it matched anywhere in a stretch.
	StringIn struct {
		ID       string
		From, To Expr
	}
)

func (s StringRef) String() string   { return s.ID }
func (s StringCount) String() string { return "#" + strings.TrimPrefix(s.ID, "$") }
func (s StringOffset) String() string {
	return "@" + strings.TrimPrefix(s.ID, "$") + "[" + s.Index.String() + "]"
}

func (s StringLengthOf) String() string {
	return "!" + strings.TrimPrefix(s.ID, "$") + "[" + s.Index.String() + "]"
}
func (s StringAt) String() string { return "(at " + s.ID + " " + s.Offset.String() + ")" }
func (s StringIn) String() string {
	return "(in " + s.ID + " " + s.From.String() + " " + s.To.String() + ")"
}

// The ways conditions combine.
type (
	// Not inverts what follows it.
	Not struct{ X Expr }
	// Defined asks whether something has a value at all.
	Defined struct{ X Expr }
	// Unary is a sign or a bitwise complement.
	Unary struct {
		Op string
		X  Expr
	}
	// Binary is everything written between two operands.
	Binary struct {
		Op   string
		L, R Expr
	}
	// IntFunc reads a number out of the data being scanned.
	IntFunc struct {
		Name string
		X    Expr
	}
)

func (n Not) String() string     { return "(not " + n.X.String() + ")" }
func (d Defined) String() string { return "(defined " + d.X.String() + ")" }
func (u Unary) String() string   { return "(" + u.Op + " " + u.X.String() + ")" }
func (b Binary) String() string {
	return "(" + b.Op + " " + b.L.String() + " " + b.R.String() + ")"
}
func (f IntFunc) String() string { return "(" + f.Name + " " + f.X.String() + ")" }

// Quantifier is how many of a set have to hold.
type Quantifier struct {
	// Kind is "any", "all", "none", or "count" when a number was given.
	Kind  string
	Count Expr
}

// String renders a quantifier as it was written.
func (q Quantifier) String() string {
	if q.Kind == "count" {
		return q.Count.String()
	}
	return q.Kind
}

// StringSet is the strings an of-expression or a loop runs over: either all of
// them, or a list which may hold wildcards.
type StringSet struct {
	Them  bool
	Items []string
}

// String renders a set as it was written.
func (s StringSet) String() string {
	if s.Them {
		return "them"
	}
	return "(" + strings.Join(s.Items, " ") + ")"
}

// The two shapes that run a condition over several things.
type (
	// Of is "any of them" and its relatives.
	Of struct {
		Quantifier Quantifier
		Set        StringSet
	}
	// ForRange walks a stretch of numbers.
	ForRange struct {
		Quantifier Quantifier
		Var        string
		From, To   Expr
		Body       Expr
	}
	// ForOf walks a set of strings.
	ForOf struct {
		Quantifier Quantifier
		Set        StringSet
		Body       Expr
	}
)

func (o Of) String() string { return "(of " + o.Quantifier.String() + " " + o.Set.String() + ")" }
func (f ForRange) String() string {
	return fmt.Sprintf("(for %s %s in (%s %s) %s)",
		f.Quantifier, f.Var, f.From, f.To, f.Body)
}

func (f ForOf) String() string {
	return fmt.Sprintf("(for %s of %s %s)", f.Quantifier, f.Set, f.Body)
}

// What a module offers.
type (
	// ModuleStep is one step of the way to a value a module offers: a name to
	// look up, and then whatever follows it — a place in a list, a key in a
	// table, or a call.
	ModuleStep struct {
		Name string
		// Index is set when the step picks one out of a list or a table.
		Index Expr
		// Args is what a call was given, and Call says the step is a call at
		// all, since a call may be given nothing.
		Args []Expr
		Call bool
	}
	// ModuleRef is a value a module offers, such as pe.is_pe or
	// pe.sections[0].name.
	ModuleRef struct {
		Module string
		Steps  []ModuleStep
	}
)

func (m ModuleRef) String() string {
	var b strings.Builder
	b.WriteString(m.Module)
	for _, step := range m.Steps {
		b.WriteString("." + step.Name)
		if step.Index != nil {
			b.WriteString("[" + step.Index.String() + "]")
		}
		if step.Call {
			args := make([]string, 0, len(step.Args))
			for _, a := range step.Args {
				args = append(args, a.String())
			}
			b.WriteString("(" + strings.Join(args, ", ") + ")")
		}
	}
	return b.String()
}
