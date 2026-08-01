package ops

import (
	"fmt"
	"strings"
	"unicode/utf16"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(ShowBase64Offsets{})
}

// ShowBase64Offsets shows the three possible Base64 encodings of a string
// depending on its byte offset within a larger block.
type ShowBase64Offsets struct{}

// Meta returns the operation metadata.
func (ShowBase64Offsets) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Show Base64 offsets",
		Module:      "Default",
		Description: "When a string is within a block of data and the whole block is Base64'd, the string itself could be represented in Base64 in three distinct ways depending on its offset within the block. This operation shows all possible offsets for a given string so that each possible encoding can be considered.",
		InfoURL:     "https://wikipedia.org/wiki/Base64#Output_padding",
		InputType:   core.TypeByteArray,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ShowBase64Offsets) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Alphabet", Type: core.ArgString, Value: "A-Za-z0-9+/="},
		{Name: "Show variable chars and padding", Type: core.ArgBoolean, Value: true},
		{Name: "Input format", Type: core.ArgOption, Value: []string{"Raw", "Base64"}},
	}
}

// Run builds the offsets view. Ported from CyberChef ShowBase64Offsets.mjs.
func (ShowBase64Offsets) Run(in *core.Dish, args []any) (*core.Dish, error) {
	alphabet := args[0].(string)
	showVariable := args[1].(bool)
	format := args[2].(string)

	input := in.Bytes()
	if format == "Base64" {
		input, _ = fromBase64(byteArrayToUtf8(input), "A-Za-z0-9+/=", true, false)
	}
	if len(input) < 1 {
		return nil, fmt.Errorf("please enter a string")
	}

	offset0 := b64OffsetSection(toBase64(input, alphabet), 0, alphabet, showVariable)
	offset1 := b64OffsetSection(toBase64(append([]byte{0}, input...), alphabet), 1, alphabet, showVariable)
	offset2 := b64OffsetSection(toBase64(append([]byte{0, 0}, input...), alphabet), 2, alphabet, showVariable)

	if !showVariable {
		return core.NewDish([]byte(offset0+"\n"+offset1+"\n"+offset2), core.TypeString), nil
	}

	const script = "<script type='application/javascript'>$('[data-toggle=\"tooltip\"]').tooltip()</script>"
	out := "Characters highlighted in <span class='hl5'>green</span> could change if the input is surrounded by more data." +
		"\nCharacters highlighted in <span class='hl3'>red</span> are for padding purposes only." +
		"\nUnhighlighted characters are <span data-toggle='tooltip' data-placement='top' title='Tooltip on left'>static</span>." +
		"\nHover over the static sections to see what they decode to on their own.\n" +
		"\nOffset 0: " + offset0 +
		"\nOffset 1: " + offset1 +
		"\nOffset 2: " + offset2 +
		script
	return core.NewDish([]byte(out), core.TypeString), nil
}

// b64OffsetSection renders one offset's Base64 string with highlight/tooltip
// markup (or, when showVariable is false, just the escaped static section). n is
// the number of zero bytes prepended for this offset (0, 1 or 2).
func b64OffsetSection(offsetStr string, n int, alphabet string, showVariable bool) string {
	lenPad := strings.IndexByte(offsetStr, '=') // first padding char, or -1

	leading, s := "", offsetStr
	prefixA := ""
	if n > 0 {
		// The leading n+1 chars encode the prepended zero byte(s): the first n are
		// padding-derived (red), the next one is variable (green).
		leading = "<span class='hl3'>" + escapeHTML(s[:n]) + "</span>" +
			"<span class='hl5'>" + escapeHTML(s[n:n+1]) + "</span>"
		s = s[n+1:]
		prefixA = strings.Repeat("A", n+1)
	}

	decodeTip := func(static string, dropEnd int) string {
		b, _ := fromBase64(prefixA+static, alphabet, true, false)
		return escapeHTML(sliceUTF16(byteArrayToUtf8(b), n, dropEnd))
	}
	tooltip := func(tip, static string) string {
		return "<span data-toggle='tooltip' data-placement='top' title='" + tip + "'>" +
			escapeHTML(static) + "</span>"
	}

	var static, body string
	switch lenPad % 4 {
	case 2: // two padding chars: 1 variable char then "=="
		static = s[:len(s)-3]
		body = tooltip(decodeTip(static, 2), static) +
			"<span class='hl5'>" + escapeHTML(s[len(s)-3:len(s)-2]) + "</span>" +
			"<span class='hl3'>" + escapeHTML(s[len(s)-2:]) + "</span>"
	case 3: // one padding char: 1 variable char then "="
		static = s[:len(s)-2]
		dropEnd := 1
		if n == 2 {
			dropEnd = 2 // reproduces CyberChef's offset-2 slice quirk
		}
		body = tooltip(decodeTip(static, dropEnd), static) +
			"<span class='hl5'>" + escapeHTML(s[len(s)-2:len(s)-1]) + "</span>" +
			"<span class='hl3'>" + escapeHTML(s[len(s)-1:]) + "</span>"
	default: // no padding: the whole section is static
		static = s
		body = tooltip(decodeTip(static, 0), static)
	}

	if !showVariable {
		return escapeHTML(static)
	}
	return leading + body
}

// sliceUTF16 mimics JavaScript String.prototype.slice(start, len-dropEnd),
// operating on UTF-16 code units so it matches CyberChef's byte-decoded strings.
func sliceUTF16(s string, start, dropEnd int) string {
	u := utf16.Encode([]rune(s))
	if start > len(u) {
		start = len(u)
	}
	end := max(len(u)-dropEnd, start)
	return string(utf16.Decode(u[start:end]))
}
