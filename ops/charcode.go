package ops

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/opsutil"
)

func init() {
	core.Register(ToCharcode{})
	core.Register(FromCharcode{})
}

// ToCharcode converts input characters to their numeric code points.
type ToCharcode struct{}

// Meta returns the operation metadata.
func (ToCharcode) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "To Charcode",
		Module:      "Default",
		Description: "Converts text to its unicode character code equivalent, in the given base.",
		InfoURL:     "https://wikipedia.org/wiki/Character_encoding",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ToCharcode) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Delimiter", Type: core.ArgOption, Value: decimalDelims},
		{Name: "Base", Type: core.ArgNumber, Integer: true, Value: 16},
	}
}

// Run encodes the input.
func (ToCharcode) Run(in *core.Dish, args []any) (*core.Dish, error) {
	delim := charRep(args[0].(string))
	base := int(args[1].(float64))
	if base < 2 || base > 36 {
		return nil, fmt.Errorf("base argument must be between 2 and 36")
	}

	var parts []string
	for _, r := range dishText(in) {
		ordinal := int64(r)
		if base == 16 {
			parts = append(parts, leftPad(strconv.FormatInt(ordinal, 16), charcodeHexPad(ordinal)))
		} else {
			parts = append(parts, strconv.FormatInt(ordinal, base))
		}
	}
	return core.NewDish([]byte(strings.Join(parts, delim)), core.TypeString), nil
}

// charcodeHexPad returns the zero-padding width CyberChef uses for a base-16
// code point.
func charcodeHexPad(v int64) int {
	switch {
	case v < 256:
		return 2
	case v < 65536:
		return 4
	case v < 16777216:
		return 6
	case v < 4294967296:
		return 8
	default:
		return 2
	}
}

// FromCharcode converts numeric character codes back into text.
type FromCharcode struct{}

// Meta returns the operation metadata.
func (FromCharcode) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "From Charcode",
		Module:      "Default",
		Description: "Converts unicode character codes back into text.",
		InfoURL:     "https://wikipedia.org/wiki/Character_encoding",
		InputType:   core.TypeString,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (FromCharcode) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Delimiter", Type: core.ArgOption, Value: decimalDelims},
		{Name: "Base", Type: core.ArgNumber, Integer: true, Value: 16},
	}
}

// Run decodes the input.
func (FromCharcode) Run(in *core.Dish, args []any) (*core.Dish, error) {
	delim := charRep(args[0].(string))
	base := int(args[1].(float64))
	if base < 2 || base > 36 {
		return nil, fmt.Errorf("base argument must be between 2 and 36")
	}
	input := in.String()
	if len(input) == 0 {
		return core.NewDish(nil, core.TypeByteArray), nil
	}

	bites := strings.Split(input, delim)
	// If the whole string is concatenated with no delimiter and too long to be a
	// single character, split it into pairs (matching CyberChef).
	if len(bites) == 1 && len(input) > 17 {
		bites = nil
		for i := 0; i < len(input); i += 2 {
			end := min(i+2, len(input))
			bites = append(bites, input[i:end])
		}
	}

	var sb strings.Builder
	for _, b := range bites {
		if b == "" {
			continue
		}
		v, err := strconv.ParseInt(b, base, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid charcode %q: %w", b, err)
		}
		sb.WriteRune(rune(v))
	}
	return core.NewDish(opsutil.TextAsBytes(sb.String()), core.TypeByteArray), nil
}
