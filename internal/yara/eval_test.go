package yara

import (
	"fmt"
	"strings"
	"testing"
)

// scan compiles a rule set and runs it, returning the names of the rules that
// held.
func scan(t *testing.T, src, data string) []string {
	t.Helper()
	set, err := Parse(src)
	if err != nil {
		t.Fatalf("parsing %q: %v", src, err)
	}
	rules, err := Compile(set)
	if err != nil {
		t.Fatalf("compiling %q: %v", src, err)
	}
	results, _, err := rules.Scan([]byte(data))
	if err != nil {
		t.Fatalf("scanning: %v", err)
	}
	out := make([]string, 0, len(results))
	for _, r := range results {
		out = append(out, r.Rule.Name)
	}
	return out
}

// holds reports whether a single rule with the given condition held.
//
// Both strings are named in every rule, since a string a rule declares but never
// refers to is a fault; the two counts are always true and so leave the
// condition under test to decide the answer.
func holds(t *testing.T, cond, data string) bool {
	t.Helper()
	src := `rule R { strings: $a = "lo" $b = "l" ` +
		`condition: (` + cond + `) and #a >= 0 and #b >= 0 }`
	return len(scan(t, src, data)) == 1
}

func TestEvalConditions(t *testing.T) {
	const data = "hello world"
	cases := []struct {
		cond string
		want bool
	}{
		{`true`, true},
		{`false`, false},
		{`$a`, true},
		{`$a and $b`, true},
		{`$a and false`, false},
		{`$a or false`, true},
		{`not $a`, false},
		{`not false`, true},
		{`filesize == 11`, true},
		{`filesize > 100`, false},
		{`#a == 1`, true},
		{`#b == 3`, true},
		{`#b > 2 and #a == 1`, true},
		{`@a == 3`, true},
		{`@b[2] == 3`, true},
		{`@b[9] == 0`, false},
		{`!a == 2`, true},
		{`$a at 3`, true},
		{`$a at 0`, false},
		{`$a in (0..5)`, true},
		{`$a in (5..10)`, false},
		{`any of them`, true},
		{`all of them`, true},
		{`none of them`, false},
		{`2 of them`, true},
		{`3 of them`, false},
		{`any of ($a, $b)`, true},
		{`all of ($a*)`, true},
		{`1 + 2 * 3 == 7`, true},
		{`(1 + 2) * 3 == 9`, true},
		{`7 \ 2 == 3`, true},
		{`7 % 2 == 1`, true},
		{`-5 + 5 == 0`, true},
		{`~0 == -1`, true},
		{`(6 & 3) == 2 and (6 | 3) == 7 and (6 ^ 3) == 5`, true},
		{`1 << 4 == 16 and 16 >> 4 == 1`, true},
		{`1 << 64 == 0`, false},
		{`1 \ 0 == 0`, false},
		{`uint8(0) == 0x68`, true},
		{`uint16(0) == 0x6568`, true},
		{`uint16be(0) == 0x6865`, true},
		{`uint32(0) == 0x6c6c6568`, true},
		{`uint32be(0) == 0x68656c6c`, true},
		{`int8(0) == 104`, true},
		{`uint8(100) == 0`, false},
		{`"abc" == "abc"`, true},
		{`"abc" != "abd"`, true},
		{`"hello world" contains "lo w"`, true},
		{`"HELLO" icontains "hello"`, true},
		{`"hello" startswith "he"`, true},
		{`"hello" istartswith "HE"`, true},
		{`"hello" endswith "lo"`, true},
		{`"hello" iendswith "LO"`, true},
		{`"hello" iequals "HELLO"`, true},
		{`1.5 + 1.5 == 3.0`, true},
		{`3.0 > 2`, true},
		{`defined filesize`, true},
		{`defined entrypoint`, false},
		{`entrypoint == 0`, false},
		{`for any i in (1..#b) : ( @b[i] == 3 )`, true},
		{`for all i in (1..#b) : ( @b[i] > 1 )`, true},
		{`for all i in (1..#b) : ( @b[i] > 5 )`, false},
		{`for any of them : ( $ )`, true},
		{`for all of them : ( $ )`, true},
		{`for 2 of them : ( $ )`, true},
		{`for none of them : ( $ )`, false},
		{`none of ($a)`, false},
		{`"a" < "b"`, false},
		{`true + 1.0 == 1.0`, false},
		{`5 - 3 == 2`, true},
		{`5 <= 5 and 5 >= 5`, true},
		{`2 <= 1`, false},
		{`1 < 2`, true},
		{`1 != 2`, true},
		{`"hello" matches /h.llo/`, true},
		{`"hello" matches /^h/`, true},
		{`"hello" matches /HELLO/i`, true},
		{`"hello" matches /a.b/s`, false},
		{`"hello" matches /zzz/`, false},
		{`1 matches /1/`, false},
		{`"hello" matches "notaregex"`, false},
		// A text operator given two numbers has no meaning, so no answer.
		{`1 contains 2`, false},
		{`1 >= 2`, false},
		{`1.0 <= 1.0 and 1.0 >= 1.0`, true},
		{`for any i in (1..2) : ( for any i in (1..2) : ( i == 2 ) )`, true},
		{`"a" == 1`, false},
		{`1 == "a"`, false},
		{`true == true`, true},
		{`true != false`, true},
		{`true < false`, false},
		{`1.0 \ 2.0 > 0.4`, true},
		{`1.0 \ 0.0 == 0`, false},
		{`1.0 - 0.5 == 0.5`, true},
		{`2.0 * 2.0 == 4.0`, true},
		{`1 >> -1 == 0`, false},
		{`"a" contains 1`, false},
		{`-1.5 < 0`, true},
		{`-"a" == 0`, false},
		{`~1.5 == 0`, false},
		{`uint8(-1) == 0`, false},
		{`@b[0] == 0`, false},
		{`$a at "x"`, false},
		{`$a in (0.."x")`, false},
		{`for any i in ("x".."y") : ( true )`, false},
	}
	for _, c := range cases {
		t.Run(c.cond, func(t *testing.T) {
			if got := holds(t, c.cond, data); got != c.want {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

// TestEvalUndefinedSpreads covers YARA's rule that a question with no answer
// makes everything worked out from it unanswered too, and that a condition
// coming to no answer does not hold.
func TestEvalUndefinedSpreads(t *testing.T) {
	cases := []struct {
		cond string
		want bool
	}{
		{`@a[9] == 0`, false},
		{`@a[9] != 0`, false},
		{`@a[9] > 0 or true`, true},
		{`not @a[9]`, false},
		{`defined @a[9]`, false},
		{`defined @a[1]`, true},
		{`@a[9] + 1 == 1`, false},
		{`-@a[9] == 0`, false},
		{`~@a[9] == 0`, false},
	}
	for _, c := range cases {
		t.Run(c.cond, func(t *testing.T) {
			if got := holds(t, c.cond, "hello world"); got != c.want {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

// TestScanReportsRulesInOrder covers the order rules come back in, which is the
// order they were written.
func TestScanReportsRulesInOrder(t *testing.T) {
	got := scan(t, `rule Z { strings: $z = "world" condition: $z }
		rule A { strings: $a = "hello" condition: $a }`, "hello world")
	if strings.Join(got, ",") != "Z,A" {
		t.Errorf("got %v, want [Z A]", got)
	}
}

// TestScanPrivateRules covers a rule another may lean on but which is never
// reported itself.
func TestScanPrivateRules(t *testing.T) {
	got := scan(t, `private rule P { strings: $a = "hello" condition: $a }
		rule R { condition: P }`, "hello world")
	if strings.Join(got, ",") != "R" {
		t.Errorf("got %v, want [R]", got)
	}
}

// TestScanRuleReference covers one rule leaning on another that did not hold.
func TestScanRuleReference(t *testing.T) {
	got := scan(t, `rule A { strings: $a = "nope" condition: $a }
		rule B { condition: A }`, "hello world")
	if len(got) != 0 {
		t.Errorf("got %v, want nothing", got)
	}
}

// TestScanCollectsMatches covers what comes back with a rule that held: every
// place each of its strings was found, its strings in the order declared.
func TestScanCollectsMatches(t *testing.T) {
	set, err := Parse(`rule R { strings: $a = "l" $b = "o" condition: any of them }`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	rules, err := Compile(set)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	results, _, err := rules.Scan([]byte("hello world"))
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}

	var got []string
	for _, m := range results[0].Matches {
		got = append(got, m.ID)
	}
	if strings.Join(got, " ") != "$a $a $a $b $b" {
		t.Errorf("got %v, want three of $a then two of $b", got)
	}
	if first := results[0].Matches[0]; first.Offset != 2 || first.Length != 1 {
		t.Errorf("the first match is at %d for %d bytes", first.Offset, first.Length)
	}
}

// matchPlaces scans and gives back where each of a rule's strings was found, in
// the order they are reported.
func matchPlaces(t *testing.T, src, data string) []int {
	t.Helper()
	set, err := Parse(src)
	if err != nil {
		t.Fatalf("parsing %q: %v", src, err)
	}
	rules, err := Compile(set)
	if err != nil {
		t.Fatalf("compiling %q: %v", src, err)
	}
	results, _, err := rules.Scan([]byte(data))
	if err != nil {
		t.Fatalf("scanning: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("%d rules held, want 1", len(results))
	}
	out := make([]int, 0, len(results[0].Matches))
	for _, m := range results[0].Matches {
		out = append(out, m.Offset)
	}
	return out
}

// TestScanMatchesRunInOrder covers a string looked for in more than one form at
// once, whose matches are reported in the order they appear in the data rather
// than gathered form by form.
func TestScanMatchesRunInOrder(t *testing.T) {
	cases := []struct {
		name string
		src  string
		data string
		want []int
	}{
		{
			// Asked for under every key, the string is looked for in 256 forms,
			// and what each finds must still be reported in order.
			"under any key",
			`rule R { strings: $a = "lo" xor condition: any of them }`,
			"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f",
			[]int{1, 5, 9, 13},
		},
		{
			"under a range of keys",
			`rule R { strings: $a = "lo" xor(0-4) condition: any of them }`,
			"lo\x00\x00nm",
			[]int{0, 4},
		},
		{
			// Asked for both widths, the narrow and wide forms likewise.
			"in both widths",
			`rule R { strings: $a = "hi" wide ascii condition: any of them }`,
			"h\x00i\x00 hi",
			[]int{0, 5},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := matchPlaces(t, c.src, c.data)
			if fmt.Sprint(got) != fmt.Sprint(c.want) {
				t.Errorf("found at %v, want %v", got, c.want)
			}
		})
	}
}

// TestScanAnchoredRegex covers the marks that tie a regular expression to the
// ends of the data. In YARA they mean the start and end of what is being
// scanned, not of a line within it.
func TestScanAnchoredRegex(t *testing.T) {
	cases := []struct {
		name string
		src  string
		data string
		want []int
	}{
		{"tied to the start, and not there", `$a = /^h/`, "world hello", nil},
		{"tied to the start, and there", `$a = /^h/`, "hello world", []int{0}},
		{"tied to the start over several lines", `$a = /^h/`, "world\nhello", nil},
		{"tied to the end, and there", `$a = /d$/`, "hello world", []int{10}},
		{"tied to the end, and not there", `$a = /d$/`, "world hello", nil},
		{"tied to the end over several lines", `$a = /d$/`, "world\nhello", nil},
		{"tied at both ends", `$a = /^hello$/`, "hello", []int{0}},
		{"tied at both ends, with more after", `$a = /^hello$/`, "hello there", nil},
		// A mark that is only one way round still ties that way, and the other
		// side of the choice is free to match anywhere.
		{"tied one way among choices", `$a = /^w|lo/`, "hello world", []int{3}},
		{
			"a mark within a class is a plain character",
			`$a = /[^a-z]/`, "abc def",
			[]int{3},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			src := `rule R { strings: ` + c.src + ` condition: any of them }`
			if c.want == nil {
				if got := scan(t, src, c.data); len(got) != 0 {
					t.Errorf("the rule held, want it not to")
				}
				return
			}
			got := matchPlaces(t, src, c.data)
			if fmt.Sprint(got) != fmt.Sprint(c.want) {
				t.Errorf("found at %v, want %v", got, c.want)
			}
		})
	}
}

// TestScanAtOnlyNarrowsPlainStrings covers where a string is looked for when a
// condition ties it to one place. A plain run of bytes is looked for only
// there, but one that can match in more than one way is looked for throughout,
// which is what YARA itself does.
func TestScanAtOnlyNarrowsPlainStrings(t *testing.T) {
	cases := []struct {
		name string
		str  string
		want []int
	}{
		{"a plain string", `$a = "l"`, []int{2}},
		{"a plain run of bytes", `$a = { 6c }`, []int{2}},
		// These can each match in more than one way, so every match is kept
		// even though the condition names one place.
		{"a run of bytes offering a choice", `$a = { 6c ( 6c | 6f ) }`, []int{2, 3}},
		{"a pattern that can stretch", `$a = /l+/`, []int{2, 3}},
		{"a pattern offering a choice", `$a = /ll|lo/`, []int{2, 3}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			src := `rule R { strings: ` + c.str + ` condition: $a at 2 }`
			got := matchPlaces(t, src, "hello")
			if fmt.Sprint(got) != fmt.Sprint(c.want) {
				t.Errorf("found at %v, want %v", got, c.want)
			}
		})
	}
}

// TestScanAtDoesNotNarrowBase64 covers a string asked for as base64, which is
// looked for at three shifts at once and so is not narrowed down to one place
// even when a condition names one.
func TestScanAtDoesNotNarrowBase64(t *testing.T) {
	// "lll" encoded runs across the shifts, so the string is found more than
	// once even though the condition names the very beginning.
	src := `rule R { strings: $a = "lll" base64 condition: $a at 0 }`
	if got := matchPlaces(t, src, "bGxs bGxsbGxs"); len(got) < 2 {
		t.Errorf("found at %v, want every match kept rather than only the first", got)
	}
}

// TestScanLeavesOutPrivateStrings covers a string a rule uses but does not want
// reported.
func TestScanLeavesOutPrivateStrings(t *testing.T) {
	set, err := Parse(`rule R { strings: $a = "l" private $b = "o" condition: any of them }`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	rules, err := Compile(set)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	results, _, err := rules.Scan([]byte("hello world"))
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	for _, m := range results[0].Matches {
		if m.ID == "$a" {
			t.Error("a private string was reported")
		}
	}
}

// TestValueTruth covers how each sort of value reads as a condition, which is
// what decides whether a rule holds.
func TestValueTruth(t *testing.T) {
	cases := []struct {
		name string
		v    value
		want bool
	}{
		{"nothing at all", undefined, false},
		{"true", yes, true},
		{"false", no, false},
		{"a number", intValue(1), true},
		{"zero", intValue(0), false},
		{"a fraction", floatValue(0.5), true},
		{"no fraction", floatValue(0), false},
		{"some text", stringValue("x"), true},
		{"no text", stringValue(""), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.v.truth(); got != c.want {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

// TestValueNumber covers reading a value as a number, which only the two sorts
// of number answer to.
func TestValueNumber(t *testing.T) {
	if n, ok := intValue(3).number(); !ok || n != 3 {
		t.Errorf("a whole number read as %v %v", n, ok)
	}
	if n, ok := floatValue(1.5).number(); !ok || n != 1.5 {
		t.Errorf("a fraction read as %v %v", n, ok)
	}
	for _, v := range []value{stringValue("x"), yes, undefined} {
		if _, ok := v.number(); ok {
			t.Errorf("%v read as a number", v.kind)
		}
	}
}

// TestEvalIdentUnknown covers a name that is neither a loop's variable nor a
// rule, which the checks refuse before a scan but which comes to nothing here.
func TestEvalIdentUnknown(t *testing.T) {
	e := &evaluator{vars: map[string]int64{}, matched: map[string]bool{}}
	if got := e.evalIdent("nothing"); got.kind != valueUndefined {
		t.Errorf("got %v, want nothing at all", got.kind)
	}
}

// TestEvalOfSomethingItDoesNotKnow covers a piece of a condition the parser
// never builds, which is refused rather than read as an answer.
func TestEvalOfSomethingItDoesNotKnow(t *testing.T) {
	e := &evaluator{vars: map[string]int64{}, matched: map[string]bool{}}
	if _, err := e.eval(RegexLit{Body: "x"}); err == nil {
		t.Fatal("worked out something it has no meaning for")
	}
}

// faulting is a piece of a condition that cannot be worked out at all: it names
// a module that does not exist, which the checks refuse before a scan, so it is
// built here rather than written in a rule.
var faulting = ModuleRef{Module: "nonsense", Steps: []ModuleStep{{Name: "thing"}}}

// TestEvalCarriesFaultsOutward covers every place a piece of a condition can
// fail to be worked out at all. Whatever it is wrapped in, the fault has to
// reach the caller rather than be read as an answer.
func TestEvalCarriesFaultsOutward(t *testing.T) {
	strings := StringSet{Items: []string{"$a"}}
	anyOf := Quantifier{Kind: "any"}
	cases := []struct {
		name string
		expr Expr
	}{
		{"on the right of a comparison", Binary{Op: "==", L: IntLit(1), R: faulting}},
		{"on the left of a comparison", Binary{Op: "==", L: faulting, R: IntLit(1)}},
		{"on the right of an and", Binary{Op: "and", L: BoolLit(true), R: faulting}},
		{"on the right of an or", Binary{Op: "or", L: BoolLit(false), R: faulting}},
		{"under a not", Not{X: faulting}},
		{"under a minus", Unary{Op: "-", X: faulting}},
		{"as the place to read a number from", IntFunc{Name: "uint8", X: faulting}},
		{"as which match to take the offset of", StringOffset{ID: "$a", Index: faulting}},
		{"as which match to take the length of", StringLengthOf{ID: "$a", Index: faulting}},
		{"as where a string must be", StringAt{ID: "$a", Offset: faulting}},
		{"as the start of a stretch", StringIn{ID: "$a", From: faulting, To: IntLit(1)}},
		{"as the end of a stretch", StringIn{ID: "$a", From: IntLit(0), To: faulting}},
		{
			"as the start of a loop",
			ForRange{Quantifier: anyOf, Var: "i", From: faulting, To: IntLit(1), Body: BoolLit(true)},
		},
		{
			"as the end of a loop",
			ForRange{Quantifier: anyOf, Var: "i", From: IntLit(0), To: faulting, Body: BoolLit(true)},
		},
		{
			"inside a loop over numbers",
			ForRange{Quantifier: anyOf, Var: "i", From: IntLit(0), To: IntLit(1), Body: faulting},
		},
		{"inside a loop over strings", ForOf{Quantifier: anyOf, Set: strings, Body: faulting}},
		{"under a question of whether it is there at all", Defined{X: faulting}},
		{"as the text a pattern is tried against", Binary{Op: "matches", L: faulting, R: RegexLit{Body: "x"}}},
		{"on its own", faulting},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := &evaluator{
				buf:     newBuffer([]byte("hello world")),
				matches: map[string][]Match{"$a": {{Offset: 3, Length: 2}}},
				matched: map[string]bool{},
				vars:    map[string]int64{},
			}
			if _, err := e.eval(c.expr); err == nil {
				t.Fatal("a fault was read as an answer")
			}
		})
	}
}

// TestEvalIntFuncPastTheEnd covers reading a number from where the data does
// not reach, which has no answer rather than being an error.
func TestEvalIntFuncPastTheEnd(t *testing.T) {
	for _, cond := range []string{
		`uint32(9) == 0`,
		`uint32(-1) == 0`,
		`int16(10) == 0`,
		`defined uint32(9)`,
	} {
		t.Run(cond, func(t *testing.T) {
			if holds(t, cond, "hello world") {
				t.Error("read a number from past the end of the data")
			}
		})
	}
}

// TestEvalMatchesRefusesABadExpression covers a regular expression that cannot
// be read, which stops the scan rather than coming to nothing.
func TestEvalMatchesRefusesABadExpression(t *testing.T) {
	e := &evaluator{vars: map[string]int64{}, matched: map[string]bool{}}
	_, err := e.eval(Binary{Op: "matches", L: StringLit("x"), R: RegexLit{Body: "("}})
	if err == nil {
		t.Fatal("read a regular expression that does not make sense")
	}
}

// TestEvalMatchesFlags covers the letters a regular expression written into a
// condition may carry.
func TestEvalMatchesFlags(t *testing.T) {
	e := &evaluator{vars: map[string]int64{}, matched: map[string]bool{}}
	for _, c := range []struct {
		re   RegexLit
		want bool
	}{
		{RegexLit{Body: "A.C", Flags: "is"}, true},
		{RegexLit{Body: "A.C", Flags: "i"}, false},
		{RegexLit{Body: "a.c", Flags: "s"}, true},
	} {
		got, err := e.eval(Binary{Op: "matches", L: StringLit("a\nc"), R: c.re})
		if err != nil {
			t.Fatalf("eval: %v", err)
		}
		if got.truth() != c.want {
			t.Errorf("/%s/%s gave %v, want %v", c.re.Body, c.re.Flags, got.truth(), c.want)
		}
	}
}

// TestScanGlobalRule covers a rule marked global, which every other rule in the
// set depends on: when it does not hold, none of them do.
func TestScanGlobalRule(t *testing.T) {
	const rules = `global rule G { condition: filesize %s }
		rule R { strings: $a = "hello" condition: $a }`

	held := scan(t, fmt.Sprintf(rules, "> 0"), "hello world")
	if strings.Join(held, ",") != "G,R" {
		t.Errorf("with the global rule holding, got %v, want [G R]", held)
	}
	failed := scan(t, fmt.Sprintf(rules, "> 1000"), "hello world")
	if len(failed) != 0 {
		t.Errorf("with the global rule failing, got %v, want nothing", failed)
	}
}

// TestLoopOverAnEmptyRange covers a loop with nothing to walk, which does not
// hold however many of nothing were asked for. libyara reports no match for
// every quantifier, "none" included, so an empty range is not vacuously true.
func TestLoopOverAnEmptyRange(t *testing.T) {
	for _, quantifier := range []string{"all", "any", "none", "1"} {
		src := `rule R { strings: $a = "z" condition: for ` + quantifier +
			` i in (1..#a) : ( true ) }`
		t.Run(quantifier, func(t *testing.T) {
			if got := scan(t, src, "hello world"); len(got) != 0 {
				t.Errorf("a loop over nothing held: %v", got)
			}
		})
	}
	// A range that does have something in it still walks it.
	held := scan(t, `rule R { strings: $a = "l" condition: `+
		`for all i in (1..#a) : ( @a[i] >= 0 ) }`, "hello world")
	if len(held) != 1 {
		t.Errorf("a loop over three matches did not hold: %v", held)
	}
}

// TestStringSearchedAtOneOffset covers a string whose only mention in a
// condition fixes where it must be. libyara looks for it only there, so only
// that one match is reported; a string mentioned any other way is looked for
// throughout.
func TestStringSearchedAtOneOffset(t *testing.T) {
	cases := []struct {
		name      string
		condition string
		want      int
	}{
		{"only ever at one place", `$a at 2`, 1},
		{"at a place worked out from numbers", `$a at (1 + 1)`, 1},
		{"at one place, alongside something else", `$a at 2 and true`, 1},
		// Any other mention of the string means it is looked for throughout.
		{"at one place and on its own", `$a at 2 and $a`, 3},
		{"at one place and counted", `$a at 2 and #a > 0`, 3},
		{"at either of two places", `$a at 2 or $a at 9`, 3},
		{"within a stretch", `$a in (0..3)`, 3},
		{"at a place worked out while running", `$a at #a`, 3},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			src := `rule R { strings: $a = "l" condition: ` + c.condition + ` }`
			set, err := Parse(src)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			rules, err := Compile(set)
			if err != nil {
				t.Fatalf("compile: %v", err)
			}
			results, _, err := rules.Scan([]byte("hello world"))
			if err != nil {
				t.Fatalf("scan: %v", err)
			}
			if len(results) == 0 {
				t.Fatal("the rule did not hold at all")
			}
			if got := len(results[0].Matches); got != c.want {
				t.Errorf("reported %d matches, want %d", got, c.want)
			}
		})
	}
}

// TestStringSearchedAtOneOffsetWithASet covers a string named both at a settled
// place and through a set, which means it is looked for throughout after all.
func TestStringSearchedAtOneOffsetWithASet(t *testing.T) {
	for _, condition := range []string{
		`$a at 2 and any of them`,
		`$a at 2 and any of ($a*)`,
		`$a at 2 and any of ($a)`,
		`$a at 2 and for any of them : ( $ )`,
	} {
		t.Run(condition, func(t *testing.T) {
			set, err := Parse(`rule R { strings: $a = "l" condition: ` + condition + ` }`)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			rules, err := Compile(set)
			if err != nil {
				t.Fatalf("compile: %v", err)
			}
			results, _, err := rules.Scan([]byte("hello world"))
			if err != nil {
				t.Fatalf("scan: %v", err)
			}
			if len(results) == 0 {
				t.Fatal("the rule did not hold at all")
			}
			if got := len(results[0].Matches); got != 3 {
				t.Errorf("reported %d matches, want all 3", got)
			}
		})
	}
}

// TestSettledNumbers covers the sums a place can be written as, which have to
// be worked out before the rule runs for a string to be looked for in one place
// only.
func TestSettledNumbers(t *testing.T) {
	cases := []struct {
		src  string
		want int64
	}{
		{"7", 7},
		{"-3", -3},
		{"~0", -1},
		{"2 + 3", 5},
		{"9 - 4", 5},
		{"3 * 4", 12},
		{"(1 + 2) * 3 - 1", 8},
	}
	for _, c := range cases {
		t.Run(c.src, func(t *testing.T) {
			set, err := Parse(`rule R { strings: $a = "l" condition: $a at ` + c.src + ` }`)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			got := fixedOffsets(set.Rules[0])
			if got["$a"] != c.want {
				t.Errorf("came to %d, want %d", got["$a"], c.want)
			}
		})
	}

	// Anything that cannot be worked out beforehand leaves the string to be
	// looked for throughout.
	for _, src := range []string{"#a", "filesize", "uint8(0)", "-#a", "~#a"} {
		t.Run("not settled: "+src, func(t *testing.T) {
			set, err := Parse(`rule R { strings: $a = "l" condition: $a at ` + src + ` }`)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if got := fixedOffsets(set.Rules[0]); len(got) != 0 {
				t.Errorf("settled on %v, want nothing settled", got)
			}
		})
	}
}

// TestSettledNumbersOfEveryOperator covers each way a place may be worked out
// before the rule runs, which decides whether the string is looked for in one
// place only.
func TestSettledNumbersOfEveryOperator(t *testing.T) {
	cases := []struct {
		src  string
		want int64
	}{
		{"2 | 1", 3},
		{"6 ^ 5", 3},
		{"7 & 3", 3},
		{"1 << 2", 4},
		{"12 >> 2", 3},
		{`7 \ 2`, 3},
		{"7 % 4", 3},
	}
	for _, c := range cases {
		t.Run(c.src, func(t *testing.T) {
			set, err := Parse(`rule R { strings: $a = "l" condition: $a at ` + c.src + ` }`)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if got := fixedOffsets(set.Rules[0])["$a"]; got != c.want {
				t.Errorf("came to %d, want %d", got, c.want)
			}
		})
	}
	// A sum with no answer leaves the string to be looked for throughout.
	for _, src := range []string{`1 \ 0`, "1 % 0", "1 << 99", "1 >> 99", "1 << -1"} {
		t.Run("no answer: "+src, func(t *testing.T) {
			set, err := Parse(`rule R { strings: $a = "l" condition: $a at ` + src + ` }`)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if got := fixedOffsets(set.Rules[0]); len(got) != 0 {
				t.Errorf("settled on %v, want nothing settled", got)
			}
		})
	}
}

// TestSettledNumbersOfSomethingUnsettled covers a sum one half of which cannot
// be worked out beforehand, which leaves the whole thing unsettled.
func TestSettledNumbersOfSomethingUnsettled(t *testing.T) {
	for _, src := range []string{"#a + 1", "1 + #a", "filesize * 2"} {
		t.Run(src, func(t *testing.T) {
			set, err := Parse(`rule R { strings: $a = "l" condition: $a at ` + src + ` }`)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if got := fixedOffsets(set.Rules[0]); len(got) != 0 {
				t.Errorf("settled on %v, want nothing settled", got)
			}
		})
	}
}

// TestSettledPairOfAnUnfoldedOperator covers a sum written with an operator the
// folding does not do. No condition can reach it, since every operator that may
// follow "at" is folded, so it is called directly.
func TestSettledPairOfAnUnfoldedOperator(t *testing.T) {
	if _, ok := settledPair(Binary{Op: "==", L: IntLit(1), R: IntLit(1)}); ok {
		t.Error("folded a comparison into a place")
	}
}
