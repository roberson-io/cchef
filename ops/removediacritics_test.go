package ops

import "testing"

// Expected outputs recorded from CyberChef's Node API.

// TestRemoveDiacritics covers precomposed and combining accents, characters
// that carry no combining mark and so survive (ø, đ, ß), and text outside
// Latin entirely.
func TestRemoveDiacritics(t *testing.T) {
	cases := []struct {
		name, input, want string
	}{
		{"precomposed accents", "héllo wörld café", "hello world cafe"},
		{"stacked combining marks", "é à̂", "e a"},
		{"letters that are not accents", "ñ ç ø đ ß 日本 ي", "n c ø đ ß 日本 ي"},
		{"already plain", "plain text", "plain text"},
		{"empty", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := runOp(t, "Remove Diacritics", c.input)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}
