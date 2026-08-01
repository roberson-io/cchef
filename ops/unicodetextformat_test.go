package ops

import (
	"encoding/hex"
	"testing"
)

// ASCII expectations are byte-exact recordings from CyberChef's Node API. The
// multi-byte expectations are the corrected behaviour — upstream inserts the
// combining characters after every byte, splitting multi-byte characters into
// invalid UTF-8 (see the bug log) — so here each combining character follows a
// whole character.
func TestUnicodeTextFormat(t *testing.T) {
	cases := []struct {
		name, input string
		underline   bool
		strike      bool
		wantHex     string
	}{
		{"neither", "ab", false, false, "6162"},
		{"strikethrough", "ab", false, true, "61ccb662ccb6"},
		{"underline", "ab", true, false, "61ccb262ccb2"},
		{"both, strikethrough first", "abc", true, true, "61ccb6ccb262ccb6ccb263ccb6ccb2"},
		{"multi-byte characters kept whole", "é日", false, true, "c3a9ccb6e697a5ccb6"},
		{"an invalid byte is one unit", "\xff a", false, true, "ffccb620ccb661ccb6"},
		{"empty", "", true, true, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := runOp(t, "Unicode Text Format", c.input, c.underline, c.strike)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if hex.EncodeToString([]byte(got)) != c.wantHex {
				t.Errorf("got %x, want %s", got, c.wantHex)
			}
		})
	}
}
