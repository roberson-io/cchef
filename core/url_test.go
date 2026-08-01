package core

import (
	"strings"
	"testing"
)

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
	got := BuildURL(DefaultBaseURL, r, []byte("hello"))
	want := "https://gchq.github.io/CyberChef/#recipe=To_Hex('Space')&input=aGVsbG8"
	if got != want {
		t.Fatalf("BuildURL = %q\nwant %q", got, want)
	}
}

func TestBuildURLNoInput(t *testing.T) {
	r := Recipe{{Op: "MD5"}}
	got := BuildURL(DefaultBaseURL, r, nil)
	want := "https://gchq.github.io/CyberChef/#recipe=MD5()"
	if got != want {
		t.Fatalf("BuildURL = %q\nwant %q", got, want)
	}
}

// TestBuildURLBase checks that a self-hosted instance replaces the public one
// and that the fragment is unchanged by it.
func TestBuildURLBase(t *testing.T) {
	r := Recipe{{Op: "To Hex", Args: []any{"Space"}}}
	for _, base := range []string{
		"https://cyberchef.internal.example/",
		"https://example.test/tools/cyberchef/",
		"http://localhost:8080/",
	} {
		got := BuildURL(base, r, []byte("hello"))
		if !strings.HasPrefix(got, base+"#recipe=") {
			t.Errorf("BuildURL(%q) = %q, want it to start with %q", base, got, base+"#recipe=")
		}
		if want := BuildURL(DefaultBaseURL, r, []byte("hello")); strings.TrimPrefix(got, base) != strings.TrimPrefix(want, DefaultBaseURL) {
			t.Errorf("the fragment differs between bases:\n %q\n %q", got, want)
		}
	}
}
