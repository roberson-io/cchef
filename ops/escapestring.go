package ops

import (
	"fmt"
	"strings"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(EscapeString{})
}

// escapeNamed are the single-character escape sequences used for common control
// characters.
var escapeNamed = map[rune]string{
	'\b': `\b`, '\t': `\t`, '\n': `\n`, '\v': `\v`, '\f': `\f`, '\r': `\r`,
}

// EscapeString escapes special characters in a string. This is a from-scratch
// implementation of the behaviour CyberChef gets from the jsesc library; it
// covers the documented options but is not guaranteed byte-identical to jsesc in
// every edge case.
type EscapeString struct{}

// Meta returns the operation metadata.
func (EscapeString) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Escape string",
		Module:      "Default",
		Description: "Escapes special characters in a string so they do not cause conflicts. Supports several escape levels, quote styles, JSON/ES6 compatibility and hex casing.",
		InfoURL:     "https://wikipedia.org/wiki/Escape_sequence",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (EscapeString) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Escape level", Type: core.ArgOption, Value: []string{"Special chars", "Everything", "Minimal"}},
		{Name: "Escape quote", Type: core.ArgOption, Value: []string{"Single", "Double", "Backtick"}},
		{Name: "JSON compatible", Type: core.ArgBoolean, Value: false},
		{Name: "ES6 compatible", Type: core.ArgBoolean, Value: true},
		{Name: "Uppercase hex", Type: core.ArgBoolean, Value: false},
	}
}

// Run escapes the input.
func (EscapeString) Run(in *core.Dish, args []any) (*core.Dish, error) {
	level := args[0].(string)
	q := quoteRune(args[1].(string))
	jsonCompat := args[2].(bool)
	es6 := args[3].(bool)
	upper := args[4].(bool)

	runes := []rune(in.String())
	var sb strings.Builder
	for i, r := range runes {
		sb.WriteString(escapeRune(r, i, runes, level, q, jsonCompat, es6, upper))
	}
	out := sb.String()
	// JSON-compatible mode wraps the result in the selected quote character and
	// escapes that quote inside (jsesc's json+quotes behaviour); the quote char
	// itself is not forced to double.
	if jsonCompat {
		out = string(q) + out + string(q)
	}
	return core.NewDish([]byte(out), core.TypeString), nil
}

// escapeRune returns the escaped representation of a single rune.
func escapeRune(r rune, i int, runes []rune, level string, q rune, jsonCompat, es6, upper bool) string {
	switch r {
	case '\\':
		return `\\`
	case q:
		return `\` + string(q)
	}
	if e, ok := escapeNamed[r]; ok {
		return e
	}
	if r == 0 {
		return escapeNull(i, runes, jsonCompat, es6, upper)
	}
	// U+2028/U+2029 are invalid in JS string literals, so jsesc escapes them at
	// every level, including Minimal.
	if r == '\u2028' || r == '\u2029' {
		return escHex(r, jsonCompat, es6, upper)
	}
	// Minimal mode escapes only backslash, the quote, named escapes and the line
	// separators above; everything else (control chars and non-ASCII) is literal.
	if level == "Minimal" {
		return string(r)
	}
	if r >= asciiPrintableMin && r <= asciiPrintableMax {
		return escapePrintable(r, level, jsonCompat, es6, upper)
	}
	return escHex(r, jsonCompat, es6, upper)
}

// asciiPrintableMin/Max bound the printable ASCII range (space through '~').
const (
	asciiPrintableMin = 0x20
	asciiPrintableMax = 0x7e
)

// escapeNull escapes a NUL byte. It is ambiguous when followed by a digit, and
// invalid in JSON, so those cases hex-escape it; otherwise "\0". (Reduced
// fidelity: jsesc leaves a raw null byte in Minimal mode; we escape it, which
// differs only for that rarely-used combination.)
func escapeNull(i int, runes []rune, jsonCompat, es6, upper bool) string {
	if jsonCompat || (i+1 < len(runes) && runes[i+1] >= '0' && runes[i+1] <= '9') {
		return escHex(0, jsonCompat, es6, upper)
	}
	return `\0`
}

// escapePrintable handles a printable-ASCII rune (not Minimal mode): a literal,
// except that Everything mode hex-escapes it, with quotes backslash-escaped.
func escapePrintable(r rune, level string, jsonCompat, es6, upper bool) string {
	if level == "Everything" {
		// jsesc escapes every quote character as a backslash escape even in
		// escapeEverything mode (the selected quote is handled above).
		if r == '\'' || r == '"' || r == '`' {
			return `\` + string(r)
		}
		return escHex(r, jsonCompat, es6, upper)
	}
	return string(r)
}

// quoteRune maps a quote-style name to its character.
func quoteRune(name string) rune {
	switch name {
	case "Double":
		return '"'
	case "Backtick":
		return '`'
	default:
		return '\''
	}
}

// escHex escapes a rune as \xNN, \uNNNN, \u{...} (ES6) or a UTF-16 surrogate pair.
func escHex(r rune, jsonCompat, es6, upper bool) string {
	hex := func(v uint32, width int) string {
		s := fmt.Sprintf("%0*x", width, v)
		if upper {
			s = strings.ToUpper(s)
		}
		return s
	}
	switch {
	case r < 0x100:
		if jsonCompat {
			return `\u` + hex(uint32(r), 4) // #nosec G115 -- rune widened to uint32 for hex formatting
		}
		return `\x` + hex(uint32(r), 2) // #nosec G115 -- rune widened to uint32 for hex formatting
	case r <= 0xFFFF:
		return `\u` + hex(uint32(r), 4)
	case es6:
		return `\u{` + hex(uint32(r), 1) + `}`
	default:
		cp := uint32(r) - 0x10000
		hi := 0xD800 + (cp >> 10)
		lo := 0xDC00 + (cp & 0x3FF)
		return `\u` + hex(hi, 4) + `\u` + hex(lo, 4)
	}
}
