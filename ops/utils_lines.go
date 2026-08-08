package ops

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/uregex"
)

// inputDelims are the delimiter options for line-oriented operations (CyberChef
// INPUT_DELIM_OPTIONS).
var inputDelims = []string{"Line feed", "CRLF", "Space", "Comma", "Semi-colon", "Colon", "Nothing (separate chars)"}

func init() {
	core.Register(Filter{})
	core.Register(Sort{})
	core.Register(Unique{})
}

// splitByDelim splits using charRep semantics; "" delimiter splits into chars.
func splitByDelim(s, delim string) []string {
	if delim == "" {
		return strings.Split(s, "")
	}
	return strings.Split(s, delim)
}

// Filter keeps only the sections matching (or not matching) a regex.
type Filter struct{}

// Meta returns the operation metadata.
func (Filter) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Filter",
		Module:      "Default",
		Description: "Splits the input on the given delimiter and keeps only the sections matching the regular expression (or the non-matching sections, if inverted).",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (Filter) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Delimiter", Type: core.ArgOption, Value: inputDelims},
		{Name: "Regex", Type: core.ArgString, Value: ""},
		{Name: "Invert condition", Type: core.ArgBoolean, Value: false},
	}
}

// Run filters the sections.
func (Filter) Run(in *core.Dish, args []any) (*core.Dish, error) {
	delim := charRep(args[0].(string))
	re, err := uregex.Compile(args[1].(string))
	if err != nil {
		return nil, fmt.Errorf("invalid regex: %w", err)
	}
	invert := args[2].(bool)

	var kept []string
	for _, section := range splitByDelim(in.String(), delim) {
		if re.MatchString(section) != invert {
			kept = append(kept, section)
		}
	}
	return core.NewDish([]byte(strings.Join(kept, delim)), core.TypeString), nil
}

// Sort orders the sections of the input.
type Sort struct{}

// Meta returns the operation metadata.
func (Sort) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Sort",
		Module:      "Default",
		Description: "Sorts the sections of the input, split on the given delimiter, using the chosen ordering.",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (Sort) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Delimiter", Type: core.ArgOption, Value: inputDelims},
		{Name: "Reverse", Type: core.ArgBoolean, Value: false},
		{Name: "Order", Type: core.ArgOption, Value: []string{
			"Alphabetical (case sensitive)", "Alphabetical (case insensitive)",
			"IP address", "Numeric", "Numeric (hexadecimal)", "Length",
		}},
	}
}

// Run sorts the sections.
func (Sort) Run(in *core.Dish, args []any) (*core.Dish, error) {
	delim := charRep(args[0].(string))
	reverse := args[1].(bool)
	order := args[2].(string)

	sections := splitByDelim(in.String(), delim)
	var less func(a, b string) bool
	switch order {
	case "Alphabetical (case insensitive)":
		less = func(a, b string) bool { return strings.ToLower(a) < strings.ToLower(b) }
	case "IP address":
		less = ipLess
	case "Numeric":
		less = func(a, b string) bool { return naturalCompare(a, b, false) < 0 }
	case "Numeric (hexadecimal)":
		less = func(a, b string) bool { return naturalCompare(a, b, true) < 0 }
	case "Length":
		less = func(a, b string) bool { return len(a) < len(b) }
	default: // Alphabetical (case sensitive)
		less = func(a, b string) bool { return a < b }
	}
	sort.SliceStable(sections, func(i, j int) bool { return less(sections[i], sections[j]) })
	if reverse {
		for i, j := 0, len(sections)-1; i < j; i, j = i+1, j-1 {
			sections[i], sections[j] = sections[j], sections[i]
		}
	}
	return core.NewDish([]byte(strings.Join(sections, delim)), core.TypeString), nil
}

// ipLess compares two dotted-quad IPv4 strings numerically.
func ipLess(a, b string) bool {
	av, aok := ipToUint(a)
	bv, bok := ipToUint(b)
	switch {
	case !aok && bok:
		return false
	case aok && !bok:
		return true
	case !aok && !bok:
		return a < b
	default:
		return av < bv
	}
}

func ipToUint(s string) (uint64, bool) {
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return 0, false
	}
	var v uint64
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 || n > 255 {
			return 0, false
		}
		v = v<<8 | uint64(n)
	}
	return v, true
}

// naturalCompare compares two strings the way CyberChef's numericSort /
// hexadecimalSort do (Sort's "Numeric" and "Numeric (hexadecimal)" orders): each
// string is split on runs of non-(hex-)digits, keeping empty boundary segments,
// and compared segment by segment. Empty and whitespace segments coerce to the
// number 0 — mirroring JavaScript's isNaN("") === false — so a line beginning
// with text sorts before a line beginning with a number. When hex is true,
// numeric segments are parsed as hexadecimal.
//
// One CyberChef quirk is deliberately not reproduced: its loop bound is the
// second string's character length rather than its segment count, so for some
// inputs it reads past the segment array and compares against the literal string
// "undefined". That is undefined behaviour affecting only pathological mixed
// inputs (~1% in differential testing); we use the sane segment-count bound.
func naturalCompare(a, b string, hex bool) int {
	as, bs := sortSegments(a, hex), sortSegments(b, hex)
	for i := 0; i < len(as) && i < len(bs); i++ {
		if r, decided := compareSeg(as[i], bs[i]); decided {
			return r
		}
	}
	return localeCompareASCII(a, b)
}

// compareSeg compares two sort segments, returning (result, decided). decided is
// false when the segments are equal and the caller should move to the next pair.
// Note: sortSegments always aligns numeric/separator segments positionally, so
// naturalCompare never calls this with segments of differing isNum — but the
// cross-type arms are correct comparisons in their own right (a non-numeric
// segment sorts after a numeric one) and are covered directly by tests.
func compareSeg(x, y numSeg) (int, bool) {
	switch {
	case !x.isNum && y.isNum:
		return 1, true
	case x.isNum && !y.isNum:
		return -1, true
	case x.isNum && y.isNum:
		if x.num != y.num {
			if x.num < y.num {
				return -1, true
			}
			return 1, true
		}
		return 0, false
	default:
		return localeCompareASCII(x.str, y.str), x.str != y.str
	}
}

// localeCompareASCII approximates JavaScript's String.prototype.localeCompare for
// ASCII text (used by CyberChef's numericSort): strings are ordered
// case-insensitively, and when they differ only in case the lowercase form sorts
// first. This matches the default ICU collation for the alphanumeric inputs sort
// handles; it does not reproduce full Unicode collation of punctuation or accents.
func localeCompareASCII(a, b string) int {
	if c := strings.Compare(strings.ToLower(a), strings.ToLower(b)); c != 0 {
		return c
	}
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] == b[i] {
			continue
		}
		aLower := a[i] >= 'a' && a[i] <= 'z'
		bLower := b[i] >= 'a' && b[i] <= 'z'
		// The loop only runs when ToLower(a)==ToLower(b), so any differing byte is
		// the same letter in opposite case; one of the two case branches always
		// fires and the raw byte-order comparisons below cannot be reached.
		switch {
		case aLower && !bLower:
			return -1
		case !aLower && bLower:
			return 1
		case a[i] < b[i]:
			return -1
		default:
			return 1
		}
	}
	return strings.Compare(a, b)
}

type numSeg struct {
	str   string
	num   float64
	isNum bool
}

// sortSegments splits s the way JavaScript's String.split(/([^\d]+)/) (or the
// /([^\da-f]+)/i hex variant) does: alternating (possibly empty) digit-runs and
// non-digit separators, with empty strings at the boundaries when s starts or
// ends with a separator. Each segment is then classified as numeric or text.
func sortSegments(s string, hex bool) []numSeg {
	isDigit := func(c byte) bool {
		if c >= '0' && c <= '9' {
			return true
		}
		return hex && ((c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F'))
	}
	var segs []numSeg
	var cur strings.Builder
	for i := 0; i < len(s); {
		if isDigit(s[i]) {
			cur.WriteByte(s[i])
			i++
			continue
		}
		// Separator run: flush the accumulated digit-part, then the separator.
		segs = append(segs, classifySeg(cur.String(), hex))
		cur.Reset()
		j := i
		for j < len(s) && !isDigit(s[j]) {
			j++
		}
		segs = append(segs, classifySeg(s[i:j], hex))
		i = j
	}
	segs = append(segs, classifySeg(cur.String(), hex))
	return segs
}

// classifySeg mirrors JS isNaN(Number(seg)) / isNaN(parseInt(seg,16)): an empty
// or whitespace segment is the number 0; otherwise a segment is numeric only if
// it parses fully in the relevant base.
func classifySeg(part string, hex bool) numSeg {
	seg := numSeg{str: part}
	t := strings.TrimSpace(part)
	switch {
	case t == "":
		seg.isNum, seg.num = true, 0
	case hex:
		if n, err := strconv.ParseUint(t, 16, 64); err == nil {
			seg.isNum, seg.num = true, float64(n)
		}
	default:
		if n, err := strconv.ParseFloat(t, 64); err == nil {
			seg.isNum, seg.num = true, n
		}
	}
	return seg
}

// Unique removes duplicate sections, optionally displaying occurrence counts.
type Unique struct{}

// Meta returns the operation metadata.
func (Unique) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Unique",
		Module:      "Default",
		Description: "Removes duplicate sections of the input, split on the given delimiter. Optionally prefixes each with its occurrence count.",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (Unique) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Delimiter", Type: core.ArgOption, Value: inputDelims},
		{Name: "Display count", Type: core.ArgBoolean, Value: false},
	}
}

// Run deduplicates the sections.
func (Unique) Run(in *core.Dish, args []any) (*core.Dish, error) {
	delim := charRep(args[0].(string))
	count := args[1].(bool)
	sections := splitByDelim(in.String(), delim)

	var order []string
	counts := map[string]int{}
	for _, s := range sections {
		if _, seen := counts[s]; !seen {
			order = append(order, s)
		}
		counts[s]++
	}

	out := make([]string, len(order))
	for i, s := range order {
		if count {
			out[i] = fmt.Sprintf("%d %s", counts[s], s)
		} else {
			out[i] = s
		}
	}
	return core.NewDish([]byte(strings.Join(out, delim)), core.TypeString), nil
}
