package core

import "testing"

func TestKebab(t *testing.T) {
	cases := map[string]string{
		"To Base64":     "to-base64",
		"From Base64":   "from-base64",
		"ROT13":         "rot13",
		"XOR":           "xor",
		"To Upper case": "to-upper-case",
	}
	for in, want := range cases {
		if got := Kebab(in); got != want {
			t.Errorf("Kebab(%q) = %q, want %q", in, got, want)
		}
	}
}
