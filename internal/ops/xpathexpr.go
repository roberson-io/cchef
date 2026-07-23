package ops

import (
	"errors"
	"sort"
	"strings"

	"github.com/antchfx/xpath"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(XPathExpression{})
}

// XPathExpression extracts information from an XML document with an XPath query.
// Ported from CyberChef XPathExpression.mjs, which wraps @xmldom/xmldom and the
// npm `xpath` library. cchef reuses the from-scratch xmldom parser/serializer and
// evaluates the query with antchfx/xpath over the parsed tree. Only node-set
// results are supported (as in the original): a number/string/boolean result is
// an error, matching the `xpath` library's "Cannot convert X to nodeset".
//
// Known divergence: antchfx/xpath has no processing-instruction node type (it
// compiles `processing-instruction()` to a match-all test), so that rarely-used
// node test returns every node instead of only PIs. All other node tests —
// including `comment()`, which correctly skips the XML declaration — match the
// oracle.
type XPathExpression struct{}

// Meta returns the operation metadata.
func (XPathExpression) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "XPath expression",
		Module:      "Code",
		Description: "Extract information from an XML document with an XPath query.",
		InfoURL:     "https://wikipedia.org/wiki/XPath",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (XPathExpression) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "XPath", Type: core.ArgString, Value: ""},
		{Name: "Result delimiter", Type: core.ArgString, Value: `\n`},
	}
}

// Run evaluates the XPath query and joins the serialized result nodes.
func (XPathExpression) Run(in *core.Dish, args []any) (*core.Dish, error) {
	query := args[0].(string)
	delim := parseEscapedChars(args[1].(string))

	expr, err := xpath.Compile(query)
	if err != nil {
		return nil, xpathError(err.Error())
	}
	doc := parseXML(in.String())
	result := expr.Evaluate(newXMLNav(doc, true))
	iter, ok := result.(*xpath.NodeIterator)
	if !ok {
		return nil, xpathError("Cannot convert " + xpathResultType(result) + " to nodeset")
	}

	var snaps []navSnap
	for iter.MoveNext() {
		if nav, ok := iter.Current().(*xmlNav); ok {
			snaps = append(snaps, navSnap{nav.cur, nav.attr})
		}
	}
	snaps = orderNavSnaps(doc, snaps)
	parts := make([]string, len(snaps))
	for i, s := range snaps {
		parts[i] = s.serialize()
	}
	return core.NewDish([]byte(strings.Join(parts, delim)), core.TypeString), nil
}

// navSnap captures a result node position: an element/text/comment/cdata node, or
// (when attr >= 0) a specific attribute of that node.
type navSnap struct {
	node *xmlNode
	attr int
}

// serialize reproduces node.toString() for the snapshot. Attribute nodes render
// as ` name="value"` (with the leading space), matching xmldom.
func (s navSnap) serialize() string {
	if s.attr != -1 {
		a := s.node.attrs[s.attr]
		return " " + a.name + `="` + escapeAttr(a.value) + `"`
	}
	return xmlSerialize(s.node)
}

// orderNavSnaps deduplicates snapshots and returns them in document order (the
// npm `xpath` library yields node-sets in document order; antchfx concatenates
// union operands). Attributes sort after their owning element by attribute index.
func orderNavSnaps(doc *xmlNode, snaps []navSnap) []navSnap {
	if len(snaps) < 2 {
		return snaps
	}
	index := buildDocIndex(doc)
	seen := map[navSnap]bool{}
	out := make([]navSnap, 0, len(snaps))
	for _, s := range snaps {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	sort.Slice(out, func(a, b int) bool {
		if ia, ib := index[out[a].node], index[out[b].node]; ia != ib {
			return ia < ib
		}
		return out[a].attr < out[b].attr
	})
	return out
}

// xpathError formats an error message the way CyberChef's operation does:
// "Invalid XPath. Details:\n<message>." (with the trailing period).
func xpathError(msg string) error {
	//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	return errors.New("Invalid XPath. Details:\n" + msg + ".")
}

// xpathResultType names a non-node-set XPath result for the error message.
func xpathResultType(v any) string {
	switch v.(type) {
	case float64:
		return "number"
	case bool:
		return "boolean"
	default: // string
		return "string"
	}
}
