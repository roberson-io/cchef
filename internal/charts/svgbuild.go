package charts

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

// SVGNamespace is the SVG XML namespace. CyberChef omits it because its output
// is injected into an HTML document, where the namespace is implied; a
// standalone .svg file without it is not recognised as SVG and will not render.
const SVGNamespace = "http://www.w3.org/2000/svg"

// svgAttr is one attribute, kept in the order it was first set.
type svgAttr struct{ name, value string }

// SVGEl is an element in the SVG tree.
type SVGEl struct {
	tag        string
	className  string
	styles     []svgAttr
	attrs      []svgAttr
	content    string
	hasContent bool
	children   []*SVGEl
}

// NewSVGEl creates a detached element.
func NewSVGEl(tag string) *SVGEl { return &SVGEl{tag: tag} }

// Append creates a child element and returns it.
func (e *SVGEl) Append(tag string) *SVGEl {
	child := NewSVGEl(tag)
	e.children = append(e.children, child)
	return child
}

// Attr sets an attribute, replacing any existing value in place.
func (e *SVGEl) Attr(name, value string) *SVGEl {
	for i := range e.attrs {
		if e.attrs[i].name == name {
			e.attrs[i].value = value
			return e
		}
	}
	e.attrs = append(e.attrs, svgAttr{name, value})
	return e
}

// Class sets the element's Class attribute.
func (e *SVGEl) Class(v string) *SVGEl {
	e.className = v
	return e
}

// Style sets one Style property, replacing any existing value in place.
func (e *SVGEl) Style(name, value string) *SVGEl {
	for i := range e.styles {
		if e.styles[i].name == name {
			e.styles[i].value = value
			return e
		}
	}
	e.styles = append(e.styles, svgAttr{name, value})
	return e
}

// Text sets the element's Text content.
func (e *SVGEl) Text(v string) *SVGEl {
	e.content, e.hasContent = v, true
	return e
}

// Render serialises the element and its subtree.
func (e *SVGEl) Render() string {
	var b strings.Builder
	e.RenderTo(&b)
	return b.String()
}

// RenderTo writes the element and its subtree to b.
func (e *SVGEl) RenderTo(b *strings.Builder) {
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
			c.RenderTo(b)
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
