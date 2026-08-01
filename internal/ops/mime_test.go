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
// Invalid UTF-8 is decoded the way CyberChef's codepage library does, not
// replaced with U+FFFD. Verified against the oracle.
func TestMIMEDecodingInvalidUTF8(t *testing.T) {
	out, err := (MIMEDecoding{}).Run(sdish("=?UTF-8?B?/w==?="), nil)
	if err != nil {
		t.Fatal(err)
	}
	// cptable does not replace an invalid sequence; it reads the byte as the
	// start of a four-byte one and lands on an unassigned plane-15 code point.
	if want := "\U000C0000"; out.String() != want {
		t.Fatalf("invalid utf-8 = %q want %q", out.String(), want)
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

// TestMimeLocateWord documents the encoded-word locator extracted from
// mimeDecodeHeaders: it parses the =?charset?enc?text?= structure.
func TestMimeLocateWord(t *testing.T) {
	w, ok := mimeLocateWord([]rune("=?utf-8?B?aGVsbG8=?="))
	if !ok {
		t.Fatal("expected an encoded word")
	}
	if w.start != 0 || w.charset != "utf-8" || w.enc != 'B' ||
		string(w.text) != "aGVsbG8=" || w.end != 20 {
		t.Fatalf("got %+v", w)
	}

	if _, ok := mimeLocateWord([]rune("plain text, no words")); ok {
		t.Fatal("expected no encoded word")
	}
	// A "=?" with no closing "?=" is not a complete word.
	if _, ok := mimeLocateWord([]rune("=?utf-8?B?abc")); ok {
		t.Fatal("expected incomplete word to be rejected")
	}
}

// Every ISO-8859 part CyberChef accepts, decoded through the in-repo codepage
// engine. Verified against the oracle; part 11 (Thai) is the one x/text has no
// table for, and part 12 was never standardized.
func TestMIMEDecodingISO8859Family(t *testing.T) {
	runCases(t, []opCase{
		{"part 1", "Subject: =?ISO-8859-1?Q?=A1=C0=E9?=", "Subject: ¡Àé", mimeRecipe},
		{"part 2", "Subject: =?ISO-8859-2?Q?=A1=C0=E9?=", "Subject: ĄŔé", mimeRecipe},
		{"part 3", "Subject: =?ISO-8859-3?Q?=A1=C0=E9?=", "Subject: ĦÀé", mimeRecipe},
		{"part 4", "Subject: =?ISO-8859-4?Q?=A1=C0=E9?=", "Subject: ĄĀé", mimeRecipe},
		{"part 5", "Subject: =?ISO-8859-5?Q?=A1=C0=E9?=", "Subject: ЁРщ", mimeRecipe},
		{"part 6", "Subject: =?ISO-8859-6?Q?=A1=C0=E9?=", "Subject: ��ى", mimeRecipe},
		{"part 7", "Subject: =?ISO-8859-7?Q?=A1=C0=E9?=", "Subject: ‘ΐι", mimeRecipe},
		{"part 8", "Subject: =?ISO-8859-8?Q?=A1=C0=E9?=", "Subject: ��י", mimeRecipe},
		{"part 9", "Subject: =?ISO-8859-9?Q?=A1=C0=E9?=", "Subject: ¡Àé", mimeRecipe},
		{"part 10", "Subject: =?ISO-8859-10?Q?=A1=C0=E9?=", "Subject: ĄĀé", mimeRecipe},
		{"part 11 Thai", "Subject: =?ISO-8859-11?Q?=A1=C0=E9?=", "Subject: กภ้", mimeRecipe},
		{"part 13", "Subject: =?ISO-8859-13?Q?=A1=C0=E9?=", "Subject: ”Ąé", mimeRecipe},
		{"part 14", "Subject: =?ISO-8859-14?Q?=A1=C0=E9?=", "Subject: ḂÀé", mimeRecipe},
		{"part 15", "Subject: =?ISO-8859-15?Q?=A1=C0=E9?=", "Subject: ¡Àé", mimeRecipe},
		{"part 16", "Subject: =?ISO-8859-16?Q?=A1=C0=E9?=", "Subject: ĄÀé", mimeRecipe},
		// Base64-encoded words take the same decoding path.
		{"part 1 base64", "Subject: =?ISO-8859-1?B?wqk=?=", "Subject: Â©", mimeRecipe},
		{"utf-8", "Subject: =?utf-8?Q?caf=C3=A9?=", "Subject: café", mimeRecipe},
		{"us-ascii", "Subject: =?US-ASCII?Q?plain?=", "Subject: plain", mimeRecipe},
	})
}

// Part 12 was never standardized, so no codepage exists for it; a part outside
// the family is refused with CyberChef's own message.
func TestMIMEDecodingUnsupportedCharsets(t *testing.T) {
	for _, in := range []string{
		"Subject: =?ISO-8859-12?Q?=A1?=",
		"Subject: =?ISO-8859-99?Q?x?=",
		"Subject: =?WINDOWS-1252?Q?x?=",
	} {
		if _, err := runOp(t, "MIME Decoding", in); err == nil {
			t.Errorf("%q: expected an error", in)
		}
	}
}
