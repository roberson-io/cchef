package yara

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
)

// Finding strings in the data.
//
// Go's regular expressions work in characters, not bytes, so a pattern holding
// a byte above 0x7f would be read as the two bytes that spell it in UTF-8 and
// would not match. The way round it is to work in a space where every byte is
// the character of the same number: the data is mapped into that space once,
// the patterns are built in it, and a table maps positions back. Every pattern
// is anchored and tried at every position, so overlapping matches all come out,
// as they do in YARA.

// Match is one place a string was found.
type Match struct {
	Offset int
	Length int
	Data   []byte
}

// anyByte matches one byte of anything, in the mapped space.
const anyByte = `[\x{00}-\x{ff}]`

// buffer is the data being scanned, mapped into the space patterns match in.
type buffer struct {
	data   []byte
	mapped string
	// starts maps each position of mapped back to the byte it came from.
	starts []int
	// offsets maps each byte to where it begins in mapped.
	offsets []int
}

// newBuffer maps data into the space the patterns are matched in.
func newBuffer(data []byte) *buffer {
	var b strings.Builder
	b.Grow(len(data))
	starts := make([]int, 0, len(data)+1)
	offsets := make([]int, len(data)+1)
	for i, c := range data {
		offsets[i] = len(starts)
		starts = append(starts, i)
		if c >= 0x80 {
			starts = append(starts, i)
		}
		b.WriteRune(rune(c))
	}
	offsets[len(data)] = len(starts)
	starts = append(starts, len(data))
	return &buffer{data: data, mapped: b.String(), starts: starts, offsets: offsets}
}

// latin1 maps bytes into the space the patterns are matched in.
func latin1(data []byte) string {
	var b strings.Builder
	b.Grow(len(data))
	for _, c := range data {
		b.WriteRune(rune(c))
	}
	return b.String()
}

// pattern is one thing to look for, compiled and ready.
type pattern struct {
	re *regexp.Regexp
	// reElsewhere is the same pattern with every mark tying it to the start of
	// the data made impossible, for trying it anywhere but the beginning. It is
	// left unset when the pattern carries no such mark.
	reElsewhere *regexp.Regexp
	// fullword says a match has to stand alone rather than sit inside a longer
	// word.
	fullword bool
	// wide says the match carries a null after every byte, so the characters
	// either side of it are two bytes wide as well.
	wide bool
	// startEdge and endEdge are what the pattern asks about the edge of a word
	// at its ends, which is checked once a match is found because the edge falls
	// between whole wide characters.
	startEdge, endEdge wordEdge
}

// findAll returns every place a pattern matches, including matches that overlap
// one another: YARA reports one for each position a match can begin at, each
// reaching as far as it can.
func (p *pattern) findAll(buf *buffer) []Match {
	var out []Match
	for at := range buf.data {
		re := p.re
		if at > 0 && p.reElsewhere != nil {
			re = p.reElsewhere
		}
		loc := re.FindStringIndex(buf.mapped[buf.offsets[at]:])
		if loc == nil {
			continue
		}
		end := buf.starts[buf.offsets[at]+loc[1]]
		if p.fullword && !standsAlone(buf.data, at, end, p.wide) {
			continue
		}
		if !p.wordEdgesHold(buf.data, at, end) {
			continue
		}
		out = append(out, Match{Offset: at, Length: end - at, Data: buf.data[at:end]})
	}
	return out
}

// wordEdgesHold checks what a pattern asks about the edge of a word at its ends.
// Only a pattern looked for wide asks, since for single bytes the expression
// itself already has it right.
func (p *pattern) wordEdgesHold(data []byte, start, end int) bool {
	return edgeHolds(p.startEdge, wideWordAt(data, start-2), wideWordAt(data, start)) &&
		edgeHolds(p.endEdge, wideWordAt(data, end-2), wideWordAt(data, end))
}

// edgeHolds says whether what a pattern asked for is met by what lies either
// side of one of its ends.
func edgeHolds(edge wordEdge, before, after bool) bool {
	switch edge {
	case edgeBoundary:
		return before != after
	case edgeNoBoundary:
		return before == after
	}
	return true
}

// wideWordAt says whether the wide character beginning somewhere is one a word is
// made of. Nothing at all — before the data, or past its end — is not.
func wideWordAt(data []byte, at int) bool {
	return at >= 0 && at+1 < len(data) && data[at+1] == 0 && isWordByte(data[at])
}

// standsAlone reports whether a match has something other than a letter or a
// digit on either side of it, which is what "fullword" asks for.
//
// Looked for wide, what sits either side is a pair of bytes rather than one, and
// a pair only makes a character of a word when its second byte is a null and its
// first is a letter or a digit. So plain narrow text after a wide match does not
// run into it, since the pair there is not a wide character at all.
func standsAlone(data []byte, start, end int, wide bool) bool {
	if wide {
		if start >= 2 && data[start-1] == 0 && isWordByte(data[start-2]) {
			return false
		}
		return end+1 >= len(data) || data[end+1] != 0 || !isWordByte(data[end])
	}
	if start >= 1 && isWordByte(data[start-1]) {
		return false
	}
	return end >= len(data) || !isWordByte(data[end])
}

// isWordByte reports whether a byte is one a word is made of, which is a letter
// or a digit and nothing else. An underscore is not one, as it is not in YARA.
func isWordByte(c byte) bool {
	return isLetter(c) || (c >= '0' && c <= '9')
}

// compileStrings turns one of a rule's strings into the patterns that find it.
// A string may come to several: asked for both widths, or for a range of xor
// keys, it is looked for in each form.
func compileStrings(str *String) ([]*pattern, error) {
	bodies, err := patternBodies(str)
	if err != nil {
		return nil, err
	}

	out := make([]*pattern, 0, len(bodies))
	for _, body := range bodies {
		// Anchored, so that trying it at each position gives the match starting
		// there. Whether a dot reaches across a newline is left to the string's
		// own flags, as it is in YARA.
		// A mark tying the pattern to the start means the start of the data, so
		// away from the beginning a second form is needed in which that mark
		// can never be met. Without such a mark the one form does for anywhere.
		forms := []string{body.text}
		if rest, tied := withoutStartMark(body.text); tied {
			forms = append(forms, rest)
		}
		built := make([]*regexp.Regexp, 0, len(forms))
		for _, text := range forms {
			re, err := regexp.Compile(`\A(?:` + text + `)`)
			if err != nil {
				return nil, badRegex(str, "")
			}
			built = append(built, re)
		}
		p := &pattern{
			re: built[0], fullword: str.Mods.Fullword, wide: body.wide,
			startEdge: body.startEdge, endEdge: body.endEdge,
		}
		if len(built) > 1 {
			p.reElsewhere = built[1]
		}
		out = append(out, p)
	}
	return out, nil
}

// neverMet is a piece of pattern that cannot be matched: the end of the data
// with something after it. It stands in for a mark tying a pattern to the start
// of the data when the pattern is being tried anywhere else.
const neverMet = `(?:\z.)`

// withoutStartMark rewrites every mark tying a pattern to the start of the data
// into one that can never be met, and says whether there were any. A mark
// within a class of characters, or one written out as itself, is left alone.
func withoutStartMark(text string) (string, bool) {
	var out strings.Builder
	tied := false
	for i := 0; i < len(text); {
		switch text[i] {
		case '\\':
			out.WriteString(text[i:min(i+2, len(text))])
			i += 2
		case '[':
			end := min(skipClass(text, i), len(text))
			out.WriteString(text[i:end])
			i = end
		case '^':
			out.WriteString(neverMet)
			tied = true
			i++
		default:
			out.WriteByte(text[i])
			i++
		}
	}
	return out.String(), tied
}

// body is one shape a string is looked for in.
type body struct {
	text string
	wide bool
	// startEdge and endEdge are what the shape asks for at its ends about the
	// edge of a word, which only a wide shape carries.
	startEdge, endEdge wordEdge
}

// patternBodies builds every shape of a string, in the mapped space.
func patternBodies(str *String) ([]body, error) {
	if str.Kind == stringText {
		if str.Mods.Base64 || str.Mods.Base64Wide {
			return base64Bodies(str)
		}
		return textBodies(str), nil
	}
	if str.Kind == stringHex {
		return []body{{text: str.pattern}}, nil
	}
	return regexBodies(str)
}

// plainClassBrackets writes out an opening bracket inside a class of characters
// as itself. YARA has no named classes, so `[[:alpha:]]` is a class holding
// those very characters followed by one or more closing brackets, where Go's own
// expressions would read `[:alpha:]` as a name.
func plainClassBrackets(text string) string {
	var out strings.Builder
	for i := 0; i < len(text); {
		if text[i] == '\\' {
			out.WriteString(text[i:min(i+2, len(text))])
			i += 2
			continue
		}
		if text[i] != '[' {
			out.WriteByte(text[i])
			i++
			continue
		}
		end := min(skipClass(text, i), len(text))
		// The class runs from the opening bracket to the closing one. Everything
		// between them is copied over, with a further opening bracket written
		// out as itself and anything already written that way left alone.
		out.WriteByte('[')
		for at := i + 1; at < end; {
			switch text[at] {
			case '\\':
				out.WriteString(text[at:min(at+2, end)])
				at += 2
			case '[':
				out.WriteString(`\[`)
				at++
			default:
				out.WriteByte(text[at])
				at++
			}
		}
		i = end
	}
	return out.String()
}

// badRegex reports a pattern that cannot be read, by the name the rule gave it
// rather than by the pattern itself, which is how YARA reports one. A reason is
// added when there is one to give.
func badRegex(str *String, why string) error {
	msg := fmt.Sprintf("invalid %s %q", str.Kind, str.ID)
	if why != "" {
		msg += ": " + why
	}
	return &compileError{line: str.Line, msg: msg}
}

// regexBodies builds every shape of a regular expression: as written, and with
// a null after every character it matches, in whichever widths were asked for.
func regexBodies(str *String) ([]body, error) {
	written := latin1([]byte(str.Text))
	// YARA's regular expressions have brackets only for grouping, so brackets
	// written to do anything else are refused rather than read as something the
	// rule did not ask for.
	if strings.Contains(written, "(?") {
		return nil, badRegex(str, "syntax error, unexpected '?'")
	}
	// A byte written as its value takes exactly two digits after the x, so the
	// braced form Go's own expressions allow is refused too.
	if strings.Contains(written, `\x{`) {
		return nil, badRegex(str, "illegal escape sequence")
	}
	written = plainClassBrackets(written)
	var out []body
	if !str.Mods.Wide || str.Mods.ASCII {
		out = append(out, body{text: withRegexFlags(str, written)})
	}
	if str.Mods.Wide {
		lifted, settled := liftWordEdges(written)
		if !settled {
			return nil, badRegex(str, "a word boundary looked for wide must be at "+
				"one end of the pattern or between two plain characters")
		}
		widened := neverMet
		if !lifted.impossible {
			var ok bool
			if widened, ok = widenRegex(lifted.text); !ok {
				return nil, badRegex(str, "")
			}
		}
		out = append(out, body{
			text: withRegexFlags(str, widened), wide: true,
			startEdge: lifted.atStart, endEdge: lifted.atEnd,
		})
	}
	return out, nil
}

// zeroWidthMarks are the letters of an escape that stands between characters
// rather than matching one: the edges of a word, and the ends of the data.
const zeroWidthMarks = "bBAzZ"

// liftedEdges is what taking the word-edge marks out of a pattern came to: what
// is left of the pattern, what it now asks at each of its ends, and whether a
// mark in the middle turned out never to hold.
type liftedEdges struct {
	text           string
	atStart, atEnd wordEdge
	impossible     bool
}

// wordEdge is what a pattern asks for at one of its ends: nothing in
// particular, the edge of a word, or the absence of one.
type wordEdge int

const (
	edgeAny wordEdge = iota
	edgeBoundary
	edgeNoBoundary
)

// liftWordEdges takes the marks asking for the edge of a word out of a pattern
// that is to be looked for wide, since the edge there falls between whole wide
// characters and Go's own expressions look between single bytes.
//
// A mark at either end of the pattern is lifted out, to be checked against the
// data once a match is found. One in the middle has a character on each side
// within the pattern itself, so whether it holds is settled here: either it
// does, and the mark is simply dropped, or it cannot, and the pattern is left
// unable to match anything at all. A mark whose neighbours are not plain
// characters cannot be settled either way, and is turned away.
func liftWordEdges(text string) (liftedEdges, bool) {
	var out strings.Builder
	lifted := liftedEdges{atStart: edgeAny, atEnd: edgeAny}
	possible := true
	for i := 0; i < len(text); {
		switch {
		case isWordEdgeMark(text, i):
			mark := edgeBoundary
			if text[i+1] == 'B' {
				mark = edgeNoBoundary
			}
			switch {
			case !holdsACharacter(out.String()):
				lifted.atStart = mark
			case onlyEndMarksAfter(text, i+2):
				lifted.atEnd = mark
			default:
				holds, settled := middleEdgeHolds(mark, text, i)
				if !settled {
					return liftedEdges{}, false
				}
				possible = possible && holds
			}
			i += 2
		case text[i] == '\\':
			end, esc := escapeEnd(text, i)
			if !esc {
				return liftedEdges{}, false
			}
			out.WriteString(text[i:end])
			i = end
		case text[i] == '[':
			end := min(skipClass(text, i), len(text))
			out.WriteString(text[i:end])
			i = end
		default:
			out.WriteByte(text[i])
			i++
		}
	}
	// A mark in the middle that cannot hold leaves a pattern matching nothing at
	// all, which is an answer rather than a fault.
	lifted.text, lifted.impossible = out.String(), !possible
	return lifted, true
}

// isWordEdgeMark says whether a mark asking about the edge of a word begins at a
// place in a pattern.
func isWordEdgeMark(text string, i int) bool {
	return text[i] == '\\' && i+1 < len(text) &&
		(text[i+1] == 'b' || text[i+1] == 'B')
}

// holdsACharacter says whether what has been written of a pattern so far can
// match a character of its own. What only ties the pattern to somewhere does
// not, so a mark after it is still at the front.
func holdsACharacter(text string) bool {
	return strings.Trim(text, "^") != ""
}

// onlyEndMarksAfter says whether the rest of a pattern only ties it to the end,
// so a mark before that is still at the back.
func onlyEndMarksAfter(text string, i int) bool {
	return strings.Trim(text[i:], "$") == ""
}

// middleEdgeHolds settles a mark with a character on each side of it, and says
// whether it could be settled at all. Both neighbours have to be plain
// characters, since anything else could match either a letter or not.
func middleEdgeHolds(mark wordEdge, text string, i int) (holds, settled bool) {
	before, gotBefore := plainCharBefore(text, i)
	after, gotAfter := plainCharAfter(text, i+2)
	if !gotBefore || !gotAfter {
		return false, false
	}
	atEdge := isWordByte(before) != isWordByte(after)
	return (mark == edgeBoundary) == atEdge, true
}

// regexSpecial are the characters of a pattern that say something about it
// rather than standing for themselves.
const regexSpecial = `.*+?()[]{}|^$\`

// plainCharBefore gives the character a pattern matches just before a place in
// it, when that is a single plain character.
func plainCharBefore(text string, i int) (byte, bool) {
	if i == 0 || strings.IndexByte(regexSpecial, text[i-1]) >= 0 {
		return 0, false
	}
	// A character that something else made special is not a plain one.
	if i >= 2 && text[i-2] == '\\' {
		return 0, false
	}
	return text[i-1], true
}

// plainCharAfter gives the character a pattern matches just after a place in it,
// when that is a single plain character.
func plainCharAfter(text string, i int) (byte, bool) {
	if i >= len(text) || strings.IndexByte(regexSpecial, text[i]) >= 0 {
		return 0, false
	}
	// One that something following makes into more than itself is not plain.
	if i+1 < len(text) && strings.IndexByte("*+?{", text[i+1]) >= 0 {
		return 0, false
	}
	return text[i], true
}

// widenRegex rewrites a pattern so that every character it matches is followed
// by a null. Each piece is taken with its null as one thing, so that whatever
// says how many times it may repeat counts the pair rather than the character
// alone: written wide, `l+` is a run of `l` and a null, not a run of `l` with a
// single null after it.
func widenRegex(text string) (string, bool) {
	var out strings.Builder
	for i := 0; i < len(text); {
		switch c := text[i]; c {
		case '\\':
			end, ok := escapeEnd(text, i)
			if !ok {
				return "", false
			}
			// A mark that stands between characters rather than matching one
			// takes no null, and nothing repeats it either.
			if end == i+2 && strings.ContainsRune(zeroWidthMarks, rune(text[i+1])) {
				out.WriteString(text[i:end])
				i = end
				continue
			}
			i = widenPiece(&out, text[i:end], text, end)
		case '[':
			end := skipClass(text, i)
			if end > len(text) {
				return "", false
			}
			i = widenPiece(&out, text[i:end], text, end)
		case '(':
			// Brackets match no character of their own, so they take no null.
			out.WriteByte(c)
			i++
		case ')':
			// A group matches no character of its own, so it takes no null, but
			// whatever says how many times it repeats still follows it.
			out.WriteByte(c)
			i = copyRepeat(&out, text, i+1)
		case '|', '^', '$':
			// What holds pieces together, and what ties them to the ends of the
			// data, matches no character of its own either.
			out.WriteByte(c)
			i++
		default:
			i = widenPiece(&out, text[i:i+1], text, i+1)
		}
	}
	return out.String(), true
}

// escapeEnd says where an escape ends. Most name one character in two bytes,
// but a byte written as its value takes two digits after the x, or as many as
// are wanted within braces.
func escapeEnd(text string, i int) (int, bool) {
	if i+1 >= len(text) {
		return 0, false
	}
	if text[i+1] != 'x' {
		return i + 2, true
	}
	if i+2 < len(text) && text[i+2] == '{' {
		end := i + 3
		for end < len(text) && text[end] != '}' {
			end++
		}
		if end == len(text) {
			return 0, false
		}
		return end + 1, true
	}
	// Two digits follow the x.
	const digits = 2
	if i+2+digits > len(text) {
		return 0, false
	}
	return i + 2 + digits, true
}

// widenPiece writes one piece of a pattern with a null after it, then copies
// whatever says how many times that pair may repeat.
func widenPiece(out *strings.Builder, piece, text string, i int) int {
	out.WriteString("(?:")
	out.WriteString(piece)
	out.WriteString(`\x00)`)
	return copyRepeat(out, text, i)
}

// copyRepeat copies whatever follows a piece to say how many times it repeats,
// which is left as it was written since it now counts the pair.
func copyRepeat(out *strings.Builder, text string, i int) int {
	if i >= len(text) {
		return i
	}
	switch text[i] {
	case '*', '+', '?':
		out.WriteByte(text[i])
		i++
	case '{':
		end := i
		for end < len(text) && text[end] != '}' {
			end++
		}
		if end == len(text) {
			// Nothing closes it, so it is a plain character rather than a
			// count, and it was already written with a null after it.
			return i
		}
		out.WriteString(text[i : end+1])
		i = end + 1
	default:
		return i
	}
	// A repeat may be followed by a mark asking it to take as little as it can.
	if i < len(text) && text[i] == '?' {
		out.WriteByte('?')
		i++
	}
	return i
}

// textBodies builds the shapes of a plain string: each width it was asked for,
// and each key it may have been hidden with.
func textBodies(str *String) []body {
	raw := []byte(str.Text)
	var out []body
	for _, key := range xorKeys(str.Mods) {
		hidden := xorBytes(raw, key)
		if !str.Mods.Wide || str.Mods.ASCII {
			out = append(out, body{text: literalPattern(hidden, str.Mods.Nocase)})
		}
		if str.Mods.Wide {
			out = append(out, body{text: literalPattern(widenBytes(hidden), str.Mods.Nocase), wide: true})
		}
	}
	return out
}

// xorKeys gives the keys a string may have been hidden with, which is just the
// one that changes nothing unless the rule asked for more.
func xorKeys(mods Modifiers) []byte {
	if !mods.XOR {
		return []byte{0}
	}
	keys := make([]byte, 0, mods.XORMax-mods.XORMin+1)
	for key := mods.XORMin; key <= mods.XORMax; key++ {
		keys = append(keys, byte(key)) // #nosec G115 -- a key is one byte, checked when parsed
	}
	return keys
}

// xorBytes hides a string under a key, which is what the xor modifier looks for.
func xorBytes(raw []byte, key byte) []byte {
	if key == 0 {
		return raw
	}
	out := make([]byte, len(raw))
	for i, c := range raw {
		out[i] = c ^ key
	}
	return out
}

// widenBytes puts a null after every byte, which is how the wide modifier
// writes a string.
func widenBytes(raw []byte) []byte {
	out := make([]byte, 0, 2*len(raw))
	for _, c := range raw {
		out = append(out, c, 0)
	}
	return out
}

// literalPattern turns bytes into a pattern matching exactly them. Asked to
// ignore case, each letter is written as the pair it could be, which is exact
// where a case-insensitive match over characters would not be.
func literalPattern(raw []byte, nocase bool) string {
	var b strings.Builder
	for _, c := range raw {
		if nocase && isLetter(c) {
			b.WriteString("[" + escapeByte(c|0x20) + escapeByte(c&^0x20) + "]")
			continue
		}
		b.WriteString(escapeByte(c))
	}
	return b.String()
}

// escapeByte writes one byte as a pattern matching just it.
func escapeByte(c byte) string { return fmt.Sprintf(`\x{%02x}`, c) }

// regexPattern maps a regular expression into the space patterns match in. What
// was written stays as it was, since everything a rule may write is either
// plain ASCII or an escape naming a byte.
func withRegexFlags(str *String, pattern string) string {
	var flags string
	if strings.Contains(str.Flags, "i") || str.Mods.Nocase {
		flags += "i"
	}
	if strings.Contains(str.Flags, "s") {
		flags += "s"
	}
	if flags == "" {
		return pattern
	}
	return "(?" + flags + ":" + pattern + ")"
}

// hexParser walks the body of a braced hex pattern.
type hexParser struct {
	src  string
	pos  int
	line int
}

// hexPattern turns a braced hex body into a pattern in the mapped space.
func hexPattern(str *String) (string, error) {
	p := &hexParser{src: str.Text, line: str.Line}
	out, err := p.sequence()
	if err != nil {
		return "", err
	}
	if p.skipSpace(); p.pos < len(p.src) {
		return "", p.fail()
	}
	return out, nil
}

// fail reports a hex pattern that does not make sense, which libyara reports
// like any other fault in the shape of the rules.
func (p *hexParser) fail() error {
	return &compileError{line: p.line, msg: "syntax error"}
}

// skipSpace moves past whitespace.
func (p *hexParser) skipSpace() {
	for p.pos < len(p.src) && (p.src[p.pos] == ' ' || p.src[p.pos] == '\t' ||
		p.src[p.pos] == '\n' || p.src[p.pos] == '\r') {
		p.pos++
	}
}

// sequence reads items until the end of the body or the end of a group.
func (p *hexParser) sequence() (string, error) {
	var b strings.Builder
	for {
		p.skipSpace()
		if p.pos >= len(p.src) || p.src[p.pos] == ')' || p.src[p.pos] == '|' {
			return b.String(), nil
		}
		piece, err := p.item()
		if err != nil {
			return "", err
		}
		b.WriteString(piece)
	}
}

// item reads one piece of a hex pattern: a byte, a byte with either half left
// open, a jump over any bytes at all, or a group of alternatives.
func (p *hexParser) item() (string, error) {
	switch p.src[p.pos] {
	case '[':
		return p.jump()
	case '(':
		return p.group()
	}
	return p.byteItem()
}

// byteItem reads two hex digits, either of which may be a question mark
// standing for any value.
func (p *hexParser) byteItem() (string, error) {
	if p.pos+2 > len(p.src) {
		return "", p.fail()
	}
	high, low := p.src[p.pos], p.src[p.pos+1]
	if !isHexDigitOrAny(high) || !isHexDigitOrAny(low) {
		return "", p.fail()
	}
	p.pos += 2

	switch {
	case high == '?' && low == '?':
		return anyByte, nil
	case low == '?':
		// The high half is fixed, so the byte is somewhere in a run of sixteen.
		start := hexValue(high) << 4
		return fmt.Sprintf(`[\x{%02x}-\x{%02x}]`, start, start+0x0f), nil
	case high == '?':
		// The low half is fixed, so the byte is one of sixteen spread out.
		var b strings.Builder
		b.WriteString("[")
		for top := range 16 {
			// #nosec G115 -- a nibble in each half, so never more than a byte
			b.WriteString(escapeByte(byte(top<<4 | hexValue(low))))
		}
		b.WriteString("]")
		return b.String(), nil
	}
	// #nosec G115 -- two hex digits, so never more than a byte
	return escapeByte(byte(hexValue(high)<<4 | hexValue(low))), nil
}

// jump reads a stretch of bytes to skip over, written as a range in brackets.
func (p *hexParser) jump() (string, error) {
	end := strings.IndexByte(p.src[p.pos:], ']')
	if end < 0 {
		return "", p.fail()
	}
	inside := strings.TrimSpace(p.src[p.pos+1 : p.pos+end])
	p.pos += end + 1

	low, high, ranged := strings.Cut(inside, "-")
	low, high = strings.TrimSpace(low), strings.TrimSpace(high)
	switch {
	case !ranged:
		// A single number, which is an exact number of bytes.
		n, err := hexJumpBound(low)
		if err != nil {
			return "", p.fail()
		}
		return fmt.Sprintf("%s{%d}", anyByte, n), nil
	case low == "" && high == "":
		return anyByte + "*", nil
	case high == "":
		n, err := hexJumpBound(low)
		if err != nil {
			return "", p.fail()
		}
		return fmt.Sprintf("%s{%d,}", anyByte, n), nil
	}
	from, err := hexJumpBound(low)
	if err != nil {
		return "", p.fail()
	}
	to, err := hexJumpBound(high)
	if err != nil {
		return "", p.fail()
	}
	return fmt.Sprintf("%s{%d,%d}", anyByte, from, to), nil
}

// hexJumpBound reads one end of a jump, which libyara writes in base ten.
func hexJumpBound(s string) (int, error) {
	if s == "" {
		return 0, fmt.Errorf("empty bound")
	}
	n := 0
	for i := range len(s) {
		if s[i] < '0' || s[i] > '9' {
			return 0, fmt.Errorf("%q is not a number", s)
		}
		n = n*10 + int(s[i]-'0')
	}
	return n, nil
}

// group reads a bracketed set of alternatives.
func (p *hexParser) group() (string, error) {
	p.pos++ // the opening bracket
	var options []string
	for {
		option, err := p.sequence()
		if err != nil {
			return "", err
		}
		// libyara's hex grammar wants at least one token in every branch, so an
		// empty one is a fault rather than something optional.
		if option == "" {
			return "", p.fail()
		}
		options = append(options, option)
		p.skipSpace()
		if p.pos >= len(p.src) {
			return "", p.fail()
		}
		if p.src[p.pos] == ')' {
			p.pos++
			return "(?:" + strings.Join(options, "|") + ")", nil
		}
		// A branch only ever stops at a bar, a closing bracket or the end, and
		// the other two are already dealt with.
		p.pos++
	}
}

// isHexDigitOrAny reports whether a character is a hex digit or the question
// mark that stands for any value.
func isHexDigitOrAny(c byte) bool {
	return c == '?' || (c >= '0' && c <= '9') ||
		(lower(c) >= 'a' && lower(c) <= 'f')
}

// hexValue reads one hex digit.
func hexValue(c byte) int {
	if c >= '0' && c <= '9' {
		return int(c - '0')
	}
	return int(lower(c)-'a') + 10
}

// compileRegexLit builds a regular expression written into a condition, in the
// space patterns are matched in.
func compileRegexLit(re RegexLit) (*regexp.Regexp, error) {
	pattern := latin1([]byte(re.Body))
	var flags string
	if strings.Contains(re.Flags, "i") {
		flags += "i"
	}
	if strings.Contains(re.Flags, "s") {
		flags += "s"
	}
	if flags != "" {
		pattern = "(?" + flags + ":" + pattern + ")"
	}
	return regexp.Compile(pattern)
}

// base64Bodies builds the shapes of a string encoded as base64.
//
// A string can sit at any of three places within the groups of three bytes that
// base64 encodes, and each place gives different characters, so all three are
// looked for. The characters at either end are dropped: they are made partly
// from whatever sits beside the string, which is not known.
func base64Bodies(str *String) ([]body, error) {
	if str.Mods.Base64Alphabet != "" && len(str.Mods.Base64Alphabet) != base64AlphabetSize {
		return nil, &compileError{
			line: str.Line,
			msg:  fmt.Sprintf("a base64 alphabet must be %d characters", base64AlphabetSize),
		}
	}
	alphabet := base64.StdEncoding
	if str.Mods.Base64Alphabet != "" {
		alphabet = base64.NewEncoding(str.Mods.Base64Alphabet).WithPadding(base64.NoPadding)
	}

	var out []body
	for _, form := range base64Forms([]byte(str.Text), alphabet) {
		raw := []byte(form)
		if str.Mods.Base64 {
			out = append(out, body{text: literalPattern(raw, false)})
		}
		if str.Mods.Base64Wide {
			out = append(out, body{text: literalPattern(widenBytes(raw), false), wide: true})
		}
	}
	return out, nil
}

// base64AlphabetSize is how many characters an alphabet must offer.
const base64AlphabetSize = 64

// base64Group is how many bytes base64 encodes at a time.
const base64Group = 3

// base64Forms gives the three encodings of a string, each trimmed to the
// characters that depend on the string alone.
func base64Forms(raw []byte, alphabet *base64.Encoding) []string {
	var out []string
	for shift := range base64Group {
		padded := append(make([]byte, shift), raw...)
		encoded := strings.TrimRight(alphabet.EncodeToString(padded), "=")

		// The opening characters are made only of padding, and the closing ones
		// partly of whatever would follow the string.
		from := (shift*8 + 5) / 6
		to := min((shift+len(raw))*8/6, len(encoded))
		if to > from {
			out = append(out, encoded[from:to])
		}
	}
	return out
}
