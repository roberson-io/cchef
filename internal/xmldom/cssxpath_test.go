package xmldom

import "testing"

func TestParseNth(t *testing.T) {
	cases := []struct {
		in      string
		a, b    int
		wantErr bool
	}{
		{"odd", 2, 1, false},
		{"even", 2, 0, false},
		{"3", 0, 3, false},
		{"n", 1, 0, false},
		{"-n", -1, 0, false},
		{"2n+1", 2, 1, false},
		{"2n-1", 2, -1, false},
		{"-n+3", -1, 3, false},
		{"+n+2", 1, 2, false},
		{"", 0, 0, true},
		{"x", 0, 0, true},
		{"2n+q", 0, 0, true},
		{"zn", 0, 0, true},
	}
	for _, c := range cases {
		a, b, err := parseNth(c.in)
		if (err != nil) != c.wantErr {
			t.Fatalf("parseNth(%q) err=%v wantErr=%v", c.in, err, c.wantErr)
		}
		if err == nil && (a != c.a || b != c.b) {
			t.Fatalf("parseNth(%q) = (%d,%d) want (%d,%d)", c.in, a, b, c.a, c.b)
		}
	}
}

func TestXPathLiteral(t *testing.T) {
	cases := []struct{ in, want string }{
		{"abc", "'abc'"},
		{"a'b", `"a'b"`},
		{`a"b`, `'a"b'`},
		{`a'b"c`, `concat('a',"'",'b"c')`},
	}
	for _, c := range cases {
		if got := xpathLiteral(c.in); got != c.want {
			t.Fatalf("xpathLiteral(%q) = %q want %q", c.in, got, c.want)
		}
	}
}

func TestCSSToXPathErrors(t *testing.T) {
	for _, sel := range []string{
		"",                 // empty
		"a,,b",             // empty group
		"p[",               // unterminated attribute
		"p:bogus",          // unsupported pseudo-class
		"p:nth-child(x)",   // bad nth argument
		"p&q",              // unexpected character
		"p:nth-child(",     // unterminated pseudo argument
		"a b[",             // error in a non-first compound (propagates)
		"a b:bogus",        // unsupported pseudo in later compound
		"a b:nth-child(z)", // bad nth in later compound
		":not(p[)",         // error inside :not()
		"p::bogus",         // pseudo-element then unsupported pseudo
	} {
		if _, err := CSSToXPath(sel); err == nil {
			t.Fatalf("cssToXPath(%q) expected error", sel)
		}
	}
}

func TestSplitCombinatorsGuards(t *testing.T) {
	// Whitespace-only input yields no steps (guarded directly since cssToXPath
	// rejects empty groups before reaching here).
	if _, err := splitCombinators("   "); err == nil {
		t.Fatal("expected error for whitespace-only selector")
	}
	// Trailing whitespace after the last compound must not panic.
	steps, err := splitCombinators("a b ")
	if err != nil || len(steps) != 2 {
		t.Fatalf("splitCombinators(\"a b \") = %v, %v", steps, err)
	}
}

func TestComplexToXPathWhitespace(t *testing.T) {
	if _, err := complexToXPath("   "); err == nil {
		t.Fatal("expected error for whitespace-only complex selector")
	}
}

func TestCSSToXPathValid(t *testing.T) {
	// A few representative translations to lock the contract.
	cases := map[string]string{
		"p":                     "//p",
		"*":                     "//*",
		".c":                    "//*[contains(concat(' ',normalize-space(@class),' '),' c ')]",
		"#i":                    "//*[@id='i']",
		"a > b":                 "//a/b",
		"a b":                   "//a//b",
		"a + b":                 "//a/following-sibling::*[1]/self::b",
		"a ~ b":                 "//a/following-sibling::b",
		"[disabled]":            "//*[@disabled]",
		"[checked]":             "//*[1=0]",
		"[data-x|=en]":          "//*[(@data-x='en' or starts-with(@data-x,'en-'))]",
		`[title="a,b"]`:         "//*[@title='a,b']",
		"[title='x']":           "//*[@title='x']",
		":not(*)":               "//*[not(self::*)]",
		"p:nth-last-of-type(1)": "//p[(count(following-sibling::p)+1)=1]",
	}
	for sel, want := range cases {
		got, err := CSSToXPath(sel)
		if err != nil || got != want {
			t.Fatalf("cssToXPath(%q) = %q, %v; want %q", sel, got, err, want)
		}
	}
}
