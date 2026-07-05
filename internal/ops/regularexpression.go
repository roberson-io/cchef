package ops

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(RegularExpression{})
}

// reCapturePlaceholder matches the placeholders used while highlighting.
var reCapturePlaceholder = regexp.MustCompile(`\[cc_capture_group_(\d+)\]`)

// RegularExpression highlights or lists the matches of a regular expression.
type RegularExpression struct{}

// Meta returns the operation metadata.
func (RegularExpression) Meta() core.OpMeta {
	return core.OpMeta{
		Name:   "Regular expression",
		Module: "Default",
		Description: "Define your own regular expression (using Go's RE2 syntax) to search the input with, optionally highlighting or listing the matches. " +
			"Note: RE2 does not support lookaround or backreferences, so some XRegExp-only patterns will not compile.",
		InfoURL:    "https://wikipedia.org/wiki/Regular_expression",
		InputType:  core.TypeString,
		OutputType: core.TypeString,
	}
}

// Args returns the argument definitions. The first argument mirrors CyberChef's
// "Built in regexes" populate-option, which is ignored at run time; supply the
// pattern via the Regex argument.
func (RegularExpression) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Built in regexes", Type: core.ArgString, Value: ""},
		{Name: "Regex", Type: core.ArgString, Value: ""},
		{Name: "Case insensitive", Type: core.ArgBoolean, Value: true},
		{Name: "^ and $ match at newlines", Type: core.ArgBoolean, Value: true},
		{Name: "Dot matches all", Type: core.ArgBoolean, Value: false},
		{Name: "Unicode support", Type: core.ArgBoolean, Value: false},
		{Name: "Astral support", Type: core.ArgBoolean, Value: false},
		{Name: "Display total", Type: core.ArgBoolean, Value: false},
		{Name: "Output format", Type: core.ArgOption, Value: []string{
			"Highlight matches", "List matches", "List capture groups", "List matches with capture groups",
		}},
	}
}

// Run applies the regex. Ported from CyberChef RegularExpression.mjs.
func (RegularExpression) Run(in *core.Dish, args []any) (*core.Dish, error) {
	userRegex := args[1].(string)
	input := in.String()
	if userRegex == "" || userRegex == "^" || userRegex == "$" {
		return core.NewDish([]byte(escapeHTML(input)), core.TypeString), nil
	}

	flags := ""
	if args[2].(bool) {
		flags += "i"
	}
	if args[3].(bool) {
		flags += "m"
	}
	if args[4].(bool) {
		flags += "s"
	}
	pattern := userRegex
	if flags != "" {
		pattern = "(?" + flags + ")" + pattern
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex: %w", err)
	}

	displayTotal := args[7].(bool)
	var out string
	switch args[8].(string) {
	case "List matches":
		out = escapeHTML(regexList(input, re, displayTotal, true, false))
	case "List capture groups":
		out = escapeHTML(regexList(input, re, displayTotal, false, true))
	case "List matches with capture groups":
		out = escapeHTML(regexList(input, re, displayTotal, true, true))
	default: // Highlight matches
		out = regexHighlight(input, re, displayTotal)
	}
	return core.NewDish([]byte(out), core.TypeString), nil
}

// regexList lists matches and/or capture groups. Ported from RegularExpression.mjs.
func regexList(input string, re *regexp.Regexp, displayTotal, matches, captureGroups bool) string {
	var b strings.Builder
	total := 0
	for _, m := range re.FindAllStringSubmatch(input, -1) {
		total++
		if matches {
			b.WriteString(m[0] + "\n")
		}
		if captureGroups {
			for i := 1; i < len(m); i++ {
				if matches {
					b.WriteString("  Group " + strconv.Itoa(i) + ": ")
				}
				b.WriteString(m[i] + "\n")
			}
		}
	}
	out := b.String()
	if displayTotal {
		out = "Total found: " + strconv.Itoa(total) + "\n\n" + out
	}
	return strings.TrimSuffix(out, "\n")
}

// regexHighlight wraps matches in <span> tags. Ported from RegularExpression.mjs.
func regexHighlight(input string, re *regexp.Regexp, displayTotal bool) string {
	var spans []string
	hl, total := 1, 0
	var sb strings.Builder
	last := 0
	for _, m := range re.FindAllStringSubmatchIndex(input, -1) {
		sb.WriteString(input[last:m[0]])
		match := input[m[0]:m[1]]
		var title strings.Builder
		fmt.Fprintf(&title, "Offset: %d\n", m[0])
		groups := len(m)/2 - 1
		if groups > 0 {
			title.WriteString("Groups:\n")
			for i := 1; i <= groups; i++ {
				g := ""
				if m[2*i] >= 0 {
					g = input[m[2*i]:m[2*i+1]]
				}
				fmt.Fprintf(&title, "\t%d: %s\n", i, escapeHTML(g))
			}
		}
		hl = 3 - hl // toggle 1/2
		spans = append(spans, fmt.Sprintf("<span class='hl%d' title='%s'>%s</span>", hl, title.String(), escapeHTML(match)))
		fmt.Fprintf(&sb, "[cc_capture_group_%d]", total)
		total++
		last = m[1]
	}
	sb.WriteString(input[last:])

	out := escapeHTML(sb.String())
	out = reCapturePlaceholder.ReplaceAllStringFunc(out, func(ph string) string {
		idx := reCapturePlaceholder.FindStringSubmatch(ph)
		i, _ := strconv.Atoi(idx[1])
		return spans[i]
	})
	if displayTotal {
		out = "Total found: " + strconv.Itoa(total) + "\n\n" + out
	}
	return out
}
