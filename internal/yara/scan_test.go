package yara

import (
	"fmt"
	"strings"
	"testing"
)

// findIn compiles the one string a rule declares and reports where it matched,
// as "offset:text" for each match.
func findIn(t *testing.T, decl, data string) []string {
	t.Helper()
	rule := parseOne(t, `rule R { strings: `+decl+` condition: $a }`)
	patterns, err := compileStrings(rule.Strings[0])
	if err != nil {
		t.Fatalf("compiling %s: %v", decl, err)
	}

	buf := newBuffer([]byte(data))
	var out []string
	for _, p := range patterns {
		for _, m := range p.findAll(buf) {
			out = append(out, fmt.Sprintf("%d:%s", m.Offset, m.Data))
		}
	}
	return out
}

// assertMatches compares what was found against what was wanted.
func assertMatches(t *testing.T, got, want []string) {
	t.Helper()
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("got  [%s]\nwant [%s]", strings.Join(got, " "), strings.Join(want, " "))
	}
}

func TestFindPlainText(t *testing.T) {
	assertMatches(t, findIn(t, `$a = "lo"`, "hello world"), []string{"3:lo"})
	assertMatches(t, findIn(t, `$a = "l"`, "hello world"), []string{"2:l", "3:l", "9:l"})
	assertMatches(t, findIn(t, `$a = "zz"`, "hello world"), nil)
}

// TestFindOverlapping covers the matches YARA reports at every position one can
// begin at, each reaching as far as it can.
func TestFindOverlapping(t *testing.T) {
	assertMatches(t, findIn(t, `$a = /a+/`, "aaa"), []string{"0:aaa", "1:aa", "2:a"})
	assertMatches(t, findIn(t, `$a = "aa"`, "aaaa"), []string{"0:aa", "1:aa", "2:aa"})
}

func TestFindNocase(t *testing.T) {
	assertMatches(t, findIn(t, `$a = "HeLLo" nocase`, "hello HELLO"), []string{"0:hello", "6:HELLO"})
	assertMatches(t, findIn(t, `$a = "hello"`, "HELLO"), nil)
}

// TestFindWide covers a string written with a null after every byte, and the
// ascii modifier that asks for the plain form as well.
func TestFindWide(t *testing.T) {
	wide := "h\x00i\x00 there"
	assertMatches(t, findIn(t, `$a = "hi" wide`, wide), []string{"0:h\x00i\x00"})
	assertMatches(t, findIn(t, `$a = "hi" wide`, "hi"), nil)
	assertMatches(t, findIn(t, `$a = "hi" wide ascii`, "hi "+wide),
		[]string{"0:hi", "3:h\x00i\x00"})
}

func TestFindFullword(t *testing.T) {
	assertMatches(t, findIn(t, `$a = "cat" fullword`, "cat"), []string{"0:cat"})
	assertMatches(t, findIn(t, `$a = "cat" fullword`, "the cat sat"), []string{"4:cat"})
	assertMatches(t, findIn(t, `$a = "cat" fullword`, "concatenate"), nil)
	assertMatches(t, findIn(t, `$a = "cat" fullword`, "cat9"), nil)
	// An underscore is not a letter or a digit, so it does not join a word.
	assertMatches(t, findIn(t, `$a = "cat" fullword`, "_cat"), []string{"1:cat"})
	assertMatches(t, findIn(t, `$a = "cat" fullword`, "(cat)"), []string{"1:cat"})
	// A wide match looks two bytes either side, since that is how wide a
	// character is there.
	assertMatches(t, findIn(t, `$a = "cat" fullword wide`, "c\x00a\x00t\x00"),
		[]string{"0:c\x00a\x00t\x00"})
	assertMatches(t, findIn(t, `$a = "cat" fullword wide`, "x\x00c\x00a\x00t\x00"), nil)
	assertMatches(t, findIn(t, `$a = "cat" fullword wide`, "c\x00a\x00t\x00x\x00"), nil)
}

// TestFindXor covers a string hidden under a key, which is what a packer or a
// dropper does to keep it out of sight.
func TestFindXor(t *testing.T) {
	hidden := make([]byte, 0, 3)
	for _, c := range []byte("cat") {
		hidden = append(hidden, c^0x2a)
	}
	assertMatches(t, findIn(t, `$a = "cat" xor`, string(hidden)), []string{"0:" + string(hidden)})
	assertMatches(t, findIn(t, `$a = "cat" xor(1-16)`, string(hidden)), nil)
	assertMatches(t, findIn(t, `$a = "cat" xor(0x2a-0x2a)`, string(hidden)),
		[]string{"0:" + string(hidden)})
	// A key of nothing leaves the string as it was, so xor still finds it plain.
	assertMatches(t, findIn(t, `$a = "cat" xor`, "cat"), []string{"0:cat"})
}

// TestFindBytesAboveASCII covers the bytes a character-oriented matcher would
// read as something else entirely.
func TestFindBytesAboveASCII(t *testing.T) {
	data := "\xff\xfe\x00\xc3\xa9"
	assertMatches(t, findIn(t, `$a = { ff fe }`, data), []string{"0:\xff\xfe"})
	assertMatches(t, findIn(t, `$a = { c3 a9 }`, data), []string{"3:\xc3\xa9"})
	assertMatches(t, findIn(t, `$a = { 00 c3 }`, data), []string{"2:\x00\xc3"})
}

func TestFindHexPatterns(t *testing.T) {
	cases := []struct {
		name string
		decl string
		data string
		want []string
	}{
		{"exact bytes", `$a = { 68 65 6c }`, "hello", []string{"0:hel"}},
		{"a wildcard byte", `$a = { 68 ?? 6c }`, "hello", []string{"0:hel"}},
		{"a fixed high half", `$a = { 6? 65 }`, "hello", []string{"0:he"}},
		{"a fixed low half", `$a = { ?8 65 }`, "hello", []string{"0:he"}},
		{"a jump", `$a = { 68 [1-3] 6f }`, "hello", []string{"0:hello"}},
		{"an exact jump", `$a = { 68 [3] 6f }`, "hello", []string{"0:hello"}},
		{"an open jump", `$a = { 68 [-] 6f }`, "hello", []string{"0:hello"}},
		{"a jump with no upper bound", `$a = { 68 [2-] 6f }`, "hello", []string{"0:hello"}},
		{"alternatives", `$a = { 68 ( 65 | 66 ) 6c }`, "hello", []string{"0:hel"}},
		{"the other alternative", `$a = { ( 61 | 68 ) 65 }`, "hello", []string{"0:he"}},
		{"nested alternatives", `$a = { 68 ( 65 ( 6c | 6d ) | 66 ) }`, "hello", []string{"0:hel"}},
		{"nothing to find", `$a = { 99 99 }`, "hello", nil},
		{"a jump of nothing at all", `$a = { 68 [0] 65 }`, "hello", []string{"0:he"}},
		// The branches need not be the same length; the first that fits wins.
		{"alternatives of unequal length", `$a = { ( 68 65 | 68 ) 6c }`, "hello", []string{"0:hel"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assertMatches(t, findIn(t, c.decl, c.data), c.want)
		})
	}
}

func TestFindRegex(t *testing.T) {
	cases := []struct {
		name string
		decl string
		data string
		want []string
	}{
		{"a class", `$a = /h[ea]llo/`, "hello", []string{"0:hello"}},
		{"case folded by a flag", `$a = /HELLO/i`, "hello", []string{"0:hello"}},
		{"case folded by a modifier", `$a = /HELLO/ nocase`, "hello", []string{"0:hello"}},
		{"alternatives", `$a = /hello|goodbye/`, "goodbye", []string{"0:goodbye"}},
		{"an anchor", `$a = /^he/`, "hello", []string{"0:he"}},
		{"a dot stopping at a newline", `$a = /a.b/`, "a\nb", nil},
		{"a dot crossing one", `$a = /a.b/s`, "a\nb", []string{"0:a\nb"}},
		{"a repeat", `$a = /l{2}/`, "hello", []string{"2:ll"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assertMatches(t, findIn(t, c.decl, c.data), c.want)
		})
	}
}

// TestStringsRefuseModifiersTheyCannotTake covers the words a string may not
// carry. A hex pattern takes only "private" and a regular expression will not
// take the two that rewrite the bytes, both of which libyara refuses as it
// reads them; widening a regular expression is not yet done here and is refused
// when the pattern is built.
func TestStringsRefuseModifiersTheyCannotTake(t *testing.T) {
	for _, decl := range []string{
		`$a = { 68 65 } wide`,
		`$a = { 68 65 } xor`,
		`$a = { 68 65 } nocase`,
		`$a = { 68 65 } base64wide`,
		`$a = /he/ xor`,
		`$a = /he/ base64`,
	} {
		t.Run(decl, func(t *testing.T) {
			set, err := Parse(`rule R { strings: ` + decl + ` condition: $a }`)
			if err != nil {
				return // refused as it was read, which is libyara's own answer
			}
			if _, err := compileStrings(set.Rules[0].Strings[0]); err == nil {
				t.Fatal("accepted a modifier that is not supported here")
			}
		})
	}
}

// TestHexStringsTakePrivate covers the one word a hex pattern does carry.
func TestHexStringsTakePrivate(t *testing.T) {
	if _, err := Parse(`rule R { strings: $a = { 68 65 } private condition: $a }`); err != nil {
		t.Errorf("refused the one modifier a hex pattern takes: %v", err)
	}
}

// TestRegexStringsTakeTheirModifiers covers the words a regular expression does
// carry, other than the widening that is still to come.
func TestRegexStringsTakeTheirModifiers(t *testing.T) {
	for _, decl := range []string{
		`$a = /he/ nocase`, `$a = /he/ ascii`, `$a = /he/ fullword`, `$a = /he/ private`,
	} {
		t.Run(decl, func(t *testing.T) {
			if _, err := Parse(`rule R { strings: ` + decl + ` condition: $a }`); err != nil {
				t.Errorf("refused a modifier a regular expression takes: %v", err)
			}
		})
	}
}

// TestFindBase64 covers a string looked for as base64, which may sit at any of
// three places within the groups of three bytes the encoding works in.
func TestFindBase64(t *testing.T) {
	// "aGVsbG8gd29ybGQ=" is "hello world" encoded.
	assertMatches(t, findIn(t, `$a = "hello" base64`, "aGVsbG8gd29ybGQ="),
		[]string{"0:aGVsbG"})
	// A string that does not begin on a group boundary is found through one of
	// the other two encodings.
	if got := findIn(t, `$a = "ello" base64`, "aGVsbG8gd29ybGQ="); len(got) == 0 {
		t.Error("a string starting mid-group was not found")
	}
	assertMatches(t, findIn(t, `$a = "nothing" base64`, "aGVsbG8gd29ybGQ="), nil)
	// Asked for both, it is looked for plain and with a null after every byte.
	if got := findIn(t, `$a = "hello" base64 base64wide`, "aGVsbG8gd29ybGQ="); len(got) != 1 {
		t.Errorf("got %v, want the plain form only", got)
	}
}

// TestBase64Alphabet covers an alphabet of a rule's own, and one of the wrong
// size.
func TestBase64Alphabet(t *testing.T) {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	rule := parseOne(t, `rule R { strings: $a = "hello" base64("`+alphabet+`") condition: $a }`)
	if _, err := compileStrings(rule.Strings[0]); err != nil {
		t.Errorf("refused an alphabet of the right size: %v", err)
	}

	short := parseOne(t, `rule R { strings: $a = "hello" base64("ABC") condition: $a }`)
	if _, err := compileStrings(short.Strings[0]); err == nil {
		t.Fatal("accepted an alphabet that is not 64 characters")
	}
}

// TestCompileStringsRefusesBadPatterns covers patterns that do not make sense.
func TestCompileStringsRefusesBadPatterns(t *testing.T) {
	for _, decl := range []string{
		`$a = { 6g 65 }`,
		`$a = { 68 6 }`,
		`$a = { 68 [1-2 }`,
		`$a = { 68 [x] }`,
		`$a = { 68 [1-x] }`,
		`$a = { 68 [x-] }`,
		`$a = { 68 [] }`,
		`$a = { 68 [1-2-3] }`,
		`$a = { 68 [-x] }`,
		`$a = { ( 68 [x] ) }`,
		`$a = { ( 68 | ) 65 6c 6f 78 }`,
		`$a = { ( 68 65 | 6c 6f ) ( }`,
		`$a = { ( 68 }`,
		`$a = { ( 68 65 }`,
		`$a = { 68 ) }`,
		`$a = /he(/`,
		`$a = /he[/`,
	} {
		t.Run(decl, func(t *testing.T) {
			// A hex pattern is checked as it is read, so some of these are
			// refused before a rule is even built; the rest when the pattern is
			// turned into something to look for.
			set, err := Parse(`rule R { strings: ` + decl + ` condition: $a }`)
			if err != nil {
				return
			}
			if _, err := compileStrings(set.Rules[0].Strings[0]); err == nil {
				t.Fatal("accepted a pattern that does not make sense")
			}
		})
	}
}

// TestBufferMapsPositionsBack covers the table that turns a position in the
// mapped text back into the byte it came from.
func TestBufferMapsPositionsBack(t *testing.T) {
	data := []byte{0x41, 0xff, 0x42, 0x80}
	buf := newBuffer(data)
	for i := range data {
		if got := buf.starts[buf.offsets[i]]; got != i {
			t.Errorf("byte %d maps to %d and back to %d", i, buf.offsets[i], got)
		}
	}
	if buf.starts[buf.offsets[len(data)]] != len(data) {
		t.Error("the end does not map back to the end")
	}
}

// TestWideRegex covers a regular expression asked for wide, which is looked for
// with a null after every character it matches. What repeats has to count the
// pair rather than the character alone, or a pattern that stretches would find
// runs that are not there.
func TestWideRegex(t *testing.T) {
	wide := func(s string) string {
		var out []byte
		for i := range len(s) {
			out = append(out, s[i], 0)
		}
		return string(out)
	}
	cases := []struct {
		name string
		str  string
		data string
		want []int
	}{
		{"a pattern that stretches", `$a = /hel+o/ wide`, wide("hello"), []int{0}},
		{"the same pattern narrow, which it is not", `$a = /hel+o/ wide`, "hello", nil},
		{
			"asked for both widths", `$a = /hello/ wide ascii`,
			wide("hello") + "hello",
			[]int{0, 10},
		},
		{"a class of characters", `$a = /[a-c]+/ wide`, wide("abc"), []int{0, 2, 4}},
		{"one that may or may not be there", `$a = /ab?c/ wide`, wide("ac"), []int{0}},
		{"a choice between two", `$a = /ab|cd/ wide`, wide("cd"), []int{0}},
		{"a group repeated", `$a = /(ab)+/ wide`, wide("abab"), []int{0, 4}},
		{"a count of repeats", `$a = /a{2,3}/ wide`, wide("aaa"), []int{0, 2}},
		{"anything at all", `$a = /a.c/ wide`, wide("abc"), []int{0}},
		{"a character written as its value", `$a = /a\x62c/ wide`, wide("abc"), []int{0}},
		{"tied to the start", `$a = /^ab/ wide`, wide("ab"), []int{0}},
		{"tied to the start and not there", `$a = /^b/ wide`, wide("ab"), nil},
		{"whatever the case", `$a = /HELLO/ wide nocase`, wide("hello"), []int{0}},
		// A mark that stands between characters rather than matching one takes
		// no null of its own.
		{"tied to a word boundary", `$a = /\bhello/ wide`, wide("hello"), []int{0}},
		{"tied to a boundary mid-text", `$a = /\bworld/ wide`, wide("hi world"), []int{6}},
		{"tied to the very start", `$a = /\Ahello/ wide`, wide("hello"), []int{0}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			src := `rule R { strings: ` + c.str + ` condition: any of them }`
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

// TestFullwordStandsAlone covers what counts as a string standing alone rather
// than sitting inside a longer word. Only letters and digits make a word, so a
// string with punctuation either side of it stands alone.
func TestFullwordStandsAlone(t *testing.T) {
	cases := []struct {
		name string
		data string
		want []int
	}{
		{"punctuation either side", "_hello_", []int{1}},
		{"a hyphen either side", "-hello-", []int{1}},
		{"a digit before it", "1hello", nil},
		{"a letter after it", "hellox", nil},
		{"nothing either side", "hello", []int{0}},
		{"spaces either side", " hello ", []int{1}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			src := `rule R { strings: $a = "hello" fullword condition: $a }`
			if c.want == nil {
				if got := scan(t, src, c.data); len(got) != 0 {
					t.Errorf("the rule held, want it not to")
				}
				return
			}
			if got := matchPlaces(t, src, c.data); fmt.Sprint(got) != fmt.Sprint(c.want) {
				t.Errorf("found at %v, want %v", got, c.want)
			}
		})
	}
}

// TestFullwordWideStandsAlone covers the same for a string looked for wide,
// where what sits either side is a pair of bytes rather than one. A pair only
// makes a word when its second byte is a null and its first is a letter or a
// digit, so plain narrow text after a wide match does not run into it.
func TestFullwordWideStandsAlone(t *testing.T) {
	wide := func(s string) string {
		var out []byte
		for i := range len(s) {
			out = append(out, s[i], 0)
		}
		return string(out)
	}
	cases := []struct {
		name string
		data string
		want []int
	}{
		{"nothing either side", wide("hello"), []int{0}},
		{"punctuation either side", wide("_hello_"), []int{2}},
		{"a wide letter before it", wide("ahello"), nil},
		{"a wide letter after it", wide("hellox"), nil},
		// The bytes after the match are narrow text, so the pair there is not a
		// wide character at all and the match still stands alone.
		{"narrow text after it", wide("hello") + "hello", []int{0}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			src := `rule R { strings: $a = "hello" wide fullword condition: $a }`
			if c.want == nil {
				if got := scan(t, src, c.data); len(got) != 0 {
					t.Errorf("the rule held, want it not to")
				}
				return
			}
			if got := matchPlaces(t, src, c.data); fmt.Sprint(got) != fmt.Sprint(c.want) {
				t.Errorf("found at %v, want %v", got, c.want)
			}
		})
	}
}

// TestOneMatchPerPlace covers a string looked for in several forms at once that
// match in the same place. YARA keeps one match per place, the first found, so
// asking for both widths does not report the same place twice.
func TestOneMatchPerPlace(t *testing.T) {
	wide := func(s string) string {
		var out []byte
		for i := range len(s) {
			out = append(out, s[i], 0)
		}
		return string(out)
	}
	// Over wide text, the narrow form matches each letter on its own and the
	// wide form matches the whole run. Both begin at the same places, and the
	// narrow one is the form looked for first.
	src := `rule R { strings: $a = /[a-z]+/ wide ascii condition: $a }`
	set, err := Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	rules, err := Compile(set)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	results, _, err := rules.Scan([]byte(wide("abc")))
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("%d rules held, want 1", len(results))
	}
	var got []string
	for _, m := range results[0].Matches {
		got = append(got, fmt.Sprintf("%d/%d", m.Offset, m.Length))
	}
	if want := "0/1 2/1 4/1"; strings.Join(got, " ") != want {
		t.Errorf("found %v, want %v", got, want)
	}
}

// TestWidenRegexShapes covers rewriting a pattern to be looked for wide, piece
// by piece, including the shapes that are not a piece at all and the ones that
// cannot be rewritten.
func TestWidenRegexShapes(t *testing.T) {
	const nul = `\x00`
	cases := []struct {
		in   string
		want string
	}{
		{"ab", `(?:a` + nul + `)(?:b` + nul + `)`},
		{"a+", `(?:a` + nul + `)+`},
		{"a*?", `(?:a` + nul + `)*?`},
		{"a{2,3}", `(?:a` + nul + `){2,3}`},
		{"a{2,3}?", `(?:a` + nul + `){2,3}?`},
		{"[a-c]", `(?:[a-c]` + nul + `)`},
		{`\d`, `(?:\d` + nul + `)`},
		{`\x41`, `(?:\x41` + nul + `)`},
		{`\x{41}`, `(?:\x{41}` + nul + `)`},
		// What holds pieces together takes no null of its own, but a repeat
		// after a group still counts the whole group.
		{"(a)+", `((?:a` + nul + `))+`},
		{"a|b", `(?:a` + nul + `)|(?:b` + nul + `)`},
		{"^a$", `^(?:a` + nul + `)$`},
		// A brace that nothing closes is a plain character rather than a count,
		// and was already given a null of its own.
		{"a{2", `(?:a` + nul + `)(?:{` + nul + `)(?:2` + nul + `)`},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got, ok := widenRegex(c.in)
			if !ok {
				t.Fatal("the pattern could not be rewritten")
			}
			if got != c.want {
				t.Errorf("reads as %q, want %q", got, c.want)
			}
		})
	}
}

// TestWidenRegexRefused covers patterns that cannot be rewritten to be looked
// for wide, which are refused rather than half rewritten.
func TestWidenRegexRefused(t *testing.T) {
	for _, in := range []string{
		`a\`,     // an escape naming nothing
		`a\x4`,   // a byte written as too few digits
		`a\x{41`, // braces that nothing closes
		`a[b\`,   // a class ending in an escape naming nothing
	} {
		t.Run(in, func(t *testing.T) {
			if _, ok := widenRegex(in); ok {
				t.Error("the pattern was rewritten, want it refused")
			}
		})
	}
}

// TestWideRegexRefusedWhenCompiled covers a pattern that cannot be rewritten,
// which is reported against the string that asked for it.
func TestWideRegexRefusedWhenCompiled(t *testing.T) {
	set, err := Parse(`rule R { strings: $a = /a\x4/ wide condition: $a }`)
	if err != nil {
		return // refused as it was read, which is answer enough
	}
	if _, err := compileStrings(set.Rules[0].Strings[0]); err == nil {
		t.Error("accepted a pattern that cannot be looked for wide")
	}
}

// TestClassBracketsAreLiteral covers a class of characters written to look like
// one of the named classes other regular expression dialects have. YARA has no
// named classes, so `[[:alpha:]]` is a class holding those very characters —
// `[`, `:`, `a`, `l`, `p`, `h` — followed by one or more closing brackets. Each
// answer here is the one CyberChef gave for the same pattern.
func TestClassBracketsAreLiteral(t *testing.T) {
	cases := []struct {
		pattern string
		data    string
		want    []int
	}{
		{"[[:alpha:]]+", "a]]] x", []int{0}},
		{"[[:alpha:]]+", "hello", nil},
		{"[[:alpha:]]+", ":]", []int{0}},
		{"[[:alpha:]]+", "[]", []int{0}},
		{"[[:alpha:]]+", "p]", []int{0}},
		{"[[:alpha:]]+", "z]", nil},
		{"[[:digit:]]+", "a]", nil},
		{"[[:digit:]]+", "d]", []int{0}},
		// A class holding no bracket at all is unaffected.
		{"[a-c]+", "abc", []int{0, 1, 2}},
		{"[^a-z]+", "ab1", []int{2}},
		// A bracket already written out as itself is left as it was.
		{`[\[a]+`, "[a", []int{0, 1}},
		{`[\]a]+`, "]a", []int{0, 1}},
	}
	for _, c := range cases {
		t.Run(c.pattern+" over "+c.data, func(t *testing.T) {
			src := `rule R { strings: $a = /` + c.pattern + `/ condition: $a }`
			if c.want == nil {
				if got := scan(t, src, c.data); len(got) != 0 {
					t.Errorf("the rule held, want it not to")
				}
				return
			}
			if got := matchPlaces(t, src, c.data); fmt.Sprint(got) != fmt.Sprint(c.want) {
				t.Errorf("found at %v, want %v", got, c.want)
			}
		})
	}
}

// TestWideWordEdges covers the marks asking for the edge of a word in a pattern
// looked for wide. The edge there falls between wide characters, not between
// single bytes, so a null sitting before a letter is not an edge. Each answer
// here is the one CyberChef gave for the same pattern.
func TestWideWordEdges(t *testing.T) {
	w := func(s string) string {
		var out []byte
		for i := range len(s) {
			out = append(out, s[i], 0)
		}
		return string(out)
	}
	cases := []struct {
		name    string
		str     string
		data    string
		want    []int
		lengths []int
	}{
		{"an edge at the front", `$a = /\bhello/ wide`, w("hellohello"), []int{0}, []int{10}},
		{"an edge at the back", `$a = /hello\b/ wide`, w("hellohello"), []int{10}, []int{10}},
		{"an edge at both ends", `$a = /\bhello\b/ wide`, w("hello"), []int{0}, []int{10}},
		{"an edge either side of a word", `$a = /\bhello\b/ wide`, w("x hello y"), []int{4}, []int{10}},
		{"asking for no edge at the front", `$a = /\Bello/ wide`, w("hello"), []int{2}, []int{8}},
		{"asking for an edge where there is none", `$a = /\bello/ wide`, w("hello"), nil, nil},
		{"asking for no edge at the back", `$a = /hell\B/ wide`, w("hello"), []int{0}, []int{8}},
		{"an edge before a space", `$a = /hel\b/ wide`, w("hel lo"), []int{0}, []int{6}},
		// A mark in the middle has a character each side of it within the
		// pattern, so whether it holds is settled as the rule is read.
		{"an edge between two letters", `$a = /hel\blo/ wide`, w("hello"), nil, nil},
		{"no edge between two letters", `$a = /hel\Blo/ wide`, w("hello"), []int{0}, []int{10}},
		// Asked for both widths, the plain form keeps Go's own reading, which is
		// right for single bytes.
		{
			"both widths at once", `$a = /\bhello/ wide ascii`, w("hello") + "hello",
			[]int{0, 10},
			[]int{10, 5},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			src := `rule R { strings: ` + c.str + ` condition: any of them }`
			if c.want == nil {
				if got := scan(t, src, c.data); len(got) != 0 {
					t.Errorf("the rule held, want it not to")
				}
				return
			}
			set, err := Parse(src)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			rules, err := Compile(set)
			if err != nil {
				t.Fatalf("compile: %v", err)
			}
			results, _, err := rules.Scan([]byte(c.data))
			if err != nil {
				t.Fatalf("scan: %v", err)
			}
			if len(results) != 1 {
				t.Fatalf("%d rules held, want 1", len(results))
			}
			var got, lengths []int
			for _, m := range results[0].Matches {
				got, lengths = append(got, m.Offset), append(lengths, m.Length)
			}
			if fmt.Sprint(got) != fmt.Sprint(c.want) {
				t.Errorf("found at %v, want %v", got, c.want)
			}
			if fmt.Sprint(lengths) != fmt.Sprint(c.lengths) {
				t.Errorf("lengths %v, want %v", lengths, c.lengths)
			}
		})
	}
}

// TestWideWordEdgeThatCannotBeSettled covers a mark in the middle of a wide
// pattern whose neighbours are not plain characters. Whether it holds depends on
// what those neighbours matched, which is not known as the rule is read, so it
// is refused rather than answered wrongly.
func TestWideWordEdgeThatCannotBeSettled(t *testing.T) {
	for _, pattern := range []string{
		`h[a-z]\blo`, `h\d\blo`, `hel\b[a-z]o`, `hel\b\do`,
		// A character that repeats might match a letter or nothing at all.
		`hel\bl+o`, `hel\bl{2}o`,
	} {
		t.Run(pattern, func(t *testing.T) {
			set, err := Parse(`rule R { strings: $a = /` + pattern + `/ wide condition: $a }`)
			if err != nil {
				return // refused as it was read, which is answer enough
			}
			if _, err := compileStrings(set.Rules[0].Strings[0]); err == nil {
				t.Error("accepted a word edge that cannot be settled either way")
			}
		})
	}
}

// TestWideRegexThatCannotBeRewritten covers a pattern that survives having its
// word-edge marks taken out but still cannot be rewritten to be looked for wide.
// It is built here rather than written in a rule, since the reader would not let
// such a pattern through.
func TestWideRegexThatCannotBeRewritten(t *testing.T) {
	str := &String{
		ID: "$a", Kind: stringRegex, Text: `[a\`,
		Mods: Modifiers{Wide: true},
	}
	if _, err := compileStrings(str); err == nil {
		t.Error("accepted a pattern that cannot be looked for wide")
	}
}
