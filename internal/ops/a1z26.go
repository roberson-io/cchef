package ops

import (
	"errors"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(A1Z26CipherEncode{})
	core.Register(A1Z26CipherDecode{})
}

// a1z26Delims are the delimiter options for the A1Z26 operations (CyberChef
// DELIM_OPTIONS).
var a1z26Delims = []string{"Space", "Comma", "Semi-colon", "Colon", "Line feed", "CRLF"}

// A1Z26CipherEncode converts alphabet characters into their alphabet-order
// number (a=1 ... z=26), dropping non-alphabet characters.
type A1Z26CipherEncode struct{}

// Meta returns the operation metadata.
func (A1Z26CipherEncode) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "A1Z26 Cipher Encode",
		Module:      "Ciphers",
		Description: "Converts alphabet characters into their corresponding alphabet order number. e.g. a becomes 1 and b becomes 2. Non-alphabet characters are dropped.",
		InfoURL:     "",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (A1Z26CipherEncode) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Delimiter", Type: core.ArgOption, Value: a1z26Delims},
	}
}

// Run encodes the input. Ported from CyberChef A1Z26CipherEncode.mjs.
func (A1Z26CipherEncode) Run(in *core.Dish, args []any) (*core.Dish, error) {
	delim := charRep(args[0].(string))
	lower := strings.ToLower(in.String())

	var parts []string
	for _, r := range lower {
		ordinal := int(r) - 96
		if ordinal > 0 && ordinal <= 26 {
			parts = append(parts, strconv.Itoa(ordinal))
		}
	}
	return core.NewDish([]byte(strings.Join(parts, delim)), core.TypeString), nil
}

// A1Z26CipherDecode converts alphabet-order numbers back into their
// corresponding alphabet characters (1=a ... 26=z).
type A1Z26CipherDecode struct{}

// Meta returns the operation metadata.
func (A1Z26CipherDecode) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "A1Z26 Cipher Decode",
		Module:      "Ciphers",
		Description: "Converts alphabet order numbers into their corresponding alphabet character. e.g. 1 becomes a and 2 becomes b.",
		InfoURL:     "",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (A1Z26CipherDecode) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Delimiter", Type: core.ArgOption, Value: a1z26Delims},
	}
}

// Run decodes the input. Ported from CyberChef A1Z26CipherDecode.mjs.
func (A1Z26CipherDecode) Run(in *core.Dish, args []any) (*core.Dish, error) {
	delim := charRep(args[0].(string))
	input := in.String()
	if len(input) == 0 {
		return core.NewDish([]byte{}, core.TypeString), nil
	}

	bites := strings.Split(input, delim)
	var latin1 []rune
	for _, b := range bites {
		// The range check mirrors JS `bites[i] < 1 || bites[i] > 26`, which
		// coerces the token with Number(): an empty token is 0 (out of range),
		// and a non-numeric token is NaN (both comparisons false, so it slips
		// through). classifySeg reproduces that Number() coercion.
		if seg := classifySeg(b, false); seg.isNum && (seg.num < 1 || seg.num > 26) {
			return nil, errors.New("Error: all numbers must be between 1 and 26.") //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		}
		// The value uses parseInt(b, 10); a token with no leading digits is NaN,
		// and chr(NaN + 96) yields a NUL byte.
		v, ok := jsParseInt(b, 10)
		if !ok {
			latin1 = append(latin1, 0)
			continue
		}
		latin1 = append(latin1, rune(v+96)) // #nosec G115 -- code point narrowed to a rune, mirroring CyberChef Utils.chr
	}
	return core.NewDish([]byte(string(latin1)), core.TypeString), nil
}
