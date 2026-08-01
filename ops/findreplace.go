package ops

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(FindReplace{})
}

// FindReplace replaces matches of a pattern with a replacement string.
type FindReplace struct{}

// Meta returns the operation metadata.
func (FindReplace) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Find / Replace",
		Module:      "Default",
		Description: "Replaces all occurrences of the first string with the second. Supports simple strings, extended escapes, and regular expressions.",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (FindReplace) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Find", Type: core.ArgToggleString, Value: "", ToggleValues: []string{"Regex", "Extended (\\n, \\t, \\x...)", "Simple string"}},
		{Name: "Replace", Type: core.ArgString, Value: ""},
		{Name: "Global match", Type: core.ArgBoolean, Value: true},
		{Name: "Case insensitive", Type: core.ArgBoolean, Value: false},
		{Name: "Multiline matching", Type: core.ArgBoolean, Value: true},
		{Name: "Dot matches all", Type: core.ArgBoolean, Value: false},
	}
}

// Run performs the replacement. Ported from CyberChef FindReplace.mjs.
func (FindReplace) Run(in *core.Dish, args []any) (*core.Dish, error) {
	find := args[0].(core.ToggleString)
	replace := args[1].(string)
	global := args[2].(bool)

	// Build the Go inline flags.
	flags := ""
	if args[3].(bool) {
		flags += "i"
	}
	if args[4].(bool) {
		flags += "m"
	}
	if args[5].(bool) {
		flags += "s"
	}

	pattern := find.Value
	switch {
	case find.Option == "Regex":
		// use pattern as-is
	case strings.HasPrefix(find.Option, "Extended"):
		pattern = regexp.QuoteMeta(parseEscapedChars(find.Value))
	default: // Simple string
		pattern = regexp.QuoteMeta(find.Value)
	}
	if flags != "" {
		pattern = "(?" + flags + ")" + pattern
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex: %w", err)
	}

	input := in.String()
	var out string
	if global {
		out = re.ReplaceAllString(input, replace)
	} else {
		out = replaceFirst(re, input, replace)
	}
	return core.NewDish([]byte(out), core.TypeString), nil
}

// replaceFirst replaces only the first match of re in input, expanding $-refs.
func replaceFirst(re *regexp.Regexp, input, replace string) string {
	m := re.FindStringSubmatchIndex(input)
	if m == nil {
		return input
	}
	out := []byte(input[:m[0]])
	out = re.ExpandString(out, replace, input, m)
	out = append(out, input[m[1]:]...)
	return string(out)
}

// reEscapedChars matches the backslash escape sequences recognised by
// parseEscapedChars. Ported from CyberChef Utils.parseEscapedChars.
var reEscapedChars = regexp.MustCompile(`\\([abfnrtv'"]|[0-3][0-7]{2}|[0-7]{1,2}|x[0-9a-fA-F]{2}|u[0-9a-fA-F]{4}|u\{[0-9a-fA-F]{1,6}\}|\\)`)

// parseEscapedChars converts recognised backslash escape sequences into their
// literal characters. Unrecognised sequences (e.g. "\d") are left intact.
func parseEscapedChars(s string) string {
	return reEscapedChars.ReplaceAllStringFunc(s, func(m string) string {
		a := m[1:] // drop the leading backslash
		switch a[0] {
		case '\\':
			return "\\"
		case 'a':
			return "\x07"
		case 'b':
			return "\b"
		case 't':
			return "\t"
		case 'n':
			return "\n"
		case 'v':
			return "\v"
		case 'f':
			return "\f"
		case 'r':
			return "\r"
		case '"':
			return "\""
		case '\'':
			return "'"
		case 'x':
			v, _ := strconv.ParseInt(a[1:], 16, 32)
			return string(rune(v))
		case 'u':
			if a[1] == '{' {
				v, _ := strconv.ParseInt(a[2:len(a)-1], 16, 32)
				return string(rune(v))
			}
			v, _ := strconv.ParseInt(a[1:], 16, 32)
			return string(rune(v))
		default: // octal 0-7
			v, _ := strconv.ParseInt(a, 8, 32)
			return string(rune(v))
		}
	})
}
