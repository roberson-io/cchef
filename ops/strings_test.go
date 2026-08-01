package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

// stringsRecipe builds a recipe for the operation with the arguments given.
func stringsRecipe(encoding string, minLen float64, match string, total, sorted, unique bool) core.Recipe {
	return core.Recipe{{
		Op:   "Strings",
		Args: []any{encoding, minLen, match, total, sorted, unique},
	}}
}

// TestStringsCases covers the operation against the CyberChef-server oracle,
// which is the only authority available: CyberChef ships no fixtures for it.
func TestStringsCases(t *testing.T) {
	for _, tc := range []struct {
		name   string
		input  string
		recipe core.Recipe
		want   string
	}{
		{
			"runs shorter than the minimum are passed over",
			"abc defg hi jklmn",
			stringsRecipe(stringsSingleByte, 4, stringsPrintASCII, false, false, false),
			"abc defg hi jklmn",
		},
		{
			"a minimum of one takes every character",
			"a\x00bc",
			stringsRecipe(stringsSingleByte, 1, stringsPrintASCII, false, false, false),
			"a\nbc",
		},
		{
			"a tilde is not alphanumeric or punctuation",
			"abcd~efgh",
			stringsRecipe(stringsSingleByte, 4, stringsAlnumASCII, false, false, false),
			"abcd\nefgh",
		},
		{
			"a tilde is printable",
			"abcd~efgh",
			stringsRecipe(stringsSingleByte, 4, stringsPrintASCII, false, false, false),
			"abcd~efgh",
		},
		{
			"accented letters count as letters",
			"héllo wörld",
			stringsRecipe(stringsSingleByte, 4, stringsAlnumUnicode, false, false, false),
			"héllo wörld",
		},
		{
			"a currency sign is a symbol rather than punctuation",
			"abcd€efgh",
			stringsRecipe(stringsSingleByte, 4, stringsAlnumUnicode, false, false, false),
			"abcd\nefgh",
		},
		{
			"the printable set takes symbols",
			"abcd€efgh",
			stringsRecipe(stringsSingleByte, 4, stringsPrintUnicode, false, false, false),
			"abcd€efgh",
		},
		{
			"a total before them",
			"abcd efgh",
			stringsRecipe(stringsSingleByte, 4, stringsPrintASCII, true, false, false),
			"Total found: 1\n\nabcd efgh",
		},
		{
			"one of each",
			"abcd\x00abcd\x00wxyz",
			stringsRecipe(stringsSingleByte, 4, stringsPrintASCII, false, false, true),
			"abcd\nwxyz",
		},
		{
			"sorted",
			"zebra\x00apple\x00mango",
			stringsRecipe(stringsSingleByte, 4, stringsPrintASCII, false, true, false),
			"apple\nmango\nzebra",
		},
		{
			"nothing readable",
			"\x01\x02\x03",
			stringsRecipe(stringsSingleByte, 4, stringsPrintASCII, false, false, false),
			"",
		},
		{
			"a total when nothing was found",
			"\x01\x02\x03",
			stringsRecipe(stringsSingleByte, 4, stringsPrintASCII, true, false, false),
			"Total found: 0\n\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runCases(t, []opCase{{tc.name, tc.input, tc.want, tc.recipe}})
		})
	}
}

// stringsMixed holds ASCII text, then the same word stored two bytes to the
// character each way round, which is what the encodings tell apart.
const stringsMixed = "Hello\x00wor\x00Testing123!\x00" +
	"W\x00i\x00d\x00e\x00" + "\x00\x00" + "\x00B\x00E\x00w\x00i\x00d\x00e"

// TestStringsEncodings covers the four layouts a run of characters can have in
// the bytes. The expected values are the oracle's.
func TestStringsEncodings(t *testing.T) {
	for _, tc := range []struct{ name, encoding, want string }{
		{
			"one byte to the character", stringsSingleByte,
			"Hello\nTesting123!",
		},
		{
			"two bytes, low byte first", stringsUTF16LE,
			"!\x00W\x00i\x00d\x00e\x00\nB\x00E\x00w\x00i\x00d\x00",
		},
		{
			"two bytes, high byte first", stringsUTF16BE,
			"\x00W\x00i\x00d\x00e\n\x00B\x00E\x00w\x00i\x00d\x00e",
		},
		{
			// The any-width pattern allows a null on either side of each
			// character, so it runs through the one-byte text and the two-byte
			// text together rather than stopping between them.
			"either width", stringsAnyWidth,
			"Hello\x00wor\x00Testing123!\x00W\x00i\x00d\x00e\x00\n\x00B\x00E\x00w\x00i\x00d\x00e",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runCases(t, []opCase{{
				tc.name, stringsMixed, tc.want,
				stringsRecipe(tc.encoding, 4, stringsPrintASCII, false, false, false),
			}})
		})
	}
}

// TestStringsNullTerminated covers the two kinds that take the byte ending the
// run as part of it.
func TestStringsNullTerminated(t *testing.T) {
	for _, match := range []string{stringsNullASCII, stringsNullUnicode} {
		t.Run(match, func(t *testing.T) {
			runCases(t, []opCase{{
				match, stringsMixed, "Hello\x00\nTesting123!\x00",
				stringsRecipe(stringsSingleByte, 4, match, false, false, false),
			}})
		})
	}
}

// TestStringsDecodesInputAsCyberChefDoes covers the reading of the bytes. A run
// that is valid UTF-8 is read as UTF-8, so its accented letters are letters; a
// run that is not is read a character per byte, which turns the same bytes into
// two characters each. Both readings come from the oracle.
func TestStringsDecodesInputAsCyberChefDoes(t *testing.T) {
	runCases(t, []opCase{
		{
			"valid UTF-8 throughout",
			"café",
			"café",
			stringsRecipe(stringsSingleByte, 3, stringsAlnumUnicode, false, false, false),
		},
		{
			// The stray 0xef makes the whole input invalid UTF-8, so every byte
			// is read as its own character and the two bytes of the é become
			// "Ã" and "©" — of which only the first is a letter.
			"a stray byte makes the whole input Latin-1",
			"caf\xc3\xa9\x00n\xef",
			"cafÃ",
			stringsRecipe(stringsSingleByte, 3, stringsAlnumUnicode, false, false, false),
		},
	})
}

// TestStringsOptionsAreUngrouped covers the Match list. CyberChef's carries two
// headings naming the groups the choices fall into; selecting one builds a
// pattern that matches nothing useful, so cchef offers only the choices.
func TestStringsOptionsAreUngrouped(t *testing.T) {
	op, ok := core.Default.Get("Strings")
	if !ok {
		t.Fatal("Strings is not registered")
	}
	choices, _ := op.Args()[2].Value.([]string)
	if len(choices) != 6 {
		t.Errorf("offers %d choices, want 6: %v", len(choices), choices)
	}
	for _, c := range choices {
		if c == "[ASCII]" || c == "[Unicode]" {
			t.Errorf("%q is a heading rather than a choice", c)
		}
	}
}

// TestStringsUnusableLength covers a minimum length that is not a count. The
// repetition it is written into is then not a repetition at all, and both
// engines fall back to reading those characters literally — so the pattern looks
// for the text "{-1,}" after a printable character. Confirmed against the oracle.
func TestStringsUnusableLength(t *testing.T) {
	runCases(t, []opCase{{
		"a negative minimum length",
		"some text {-1,} here",
		" {-1,}",
		stringsRecipe(stringsSingleByte, -1, stringsPrintASCII, false, false, false),
	}})
}

// TestStringsZeroLength covers a minimum of none, which takes the whole input as
// one run since every position matches.
func TestStringsZeroLength(t *testing.T) {
	runCases(t, []opCase{{
		"a minimum of none",
		"some text {-1,} here",
		"some text {-1,} here\n",
		stringsRecipe(stringsSingleByte, 0, stringsPrintASCII, false, false, false),
	}})
}

// TestStringsCharacterSetUnknownKind covers a kind of run that is not one of the
// six offered. CyberChef's list carries two headings that fall here, and picking
// one leaves the pattern with no characters to repeat at all — which is why they
// are not offered.
func TestStringsCharacterSetUnknownKind(t *testing.T) {
	if got := stringsCharacterSet("[ASCII]"); got != "" {
		t.Errorf("got %q, want no characters", got)
	}
}

// TestStringsRejectsAnUnreadablePattern covers a set of characters that cannot
// be compiled, which is reported rather than silently finding nothing.
func TestStringsRejectsAnUnreadablePattern(t *testing.T) {
	if _, err := stringsPattern(stringsSingleByte, "[ASCII]", 4); err == nil {
		t.Error("a pattern with nothing to repeat was compiled")
	}
}

// TestStringsRejectsAnUnbuildableLength covers a minimum length too large to be
// a repetition count, which the operation reports rather than working around.
func TestStringsRejectsAnUnbuildableLength(t *testing.T) {
	_, err := runOp(t, "Strings", "some text",
		stringsSingleByte, 1e18, stringsPrintASCII, false, false, false)
	if err == nil {
		t.Error("an impossible repetition count was accepted")
	}
}
