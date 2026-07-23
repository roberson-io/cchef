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
}

func newXMLNav(root *xmlNode) *xmlNav { return &xmlNav{cur: root, root: root, attr: -1} }

func (n *xmlNav) NodeType() xpath.NodeType {
	switch n.cur.typ {
	case xmlComment, xmlCData, xmlPI:
		// CDATA and PIs are exposed as non-text, non-element nodes so that
		// text() and :empty ignore them, matching nwmatcher (which keys on
		// nodeType 1/3); their character data still contributes to string-value
		// via collectText, so :contains keeps working.
		return xpath.CommentNode
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
