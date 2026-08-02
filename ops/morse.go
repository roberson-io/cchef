package ops

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf16"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(ToMorseCode{})
	core.Register(FromMorseCode{})
}

// morseTable maps each supported character to its International Morse Code
// pattern (dots and dashes). Transcribed from CyberChef's MORSE_TABLE.
var morseTable = map[rune]string{
	'A': ".-", 'B': "-...", 'C': "-.-.", 'D': "-..", 'E': ".", 'F': "..-.",
	'G': "--.", 'H': "....", 'I': "..", 'J': ".---", 'K': "-.-", 'L': ".-..",
	'M': "--", 'N': "-.", 'O': "---", 'P': ".--.", 'Q': "--.-", 'R': ".-.",
	'S': "...", 'T': "-", 'U': "..-", 'V': "...-", 'W': ".--", 'X': "-..-",
	'Y': "-.--", 'Z': "--..",
	'1': ".----", '2': "..---", '3': "...--", '4': "....-", '5': ".....",
	'6': "-....", '7': "--...", '8': "---..", '9': "----.", '0': "-----",
	'.': ".-.-.-", ',': "--..--", ':': "---...", ';': "-.-.-.", '!': "-.-.--",
	'?': "..--..", '\'': ".----.", '"': ".-..-.", '/': "-..-.", '-': "-....-",
	'+': ".-.-.", '(': "-.--.", ')': "-.--.-", '@': ".--.-.", '=': "-...-",
	'&': ".-...", '_': "..--.-", '$': "...-..-", ' ': ".......",
}

// morseReverse maps a Morse pattern back to its character (built from morseTable;
// every pattern is unique).
var morseReverse = func() map[string]rune {
	m := make(map[string]rune, len(morseTable))
	for r, sig := range morseTable {
		m[sig] = r
	}
	return m
}()

// morseFormats lists the dash/dot rendering options.
var morseFormats = []string{"-/.", "_/.", "Dash/Dot", "DASH/DOT", "dash/dot"}

// morseLetterDelims / morseWordDelims mirror CyberChef's LETTER_DELIM_OPTIONS and
// WORD_DELIM_OPTIONS.
var (
	morseLetterDelims = []string{"Space", "Line feed", "CRLF", "Forward slash", "Backslash", "Comma", "Semi-colon", "Colon"}
	morseWordDelims   = []string{"Line feed", "CRLF", "Forward slash", "Backslash", "Comma", "Semi-colon", "Colon"}
)

// morseLineSplit splits on LF or CRLF; morseWordSplit splits on runs of spaces.
var (
	morseLineSplit = regexp.MustCompile(`\r?\n`)
	morseWordSplit = regexp.MustCompile(` +`)
	// morseDashChars matches every accepted dash spelling; morseDotChars the dots.
	morseDashChars = regexp.MustCompile(`(?i)-|‐|−|_|–|—|dash`)
	morseDotChars  = regexp.MustCompile(`(?i)\.|·|dot`)
)

// ToMorseCode translates text into International Morse Code.
type ToMorseCode struct{}

// Meta returns the operation metadata.
func (ToMorseCode) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "To Morse Code",
		Module:      "Default",
		Description: "Translates alphanumeric characters into International Morse Code.<br><br>Ignores non-Morse characters.<br><br>e.g. <code>SOS</code> becomes <code>... --- ...</code>",
		InfoURL:     "https://wikipedia.org/wiki/Morse_code",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ToMorseCode) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Format options", Type: core.ArgOption, Value: morseFormats},
		{Name: "Letter delimiter", Type: core.ArgOption, Value: morseLetterDelims},
		{Name: "Word delimiter", Type: core.ArgOption, Value: morseWordDelims},
	}
}

// Run encodes the input.
func (ToMorseCode) Run(in *core.Dish, args []any) (*core.Dish, error) {
	format := strings.SplitN(args[0].(string), "/", 2)
	dash, dot := format[0], format[1]
	letterDelim := charRep(args[1].(string))
	wordDelim := charRep(args[2].(string))

	lines := morseLineSplit.Split(in.String(), -1)
	for i, line := range lines {
		words := morseWordSplit.Split(line, -1)
		for j, word := range words {
			words[j] = morseEncodeWord(word, dash, dot, letterDelim)
		}
		lines[i] = strings.Join(words, wordDelim)
	}
	return core.NewDish([]byte(strings.Join(lines, "\n")), core.TypeString), nil
}

// morseEncodeWord encodes a single word. CyberChef iterates UTF-16 code units, so
// a non-BMP character becomes two (empty) letters, each still delimited.
func morseEncodeWord(word, dash, dot, letterDelim string) string {
	units := utf16.Encode([]rune(word))
	letters := make([]string, len(units))
	for i, u := range units {
		pattern, ok := morseTable[unicode.ToUpper(rune(u))]
		if !ok {
			continue // empty letter for a non-Morse code unit
		}
		letters[i] = strings.NewReplacer(".", dot, "-", dash).Replace(pattern)
	}
	return strings.Join(letters, letterDelim)
}

// FromMorseCode translates Morse Code back into upper-case text.
type FromMorseCode struct{}

// Meta returns the operation metadata.
func (FromMorseCode) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "From Morse Code",
		Module:      "Default",
		Description: "Translates Morse Code into (upper case) alphanumeric characters.",
		InfoURL:     "https://wikipedia.org/wiki/Morse_code",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (FromMorseCode) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Letter delimiter", Type: core.ArgOption, Value: morseLetterDelims},
		{Name: "Word delimiter", Type: core.ArgOption, Value: morseWordDelims},
	}
}

// Run decodes the input.
func (FromMorseCode) Run(in *core.Dish, args []any) (*core.Dish, error) {
	letterDelim := charRep(args[0].(string))
	wordDelim := charRep(args[1].(string))

	s := morseDashChars.ReplaceAllString(in.String(), "-")
	s = morseDotChars.ReplaceAllString(s, ".")

	words := strings.Split(s, wordDelim)
	for i, word := range words {
		var b strings.Builder
		for signal := range strings.SplitSeq(word, letterDelim) {
			if r, ok := morseReverse[signal]; ok {
				b.WriteRune(r)
			}
		}
		words[i] = b.String()
	}
	return core.NewDish([]byte(strings.Join(words, " ")), core.TypeString), nil
}
