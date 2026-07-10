package ops

import (
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(EscapeSmartCharacters{})
}

// EscapeSmartCharacters converts typographic Unicode characters to their plain
// ASCII equivalents.
type EscapeSmartCharacters struct{}

// Meta returns the operation metadata.
func (EscapeSmartCharacters) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Escape Smart Characters",
		Module:      "Default",
		Description: "Converts smart (typographic) Unicode characters — e.g. smart quotes, em/en dashes, ellipses, ©, ®, ™, arrows — into their plain ASCII equivalents. Characters with no ASCII mapping are handled according to the 'Unmappable characters' option.",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (EscapeSmartCharacters) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Unmappable characters", Type: core.ArgOption, Value: []string{
			"Include", "Remove", "Replace with '.'",
		}},
	}
}

// Run escapes the smart characters. Ported from CyberChef EscapeSmartCharacters.mjs.
func (EscapeSmartCharacters) Run(in *core.Dish, args []any) (*core.Dish, error) {
	unmappable := args[0].(string)
	var sb strings.Builder
	for _, ch := range in.String() {
		switch {
		case ch < 128:
			sb.WriteRune(ch)
		case smartCharMap[ch] != "":
			sb.WriteString(smartCharMap[ch])
		default:
			switch unmappable {
			case "Remove":
				// Drop the character.
			case "Replace with '.'":
				sb.WriteByte('.')
			default: // Include
				sb.WriteRune(ch)
			}
		}
	}
	return core.NewDish([]byte(sb.String()), core.TypeString), nil
}

// smartCharMap maps typographic Unicode characters to their ASCII equivalents.
// Ported from CyberChef's SMART_MAP. Every value is non-empty, so a zero-value
// lookup means "not a smart character".
var smartCharMap = map[rune]string{
	// Smart double quotes: “ ” „ ‟ ″
	'“': `"`, '”': `"`, '„': `"`, '‟': `"`, '″': `"`,
	// Smart single quotes / apostrophes: ‘ ’ ‚ ‛ ′
	'‘': "'", '’': "'", '‚': "'", '‛': "'", '′': "'",
	// Dashes & hyphens: ‐ ‑ ‒ – — ―
	'‐': "-", '‑': "-", '‒': "-", '–': "-",
	'—': "--", '―': "--",
	// Ellipsis: …
	'…': "...",
	// Trademark / copyright: © ® ™
	'©': "(c)", '®': "(r)", '™': "(tm)",
	// Arrows: ← → ↑ ↓ ↔ ⇐ ⇒ ⇔
	'←': "<--", '→': "-->", '↑': "^", '↓': "v",
	'↔': "<->", '⇐': "<==", '⇒': "==>", '⇔': "<=>",
	// Guillemets: « » ‹ ›
	'«': "<<", '»': ">>", '‹': "<", '›': ">",
	// Math & misc symbols: × ÷ ± • ·
	'×': "x", '÷': "/", '±': "+/-", '•': "*", '·': ".",
	// Non-ASCII spaces: NBSP, en, em, thin, hair
	' ': " ", ' ': " ", ' ': " ", ' ': " ", ' ': " ",
}
