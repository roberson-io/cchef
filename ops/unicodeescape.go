package ops

import (
	"regexp"
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(EscapeUnicodeCharacters{})
	core.Register(UnescapeUnicodeCharacters{})
}

// unicodeEscapePrefixes are the prefix options shared by the two operations.
var unicodeEscapePrefixes = []string{"\\u", "%u", "U+"}

// EscapeUnicodeCharacters converts characters to unicode-escaped notation.
type EscapeUnicodeCharacters struct{}

// Meta returns the operation metadata.
func (EscapeUnicodeCharacters) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Escape Unicode Characters",
		Module:      "Default",
		Description: "Converts characters to their unicode-escaped notations. Supports the prefixes \\u, %u, and U+, e.g. σου becomes \\u03C3\\u03BF\\u03C5.",
		InfoURL:     "https://wikipedia.org/wiki/Unicode",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (EscapeUnicodeCharacters) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Prefix", Type: core.ArgOption, Value: unicodeEscapePrefixes},
		{Name: "Encode all chars", Type: core.ArgBoolean, Value: false},
		{Name: "Padding", Type: core.ArgNumber, Integer: true, Value: float64(4)},
		{Name: "Uppercase hex", Type: core.ArgBoolean, Value: true},
	}
}

// Run escapes the input.
func (EscapeUnicodeCharacters) Run(in *core.Dish, args []any) (*core.Dish, error) {
	prefix := args[0].(string)
	encodeAll := args[1].(bool)
	padding := int(args[2].(float64))
	upperHex := args[3].(bool)

	var sb strings.Builder
	for _, u := range utf16.Encode([]rune(in.String())) {
		if !encodeAll && u >= 0x20 && u <= 0x7e {
			sb.WriteRune(rune(u))
			continue
		}
		cp := strconv.FormatUint(uint64(u), 16)
		if upperHex {
			cp = strings.ToUpper(cp)
		}
		if len(cp) < padding {
			cp = strings.Repeat("0", padding-len(cp)) + cp
		}
		sb.WriteString(prefix + cp)
	}
	return core.NewDish([]byte(sb.String()), core.TypeString), nil
}

// UnescapeUnicodeCharacters converts unicode-escaped notation back to characters.
type UnescapeUnicodeCharacters struct{}

// Meta returns the operation metadata.
func (UnescapeUnicodeCharacters) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Unescape Unicode Characters",
		Module:      "Default",
		Description: "Converts unicode-escaped character notation back into raw characters. Supports the prefixes \\u, %u, and U+, e.g. \\u03c3\\u03bf\\u03c5 becomes σου.",
		InfoURL:     "https://wikipedia.org/wiki/Unicode",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (UnescapeUnicodeCharacters) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Prefix", Type: core.ArgOption, Value: unicodeEscapePrefixes},
	}
}

// unescapePrefixRegex maps each prefix option to its regex-escaped form.
var unescapePrefixRegex = map[string]string{"\\u": `\\u`, "%u": `%u`, "U+": `U\+`}

// Run unescapes the input. Only the U+ prefix admits 4-6 hex digits (for
// astral code points); the others are fixed at 4.
func (UnescapeUnicodeCharacters) Run(in *core.Dish, args []any) (*core.Dish, error) {
	prefix := args[0].(string)
	quant := "{4}"
	if prefix == "U+" {
		quant = "{4,6}"
	}
	re := regexp.MustCompile(`(?i)` + unescapePrefixRegex[prefix] + `([a-f\d]` + quant + `)`)

	input := in.String()
	var sb strings.Builder
	last := 0
	for _, m := range re.FindAllStringSubmatchIndex(input, -1) {
		sb.WriteString(input[last:m[0]])
		cp, _ := strconv.ParseInt(input[m[2]:m[3]], 16, 32)
		sb.WriteRune(rune(cp))
		last = m[1]
	}
	sb.WriteString(input[last:])
	return core.NewDish([]byte(sb.String()), core.TypeString), nil
}
