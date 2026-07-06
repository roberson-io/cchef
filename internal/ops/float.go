package ops

import (
	"encoding/binary"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(ToFloat{})
	core.Register(FromFloat{})
}

// floatDelims are the delimiter options shared by the Float operations
// (lib/Delim.mjs DELIM_OPTIONS).
var floatDelims = []string{"Space", "Comma", "Semi-colon", "Colon", "Line feed", "CRLF"}

// floatSizes are the size options for the Float operations.
var floatSizes = []string{"Float (4 bytes)", "Double (8 bytes)"}

// floatEndian are the endianness options for the Float operations.
var floatEndian = []string{"Big Endian", "Little Endian"}

// ToFloat converts IEEE754 bytes into their decimal representation.
type ToFloat struct{}

// Meta returns the operation metadata.
func (ToFloat) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "To Float",
		Module:      "Default",
		Description: "Convert to IEEE754 Floating Point Numbers.",
		InfoURL:     "https://wikipedia.org/wiki/IEEE_754",
		InputType:   core.TypeByteArray,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ToFloat) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Endianness", Type: core.ArgOption, Value: floatEndian},
		{Name: "Size", Type: core.ArgOption, Value: floatSizes},
		{Name: "Delimiter", Type: core.ArgOption, Value: floatDelims},
	}
}

// Run decodes the input. Ported from ToFloat.mjs.
func (ToFloat) Run(in *core.Dish, args []any) (*core.Dish, error) {
	isLE := args[0].(string) == "Little Endian"
	byteSize := 4
	if args[1].(string) == "Double (8 bytes)" {
		byteSize = 8
	}
	delim := charRep(args[2].(string))

	data := in.Bytes()
	if len(data)%byteSize != 0 {
		return nil, fmt.Errorf("input is not a multiple of %d", byteSize)
	}

	order := binary.ByteOrder(binary.BigEndian)
	if isLE {
		order = binary.LittleEndian
	}

	parts := make([]string, 0, len(data)/byteSize)
	for i := 0; i < len(data); i += byteSize {
		var f float64
		if byteSize == 4 {
			f = float64(math.Float32frombits(order.Uint32(data[i : i+4])))
		} else {
			f = math.Float64frombits(order.Uint64(data[i : i+8]))
		}
		parts = append(parts, floatToJS(f))
	}
	return core.NewDish([]byte(strings.Join(parts, delim)), core.TypeString), nil
}

// FromFloat converts decimal numbers into their IEEE754 byte representation.
type FromFloat struct{}

// Meta returns the operation metadata.
func (FromFloat) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "From Float",
		Module:      "Default",
		Description: "Convert from IEEE754 Floating Point Numbers.",
		InfoURL:     "https://wikipedia.org/wiki/IEEE_754",
		InputType:   core.TypeString,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (FromFloat) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Endianness", Type: core.ArgOption, Value: floatEndian},
		{Name: "Size", Type: core.ArgOption, Value: floatSizes},
		{Name: "Delimiter", Type: core.ArgOption, Value: floatDelims},
	}
}

// Run encodes the input. Ported from FromFloat.mjs.
func (FromFloat) Run(in *core.Dish, args []any) (*core.Dish, error) {
	s := in.String()
	if len(s) == 0 {
		return core.NewDish(nil, core.TypeByteArray), nil
	}

	isLE := args[0].(string) == "Little Endian"
	byteSize := 4
	if args[1].(string) == "Double (8 bytes)" {
		byteSize = 8
	}
	delim := charRep(args[2].(string))

	order := binary.ByteOrder(binary.BigEndian)
	if isLE {
		order = binary.LittleEndian
	}

	floats := strings.Split(s, delim)
	out := make([]byte, len(floats)*byteSize)
	for i, tok := range floats {
		f := jsParseFloat(tok)
		buf := out[i*byteSize : i*byteSize+byteSize]
		if byteSize == 4 {
			order.PutUint32(buf, float32ToIEEEBits(f))
		} else {
			order.PutUint64(buf, float64ToIEEEBits(f))
		}
	}
	return core.NewDish(out, core.TypeByteArray), nil
}

// floatToJS formats a float64 exactly as JavaScript's Number.prototype.toString
// (ECMAScript Number::toString), which is what CyberChef's To Float emits.
func floatToJS(f float64) string {
	switch {
	case math.IsNaN(f):
		return "NaN"
	case math.IsInf(f, 1):
		return "Infinity"
	case math.IsInf(f, -1):
		return "-Infinity"
	case f == 0:
		// JavaScript prints both +0 and -0 as "0".
		return "0"
	}

	neg := math.Signbit(f)
	// Shortest round-tripping scientific form, e.g. "5e-01" or "1.234e+05".
	es := strconv.FormatFloat(math.Abs(f), 'e', -1, 64)
	ei := strings.IndexByte(es, 'e')
	exp, _ := strconv.Atoi(es[ei+1:])
	digits := strings.Replace(es[:ei], ".", "", 1)
	k := len(digits) // number of significant digits
	n := exp + 1     // position of the decimal point (ECMAScript "n")

	var b strings.Builder
	if neg {
		b.WriteByte('-')
	}
	switch {
	case k <= n && n <= 21:
		b.WriteString(digits)
		b.WriteString(strings.Repeat("0", n-k))
	case 0 < n && n <= 21:
		b.WriteString(digits[:n])
		b.WriteByte('.')
		b.WriteString(digits[n:])
	case -6 < n && n <= 0:
		b.WriteString("0.")
		b.WriteString(strings.Repeat("0", -n))
		b.WriteString(digits)
	default:
		b.WriteString(digits[:1])
		if k > 1 {
			b.WriteByte('.')
			b.WriteString(digits[1:])
		}
		b.WriteByte('e')
		e := n - 1
		if e >= 0 {
			b.WriteByte('+')
		} else {
			b.WriteByte('-')
			e = -e
		}
		b.WriteString(strconv.Itoa(e))
	}
	return b.String()
}

// floatToken matches the numeric prefix that JavaScript's parseFloat would
// consume (it stops at the first character that cannot extend the number).
var floatToken = regexp.MustCompile(`^[+-]?(Infinity|\d+\.?\d*([eE][+-]?\d+)?|\.\d+([eE][+-]?\d+)?)`)

// jsParseFloat mirrors JavaScript's parseFloat: skip leading whitespace, then
// parse the longest valid numeric prefix, yielding NaN when none is present.
func jsParseFloat(s string) float64 {
	s = strings.TrimLeft(s, " \t\n\r\f\v")
	m := floatToken.FindString(s)
	if m == "" {
		return math.NaN()
	}
	f, _ := strconv.ParseFloat(m, 64)
	return f
}

// float32ToIEEEBits and float64ToIEEEBits encode a value like the `ieee754` npm
// package CyberChef wraps. That library writes a quiet NaN as an all-ones
// exponent with a mantissa of 1 (0x7f800001 / 0x7ff0000000000001) rather than
// Go's canonical 0x7fc00000 / 0x7ff8000000000000, so NaN is special-cased to
// keep byte-for-byte parity.
func float32ToIEEEBits(f float64) uint32 {
	if math.IsNaN(f) {
		return 0x7f800001
	}
	return math.Float32bits(float32(f))
}

func float64ToIEEEBits(f float64) uint64 {
	if math.IsNaN(f) {
		return 0x7ff0000000000001
	}
	return math.Float64bits(f)
}
