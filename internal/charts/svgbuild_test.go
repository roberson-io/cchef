package charts

import (
	"testing"
)

// Elements always get an explicit closing tag, and attributes keep the order
// they were set in — both needed to match CyberChef's serialised charts.
func TestSVGRenderBasic(t *testing.T) {
	svg := NewSVGEl("svg").Attr("width", "100%").Attr("viewBox", "0 0 500 500")
	svg.Append("rect").Attr("width", "470").Attr("height", "450")
	want := `<svg width="100%" viewBox="0 0 500 500"><rect width="470" height="450"></rect></svg>`
	if got := svg.Render(); got != want {
		t.Errorf("render =\n%s\nwant\n%s", got, want)
	}
}

// nodom emits class and style before any attribute, whatever order they were
// set in. CyberChef relies on this, so the port has to reproduce it.
func TestSVGRenderClassAndStyleComeFirst(t *testing.T) {
	el := NewSVGEl("g").Attr("transform", "translate(30, 20)").Class("axis axis--y")
	if got, want := el.Render(), `<g class="axis axis--y" transform="translate(30, 20)"></g>`; got != want {
		t.Errorf("render = %s, want %s", got, want)
	}

	txt := NewSVGEl("text").Attr("x", "-225").Attr("dy", "1em").Style("text-anchor", "middle")
	if got, want := txt.Render(), `<text style="text-anchor: middle" x="-225" dy="1em"></text>`; got != want {
		t.Errorf("render = %s, want %s", got, want)
	}
}

// Text content is escaped for &, < and > — the same three characters nodom
// escapes. Quotes are left alone, as they are harmless in element content.
func TestSVGRenderEscapesText(t *testing.T) {
	el := NewSVGEl("text").Text(`x"><script>a&b</script>`)
	want := `<text>x"&gt;&lt;script&gt;a&amp;b&lt;/script&gt;</text>`
	if got := el.Render(); got != want {
		t.Errorf("render = %s, want %s", got, want)
	}
}

// Newlines in tooltip text survive verbatim.
func TestSVGRenderTextNewlines(t *testing.T) {
	if got, want := NewSVGEl("title").Text("X: 1\nY: 2\n").Render(), "<title>X: 1\nY: 2\n</title>"; got != want {
		t.Errorf("render = %q, want %q", got, want)
	}
}

// An element with both text and children renders its children; CyberChef never
// mixes the two.
func TestSVGRenderNested(t *testing.T) {
	g := NewSVGEl("g")
	c := g.Append("circle").Attr("r", "5")
	c.Append("title").Text("hi")
	want := `<g><circle r="5"><title>hi</title></circle></g>`
	if got := g.Render(); got != want {
		t.Errorf("render = %s, want %s", got, want)
	}
}

// Several style properties are joined with "; ", in the order set.
func TestSVGRenderMultipleStyles(t *testing.T) {
	el := NewSVGEl("text").Style("text-anchor", "middle").Style("fill", "red")
	if got, want := el.Render(), `<text style="text-anchor: middle; fill: red"></text>`; got != want {
		t.Errorf("render = %s, want %s", got, want)
	}
}

// Setting an attribute twice replaces the value and keeps its original slot.
func TestSVGAttrReplace(t *testing.T) {
	el := NewSVGEl("g").Attr("a", "1").Attr("b", "2").Attr("a", "3")
	if got, want := el.Render(), `<g a="3" b="2"></g>`; got != want {
		t.Errorf("render = %s, want %s", got, want)
	}
}

// Setting a style property twice replaces it in place.
func TestSVGStyleReplace(t *testing.T) {
	el := NewSVGEl("text").Style("fill", "red").Style("fill", "blue")
	if got, want := el.Render(), `<text style="fill: blue"></text>`; got != want {
		t.Errorf("render = %s, want %s", got, want)
	}
}
