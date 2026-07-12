package ops

// Tests for the MIME Decoding operation (RFC 2047 encoded-word header decoding).
//
// The first ten cases are transcribed from CyberChef's fixture,
// ../CyberChef/tests/operations/tests/MIMEDecoding.mjs; the remainder were
// derived from the CyberChef-server oracle (charset edge cases, the unknown-
// encoding path, and the two error messages).

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

var mimeRecipe = core.Recipe{{Op: "MIME Decoding"}}

// TestMIMEDecodingFixtures transcribes CyberChef's MIMEDecoding.mjs cases.
func TestMIMEDecodingFixtures(t *testing.T) {
	runCases(t, []opCase{
		{"encoded comments", "(=?ISO-8859-1?Q?a?=)", "(a)", mimeRecipe},
		{"adjacent comment whitespace", "(=?ISO-8859-1?Q?a?= b)", "(a b)", mimeRecipe},
		{"adjacent single whitespace ignored", "(=?ISO-8859-1?Q?a?= =?ISO-8859-1?Q?b?=)", "(ab)", mimeRecipe},
		{"adjacent double whitespace ignored", "(=?ISO-8859-1?Q?a?=  =?ISO-8859-1?Q?b?=)", "(ab)", mimeRecipe},
		{"adjacent CRLF whitespace ignored", "(=?ISO-8859-1?Q?a?=\r\n =?ISO-8859-1?Q?b?=)", "(ab)", mimeRecipe},
		{
			"utf-8 multiple headers",
			"=?utf-8?q?=C3=89ric?= <eric@example.org>, =?utf-8?q?Ana=C3=AFs?= <anais@example.org>",
			"Éric <eric@example.org>, Anaïs <anais@example.org>", mimeRecipe,
		},
		{"utf-8 base64 non-ASCII", "Subject: =?UTF-8?B?Y2Fmw6k=?=", "Subject: café", mimeRecipe},
		{"utf-8 base64 CJK", "Subject: =?UTF-8?B?5pel5pys6Kqe?=", "Subject: 日本語", mimeRecipe},
		{"utf-8 base64 ASCII", "Subject: =?UTF-8?B?aGVsbG8=?=", "Subject: hello", mimeRecipe},
		{
			"ISO decoding multiline",
			"From: =?US-ASCII?Q?Keith_Moore?= <moore@cs.utk.edu>\nTo: =?ISO-8859-1?Q?Keld_J=F8rn_Simonsen?= <keld@dkuug.dk>\nCC: =?ISO-8859-1?Q?Andr=E9?= Pirard <PIRARD@vm1.ulg.ac.be>\nSubject: =?ISO-8859-1?B?SWYgeW91IGNhbiByZWFkIHRoaXMgeW8=?=\n=?ISO-8859-2?B?dSB1bmRlcnN0YW5kIHRoZSBleGFtcGxlLg==?=",
			"From: Keith Moore <moore@cs.utk.edu>\nTo: Keld Jørn Simonsen <keld@dkuug.dk>\nCC: André Pirard <PIRARD@vm1.ulg.ac.be>\nSubject: If you can read this you understand the example.",
			mimeRecipe,
		},
	})
}

// TestMIMEDecodingVectors covers oracle-verified edge cases: passthrough,
// charset variants, and the (quirky) unknown-encoding and unterminated paths.
func TestMIMEDecodingVectors(t *testing.T) {
	runCases(t, []opCase{
		{"no encoded words", "plain header text", "plain header text", mimeRecipe},
		{"empty input", "", "", mimeRecipe},
		{"us-ascii latin1 high byte", "=?US-ASCII?Q?=E9?=", "é", mimeRecipe},
		{"iso-8859-15 euro", "=?ISO-8859-15?Q?=A4?=", "€", mimeRecipe},
		{"iso-8859-2 base64", "=?ISO-8859-2?B?dSB1bmRlcnN0YW5k?=", "u understand", mimeRecipe},
		{"underscore is space", "=?UTF-8?Q?a_b?=", "a b", mimeRecipe},
		{"leading text before first word", "Hi =?UTF-8?Q?there?=", "Hi there", mimeRecipe},
		{"unknown encoding passthrough", "=?UTF-8?X?abc?=", "=?abc", mimeRecipe},
		{"unterminated encoded word", "=?UTF-8?Q?abc", "=?UTF-8?Q?abc", mimeRecipe},
		{"trailing text after word", "=?UTF-8?Q?hi?= there", "hi there", mimeRecipe},
		{"no question after charset", "=?UTF-8", "=?UTF-8", mimeRecipe},
		{"too short after charset", "=?U?", "=?U?", mimeRecipe},
		{"missing question after encoding", "=?a?bcdef?=", "=?a?bcdef?=", mimeRecipe},
		{"invalid hex contributes nothing", "=?UTF-8?Q?=GG?=", "", mimeRecipe},
		{"lowercase hex", "=?UTF-8?Q?=c3=a9?=", "é", mimeRecipe},
		{"iso number with trailing letters", "=?ISO-8859-1x?Q?a?=", "a", mimeRecipe},
	})
}

// TestMIMEByteArrayToUtf8 covers the Latin-1 fallback for input that is not
// valid UTF-8 (ported from Utils.byteArrayToUtf8).
func TestMIMEByteArrayToUtf8(t *testing.T) {
	if got := mimeByteArrayToUtf8([]byte{0xff, 0x41}); got != "ÿA" {
		t.Fatalf("byteArrayToUtf8 = %q want %q", got, "ÿA")
	}
}

// TestMIMEDecodingInvalidUTF8 pins cchef's handling of invalid UTF-8 inside a
// Base64 UTF-8 word: it substitutes U+FFFD, whereas CyberChef's cptable maps the
// bytes to a non-standard high code point. A documented divergence on malformed
// input.
func TestMIMEDecodingInvalidUTF8(t *testing.T) {
	out, err := (MIMEDecoding{}).Run(sdish("=?UTF-8?B?/w==?="), nil)
	if err != nil {
		t.Fatal(err)
	}
	if out.String() != "�" {
		t.Fatalf("invalid utf-8 = %q want %q", out.String(), "�")
	}
}

// TestMIMESliceHelpers directly exercises the JS-slice-semantics helpers whose
// clamping branches the decoder's call sites never reach (they always pass
// in-range, non-negative bounds).
func TestMIMESliceHelpers(t *testing.T) {
	r := []rune("abcdef")
	// mimeIndexOf empty needle returns 0.
	if got := mimeIndexOf(r, ""); got != 0 {
		t.Fatalf("mimeIndexOf empty = %d want 0", got)
	}
	// mimeSlice clamps a negative start, an over-long end, and an inverted range.
	if got := string(mimeSlice(r, -3, 3)); got != "abc" {
		t.Fatalf("mimeSlice(-3,3) = %q want %q", got, "abc")
	}
	if got := string(mimeSlice(r, 3, 100)); got != "def" {
		t.Fatalf("mimeSlice(3,100) = %q want %q", got, "def")
	}
	if got := mimeSlice(r, 4, 2); got != nil {
		t.Fatalf("mimeSlice(4,2) = %v want nil", got)
	}
	// mimeSliceFrom clamps a negative start and returns empty past the end.
	if got := string(mimeSliceFrom(r, -1)); got != "abcdef" {
		t.Fatalf("mimeSliceFrom(-1) = %q want %q", got, "abcdef")
	}
	if got := mimeSliceFrom(r, 10); got != nil {
		t.Fatalf("mimeSliceFrom(10) = %v want nil", got)
	}
}

// TestMIMEDecodingErrors covers the two OperationError paths.
func TestMIMEDecodingErrors(t *testing.T) {
	for _, in := range []string{
		"=?BADCHARSET?Q?a?=", // Unhandled Charset
		"=?UTF-8?Q?a=?=",     // Incorrectly Encoded Word ("=" with too few following chars)
		"=?UTF-8?Q?é?=",      // Incorrectly Encoded Word (non-ASCII char in a Q word)
	} {
		if _, err := (MIMEDecoding{}).Run(sdish(in), nil); err == nil {
			t.Fatalf("decode %q: expected error", in)
		}
	}
}
