package ops

import (
	"errors"
	"strings"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(ParityBit{})
}

// ParityBit adds or removes a parity bit on a string of binary digits. Ported
// from CyberChef's ParityBit operation (and its lib/ParityBit.mjs helpers): with
// a delimiter set, each delimited token is handled independently.
type ParityBit struct{}

// Meta returns the operation metadata.
func (ParityBit) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Parity Bit",
		Module:      "Default",
		Description: "In information theory and telecommunications, a parity bit is a bit added to a string of binary code. Parity bits are a simple form of error detecting code and can be applied to a single byte of information or across a set of bytes.",
		InfoURL:     "https://wikipedia.org/wiki/Parity_bit",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ParityBit) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Mode", Type: core.ArgOption, Value: []string{"Even Parity", "Odd Parity"}},
		{Name: "Postion", Type: core.ArgOption, Value: []string{"Start", "End"}}, //nolint:misspell // "Postion" matches CyberChef's (misspelled) argument name / CLI flag
		{Name: "Encode or Decode", Type: core.ArgOption, Value: []string{"Encode", "Decode"}},
		{Name: "Delimiter", Type: core.ArgString, Value: ""},
	}
}

// Run applies the parity operation to the input (per delimited token if a
// delimiter is given).
func (ParityBit) Run(in *core.Dish, args []any) (*core.Dish, error) {
	input := in.String()
	if len(input) == 0 {
		return core.NewDish([]byte(input), core.TypeString), nil
	}
	mode := args[0].(string)
	position := args[1].(string)
	encode := args[2].(string) == "Encode"
	delim := args[3].(string)

	apply := func(s string) (string, error) {
		if encode {
			return calculateParityBit(s, mode, position, delim)
		}
		return decodeParityBit(s, position), nil
	}

	if len(delim) > 0 {
		tokens := strings.Split(input, delim)
		for i, tok := range tokens {
			out, err := apply(tok)
			if err != nil {
				return nil, err
			}
			tokens[i] = out
		}
		return core.NewDish([]byte(strings.Join(tokens, delim)), core.TypeString), nil
	}
	out, err := apply(input)
	if err != nil {
		return nil, err
	}
	return core.NewDish([]byte(out), core.TypeString), nil
}

// calculateParityBit prepends or appends a parity bit so the total number of set
// bits matches the chosen parity. Non-binary characters (other than spaces and
// the delimiter) are rejected.
func calculateParityBit(input, mode, position, delim string) (string, error) {
	count1s := 0
	for i := 0; i < len(input); i++ {
		c := input[i : i+1]
		switch {
		case c == "1":
			count1s++
		case c != delim && c != "0" && c != " ":
			return "", errors.New(`unexpected character encountered: "` + c + `"`) //nolint:staticcheck,revive // verbatim CyberChef text
		}
	}
	parityBit := "1"
	flipflop := 0
	if mode != "Even Parity" {
		flipflop = 1
	}
	if count1s%2 == flipflop {
		parityBit = "0"
	}
	if position == "End" {
		return input + parityBit, nil
	}
	return parityBit + input, nil
}

// decodeParityBit removes the parity bit from the start or end of input.
func decodeParityBit(input, position string) string {
	if position == "End" {
		return input[:len(input)-1]
	}
	return input[1:]
}
