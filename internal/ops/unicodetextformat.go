package ops

import (
	"unicode/utf8"

	"github.com/roberson-io/cchef/internal/core"
)

// The combining characters, as UTF-8 bytes: U+0336 long stroke overlay and
// U+0332 low line.
var (
	utfStrikethrough = []byte{0xcc, 0xb6}
	utfUnderline     = []byte{0xcc, 0xb2}
)

func init() {
	core.Register(UnicodeTextFormat{})
}

// UnicodeTextFormat decorates text with Unicode combining characters. Ported
// from CyberChef UnicodeTextFormat.mjs.
type UnicodeTextFormat struct{}

// Meta returns the operation metadata.
func (UnicodeTextFormat) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Unicode Text Format",
		Module:      "Default",
		Description: "Adds Unicode combining characters to change formatting of plaintext.",
		InfoURL:     "https://wikipedia.org/wiki/Combining_character",
		InputType:   core.TypeByteArray,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the two formats.
func (UnicodeTextFormat) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Underline", Type: core.ArgBoolean, Value: false},
		{Name: "Strikethrough", Type: core.ArgBoolean, Value: false},
	}
}

// Run decorates the input: after every character — a whole UTF-8 sequence, or
// a single byte where the bytes are not UTF-8 — the strikethrough mark and
// then the underline mark are appended, for whichever formats are asked for.
// Upstream appends them after every byte instead, splitting multi-byte
// characters apart, which is logged as a CyberChef bug.
func (UnicodeTextFormat) Run(in *core.Dish, args []any) (*core.Dish, error) {
	underline := args[0].(bool)
	strikethrough := args[1].(bool)

	data := in.Bytes()
	out := make([]byte, 0, len(data))
	for i := 0; i < len(data); {
		_, size := utf8.DecodeRune(data[i:])
		out = append(out, data[i:i+size]...)
		if strikethrough {
			out = append(out, utfStrikethrough...)
		}
		if underline {
			out = append(out, utfUnderline...)
		}
		i += size
	}
	return core.NewDish(out, core.TypeByteArray), nil
}
