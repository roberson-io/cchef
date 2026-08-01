package ops

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

// xmlAttr is a single element attribute, preserving source order and case.
type xmlAttr struct {
	name  string
	value string
}

// xmlNode is a minimal DOM node reproducing the subset of @xmldom/xmldom that
// CSS selector / XPath expression rely on. Element case and attribute order are
// preserved exactly, matching xmldom's XML-mode parsing.
type xmlNode struct {
	typ      xmlNodeType
	name     string // element tag name (original case)
	data     string // text / comment / cdata content
	attrs    []xmlAttr
	parent   *xmlNode
	children []*xmlNode
}

// xmlSerialize reproduces @xmldom/xmldom node.toString() with isHTML=false.
func xmlSerialize(n *xmlNode) string {
	var b strings.Builder
	serializeNode(&b, n)
	return b.String()
}

func serializeNode(b *strings.Builder, n *xmlNode) {
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

func serializeElement(b *strings.Builder, n *xmlNode) {
	b.WriteByte('<')
	b.WriteString(n.name)
	for _, a := range n.attrs {
		b.WriteByte(' ')
		b.WriteString(a.name)
		b.WriteString(`="`)
		b.WriteString(escapeAttr(a.value))
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

// escapeAttr mirrors xmldom's addSerializedAttribute: replace [<>&"\t\n\r].
func escapeAttr(s string) string {
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
