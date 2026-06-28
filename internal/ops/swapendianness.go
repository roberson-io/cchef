package ops

import (
	"fmt"
	"strconv"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(SwapEndianness{})
}

// SwapEndianness reverses the byte order within fixed-length words.
type SwapEndianness struct{}

// Meta returns the operation metadata.
func (SwapEndianness) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Swap endianness",
		Module:      "Default",
		Description: "Switches the data unit (word) endianness by reversing the byte order within each fixed-length word.",
		InfoURL:     "https://wikipedia.org/wiki/Endianness",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (SwapEndianness) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Data format", Type: core.ArgOption, Value: []string{"Hex", "Raw"}},
		{Name: "Word length (bytes)", Type: core.ArgNumber, Value: 4},
		{Name: "Pad incomplete words", Type: core.ArgBoolean, Value: true},
	}
}

// Run swaps endianness. Ported from CyberChef SwapEndianness.mjs.
func (SwapEndianness) Run(in *core.Dish, args []any) (*core.Dish, error) {
	dataFormat := args[0].(string)
	wordLength := int(args[1].(float64))
	pad := args[2].(bool)
	if wordLength <= 0 {
		return nil, fmt.Errorf("word length must be greater than 0")
	}

	// Decode input into raw bytes per the data format.
	var data []byte
	switch dataFormat {
	case "Hex":
		var err error
		data, err = swapParseHex(in.String())
		if err != nil {
			return nil, err
		}
	default: // Raw
		data = in.Bytes()
	}

	// Split into words (optionally padding the last), reverse each, flatten.
	var result []byte
	for i := 0; i < len(data); i += wordLength {
		end := i + wordLength
		if end > len(data) {
			end = len(data)
		}
		word := append([]byte(nil), data[i:end]...)
		if pad {
			for len(word) < wordLength {
				word = append(word, 0)
			}
		}
		for j := len(word) - 1; j >= 0; j-- {
			result = append(result, word[j])
		}
	}

	var out string
	if dataFormat == "Hex" {
		out = toHex(result, " ", "")
	} else {
		out = string(result)
	}
	return core.NewDish([]byte(out), core.TypeString), nil
}

// swapParseHex parses hex text into bytes, splitting on any non-hex run
// (CyberChef's fromHex "Auto" behaviour).
func swapParseHex(s string) ([]byte, error) {
	var out []byte
	for _, part := range nonHex.Split(s, -1) {
		for j := 0; j+2 <= len(part); j += 2 {
			v, err := strconv.ParseUint(part[j:j+2], 16, 8)
			if err != nil {
				return nil, fmt.Errorf("invalid hex byte %q: %w", part[j:j+2], err)
			}
			out = append(out, byte(v))
		}
	}
	return out, nil
}
