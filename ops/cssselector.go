package ops

import (
	"errors"
	"sort"
	"strings"

	"github.com/antchfx/xpath"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/opsutil"
	"github.com/roberson-io/cchef/internal/xmldom"
)

func init() {
	core.Register(CSSSelector{})
}

// CSSSelector extracts information from an HTML document with a CSS selector.
// Ported from CyberChef CSSSelector.mjs, which wraps @xmldom/xmldom (a lenient
// XML DOM parser) and nwmatcher (a CSS3 selector engine): the matched nodes are
// serialized via the DOM's node.toString() and joined by a delimiter. cchef
// reimplements the parser and serializer from scratch and evaluates selection by
// translating the CSS selector to XPath (cssToXPath) over the parsed tree via
// antchfx/xpath, reproducing xmldom+nwmatcher's exact output.
//
// Known minor divergences (all confined to inputs that essentially never occur
// in real, lowercase HTML, verified against the CyberChef-server oracle):
//   - Selectors that combine a mixed/upper-case type with a structural pseudo
//     (e.g. "P:first-child" against "<P>") hit an nwmatcher code path that is
//     case-sensitive for XML documents; cchef stays case-insensitive.
//   - Serializing a namespace-prefixed element in isolation does not re-emit the
//     ancestor's xmlns declaration.
//   - CSS identifier escapes (e.g. "x\:item") are unsupported and error.
type CSSSelector struct{}

// Meta returns the operation metadata.
func (CSSSelector) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "CSS selector",
		Module:      "Code",
		Description: "Extract information from an HTML document with a CSS selector.",
		InfoURL:     "https://wikipedia.org/wiki/Cascading_Style_Sheets#Selector",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (CSSSelector) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "CSS selector", Type: core.ArgString, Value: ""},
		{Name: "Delimiter", Type: core.ArgString, Value: `\n`},
	}
}

// Run selects nodes matching the CSS selector and joins their serializations.
func (CSSSelector) Run(in *core.Dish, args []any) (*core.Dish, error) {
	query := args[0].(string)
	delim := opsutil.ParseEscapedChars(args[1].(string))
	input := in.String()
	if query == "" || input == "" {
		return core.NewDish([]byte{}, core.TypeString), nil
	}
	nodes, err := selectNodes(xmldom.Parse(input), query)
	if err != nil {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError prefix
		return nil, errors.New("Invalid CSS Selector. Details:\n" + err.Error())
	}
	parts := make([]string, len(nodes))
	for i, n := range nodes {
		parts[i] = xmldom.Serialize(n)
	}
	return core.NewDish([]byte(strings.Join(parts, delim)), core.TypeString), nil
}

// selectNodes compiles the CSS selector to XPath and returns the matching
// element nodes in document order.
func selectNodes(doc *xmldom.Node, selector string) ([]*xmldom.Node, error) {
	xp, err := xmldom.CSSToXPath(selector)
	if err != nil {
		return nil, err
	}
	expr, err := xpath.Compile(xp)
	if err != nil {
		return nil, err
	}
	var nodes []*xmldom.Node
	iter := expr.Select(xmldom.NewNav(doc, false))
	for iter.MoveNext() {
		if nav, ok := iter.Current().(*xmldom.Nav); ok {
			nodes = append(nodes, nav.Cur)
		}
	}
	return docOrder(doc, nodes), nil
}

// docOrder deduplicates nodes and returns them in document (pre-order) order,
// matching nwmatcher's result ordering (a union XPath such as "a | b" yields
// per-operand order, which nwmatcher does not).
func docOrder(doc *xmldom.Node, nodes []*xmldom.Node) []*xmldom.Node {
	if len(nodes) < 2 {
		return nodes
	}
	index := xmldom.DocIndex(doc)
	seen := map[*xmldom.Node]bool{}
	out := make([]*xmldom.Node, 0, len(nodes))
	for _, n := range nodes {
		if !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	sort.Slice(out, func(a, b int) bool { return index[out[a]] < index[out[b]] })
	return out
}
