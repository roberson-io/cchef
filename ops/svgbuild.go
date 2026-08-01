package ops

import (
	"strings"
)

// A small SVG element tree with a serialiser matching the one CyberChef's chart
// operations get from nodom: explicit closing tags, class and style emitted
// before any other attribute, and text content escaped for & < > only.
//
// Three deliberate deviations from nodom, so cchef emits valid, safe SVG that
// renders as a standalone file: D3's internal `__data__` bindings are never
// serialised as attributes (nodom leaks them, carrying unescaped input; upstream
// added the same fix to Series chart only), style text has no trailing "; ", and
// the root element carries the SVG namespace.

// svgNamespace is the SVG XML namespace. CyberChef omits it because its output
// is injected into an HTML document, where the namespace is implied; a
// standalone .svg file without it is not recognised as SVG and will not render.
const svgNamespace = "http://www.w3.org/2000/svg"

// svgAttr is one attribute, kept in the order it was first set.
type svgAttr struct{ name, value string }

// svgEl is an element in the SVG tree.
type svgEl struct {
	tag        string
	className  string
	styles     []svgAttr
	attrs      []svgAttr
	content    string
	hasContent bool
	children   []*svgEl
}

// newSVGEl creates a detached element.
func newSVGEl(tag string) *svgEl { return &svgEl{tag: tag} }

// append creates a child element and returns it.
func (e *svgEl) append(tag string) *svgEl {
	child := newSVGEl(tag)
	e.children = append(e.children, child)
	return child
}

// attr sets an attribute, replacing any existing value in place.
func (e *svgEl) attr(name, value string) *svgEl {
	for i := range e.attrs {
		if e.attrs[i].name == name {
			e.attrs[i].value = value
			return e
		}
	}
	e.attrs = append(e.attrs, svgAttr{name, value})
	return e
}

// class sets the element's class attribute.
func (e *svgEl) class(v string) *svgEl {
	e.className = v
	return e
}

// style sets one style property, replacing any existing value in place.
func (e *svgEl) style(name, value string) *svgEl {
	for i := range e.styles {
		if e.styles[i].name == name {
			e.styles[i].value = value
			return e
		}
	}
	e.styles = append(e.styles, svgAttr{name, value})
	return e
}

// text sets the element's text content.
func (e *svgEl) text(v string) *svgEl {
	e.content, e.hasContent = v, true
	return e
}

// render serialises the element and its subtree.
func (e *svgEl) render() string {
	var b strings.Builder
	e.renderTo(&b)
	return b.String()
}

// renderTo writes the element and its subtree to b.
func (e *svgEl) renderTo(b *strings.Builder) {
	b.WriteByte('<')
	b.WriteString(e.tag)
	if e.className != "" {
		b.WriteString(` class="`)
		b.WriteString(e.className)
		b.WriteByte('"')
	}
	if len(e.styles) > 0 {
		b.WriteString(` style="`)
		for i, s := range e.styles {
			if i > 0 {
				b.WriteString("; ")
			}
			b.WriteString(s.name)
			b.WriteString(": ")
			b.WriteString(s.value)
		}
		b.WriteByte('"')
	}
	for _, a := range e.attrs {
		b.WriteByte(' ')
		b.WriteString(a.name)
		b.WriteString(`="`)
		b.WriteString(a.value)
		b.WriteByte('"')
	}
	b.WriteByte('>')

	// Children take precedence over text content, as nodom's renderer does.
	switch {
	case len(e.children) > 0:
		for _, c := range e.children {
			c.renderTo(b)
		}
	case e.hasContent:
		b.WriteString(escapeSVGText(e.content))
	}

	b.WriteString("</")
	b.WriteString(e.tag)
	b.WriteByte('>')
}

// svgTextEscaper escapes the three characters nodom escapes in text nodes.
var svgTextEscaper = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
)

// escapeSVGText escapes element text content.
func escapeSVGText(s string) string { return svgTextEscaper.Replace(s) }
