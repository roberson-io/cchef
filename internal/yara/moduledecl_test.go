package yara

import (
	"strings"
	"testing"
)

// What a module offers is checked while a rule compiles, so most of the ways of
// getting it wrong never reach a scan. These cover the checking itself, and the
// few steps a scan still has to take carefully.

// TestModuleRefRendering covers a reference written back out as it was in the
// rule, which is what a condition is shown as.
func TestModuleRefRendering(t *testing.T) {
	cases := []struct{ src, want string }{
		{`pe.is_pe`, "pe.is_pe"},
		{`pe.sections[0].name == "x"`, "pe.sections[0].name"},
		{`pe.rva_to_offset(16) == 0`, "pe.rva_to_offset(16)"},
		{`pe.imports("a.dll", "f")`, `pe.imports("a.dll", "f")`},
		{`pe.calculate_checksum() == 0`, "pe.calculate_checksum()"},
		{`pe.linker_version.major == 0`, "pe.linker_version.major"},
	}
	for _, c := range cases {
		t.Run(c.src, func(t *testing.T) {
			set, err := Parse(`import "pe" rule R { condition: ` + c.src + ` }`)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if got := set.Rules[0].Condition.String(); !strings.Contains(got, c.want) {
				t.Errorf("rendered as %q, want it to hold %q", got, c.want)
			}
		})
	}
}

// TestModuleArgumentsAreChecked covers a function given something of the wrong
// kind, which is refused when the rule compiles rather than at a scan. What can
// only be told by running the rule is let through.
func TestModuleArgumentsAreChecked(t *testing.T) {
	cases := []struct {
		name    string
		src     string
		refused bool
	}{
		{"text where a number belongs", `pe.rva_to_offset("x") == 0`, true},
		{"a number where text belongs", `pe.section_index(0) == 0`, false},
		{
			"a whole number standing in for a fraction",
			`math.in_range(5, 1, 10)`, false,
		},
		{
			"a fraction where a whole number belongs",
			`pe.rva_to_offset(1.5) == 0`, true,
		},
		{"a count, which is always a number", `pe.rva_to_offset(#a) == 0`, false},
		{"a value read from the data", `pe.rva_to_offset(uint8(0)) == 0`, false},
		{
			"one module's answer handed to another",
			`pe.rva_to_offset(pe.number_of_sections) == 0`, false,
		},
		{
			"text from one module where a number belongs",
			`pe.rva_to_offset(pe.sections[0].name) == 0`, true,
		},
		{
			"a regular expression where text belongs",
			`pe.section_index(/x/) == 0`, true,
		},
		{"a yes or no where a number belongs", `pe.rva_to_offset(true) == 0`, true},
		{"a yes or no where one belongs", `math.to_number(true) == 1`, false},
		{"too few arguments", `math.in_range(1.0, 2.0)`, true},
		{"too many arguments", `pe.rva_to_offset(1, 2) == 0`, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			src := `import "pe" import "math" rule R { strings: $a = "x" ` +
				`condition: $a and (` + c.src + `) }`
			err := compileText(t, src)
			if c.refused && err == nil {
				t.Error("accepted a call that libyara would refuse")
			}
			if !c.refused && err != nil {
				t.Errorf("refused a call libyara accepts: %v", err)
			}
		})
	}
}

// TestModuleIndexKinds covers the two ways of picking one value out of many: a
// place in a list, which is a number, and a key in a table, which is text.
func TestModuleIndexKinds(t *testing.T) {
	cases := []struct {
		name    string
		src     string
		refused bool
	}{
		{"a number naming a place in a list", `pe.sections[0].name == "x"`, false},
		{"text where a place belongs", `pe.sections["x"].name == "y"`, true},
		{"a fraction where a place belongs", `pe.sections[1.5].name == "y"`, true},
		{
			"a place worked out while the rule runs",
			`pe.sections[pe.number_of_sections - 1].name == "x"`, false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := compileText(t, `import "pe" rule R { condition: `+c.src+` }`)
			if c.refused && err == nil {
				t.Error("accepted a way of picking that libyara would refuse")
			}
			if !c.refused && err != nil {
				t.Errorf("refused a way of picking libyara accepts: %v", err)
			}
		})
	}
}

// TestModuleIndexAtScanTime covers picking one value out of many once the rule
// is running: a place past the end, or a key a table does not hold, comes to
// nothing rather than being a fault.
func TestModuleIndexAtScanTime(t *testing.T) {
	e := &evaluator{
		buf: newBuffer([]byte("x")), vars: map[string]int64{}, matched: map[string]bool{},
		modules: map[string]modValue{
			"m": structOf(map[string]modValue{
				"list":  listOf([]modValue{valueOf(intValue(7))}),
				"table": {table: map[string]modValue{"there": valueOf(intValue(9))}},
			}),
		},
	}
	read := func(name string, index Expr) value {
		t.Helper()
		got, err := e.evalModuleRef(ModuleRef{
			Module: "m", Steps: []ModuleStep{{Name: name, Index: index}},
		})
		if err != nil {
			t.Fatalf("reading %s: %v", name, err)
		}
		return got
	}

	wantInt(t, read("list", IntLit(0)), 7, "the only thing in the list")
	wantNothing(t, read("list", IntLit(1)), "a place past the end")
	wantNothing(t, read("list", IntLit(-1)), "a place before the start")
	wantNothing(t, read("list", StringLit("x")), "a place named by text")
	wantInt(t, read("table", StringLit("there")), 9, "a key the table holds")
	wantNothing(t, read("table", StringLit("elsewhere")), "a key the table does not hold")
	wantNothing(t, read("table", IntLit(0)), "a key that is a number")

	// A name the module does not offer at all comes to nothing, which the
	// checks refuse before a scan but which has to be safe here too.
	got, err := e.evalModuleRef(ModuleRef{
		Module: "m", Steps: []ModuleStep{{Name: "nothing"}},
	})
	if err != nil {
		t.Fatalf("reading a name that is not there: %v", err)
	}
	wantNothing(t, got, "a name the module does not offer")
}

// TestModuleWithoutASchema covers a module whose names are not declared, which
// is let through rather than refused, since there is nothing to check against.
func TestModuleWithoutASchema(t *testing.T) {
	rule := &Rule{Name: "R", EndLine: 1}
	ref := ModuleRef{Module: "nodecl", Steps: []ModuleStep{{Name: "anything"}}}
	if err := checkModuleRef(rule, ref); err != nil {
		t.Errorf("refused a name in a module that declares none: %v", err)
	}
	if _, known := moduleRefKind(ref); known {
		t.Error("claimed to know what a module without declarations offers")
	}
	if _, known := exprKind(Not{X: BoolLit(true)}); known {
		t.Error("claimed to know the kind of something it cannot place")
	}
}

// TestScanCarriesAFaultOutward covers a fault reaching the caller of a scan
// rather than being read as a rule that did not hold. The checks refuse a
// module that does not exist, so the rules are built here rather than written.
func TestScanCarriesAFaultOutward(t *testing.T) {
	missing := ModuleRef{Module: "nonsense", Steps: []ModuleStep{{Name: "thing"}}}
	cases := []struct {
		name      string
		condition Expr
	}{
		{"on its own", missing},
		{"on the right of an and", Binary{Op: "and", L: BoolLit(true), R: missing}},
		{"on the right of an or", Binary{Op: "or", L: BoolLit(false), R: missing}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rules := &Rules{
				Rules:    []*Rule{{Name: "R", Condition: c.condition}},
				patterns: []map[string][]*pattern{{}},
			}
			if _, _, err := rules.Scan([]byte("x")); err == nil {
				t.Fatal("a fault was read as a rule that did not hold")
			}
		})
	}
}

// TestModuleStepsThatLeadNowhere covers the steps a scan takes once a reference
// has been checked: a call, and a name that turns out to hold nothing.
func TestModuleStepsThatLeadNowhere(t *testing.T) {
	e := &evaluator{
		buf: newBuffer([]byte("x")), vars: map[string]int64{}, matched: map[string]bool{},
		modules: map[string]modValue{
			"m": structOf(map[string]modValue{
				"nothing": {},
				"call": funcOf(func(*evaluator, []value) (value, error) {
					return intValue(3), nil
				}),
			}),
		},
	}
	read := func(step ModuleStep) (value, error) {
		return e.evalModuleRef(ModuleRef{Module: "m", Steps: []ModuleStep{step}})
	}

	got, err := read(ModuleStep{Name: "call", Call: true})
	if err != nil {
		t.Fatalf("calling: %v", err)
	}
	wantInt(t, got, 3, "what the call came to")

	got, err = read(ModuleStep{Name: "nothing"})
	if err != nil {
		t.Fatalf("reading a name holding nothing: %v", err)
	}
	wantNothing(t, got, "a name holding nothing")

	// What a call is given is worked out first, so a fault in an argument
	// reaches the caller rather than being read as an answer.
	bad := ModuleRef{Module: "nonsense", Steps: []ModuleStep{{Name: "thing"}}}
	if _, err := read(ModuleStep{Name: "call", Call: true, Args: []Expr{bad}}); err == nil {
		t.Error("a fault in an argument was read as an answer")
	}
	// The same goes for working out which one of many to take.
	if _, err := read(ModuleStep{Name: "nothing", Index: bad}); err == nil {
		t.Error("a fault in a place was read as an answer")
	}
}

// TestModuleRefKindOfEveryStep covers following a reference through to what it
// stands for, which is how an argument's kind is told before a rule runs.
func TestModuleRefKindOfEveryStep(t *testing.T) {
	cases := []struct {
		name  string
		src   string
		want  modKind
		known bool
	}{
		{"a plain number", "pe.number_of_sections", modInt, true},
		{"a name", "pe.sections[0].name", modString, true},
		{"one of a list", "pe.sections[0].virtual_size", modInt, true},
		{"what a call gives back", "pe.calculate_checksum()", modInt, true},
		{"a name the module does not offer", "pe.nonsense", modInt, false},
		{
			"a place asked for in something that is not a list",
			"pe.number_of_sections[0]", modInt, false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			set, err := Parse(`import "pe" rule R { condition: ` + c.src + ` == 0 }`)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			ref, ok := set.Rules[0].Condition.(Binary).L.(ModuleRef)
			if !ok {
				t.Fatal("did not read as a reference into a module")
			}
			kind, known := moduleRefKind(ref)
			if known != c.known {
				t.Errorf("known %v, want %v", known, c.known)
			}
			if known && kind != c.want {
				t.Errorf("stands for %v, want %v", kind, c.want)
			}
		})
	}
}

// TestFaultOnTheLeftOfALogicalOperator covers a fault in the first half of an
// and or an or, which stops the whole thing rather than the second half
// deciding.
func TestFaultOnTheLeftOfALogicalOperator(t *testing.T) {
	missing := ModuleRef{Module: "nonsense", Steps: []ModuleStep{{Name: "thing"}}}
	e := &evaluator{buf: newBuffer([]byte("x")), vars: map[string]int64{}, matched: map[string]bool{}}
	for _, op := range []string{"and", "or"} {
		t.Run(op, func(t *testing.T) {
			if _, err := e.eval(Binary{Op: op, L: missing, R: BoolLit(true)}); err == nil {
				t.Error("a fault on the left was read as an answer")
			}
		})
	}
}

// TestDictionaryKeyKind covers the check on a key into a table. Nothing offers
// a table yet, so the declaration is built here.
func TestDictionaryKeyKind(t *testing.T) {
	rule := &Rule{Name: "R", EndLine: 1}
	table := &modDecl{kind: modDict, item: &modDecl{kind: modString}}
	if _, err := stepDecl(rule, ModuleStep{Name: "t", Index: StringLit("k")}, table); err != nil {
		t.Errorf("refused text as a key: %v", err)
	}
	_, err := stepDecl(rule, ModuleStep{Name: "t", Index: IntLit(0)}, table)
	if err == nil {
		t.Fatal("accepted a number as a key")
	}
	if want := "dictionary keys must be of string type"; !strings.Contains(err.Error(), want) {
		t.Errorf("said %q, want it to hold %q", err, want)
	}
}
