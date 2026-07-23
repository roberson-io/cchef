package ops

import "testing"

// The serializer reproduces @xmldom/xmldom's node.toString() (isHTML=false):
//   - element with no children -> self-closing "<tag/>"
//   - element with children    -> "<tag>...children...</tag>"
//   - attribute value escapes [<>&"\t\n\r] via numeric/named refs
//   - text escapes [<&>]
//   - comment -> "<!--data-->", CDATA -> "<![CDATA[data]]>"
// See ../CyberChef/node_modules/@xmldom/xmldom/lib/dom.js serializeToString.

func el(name string, attrs []xmlAttr, children ...*xmlNode) *xmlNode {
	n := &xmlNode{typ: xmlElement, name: name, attrs: attrs}
	for _, c := range children {
		c.parent = n
		n.children = append(n.children, c)
	}
	return n
}
func txt(s string) *xmlNode    { return &xmlNode{typ: xmlText, data: s} }
func attr(n, v string) xmlAttr { return xmlAttr{name: n, value: v} }

func TestXMLSerialize(t *testing.T) {
	cases := []struct {
		name string
		node *xmlNode
		want string
	}{
		{"empty element self-closes", el("div", nil), "<div/>"},
		{"element with text child", el("p", nil, txt("hi")), "<p>hi</p>"},
		{
			"attribute preserved and quoted",
			el("p", []xmlAttr{attr("class", "a")}, txt("x")),
			`<p class="a">x</p>`,
		},
		{
			"void element with attr self-closes",
			el("img", []xmlAttr{attr("src", "a.png")}),
			`<img src="a.png"/>`,
		},
		{
			"case preserved",
			el("P", nil, txt("x")), "<P>x</P>",
		},
		{
			"nested elements",
			el("div", nil, el("p", nil, txt("x"), el("b", nil, txt("y")), txt("z"))),
			"<div><p>x<b>y</b>z</p></div>",
		},
		{
			"text escapes lt amp gt",
			el("p", nil, txt("a < b & c > d")),
			"<p>a &lt; b &amp; c &gt; d</p>",
		},
		{
			"attr value escapes quote amp lt gt",
			el("a", []xmlAttr{attr("t", `x&y"z<>`)}),
			`<a t="x&amp;y&quot;z&lt;&gt;"/>`,
		},
		{
			"attr value escapes whitespace",
			el("a", []xmlAttr{attr("t", "x\ty\nz\r")}),
			`<a t="x&#9;y&#10;z&#13;"/>`,
		},
		{
			"multiple attributes in order",
			el("input", []xmlAttr{attr("type", "text"), attr("name", "q")}),
			`<input type="text" name="q"/>`,
		},
		{
			"comment",
			&xmlNode{typ: xmlComment, data: "c"}, "<!--c-->",
		},
		{
			"cdata",
			&xmlNode{typ: xmlCData, data: "a<b"}, "<![CDATA[a<b]]>",
		},
		{
			"cdata with terminator split",
			&xmlNode{typ: xmlCData, data: "a]]>b"}, "<![CDATA[a]]]]><![CDATA[>b]]>",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := xmlSerialize(c.node); got != c.want {
				t.Fatalf("got %q want %q", got, c.want)
			}
		})
	}
}
