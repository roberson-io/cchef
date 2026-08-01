package ops

import (
	"strings"

	"github.com/antchfx/xpath"
)

// xmlNav is an xpath.NodeNavigator cursor over an xmlNode tree, letting the
// antchfx/xpath engine evaluate XPath (and CSS-translated-to-XPath) expressions
// against our xmldom-faithful DOM. LocalName is lowercased so element type and
// attribute name matching are case-insensitive, matching nwmatcher's HTML mode;
// serialization is unaffected as it reads xmlNode fields directly.
type xmlNav struct {
	cur  *xmlNode
	root *xmlNode
	attr int // -1 when positioned on the node itself, else index into cur.attrs
	// cdataAsText controls whether CDATA sections report as text nodes. The npm
	// `xpath` library (XPath expression) matches CDATA with text(), whereas
	// nwmatcher (CSS selector) treats CDATA as non-text so :empty ignores it.
	cdataAsText bool
}

func newXMLNav(root *xmlNode, cdataAsText bool) *xmlNav {
	return &xmlNav{cur: root, root: root, attr: -1, cdataAsText: cdataAsText}
}

func (n *xmlNav) NodeType() xpath.NodeType {
	switch n.cur.typ {
	case xmlCData:
		if n.cdataAsText {
			return xpath.TextNode
		}
		return xpath.CommentNode
	case xmlComment:
		return xpath.CommentNode
	case xmlPI:
		// antchfx/xpath has no processing-instruction node type (its exported
		// enum stops at CommentNode). Reporting AttributeNode keeps PIs out of
		// element/text/comment node tests (so //comment() does not match the XML
		// declaration) while node() still selects them; the attribute axis never
		// visits child PIs, so this has no other effect.
		return xpath.AttributeNode
	case xmlText:
		return xpath.TextNode
	case xmlDocument:
		return xpath.RootNode
	default: // xmlElement
		if n.attr != -1 {
			return xpath.AttributeNode
		}
		return xpath.ElementNode
	}
}

func (n *xmlNav) LocalName() string {
	if n.attr != -1 {
		return strings.ToLower(localPart(n.cur.attrs[n.attr].name))
	}
	return strings.ToLower(localPart(n.cur.name))
}

func (n *xmlNav) Prefix() string {
	name := n.cur.name
	if n.attr != -1 {
		name = n.cur.attrs[n.attr].name
	}
	if before, _, ok := strings.Cut(name, ":"); ok {
		return before
	}
	return ""
}

func (n *xmlNav) Value() string {
	if n.attr != -1 {
		return n.cur.attrs[n.attr].value
	}
	switch n.cur.typ {
	case xmlComment:
		return n.cur.data
	case xmlText, xmlCData:
		return n.cur.data
	case xmlElement, xmlDocument:
		var b strings.Builder
		collectText(&b, n.cur)
		return b.String()
	}
	return ""
}

func (n *xmlNav) Copy() xpath.NodeNavigator {
	c := *n
	return &c
}

func (n *xmlNav) MoveToRoot() { n.cur = n.root; n.attr = -1 }

func (n *xmlNav) MoveToParent() bool {
	if n.attr != -1 {
		n.attr = -1
		return true
	}
	if n.cur.parent != nil {
		n.cur = n.cur.parent
		return true
	}
	return false
}

func (n *xmlNav) MoveToNextAttribute() bool {
	if n.attr >= len(n.cur.attrs)-1 {
		return false
	}
	n.attr++
	return true
}

func (n *xmlNav) MoveToChild() bool {
	if n.attr != -1 {
		return false
	}
	if len(n.cur.children) > 0 {
		n.cur = n.cur.children[0]
		return true
	}
	return false
}

func (n *xmlNav) MoveToFirst() bool {
	if n.attr != -1 || n.cur.parent == nil {
		return false
	}
	first := n.cur.parent.children[0]
	if first == n.cur {
		return false
	}
	n.cur = first
	return true
}

func (n *xmlNav) MoveToNext() bool {
	if n.attr != -1 {
		return false
	}
	if sib := siblingAt(n.cur, +1); sib != nil {
		n.cur = sib
		return true
	}
	return false
}

func (n *xmlNav) MoveToPrevious() bool {
	if n.attr != -1 {
		return false
	}
	if sib := siblingAt(n.cur, -1); sib != nil {
		n.cur = sib
		return true
	}
	return false
}

func (n *xmlNav) MoveTo(other xpath.NodeNavigator) bool {
	o, ok := other.(*xmlNav)
	if !ok || o.root != n.root {
		return false
	}
	n.cur = o.cur
	n.attr = o.attr
	return true
}

// siblingAt returns the sibling delta positions from n within its parent, or nil.
func siblingAt(n *xmlNode, delta int) *xmlNode {
	if n.parent == nil {
		return nil
	}
	sibs := n.parent.children
	j := -1
	for i, c := range sibs {
		if c == n {
			j = i + delta
			break
		}
	}
	if j >= 0 && j < len(sibs) {
		return sibs[j]
	}
	return nil
}

// localPart strips any namespace prefix from a qualified name.
func localPart(name string) string {
	if _, after, ok := strings.Cut(name, ":"); ok {
		return after
	}
	return name
}

// collectText appends the concatenated text/cdata content of n's subtree.
func collectText(b *strings.Builder, n *xmlNode) {
	if n.typ == xmlText || n.typ == xmlCData {
		b.WriteString(n.data)
		return
	}
	for _, c := range n.children {
		collectText(b, c)
	}
}
