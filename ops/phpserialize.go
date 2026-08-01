package ops

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/jsonval"
)

func init() {
	core.Register(PHPSerialize{})
}

// PHPSerialize struct.
type PHPSerialize struct{}

// Meta returns the operation metadata.
func (PHPSerialize) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "PHP Serialize",
		Module:      "Default",
		Description: "Performs PHP serialization on JSON data.",
		InfoURL:     "https://wikipedia.org/wiki/Serialization#Programming_language_support",
		InputType:   core.TypeJSON,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (PHPSerialize) Args() []core.ArgDef { return nil }

// Run serialises the JSON input to PHP serialized form.
func (PHPSerialize) Run(in *core.Dish, _ []any) (*core.Dish, error) {
	v, err := jsonval.ParseOrdered(in.Bytes())
	if err != nil {
		return nil, fmt.Errorf("PHP Serialize: parse JSON input: %w", err)
	}
	return core.NewDish([]byte(phpSerialize(v)), core.TypeString), nil
}

// phpSerialize walks the ordered JSON value tree producing PHP serialized output,
// matching PHPSerialize.mjs (arrays and objects both become `a:N:{...}`).
func phpSerialize(v any) string {
	switch x := v.(type) {
	case nil:
		return "N;"
	case bool:
		if x {
			return "b:1;"
		}
		return "b:0;"
	case float64:
		return phpSerializeNumber(x) + ";"
	case string:
		return fmt.Sprintf(`s:%d:"%s";`, phpUTF16Len(x), x)
	case []any:
		var b strings.Builder
		fmt.Fprintf(&b, "a:%d:{", len(x))
		for i, e := range x {
			b.WriteString(phpSerialize(float64(i)))
			b.WriteString(phpSerialize(e))
		}
		b.WriteByte('}')
		return b.String()
	case jsonval.Object:
		var b strings.Builder
		fmt.Fprintf(&b, "a:%d:{", len(x))
		for _, p := range x {
			b.WriteString(phpSerialize(p.K))
			b.WriteString(phpSerialize(p.V))
		}
		b.WriteByte('}')
		return b.String()
	}
	return ""
}

// phpSerializeNumber renders a number as `i:N` when it is an integer (per the
// original's parseInt check) or `d:N` otherwise, using JS Number-to-string.
func phpSerializeNumber(v float64) string {
	s := jsonval.FormatNumber(v)
	if p := phpParseIntPrefix(s); !math.IsNaN(p) && p == v {
		return "i:" + s
	}
	return "d:" + s
}

// phpParseIntPrefix mimics parseInt(s, 10): the leading optional sign and digits
// parsed as a float, or NaN if there are no digits.
func phpParseIntPrefix(s string) float64 {
	i := 0
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		i++
	}
	start := i
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == start {
		return math.NaN()
	}
	n, err := strconv.ParseFloat(s[:i], 64)
	if err != nil {
		return math.NaN()
	}
	return n
}

// phpUTF16Len returns the number of UTF-16 code units in s, matching JS's
// String.length used for the `s:LEN:` field.
func phpUTF16Len(s string) int {
	return len(utf16.Encode([]rune(s)))
}
