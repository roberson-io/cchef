package ops

import (
	"math/big"
	"strings"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(UNIXTimestampToWindowsFiletime{})
	core.Register(WindowsFiletimeToUNIXTimestamp{})
}

// filetimeFormats are the radix options shared by the two Filetime operations.
var filetimeFormats = []string{"Decimal", "Hex (big endian)", "Hex (little endian)"}

// filetimeEpochDelta is the number of 100 ns intervals between the Windows
// (1601-01-01) and UNIX (1970-01-01) epochs.
func filetimeEpochDelta() bigNum { n, _ := parseBigNum("116444736000000000"); return n }

// filetimeUnitScale returns the bigNum by which a UNIX value in the given unit is
// multiplied to reach 100 ns Filetime ticks; ns is the reciprocal (÷100), so the
// second return value flags that the caller should divide instead of multiply.
func filetimeUnitScale(units string) (scale bigNum, divide bool) {
	switch units {
	case "Seconds (s)":
		n, _ := parseBigNum("10000000")
		return n, false
	case "Milliseconds (ms)":
		n, _ := parseBigNum("10000")
		return n, false
	case "Microseconds (μs)":
		n, _ := parseBigNum("10")
		return n, false
	default: // Nanoseconds (ns)
		n, _ := parseBigNum("100")
		return n, true
	}
}

// bigNumFromHex parses an unprefixed hex integer into a bigNum, returning NaN on
// failure (mirroring bignumber.js's BigNumber(str, 16)).
func bigNumFromHex(s string) bigNum {
	iv, ok := new(big.Int).SetString(strings.TrimSpace(s), 16)
	if !ok {
		return bnNaN
	}
	return finite(new(big.Rat).SetInt(iv))
}

// bigNumHexString renders an integer bigNum as lowercase hex (bignumber.js
// toString(16)). Fractional values (only reachable via the ns unit with an input
// that is not a multiple of 100) are truncated toward zero — a documented gap, as
// a fractional Filetime tick is not physically meaningful.
func bigNumHexString(n bigNum) string {
	if n.nan || n.inf != 0 {
		return n.String()
	}
	iv := new(big.Int).Quo(n.val.Num(), n.val.Denom())
	return iv.Text(16)
}

// filetimeFlipToLE reverses the byte order of a hex string for little-endian
// output.
func filetimeFlipToLE(result string) string {
	var b strings.Builder
	for i := len(result) - 2; i >= 0; i -= 2 {
		b.WriteByte(result[i])
		b.WriteByte(result[i+1])
	}
	if len(result)%2 != 0 {
		b.WriteByte('0')
		b.WriteByte(result[0])
	}
	return b.String()
}

// filetimeFlipFromLE reverses a little-endian hex string back to big-endian.
func filetimeFlipFromLE(input string) string {
	var b strings.Builder
	if len(input)%2 != 0 {
		b.WriteByte(input[len(input)-1])
	}
	for i := len(input) - len(input)%2 - 2; i >= 0; i -= 2 {
		b.WriteByte(input[i])
		b.WriteByte(input[i+1])
	}
	return b.String()
}

// UNIXTimestampToWindowsFiletime converts a UNIX timestamp to a Windows Filetime.
type UNIXTimestampToWindowsFiletime struct{}

// Meta returns the operation metadata.
func (UNIXTimestampToWindowsFiletime) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "UNIX Timestamp to Windows Filetime",
		Module:      "Default",
		Description: "Converts a UNIX timestamp to a Windows Filetime value.<br><br>A Windows Filetime is a 64-bit value representing the number of 100-nanosecond intervals since January 1, 1601 UTC.<br><br>A UNIX timestamp is a 32-bit value representing the number of seconds since January 1, 1970 UTC (the UNIX epoch).<br><br>This operation also supports UNIX timestamps in milliseconds, microseconds and nanoseconds.",
		InfoURL:     "https://msdn.microsoft.com/en-us/library/windows/desktop/ms724284(v=vs.85).aspx",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (UNIXTimestampToWindowsFiletime) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Input units", Type: core.ArgOption, Value: dateTimeUnits},
		{Name: "Output format", Type: core.ArgOption, Value: filetimeFormats},
	}
}

// Run converts the timestamp.
func (UNIXTimestampToWindowsFiletime) Run(in *core.Dish, args []any) (*core.Dish, error) {
	units := args[0].(string)
	format := args[1].(string)
	input := in.String()
	if input == "" {
		return core.NewDish([]byte(""), core.TypeString), nil
	}

	n, _ := parseBigNum(input)
	scale, divide := filetimeUnitScale(units)
	if divide {
		n = n.div(scale)
	} else {
		n = n.times(scale)
	}
	n = n.plus(filetimeEpochDelta())

	var result string
	if strings.HasPrefix(format, "Hex") {
		result = bigNumHexString(n)
	} else {
		result = n.String()
	}
	if format == "Hex (little endian)" {
		result = filetimeFlipToLE(result)
	}
	return core.NewDish([]byte(result), core.TypeString), nil
}

// WindowsFiletimeToUNIXTimestamp converts a Windows Filetime to a UNIX timestamp.
type WindowsFiletimeToUNIXTimestamp struct{}

// Meta returns the operation metadata.
func (WindowsFiletimeToUNIXTimestamp) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Windows Filetime to UNIX Timestamp",
		Module:      "Default",
		Description: "Converts a Windows Filetime value to a UNIX timestamp.<br><br>A Windows Filetime is a 64-bit value representing the number of 100-nanosecond intervals since January 1, 1601 UTC.<br><br>A UNIX timestamp is a 32-bit value representing the number of seconds since January 1, 1970 UTC (the UNIX epoch).<br><br>This operation also supports UNIX timestamps in milliseconds, microseconds and nanoseconds.",
		InfoURL:     "https://msdn.microsoft.com/en-us/library/windows/desktop/ms724284(v=vs.85).aspx",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (WindowsFiletimeToUNIXTimestamp) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Output units", Type: core.ArgOption, Value: dateTimeUnits},
		{Name: "Input format", Type: core.ArgOption, Value: filetimeFormats},
	}
}

// Run converts the filetime.
func (WindowsFiletimeToUNIXTimestamp) Run(in *core.Dish, args []any) (*core.Dish, error) {
	units := args[0].(string)
	format := args[1].(string)
	input := in.String()
	if input == "" {
		return core.NewDish([]byte(""), core.TypeString), nil
	}

	if format == "Hex (little endian)" {
		input = filetimeFlipFromLE(input)
	}
	var n bigNum
	if strings.HasPrefix(format, "Hex") {
		n = bigNumFromHex(input)
	} else {
		n, _ = parseBigNum(input)
	}
	n = n.minus(filetimeEpochDelta())

	scale, mult := filetimeUnitScale(units) // for ns, filetimeUnitScale divides on the way in, so multiply here
	if mult {
		n = n.times(scale)
	} else {
		n = n.div(scale)
	}
	return core.NewDish([]byte(n.String()), core.TypeString), nil
}
