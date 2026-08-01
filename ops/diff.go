package ops

import (
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(Diff{})
}

// Character classes, written out because Go's \s is narrower than JavaScript's.
const (
	// diffWordChars is what counts as part of a word: ASCII alphanumerics and
	// underscore, plus the Latin script blocks, minus the multiplication and
	// division signs and the spacing modifiers that look like accents.
	diffWordChars = `a-zA-Z0-9_` +
		`\x{AD}\x{C0}-\x{D6}\x{D8}-\x{F6}\x{F8}-\x{2C6}\x{2C8}-\x{2D7}\x{2DE}-\x{2FF}\x{1E00}-\x{1EFF}`
	// diffSpaceChars is JavaScript's \s.
	diffSpaceChars = `\t\n\v\f\r ` +
		`\x{A0}\x{1680}\x{2000}-\x{200A}\x{2028}\x{2029}\x{202F}\x{205F}\x{3000}\x{FEFF}`
	// diffInlineSpaceChars is JavaScript's \s without the two characters that
	// end a line, so a newline can be a token of its own.
	diffInlineSpaceChars = `\t\v\f ` +
		`\x{A0}\x{1680}\x{2000}-\x{200A}\x{2028}\x{2029}\x{202F}\x{205F}\x{3000}\x{FEFF}`
)

var (
	// A word run, a run of space, or one other character. Runs of space get
	// their own match here and are stitched onto their neighbours afterwards.
	reDiffWords = regexp.MustCompile(
		`[` + diffWordChars + `]+|[` + diffSpaceChars + `]+|[^` + diffWordChars + `]`)
	// As above, except that each line ending is a token in its own right rather
	// than being merged into the space around it.
	reDiffWordsWithSpace = regexp.MustCompile(
		`\r?\n|[` + diffWordChars + `]+|[` + diffInlineSpaceChars + `]+|[^` + diffWordChars + `]`)
	// The separators CSS is split on, which are kept as tokens themselves.
	reDiffCSSSeparators = regexp.MustCompile(`[{}:;,]|[` + diffSpaceChars + `]+`)
	// A comma at the end of a line, which two lines of pretty-printed JSON are
	// allowed to differ by.
	reDiffTrailingComma = regexp.MustCompile(`,([\r\n])`)
)

// diffChange is one run of text in a result: added, removed, or common to both
// samples.
type diffChange struct {
	Text    string
	Added   bool
	Removed bool
}

// diffKind is one granularity: how a sample is split into tokens, when two
// tokens count as the same, how tokens are put back together, and any tidying
// applied to the finished result.
type diffKind struct {
	tokenize    func(string) []string
	equals      func(a, b string) bool
	join        func([]string) string
	postProcess func([]diffChange) []diffChange
	// useLongestToken keeps the longer of two equal tokens, which is how a line
	// with a trailing comma wins over the same line without one.
	useLongestToken bool
}

// diffKindFor returns the granularity named by the "Diff by" argument.
func diffKindFor(diffBy string, ignoreWhitespace bool) (diffKind, error) {
	exact := func(a, b string) bool { return a == b }
	trimmed := func(a, b string) bool { return jsTrimSpace(a) == jsTrimSpace(b) }
	concat := func(tokens []string) string { return strings.Join(tokens, "") }

	switch diffBy {
	case "Character":
		return diffKind{tokenize: diffTokenizeChars, equals: exact, join: concat}, nil
	case "Word":
		if ignoreWhitespace {
			return diffKind{
				tokenize:    diffTokenizeWords,
				equals:      trimmed,
				join:        diffJoinWords,
				postProcess: diffDedupeWordWhitespace,
			}, nil
		}
		return diffKind{tokenize: diffTokenizeWordsWithSpace, equals: exact, join: concat}, nil
	case "Line":
		if ignoreWhitespace {
			return diffKind{tokenize: diffTokenizeLines, equals: trimmed, join: concat}, nil
		}
		return diffKind{tokenize: diffTokenizeLines, equals: exact, join: concat}, nil
	case "Sentence":
		return diffKind{tokenize: diffTokenizeSentences, equals: exact, join: concat}, nil
	case "CSS":
		return diffKind{tokenize: diffTokenizeCSS, equals: exact, join: concat}, nil
	case "JSON":
		return diffKind{
			tokenize:        diffTokenizeLines,
			equals:          diffEqualsIgnoringTrailingComma,
			join:            concat,
			useLongestToken: true,
		}, nil
	default:
		return diffKind{}, fmt.Errorf("invalid 'Diff by' option")
	}
}

// diffEqualsIgnoringTrailingComma compares two lines of JSON, treating a comma
// at the end of a line as absent so that the last entry of an object matches
// the same entry somewhere in the middle.
func diffEqualsIgnoringTrailingComma(a, b string) bool {
	return reDiffTrailingComma.ReplaceAllString(a, "$1") == reDiffTrailingComma.ReplaceAllString(b, "$1")
}

// diffTokenizeChars splits into code points.
func diffTokenizeChars(s string) []string {
	var out []string
	for _, r := range s {
		out = append(out, string(r))
	}
	return out
}

// diffTokenizeWords splits into words and punctuation, each carrying the space
// around it. Space cannot be dropped, or the original text could not be
// rebuilt; nor can it be a token of its own, or a diff would happily keep the
// gaps between words while replacing the words themselves.
func diffTokenizeWords(s string) []string {
	var tokens []string
	prev := ""
	first := true
	for _, part := range reDiffWords.FindAllString(s, -1) {
		switch {
		case diffHasSpace(part):
			if first {
				tokens = append(tokens, part)
			} else {
				tokens[len(tokens)-1] += part
			}
		case !first && diffHasSpace(prev):
			// The space before this part already belongs to the previous token
			// unless that token is nothing but the space, in which case this
			// part joins it.
			if tokens[len(tokens)-1] == prev {
				tokens[len(tokens)-1] += part
			} else {
				tokens = append(tokens, prev+part)
			}
		default:
			tokens = append(tokens, part)
		}
		prev = part
		first = false
	}
	return tokens
}

// diffTokenizeWordsWithSpace splits into words, punctuation and runs of space,
// each its own token.
func diffTokenizeWordsWithSpace(s string) []string {
	return reDiffWordsWithSpace.FindAllString(s, -1)
}

// diffTokenizeLines splits into lines, each keeping the newline that ends it.
func diffTokenizeLines(s string) []string {
	var pieces []string
	last := 0
	for i := range len(s) {
		if s[i] == '\n' {
			pieces = append(pieces, s[last:i], "\n")
			last = i + 1
		}
	}
	pieces = append(pieces, s[last:])
	// Text ending in a newline leaves an empty final piece, which is not a line.
	if pieces[len(pieces)-1] == "" {
		pieces = pieces[:len(pieces)-1]
	}

	var out []string
	for i, piece := range pieces {
		if i%2 == 1 {
			out[len(out)-1] += piece
		} else {
			out = append(out, piece)
		}
	}
	return out
}

// diffTokenizeSentences alternates sentences with the space that separates
// them. A sentence ends at a full stop, exclamation or question mark that is
// followed by space.
func diffTokenizeSentences(s string) []string {
	r := []rune(s)
	var out []string
	start := 0
	for i := 0; i < len(r); i++ {
		if i == len(r)-1 {
			out = append(out, string(r[start:]))
			break
		}
		if !diffEndsSentence(r[i]) || !mimeIsJSSpace(r[i+1]) {
			continue
		}
		out = append(out, string(r[start:i+1]))
		i++
		start = i
		for i+1 < len(r) && mimeIsJSSpace(r[i+1]) {
			i++
		}
		out = append(out, string(r[start:i+1]))
		start = i + 1
	}
	return out
}

// diffEndsSentence reports whether c is punctuation that can end a sentence.
func diffEndsSentence(c rune) bool {
	return c == '.' || c == '!' || c == '?'
}

// diffTokenizeCSS splits on braces, colons, semicolons, commas and space,
// keeping the separators as tokens of their own.
func diffTokenizeCSS(s string) []string {
	var out []string
	last := 0
	for _, m := range reDiffCSSSeparators.FindAllStringIndex(s, -1) {
		out = append(out, s[last:m[0]], s[m[0]:m[1]])
		last = m[1]
	}
	return append(out, s[last:])
}

// diffJoinWords concatenates word tokens. Every token but the first carries a
// copy of the space that ended its predecessor, so that copy is dropped.
func diffJoinWords(tokens []string) string {
	var b strings.Builder
	for i, t := range tokens {
		if i > 0 {
			t = t[len(diffLeadingWS(t)):]
		}
		b.WriteString(t)
	}
	return b.String()
}

// diffHasSpace reports whether s contains any whitespace at all.
func diffHasSpace(s string) bool {
	return strings.ContainsFunc(s, mimeIsJSSpace)
}

// diffLeadingWS returns the run of whitespace s starts with.
func diffLeadingWS(s string) string {
	return s[:len(s)-len(strings.TrimLeftFunc(s, mimeIsJSSpace))]
}

// diffTrailingWS returns the run of whitespace s ends with.
func diffTrailingWS(s string) string {
	return s[len(strings.TrimRightFunc(s, mimeIsJSSpace)):]
}

// diffLongestCommonPrefix returns the longest run of code points a and b both
// start with.
func diffLongestCommonPrefix(a, b string) string {
	ar, br := []rune(a), []rune(b)
	n := 0
	for n < len(ar) && n < len(br) && ar[n] == br[n] {
		n++
	}
	return string(ar[:n])
}

// diffLongestCommonSuffix returns the longest run of code points a and b both
// end with.
func diffLongestCommonSuffix(a, b string) string {
	ar, br := []rune(a), []rune(b)
	n := 0
	for n < len(ar) && n < len(br) && ar[len(ar)-1-n] == br[len(br)-1-n] {
		n++
	}
	return string(ar[len(ar)-n:])
}

// diffMaximumOverlap returns the longest run that is both a suffix of a and a
// prefix of b.
func diffMaximumOverlap(a, b string) string {
	ar, br := []rune(a), []rune(b)
	for n := min(len(ar), len(br)); n > 0; n-- {
		if string(ar[len(ar)-n:]) == string(br[:n]) {
			return string(br[:n])
		}
	}
	return ""
}

// diffPath is how far along the old sample one candidate edit script has
// reached, and the changes it took to get there, held back to front.
type diffPath struct {
	oldPos int
	last   *diffComponent
}

// diffComponent is a run of tokens all changed the same way, linked to the run
// before it.
type diffComponent struct {
	count          int
	added, removed bool
	prev           *diffComponent
}

// diffCompute finds the shortest edit script turning oldStr into newStr, by
// Myers's algorithm: try every edit script of length one, then every one of
// length two, and so on, keeping only the furthest each diagonal has reached.
// The first script to consume both samples is the shortest, so the search
// always ends — after at most as many rounds as there are tokens in the two
// samples together.
func diffCompute(oldStr, newStr string, k diffKind) []diffChange {
	oldTokens := diffNonEmpty(k.tokenize(oldStr))
	newTokens := diffNonEmpty(k.tokenize(newStr))
	oldLen, newLen := len(oldTokens), len(newTokens)

	start := &diffPath{oldPos: -1}
	newPos := diffExtendCommon(start, oldTokens, newTokens, 0, k.equals)
	if start.oldPos+1 >= oldLen && newPos+1 >= newLen {
		return diffFinish(start.last, oldTokens, newTokens, k)
	}

	// Reaching the right edge of the graph on diagonal d means the end is now
	// at most d edits away, so nothing beyond d is worth considering; likewise
	// for the bottom edge and the diagonals below. Myers's paper extends paths
	// off the edge instead, which costs a great deal on the common case of text
	// simply appended to the end.
	minDiagonal, maxDiagonal := math.MinInt, math.MaxInt
	paths := map[int]*diffPath{0: start}

	for editLength := 1; ; editLength++ {
		for d := max(minDiagonal, -editLength); d <= min(maxDiagonal, editLength); d += 2 {
			path, ok := diffStep(paths, d, oldLen, newLen)
			if !ok {
				continue
			}
			newPos = diffExtendCommon(path, oldTokens, newTokens, d, k.equals)
			if path.oldPos+1 >= oldLen && newPos+1 >= newLen {
				return diffFinish(path.last, oldTokens, newTokens, k)
			}
			paths[d] = path
			if path.oldPos+1 >= oldLen {
				maxDiagonal = min(maxDiagonal, d-1)
			}
			if newPos+1 >= newLen {
				minDiagonal = max(minDiagonal, d+1)
			}
		}
	}
}

// diffStep extends one of the two neighbouring diagonals onto diagonal d, by an
// insertion or a deletion. It reports false when neither neighbour can be
// extended, which makes diagonal d a dead end.
func diffStep(paths map[int]*diffPath, d, oldLen, newLen int) (*diffPath, bool) {
	removePath, addPath := paths[d-1], paths[d+1]
	if removePath != nil {
		// Nothing else will read this one.
		delete(paths, d-1)
	}

	canAdd := false
	if addPath != nil {
		posAfterAdd := addPath.oldPos - d
		canAdd = posAfterAdd >= 0 && posAfterAdd < newLen
	}
	canRemove := removePath != nil && removePath.oldPos+1 < oldLen
	if !canAdd && !canRemove {
		delete(paths, d)
		return nil, false
	}

	// Branch from whichever neighbour has got further through the old sample.
	if !canRemove || (canAdd && removePath.oldPos < addPath.oldPos) {
		return diffAppend(addPath, true, false, 0), true
	}
	return diffAppend(removePath, false, true, 1), true
}

// diffAppend records one more changed token on a copy of path, merging it into
// the run at the end when that run was changed the same way.
func diffAppend(path *diffPath, added, removed bool, oldPosInc int) *diffPath {
	last := path.last
	if last != nil && last.added == added && last.removed == removed {
		return &diffPath{
			oldPos: path.oldPos + oldPosInc,
			last:   &diffComponent{count: last.count + 1, added: added, removed: removed, prev: last.prev},
		}
	}
	return &diffPath{
		oldPos: path.oldPos + oldPosInc,
		last:   &diffComponent{count: 1, added: added, removed: removed, prev: last},
	}
}

// diffExtendCommon walks path forward over as many equal tokens as it can, for
// free, and returns how far through the new sample that leaves it.
func diffExtendCommon(path *diffPath, oldTokens, newTokens []string, diagonal int, equals func(a, b string) bool) int {
	oldPos := path.oldPos
	newPos := oldPos - diagonal
	common := 0
	for newPos+1 < len(newTokens) && oldPos+1 < len(oldTokens) &&
		equals(oldTokens[oldPos+1], newTokens[newPos+1]) {
		newPos++
		oldPos++
		common++
	}
	if common > 0 {
		path.last = &diffComponent{count: common, prev: path.last}
	}
	path.oldPos = oldPos
	return newPos
}

// diffFinish turns a finished path's back-to-front component list into changes
// carrying real text, then applies whatever tidying the granularity asks for.
func diffFinish(last *diffComponent, oldTokens, newTokens []string, k diffKind) []diffChange {
	var components []*diffComponent
	for c := last; c != nil; c = c.prev {
		components = append(components, c)
	}

	changes := make([]diffChange, 0, len(components))
	oldPos, newPos := 0, 0
	for i := len(components) - 1; i >= 0; i-- {
		c := components[i]
		if c.removed {
			changes = append(changes, diffChange{
				Text:    k.join(oldTokens[oldPos : oldPos+c.count]),
				Removed: true,
			})
			oldPos += c.count
			continue
		}
		tokens := newTokens[newPos : newPos+c.count]
		if !c.added && k.useLongestToken {
			tokens = diffLongestTokens(tokens, oldTokens[oldPos:oldPos+c.count])
		}
		changes = append(changes, diffChange{Text: k.join(tokens), Added: c.added})
		newPos += c.count
		if !c.added {
			oldPos += c.count
		}
	}

	if k.postProcess != nil {
		changes = k.postProcess(changes)
	}
	return changes
}

// diffLongestTokens pairs up two equal runs of tokens and keeps the longer of
// each pair.
func diffLongestTokens(newTokens, oldTokens []string) []string {
	out := make([]string, len(newTokens))
	for i, t := range newTokens {
		if diffTokenLength(oldTokens[i]) > diffTokenLength(t) {
			t = oldTokens[i]
		}
		out[i] = t
	}
	return out
}

// diffTokenLength counts the sixteen-bit units a token occupies, which is the
// length JavaScript compares.
func diffTokenLength(s string) int {
	n := 0
	for _, r := range s {
		n++
		if r > 0xFFFF {
			n++
		}
	}
	return n
}

// diffNonEmpty drops tokens with no text in them, which a splitter can produce
// where two separators meet.
func diffNonEmpty(tokens []string) []string {
	out := tokens[:0]
	for _, t := range tokens {
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

// diffDedupeWordWhitespace tidies the space around each change in word mode.
// Every token carries the space around it, so the space between a change and
// the text on either side is otherwise repeated in both.
func diffDedupeWordWhitespace(changes []diffChange) []diffChange {
	var lastKeep, insertion, deletion *diffChange
	for i := range changes {
		c := &changes[i]
		switch {
		case c.Added:
			insertion = c
		case c.Removed:
			deletion = c
		default:
			if insertion != nil || deletion != nil {
				diffDedupeBetween(lastKeep, deletion, insertion, c)
			}
			lastKeep, insertion, deletion = c, nil, nil
		}
	}
	if insertion != nil || deletion != nil {
		diffDedupeBetween(lastKeep, deletion, insertion, nil)
	}
	return changes
}

// diffDedupeBetween settles which of the changes around one edit owns the space
// between them. Between two kept runs there is either a deletion, an insertion,
// or one of each; either kept run is absent at the very start or end of the
// text.
func diffDedupeBetween(startKeep, deletion, insertion, endKeep *diffChange) {
	switch {
	case deletion != nil && insertion != nil:
		diffDedupeReplacement(startKeep, deletion, insertion, endKeep)
	case insertion != nil:
		// Every run's space came from the new text, so there is nothing to say
		// about what was added or removed; each run keeps its trailing space
		// and gives up its leading space.
		if startKeep != nil {
			insertion.Text = insertion.Text[len(diffLeadingWS(insertion.Text)):]
		}
		if endKeep != nil {
			endKeep.Text = endKeep.Text[len(diffLeadingWS(endKeep.Text)):]
		}
	case startKeep != nil && endKeep != nil:
		diffDedupeDeletion(startKeep, deletion, endKeep)
	case endKeep != nil:
		// At the start of the text: the kept run keeps all of its space, and
		// the deletion gives up whatever it repeats.
		overlap := diffMaximumOverlap(diffTrailingWS(deletion.Text), diffLeadingWS(endKeep.Text))
		deletion.Text = strings.TrimSuffix(deletion.Text, overlap)
	case startKeep != nil:
		// At the end of the text, the mirror image of the case above.
		overlap := diffMaximumOverlap(diffTrailingWS(startKeep.Text), diffLeadingWS(deletion.Text))
		deletion.Text = strings.TrimPrefix(deletion.Text, overlap)
	}
}

// diffDedupeReplacement handles a deletion and an insertion together: space the
// old and new text agree on belongs to the kept run beside it, and neither the
// deletion nor the insertion should repeat it.
func diffDedupeReplacement(startKeep, deletion, insertion, endKeep *diffChange) {
	oldPrefix, oldSuffix := diffLeadingWS(deletion.Text), diffTrailingWS(deletion.Text)
	newPrefix, newSuffix := diffLeadingWS(insertion.Text), diffTrailingWS(insertion.Text)

	if startKeep != nil {
		common := diffLongestCommonPrefix(oldPrefix, newPrefix)
		startKeep.Text = strings.TrimSuffix(startKeep.Text, newPrefix) + common
		deletion.Text = strings.TrimPrefix(deletion.Text, common)
		insertion.Text = strings.TrimPrefix(insertion.Text, common)
	}
	if endKeep != nil {
		common := diffLongestCommonSuffix(oldSuffix, newSuffix)
		endKeep.Text = common + strings.TrimPrefix(endKeep.Text, newSuffix)
		deletion.Text = strings.TrimSuffix(deletion.Text, common)
		insertion.Text = strings.TrimSuffix(insertion.Text, common)
	}
}

// diffDedupeDeletion handles a deletion between two kept runs. Space the
// deletion shares with the run after it is handed to whichever run comes first,
// and anything left over goes to the run before.
func diffDedupeDeletion(startKeep, deletion, endKeep *diffChange) {
	newSpace := diffLeadingWS(endKeep.Text)
	delStart, delEnd := diffLeadingWS(deletion.Text), diffTrailingWS(deletion.Text)

	toStart := diffLongestCommonPrefix(newSpace, delStart)
	deletion.Text = strings.TrimPrefix(deletion.Text, toStart)

	toEnd := diffLongestCommonSuffix(strings.TrimPrefix(newSpace, toStart), delEnd)
	deletion.Text = strings.TrimSuffix(deletion.Text, toEnd)

	endKeep.Text = toEnd + strings.TrimPrefix(endKeep.Text, newSpace)
	startKeep.Text = strings.TrimSuffix(startKeep.Text, newSpace) + newSpace[:len(newSpace)-len(toEnd)]
}

// diffRender turns changes into CyberChef's markup, honouring the show flags.
func diffRender(changes []diffChange, showAdded, showRemoved, showSubtraction bool) string {
	var b strings.Builder
	for _, c := range changes {
		switch {
		case c.Added:
			if showAdded {
				b.WriteString("<ins>" + escapeHTML(c.Text) + "</ins>")
			}
		case c.Removed:
			if showRemoved {
				b.WriteString("<del>" + escapeHTML(c.Text) + "</del>")
			}
		case !showSubtraction:
			b.WriteString(escapeHTML(c.Text))
		}
	}
	return b.String()
}

// Diff compares two samples and highlights the differences.
type Diff struct{}

// Meta returns the operation metadata.
func (Diff) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Diff",
		Module:      "Diff",
		Description: "Compares two inputs (separated by the specified delimiter) and highlights the differences between them.",
		InfoURL:     "https://wikipedia.org/wiki/File_comparison",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (Diff) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Sample delimiter", Type: core.ArgString, Value: `\n\n`},
		{Name: "Diff by", Type: core.ArgOption, Value: []string{"Character", "Word", "Line", "Sentence", "CSS", "JSON"}},
		{Name: "Show added", Type: core.ArgBoolean, Value: true},
		{Name: "Show removed", Type: core.ArgBoolean, Value: true},
		{Name: "Show subtraction", Type: core.ArgBoolean, Value: false},
		{Name: "Ignore whitespace", Type: core.ArgBoolean, Value: false},
	}
}

// Run computes the diff.
func (Diff) Run(in *core.Dish, args []any) (*core.Dish, error) {
	sampleDelim := parseEscapedChars(args[0].(string))
	diffBy := args[1].(string)
	showAdded := args[2].(bool)
	showRemoved := args[3].(bool)
	showSubtraction := args[4].(bool)
	ignoreWhitespace := args[5].(bool)

	samples := strings.Split(in.String(), sampleDelim)
	if len(samples) != 2 {
		return nil, fmt.Errorf("incorrect number of samples, perhaps you need to modify the sample delimiter or add more samples?")
	}

	kind, err := diffKindFor(diffBy, ignoreWhitespace)
	if err != nil {
		return nil, err
	}
	out := diffRender(diffCompute(samples[0], samples[1], kind), showAdded, showRemoved, showSubtraction)
	return core.NewDish([]byte(out), core.TypeString), nil
}
