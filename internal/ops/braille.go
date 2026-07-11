package ops

import (
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

// brailleASCII and brailleDot6 are CyberChef's BRAILLE_LOOKUP tables
// (lib/Braille.mjs): the character at index i in one maps to index i in the
// other. brailleASCII is all upper-case, so To Braille is case-insensitive.
const (
	brailleASCII = ` A1B'K2L@CIF/MSP"E3H9O6R^DJG>NTQ,*5<-U8V.%[$+X!&;:4\0Z7(_?W]#Y)=`
	brailleDot6  = `⠀⠁⠂⠃⠄⠅⠆⠇⠈⠉⠊⠋⠌⠍⠎⠏⠐⠑⠒⠓⠔⠕⠖⠗⠘⠙⠚⠛⠜⠝⠞⠟⠠⠡⠢⠣⠤⠥⠦⠧⠨⠩⠪⠫⠬⠭⠮⠯⠰⠱⠲⠳⠴⠵⠶⠷⠸⠹⠺⠻⠼⠽⠾⠿`
)

// brailleDot6Runes indexes the dot6 table by position; brailleFromDot6 is the
// reverse lookup from a braille rune to its position in the tables.
var (
	brailleDot6Runes = []rune(brailleDot6)
	brailleFromDot6  = func() map[rune]int {
		m := make(map[rune]int, len(brailleDot6Runes))
		for i, r := range brailleDot6Runes {
			m[r] = i
		}
		return m
	}()
)

// ToBraille converts text to six-dot braille symbols.
type ToBraille struct{}

// Meta returns the operation metadata.
func (ToBraille) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "To Braille",
		Module:      "Default",
		Description: "Converts text to six-dot braille symbols.",
		InfoURL:     "https://wikipedia.org/wiki/Braille",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ToBraille) Args() []core.ArgDef { return nil }

// Run converts text to braille.
func (ToBraille) Run(in *core.Dish, args []any) (*core.Dish, error) {
	var sb strings.Builder
	for _, r := range string(in.Bytes()) {
		// brailleASCII is pure ASCII, so the byte index equals the position
		// into the dot6 table. Characters not in the table are passed through.
		if idx := strings.Index(brailleASCII, strings.ToUpper(string(r))); idx >= 0 {
			sb.WriteRune(brailleDot6Runes[idx])
		} else {
			sb.WriteRune(r)
		}
	}
	return core.NewDish([]byte(sb.String()), core.TypeString), nil
}

// FromBraille converts six-dot braille symbols to text.
type FromBraille struct{}

// Meta returns the operation metadata.
func (FromBraille) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "From Braille",
		Module:      "Default",
		Description: "Converts six-dot braille symbols to text.",
		InfoURL:     "https://wikipedia.org/wiki/Braille",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (FromBraille) Args() []core.ArgDef { return nil }

// Run converts braille to text.
func (FromBraille) Run(in *core.Dish, args []any) (*core.Dish, error) {
	var sb strings.Builder
	for _, r := range string(in.Bytes()) {
		if idx, ok := brailleFromDot6[r]; ok {
			sb.WriteByte(brailleASCII[idx])
		} else {
			sb.WriteRune(r)
		}
	}
	return core.NewDish([]byte(sb.String()), core.TypeString), nil
}

func init() {
	core.Register(ToBraille{})
	core.Register(FromBraille{})
}
