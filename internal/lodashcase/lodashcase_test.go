package lodashcase

import (
	"testing"
	"unicode/utf8"
)

// TestUpperFirst directly covers upperFirst, including the empty-string guard that
// the case ops never reach (word lists never contain empty words).
func TestUpperFirst(t *testing.T) {
	cases := map[string]string{"": "", "foo": "Foo", "Bar": "Bar", "a": "A"}
	for in, want := range cases {
		if got := upperFirst(in); got != want {
			t.Errorf("upperFirst(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestDeburrCoversLatinRange asserts the invariant relied on by lodashDeburr: every
// code point reLatin matches is present in deburrLetters (so no character is
// silently dropped).
func TestDeburrCoversLatinRange(t *testing.T) {
	ranges := [][2]rune{{0xc0, 0xd6}, {0xd8, 0xf6}, {0xf8, 0xff}, {0x100, 0x17f}}
	for _, rg := range ranges {
		for r := rg[0]; r <= rg[1]; r++ {
			buf := make([]byte, utf8.RuneLen(r))
			utf8.EncodeRune(buf, r)
			if _, ok := deburrLetters[string(buf)]; !ok {
				t.Errorf("deburrLetters missing U+%04X (matched by reLatin)", r)
			}
		}
	}
}
