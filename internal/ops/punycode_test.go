package ops

// Tests for the To Punycode / From Punycode operations.
//
// CyberChef's Punycode operations are thin wrappers around the `punycode.js`
// library (v2.3.1), calling punycode.encode/decode (raw, non-IDN) and
// punycode.toASCII/toUnicode (IDN). This is a from-scratch RFC 3492 port; there
// are no upstream fixture files, so every vector below was derived from that
// exact library used as an oracle (the CyberChef-server /bake endpoint). They
// are ordinary tests — edit as needed.

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

var (
	tpRaw = core.Recipe{{Op: "To Punycode", Args: []any{false}}}
	tpIDN = core.Recipe{{Op: "To Punycode", Args: []any{true}}}
	fpRaw = core.Recipe{{Op: "From Punycode", Args: []any{false}}}
	fpIDN = core.Recipe{{Op: "From Punycode", Args: []any{true}}}
)

// TestToPunycodeRaw covers raw (non-IDN) encoding: whole-string Punycode with a
// trailing delimiter for basic code points, across several scripts.
func TestToPunycodeRaw(t *testing.T) {
	runCases(t, []opCase{
		{"münchen", "münchen", "mnchen-3ya", tpRaw},
		{"bücher", "bücher", "bcher-kva", tpRaw},
		{"all-ASCII gets trailing dash", "abc", "abc-", tpRaw},
		{"single ASCII", "a", "a-", tpRaw},
		{"ASCII with hyphen", "Hello-World", "Hello-World-", tpRaw},
		{"snowman", "☃", "n3h", tpRaw},
		{"two snowmen", "☃☃", "n3ha", tpRaw},
		{"japanese", "日本語", "wgv71a119e", tpRaw},
		{"emoji (astral)", "😀", "e28h", tpRaw},
		{"greek", "αβγ", "mxacd", tpRaw},
		{"single non-ASCII", "ü", "tda", tpRaw},
		{"mixed", "façade", "faade-zra", tpRaw},
		{"greek word", "Ελληνικά", "twa0c6aifdar", tpRaw},
		{"cyrillic", "правда", "80aafi6cg", tpRaw},
		{"arabic", "العربية", "mgbcd4a2b0d2b", tpRaw},
		{"korean", "한국어", "3e0bk47br7k", tpRaw},
		{"alnum mixed", "a1b2ü", "a1b2-3ra", tpRaw},
		{"longer mixed", "münchenüber", "mnchenber-q9af", tpRaw},
	})
}

// TestToPunycodeIDN covers IDN encoding (toASCII): per-label xn-- prefixing,
// ASCII labels left intact, RFC 3490 separator normalisation, the email local
// part preserved, and punycode.js's quirk of keeping only the first two @-parts.
func TestToPunycodeIDN(t *testing.T) {
	runCases(t, []opCase{
		{"domain", "münchen.de", "xn--mnchen-3ya.de", tpIDN},
		{"single label", "münchen", "xn--mnchen-3ya", tpIDN},
		{"all-ASCII domain unchanged", "example.com", "example.com", tpIDN},
		{"snowman domain", "☃.net", "xn--n3h.net", tpIDN},
		{"japanese domain", "日本語.jp", "xn--wgv71a119e.jp", tpIDN},
		{"email local part preserved", "user@münchen.de", "user@xn--mnchen-3ya.de", tpIDN},
		{"multi-label ASCII", "a.b.c", "a.b.c", tpIDN},
		{"ideographic full stop separator", "münchen。de", "xn--mnchen-3ya.de", tpIDN},
		{"fullwidth full stop separator", "münchen．com", "xn--mnchen-3ya.com", tpIDN},
		{"multi-@ keeps first two parts", "a@b@münchen.de", "a@b", tpIDN},
		{"email with non-ASCII domain", "x@münchen.de", "x@xn--mnchen-3ya.de", tpIDN},
	})
}

// TestFromPunycodeRaw covers raw (non-IDN) decoding, the inverse of the encode
// vectors plus a couple of delimiter edge cases.
func TestFromPunycodeRaw(t *testing.T) {
	runCases(t, []opCase{
		{"münchen", "mnchen-3ya", "münchen", fpRaw},
		{"bücher", "bcher-kva", "bücher", fpRaw},
		{"trailing dash", "abc-", "abc", fpRaw},
		{"single", "a-", "a", fpRaw},
		{"snowman", "n3h", "☃", fpRaw},
		{"japanese", "wgv71a119e", "日本語", fpRaw},
		{"emoji", "e28h", "😀", fpRaw},
		{"greek", "mxacd", "αβγ", fpRaw},
		{"single non-ASCII", "tda", "ü", fpRaw},
		{"mixed", "faade-zra", "façade", fpRaw},
		{"uppercase extended digits", "mnchen-3YA", "münchen", fpRaw},
		{"xn-- delimiter edge", "xn--", "xn-", fpRaw},
		{"double delimiter", "--", "-", fpRaw},
	})
}

// TestFromPunycodeIDN covers IDN decoding (toUnicode): only xn-- labels are
// decoded (lower-cased first), the ACE prefix match is case-sensitive so an
// upper-case XN-- passes through untouched, and the email local part is kept.
func TestFromPunycodeIDN(t *testing.T) {
	runCases(t, []opCase{
		{"domain", "xn--mnchen-3ya.de", "münchen.de", fpIDN},
		{"single label", "xn--mnchen-3ya", "münchen", fpIDN},
		{"all-ASCII unchanged", "example.com", "example.com", fpIDN},
		{"snowman domain", "xn--n3h.net", "☃.net", fpIDN},
		{"japanese domain", "xn--wgv71a119e.jp", "日本語.jp", fpIDN},
		{"email local part preserved", "user@xn--mnchen-3ya.de", "user@münchen.de", fpIDN},
		{"uppercase label after prefix", "xn--MNCHEN-3ya", "münchen", fpIDN},
		{"uppercase prefix passes through", "XN--mnchen-3ya", "XN--mnchen-3ya", fpIDN},
	})
}

// TestPunycodeEmpty checks that empty input yields empty output in every mode
// (the oracle server rejects empty input, so these are exercised directly).
func TestPunycodeEmpty(t *testing.T) {
	for _, tc := range []struct {
		name string
		op   core.Operation
		idn  bool
	}{
		{"to raw", ToPunycode{}, false},
		{"to idn", ToPunycode{}, true},
		{"from raw", FromPunycode{}, false},
		{"from idn", FromPunycode{}, true},
	} {
		out, err := tc.op.Run(sdish(""), []any{tc.idn})
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if out.String() != "" {
			t.Fatalf("%s: got %q want empty", tc.name, out.String())
		}
	}
}

// TestFromPunycodeErrors covers the RFC 3492 decode failure modes: a non-basic
// code point in the literal portion, invalid digits, overflow of the internal
// integers, and decoding to a code point beyond the Unicode range (which
// punycode.js surfaces as a String.fromCodePoint RangeError).
func TestFromPunycodeErrors(t *testing.T) {
	bad := []string{
		"münchen",         // non-ASCII in the extended part
		"0",               // truncated generalized integer
		"-",               // invalid digit
		"ü-a",             // non-basic code point in the literal part
		"ïïï",             // non-ASCII throughout
		"999999999999x",   // overflow
		"zzzz",            // overflow
		"9zzz",            // overflow
		"a999999",         // overflow
		"99999999",        // overflow (digit exceeds the remaining integer width)
		"999999999999999", // overflow (longer digit run)
		"en32g",           // decodes to U+110000 (out of Unicode range)
		"8352o",           // decodes to U+200000 (out of Unicode range)
		"9016146o",        // overflow adding the decoded delta to n (near maxInt)
	}
	for _, in := range bad {
		if _, err := (FromPunycode{}).Run(sdish(in), []any{false}); err == nil {
			t.Fatalf("decode %q: expected error", in)
		}
	}
	// In IDN mode a malformed xn-- label propagates the decode error.
	for _, in := range []string{"xn--0", "xn--0.com", "good.xn--0"} {
		if _, err := (FromPunycode{}).Run(sdish(in), []any{true}); err == nil {
			t.Fatalf("toUnicode %q: expected error", in)
		}
	}
}

// TestPunycodeRoundTrip encodes then decodes each Unicode sample and checks it
// is unchanged.
func TestPunycodeRoundTrip(t *testing.T) {
	for _, s := range []string{
		"münchen", "bücher", "☃", "日本語", "😀", "αβγ", "façade",
		"правда", "한국어", "münchenüber", "a1b2ü",
	} {
		enc, err := (ToPunycode{}).Run(sdish(s), []any{false})
		if err != nil {
			t.Fatalf("encode %q: %v", s, err)
		}
		dec, err := (FromPunycode{}).Run(enc, []any{false})
		if err != nil {
			t.Fatalf("decode %q: %v", s, err)
		}
		if dec.String() != s {
			t.Fatalf("round-trip %q = %q", s, dec.String())
		}
	}
}
