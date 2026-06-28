package core

import "testing"

func TestEncodeURIFragment(t *testing.T) {
	cases := map[string]string{
		"aGVsbG8":   "aGVsbG8",   // base64-ish, all safe
		"To_Hex()":  "To_Hex()",  // parens and underscore are safe
		"a+b=c":     "a%2Bb%3Dc", // + and = stay escaped
		"x&y":       "x%26y",     // & stays escaped
		"a b":       "a%20b",     // space encoded
		"'/?:@,;!$": "'/?:@,;!$", // safe set kept literal
	}
	for in, want := range cases {
		if got := EncodeURIFragment(in); got != want {
			t.Errorf("EncodeURIFragment(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestBuildURL(t *testing.T) {
	r := Recipe{{Op: "To Hex", Args: []any{"Space"}}}
	got := BuildURL(r, []byte("hello"))
	want := "https://gchq.github.io/CyberChef/#recipe=To_Hex('Space')&input=aGVsbG8"
	if got != want {
		t.Fatalf("BuildURL = %q\nwant %q", got, want)
	}
}

func TestBuildURLNoInput(t *testing.T) {
	r := Recipe{{Op: "MD5"}}
	got := BuildURL(r, nil)
	want := "https://gchq.github.io/CyberChef/#recipe=MD5()"
	if got != want {
		t.Fatalf("BuildURL = %q\nwant %q", got, want)
	}
}
