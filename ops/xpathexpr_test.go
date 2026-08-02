package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// xpathRecipe builds a single-step XPath expression recipe.
func xpathRecipe(query, delim string) core.Recipe {
	return core.Recipe{{Op: "XPath expression", Args: []any{query, delim}}}
}

// XPath expression has no upstream fixture file (CyberChef wraps @xmldom/xmldom
// + the npm `xpath` library); these cases are authoritative outputs captured from
// the CyberChef-server oracle. The document is parsed as XML (same lenient parser
// as CSS selector) and each selected node is serialized via node.toString().
func TestXPathExpressionFixtures(t *testing.T) {
	const doc = `<r><a href="u1">one</a><a href="u2">two</a><b>x&amp;y</b></r>`
	runCases(t, []opCase{
		{
			"XPath: elements", doc,
			`<a href="u1">one</a>|<a href="u2">two</a>`,
			xpathRecipe("//a", "|"),
		},
		{
			"XPath: entity in text", doc,
			`<b>x&amp;y</b>`,
			xpathRecipe("//b", "|"),
		},
		{
			"XPath: attribute nodes (leading space)", doc,
			` href="u1"| href="u2"`,
			xpathRecipe("//@href", "|"),
		},
		{
			"XPath: positional predicate", doc,
			`<a href="u1">one</a>`,
			xpathRecipe("//a[1]", "|"),
		},
		{
			"XPath: no match yields empty", doc, "",
			xpathRecipe("//z", "|"),
		},
		{
			"XPath: text nodes", doc,
			`one|two`,
			xpathRecipe("//a/text()", "|"),
		},
		{
			"XPath: newline delimiter",
			`<r><a>1</a><a>2</a></r>`,
			"<a>1</a>\n<a>2</a>",
			xpathRecipe("//a", `\n`),
		},
		{
			"XPath: comment node",
			`<r><!-- hi --></r>`,
			`<!-- hi -->`,
			xpathRecipe("//comment()", "|"),
		},
		{
			"XPath: text matches cdata",
			`<r><![CDATA[a<b]]></r>`,
			`<![CDATA[a<b]]>`,
			xpathRecipe("//r/text()", "|"),
		},
		{
			"XPath: text node escaping",
			`<a>x &lt; y &amp; z > w</a>`,
			`x &lt; y &amp; z &gt; w`,
			xpathRecipe("//a/text()", "|"),
		},
		{
			"XPath: attribute node escaping",
			`<a t="x&amp;y&quot;z">1</a>`,
			` t="x&amp;y&quot;z"`,
			xpathRecipe("//a/@t", "|"),
		},
		{
			"XPath: only first root queried",
			`<a>1</a><b>2</b>`,
			`<a>1</a>`,
			xpathRecipe("//a", "|"),
		},
		{
			"XPath: multiple attributes of one element",
			`<a x="1" y="2"/>`,
			` x="1"| y="2"`,
			xpathRecipe("//a/@*", "|"),
		},
		{
			"XPath: union deduplicates in document order",
			`<r><a>1</a><a>2</a></r>`,
			`<a>1</a>|<a>2</a>`,
			xpathRecipe("//a | //a", "|"),
		},
		{
			"XPath: comment does not match XML declaration",
			`<?xml version="1.0"?><r><!--c--><a>1</a></r>`,
			`<!--c-->`,
			xpathRecipe("//comment()", "|"),
		},
		{
			"XPath: node() includes processing instructions",
			`<r><?pi d?><a>1</a></r>`,
			`<?pi d?>|<a>1</a>`,
			xpathRecipe("/r/node()", "|"),
		},
	})
}

func TestXPathExpressionErrors(t *testing.T) {
	cases := []struct {
		name, input, query, wantErr string
	}{
		{
			"number result", `<r><a/><a/></r>`, "count(//a)",
			"Invalid XPath. Details:\nCannot convert number to nodeset.",
		},
		{
			"string result", `<r><a>x</a></r>`, "string(//a)",
			"Invalid XPath. Details:\nCannot convert string to nodeset.",
		},
		{
			"boolean result", `<r><a/></r>`, "boolean(//a)",
			"Invalid XPath. Details:\nCannot convert boolean to nodeset.",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := runOp(t, "XPath expression", c.input, c.query, "|")
			if err == nil || err.Error() != c.wantErr {
				t.Fatalf("got err %v, want %q", err, c.wantErr)
			}
		})
	}
	if _, err := runOp(t, "XPath expression", "<r/>", "//[", "|"); err == nil ||
		!strings.HasPrefix(err.Error(), "Invalid XPath. Details:\n") {
		t.Fatalf("expected XPath parse error, got %v", err)
	}
}
