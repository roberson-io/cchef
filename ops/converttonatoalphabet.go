package ops

import (
	"strings"
	"unicode"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(ConvertToNATOAlphabet{})
}

// natoWords maps each character the operation reads out to its word, each
// carrying its own trailing space, exactly as CyberChef's table does.
var natoWords = map[rune]string{
	'A': "Alfa ", 'B': "Bravo ", 'C': "Charlie ", 'D': "Delta ", 'E': "Echo ",
	'F': "Foxtrot ", 'G': "Golf ", 'H': "Hotel ", 'I': "India ", 'J': "Juliett ",
	'K': "Kilo ", 'L': "Lima ", 'M': "Mike ", 'N': "November ", 'O': "Oscar ",
	'P': "Papa ", 'Q': "Quebec ", 'R': "Romeo ", 'S': "Sierra ", 'T': "Tango ",
	'U': "Uniform ", 'V': "Victor ", 'W': "Whiskey ", 'X': "X-ray ", 'Y': "Yankee ",
	'Z': "Zulu ", '0': "Zero ", '1': "One ", '2': "Two ", '3': "Three ",
	'4': "Four ", '5': "Five ", '6': "Six ", '7': "Seven ", '8': "Eight ",
	'9': "Nine ", ',': "Comma ", '/': "Fraction bar ", '.': "Full stop ",
}

// ConvertToNATOAlphabet spells characters out in the NATO phonetic alphabet.
type ConvertToNATOAlphabet struct{}

// Meta returns the operation metadata.
func (ConvertToNATOAlphabet) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Convert to NATO alphabet",
		Module:      "Default",
		Description: "Converts characters to their representation in the NATO phonetic alphabet.",
		InfoURL:     "https://wikipedia.org/wiki/NATO_phonetic_alphabet",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns no arguments; the operation takes none.
func (ConvertToNATOAlphabet) Args() []core.ArgDef { return nil }

// Run spells the input out. Characters outside the table pass through, so
// spacing between words follows the input's own.
func (ConvertToNATOAlphabet) Run(in *core.Dish, args []any) (*core.Dish, error) {
	var out strings.Builder
	for _, r := range dishText(in) {
		if word, there := natoWords[unicode.ToUpper(r)]; there && r < 0x80 {
			out.WriteString(word)
		} else {
			out.WriteRune(r)
		}
	}
	return core.NewDish([]byte(out.String()), core.TypeString), nil
}
