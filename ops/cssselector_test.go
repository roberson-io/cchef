package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// cssRecipe builds a single-step CSS selector recipe with the given selector and
// delimiter arguments (mirroring CyberChef's arg order).
func cssRecipe(query, delim string) core.Recipe {
	return core.Recipe{{Op: "CSS selector", Args: []any{query, delim}}}
}

// CSS selector has no upstream fixture file (CyberChef wraps @xmldom/xmldom +
// nwmatcher, which have no CyberChef-side logic), so these cases are authoritative
// outputs captured from the CyberChef-server oracle. Selection and node.toString()
// serialization are reproduced from the oracle; the delimiter follows cchef's
// binaryShortString convention (parseEscapedChars: "\n" -> newline), matching the
// CyberChef browser and every other cchef delimiter op.
func TestCSSSelectorFixtures(t *testing.T) {
	const doc = `<html><body><p class="a">Hello</p><p>World</p><div>D</div></body></html>`
	runCases(t, []opCase{
		{
			"CSS selector: type, newline delim", doc,
			"<p class=\"a\">Hello</p>\n<p>World</p>",
			cssRecipe("p", `\n`),
		},
		{
			"CSS selector: class", doc,
			`<p class="a">Hello</p>`,
			cssRecipe(".a", "|"),
		},
		{
			"CSS selector: id",
			`<body><div id="x">A</div><div>B</div></body>`,
			`<div id="x">A</div>`,
			cssRecipe("#x", "|"),
		},
		{
			"CSS selector: attribute presence",
			`<body><a href="u1">1</a><a>2</a><a href="u2">3</a></body>`,
			`<a href="u1">1</a>|<a href="u2">3</a>`,
			cssRecipe("a[href]", "|"),
		},
		{
			"CSS selector: attribute equals",
			`<body><input type="text"><input type="button"></body>`,
			`<input type="text"/>`,
			cssRecipe("input[type=text]", "|"),
		},
		{
			"CSS selector: attribute prefix",
			`<div><a href="http://x">1</a><a href="ftp://y">2</a></div>`,
			`<a href="http://x">1</a>`,
			cssRecipe("a[href^=http]", "|"),
		},
		{
			"CSS selector: child combinator",
			`<div><p>1</p><span><p>2</p></span></div>`,
			`<p>1</p>`,
			cssRecipe("div > p", "|"),
		},
		{
			"CSS selector: descendant combinator",
			`<div><p>1</p><span><p>2</p></span></div>`,
			`<p>1</p>|<p>2</p>`,
			cssRecipe("div p", "|"),
		},
		{
			"CSS selector: group",
			`<div><p>1</p><span>2</span></div>`,
			`<p>1</p>|<span>2</span>`,
			cssRecipe("p, span", "|"),
		},
		{
			"CSS selector: nth-child",
			`<ul><li>a</li><li>b</li><li>c</li></ul>`,
			`<li>b</li>`,
			cssRecipe("li:nth-child(2)", "|"),
		},
		{
			"CSS selector: first-child",
			`<ul><li>a</li><li>b</li></ul>`,
			`<li>a</li>`,
			cssRecipe("li:first-child", "|"),
		},
		{
			"CSS selector: adjacent sibling",
			`<div><h1>H</h1><p>1</p><p>2</p></div>`,
			`<p>1</p>`,
			cssRecipe("h1 + p", "|"),
		},
		{
			"CSS selector: general sibling",
			`<div><h1>H</h1><p>1</p><p>2</p></div>`,
			`<p>1</p>|<p>2</p>`,
			cssRecipe("h1 ~ p", "|"),
		},
		{
			"CSS selector: case-insensitive type", `<DIV><P>x</P></DIV>`,
			`<P>x</P>`,
			cssRecipe("p", "|"),
		},
		{
			"CSS selector: void element self-closes",
			`<body><img src="a.png"></body>`,
			`<img src="a.png"/>`,
			cssRecipe("img", "|"),
		},
		{
			"CSS selector: value-less attribute expands",
			`<body><input disabled></body>`,
			`<input disabled="disabled"/>`,
			cssRecipe("input", "|"),
		},
		{
			"CSS selector: text entities escaped",
			`<div><p>a &lt; b &amp; c > d</p></div>`,
			`<p>a &lt; b &amp; c &gt; d</p>`,
			cssRecipe("p", "|"),
		},
		{
			"CSS selector: empty query yields empty", doc, "",
			cssRecipe("", "|"),
		},
		{
			"CSS selector: empty input yields empty", "", "",
			cssRecipe("p", "|"),
		},
		{
			"CSS selector: no match yields empty", `<p>x</p>`, "",
			cssRecipe("span", "|"),
		},
		{
			"CSS selector: only first root element is queried",
			`<span>2</span><p>1</p>`, "",
			cssRecipe("p", "|"),
		},
		{
			"CSS selector: attribute prefix negative",
			`<div><a href="ftp://y">2</a></div>`, "",
			cssRecipe("a[href^=http]", "|"),
		},
		{
			"CSS selector: attribute suffix",
			`<div><a href="a.com">1</a><a href="b.org">2</a></div>`,
			`<a href="a.com">1</a>`, cssRecipe("a[href$=com]", "|"),
		},
		{
			"CSS selector: attribute substring",
			`<div><a href="abcd">1</a></div>`, `<a href="abcd">1</a>`,
			cssRecipe("a[href*=bc]", "|"),
		},
		{
			"CSS selector: attribute whitespace-list",
			`<div><p class="a b c">y</p></div>`, `<p class="a b c">y</p>`,
			cssRecipe("p[class~=b]", "|"),
		},
		{
			"CSS selector: enumerated attribute value is case-insensitive",
			`<body><input type="TEXT"></body>`, `<input type="TEXT"/>`,
			cssRecipe("input[type=text]", "|"),
		},
		{
			"CSS selector: non-enumerated attribute value is case-sensitive",
			`<div><p data-x="ABC">y</p></div>`, "",
			cssRecipe("p[data-x=abc]", "|"),
		},
		{
			"CSS selector: class is case-sensitive",
			`<div><p class="foo">y</p></div>`, "",
			cssRecipe(".Foo", "|"),
		},
		{
			"CSS selector: general sibling",
			`<div><h1>H</h1><p>1</p><span>s</span><p>2</p></div>`,
			`<p>1</p>|<p>2</p>`, cssRecipe("h1 ~ p", "|"),
		},
		{
			"CSS selector: nth-child an+b",
			`<ul><li>1</li><li>2</li><li>3</li><li>4</li></ul>`,
			`<li>1</li>|<li>3</li>`, cssRecipe("li:nth-child(2n+1)", "|"),
		},
		{
			"CSS selector: negation",
			`<ul><li class="x">1</li><li>2</li></ul>`, `<li>2</li>`,
			cssRecipe("li:not(.x)", "|"),
		},
		{
			"CSS selector: union is document order",
			`<div><p>1</p><span>2</span><p>3</p></div>`,
			`<p>1</p>|<span>2</span>|<p>3</p>`, cssRecipe("span, p", "|"),
		},
		{
			"CSS selector: checked attribute never matches",
			`<body><input checked></body>`, "",
			cssRecipe("[checked]", "|"),
		},
		{
			"CSS selector: disabled attribute matches",
			`<body><input disabled></body>`, `<input disabled="disabled"/>`,
			cssRecipe("[disabled]", "|"),
		},
		{
			"CSS selector: dynamic pseudo matches nothing",
			`<body><input disabled></body>`, "",
			cssRecipe(":disabled", "|"),
		},
		{
			"CSS selector: empty ignores cdata",
			`<root><script><![CDATA[x]]></script></root>`,
			`<script><![CDATA[x]]></script>`, cssRecipe("script:empty", "|"),
		},
		{
			"CSS selector: empty excludes text",
			`<div><p>x</p><p></p></div>`, `<p/>`,
			cssRecipe("p:empty", "|"),
		},
		{
			"CSS selector: last-child",
			`<div><span>s</span><p>1</p><p>2</p></div>`, `<p>2</p>`,
			cssRecipe("p:last-child", "|"),
		},
		{
			"CSS selector: only-child",
			`<div><p>only</p></div>`, `<p>only</p>`,
			cssRecipe("p:only-child", "|"),
		},
		{
			"CSS selector: first-of-type",
			`<div><span>s</span><p>1</p><p>2</p></div>`, `<p>1</p>`,
			cssRecipe("p:first-of-type", "|"),
		},
		{
			"CSS selector: last-of-type",
			`<div><span>s</span><p>1</p><p>2</p></div>`, `<p>2</p>`,
			cssRecipe("p:last-of-type", "|"),
		},
		{
			"CSS selector: only-of-type",
			`<div><span>s</span><p>x</p></div>`, `<p>x</p>`,
			cssRecipe("p:only-of-type", "|"),
		},
		{
			"CSS selector: nth-last-child",
			`<div><p>1</p><p>2</p><p>3</p></div>`, `<p>3</p>`,
			cssRecipe("p:nth-last-child(1)", "|"),
		},
		{
			"CSS selector: nth-of-type",
			`<div><span>s</span><p>1</p><p>2</p><p>3</p></div>`, `<p>2</p>`,
			cssRecipe("p:nth-of-type(2)", "|"),
		},
		{
			"CSS selector: nth-child negative coefficient",
			`<ul><li>1</li><li>2</li><li>3</li></ul>`, `<li>1</li>|<li>2</li>`,
			cssRecipe("li:nth-child(-n+2)", "|"),
		},
		{
			"CSS selector: root",
			`<html><body>x</body></html>`, `<html><body>x</body></html>`,
			cssRecipe(":root", "|"),
		},
		{
			"CSS selector: not with type",
			`<div><p>1</p><span>2</span></div>`, `<span>2</span>`,
			cssRecipe("div :not(p)", "|"),
		},
	})
}

func TestCSSSelectorInvalid(t *testing.T) {
	_, err := runOp(t, "CSS selector", "<p>x</p>", "p[", `\n`)
	if err == nil {
		t.Fatal("expected error for malformed selector")
	}
	if !strings.HasPrefix(err.Error(), "Invalid CSS Selector. Details:") {
		t.Fatalf("unexpected error text: %v", err)
	}
}
