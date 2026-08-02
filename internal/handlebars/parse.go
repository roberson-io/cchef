package handlebars

import (
	"fmt"
	"strings"
)

// The kinds of piece a template is made of.
type hbTokenKind int

const (
	hbTokText       hbTokenKind = iota
	hbTokVariable               // {{ path }} or {{{ path }}}
	hbTokComment                // {{! … }}
	hbTokBlockOpen              // {{#helper arg}}
	hbTokBlockClose             // {{/helper}}
	hbTokElse                   // {{else}}
	hbTokInlineOpen             // {{#*inline "name"}}
	hbTokPartial                // {{> name}}
)

// hbToken is one piece of a template, with enough of its surroundings kept to
// decide later whether the line it sits on should survive.
type hbToken struct {
	kind   hbTokenKind
	text   string // the tag's contents, or the literal text
	helper string
	arg    string
	escape bool
	// standalone records that the tag was alone on its line, in which case the
	// line's whitespace and newline are dropped.
	standalone bool
	indent     string
}

// hbLex breaks a template into text and tags.
func hbLex(source string) ([]*hbToken, error) {
	var tokens []*hbToken
	for {
		open := strings.Index(source, "{{")
		if open < 0 {
			break
		}
		if open > 0 {
			tokens = append(tokens, &hbToken{kind: hbTokText, text: source[:open]})
		}
		source = source[open:]

		// A triple-braced tag writes its value without escaping.
		width, closing := 2, "}}"
		if strings.HasPrefix(source, "{{{") {
			width, closing = 3, "}}}"
		}
		end := strings.Index(source[width:], closing)
		if end < 0 {
			//nolint:staticcheck,revive // Handlebars' verbatim error text
			return nil, fmt.Errorf("Parse error: unclosed tag %s", truncateForError(source))
		}
		inner := source[width : width+end]
		source = source[width+end+len(closing):]

		tok, err := hbClassify(inner, width == 3)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, tok)
	}
	if source != "" {
		tokens = append(tokens, &hbToken{kind: hbTokText, text: source})
	}
	return tokens, nil
}

// truncateForError shortens a piece of template for an error message.
func truncateForError(s string) string {
	const most = 20
	if len(s) > most {
		return s[:most] + "…"
	}
	return s
}

// hbClassify works out what kind of tag the contents describe.
func hbClassify(inner string, raw bool) (*hbToken, error) {
	trimmed := strings.TrimSpace(inner)
	switch {
	case trimmed == "":
		//nolint:staticcheck,revive // Handlebars' verbatim error text
		return nil, fmt.Errorf("Parse error: empty tag")

	case strings.HasPrefix(trimmed, "!"):
		return &hbToken{kind: hbTokComment, text: trimmed}, nil

	case strings.HasPrefix(trimmed, ">"):
		return &hbToken{
			kind: hbTokPartial,
			text: trimmed,
			arg:  strings.TrimSpace(strings.TrimPrefix(trimmed, ">")),
		}, nil

	case strings.HasPrefix(trimmed, "#*inline"):
		name := strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "#*inline")), `"'`)
		if name == "" {
			//nolint:staticcheck,revive // Handlebars' verbatim error text
			return nil, fmt.Errorf("Parse error: inline partial with no name")
		}
		return &hbToken{kind: hbTokInlineOpen, text: trimmed, arg: name}, nil

	case strings.HasPrefix(trimmed, "#"):
		helper, arg := hbSplitTag(strings.TrimPrefix(trimmed, "#"))
		return &hbToken{kind: hbTokBlockOpen, text: trimmed, helper: helper, arg: arg}, nil

	case strings.HasPrefix(trimmed, "/"):
		return &hbToken{
			kind:   hbTokBlockClose,
			text:   trimmed,
			helper: strings.TrimSpace(strings.TrimPrefix(trimmed, "/")),
		}, nil

	case trimmed == "else":
		return &hbToken{kind: hbTokElse, text: trimmed}, nil
	}

	return &hbToken{kind: hbTokVariable, text: trimmed, escape: !raw}, nil
}

// hbSplitTag separates a helper's name from its one argument.
func hbSplitTag(s string) (helper, arg string) {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, " \t"); i >= 0 {
		return s[:i], strings.TrimSpace(s[i+1:])
	}
	return s, ""
}

// hbStandaloneKinds are the tags that take their whole line with them when they
// are the only thing on it. A variable never does.
var hbStandaloneKinds = map[hbTokenKind]bool{
	hbTokComment:    true,
	hbTokBlockOpen:  true,
	hbTokBlockClose: true,
	hbTokElse:       true,
	hbTokInlineOpen: true,
	hbTokPartial:    true,
}

// hbStripStandalone applies Handlebars' whitespace rule: a block tag, comment or
// partial that is alone on its line takes the line's indentation and its newline
// with it, so writing a template over several lines does not fill the output
// with blank ones. A partial keeps the indentation, which it puts in front of
// every line it renders.
//
// The work is done a line at a time rather than by looking at each tag's
// neighbours, because several standalone tags can sit on consecutive lines and
// deciding about one must not depend on what removing an earlier one left
// behind.
func hbStripStandalone(tokens []*hbToken) []*hbToken {
	tokens = hbSplitTextAtNewlines(tokens)

	for _, line := range hbLines(tokens) {
		tag, ok := hbLoneTagOn(tokens, line)
		if !ok {
			continue
		}
		tokens[tag].standalone = true
		if tokens[tag].kind == hbTokPartial {
			tokens[tag].indent = hbIndentBefore(tokens, line, tag)
		}
		// Everything on the line but the tag is whitespace, so it all goes.
		for i := line.from; i <= line.to; i++ {
			if i != tag {
				tokens[i].text = ""
			}
		}
	}
	return tokens
}

// hbSplitTextAtNewlines breaks each run of literal text so that no piece holds a
// newline anywhere but at its end. Every line then ends at a token boundary.
func hbSplitTextAtNewlines(tokens []*hbToken) []*hbToken {
	out := make([]*hbToken, 0, len(tokens))
	for _, tok := range tokens {
		if tok.kind != hbTokText {
			out = append(out, tok)
			continue
		}
		rest := tok.text
		for {
			nl := strings.IndexByte(rest, '\n')
			if nl < 0 {
				break
			}
			out = append(out, &hbToken{kind: hbTokText, text: rest[:nl+1]})
			rest = rest[nl+1:]
		}
		if rest != "" {
			out = append(out, &hbToken{kind: hbTokText, text: rest})
		}
	}
	return out
}

// hbLine is the range of tokens making up one line of the template.
type hbLine struct{ from, to int }

// hbLines groups the tokens into lines, each ending at the piece of text that
// carries the newline.
func hbLines(tokens []*hbToken) []hbLine {
	var lines []hbLine
	from := 0
	for i, tok := range tokens {
		if tok.kind == hbTokText && strings.HasSuffix(tok.text, "\n") {
			lines = append(lines, hbLine{from: from, to: i})
			from = i + 1
		}
	}
	if from < len(tokens) {
		lines = append(lines, hbLine{from: from, to: len(tokens) - 1})
	}
	return lines
}

// hbLoneTagOn returns the one tag on the line, when the line holds exactly one
// tag of a kind that takes its line with it and nothing else but whitespace.
func hbLoneTagOn(tokens []*hbToken, line hbLine) (int, bool) {
	tag := -1
	for i := line.from; i <= line.to; i++ {
		tok := tokens[i]
		if tok.kind == hbTokText {
			if strings.Trim(tok.text, " \t\r\n") != "" {
				return 0, false
			}
			continue
		}
		if !hbStandaloneKinds[tok.kind] || tag >= 0 {
			return 0, false
		}
		tag = i
	}
	return tag, tag >= 0
}

// hbIndentBefore returns the whitespace between the start of the line and the
// tag on it.
func hbIndentBefore(tokens []*hbToken, line hbLine, tag int) string {
	var indent strings.Builder
	for i := line.from; i < tag; i++ {
		indent.WriteString(tokens[i].text)
	}
	return indent.String()
}

// hbParser turns the tokens into a tree.
type hbParser struct {
	tokens []*hbToken
	pos    int
}

// parseUntil reads nodes up to the close of the named block, or to the end of
// the template when the name is empty.
func (p *hbParser) parseUntil(closing string) ([]hbNode, error) {
	var nodes []hbNode
	for p.pos < len(p.tokens) {
		tok := p.tokens[p.pos]

		switch tok.kind {
		case hbTokBlockClose, hbTokElse:
			if closing == "" && tok.kind == hbTokBlockClose {
				//nolint:staticcheck,revive // Handlebars' verbatim error text
				return nil, fmt.Errorf("Parse error: {{/%s}} closes nothing", tok.helper)
			}
			return nodes, nil

		case hbTokText:
			p.pos++
			if tok.text != "" {
				nodes = append(nodes, hbText(tok.text))
			}

		case hbTokComment:
			p.pos++

		case hbTokVariable:
			p.pos++
			nodes = append(nodes, hbVariable{path: tok.text, escape: tok.escape})

		case hbTokPartial:
			p.pos++
			nodes = append(nodes, hbPartialUse{name: tok.arg, indent: tok.indent})

		case hbTokInlineOpen:
			node, err := p.parseInline(tok)
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, node)

		case hbTokBlockOpen:
			node, err := p.parseBlock(tok)
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, node)
		}
	}

	if closing != "" {
		//nolint:staticcheck,revive // Handlebars' verbatim error text
		return nil, fmt.Errorf("Parse error: {{#%s}} is never closed", closing)
	}
	return nodes, nil
}

// parseInline reads an inline partial definition.
func (p *hbParser) parseInline(open *hbToken) (hbNode, error) {
	p.pos++
	body, err := p.parseUntil("inline")
	if err != nil {
		return nil, err
	}
	if err := p.expectClose("inline"); err != nil {
		return nil, err
	}
	return hbPartialDef{name: open.arg, body: body}, nil
}

// parseBlock reads a block helper, and the alternative after an {{else}}.
func (p *hbParser) parseBlock(open *hbToken) (hbNode, error) {
	p.pos++
	body, err := p.parseUntil(open.helper)
	if err != nil {
		return nil, err
	}

	block := &hbBlock{helper: open.helper, arg: open.arg, body: body}

	if p.pos < len(p.tokens) && p.tokens[p.pos].kind == hbTokElse {
		p.pos++
		inverse, err := p.parseUntil(open.helper)
		if err != nil {
			return nil, err
		}
		block.inverse = inverse
	}

	if err := p.expectClose(open.helper); err != nil {
		return nil, err
	}
	return block, nil
}

// expectClose consumes the tag closing the named block.
func (p *hbParser) expectClose(helper string) error {
	if p.pos >= len(p.tokens) || p.tokens[p.pos].kind != hbTokBlockClose {
		//nolint:staticcheck,revive // Handlebars' verbatim error text
		return fmt.Errorf("Parse error: {{#%s}} is never closed", helper)
	}
	if got := p.tokens[p.pos].helper; got != helper {
		//nolint:staticcheck,revive // Handlebars' verbatim error text
		return fmt.Errorf("Parse error: {{#%s}} is closed by {{/%s}}", helper, got)
	}
	p.pos++
	return nil
}
