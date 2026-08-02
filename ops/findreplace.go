package ops

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/opsutil"
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

// Run performs the replacement.
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
		pattern = regexp.QuoteMeta(opsutil.ParseEscapedChars(find.Value))
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
