package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// HTML Entity has no upstream fixture file; these cases are verified against the
// CyberChef-server oracle. Args: convert-all, "Convert to" mode.
func TestToHTMLEntity(t *testing.T) {
	to := func(all bool, mode string) core.Recipe {
		return core.Recipe{{Op: "To HTML Entity", Args: []any{all, mode}}}
	}
	runCases(t, []opCase{
		{"default named", "a & b < c > \"d\" é", "a &amp; b &lt; c &gt; &quot;d&quot; &eacute;", to(false, "Named entities")},
		{"convert-all named", "a & b", "&#97;&#32;&amp;&#32;&#98;", to(true, "Named entities")},
		{"convert-all numeric", "AB é", "&#65;&#66;&#32;&#233;", to(true, "Numeric entities")},
		{"convert-all hex", "AB é", "&#x41;&#x42;&#x20;&#xe9;", to(true, "Hex entities")},
		{"numeric non-all", "A é €", "A &#233; &#8364;", to(false, "Numeric entities")},
		{"hex non-all", "A é €", "A &#xe9; &#x20ac;", to(false, "Hex entities")},
		{"astral code point", "😀", "&#128512;", to(false, "Named entities")},
	})
}

func TestFromHTMLEntity(t *testing.T) {
	from := core.Recipe{{Op: "From HTML Entity", Args: []any{}}}
	runCases(t, []opCase{
		// &eacute; is U+00E9, which fits in a byte, so it comes out as the one
		// byte 0xE9 rather than its two-byte UTF-8 form — the same bytes
		// CyberChef writes. Reading those bytes back as text gives "é" again.
		{"named entities", "a &amp; b &lt;c&gt; &quot;d&quot; &eacute;", "a & b <c> \"d\" \xe9", from},
		{"numeric, hex, and invalid", "&#65;&#x42;&#8364; plain &notreal; end", "AB€ plain &notreal; end", from},
		// The spec puts &epsi; at U+03B5, the ordinary epsilon; the lunate
		// U+03F5 is &epsiv;. CyberChef's old hand-written table had them
		// confused and wrote a stray comma into the name besides, which cchef
		// reproduced for parity until both were fixed upstream.
		{"epsi is the ordinary epsilon", "&epsi;,", "ε,", from},
		{"epsiv is the lunate epsilon", "&epsiv;", "ϵ", from},
	})
}

// --- direct tests for htmlEncodeRune, extracted from ToHTMLEntity.Run ---

// TestHTMLEncodeRune documents the per-rune encoding across modes and the
// convert-all flag, for a named char ('&'), a plain ASCII char ('A'), and a
// char above the Latin-1 range ('€').
func TestHTMLEncodeRune(t *testing.T) {
	cases := []struct {
		name                      string
		r                         rune
		convertAll, numeric, hexa bool
		want                      string
	}{
		{"named default", '&', false, false, false, "&amp;"},
		{"named numeric", '&', false, true, false, "&#38;"},
		{"named hex", '&', false, false, true, "&#x26;"},
		{"named convertAll", '&', true, false, false, "&amp;"},
		{"named convertAll+numeric", '&', true, true, false, "&#38;"},
		{"named convertAll+hex", '&', true, false, true, "&#x26;"},
		{"ascii default", 'A', false, false, false, "A"},
		{"ascii numeric", 'A', false, true, false, "A"},
		{"ascii hex", 'A', false, false, true, "A"},
		{"ascii convertAll", 'A', true, false, false, "&#65;"},
		{"supra default", '中', false, false, false, "&#20013;"},
		{"supra numeric", '中', false, true, false, "&#20013;"},
		{"supra hex", '中', false, false, true, "&#x4e2d;"},
		{"named supra default", '€', false, false, false, "&euro;"},
	}
	for _, c := range cases {
		if got := htmlEncodeRune(c.r, c.convertAll, c.numeric, c.hexa); got != c.want {
			t.Errorf("%s: got %q want %q", c.name, got, c.want)
		}
	}
}
