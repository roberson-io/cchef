package yara

import (
	"errors"
	"strings"
	"testing"
)

// parseOne parses a rule set holding a single rule and returns it.
func parseOne(t *testing.T, src string) *Rule {
	t.Helper()
	set, err := Parse(src)
	if err != nil {
		t.Fatalf("parsing %q: %v", src, err)
	}
	if len(set.Rules) != 1 {
		t.Fatalf("got %d rules, want 1", len(set.Rules))
	}
	return set.Rules[0]
}

// condition parses a rule with the given condition and renders what it came to.
func condition(t *testing.T, cond string) string {
	t.Helper()
	src := `rule R { strings: $a = "x" $b = "y" condition: ` + cond + ` }`
	return parseOne(t, src).Condition.String()
}

func TestParseRuleShape(t *testing.T) {
	r := parseOne(t, `
		global private rule Example : alpha beta {
			meta:
				author = "someone"
				count = 7
				ok = true
				offset = -2
			strings:
				$a = "hello" nocase wide
			condition:
				$a
		}`)

	if r.Name != "Example" {
		t.Errorf("name %q", r.Name)
	}
	if !r.Global || !r.Private {
		t.Errorf("global %v, private %v, want both", r.Global, r.Private)
	}
	if strings.Join(r.Tags, ",") != "alpha,beta" {
		t.Errorf("tags %v", r.Tags)
	}
	if len(r.Meta) != 4 || r.Meta[0].Key != "author" || r.Meta[0].Value != "someone" {
		t.Errorf("meta %v", r.Meta)
	}
	if r.Meta[3].Value != int64(-2) {
		t.Errorf("negative meta came out as %v", r.Meta[3].Value)
	}
	if r.Meta[1].Value != int64(7) || r.Meta[2].Value != true {
		t.Errorf("meta values %v %v", r.Meta[1].Value, r.Meta[2].Value)
	}
	if len(r.Strings) != 1 {
		t.Fatalf("got %d strings", len(r.Strings))
	}
	s := r.Strings[0]
	if s.ID != "$a" || s.Kind != stringText || s.Text != "hello" {
		t.Errorf("string %+v", s)
	}
	if !s.Mods.Nocase || !s.Mods.Wide {
		t.Errorf("modifiers %+v", s.Mods)
	}
}

func TestParseImports(t *testing.T) {
	set, err := Parse(`import "hash" import "math" rule R { condition: true }`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if strings.Join(set.Imports, ",") != "hash,math" {
		t.Errorf("imports %v", set.Imports)
	}
}

func TestParseStringKinds(t *testing.T) {
	r := parseOne(t, `rule R {
		strings:
			$t = "text"
			$h = { 68 ?? [1-3] ( 61 | 62 ) }
			$r = /ab+c/is
		condition: any of them
	}`)
	kinds := []stringKind{stringText, stringHex, stringRegex}
	bodies := []string{"text", "68 ?? [1-3] ( 61 | 62 )", "ab+c"}
	for i, s := range r.Strings {
		if s.Kind != kinds[i] || s.Text != bodies[i] {
			t.Errorf("string %d is %v %q", i, s.Kind, s.Text)
		}
	}
	if r.Strings[2].Flags != "is" {
		t.Errorf("regex flags %q", r.Strings[2].Flags)
	}
}

func TestParseStringModifiers(t *testing.T) {
	r := parseOne(t, `rule R {
		strings:
			$a = "one" nocase wide ascii fullword private
			$b = "two" xor
			$c = "three" xor(1-255)
			$d = "four" base64
			$e = "five" base64wide("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/")
		condition: any of them
	}`)
	a := r.Strings[0].Mods
	if !a.Nocase || !a.Wide || !a.ASCII || !a.Fullword || !a.Private {
		t.Errorf("modifiers %+v", a)
	}
	b := r.Strings[1].Mods
	if !b.XOR || b.XORMin != 0 || b.XORMax != 255 {
		t.Errorf("bare xor came out as %+v", b)
	}
	c := r.Strings[2].Mods
	if !c.XOR || c.XORMin != 1 || c.XORMax != 255 {
		t.Errorf("xor range came out as %+v", c)
	}
	if !r.Strings[3].Mods.Base64 {
		t.Errorf("base64 %+v", r.Strings[3].Mods)
	}
	if e := r.Strings[4].Mods; !e.Base64Wide || len(e.Base64Alphabet) != 64 {
		t.Errorf("base64wide %+v", e)
	}
}

func TestParseConditions(t *testing.T) {
	cases := []struct{ src, want string }{
		{`true`, "true"},
		{`false`, "false"},
		{`$a`, "$a"},
		{`not $a`, "(not $a)"},
		{`$a and $b`, "(and $a $b)"},
		{`$a or $b`, "(or $a $b)"},
		{`$a and $b or true`, "(or (and $a $b) true)"},
		{`$a or $b and true`, "(or $a (and $b true))"},
		{`not $a and $b`, "(and (not $a) $b)"},
		{`($a or $b) and true`, "(and (or $a $b) true)"},
		{`filesize`, "filesize"},
		{`filesize > 10`, "(> filesize 10)"},
		{`#a == 3`, "(== #a 3)"},
		{`@a == 0`, "(== @a[1] 0)"},
		{`@a[2] == 3`, "(== @a[2] 3)"},
		{`!a == 5`, "(== !a[1] 5)"},
		{`$a at 0`, "(at $a 0)"},
		{`$a in (0..10)`, "(in $a 0 10)"},
		{`1 + 2 * 3`, "(+ 1 (* 2 3))"},
		{`(1 + 2) * 3`, "(* (+ 1 2) 3)"},
		{`1 + 2 == 3`, "(== (+ 1 2) 3)"},
		{`-5 + 5`, "(+ (- 5) 5)"},
		{`~0`, "(~ 0)"},
		{`7 \ 2`, `(\ 7 2)`},
		{`7 % 2`, "(% 7 2)"},
		{`1 << 4`, "(<< 1 4)"},
		{`6 & 3 | 1`, "(| (& 6 3) 1)"},
		{`uint32(0)`, "(uint32 0)"},
		{`uint16be(4)`, "(uint16be 4)"},
		{`int8(0)`, "(int8 0)"},
		{`any of them`, "(of any them)"},
		{`all of them`, "(of all them)"},
		{`none of them`, "(of none them)"},
		{`2 of them`, "(of 2 them)"},
		{`any of ($a, $b)`, "(of any ($a $b))"},
		{`all of ($a*)`, "(of all ($a*))"},
		{`"abc" == "abc"`, `(== "abc" "abc")`},
		{`"abc" contains "b"`, `(contains "abc" "b")`},
		{`"abc" matches /b/`, `(matches "abc" /b/)`},
		{`hash.md5(0, filesize)`, "hash.md5(0, filesize)"},
		{`math.entropy(0, filesize) > 7.0`, "(> math.entropy(0, filesize) 7)"},
		{`pe.is_pe`, "pe.is_pe"},
		{`for any i in (1..3) : ( i > 1 )`, "(for any i in (1 3) (> i 1))"},
		{`for all of them : ( $ )`, "(for all of them $)"},
		{`defined filesize`, "(defined filesize)"},
		{`entrypoint`, "entrypoint"},
		{`filesize == 1KB`, "(== filesize 1024)"},
		{`$a at entrypoint`, "(at $a entrypoint)"},
		{`for 2 of ($a*) : ( $ )`, "(for 2 of ($a*) $)"},
		{`none of ($a, $b)`, "(of none ($a $b))"},
		{
			`hash.md5(0, filesize) == "x" or pe.number_of_sections > 1`,
			`(or (== hash.md5(0, filesize) "x") (> pe.number_of_sections 1))`,
		},
	}
	for _, c := range cases {
		t.Run(c.src, func(t *testing.T) {
			if got := condition(t, c.src); got != c.want {
				t.Errorf("got %s, want %s", got, c.want)
			}
		})
	}
}

// TestStringKindNames covers how a string's kind is named, which is what the
// compiler puts in its messages.
func TestStringKindNames(t *testing.T) {
	for kind, want := range map[stringKind]string{
		stringText: "text", stringHex: "hex", stringRegex: "regular expression",
	} {
		if got := kind.String(); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}
}

func TestParseRuleReference(t *testing.T) {
	set, err := Parse(`rule A { condition: true } rule B { condition: A }`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := set.Rules[1].Condition.String(); got != "A" {
		t.Errorf("got %s", got)
	}
}

func TestParseTracksLines(t *testing.T) {
	set, err := Parse("rule A {\n condition: true\n}\n\nrule B {\n condition: true\n}")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if set.Rules[0].Line != 1 || set.Rules[1].Line != 5 {
		t.Errorf("rules are on lines %d and %d, want 1 and 5", set.Rules[0].Line, set.Rules[1].Line)
	}
}

func TestParseRejectsBadRules(t *testing.T) {
	cases := []struct{ name, src string }{
		{"nothing after rule", `rule`},
		{"no body", `rule R`},
		{"an unclosed body", `rule R {`},
		{"no condition", `rule R { strings: $a = "x" }`},
		{"a condition with nothing in it", `rule R { condition: }`},
		{"a string with no value", `rule R { strings: $a = condition: true }`},
		{"an unknown modifier", `rule R { strings: $a = "x" sideways condition: $a }`},
		{"a meta value that is not one", `rule R { meta: a = $b condition: true }`},
		{"an unclosed group", `rule R { condition: (true }`},
		{"a dangling operator", `rule R { condition: true and }`},
		{"an import that is not a string", `import hash rule R { condition: true }`},
		{"rubbish after the last rule", `rule R { condition: true } nonsense`},
		{"an unclosed of-list", `rule R { strings: $a = "x" condition: any of ($a }`},
		{"a for loop with no body", `rule R { condition: for any i in (1..3) : }`},
		{"a string that never ends", `rule R { condition: "never`},
		{"a colon with no tags", `rule R : { condition: true }`},
		{"meta with no colon", `rule R { meta a = "x" condition: true }`},
		{"strings with no colon", `rule R { strings $a = "x" condition: $a }`},
		{"an empty meta section", `rule R { meta: condition: true }`},
		{"an empty strings section", `rule R { strings: condition: true }`},
		{"a meta key with no value", `rule R { meta: a = condition: true }`},
		{"a meta minus with no number", `rule R { meta: a = -"x" condition: true }`},
		{"a string with no equals", `rule R { strings: $a "x" condition: $a }`},
		{"an xor range that never closes", `rule R { strings: $a = "x" xor(1-2 condition: $a }`},
		{"an xor range with no number", `rule R { strings: $a = "x" xor(x) condition: $a }`},
		{"an xor range with no upper bound", `rule R { strings: $a = "x" xor(1-) condition: $a }`},
		{"a base64 alphabet that never closes", `rule R { strings: $a = "x" base64("AB" condition: $a }`},
		{"a base64 alphabet that is not text", `rule R { strings: $a = "x" base64(7) condition: $a }`},
		{"a for loop with no quantifier", `rule R { condition: for $a in (1..3) : ( true ) }`},
		{"a for loop with no variable", `rule R { condition: for any in (1..3) : ( true ) }`},
		{"a range with no dots", `rule R { strings: $a = "x" condition: $a in (0 10) }`},
		{"a range that never closes", `rule R { strings: $a = "x" condition: $a in (0..10 }`},
		{"a range with no opening bracket", `rule R { strings: $a = "x" condition: $a in 0..10 }`},
		{"an index that never closes", `rule R { strings: $a = "x" condition: @a[1 == 0 }`},
		{"an of-set that is not one", `rule R { condition: any of (7) }`},
		{"an integer function with no bracket", `rule R { condition: uint32 0 }`},
		{"an integer function that never closes", `rule R { condition: uint32(0 }`},
		{"a module path with no member", `rule R { condition: hash. }`},
		{"a call that never closes", `rule R { condition: hash.md5(0 }`},
		{"a module call with a bad argument", `rule R { condition: hash.md5(, 1) }`},
		{"a body that never closes", `rule R { condition: true`},
		{"a condition with no colon", `rule R { condition true }`},
		{"a meta section with no colon after the key", `rule R { meta: a "x" condition: true }`},
		{"not with nothing after it", `rule R { condition: not }`},
		{"defined with nothing after it", `rule R { condition: defined }`},
		{"a minus with nothing after it", `rule R { condition: - }`},
		{"a group opening on an operator", `rule R { condition: ( and ) }`},
		{"at with nothing after it", `rule R { strings: $a = "x" condition: $a at }`},
		{"an index holding nothing", `rule R { strings: $a = "x" condition: @a[] == 0 }`},
		{"a range starting on nothing", `rule R { strings: $a = "x" condition: $a in ( .. 10) }`},
		{"a range ending on nothing", `rule R { strings: $a = "x" condition: $a in (0 .. ) }`},
		{"a keyword that is not a condition", `rule R { condition: them }`},
		{"a quantifier with no of", `rule R { strings: $a = "x" condition: any them }`},
		{"an of-set that is neither", `rule R { strings: $a = "x" condition: any of 7 }`},
		{"a for-of with a bad set", `rule R { condition: for any of 7 : ( true ) }`},
		{"a for-of with a bad body", `rule R { strings: $a = "x" condition: for any of them : ( and ) }`},
		{"a for loop with in missing", `rule R { condition: for any i (1..3) : ( true ) }`},
		{"a for loop with a bad range", `rule R { condition: for any i in (1..) : ( true ) }`},
		{"a for body with no bracket", `rule R { condition: for any i in (1..3) : true }`},
		{"a for body holding nothing", `rule R { condition: for any i in (1..3) : ( ) }`},
		{"a for body that never closes", `rule R { condition: for any i in (1..3) : ( true }`},
		{"an integer function with a bad argument", `rule R { condition: uint32(and) }`},
		{"a loop body with no colon", `rule R { strings: $a = "x" condition: for any of them ( $ ) }`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := Parse(c.src); err == nil {
				t.Fatal("parsed something that should have been refused")
			}
		})
	}
}

func TestParseErrorsCarryTheLine(t *testing.T) {
	_, err := Parse("rule A { condition: true }\n\nrule B { condition: }")
	if err == nil {
		t.Fatal("parsed a rule with an empty condition")
	}
	var ce *compileError
	if !errors.As(err, &ce) {
		t.Fatalf("got %T, want a compile error", err)
	}
	if ce.Line() != 3 {
		t.Errorf("error is on line %d, want 3", ce.Line())
	}
}

// TestParseModuleStepFaults covers a place in a list that is not written out
// properly, which is refused rather than read as something else.
func TestParseModuleStepFaults(t *testing.T) {
	for _, src := range []string{
		`import "pe" rule R { condition: pe.sections[].name == "x" }`,
		`import "pe" rule R { condition: pe.sections[0.name == "x" }`,
		`import "pe" rule R { condition: pe.sections[0 .name == "x" }`,
	} {
		t.Run(src, func(t *testing.T) {
			if _, err := Parse(src); err == nil {
				t.Error("accepted a place that is not written out properly")
			}
		})
	}
}
