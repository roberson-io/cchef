package xmldom

import "testing"

// parseXML reproduces @xmldom/xmldom's DOMParser.parseFromString(source) with no
// mimeType, i.e. XML mode (isHTML=false, XML_ENTITIES only). These cases assert
// parse+serialize round-trips against outputs captured from the CyberChef-server
// oracle (full documentElement serialization via the "*" selector's first match).
func TestParseXMLRoundTrip(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"unquoted attribute", `<a href=u>x</a>`, `<a href="u">x</a>`},
		{"single-quoted attribute", `<a href='u v'>x</a>`, `<a href="u v">x</a>`},
		{"value-less attribute", `<input disabled>`, `<input disabled="disabled"/>`},
		{"xml entities", `<p>a &amp; b &lt; c &gt; &quot; &apos;</p>`, `<p>a &amp; b &lt; c &gt; " '</p>`},
		{"decimal char ref", `<p>&#65;&#66;</p>`, `<p>AB</p>`},
		{"hex char ref", `<p>&#x41;&#x42;</p>`, `<p>AB</p>`},
		{"unknown entity kept literal", `<p>a&nbsp;b</p>`, `<p>a&amp;nbsp;b</p>`},
		{"self-closing element", `<div>a<br/>b</div>`, `<div>a<br/>b</div>`},
		{"cdata section", `<div><![CDATA[a<b]]></div>`, `<div><![CDATA[a<b]]></div>`},
		{"comment", `<div><!-- c -->x</div>`, `<div><!-- c -->x</div>`},
		{"doctype skipped", `<!DOCTYPE html><html><body>x</body></html>`, `<html><body>x</body></html>`},
		{"leading text dropped", `junk<r>x</r>`, `<r>x</r>`},
		{"whitespace in tag", `<a   href="u"   >x</a>`, `<a href="u">x</a>`},
		{"bare ampersand literal", `<p>a & b</p>`, `<p>a &amp; b</p>`},
		{"attribute case preserved", `<a HREF="u">z</a>`, `<a HREF="u">z</a>`},
		{"empty attribute value", `<a t="">z</a>`, `<a t="">z</a>`},
		{"only first root kept", `<span>2</span><p>1</p>`, `<span>2</span>`},
		{"processing instruction", `<?xml version="1.0"?><r>x</r>`, `<?xml version="1.0"?><r>x</r>`},
		{"pi inside element", `<r><?pi data?><a>x</a></r>`, `<r><?pi data?><a>x</a></r>`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Serialize(Parse(c.in)); got != c.want {
				t.Fatalf("parseXML(%q)\n got %q\nwant %q", c.in, got, c.want)
			}
		})
	}
}

// TestParseXMLRecovery covers lenient recovery matching the oracle: stray markup
// delimiters and unmatched close tags.
func TestParseXMLRecovery(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"stray lt is literal text", `<r>a < b</r>`, `<r>a &lt; b</r>`},
		{"unmatched close tag ignored", `<r>a</x>b</r>`, `<r>ab</r>`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Serialize(Parse(c.in)); got != c.want {
				t.Fatalf("parseXML(%q)\n got %q\nwant %q", c.in, got, c.want)
			}
		})
	}
}

// TestNormalizeLineEndings covers xmldom's XML 1.1 line-ending normalization.
// The U+0085 (NEL) and U+2028 (LINE SEPARATOR) inputs are built at runtime to
// keep the source ASCII.
func TestNormalizeLineEndings(t *testing.T) {
	nel := string(rune(0x85))
	ls := string(rune(0x2028))
	cases := []struct{ in, want string }{
		{"a\r\nb", "a\nb"},
		{"a\rb", "a\nb"},
		{"a" + nel + "b", "a\nb"},
		{"a" + ls + "b", "a\nb"},
		{"a\r" + nel + "b", "a\nb"},
		{"plain text", "plain text"},
	}
	for _, c := range cases {
		if got := normalizeLineEndings(c.in); got != c.want {
			t.Fatalf("normalizeLineEndings(%q) = %q want %q", c.in, got, c.want)
		}
	}
}

// TestParseXMLTruncatedMarkup exercises the unterminated-markup branches. Because
// a root with no matching close tag self-closes (fixSelfClosed), the truncated
// comment/CDATA is discarded and the root serializes empty, matching the oracle.
func TestParseXMLTruncatedMarkup(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"unclosed comment discarded", `<r>a<!-- no end`, `<r/>`},
		{"unclosed cdata discarded", `<r><![CDATA[x`, `<r/>`},
		{"attribute with unclosed quote", `<a t="v>x`, `<a t="v&gt;x"/>`},
		{"negative char ref left literal", `<p>&#-1;</p>`, `<p>&amp;#-1;</p>`},
		{"attribute value at EOF", `<a t=`, `<a t=""/>`},
		{"attributes at EOF", `<a `, `<a/>`},
		{"empty attribute name skipped", `<a =v="1">x</a>`, `<a v="1">x</a>`},
		{"trailing whitespace selector-side ok", `<r>x</r>  `, `<r>x</r>`},
		{"pi without close at document level", `<?x`, `<?x?>`},
		{"close tag without gt", `<r>x</r`, `<r>x</r>`},
		{"doctype without gt", `<!DOCTYPE`, ``},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Serialize(Parse(c.in)); got != c.want {
				t.Fatalf("parseXML(%q)\n got %q\nwant %q", c.in, got, c.want)
			}
		})
	}
}
