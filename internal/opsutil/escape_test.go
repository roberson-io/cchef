package opsutil

import "testing"

// TestParseEscapedChars covers the escape-sequence decoder's less-common arms
// (backslash, \x, \u, \u{...}) directly.
func TestParseEscapedChars(t *testing.T) {
	cases := map[string]string{
		`\\`:        "\\",
		`\x41`:      "A",
		`\u0041`:    "A",
		`\u{1F600}`: "\U0001F600",
		`\a`:        "\x07",
	}
	for in, want := range cases {
		if got := ParseEscapedChars(in); got != want {
			t.Errorf("ParseEscapedChars(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestExpandAlphRange(t *testing.T) {
	got := ExpandAlphRange("A-Za-z0-9+/=")
	want := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/="
	if got != want {
		t.Fatalf("ExpandAlphRange = %q\nwant %q", got, want)
	}
}

// TestEscapeWhitespace covers the control-character mapping into the Private
// Use Area, which keeps them visible in rendered output.
func TestEscapeWhitespace(t *testing.T) {
	if got := EscapeWhitespace("a\tb\x10"); got != "a\ue009b\ue010" {
		t.Errorf("EscapeWhitespace = %q", got)
	}
	if got := EscapeWhitespace("plain"); got != "plain" {
		t.Errorf("EscapeWhitespace left plain text as %q", got)
	}
}

// TestBytesAsLatin1 covers the always-Latin1 read: unlike BytesAsText, valid
// UTF-8 is still read one byte per character.
func TestBytesAsLatin1(t *testing.T) {
	if got := BytesAsLatin1([]byte{0xff, 'A'}); got != "\u00ffA" {
		t.Errorf("BytesAsLatin1 = %q", got)
	}
	// Two bytes that happen to be valid UTF-8 stay two characters.
	if got := BytesAsLatin1([]byte{0xc3, 0xbf}); got != "\u00c3\u00bf" {
		t.Errorf("BytesAsLatin1(UTF-8 bytes) = %q, want per-byte reading", got)
	}
}
