package ops

import (
	"bytes"
	"regexp"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(RenderMarkdown{})
}

// RenderMarkdown renders Markdown input as HTML. Ported from CyberChef
// RenderMarkdown.mjs, which uses markdown-it (with raw HTML disabled) plus
// highlight.js for fenced code blocks, wrapping the result in a div. cchef
// reimplements it over goldmark.
//
// Reduced fidelity, by design: goldmark is not markdown-it. The common Markdown
// surface (headings, emphasis, lists, links, code, blockquotes, linkify) matches
// byte-for-byte, but two areas differ and are not ported: syntax highlighting of
// fenced code blocks (CyberChef colours them with highlight.js; cchef emits the
// escaped code) and block-level raw HTML (markdown-it escapes it as text, cchef
// escapes it without the surrounding paragraph).
type RenderMarkdown struct{}

// Meta returns the operation metadata.
func (RenderMarkdown) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Render Markdown",
		Module:      "Code",
		Description: "Renders input Markdown as HTML.",
		InfoURL:     "https://wikipedia.org/wiki/Markdown",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (RenderMarkdown) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Autoconvert URLs to links", Type: core.ArgBoolean, Value: false},
		{Name: "Enable syntax highlighting", Type: core.ArgBoolean, Value: true},
		{Name: "Open links in new tab.", Type: core.ArgBoolean, Value: false},
	}
}

var mdLinkOpen = regexp.MustCompile(`<a href="([^"]*)"`)

// Run renders the Markdown input.
func (RenderMarkdown) Run(in *core.Dish, args []any) (*core.Dish, error) {
	convertLinks := args[0].(bool)
	openBlank := args[2].(bool)
	// args[1] (enable syntax highlighting) has no effect here: cchef does not port
	// highlight.js, so fenced code blocks are emitted without colouring.

	// markdown-it's default preset renders tables and strikethrough (but not task
	// lists), so enable just those two extensions here.
	exts := []goldmark.Extender{extension.Table, extension.Strikethrough}
	if convertLinks {
		exts = append(exts, extension.Linkify)
	}
	opts := []goldmark.Option{
		goldmark.WithExtensions(exts...),
		goldmark.WithRendererOptions(renderer.WithNodeRenderers(
			util.Prioritized(escapeRawHTMLRenderer{}, 1),
		)),
	}
	var buf bytes.Buffer
	if err := goldmark.New(opts...).Convert(in.Bytes(), &buf); err != nil {
		return nil, err
	}
	rendered := buf.String()
	if openBlank {
		rendered = mdLinkOpen.ReplaceAllString(rendered, `<a href="$1" target="_blank"`)
	}
	out := `<div style="font-family: var(--primary-font-family)">` + rendered + `</div>`
	return core.NewDish([]byte(out), core.TypeString), nil
}

// escapeRawHTMLRenderer overrides goldmark's default raw-HTML handling (which
// omits it) to escape it as text, matching markdown-it's html:false behaviour.
type escapeRawHTMLRenderer struct{}

func (escapeRawHTMLRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindHTMLBlock, renderEscapedHTMLBlock)
	reg.Register(ast.KindRawHTML, renderEscapedRawHTML)
	// markdown-it renders strikethrough with <s>, goldmark defaults to <del>.
	reg.Register(east.KindStrikethrough, renderStrikethrough)
}

func renderStrikethrough(w util.BufWriter, _ []byte, _ ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		_, _ = w.WriteString("<s>")
	} else {
		_, _ = w.WriteString("</s>")
	}
	return ast.WalkContinue, nil
}

func renderEscapedHTMLBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*ast.HTMLBlock)
	for i := 0; i < n.Lines().Len(); i++ {
		seg := n.Lines().At(i)
		_, _ = w.Write(util.EscapeHTML(seg.Value(source)))
	}
	if n.HasClosure() {
		_, _ = w.Write(util.EscapeHTML(n.ClosureLine.Value(source)))
	}
	return ast.WalkContinue, nil
}

func renderEscapedRawHTML(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkSkipChildren, nil
	}
	n := node.(*ast.RawHTML)
	for i := 0; i < n.Segments.Len(); i++ {
		seg := n.Segments.At(i)
		_, _ = w.Write(util.EscapeHTML(seg.Value(source)))
	}
	return ast.WalkSkipChildren, nil
}
