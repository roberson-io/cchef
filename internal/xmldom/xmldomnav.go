package xmldom

import (
	"strings"

	"github.com/antchfx/xpath"
)

// Nav is an xpath.NodeNavigator cursor over an xmlNode tree, letting the
// antchfx/xpath engine evaluate XPath (and CSS-translated-to-XPath) expressions
// against our xmldom-faithful DOM. LocalName is lowercased so element type and
// attribute name matching are case-insensitive, matching nwmatcher's HTML mode;
// serialization is unaffected as it reads xmlNode fields directly.
type Nav struct {
	Cur       *Node
	root      *Node
	AttrIndex int // -1 when positioned on the node itself, else index into cur.attrs
	// cdataAsText controls whether CDATA sections report as text nodes. The npm
	// `xpath` library (XPath expression) matches CDATA with text(), whereas
	// nwmatcher (CSS selector) treats CDATA as non-text so :empty ignores it.
	cdataAsText bool
}

// NewNav returns a cursor positioned at root. cdataAsText selects whether
// CDATA counts as text, which the two callers disagree about.
func NewNav(root *Node, cdataAsText bool) *Nav {
	return &Nav{Cur: root, root: root, AttrIndex: -1, cdataAsText: cdataAsText}
}

// NodeType reports the current node's kind in the evaluator's vocabulary.
func (n *Nav) NodeType() xpath.NodeType {
	switch n.Cur.typ {
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
		if n.AttrIndex != -1 {
			return xpath.AttributeNode
		}
		return xpath.ElementNode
	}
}

// LocalName returns the current element or attribute name without its
// prefix, lowercased so name tests are case-insensitive.
func (n *Nav) LocalName() string {
	if n.AttrIndex != -1 {
		return strings.ToLower(localPart(n.Cur.Attrs[n.AttrIndex].Name))
	}
	return strings.ToLower(localPart(n.Cur.name))
}

// Prefix returns the namespace prefix of the current element or attribute,
// or "" when it has none.
func (n *Nav) Prefix() string {
	name := n.Cur.name
	if n.AttrIndex != -1 {
		name = n.Cur.Attrs[n.AttrIndex].Name
	}
	if before, _, ok := strings.Cut(name, ":"); ok {
		return before
	}
	return ""
}

// Value returns the string value of the current position: an attribute's
// value, a text or comment node's content, or an element's text descendants
// concatenated.
func (n *Nav) Value() string {
	if n.AttrIndex != -1 {
		return n.Cur.Attrs[n.AttrIndex].Value
	}
	switch n.Cur.typ {
	case xmlComment:
		return n.Cur.data
	case xmlText, xmlCData:
		return n.Cur.data
	case xmlElement, xmlDocument:
		var b strings.Builder
		collectText(&b, n.Cur)
		return b.String()
	}
	return ""
}

// Copy returns an independent cursor at the same position.
func (n *Nav) Copy() xpath.NodeNavigator {
	c := *n
	return &c
}

// MoveToRoot returns the cursor to the document node.
func (n *Nav) MoveToRoot() { n.Cur = n.root; n.AttrIndex = -1 }

// MoveToParent steps to the owning element of an attribute, or to the parent
// node, reporting whether there was one.
func (n *Nav) MoveToParent() bool {
	if n.AttrIndex != -1 {
		n.AttrIndex = -1
		return true
	}
	if n.Cur.parent != nil {
		n.Cur = n.Cur.parent
		return true
	}
	return false
}

// MoveToNextAttribute steps to the next attribute of the current element,
// reporting whether there was one.
func (n *Nav) MoveToNextAttribute() bool {
	if n.AttrIndex >= len(n.Cur.Attrs)-1 {
		return false
	}
	n.AttrIndex++
	return true
}

// MoveToChild steps to the first child, reporting whether there was one.
// An attribute has no children.
func (n *Nav) MoveToChild() bool {
	if n.AttrIndex != -1 {
		return false
	}
	if len(n.Cur.children) > 0 {
		n.Cur = n.Cur.children[0]
		return true
	}
	return false
}

// MoveToFirst steps to the first sibling, reporting whether that moved the
// cursor.
func (n *Nav) MoveToFirst() bool {
	if n.AttrIndex != -1 || n.Cur.parent == nil {
		return false
	}
	first := n.Cur.parent.children[0]
	if first == n.Cur {
		return false
	}
	n.Cur = first
	return true
}

// MoveToNext steps to the following sibling, reporting whether there was one.
func (n *Nav) MoveToNext() bool {
	if n.AttrIndex != -1 {
		return false
	}
	if sib := siblingAt(n.Cur, +1); sib != nil {
		n.Cur = sib
		return true
	}
	return false
}

// MoveToPrevious steps to the preceding sibling, reporting whether there was
// one.
func (n *Nav) MoveToPrevious() bool {
	if n.AttrIndex != -1 {
		return false
	}
	if sib := siblingAt(n.Cur, -1); sib != nil {
		n.Cur = sib
		return true
	}
	return false
}

// MoveTo jumps to another cursor's position, which must be over the same
// document.
func (n *Nav) MoveTo(other xpath.NodeNavigator) bool {
	o, ok := other.(*Nav)
	if !ok || o.root != n.root {
		return false
	}
	n.Cur = o.Cur
	n.AttrIndex = o.AttrIndex
	return true
}

// siblingAt returns the sibling delta positions from n within its parent, or nil.
func siblingAt(n *Node, delta int) *Node {
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
func collectText(b *strings.Builder, n *Node) {
	if n.typ == xmlText || n.typ == xmlCData {
		b.WriteString(n.data)
		return
	}
	for _, c := range n.children {
		collectText(b, c)
	}
}
