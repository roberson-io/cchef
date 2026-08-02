package opsutil

import "testing"

// TestTextAsBytes pins the inverse of BytesAsText: characters that all fit in a
// byte are written one byte each, and anything wider makes the whole string
// UTF-8. This is what decides whether charcode 255 comes out as the byte 0xFF
// or as its two-byte UTF-8 form.
func TestTextAsBytes(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want []byte
	}{
		{"hello", []byte("hello")},
		{"ÿ", []byte{0xff}},
		{"ÿþ", []byte{0xff, 0xfe}},
		{"HÿI", []byte{'H', 0xff, 'I'}},
		{"€", []byte("€")}, // wider than a byte, so the whole string is UTF-8
		{"aÿ€", []byte("aÿ€")},
		{"", nil},
	} {
		got := TextAsBytes(tc.in)
		if string(got) != string(tc.want) {
			t.Errorf("TextAsBytes(%q) = % x, want % x", tc.in, got, tc.want)
		}
	}
}

// TestBytesAsText pins the reading direction: valid UTF-8 as itself, anything
// else one character per byte, so 0xFF arrives as U+00FF rather than U+FFFD.
func TestBytesAsText(t *testing.T) {
	for _, tc := range []struct {
		in   []byte
		want string
	}{
		{[]byte("hello"), "hello"},
		{[]byte{0xff}, "ÿ"},
		{[]byte{0xff, 0xfe}, "ÿþ"},
		{[]byte{'H', 0xff, 'I'}, "HÿI"},
		{[]byte{0xc3, 0xbf}, "ÿ"}, // valid UTF-8 for U+00FF, read as itself
		{[]byte("€"), "€"},
		{nil, ""},
	} {
		if got := BytesAsText(tc.in); got != tc.want {
			t.Errorf("BytesAsText(% x) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
