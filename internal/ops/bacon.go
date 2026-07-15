package ops

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(BaconCipherEncode{})
	core.Register(BaconCipherDecode{})
}

// Bacon cipher resources, ported from CyberChef lib/Bacon.mjs.

// baconAlphabet pairs the letter set with an optional A-Z index → code table.
// The Standard alphabet folds I/J and U/V together (24 letters, 26-entry codes);
// the Complete alphabet uses all 26 letters with no remapping.
type baconAlphabet struct {
	alphabet string
	codes    []int // nil means the A-Z index is used directly
}

var (
	baconStandard = baconAlphabet{
		alphabet: "ABCDEFGHIKLMNOPQRSTUWXYZ",
		codes:    []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 19, 20, 21, 22, 23},
	}
	baconComplete = baconAlphabet{alphabet: "ABCDEFGHIJKLMNOPQRSTUVWXYZ"}
)

var baconAlphabetNames = []string{"Standard (I=J and U=V)", "Complete"}

// baconAlphabetByName returns the alphabet for the selected option (the option
// list only offers the two names, so anything but "Complete" is Standard).
func baconAlphabetByName(name string) baconAlphabet {
	if name == "Complete" {
		return baconComplete
	}
	return baconStandard
}

// swapZeroAndOne swaps '0'<->'1', leaving all other bytes untouched (matches the
// CyberChef helper; safe on UTF-8 since '0'/'1' never occur in a multi-byte rune).
func swapZeroAndOne(s string) string {
	b := []byte(s)
	for i, c := range b {
		switch c {
		case '0':
			b[i] = '1'
		case '1':
			b[i] = '0'
		}
	}
	return string(b)
}

// BaconCipherEncode conceals a message as groups of five binary digits (or A/B),
// one per letter. Ported from CyberChef BaconCipherEncode.mjs.
type BaconCipherEncode struct{}

// Meta returns the operation metadata.
func (BaconCipherEncode) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Bacon Cipher Encode",
		Module:      "Default",
		Description: "Bacon's cipher or the Baconian cipher is a method of steganography devised by Francis Bacon in 1605. A message is concealed in the presentation of text, rather than its content.",
		InfoURL:     "https://wikipedia.org/wiki/Bacon%27s_cipher",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (BaconCipherEncode) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Alphabet", Type: core.ArgOption, Value: baconAlphabetNames},
		{Name: "Translation", Type: core.ArgOption, Value: []string{"0/1", "A/B"}},
		{Name: "Keep extra characters", Type: core.ArgBoolean, Value: false},
		{Name: "Invert Translation", Type: core.ArgBoolean, Value: false},
	}
}

// Run encodes the input.
func (BaconCipherEncode) Run(in *core.Dish, args []any) (*core.Dish, error) {
	alpha := baconAlphabetByName(args[0].(string))
	translation := args[1].(string)
	keep := args[2].(bool)
	invert := args[3].(bool)

	var b strings.Builder
	for _, c := range in.String() {
		// CyberChef upper-cases each character and treats it as a letter only if
		// the first code unit lands in A-Z; everything else passes through. (Go's
		// simple case folding diverges from JS only for exotic full-casing letters
		// such as 'ß', which are non-ASCII and pass through either way.)
		if up := unicode.ToUpper(c); up >= 'A' && up <= 'Z' {
			code := int(up - 'A')
			if alpha.codes != nil {
				code = alpha.codes[code]
			}
			fmt.Fprintf(&b, "%05b", code)
		} else {
			b.WriteRune(c)
		}
	}
	output := b.String()

	if invert {
		output = swapZeroAndOne(output)
	}
	if !keep {
		output = baconKeep(output, func(c byte) bool { return c == '0' || c == '1' })
		output = baconGroupBy5(output)
	}
	if translation == "A/B" {
		output = baconMapBytes(output, map[byte]byte{'0': 'A', '1': 'B'})
	}
	return core.NewDish([]byte(output), core.TypeString), nil
}

// baconKeep drops every byte for which keep returns false.
func baconKeep(s string, keep func(byte) bool) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if keep(s[i]) {
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

// baconGroupBy5 splits into non-overlapping 5-character groups joined by a space,
// discarding a trailing partial group (mirrors `str.match(/(.{5})/g).join(" ")`).
func baconGroupBy5(s string) string {
	var groups []string
	for i := 0; i+5 <= len(s); i += 5 {
		groups = append(groups, s[i:i+5])
	}
	return strings.Join(groups, " ")
}

// baconMapBytes replaces each byte present in m, leaving others untouched.
func baconMapBytes(s string, m map[byte]byte) string {
	b := []byte(s)
	for i, c := range b {
		if r, ok := m[c]; ok {
			b[i] = r
		}
	}
	return string(b)
}

// BaconCipherDecode recovers the concealed message. Ported from CyberChef
// BaconCipherDecode.mjs.
type BaconCipherDecode struct{}

// Meta returns the operation metadata.
func (BaconCipherDecode) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Bacon Cipher Decode",
		Module:      "Default",
		Description: "Bacon's cipher or the Baconian cipher is a method of steganography devised by Francis Bacon in 1605. A message is concealed in the presentation of text, rather than its content.",
		InfoURL:     "https://wikipedia.org/wiki/Bacon%27s_cipher",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (BaconCipherDecode) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Alphabet", Type: core.ArgOption, Value: baconAlphabetNames},
		{Name: "Translation", Type: core.ArgOption, Value: []string{"0/1", "A/B", "Case", "A-M/N-Z first letter"}},
		{Name: "Invert Translation", Type: core.ArgBoolean, Value: false},
	}
}

// Run decodes the input.
func (BaconCipherDecode) Run(in *core.Dish, args []any) (*core.Dish, error) {
	alpha := baconAlphabetByName(args[0].(string))
	translation := args[1].(string)
	invert := args[2].(bool)
	input := in.String()

	// Remove invalid characters (BACON_CLEARER_MAP). The A-M/N-Z variant has no
	// clearer entry, so CyberChef calls `replace(undefined, "")`, which coerces
	// undefined to the string "undefined" and strips its first literal occurrence.
	switch translation {
	case "0/1":
		input = baconKeep(input, func(c byte) bool { return c == '0' || c == '1' })
	case "A/B":
		input = baconKeep(input, func(c byte) bool { return c == 'A' || c == 'B' || c == 'a' || c == 'b' })
	case "Case":
		input = baconKeep(input, func(c byte) bool {
			return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
		})
	default: // "A-M/N-Z first letter"
		input = strings.Replace(input, "undefined", "", 1)
	}

	// Normalize to a 0/1 string (BACON_NORMALIZE_MAP / special cases).
	switch translation {
	case "A/B":
		input = baconMapBytes(input, map[byte]byte{'A': '0', 'a': '0', 'B': '1', 'b': '1'})
	case "Case":
		input = baconMapBytes(input, baconCaseMap)
	case "A-M/N-Z first letter":
		input = baconFirstLetterBits(input)
	}

	if invert {
		input = swapZeroAndOne(input)
	}

	// Group into fives and map each 5-bit code to a letter (or "?" if out of range).
	var out strings.Builder
	for i := 0; i+5 <= len(input); i += 5 {
		n := 0
		for j := range 5 {
			n = n<<1 | int(input[i+j]-'0')
		}
		if n < len(alpha.alphabet) {
			out.WriteByte(alpha.alphabet[n])
		} else {
			out.WriteByte('?')
		}
	}
	return core.NewDish([]byte(out.String()), core.TypeString), nil
}

// baconCaseMap turns upper-case letters into '1' and lower-case into '0' (the
// "Case" translation); after its clearer the input holds only ASCII letters.
var baconCaseMap = func() map[byte]byte {
	m := make(map[byte]byte, 52)
	for c := byte('A'); c <= 'Z'; c++ {
		m[c] = '1'
	}
	for c := byte('a'); c <= 'z'; c++ {
		m[c] = '0'
	}
	return m
}()

// baconFirstLetterBits implements the "A-M/N-Z first letter" translation: split
// on whitespace and emit '1' when a word's upper-cased first letter is >= 'N',
// else '0'.
func baconFirstLetterBits(s string) string {
	var b strings.Builder
	for word := range strings.FieldsSeq(s) {
		r := []rune(word)[0]
		if unicode.ToUpper(r) >= 'N' {
			b.WriteByte('1')
		} else {
			b.WriteByte('0')
		}
	}
	return b.String()
}
