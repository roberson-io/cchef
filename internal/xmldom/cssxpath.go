package xmldom

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/internal/opsutil"
)

// CSSToXPath translates a CSS selector (the nwmatcher subset CyberChef exercises)
// into an equivalent XPath 1.0 expression evaluable by antchfx/xpath over an
// xmlNode tree. Element type and attribute names are matched case-insensitively
// (the navigator lowercases LocalName, and names are lowered here to match);
// class, id and attribute values are case-sensitive except for the HTML
// enumerated attributes in ciAttrValues, mirroring nwmatcher's HTML_TABLE.
func CSSToXPath(selector string) (string, error) {
	groups := opsutil.SplitTopLevel(selector, ',')
	parts := make([]string, 0, len(groups))
	for _, g := range groups {
		g = strings.TrimSpace(g)
		if g == "" {
			return "", fmt.Errorf("empty selector in group")
		}
		xp, err := complexToXPath(g)
		if err != nil {
			return "", err
		}
		parts = append(parts, xp)
	}
	return strings.Join(parts, " | "), nil
}

// cssStep is one compound selector plus the combinator that precedes it.
type cssStep struct {
	comb byte // 0 first, ' ' descendant, '>' child, '+' adjacent, '~' general
	text string
}

func complexToXPath(sel string) (string, error) {
	steps, err := splitCombinators(sel)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, st := range steps {
		c, err := parseCompound(st.text)
		if err != nil {
			return "", err
		}
		b.WriteString(stepAxis(st.comb))
		b.WriteString(c.name)
		for _, p := range c.preds {
			b.WriteString(p)
		}
	}
	return b.String(), nil
}

func stepAxis(comb byte) string {
	switch comb {
	case '>':
		return "/"
	case '~':
		return "/following-sibling::"
	case '+':
		return "/following-sibling::*[1]/self::"
	}
	// First step (comb == 0) and descendant combinator (' ').
	return "//"
}

// splitCombinators tokenizes a complex selector into compound steps.
func splitCombinators(sel string) ([]cssStep, error) {
	var steps []cssStep
	i, n := 0, len(sel)
	comb := byte(0)
	for i < n {
		for i < n && sel[i] == ' ' {
			i++
		}
		if i >= n {
			break
		}
		if c := sel[i]; c == '>' || c == '+' || c == '~' {
			comb = c
			i++
			continue
		}
		// If we already have a step and no explicit combinator was seen, the
		// gap was descendant whitespace.
		if len(steps) > 0 && comb == 0 {
			comb = ' '
		}
		var text string
		text, i = readCompoundToken(sel, i)
		steps = append(steps, cssStep{comb: comb, text: text})
		comb = 0
	}
	if len(steps) == 0 {
		return nil, fmt.Errorf("empty selector")
	}
	return steps, nil
}

// readCompoundToken reads a single compound selector starting at sel[i], stopping
// at a top-level combinator or whitespace (brackets and parentheses are honored
// so combinator characters inside them are not treated as separators). It returns
// the token text and the index of the following character.
func readCompoundToken(sel string, i int) (string, int) {
	start, depth := i, 0
	for i < len(sel) {
		switch c := sel[i]; {
		case c == '[' || c == '(':
			depth++
		case c == ']' || c == ')':
			depth--
		case depth == 0 && (c == ' ' || c == '>' || c == '+' || c == '~'):
			return sel[start:i], i
		}
		i++
	}
	return sel[start:i], i
}

type compound struct {
	name  string // element name test (lowercased) or "*"
	preds []string
}

func parseCompound(s string) (compound, error) {
	c := compound{name: "*"}
	i := 0
	// Leading type / universal.
	if i < len(s) && (isCSSNameChar(s[i]) || s[i] == '*') {
		if s[i] == '*' {
			i++
		} else {
			start := i
			for i < len(s) && isCSSNameChar(s[i]) {
				i++
			}
			c.name = strings.ToLower(s[start:i])
		}
	}
	for i < len(s) {
		switch s[i] {
		case '.':
			i++
			val := readIdent(s, &i)
			c.preds = append(c.preds, "[contains(concat(' ',normalize-space(@class),' '),"+xpathLiteral(" "+val+" ")+")]")
		case '#':
			i++
			val := readIdent(s, &i)
			c.preds = append(c.preds, "[@id="+xpathLiteral(val)+"]")
		case '[':
			pred, ni, err := parseAttr(s, i)
			if err != nil {
				return c, err
			}
			c.preds = append(c.preds, pred)
			i = ni
		case ':':
			pred, ni, err := parsePseudo(s, i, c.name)
			if err != nil {
				return c, err
			}
			c.preds = append(c.preds, pred)
			i = ni
		default:
			return c, fmt.Errorf("unexpected %q in selector", s[i])
		}
	}
	return c, nil
}

func readIdent(s string, i *int) string {
	start := *i
	for *i < len(s) && (isCSSNameChar(s[*i]) || s[*i] == '\\') {
		*i++
	}
	return s[start:*i]
}

// isCSSNameChar reports whether c may appear in a CSS identifier (type, class,
// id, pseudo name). Unlike XML names, CSS identifiers exclude '.', ':' and '#',
// which act as selector delimiters.
func isCSSNameChar(c byte) bool {
	return c == '-' || c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') || c >= 0x80
}

// ciAttrValues is nwmatcher's HTML_TABLE: attributes whose values match
// case-insensitively in HTML documents.
var ciAttrValues = map[string]bool{
	"accept": true, "accept-charset": true, "align": true, "alink": true, "axis": true,
	"bgcolor": true, "charset": true, "checked": true, "clear": true, "codetype": true, "color": true,
	"compact": true, "declare": true, "defer": true, "dir": true, "direction": true, "disabled": true,
	"enctype": true, "face": true, "frame": true, "hreflang": true, "http-equiv": true, "lang": true,
	"language": true, "link": true, "media": true, "method": true, "multiple": true, "nohref": true,
	"noresize": true, "noshade": true, "nowrap": true, "readonly": true, "rel": true, "rev": true,
	"rules": true, "scope": true, "scrolling": true, "selected": true, "shape": true, "target": true,
	"text": true, "type": true, "valign": true, "valuetype": true, "vlink": true,
}

func parseAttr(s string, i int) (string, int, error) {
	end := strings.IndexByte(s[i:], ']')
	if end < 0 {
		return "", i, fmt.Errorf("unterminated attribute selector")
	}
	body := strings.TrimSpace(s[i+1 : i+end])
	next := i + end + 1
	name, op, val := splitAttr(body)
	name = strings.ToLower(name)
	ci := ciAttrValues[name]
	return "[" + attrTest(name, op, val, ci) + "]", next, nil
}

// splitAttr parses "name", "name=val", "name^=val" etc.
func splitAttr(body string) (name, op, val string) {
	for _, o := range []string{"~=", "|=", "^=", "$=", "*=", "="} {
		if before, after, ok := strings.Cut(body, o); ok {
			name = strings.TrimSpace(before)
			op = o
			val = strings.TrimSpace(after)
			val = unquoteCSS(val)
			return
		}
	}
	return strings.TrimSpace(body), "", ""
}

func attrTest(name, op, val string, ci bool) string {
	// nwmatcher resolves the "checked" and "selected" attributes via the
	// defaultChecked/defaultSelected DOM properties (ATTR_DEFAULT), which are
	// undefined on xmldom nodes, so any selector on them matches nothing.
	if name == "checked" || name == "selected" {
		return "1=0"
	}
	attr := "@" + name
	if op == "" {
		return attr
	}
	lhs := attr
	if ci {
		lhs = lcExpr(attr)
		val = strings.ToLower(val)
	}
	switch op {
	case "~=":
		return "contains(concat(' ',normalize-space(" + lhs + "),' ')," + xpathLiteral(" "+val+" ") + ")"
	case "|=":
		return "(" + lhs + "=" + xpathLiteral(val) + " or starts-with(" + lhs + "," + xpathLiteral(val+"-") + "))"
	case "^=":
		return "starts-with(" + lhs + "," + xpathLiteral(val) + ")"
	case "$=":
		return "substring(" + lhs + ",string-length(" + lhs + ")-" + strconv.Itoa(len(val)-1) + ")=" + xpathLiteral(val)
	case "*=":
		return "contains(" + lhs + "," + xpathLiteral(val) + ")"
	default: // "=" (the only remaining operator splitAttr produces)
		return lhs + "=" + xpathLiteral(val)
	}
}

func lcExpr(e string) string {
	return "translate(" + e + ",'ABCDEFGHIJKLMNOPQRSTUVWXYZ','abcdefghijklmnopqrstuvwxyz')"
}

func unquoteCSS(v string) string {
	if len(v) >= 2 && (v[0] == '"' || v[0] == '\'') && v[len(v)-1] == v[0] {
		return v[1 : len(v)-1]
	}
	return v
}

// xpathLiteral produces an XPath string literal for s, using concat() when s
// contains both quote characters.
func xpathLiteral(s string) string {
	if !strings.Contains(s, "'") {
		return "'" + s + "'"
	}
	if !strings.Contains(s, `"`) {
		return `"` + s + `"`
	}
	var parts []string
	for seg := range strings.SplitSeq(s, "'") {
		if seg != "" {
			parts = append(parts, "'"+seg+"'")
		}
		parts = append(parts, `"'"`)
	}
	parts = parts[:len(parts)-1] // drop trailing separator
	return "concat(" + strings.Join(parts, ",") + ")"
}
