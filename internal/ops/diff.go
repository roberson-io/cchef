package ops

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(Diff{})
}

var (
	reDiffLine     = regexp.MustCompile(`[^\n]*\n|[^\n]+`)
	reDiffSentence = regexp.MustCompile(`\S.+?[.!?]`)
	reDiffCSS      = regexp.MustCompile(`[{}:;,]|\s+`)
	reDiffWord     = regexp.MustCompile(`\s+|\S+`)
)

// Diff compares two samples and highlights the differences.
type Diff struct{}

// Meta returns the operation metadata.
func (Diff) Meta() core.OpMeta {
	return core.OpMeta{
		Name:   "Diff",
		Module: "Diff",
		Description: "Compares two inputs (separated by the specified delimiter) and highlights the differences between them.\n\n" +
			"Note: diffs are computed with a Go diff library rather than CyberChef's jsdiff. All modes match CyberChef " +
			"except Word with 'Ignore whitespace', which may attach trailing whitespace to a change differently.",
		InfoURL:    "https://wikipedia.org/wiki/File_comparison",
		InputType:  core.TypeString,
		OutputType: core.TypeString,
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

// Run computes the diff. Ported from CyberChef Diff.mjs (jsdiff replaced by go-diff).
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

	diffs, err := computeDiffs(diffBy, samples[0], samples[1], ignoreWhitespace)
	if err != nil {
		return nil, err
	}
	out := renderDiffs(diffs, showAdded, showRemoved, showSubtraction)
	return core.NewDish([]byte(out), core.TypeString), nil
}

// computeDiffs diffs samples a and b at the granularity selected by diffBy.
func computeDiffs(diffBy, a, b string, ignoreWhitespace bool) ([]diffmatchpatch.Diff, error) {
	identity := func(s string) string { return s }
	switch diffBy {
	case "Character":
		dmp := diffmatchpatch.New()
		return dmp.DiffMain(a, b, false), nil
	case "Word":
		key := identity
		if ignoreWhitespace {
			key = diffWordWSKey
		}
		return tokenDiff(diffTokenize(a, reDiffWord), diffTokenize(b, reDiffWord), key), nil
	case "Line":
		key := identity
		if ignoreWhitespace {
			key = strings.TrimSpace
		}
		return tokenDiff(diffTokenize(a, reDiffLine), diffTokenize(b, reDiffLine), key), nil
	case "Sentence":
		return tokenDiff(sentenceTokenize(a), sentenceTokenize(b), identity), nil
	case "CSS":
		return tokenDiff(splitKeep(reDiffCSS, a), splitKeep(reDiffCSS, b), identity), nil
	case "JSON":
		// CyberChef's diffJson here behaves as a line diff over the raw input.
		return tokenDiff(diffTokenize(a, reDiffLine), diffTokenize(b, reDiffLine), identity), nil
	default:
		return nil, fmt.Errorf("invalid 'Diff by' option")
	}
}

// renderDiffs turns a diff into CyberChef's HTML markup, honouring the show
// flags (<ins> for additions, <del> for removals, equal text unless subtracting).
func renderDiffs(diffs []diffmatchpatch.Diff, showAdded, showRemoved, showSubtraction bool) string {
	var b strings.Builder
	for _, d := range diffs {
		switch d.Type {
		case diffmatchpatch.DiffInsert:
			if showAdded {
				b.WriteString("<ins>" + escapeHTML(d.Text) + "</ins>")
			}
		case diffmatchpatch.DiffDelete:
			if showRemoved {
				b.WriteString("<del>" + escapeHTML(d.Text) + "</del>")
			}
		case diffmatchpatch.DiffEqual:
			if !showSubtraction {
				b.WriteString(escapeHTML(d.Text))
			}
		}
	}
	return b.String()
}

// diffTokenize returns all non-overlapping matches of re (used for word/line modes).
func diffTokenize(s string, re *regexp.Regexp) []string {
	return re.FindAllString(s, -1)
}

// sentenceTokenize splits text into alternating [gap, sentence, gap, ...] tokens,
// approximating jsdiff's sentence tokenizer (which uses a lookahead RE2 cannot express).
func sentenceTokenize(s string) []string {
	var out []string
	last := 0
	for _, m := range reDiffSentence.FindAllStringIndex(s, -1) {
		out = append(out, s[last:m[0]])
		out = append(out, s[m[0]:m[1]])
		last = m[1]
	}
	out = append(out, s[last:])
	return out
}

// splitKeep mirrors JavaScript's String.split with a capturing group: it returns the
// text between matches interleaved with the matched delimiters.
func splitKeep(re *regexp.Regexp, s string) []string {
	var out []string
	last := 0
	for _, m := range re.FindAllStringIndex(s, -1) {
		out = append(out, s[last:m[0]])
		out = append(out, s[m[0]:m[1]])
		last = m[1]
	}
	out = append(out, s[last:])
	return out
}

// tokenDiff diffs two token slices at token granularity. Each distinct token key
// (computed by keyFn) is mapped to a rune; the rune diff is then reconstructed from
// the original A/B token streams. Equal/insert segments use B's tokens, deletes use A's,
// so whitespace-insensitive comparisons (via keyFn) still emit the right original text.
func tokenDiff(a, b []string, keyFn func(string) string) []diffmatchpatch.Diff {
	enc := map[string]rune{}
	next := rune(0)
	encode := func(toks []string) []rune {
		rs := make([]rune, len(toks))
		for i, t := range toks {
			k := keyFn(t)
			r, ok := enc[k]
			if !ok {
				r = next
				enc[k] = r
				next++
				if next == 0xD800 { // skip the UTF-16 surrogate range
					next = 0xE000
				}
			}
			rs[i] = r
		}
		return rs
	}
	ra, rb := encode(a), encode(b)

	dmp := diffmatchpatch.New()
	raw := dmp.DiffMainRunes(ra, rb, false)

	var out []diffmatchpatch.Diff
	ai, bi := 0, 0
	for _, d := range raw {
		n := len([]rune(d.Text))
		switch d.Type {
		case diffmatchpatch.DiffEqual:
			out = append(out, diffmatchpatch.Diff{Type: d.Type, Text: strings.Join(b[bi:bi+n], "")})
			ai += n
			bi += n
		case diffmatchpatch.DiffDelete:
			out = append(out, diffmatchpatch.Diff{Type: d.Type, Text: strings.Join(a[ai:ai+n], "")})
			ai += n
		case diffmatchpatch.DiffInsert:
			out = append(out, diffmatchpatch.Diff{Type: d.Type, Text: strings.Join(b[bi:bi+n], "")})
			bi += n
		}
	}
	return out
}

// diffWordWSKey normalises whitespace-only word tokens to a single space so that
// whitespace-only changes are ignored (approximates jsdiff diffWords).
func diffWordWSKey(t string) string {
	if strings.TrimSpace(t) == "" {
		return " "
	}
	return t
}
