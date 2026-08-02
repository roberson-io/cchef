// Package xmldom is the XML document model the CSS-selector and XPath
// operations run against.
//
// It covers the subset of @xmldom/xmldom those operations rely on: [Parse]
// builds a tree that preserves element case and attribute order, [Serialize]
// writes it back, [NewNav] adapts it to the XPath evaluator, and [CSSToXPath]
// translates a CSS selector into the XPath that selects the same nodes.
package xmldom

import (
	"strconv"
	"strings"
)

// xmlNodeType enumerates the DOM node kinds cchef models from @xmldom/xmldom.
type xmlNodeType int

const (
	xmlDocument xmlNodeType = iota
	xmlElement
	xmlText
	xmlComment
	xmlCData
	xmlPI // processing instruction: name=target, data=remainder
)

// Attr is a single element attribute, preserving source order and case.
type Attr struct {
	Name  string
	Value string
}

// Node is a minimal DOM node reproducing the subset of @xmldom/xmldom that
// CSS selector / XPath expression rely on. Element case and attribute order are
// preserved exactly, matching xmldom's XML-mode parsing.
type Node struct {
	typ      xmlNodeType
	name     string // element tag name (original case)
	data     string // text / comment / cdata content
	Attrs    []Attr
	parent   *Node
	children []*Node
}

// Serialize reproduces @xmldom/xmldom node.toString() with isHTML=false.
func Serialize(n *Node) string {
	var b strings.Builder
	serializeNode(&b, n)
	return b.String()
}

func serializeNode(b *strings.Builder, n *Node) {
	switch n.typ {
	case xmlDocument:
		for _, c := range n.children {
			serializeNode(b, c)
		}
	case xmlElement:
		serializeElement(b, n)
	case xmlText:
		b.WriteString(escapeText(n.data))
	case xmlComment:
		b.WriteString("<!--")
		b.WriteString(n.data)
		b.WriteString("-->")
	case xmlCData:
		b.WriteString("<![CDATA[")
		b.WriteString(strings.ReplaceAll(n.data, "]]>", "]]]]><![CDATA[>"))
		b.WriteString("]]>")
	case xmlPI:
		b.WriteString("<?")
		b.WriteString(n.name)
		if n.data != "" {
			b.WriteByte(' ')
			b.WriteString(n.data)
		}
		b.WriteString("?>")
	}
}

func serializeElement(b *strings.Builder, n *Node) {
	b.WriteByte('<')
	b.WriteString(n.name)
	for _, a := range n.Attrs {
		b.WriteByte(' ')
		b.WriteString(a.Name)
		b.WriteString(`="`)
		b.WriteString(EscapeAttr(a.Value))
		b.WriteByte('"')
	}
	if len(n.children) == 0 {
		b.WriteString("/>")
		return
	}
	b.WriteByte('>')
	for _, c := range n.children {
		serializeNode(b, c)
	}
	b.WriteString("</")
	b.WriteString(n.name)
	b.WriteByte('>')
}

// escapeText mirrors xmldom's TEXT_NODE encoder: replace [<&>] with refs.
func escapeText(s string) string {
	return xmlEncode(s, "<&>")
}

// EscapeAttr mirrors xmldom's addSerializedAttribute: replace [<>&"\t\n\r].
func EscapeAttr(s string) string {
	return xmlEncode(s, "<>&\"\t\n\r")
}

// xmlEncode replaces each character of s that appears in set using xmldom's
// _xmlEncoder: named refs for <>&", numeric refs for everything else.
func xmlEncode(s, set string) string {
	if !strings.ContainsAny(s, set) {
		return s
	}
	var b strings.Builder
	for _, r := range s {
		if r < 128 && strings.ContainsRune(set, r) {
			b.WriteString(xmlEncoder(r))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func xmlEncoder(r rune) string {
	switch r {
	case '<':
		return "&lt;"
	case '>':
		return "&gt;"
	case '&':
		return "&amp;"
	case '"':
		return "&quot;"
	default:
		return "&#" + strconv.Itoa(int(r)) + ";"
	}
}

// DocIndex assigns each node a pre-order document position.
func DocIndex(doc *Node) map[*Node]int {
	index := map[*Node]int{}
	i := 0
	var walk func(n *Node)
	walk = func(n *Node) {
		index[n] = i
		i++
		for _, c := range n.children {
			walk(c)
		}
	}
	walk(doc)
	return index
}
