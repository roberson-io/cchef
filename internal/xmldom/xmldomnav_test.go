package xmldom

import (
	"testing"

	"github.com/antchfx/xpath"
)

// TestXMLNavCursor drives the NodeNavigator cursor directly over a small tree to
// exercise every movement, node-type and value method (some are reached only by
// XPath axes the CSS translator does not emit).
func TestXMLNavCursor(t *testing.T) {
	doc := Parse(`<r a="1" b="2"><x:c>hi</x:c><!--k--><![CDATA[cd]]>tail<?pi d?></r>`)
	nav := NewNav(doc, false)

	if nav.NodeType() != xpath.RootNode {
		t.Fatalf("root NodeType = %v", nav.NodeType())
	}
	if !nav.MoveToChild() { // -> <r>
		t.Fatal("MoveToChild to <r> failed")
	}
	if nav.NodeType() != xpath.ElementNode || nav.LocalName() != "r" {
		t.Fatalf("expected element r, got %v %q", nav.NodeType(), nav.LocalName())
	}
	if nav.Value() != "hicdtail" {
		t.Fatalf("element string-value = %q", nav.Value())
	}
	// Attributes.
	if !nav.MoveToNextAttribute() || nav.NodeType() != xpath.AttributeNode || nav.LocalName() != "a" || nav.Value() != "1" {
		t.Fatalf("first attribute wrong: %v %q %q", nav.NodeType(), nav.LocalName(), nav.Value())
	}
	if !nav.MoveToNextAttribute() || nav.LocalName() != "b" {
		t.Fatal("second attribute wrong")
	}
	if nav.MoveToNextAttribute() {
		t.Fatal("expected no third attribute")
	}
	// While positioned on an attribute, sibling/child moves are no-ops.
	if nav.MoveToChild() || nav.MoveToNext() || nav.MoveToPrevious() || nav.MoveToFirst() {
		t.Fatal("attribute cursor should not move to child/siblings")
	}
	if !nav.MoveToParent() || nav.NodeType() != xpath.ElementNode { // attr -> element
		t.Fatal("MoveToParent from attribute failed")
	}
	// Children: prefixed element, comment, cdata, text.
	if !nav.MoveToChild() || nav.LocalName() != "c" || nav.Prefix() != "x" {
		t.Fatalf("prefixed child wrong: %q prefix %q", nav.LocalName(), nav.Prefix())
	}
	if !nav.MoveToNext() || nav.NodeType() != xpath.CommentNode || nav.Value() != "k" {
		t.Fatalf("comment node wrong: %v %q", nav.NodeType(), nav.Value())
	}
	if !nav.MoveToNext() || nav.NodeType() != xpath.CommentNode { // cdata exposed as comment-type
		t.Fatal("cdata node type wrong")
	}
	if !nav.MoveToNext() || nav.NodeType() != xpath.TextNode || nav.Value() != "tail" {
		t.Fatalf("text node wrong: %v %q", nav.NodeType(), nav.Value())
	}
	if !nav.MoveToNext() || nav.NodeType() != xpath.AttributeNode || nav.Value() != "" {
		t.Fatalf("PI node should report AttributeNode with empty string-value, got %v %q", nav.NodeType(), nav.Value())
	}
	if nav.MoveToNext() {
		t.Fatal("expected no sibling after PI")
	}
	if !nav.MoveToPrevious() || nav.NodeType() != xpath.TextNode { // back to tail
		t.Fatal("MoveToPrevious failed")
	}
	if !nav.MoveToFirst() || nav.LocalName() != "c" {
		t.Fatalf("MoveToFirst should land on first child <c>, got %q", nav.LocalName())
	}
	if nav.MoveToFirst() { // already first
		t.Fatal("MoveToFirst on first child should return false")
	}
	// Copy / MoveTo.
	other := nav.Copy()
	nav.MoveToRoot()
	if nav.NodeType() != xpath.RootNode {
		t.Fatal("MoveToRoot failed")
	}
	if !nav.MoveTo(other) || nav.LocalName() != "c" {
		t.Fatal("MoveTo(copy) failed")
	}
	if nav.MoveTo(NewNav(Parse(`<z/>`), false)) {
		t.Fatal("MoveTo across different roots must fail")
	}
	nav.MoveToParent() // c -> r
	nav.MoveToParent() // r -> document
	if nav.MoveToParent() {
		t.Fatal("MoveToParent past the document root should fail")
	}
}
