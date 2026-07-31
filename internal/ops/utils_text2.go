package ops

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(Split{})
	core.Register(CountOccurrences{})
	core.Register(AddLineNumbers{})
	core.Register(RemoveLineNumbers{})
	core.Register(AlternatingCaps{})
	core.Register(RemoveANSIEscapeCodes{})
	core.Register(ExpandAlphabetRange{})
}

// Split splits the input on one delimiter and rejoins with another.
type Split struct{}

// Meta returns the operation metadata.
func (Split) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Split",
		Module:      "Default",
		Description: "Splits the input on the given delimiter and rejoins the parts with another. Delimiters are used literally (matching CyberChef).",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions. The default join delimiter is the
// literal two-character "\\n", matching CyberChef's "Line feed" preset.
func (Split) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Split delimiter", Type: core.ArgEditableOption, Value: ","},
		{Name: "Join delimiter", Type: core.ArgEditableOption, Value: `\n`},
	}
}

// Run splits then joins, using the delimiters literally. Ported from CyberChef
// Split.mjs.
func (Split) Run(in *core.Dish, args []any) (*core.Dish, error) {
	parts := strings.Split(in.String(), args[0].(string))
	return core.NewDish([]byte(strings.Join(parts, args[1].(string))), core.TypeString), nil
}

// CountOccurrences counts how many times a search term appears.
type CountOccurrences struct{}

// Meta returns the operation metadata.
func (CountOccurrences) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Count occurrences",
		Module:      "Default",
		Description: "Counts the number of times the provided string occurs in the input. Regex searches are case-insensitive.",
		InputType:   core.TypeString,
		OutputType:  core.TypeNumber,
	}
}

// Args returns the argument definitions.
func (CountOccurrences) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Search string", Type: core.ArgToggleString, Value: "", ToggleValues: []string{"Regex", "Extended (\\n, \\t, \\x...)", "Simple string"}},
	}
}

// Run counts occurrences. Ported from CyberChef CountOccurrences.mjs.
func (CountOccurrences) Run(in *core.Dish, args []any) (*core.Dish, error) {
	search := args[0].(core.ToggleString)
	input := in.String()
	count := 0

	switch {
	case search.Value == "":
		count = 0
	case search.Option == "Regex":
		if re, err := regexp.Compile("(?i)" + search.Value); err == nil {
			count = len(re.FindAllString(input, -1))
		}
	default:
		needle := search.Value
		if strings.HasPrefix(search.Option, "Extended") {
			needle = parseEscapedChars(needle)
		}
		count = strings.Count(input, needle)
	}
	return core.NewDish([]byte(strconv.Itoa(count)), core.TypeNumber), nil
}

// AddLineNumbers prefixes each line with its (offset) number.
type AddLineNumbers struct{}

// Meta returns the operation metadata.
func (AddLineNumbers) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Add line numbers",
		Module:      "Default",
		Description: "Adds a line number to the beginning of each line, right-aligned and zero-offset adjustable.",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (AddLineNumbers) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Offset", Type: core.ArgNumber, Integer: true, Value: 0},
	}
}

// Run adds line numbers. Ported from CyberChef AddLineNumbers.mjs.
func (AddLineNumbers) Run(in *core.Dish, args []any) (*core.Dish, error) {
	offset := int(args[0].(float64))
	lines := strings.Split(in.String(), "\n")
	width := len(strconv.Itoa(len(lines)))

	out := make([]string, len(lines))
	for n, line := range lines {
		num := strconv.Itoa(n + 1 + offset)
		out[n] = leftPadSpace(num, width) + " " + line
	}
	return core.NewDish([]byte(strings.Join(out, "\n")), core.TypeString), nil
}

// leftPadSpace right-aligns s to at least width using spaces.
func leftPadSpace(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return strings.Repeat(" ", width-len(s)) + s
}

// reLineNumbers matches a leading line number and its trailing separator.
var reLineNumbers = regexp.MustCompile(`(?m)^[ \t]{0,5}\d+[\s:|,.)\]-]`)

// RemoveLineNumbers removes leading line numbers from each line.
type RemoveLineNumbers struct{}

// Meta returns the operation metadata.
func (RemoveLineNumbers) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Remove line numbers",
		Module:      "Default",
		Description: "Removes line numbers from the beginning of each line, if they can be found.",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (RemoveLineNumbers) Args() []core.ArgDef { return nil }

// Run removes line numbers. Ported from CyberChef RemoveLineNumbers.mjs.
func (RemoveLineNumbers) Run(in *core.Dish, args []any) (*core.Dish, error) {
	out := reLineNumbers.ReplaceAllString(in.String(), "")
	return core.NewDish([]byte(out), core.TypeString), nil
}

// AlternatingCaps applies aLtErNaTiNg capitalisation.
type AlternatingCaps struct{}

// Meta returns the operation metadata.
func (AlternatingCaps) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Alternating Caps",
		Module:      "Default",
		Description: "Applies alternating capitalisation to the input, starting with lower case.",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (AlternatingCaps) Args() []core.ArgDef { return nil }

// Run alternates caps. Ported from CyberChef AlternatingCaps.mjs.
func (AlternatingCaps) Run(in *core.Dish, args []any) (*core.Dish, error) {
	var sb strings.Builder
	previousCaps := true
	for _, r := range in.String() {
		switch {
		case !unicode.IsLetter(r):
			sb.WriteRune(r)
		case previousCaps:
			sb.WriteRune(unicode.ToLower(r))
			previousCaps = false
		default:
			sb.WriteRune(unicode.ToUpper(r))
			previousCaps = true
		}
	}
	return core.NewDish([]byte(sb.String()), core.TypeString), nil
}

// reANSI matches ANSI/VT100 escape sequences.
var reANSI = regexp.MustCompile("\x1b\\[[0-?]*[ -/]*[@-~]")

// RemoveANSIEscapeCodes strips ANSI terminal escape sequences.
type RemoveANSIEscapeCodes struct{}

// Meta returns the operation metadata.
func (RemoveANSIEscapeCodes) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Remove ANSI Escape Codes",
		Module:      "Default",
		Description: "Removes ANSI escape codes (e.g. terminal colour codes) from the input.",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (RemoveANSIEscapeCodes) Args() []core.ArgDef { return nil }

// Run strips ANSI codes. Ported from CyberChef RemoveANSIEscapeCodes.mjs.
func (RemoveANSIEscapeCodes) Run(in *core.Dish, args []any) (*core.Dish, error) {
	out := reANSI.ReplaceAllString(in.String(), "")
	return core.NewDish([]byte(out), core.TypeString), nil
}

// ExpandAlphabetRange expands an alphabet range specification.
type ExpandAlphabetRange struct{}

// Meta returns the operation metadata.
func (ExpandAlphabetRange) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Expand alphabet range",
		Module:      "Default",
		Description: "Expand an alphabet range string into a list of the characters in that range. e.g. a-z becomes abcdefghijklmnopqrstuvwxyz.",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ExpandAlphabetRange) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Delimiter", Type: core.ArgString, Value: ""},
	}
}

// Run expands the range. Ported from CyberChef ExpandAlphabetRange.mjs.
func (ExpandAlphabetRange) Run(in *core.Dish, args []any) (*core.Dish, error) {
	delim := parseEscapedChars(args[0].(string))
	expanded := expandAlphRange(in.String())
	// Join the expanded characters with the delimiter.
	chars := make([]string, 0, utf8.RuneCountInString(expanded))
	for _, r := range expanded {
		chars = append(chars, string(r))
	}
	return core.NewDish([]byte(strings.Join(chars, delim)), core.TypeString), nil
}
