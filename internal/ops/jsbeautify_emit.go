package ops

// Comment emission for JavaScript Beautify — a transliteration of escodegen's
// generateComment / addComments (the non-preserve-blank-lines path). When
// comment mode is on, generateStatement/generateExpression call addComments on
// each node, prepending its leading comments and appending its trailing ones.

import (
	"strings"
	"unicode/utf16"
)

func (g *jsGen) generateComment(c *jsComment) string {
	if c.typ == "Line" {
		// A Line comment's value never ends with a line terminator: the scanner
		// slices the body as [start, index-1), stopping before the terminator
		// (jsparser_scanner.go skipSingleLineComment). escodegen guards this case
		// defensively, but it is unreachable here, so the guard is omitted.
		return "//" + c.value + "\n"
	}
	return "/*" + c.value + "*/"
}

// calculateSpaces returns the number of code units after the last line
// terminator in s (escodegen's calculateSpaces).
func calculateSpaces(s string) int {
	units := utf16.Encode([]rune(s))
	i := len(units) - 1
	for ; i >= 0; i-- {
		if jsIsLineTerm(int(units[i])) {
			break
		}
	}
	return (len(units) - 1) - i
}

// jsNodeLeading / jsNodeTrailing return the comments attached to a node, or nil.
func (g *jsGen) jsNodeLeading(node any) []*jsComment {
	if g.comments == nil {
		return nil
	}
	return g.comments.leading[jsNodeKey(node)]
}

func (g *jsGen) jsNodeTrailing(node any) []*jsComment {
	if g.comments == nil {
		return nil
	}
	return g.comments.trailing[jsNodeKey(node)]
}

func (g *jsGen) addComments(node any, result string) string {
	if g.comments == nil {
		return result
	}
	if leading := g.jsNodeLeading(node); len(leading) > 0 {
		result = g.addLeadingComments(leading, result)
	}
	if trailing := g.jsNodeTrailing(node); len(trailing) > 0 {
		result = g.addTrailingComments(trailing, result)
	}
	return result
}

func (g *jsGen) addLeadingComments(leading []*jsComment, save string) string {
	var b strings.Builder
	b.WriteString(g.generateComment(leading[0]))
	if !endsWithLineTerminator(b.String()) {
		b.WriteString(g.newline)
	}
	for i := 1; i < len(leading); i++ {
		frag := g.generateComment(leading[i])
		if !endsWithLineTerminator(frag) {
			frag += g.newline
		}
		b.WriteString(g.addIndent(frag))
	}
	b.WriteString(g.addIndent(save))
	return b.String()
}

func (g *jsGen) addTrailingComments(trailing []*jsComment, result string) string {
	tailingToStatement := !endsWithLineTerminator(result)
	specialBase := strings.Repeat(" ", calculateSpaces(g.base+result+g.indent))
	for i, c := range trailing {
		if tailingToStatement {
			if i == 0 {
				result += g.indent
			} else {
				result += specialBase
			}
			result += g.generateComment(c)
		} else {
			result += g.addIndent(g.generateComment(c))
		}
		if i != len(trailing)-1 && !endsWithLineTerminator(result) {
			result += g.newline
		}
	}
	return result
}
