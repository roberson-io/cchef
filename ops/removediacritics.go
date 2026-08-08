package ops

import (
	"strings"

	"golang.org/x/text/unicode/norm"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/opsutil"
)

func init() {
	core.Register(RemoveDiacritics{})
}

// RemoveDiacritics strips accents from characters.
type RemoveDiacritics struct{}

// Meta returns the operation metadata.
func (RemoveDiacritics) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Remove Diacritics",
		Module:      "Default",
		Description: "Replaces accented characters with their latin character equivalent. Accented characters are made up of Unicode combining characters, so unicode text formatting such as strikethroughs and underlines will also be removed.",
		InfoURL:     "https://wikipedia.org/wiki/Diacritic",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns no arguments; the operation takes none.
func (RemoveDiacritics) Args() []core.ArgDef { return nil }

// Run strips the accents: decompose to NFD, then drop every combining
// diacritical mark (U+0300 to U+036F), as upstream does. A character whose
// accent is part of the letter itself rather than a combining mark (ø, đ, ß)
// has nothing to strip and survives.
func (RemoveDiacritics) Run(in *core.Dish, args []any) (*core.Dish, error) {
	decomposed := norm.NFD.String(dishText(in))
	stripped := strings.Map(func(r rune) rune {
		if r >= 0x0300 && r <= 0x036F {
			return -1
		}
		return r
	}, decomposed)
	return core.NewDish(opsutil.TextAsBytes(stripped), core.TypeString), nil
}
