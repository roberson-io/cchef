package yara

import "testing"

// TestAtomQualityScores works the score out by hand for each shape of atom, the
// way libyara's atoms.c does.
func TestAtomQualityScores(t *testing.T) {
	cases := []struct {
		name  string
		atom  []byte
		want  int
		notes string
	}{
		{"one letter", []byte("a"), letterScore + uniqueByteBonus, "18 + 2"},
		// A lone space is one distinct byte, and a common one, so it takes the
		// run penalty rather than the variety bonus: 12 - 10.
		{"one space", []byte(" "), commonByteScore - runPenalty, "12 - 10"},
		{"one digit", []byte("7"), otherByteScore + uniqueByteBonus, "20 + 2"},
		{"two different letters", []byte("ab"), 2*letterScore + 2*uniqueByteBonus, "36 + 4"},
		{"four letters, three distinct", []byte("hell"), 4*letterScore + 3*uniqueByteBonus, "72 + 6"},
		{
			"three bytes, one common",
			[]byte{0x00, 0x01, 0x02},
			commonByteScore + 2*otherByteScore + 3*uniqueByteBonus, "12 + 40 + 6",
		},
		{"four zeroes", []byte{0, 0, 0, 0}, 4*commonByteScore - 4*runPenalty, "48 - 40"},
		{
			"four of a byte that is not common", []byte("aaaa"),
			4*letterScore + uniqueByteBonus, "72 + 2",
		},
		{"nothing at all", nil, 0, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := rawAtomQuality(c.atom); got != c.want {
				t.Errorf("scored %d, want %d (%s)", got, c.want, c.notes)
			}
		})
	}
}

// TestAtomQualityIsOffset covers the lift libyara applies, which puts the best
// possible atom at the top of the range.
func TestAtomQualityIsOffset(t *testing.T) {
	best := []byte{0x01, 0x02, 0x03, 0x04}
	if got, want := atomQuality(best), qualityOffset+rawAtomQuality(best); got != want {
		t.Errorf("got %d, want %d", got, want)
	}
	if got := atomQuality(best); got != maxQuality {
		t.Errorf("the strongest atom scores %d, want %d", got, maxQuality)
	}
}

// TestBestAtom covers the window a scan would pick out of a longer string.
func TestBestAtom(t *testing.T) {
	cases := []struct{ literal, want string }{
		{"abc", "abc"},
		{"abcd", "abcd"},
		{"hello", "hell"},
		// "a777" and "7777" both score 82; libyara keeps the first it finds and
		// only replaces on a strictly better one, so the earlier window wins.
		{"aaaa7777", "a777"},
		{"    word", "word"},
	}
	for _, c := range cases {
		t.Run(c.literal, func(t *testing.T) {
			if got := string(bestAtom([]byte(c.literal))); got != c.want {
				t.Errorf("picked %q, want %q", got, c.want)
			}
		})
	}
}

// TestSlowsScanning covers the warning as the modifiers change it. A modifier
// does not just decorate a string: it decides what a scan actually looks for,
// so a string that is fine as written can be weak once it is searched for under
// every key or as wide characters. Each answer here is the one CyberChef gave
// for the same declaration.
func TestSlowsScanning(t *testing.T) {
	cases := []struct {
		src  string
		want bool
		why  string
	}{
		{`$a = "lo"`, false, "two letters score 40"},
		{`$a = "l"`, true, "one letter scores 20"},
		{`$a = "hello"`, false, "78"},
		{`$a = " "`, true, "one space scores 2"},
		{`$a = "ab"`, false, "40, just over"},
		{`$a = "hi"`, false, "40"},
		{`$a = "\x00"`, true, "2"},
		{`$a = "\x00\x01\x02"`, false, "58"},
		{`$a = "\x00\x00\x00\x00"`, true, "a run of a common byte, penalised to 8"},
		// Under xor the scan looks for the atom under every key in the range,
		// and the weakest of those is what counts.
		{`$a = "lo" xor`, true, "key 0x6c maps it onto 00 03"},
		{`$a = "lo" xor(0-4)`, false, "those five keys all leave letters"},
		{`$a = "lo" xor(1-1)`, false, ""},
		{`$a = "lo" xor(108-108)`, true, "the one bad key, on its own"},
		{`$a = "lo" xor(0-107)`, true, "key 0x4c maps the first byte onto a space"},
		{`$a = "lo" xor(109-255)`, true, "key 0x93 maps the first byte onto 0xff"},
		{`$a = "ab" xor`, true, ""},
		{`$a = "a" xor`, true, ""},
		// Four bytes are usually enough to survive any key, unless they are all
		// the same and so can all land on one common byte at once.
		{`$a = "hello" xor`, false, "no key can weaken four different bytes"},
		{`$a = "aaaa" xor`, true, "key 0x61 maps it onto four zeroes"},
		{`$a = "aaaaa" xor`, true, ""},
		{`$a = "lll" xor`, true, ""},
		{`$a = "aaaa" xor(0-96)`, true, "key 0x41 maps it onto four spaces"},
		// Wide replaces the atom before the keys are tried, and the zeroes
		// between the characters break up the run that made it weak.
		{`$a = "aaaa" xor wide`, false, "no key can hit the byte and the zero at once"},
		{`$a = "aaaa" xor ascii wide`, true, "the plain atom is kept as well"},
		{`$a = "lo" xor wide`, false, ""},
		{`$a = "lo" xor ascii wide`, true, ""},
		{`$a = "hi" wide`, false, ""},
		{`$a = "l" wide`, true, ""},
		{`$a = "hello" wide`, false, ""},
		{`$a = "hello" nocase`, false, ""},
		{`$a = "HELLO" nocase`, false, ""},
		// The kinds that are not plain text are judged on the fixed run they
		// offer, which the modifiers above do not apply to.
		{`$a = { 68 ?? 6c }`, true, ""},
		{`$a = { 6c 6c 6f }`, false, ""},
		{`$a = /l{2}/`, false, "the count writes the character out twice"},
		{`$a = /[a-z]+/`, true, "a class offers nothing to look for"},
		{`$a = /hello/`, false, ""},
	}
	for _, c := range cases {
		t.Run(c.src, func(t *testing.T) {
			rule := parseOne(t, `rule R { strings: `+c.src+` condition: $a }`)
			if got := slowsScanning(rule.Strings[0]); got != c.want {
				t.Errorf("got %v, want %v (%s)", got, c.want, c.why)
			}
		})
	}
}

// TestRegexPiecesThatRunOut covers a regular expression that stops in the
// middle of a class or an escape. Go's own regexp refuses these before a rule
// gets this far, so they are reached directly: the run has to come to an end
// rather than read past what it was given.
func TestRegexPiecesThatRunOut(t *testing.T) {
	t.Run("a class that is never closed", func(t *testing.T) {
		if got := skipClass("a[bc", 1); got != len("a[bc") {
			t.Errorf("stopped at %d, want the end at %d", got, len("a[bc"))
		}
	})
	t.Run("a backslash with nothing after it", func(t *testing.T) {
		if _, next, literal := regexEscape(`ab\`, 2); literal || next != 3 {
			t.Errorf("got next %d literal %v, want 3 and false", next, literal)
		}
	})
	t.Run("hex digits that are not there", func(t *testing.T) {
		for _, body := range []string{`\x`, `\xg1`, `\x4`} {
			if _, _, literal := regexEscape(body, 0); literal {
				t.Errorf("%q was read as a byte", body)
			}
		}
	})
}

// TestLiteralRun covers the stretch of fixed bytes each kind of string offers,
// which is what the scan would look for and so what the warning is judged on.
func TestLiteralRun(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{"plain text", `$a = "hello"`, "hello"},
		{"hex, all fixed", `$a = { 68 65 6c }`, "hel"},
		{"hex broken by a wildcard", `$a = { 68 ?? 6c 6f 21 }`, "lo!"},
		{"hex broken by a jump", `$a = { 68 65 [2] 6c 6f 21 }`, "lo!"},
		{"hex broken by a choice", `$a = { 68 ( 65 | 66 ) 6c 6f 21 }`, "lo!"},
		{"a regular expression, all plain", `$a = /hello/`, "hello"},
		{"a regex broken by a repeat", `$a = /ab+cdef/`, "cdef"},
		{"a regex broken by a class", `$a = /ab[cd]efgh/`, "efgh"},
		{"a regex broken by an escape", `$a = /ab\dcdef/`, "cdef"},
		// Brackets only group, so they do not break a run of letters up.
		{"a regex grouped part way through", `$a = /(ab)cdef/`, "abcdef"},
		// A class offers nothing to look for, however wide its span, and what
		// is inside it is not text to be searched for.
		{"a regex that is only a class", `$a = /[a-z]+/`, ""},
		{"a class with a closing bracket in it", `$a = /[]]hello/`, "hello"},
		{"a class that excludes", `$a = /[^abc]hello/`, "hello"},
		{"an escaped bracket inside a class", `$a = /[a\]b]hello/`, "hello"},
		// A count writes the character out as many times as it must appear.
		{"a fixed repeat", `$a = /l{2}/`, "ll"},
		{"a fixed repeat in the middle", `$a = /hell{2}o/`, "helllo"},
		{"a repeat with a range", `$a = /l{2,3}/`, "ll"},
		{"a repeat with no upper bound", `$a = /l{2,}/`, "ll"},
		{"a repeat that may not happen at all", `$a = /xl{0,2}/`, "x"},
		{"a repeat with no lower bound", `$a = /ab{,2}/`, "a"},
		{"a repeat with nothing before it", `$a = /[a-z]{4}/`, ""},
		// A brace that opens nothing countable is just a character.
		{"braces around something that is not a number", `$a = /a{x}b/`, "a{x}b"},
		{"a brace that is never closed", `$a = /ab{2/`, "ab{2"},
		{"a count that is never closed", `$a = /ab{2c/`, "ab{2c"},
		{"a closing brace on its own", `$a = /a}b/`, "a}b"},
		// A quantifier that allows none takes the character before it away;
		// one that insists on at least one leaves it where it is.
		{"a character that may be missing", `$a = /ab*cd/`, "cd"},
		{"a character that is optional", `$a = /ab?c/`, "a"},
		{"a character that repeats", `$a = /ab+c/`, "ab"},
		// An escape that stands for a byte is part of the run; one that stands
		// for a class of bytes is not.
		{"bytes written in hex", `$a = /\x68\x65\x6c\x6c/`, "hell"},
		{"an escaped backslash", `$a = /a\\b/`, `a\b`},
		{"an escaped full stop", `$a = /he\.lo/`, "he.lo"},
		{"an escaped newline", `$a = /a\nb/`, "a\nb"},
		{"an escaped tab", `$a = /a\tb/`, "a\tb"},
		{"an escaped carriage return", `$a = /a\rb/`, "a\rb"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rule := parseOne(t, `rule R { strings: `+c.src+` condition: $a }`)
			if got := string(literalRun(rule.Strings[0])); got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

// TestUnboundedDotShapes covers the shapes of pattern that do and do not repeat
// anything at all without an upper bound, including ones written so oddly that
// nothing closes them.
func TestUnboundedDotShapes(t *testing.T) {
	cases := map[string]bool{
		"h.*o":     true,
		"h.{2,}o":  true,
		"h.{,5}o":  false,
		"h.{2,5}o": false,
		"h.{2}o":   false,
		// Nothing follows the stop at all, so nothing repeats.
		"ho.": false,
		// A count that nothing closes is not a count.
		"h.{2": false,
	}
	for pattern, want := range cases {
		t.Run(pattern, func(t *testing.T) {
			if got := unboundedDot(pattern); got != want {
				t.Errorf("reads as %v, want %v", got, want)
			}
		})
	}
}

// TestGroupsDoNotBreakALiteralRun covers patterns whose letters are split up by
// brackets. Brackets only group, so the letters either side of them are still
// one run and the pattern is looked up by a run long enough not to slow a scan
// down. What warns and what does not was read off CyberChef itself.
func TestGroupsDoNotBreakALiteralRun(t *testing.T) {
	for _, pattern := range []string{
		"h(e(l(l)o))", "h(ello)", "(hello)", "h(el)lo", "((hello))",
		"h(e)l(l)o", "(ab)+c", "(ab)?cd", "(ab|cd)ef", "h(e(l))",
		"(a)(b)(c)", "(a)(b)", "x(y)", "(he)(ll)(o)", "((a)(b))(c)",
	} {
		t.Run(pattern, func(t *testing.T) {
			str := &String{ID: "$a", Kind: stringRegex, Text: pattern}
			if slowsScanning(str) {
				t.Error("said it would slow a scan down, want it not to")
			}
		})
	}
}
