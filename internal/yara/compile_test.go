package yara

import (
	"errors"
	"strings"
	"testing"
)

// compileText parses and checks a rule set, returning whatever fault it found.
func compileText(t *testing.T, src string) error {
	t.Helper()
	set, err := Parse(src)
	if err != nil {
		return err
	}
	_, err = Compile(set)
	return err
}

// TestCompileAcceptsSoundRules covers rules with nothing wrong with them.
func TestCompileAcceptsSoundRules(t *testing.T) {
	cases := []string{
		`rule R { condition: true }`,
		`rule R { strings: $a = "x" condition: $a }`,
		`rule R { strings: $a = "x" $b = "y" condition: any of them }`,
		`rule R { strings: $ax = "x" $ay = "y" condition: all of ($a*) }`,
		`rule R { strings: $a = "x" condition: #a > 0 }`,
		`rule R { strings: $a = "x" condition: @a[1] == 0 }`,
		`rule R { strings: $a = "x" condition: !a == 1 }`,
		`rule R { strings: $a = "x" condition: $a at 0 }`,
		`rule A { condition: true } rule B { condition: A }`,
		`import "hash" rule R { condition: hash.md5(0, filesize) == "x" }`,
		`import "math" rule R { condition: math.entropy(0, filesize) > 1.0 }`,
		`import "time" rule R { condition: time.now() > 0 }`,
		`import "console" rule R { condition: console.log("hi") }`,
		`rule R { strings: $a = "x" condition: for any of them : ( $ ) }`,
		`rule R { strings: $a = "x" condition: for any i in (1..#a) : ( @a[i] > 0 ) }`,
		`rule R { strings: $a = "x" $b = "y" condition: any of ($a, $b) }`,
		`rule R { strings: $a = "x" $b = "y" condition: 2 of them }`,
		`rule R { strings: $a = "x" condition: $a in (0..10) }`,
		`rule R { strings: $a = "x" condition: not $a }`,
		`rule R { strings: $a = "x" condition: $a and defined filesize }`,
		`rule R { strings: $a = "x" condition: $a and -1 + ~0 < 0 }`,
		`rule R { strings: $a = "x" condition: $a and uint32(0) > 0 }`,
		`rule R { strings: $a = "x" condition: !a[1] == 1 }`,
		`import "console" rule R { strings: $a = "x" condition: $a and console.hex("b: ", int8(0)) }`,
	}
	for _, src := range cases {
		t.Run(src, func(t *testing.T) {
			if err := compileText(t, src); err != nil {
				t.Errorf("refused a sound rule: %v", err)
			}
		})
	}
}

// TestCompileErrors covers the faults libyara names, whose wording the
// operation has to reproduce.
func TestCompileErrors(t *testing.T) {
	cases := []struct{ name, src, want string }{
		{
			"a string that was never declared",
			`rule R { condition: $nope }`,
			`undefined string "$nope"`,
		},
		{
			"a count of a string that was never declared",
			`rule R { condition: #nope > 0 }`,
			`undefined string "$nope"`,
		},
		{
			"a string nothing refers to",
			`rule R { strings: $a = "x" condition: true }`,
			`unreferenced string "$a"`,
		},
		{
			"one of several strings nothing refers to",
			`rule R { strings: $ax = "x" $b = "y" condition: all of ($a*) }`,
			`unreferenced string "$b"`,
		},
		{
			"two rules with the same name",
			`rule R { condition: true } rule R { condition: true }`,
			`duplicated identifier "R"`,
		},
		{
			"a module that does not exist",
			`import "nonsense" rule R { condition: true }`,
			`unknown module "nonsense"`,
		},
		// YARA has these, but the build CyberChef runs was not made with them,
		// so a rule that imports one is refused as naming nothing at all.
		{
			"a module YARA has that this build does not",
			`import "macho" rule R { condition: true }`,
			`unknown module "macho"`,
		},
		{
			"the module that reads a file's kind",
			`import "magic" rule R { condition: true }`,
			`unknown module "magic"`,
		},
		{
			"the module that reads a sandbox report",
			`import "cuckoo" rule R { condition: true }`,
			`unknown module "cuckoo"`,
		},
		{
			"the module for Android packages",
			`import "dex" rule R { condition: true }`,
			`unknown module "dex"`,
		},
		{
			"the module for shortcut files",
			`import "lnk" rule R { condition: true }`,
			`unknown module "lnk"`,
		},
		{
			"a module used without importing it",
			`rule R { condition: hash.md5(0, filesize) == "x" }`,
			`undefined identifier "hash"`,
		},
		{
			"a rule that does not exist",
			`rule R { condition: Missing }`,
			`undefined identifier "Missing"`,
		},
		{
			"a module value read without importing it",
			`rule R { condition: pe.is_pe }`,
			`undefined identifier "pe"`,
		},
		{
			"a field a module does not have",
			`import "pe" rule R { condition: pe.is_signed }`,
			`invalid field name "is_signed"`,
		},
		{
			"a list used where a value belongs",
			`import "pe" rule R { condition: pe.sections }`,
			`wrong usage of identifier "sections"`,
		},
		{
			"a place asked for in something that is not a list",
			`import "pe" rule R { condition: pe.number_of_sections[0] == 0 }`,
			`"number_of_sections" is not an array or dictionary`,
		},
		{
			"a value called as though it were a function",
			`import "pe" rule R { condition: pe.number_of_sections() == 0 }`,
			`"number_of_sections" is not a function`,
		},
		{
			"a function given the wrong arguments",
			`import "pe" rule R { condition: pe.rva_to_offset("x") == 0 }`,
			`wrong arguments for function "rva_to_offset"`,
		},
		{
			"an of-list naming a string that is not there",
			`rule R { strings: $a = "x" condition: any of ($a, $nope) }`,
			`undefined string "$nope"`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := compileText(t, c.src)
			if err == nil {
				t.Fatal("accepted a rule that should have been refused")
			}
			var ce *compileError
			if !errors.As(err, &ce) {
				t.Fatalf("got %T, want a compile error", err)
			}
			if ce.Message() != c.want {
				t.Errorf("got %q, want %q", ce.Message(), c.want)
			}
		})
	}
}

// TestCompileErrorLines covers where a fault is reported. libyara names the line
// a rule closes on for anything it finds while finishing the rule, and the line
// a rule opens on for a name it has already seen — both checked against it.
func TestCompileErrorLines(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want int
	}{
		{
			"an undefined string, at the closing brace",
			"rule A {\n condition: true\n}\n\nrule B {\n condition: $nope\n}",
			7,
		},
		{
			"an unreferenced string, at the closing brace",
			"rule R {\n strings:\n  $a = \"x\"\n condition:\n  true\n}",
			6,
		},
		{
			"a duplicated name, at the opening",
			"rule R { condition: true }\n\nrule R { condition: true }",
			3,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var ce *compileError
			if !errors.As(compileText(t, c.src), &ce) {
				t.Fatal("want a compile error")
			}
			if ce.Line() != c.want {
				t.Errorf("reported on line %d, want %d", ce.Line(), c.want)
			}
		})
	}
}

// TestCompileModules covers which modules a rule may reach into. Every one in
// CyberChef's own build is here; the rest of YARA's are refused as naming
// nothing, which is what CyberChef does with them too.
func TestCompileModules(t *testing.T) {
	for _, name := range []string{"pe", "elf", "dotnet", "hash", "math", "console", "time"} {
		t.Run(name, func(t *testing.T) {
			if err := compileText(t, `import "`+name+`" rule R { condition: true }`); err != nil {
				t.Errorf("refused a module it has: %v", err)
			}
		})
	}
	for _, name := range []string{"macho", "dex", "lnk", "magic", "cuckoo", "string"} {
		t.Run(name, func(t *testing.T) {
			err := compileText(t, `import "`+name+`" rule R { condition: true }`)
			if err == nil {
				t.Fatal("accepted a module it does not have")
			}
			if !strings.Contains(err.Error(), name) {
				t.Errorf("the message does not name the module: %v", err)
			}
		})
	}
}

// TestWalkOfNothing covers walking a condition that is not there, which cannot
// happen through Compile but keeps the walk safe for callers that build an AST
// by hand.
func TestWalkOfNothing(t *testing.T) {
	seen := 0
	walk(nil, func(Expr) { seen++ })
	if seen != 0 {
		t.Errorf("visited %d nodes of nothing", seen)
	}
}

// TestCompileKeepsRuleOrder covers the order rules come back in, which is the
// order they were written and not the order they match.
func TestCompileKeepsRuleOrder(t *testing.T) {
	set, err := Parse(`rule Z { condition: true } rule A { condition: true }`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	rules, err := Compile(set)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if len(rules.Rules) != 2 || rules.Rules[0].Name != "Z" || rules.Rules[1].Name != "A" {
		t.Errorf("got %v", rules.Rules)
	}
}

// TestCompileWarnsAboutSlowStrings covers the warning libyara gives for a string
// too weak to search for quickly, whose wording and line the operation has to
// reproduce.
func TestCompileWarnsAboutSlowStrings(t *testing.T) {
	cases := []struct {
		name  string
		src   string
		warns []string
	}{
		{
			"a single letter",
			`rule R { strings: $a = "a" condition: $a }`,
			[]string{`Warning on line 1: string "$a" may slow down scanning`},
		},
		{
			"a single space, as in CyberChef's own fixture",
			"import \"console\"\nrule a\n{\n  strings:\n    $s=\" \"\n  condition:\n    $s\n}",
			[]string{`Warning on line 5: string "$s" may slow down scanning`},
		},
		{
			"a word worth searching for",
			`rule R { strings: $a = "hello" condition: $a }`,
			nil,
		},
		{
			"two weak strings, each named",
			`rule R { strings: $a = "a" $b = " " condition: any of them }`,
			[]string{
				`Warning on line 1: string "$a" may slow down scanning`,
				`Warning on line 1: string "$b" may slow down scanning`,
			},
		},
		{
			"a hex pattern with enough fixed bytes",
			`rule R { strings: $a = { 00 01 02 } condition: $a }`,
			nil,
		},
		{
			"a regular expression with little in it",
			`rule R { strings: $a = /a+/ condition: $a }`,
			[]string{`Warning on line 1: string "$a" may slow down scanning`},
		},
		{
			"a regular expression with a word in it",
			`rule R { strings: $a = /hello+/ condition: $a }`,
			nil,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			set, err := Parse(c.src)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			rules, err := Compile(set)
			if err != nil {
				t.Fatalf("compile: %v", err)
			}
			var got []string
			for _, w := range rules.Warnings {
				got = append(got, w.String())
			}
			if strings.Join(got, "\n") != strings.Join(c.warns, "\n") {
				t.Errorf("got  %q\nwant %q", got, c.warns)
			}
		})
	}
}

// TestCompileKeepsPatterns covers that a compiled rule carries what to look for
// for each of its strings, and more than one where a string was asked for in
// several forms.
func TestCompileKeepsPatterns(t *testing.T) {
	set, err := Parse(`rule R { strings: $a = "hello" wide ascii $b = "world" condition: any of them }`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	rules, err := Compile(set)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if len(rules.patterns) != 1 {
		t.Fatalf("got patterns for %d rules, want 1", len(rules.patterns))
	}
	if got := len(rules.patterns[0]["$a"]); got != 2 {
		t.Errorf("the string asked for in both widths came to %d patterns, want 2", got)
	}
	if got := len(rules.patterns[0]["$b"]); got != 1 {
		t.Errorf("the plain string came to %d patterns, want 1", got)
	}
}

// TestCompileRefusesAStringItCannotBuild covers a fault found while turning a
// rule's strings into what to look for.
func TestCompileRefusesAStringItCannotBuild(t *testing.T) {
	if err := compileText(t, `rule R { strings: $a = { 6g } condition: $a }`); err == nil {
		t.Fatal("accepted a hex pattern that does not make sense")
	}
}

// TestCompileRefusesABadRegex covers a regular expression that cannot be built,
// which is found when the rule's strings are turned into what to look for.
func TestCompileRefusesABadRegex(t *testing.T) {
	if err := compileText(t, `rule R { strings: $a = /he(/ condition: $a }`); err == nil {
		t.Fatal("accepted a regular expression that does not make sense")
	}
}

// TestWarnsAboutUnboundedDot covers the warning YARA gives for a pattern that
// repeats "anything at all" without an upper bound, which makes scanning slow.
// What does and does not draw it was read off CyberChef itself.
func TestWarnsAboutUnboundedDot(t *testing.T) {
	cases := []struct {
		pattern string
		want    bool
	}{
		{"h.*o", true},
		{"h.+o", true},
		{"h.{2,}o", true},
		{"h.*", true},
		{".*", true},
		{".+", true},
		{".{1,}", true},
		{"[a-z].*", true},
		// Plain brackets only group, so what is inside them still counts.
		{"(h.*o)", true},
		{"h(.*)o", true},
		// A bound, however large, is a bound.
		{"h.{2,5}o", false},
		{"h.{0,5}o", false},
		{"h.?o", false},
		// A full stop written as itself, or as one of a set, is not "anything".
		{`h\.*o`, false},
		{"h[.]*o", false},
		// A choice anywhere in the pattern leaves the warning unsaid.
		{"a|h.*o", false},
		{"h.*o|x", false},
	}
	for _, c := range cases {
		t.Run(c.pattern, func(t *testing.T) {
			src := `rule R { strings: $a = /` + c.pattern + `/ condition: $a }`
			set, err := Parse(src)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			rules, err := Compile(set)
			if err != nil {
				t.Fatalf("compile: %v", err)
			}
			said := false
			for _, w := range rules.Warnings {
				if strings.Contains(w.Message, "contains .*") {
					said = true
				}
			}
			if said != c.want {
				t.Errorf("warned %v, want %v", said, c.want)
			}
		})
	}
}

// TestUnboundedDotWarningWording covers the wording of that warning and where it
// falls among the others, which is before the one about scanning slowly.
func TestUnboundedDotWarningWording(t *testing.T) {
	set, err := Parse(`rule R { strings: $a = /h.*o/ condition: $a }`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	rules, err := Compile(set)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	want := "$a contains .*, .+ or .{x,} consider using .{,N}, .{1,N} or " +
		"{x,N} with a reasonable value for N"
	if len(rules.Warnings) == 0 || rules.Warnings[0].Message != want {
		t.Errorf("warned %v, want the first to be %q", rules.Warnings, want)
	}
}

// TestRefusesRegexBracketsThatDoMoreThanGroup covers brackets written to do
// something other than group. YARA's regular expressions have no such thing, so
// a pattern using them is refused rather than read as something else.
func TestRefusesRegexBracketsThatDoMoreThanGroup(t *testing.T) {
	for _, pattern := range []string{
		"(?:he)llo", "(?i)hello", "(?=he)llo", "(?!he)llo", "h(?#note)ello",
	} {
		t.Run(pattern, func(t *testing.T) {
			set, err := Parse(`rule R { strings: $a = /` + pattern + `/ condition: $a }`)
			if err != nil {
				return // refused as it was read, which is answer enough
			}
			_, err = compileStrings(set.Rules[0].Strings[0])
			if err == nil {
				t.Fatal("accepted brackets that do more than group")
			}
			want := `invalid regular expression "$a": syntax error, unexpected '?'`
			if !strings.Contains(err.Error(), want) {
				t.Errorf("said %q, want it to contain %q", err, want)
			}
		})
	}
}

// TestRefusesBracedByteEscape covers a byte written as its value inside braces.
// YARA only allows two digits after the x, so the braced form is refused rather
// than read as Go's own regular expressions would read it.
func TestRefusesBracedByteEscape(t *testing.T) {
	for _, pattern := range []string{`\x{68}`, `h\x{65}llo`, `[\x{68}]`} {
		t.Run(pattern, func(t *testing.T) {
			set, err := Parse(`rule R { strings: $a = /` + pattern + `/ condition: $a }`)
			if err != nil {
				return // refused as it was read, which is answer enough
			}
			_, err = compileStrings(set.Rules[0].Strings[0])
			if err == nil {
				t.Fatal("accepted a byte written as its value in braces")
			}
			want := `invalid regular expression "$a": illegal escape sequence`
			if !strings.Contains(err.Error(), want) {
				t.Errorf("said %q, want it to contain %q", err, want)
			}
		})
	}
}

// TestInvalidRegexNamesTheString covers how a pattern that cannot be read is
// reported: by the name the rule gave it, which is what YARA reports.
func TestInvalidRegexNamesTheString(t *testing.T) {
	set, err := Parse(`rule R { strings: $a = /a[b/ condition: $a }`)
	if err != nil {
		t.Skip("refused as it was read")
	}
	_, err = compileStrings(set.Rules[0].Strings[0])
	if err == nil {
		t.Fatal("accepted a pattern that cannot be read")
	}
	if want := `invalid regular expression "$a"`; !strings.Contains(err.Error(), want) {
		t.Errorf("said %q, want it to contain %q", err, want)
	}
}
