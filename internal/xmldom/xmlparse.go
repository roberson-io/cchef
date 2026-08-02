package xmldom

import (
	"strconv"
	"strings"
)

// Parse reproduces @xmldom/xmldom's DOMParser.parseFromString(source) called
// with no mimeType: XML mode (isHTML=false), decoding only the five XML named
// entities plus numeric character references. It is deliberately lenient about
// unclosed tags and stray markup, mirroring how xmldom recovers, and enforces a
// single documentElement (content after the root element closes is discarded).
func Parse(src string) *Node {
	p := &xmlParser{
		src:      normalizeLineEndings(src),
		doc:      &Node{typ: xmlDocument},
		closeMap: map[string]int{},
	}
	p.parse()
	return p.doc
}

type xmlParser struct {
	src      string
	pos      int
	doc      *Node
	stack    []*Node        // currently open elements, innermost last
	haveRoot bool           // the documentElement has been seen
	limbo    *Node          // sink for content after the root closes (discarded)
	closeMap map[string]int // cache of the last "</tag>" index per tag name
}

// parent returns the node new children attach to: the innermost open element,
// the document (before the root), or a discard sink (after the root closes).
func (p *xmlParser) parent() *Node {
	if len(p.stack) > 0 {
		return p.stack[len(p.stack)-1]
	}
	if p.haveRoot {
		if p.limbo == nil {
			p.limbo = &Node{typ: xmlDocument}
		}
		return p.limbo
	}
	return p.doc
}

func (p *xmlParser) append(n *Node) {
	par := p.parent()
	n.parent = par
	par.children = append(par.children, n)
}

func (p *xmlParser) parse() {
	for p.pos < len(p.src) {
		if p.src[p.pos] == '<' {
			p.parseMarkup()
		} else {
			p.parseText()
		}
	}
}

func (p *xmlParser) parseText() {
	start := p.pos
	for p.pos < len(p.src) && p.src[p.pos] != '<' {
		p.pos++
	}
	// Text outside the root element (document level) is discarded.
	if len(p.stack) == 0 {
		return
	}
	p.append(&Node{typ: xmlText, data: decodeEntities(p.src[start:p.pos])})
}

func (p *xmlParser) parseMarkup() {
	rest := p.src[p.pos:]
	switch {
	case strings.HasPrefix(rest, "<!--"):
		p.parseComment()
	case strings.HasPrefix(rest, "<![CDATA["):
		p.parseCData()
	case strings.HasPrefix(rest, "<!"):
		p.skipTo(">") // DOCTYPE / declarations: not represented in the DOM
	case strings.HasPrefix(rest, "<?"):
		p.parsePI()
	case strings.HasPrefix(rest, "</"):
		p.parseCloseTag()
	case len(rest) > 1 && isNameStart(rest[1]):
		p.parseStartTag()
	default:
		// A '<' not introducing markup is treated as literal text.
		p.pos++
		if len(p.stack) > 0 {
			p.append(&Node{typ: xmlText, data: "<"})
		}
	}
}

func (p *xmlParser) parseComment() {
	p.pos += len("<!--")
	end := strings.Index(p.src[p.pos:], "-->")
	if end < 0 {
		p.append(&Node{typ: xmlComment, data: p.src[p.pos:]})
		p.pos = len(p.src)
		return
	}
	p.append(&Node{typ: xmlComment, data: p.src[p.pos : p.pos+end]})
	p.pos += end + len("-->")
}

func (p *xmlParser) parseCData() {
	p.pos += len("<![CDATA[")
	end := strings.Index(p.src[p.pos:], "]]>")
	if end < 0 {
		p.append(&Node{typ: xmlCData, data: p.src[p.pos:]})
		p.pos = len(p.src)
		return
	}
	p.append(&Node{typ: xmlCData, data: p.src[p.pos : p.pos+end]})
	p.pos += end + len("]]>")
}

func (p *xmlParser) parsePI() {
	p.pos += len("<?")
	end := strings.Index(p.src[p.pos:], "?>")
	body := p.src[p.pos:]
	if end >= 0 {
		body = p.src[p.pos : p.pos+end]
		p.pos += end + len("?>")
	} else {
		p.pos = len(p.src)
	}
	target, data, _ := strings.Cut(strings.TrimLeft(body, " \t\n"), " ")
	p.append(&Node{typ: xmlPI, name: target, data: strings.TrimLeft(data, " \t\n")})
}

// skipTo advances the position just past the next occurrence of term.
func (p *xmlParser) skipTo(term string) {
	i := strings.Index(p.src[p.pos:], term)
	if i < 0 {
		p.pos = len(p.src)
		return
	}
	p.pos += i + len(term)
}

func (p *xmlParser) parseCloseTag() {
	p.pos += len("</")
	name := p.readName()
	p.skipTo(">")
	// xmldom only closes the current (innermost) open element, matching its name
	// case-insensitively; a close tag for anything else is ignored.
	if n := len(p.stack); n > 0 && strings.EqualFold(p.stack[n-1].name, name) {
		p.stack = p.stack[:n-1]
	}
}

func (p *xmlParser) parseStartTag() {
	p.pos++ // consume '<'
	name := p.readName()
	el := &Node{typ: xmlElement, name: name}
	p.parseAttributes(el)
	selfClose := false
	if p.pos < len(p.src) && p.src[p.pos] == '/' {
		selfClose = true
		p.pos++
	}
	// xmldom's fixSelfClosed heuristic: an element with no matching "</name>"
	// later in the source is treated as empty/self-closing, so its would-be
	// children become siblings at the parent level.
	if !selfClose && p.noCloseAhead(name, p.pos) {
		selfClose = true
	}
	if p.pos < len(p.src) && p.src[p.pos] == '>' {
		p.pos++
	}
	p.append(el)
	if len(p.stack) == 0 {
		p.haveRoot = true
	}
	if !selfClose {
		p.stack = append(p.stack, el)
	}
}

// noCloseAhead reports whether the source has no "</name>" close tag at or after
// gtPos, mirroring xmldom's fixSelfClosed (with its cache and its fallback to a
// bare "</name" when no full close tag exists).
func (p *xmlParser) noCloseAhead(name string, gtPos int) bool {
	pos, ok := p.closeMap[name]
	if !ok {
		pos = strings.LastIndex(p.src, "</"+name+">")
		if pos < gtPos {
			pos = strings.LastIndex(p.src, "</"+name)
		}
		p.closeMap[name] = pos
	}
	return pos < gtPos
}

func (p *xmlParser) parseAttributes(el *Node) {
	for p.pos < len(p.src) {
		p.skipSpace()
		if p.pos >= len(p.src) {
			return
		}
		c := p.src[p.pos]
		if c == '>' || c == '/' {
			return
		}
		name := p.readAttrName()
		if name == "" {
			p.pos++ // avoid stalling on an unexpected character
			continue
		}
		p.skipSpace()
		value := name // value-less attributes take their name as value
		if p.pos < len(p.src) && p.src[p.pos] == '=' {
			p.pos++
			p.skipSpace()
			value = decodeEntities(p.readAttrValue())
		}
		el.Attrs = append(el.Attrs, Attr{Name: name, Value: value})
	}
}

func (p *xmlParser) readAttrValue() string {
	if p.pos >= len(p.src) {
		return ""
	}
	q := p.src[p.pos]
	if q == '"' || q == '\'' {
		p.pos++
		start := p.pos
		for p.pos < len(p.src) && p.src[p.pos] != q {
			p.pos++
		}
		v := p.src[start:p.pos]
		if p.pos < len(p.src) {
			p.pos++ // closing quote
		}
		return v
	}
	// Unquoted value: read until whitespace or tag end.
	start := p.pos
	for p.pos < len(p.src) && !isSpace(p.src[p.pos]) && p.src[p.pos] != '>' && p.src[p.pos] != '/' {
		p.pos++
	}
	return p.src[start:p.pos]
}

func (p *xmlParser) readName() string {
	start := p.pos
	for p.pos < len(p.src) && isNameChar(p.src[p.pos]) {
		p.pos++
	}
	return p.src[start:p.pos]
}

func (p *xmlParser) readAttrName() string {
	start := p.pos
	for p.pos < len(p.src) {
		c := p.src[p.pos]
		if isSpace(c) || c == '=' || c == '>' || c == '/' {
			break
		}
		p.pos++
	}
	return p.src[start:p.pos]
}

func (p *xmlParser) skipSpace() {
	for p.pos < len(p.src) && isSpace(p.src[p.pos]) {
		p.pos++
	}
}

func isSpace(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' }

func isNameStart(c byte) bool {
	return c == '_' || c == ':' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c >= 0x80
}

func isNameChar(c byte) bool {
	return isNameStart(c) || c == '-' || c == '.' || (c >= '0' && c <= '9')
}

// decodeEntities resolves the five XML named entities and numeric character
// references. Unknown entities (e.g. &nbsp;) are left as literal text, matching
// xmldom's XML-mode recovery.
func decodeEntities(s string) string {
	if !strings.Contains(s, "&") {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] == '&' {
			if r, n, ok := matchEntity(s[i:]); ok {
				b.WriteRune(r)
				i += n
				continue
			}
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

var xmlNamedEntities = map[string]rune{"amp": '&', "lt": '<', "gt": '>', "quot": '"', "apos": '\''}

// matchEntity decodes the entity at the start of s (which begins with '&'),
// returning the rune, the number of bytes consumed, and whether it matched.
func matchEntity(s string) (rune, int, bool) {
	semi := strings.IndexByte(s, ';')
	if semi < 2 {
		return 0, 0, false
	}
	body := s[1:semi]
	if body[0] == '#' {
		var v int64
		var err error
		if len(body) > 1 && (body[1] == 'x' || body[1] == 'X') {
			v, err = strconv.ParseInt(body[2:], 16, 32)
		} else {
			v, err = strconv.ParseInt(body[1:], 10, 32)
		}
		if err != nil || v < 0 {
			return 0, 0, false
		}
		return rune(v), semi + 1, true
	}
	if r, ok := xmlNamedEntities[body]; ok {
		return r, semi + 1, true
	}
	return 0, 0, false
}

// normalizeLineEndings mirrors xmldom's XML 1.1 line-ending normalization:
// \r\n and \r\u0085 collapse to \n, then lone \r, \u0085 and \u2028 become \n.
func normalizeLineEndings(s string) string {
	if !strings.ContainsAny(s, "\r\u0085\u2028") {
		return s
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r\u0085", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = strings.ReplaceAll(s, "\u0085", "\n")
	s = strings.ReplaceAll(s, "\u2028", "\n")
	return s
}
