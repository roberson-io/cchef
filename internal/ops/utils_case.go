package ops

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(GetAllCasings{})
	core.Register(UnescapeString{})
	core.Register(ToCaseInsensitiveRegex{})
	core.Register(FromCaseInsensitiveRegex{})
}

// GetAllCasings produces every combination of upper/lower casing of the input.
type GetAllCasings struct{}

// Meta returns the operation metadata.
func (GetAllCasings) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Get All Casings",
		Module:      "Default",
		Description: "Outputs all possible combinations of upper- and lower-case for the input, one per line. The number of results doubles with each character.",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (GetAllCasings) Args() []core.ArgDef { return nil }

// Run produces all casings. Ported from CyberChef GetAllCasings.mjs.
func (GetAllCasings) Run(in *core.Dish, args []any) (*core.Dish, error) {
	lower := []rune(strings.ToLower(in.String()))
	n := len(lower)
	if n == 0 {
		return core.NewDish(nil, core.TypeString), nil
	}
	var lines []string
	for i := 0; i < (1 << n); i++ {
		temp := make([]rune, n)
		for j := 0; j < n; j++ {
			if (i>>j)&1 == 1 {
				temp[j] = unicode.ToUpper(lower[j])
			} else {
				temp[j] = lower[j]
			}
		}
		lines = append(lines, string(temp))
	}
	return core.NewDish([]byte(strings.Join(lines, "\n")), core.TypeString), nil
}

// UnescapeString converts backslash escape sequences into their raw characters.
type UnescapeString struct{}

// Meta returns the operation metadata.
func (UnescapeString) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Unescape string",
		Module:      "Default",
		Description: "Unescapes characters in a string that have been escaped (e.g. \\n, \\t, \\xNN, \\u{...}).",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (UnescapeString) Args() []core.ArgDef { return nil }

// Run unescapes the string. Ported from CyberChef UnescapeString.mjs.
func (UnescapeString) Run(in *core.Dish, args []any) (*core.Dish, error) {
	return core.NewDish([]byte(parseEscapedChars(in.String())), core.TypeString), nil
}

// ciRange is one of the sequential character-range expansions applied by
// To Case Insensitive Regex.
type ciRange struct {
	re *regexp.Regexp
	fn func(string) string
}

func up(r rune) string { return string(unicode.ToUpper(r)) }
func lo(r rune) string { return string(unicode.ToLower(r)) }

// ciRanges are ported, in order, from CyberChef ToCaseInsensitiveRegex.mjs.
var ciRanges = []ciRange{
	{regexp.MustCompile(`[A-Z]-[A-Z]|[a-z]-[a-z]`), func(m string) string {
		r := []rune(m)
		return up(r[0]) + "-" + up(r[2]) + lo(r[0]) + "-" + lo(r[2])
	}},
	{regexp.MustCompile(`[A-Z]-[a-z]`), func(m string) string {
		r := []rune(m)
		return "A-" + up(r[2]) + m + lo(r[0]) + "-z"
	}},
	{regexp.MustCompile(`\\?[\x20-\x40]-[A-Z]`), func(m string) string {
		r := []rune(m)
		return m + "a-" + lo(r[2])
	}},
	{regexp.MustCompile(`\\?[\x20-\x40]-\\?[\x5b-\x60]`), func(m string) string {
		return m + "a-z"
	}},
	{regexp.MustCompile(`[A-Z]-\\?[\x5b-\x60]`), func(m string) string {
		r := []rune(m)
		return m + lo(r[0]) + "-z"
	}},
	{regexp.MustCompile(`\\?[\x5b-\x60]-\\?[\x7b-\x7e]`), func(m string) string {
		return m + "A-Z"
	}},
	{regexp.MustCompile(`[a-z]-\\?[\x7b-\x7e]`), func(m string) string {
		r := []rune(m)
		return m + up(r[0]) + "-Z"
	}},
	{regexp.MustCompile(`\\?[\x20-\x40]-[a-z]`), func(m string) string {
		r := []rune(m)
		return string(r[0]) + "-z"
	}},
	{regexp.MustCompile(`\\?[\x5b-\x60]-[a-z]`), func(m string) string {
		r := []rune(m)
		return "A-" + up(r[2]) + m
	}},
}

// ToCaseInsensitiveRegex rewrites a regex to match either case.
type ToCaseInsensitiveRegex struct{}

// Meta returns the operation metadata.
func (ToCaseInsensitiveRegex) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "To Case Insensitive Regex",
		Module:      "Default",
		Description: "Converts a case-sensitive regular expression into a case-insensitive one, e.g. Mozilla becomes [mM][oO][zZ][iI][lL][lL][aA].",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ToCaseInsensitiveRegex) Args() []core.ArgDef { return nil }

// Run rewrites the regex. Ported from CyberChef ToCaseInsensitiveRegex.mjs.
func (ToCaseInsensitiveRegex) Run(in *core.Dish, args []any) (*core.Dish, error) {
	// Wrap each standalone letter as [lL], simulating the upstream pre-process.
	r := []rune(in.String())
	var sb strings.Builder
	for i, c := range r {
		isLetter := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
		prevDash := i > 0 && r[i-1] == '-'
		nextDash := i+1 < len(r) && r[i+1] == '-'
		if isLetter && !prevDash && !nextDash {
			sb.WriteString("[" + lo(c) + up(c) + "]")
		} else {
			sb.WriteRune(c)
		}
	}

	out := sb.String()
	for _, cr := range ciRanges {
		out = cr.re.ReplaceAllStringFunc(out, cr.fn)
	}
	return core.NewDish([]byte(out), core.TypeString), nil
}

// reFromCI matches a two-letter character class like [tT].
var reFromCI = regexp.MustCompile(`\[[a-zA-Z]{2}\]`)

// FromCaseInsensitiveRegex collapses [lL] pairs back to a single letter.
type FromCaseInsensitiveRegex struct{}

// Meta returns the operation metadata.
func (FromCaseInsensitiveRegex) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "From Case Insensitive Regex",
		Module:      "Default",
		Description: "Converts a case-insensitive regex of the form [aA][bB] back to a case-sensitive one (abc). Character classes with distinct letters are left unchanged.",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (FromCaseInsensitiveRegex) Args() []core.ArgDef { return nil }

// Run collapses the pairs. Ported from CyberChef FromCaseInsensitiveRegex.mjs.
func (FromCaseInsensitiveRegex) Run(in *core.Dish, args []any) (*core.Dish, error) {
	out := reFromCI.ReplaceAllStringFunc(in.String(), func(m string) string {
		a, b := rune(m[1]), rune(m[2])
		if unicode.ToUpper(a) == unicode.ToUpper(b) {
			return string(a)
		}
		return m
	})
	return core.NewDish([]byte(out), core.TypeString), nil
}
