package ops

import (
	"regexp"
	"strings"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(ToModhex{})
	core.Register(FromModhex{})
}

// ToModhex converts input bytes to a modhex string.
type ToModhex struct{}

// Meta returns the operation metadata.
func (ToModhex) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "To Modhex",
		Module:      "Default",
		Description: "Converts the input string to modhex bytes separated by the specified delimiter.",
		InfoURL:     "https://en.wikipedia.org/wiki/YubiKey#ModHex",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ToModhex) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Delimiter", Type: core.ArgOption, Value: toModhexDelims},
		{Name: "Bytes per line", Type: core.ArgNumber, Integer: true, Value: float64(0)},
	}
}

// Run encodes the input.
func (ToModhex) Run(in *core.Dish, args []any) (*core.Dish, error) {
	delim := charRep(args[0].(string))
	lineSize := int(args[1].(float64))
	return core.NewDish([]byte(toModhex(in.Bytes(), delim, lineSize)), core.TypeString), nil
}

// FromModhex converts a modhex string back into its raw value.
type FromModhex struct{}

// Meta returns the operation metadata.
func (FromModhex) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "From Modhex",
		Module:      "Default",
		Description: "Converts a modhex byte string back into its raw value.",
		InfoURL:     "https://en.wikipedia.org/wiki/YubiKey#ModHex",
		InputType:   core.TypeString,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (FromModhex) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Delimiter", Type: core.ArgOption, Value: append([]string{"Auto"}, toModhexDelims...)},
	}
}

// Run decodes the input.
func (FromModhex) Run(in *core.Dish, args []any) (*core.Dish, error) {
	return core.NewDish(fromModhex(in.String(), args[0].(string)), core.TypeByteArray), nil
}

// toModhexDelims are the delimiter options for To Modhex.
var toModhexDelims = []string{"Space", "Percent", "Comma", "Semi-colon", "Colon", "Line feed", "CRLF", "None"}

// Modhex substitutes the 16 hex nibbles with a consonant alphabet (Modhex.mjs).
const (
	modhexAlphabet = "cbdefghijklnrtuv"
	hexAlphabet    = "0123456789abcdef"
)

// whitespaceRE and nonModhex mirror the JS /\s/g strip and the "Auto"
// /[^cbdefghijklnrtuv]/gi split used by fromModhex.
var (
	whitespaceRE = regexp.MustCompile(`\s`)
	nonModhex    = regexp.MustCompile(`[^cbdefghijklnrtuv]`)
)

// toModhex converts bytes to a delimited modhex string, optionally inserting a
// line break after every lineSize bytes.
func toModhex(data []byte, delim string, lineSize int) string {
	if len(data) == 0 {
		return ""
	}

	hexStr := toHex(data, "", "")
	var mh strings.Builder
	for i := 0; i < len(hexStr); i++ {
		mh.WriteByte(modhexAlphabet[strings.IndexByte(hexAlphabet, hexStr[i])])
	}
	modhexString := mh.String()

	var out strings.Builder
	groups := len(modhexString) / 2
	for i := range groups {
		out.WriteString(modhexString[i*2 : i*2+2])
		out.WriteString(delim)
		if lineSize > 0 && i != groups-1 && (i+1)%lineSize == 0 {
			out.WriteString("\n")
		}
	}

	s := out.String()
	if len(delim) > 0 {
		s = s[:len(s)-len(delim)]
	}
	return s
}

// fromModhex converts a modhex string back into bytes. Each modhex letter's
// position in the alphabet is its nibble value (the alphabet is parallel to
// "0123456789abcdef"), so pairs of letters map directly to bytes.
func fromModhex(data, delim string) []byte {
	data = whitespaceRE.ReplaceAllString(strings.ToLower(data), "")

	var parts []string
	switch delim {
	case "None":
		parts = []string{data}
	case "Auto", "":
		parts = nonModhex.Split(data, -1)
	default:
		parts = strings.Split(data, charRep(delim))
	}

	var nibbles []int
	for _, p := range parts {
		for i := 0; i < len(p); i++ {
			if idx := strings.IndexByte(modhexAlphabet, p[i]); idx >= 0 {
				nibbles = append(nibbles, idx)
			}
		}
	}

	out := make([]byte, 0, len(nibbles)/2)
	for i := 0; i+1 < len(nibbles); i += 2 {
		out = append(out, byte(nibbles[i]<<4|nibbles[i+1])) // #nosec G115 -- two 4-bit nibbles combined into a byte
	}
	return out
}
