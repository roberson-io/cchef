package ops

import (
	"errors"
	"regexp"
	"strings"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(VigenereEncode{})
	core.Register(VigenereDecode{})
}

// vigenereDescription is shared by both operations.
const vigenereDescription = "The Vigenere cipher is a method of encrypting alphabetic text by using a series of different Caesar ciphers based on the letters of a keyword. It is a simple form of polyalphabetic substitution."

// vigenereKeyRe matches a key made only of letters.
var vigenereKeyRe = regexp.MustCompile(`^[a-zA-Z]+$`)

// vigenere applies the Vigenère cipher to input with the given key. decode
// selects decryption. Non-alphabet characters pass through unchanged and do not
// advance the key (matching CyberChef's fail counter).
func vigenere(input, keyRaw string, decode bool) (string, error) {
	key := strings.ToLower(keyRaw)
	if key == "" {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return "", errors.New("No key entered")
	}
	if !vigenereKeyRe.MatchString(key) {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return "", errors.New("The key must consist only of letters")
	}
	var b strings.Builder
	alphaCount := 0 // number of alphabet characters seen (CyberChef's i - fail)
	for _, c := range input {
		var base rune
		var msgIndex int
		switch {
		case c >= 'a' && c <= 'z':
			base, msgIndex = 'a', int(c-'a')
		case c >= 'A' && c <= 'Z':
			base, msgIndex = 'A', int(c-'A')
		default:
			b.WriteRune(c)
			continue
		}
		keyIndex := int(key[alphaCount%len(key)] - 'a')
		alphaCount++
		var shifted int
		if decode {
			shifted = (msgIndex - keyIndex + 26) % 26
		} else {
			shifted = (keyIndex + msgIndex) % 26
		}
		b.WriteRune(base + rune(shifted))
	}
	return b.String(), nil
}

// VigenereEncode encrypts with the Vigenère cipher.
type VigenereEncode struct{}

// Meta returns the operation metadata.
func (VigenereEncode) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Vigenère Encode",
		Module:      "Ciphers",
		Description: vigenereDescription,
		InfoURL:     "https://wikipedia.org/wiki/Vigenère_cipher",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (VigenereEncode) Args() []core.ArgDef {
	return []core.ArgDef{{Name: "Key", Type: core.ArgString, Value: ""}}
}

// Run encrypts with the Vigenère cipher.
func (VigenereEncode) Run(in *core.Dish, args []any) (*core.Dish, error) {
	out, err := vigenere(in.String(), args[0].(string), false)
	if err != nil {
		return nil, err
	}
	return core.NewDish([]byte(out), core.TypeString), nil
}

// VigenereDecode decrypts with the Vigenère cipher.
type VigenereDecode struct{}

// Meta returns the operation metadata.
func (VigenereDecode) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Vigenère Decode",
		Module:      "Ciphers",
		Description: vigenereDescription,
		InfoURL:     "https://wikipedia.org/wiki/Vigenère_cipher",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (VigenereDecode) Args() []core.ArgDef {
	return []core.ArgDef{{Name: "Key", Type: core.ArgString, Value: ""}}
}

// Run decrypts with the Vigenère cipher.
func (VigenereDecode) Run(in *core.Dish, args []any) (*core.Dish, error) {
	out, err := vigenere(in.String(), args[0].(string), true)
	if err != nil {
		return nil, err
	}
	return core.NewDish([]byte(out), core.TypeString), nil
}
