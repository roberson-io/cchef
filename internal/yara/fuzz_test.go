package yara

import "testing"

// A YARA rule is source somebody else wrote, so the parser and compiler take
// hostile input by definition. Either may refuse a rule, but neither may
// crash on one.

func FuzzParse(f *testing.F) {
	for _, seed := range []string{
		"",
		"rule a { condition: true }",
		"rule a { strings: $s = \"abc\" condition: $s }",
		"rule a { strings: $h = { 41 42 ?? 43 } condition: $h }",
		"rule a { strings: $r = /ab+c/ nocase condition: $r }",
		"rule a : tag1 tag2 { meta: author = \"x\" condition: false }",
		"private global rule a { condition: filesize > 10 }",
		"rule a { condition: uint16(0) == 0x5a4d }",
		"rule a { strings: $s = \"x\" condition: #s > 2 and @s[1] < 100 }",
		"rule a { condition: for any i in (1..3) : ( i == 2 ) }",
		"import \"pe\"\nrule a { condition: pe.number_of_sections > 1 }",
		"rule a { condition: true",
		"rule { condition: }",
		"rule a { strings: $s = { 41 42 condition: $s }",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, src string) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Parse panicked on %q: %v", src, r)
			}
		}()
		set, err := Parse(src)
		if err != nil {
			return
		}
		// Anything that parsed must also survive compilation, which is the
		// only thing a parsed rule set is for.
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Compile panicked on %q: %v", src, r)
			}
		}()
		_, _ = Compile(set)
	})
}
