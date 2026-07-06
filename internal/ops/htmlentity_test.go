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
		{"named entities", "a &amp; b &lt;c&gt; &quot;d&quot; &eacute;", "a & b <c> \"d\" é", from},
		{"numeric, hex, and invalid", "&#65;&#x42;&#8364; plain &notreal; end", "AB€ plain &notreal; end", from},
		{"epsi quirk decodes cleanly", "&epsi;,", "ϵ,", from},
	})
}
