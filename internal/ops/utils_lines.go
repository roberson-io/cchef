package ops

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
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

// Run filters the sections. Ported from CyberChef Filter.mjs.
func (Filter) Run(in *core.Dish, args []any) (*core.Dish, error) {
	delim := charRep(args[0].(string))
	re, err := regexp.Compile(args[1].(string))
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
			"IP address", "Numeric", "Numeric (hexadecimal)", "Length"}},
	}
}

// Run sorts the sections. Ported from CyberChef Sort.mjs.
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
		less = func(a, b string) bool { return ipLess(a, b) }
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

// naturalCompare compares two strings by alternating numeric and non-numeric
// segments (a "natural sort"). When hex is true, numeric segments are hex.
func naturalCompare(a, b string, hex bool) int {
	as, bs := numSegments(a, hex), numSegments(b, hex)
	for i := 0; i < len(as) && i < len(bs); i++ {
		x, y := as[i], bs[i]
		switch {
		case !x.isNum && y.isNum:
			return 1
		case x.isNum && !y.isNum:
			return -1
		case x.isNum && y.isNum:
			if x.num != y.num {
				if x.num < y.num {
					return -1
				}
				return 1
			}
		default:
			if c := strings.Compare(x.str, y.str); c != 0 {
				return c
			}
		}
	}
	return strings.Compare(a, b)
}

type numSeg struct {
	str   string
	num   float64
	isNum bool
}

// numSegments splits a string into maximal runs of digit (or hex-digit) and
// non-digit characters.
func numSegments(s string, hex bool) []numSeg {
	isDigit := func(r rune) bool {
		if hex {
			return (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
		}
		return r >= '0' && r <= '9'
	}
	var segs []numSeg
	runes := []rune(s)
	for i := 0; i < len(runes); {
		j := i
		digit := isDigit(runes[i])
		for j < len(runes) && isDigit(runes[j]) == digit {
			j++
		}
		part := string(runes[i:j])
		seg := numSeg{str: part}
		if digit {
			base := 10
			if hex {
				base = 16
			}
			if n, err := strconv.ParseInt(part, base, 64); err == nil {
				seg.isNum, seg.num = true, float64(n)
			}
		}
		segs = append(segs, seg)
		i = j
	}
	return segs
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

// Run deduplicates the sections. Ported from CyberChef Unique.mjs.
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
