package ops

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/internal/jsonval"
)

// A Handlebars renderer, written from scratch. CyberChef's Template operation
// calls the handlebars npm package; there is no maintained Go port of it, so
// this covers the part of the language a template can usefully be written in
// against JSON input: variables, paths, comments, the built-in block helpers,
// inline partials, and the whitespace rules that decide which newlines around a
// tag survive.
//
// What it does not cover is everything that needs a host language — custom
// helpers, subexpressions, block parameters and partial parameters — none of
// which CyberChef exposes a way to supply.

// hbNode is one piece of a parsed template.
type hbNode interface {
	render(out *strings.Builder, ctx *hbContext) error
}

// hbText is literal template text.
type hbText string

func (t hbText) render(out *strings.Builder, _ *hbContext) error {
	out.WriteString(string(t))
	return nil
}

// hbVariable writes a value from the current context, escaped unless the
// template asked for it raw.
type hbVariable struct {
	path   string
	escape bool
}

func (v hbVariable) render(out *strings.Builder, ctx *hbContext) error {
	out.WriteString(hbFormat(ctx.lookup(v.path), v.escape))
	return nil
}

// hbBlock is a block helper and the two pieces of template it can render: the
// body, and whatever follows an {{else}}.
type hbBlock struct {
	helper  string
	arg     string
	body    []hbNode
	inverse []hbNode
}

// hbPartialDef records an inline partial, which is defined where it appears and
// used by name later.
type hbPartialDef struct {
	name string
	body []hbNode
}

func (p hbPartialDef) render(_ *strings.Builder, ctx *hbContext) error {
	ctx.partials[p.name] = p.body
	return nil
}

// hbPartialUse renders a partial that was defined earlier.
type hbPartialUse struct {
	name   string
	indent string
}

func (p hbPartialUse) render(out *strings.Builder, ctx *hbContext) error {
	body, ok := ctx.partials[p.name]
	if !ok {
		//nolint:staticcheck,revive // Handlebars' verbatim error text
		return fmt.Errorf("The partial %s could not be found", p.name)
	}
	var inner strings.Builder
	if err := hbRenderAll(&inner, body, ctx); err != nil {
		return err
	}
	out.WriteString(hbIndentLines(inner.String(), p.indent))
	return nil
}

// hbIndentLines puts the given indentation in front of every line, which is how
// a partial written on an indented line keeps its shape.
func hbIndentLines(s, indent string) string {
	if indent == "" || s == "" {
		return s
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		// The run after a trailing newline is not a line of its own.
		if line == "" && i == len(lines)-1 {
			continue
		}
		lines[i] = indent + line
	}
	return strings.Join(lines, "\n")
}

// hbRenderAll renders a run of nodes in order.
func hbRenderAll(out *strings.Builder, nodes []hbNode, ctx *hbContext) error {
	for _, n := range nodes {
		if err := n.render(out, ctx); err != nil {
			return err
		}
	}
	return nil
}

// hbTemplate is a parsed template ready to be rendered against a value.
type hbTemplate struct{ nodes []hbNode }

// hbCompile reads a template.
func hbCompile(source string) (*hbTemplate, error) {
	tokens, err := hbLex(source)
	if err != nil {
		return nil, err
	}
	tokens = hbStripStandalone(tokens)

	p := &hbParser{tokens: tokens}
	nodes, err := p.parseUntil("")
	if err != nil {
		return nil, err
	}
	if p.pos < len(p.tokens) {
		return nil, fmt.Errorf("unexpected %s", p.tokens[p.pos].text)
	}
	return &hbTemplate{nodes: nodes}, nil
}

// render renders the template against a value.
func (t *hbTemplate) render(data any) (string, error) {
	ctx := &hbContext{
		value:    data,
		root:     data,
		partials: map[string][]hbNode{},
		locals:   map[string]any{},
	}
	var out strings.Builder
	if err := hbRenderAll(&out, t.nodes, ctx); err != nil {
		return "", err
	}
	return out.String(), nil
}

// hbEscape is the set of characters Handlebars replaces when writing a value
// into the output, and what it replaces each with.
var hbEscape = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
	`"`, "&quot;",
	"'", "&#x27;",
	"`", "&#x60;",
	"=", "&#x3D;",
)

// hbFormat writes one value the way Handlebars does. A value that is not there
// writes nothing; a number writes as JavaScript would write it.
func hbFormat(v any, escape bool) string {
	var s string
	switch value := v.(type) {
	case nil:
		return ""
	case string:
		s = value
	case bool:
		s = strconv.FormatBool(value)
	case float64:
		s = jsNum(value)
	case []any:
		parts := make([]string, len(value))
		for i, item := range value {
			parts[i] = hbFormat(item, false)
		}
		s = strings.Join(parts, ",")
	case jsonval.Object:
		s = "[object Object]"
	default:
		s = fmt.Sprint(value)
	}
	if escape {
		return hbEscape.Replace(s)
	}
	return s
}

// hbTruthy reports whether a value counts as present for {{#if}}. An empty
// list, an empty string, zero and false do not.
func hbTruthy(v any) bool {
	switch value := v.(type) {
	case nil:
		return false
	case bool:
		return value
	case string:
		return value != ""
	case float64:
		return value != 0
	case []any:
		return len(value) > 0
	case jsonval.Object:
		// An object is present whether or not it has any fields.
		return true
	}
	return true
}
