package ops

import (
	"strings"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/opsutil"
)

func init() {
	core.Register(Substitute{})
}

// Substitute is a byte-substitution cipher.
type Substitute struct{}

// Meta returns the operation metadata.
func (Substitute) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Substitute",
		Module:      "Default",
		Description: "A substitution cipher allowing you to specify bytes to replace with other byte values. This can be used to create Caesar ciphers but is more powerful as any byte value can be substituted, not just letters, and the substitution values need not be in order.<br><br>Enter the bytes you want to replace in the Plaintext field and the bytes to replace them with in the Ciphertext field.<br><br>Non-printable bytes can be specified using string escape notation. For example, a line feed character can be written as either <code>\\n</code> or <code>\\x0a</code>.<br><br>Byte ranges can be specified using a hyphen. For example, the sequence <code>0123456789</code> can be written as <code>0-9</code>.<br><br>Note that blackslash characters are used to escape special characters, so will need to be escaped themselves if you want to use them on their own (e.g.<code>\\\\</code>).",
		InfoURL:     "https://wikipedia.org/wiki/Substitution_cipher",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (Substitute) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Plaintext", Type: core.ArgString, Value: "ABCDEFGHIJKLMNOPQRSTUVWXYZ"},
		{Name: "Ciphertext", Type: core.ArgString, Value: "XYZABCDEFGHIJKLMNOPQRSTUVW"},
		{Name: "Ignore case", Type: core.ArgBoolean, Value: false},
	}
}

// substituteChar converts one character using the dictionary. When ignoreCase is
// set, the value takes the case of the input character (and a case-insensitive
// key lookup is attempted).
func substituteChar(ch string, dict map[string]string, ignoreCase bool) string {
	if !ignoreCase {
		if v, ok := dict[ch]; ok {
			return v
		}
		return ch
	}
	isUpper := ch == strings.ToUpper(ch)
	if v, ok := dict[ch]; ok {
		if isUpper {
			return strings.ToUpper(v)
		}
		return strings.ToLower(v)
	}
	if isUpper {
		if v, ok := dict[strings.ToLower(ch)]; ok {
			return strings.ToUpper(v)
		}
	} else if v, ok := dict[strings.ToUpper(ch)]; ok {
		return strings.ToLower(v)
	}
	return ch
}

// Run performs the substitution.
func (Substitute) Run(in *core.Dish, args []any) (*core.Dish, error) {
	plaintext := []rune(opsutil.ExpandAlphRange(opsutil.ParseEscapedChars(args[0].(string))))
	ciphertext := []rune(opsutil.ExpandAlphRange(opsutil.ParseEscapedChars(args[1].(string))))
	ignoreCase := args[2].(bool)

	var out strings.Builder
	if len(plaintext) != len(ciphertext) {
		out.WriteString("Warning: Plaintext and Ciphertext lengths differ\n\n")
	}

	n := min(len(plaintext), len(ciphertext))
	dict := make(map[string]string, n)
	for i := range n {
		dict[string(plaintext[i])] = string(ciphertext[i])
	}

	for _, r := range dishText(in) {
		out.WriteString(substituteChar(string(r), dict, ignoreCase))
	}
	return core.NewDish([]byte(out.String()), core.TypeString), nil
}
