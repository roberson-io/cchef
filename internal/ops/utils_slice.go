package ops

import (
	"fmt"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(Head{})
	core.Register(Tail{})
	core.Register(DropBytes{})
	core.Register(TakeBytes{})
	core.Register(DropNthBytes{})
	core.Register(TakeNthBytes{})
}

// Head keeps the first N sections of the input.
type Head struct{}

// Meta returns the operation metadata.
func (Head) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Head",
		Module:      "Default",
		Description: "Keeps only the first N sections (lines) of the input. A negative N drops the last |N| sections.",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (Head) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Delimiter", Type: core.ArgOption, Value: inputDelims},
		{Name: "Number", Type: core.ArgNumber, Integer: true, Value: 10},
	}
}

// Run keeps the first sections. Ported from CyberChef Head.mjs.
func (Head) Run(in *core.Dish, args []any) (*core.Dish, error) {
	delim := charRep(args[0].(string))
	n := int(args[1].(float64))
	sections := splitByDelim(in.String(), delim)
	limit := n
	if n < 0 {
		limit = len(sections) + n
	}
	var kept []string
	for i, s := range sections {
		if i+1 <= limit {
			kept = append(kept, s)
		}
	}
	return core.NewDish([]byte(strings.Join(kept, delim)), core.TypeString), nil
}

// Tail keeps the last N sections of the input.
type Tail struct{}

// Meta returns the operation metadata.
func (Tail) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Tail",
		Module:      "Default",
		Description: "Keeps only the last N sections (lines) of the input. A negative N drops the first |N| sections.",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (Tail) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Delimiter", Type: core.ArgOption, Value: inputDelims},
		{Name: "Number", Type: core.ArgNumber, Integer: true, Value: 10},
	}
}

// Run keeps the last sections. Ported from CyberChef Tail.mjs.
func (Tail) Run(in *core.Dish, args []any) (*core.Dish, error) {
	delim := charRep(args[0].(string))
	n := int(args[1].(float64))
	sections := splitByDelim(in.String(), delim)
	threshold := len(sections) - n
	if n < 0 {
		threshold = -n
	}
	var kept []string
	for i, s := range sections {
		if i+1 > threshold {
			kept = append(kept, s)
		}
	}
	return core.NewDish([]byte(strings.Join(kept, delim)), core.TypeString), nil
}

// byteRangeArgs are the shared Start/Length/Apply-to-each-line definitions.
func byteRangeArgs() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Start", Type: core.ArgNumber, Integer: true, Value: 0},
		{Name: "Length", Type: core.ArgNumber, Integer: true, Value: 5},
		{Name: "Apply to each line", Type: core.ArgBoolean, Value: false},
	}
}

// adjustRange normalises a (start, length) pair against a buffer of size n,
// handling negative values exactly as CyberChef does.
func adjustRange(n, start, length int) (int, int) {
	if start < 0 {
		start = n + start
	}
	if length < 0 {
		start += length
		if start < 0 {
			start = n + start
			length = start - length
		} else {
			length = -length
		}
	}
	return start, length
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// takeBytes returns the [start, start+length) slice of data, with negatives handled.
func takeBytes(data []byte, start, length int) []byte {
	start, length = adjustRange(len(data), start, length)
	s := clampInt(start, 0, len(data))
	e := clampInt(start+length, s, len(data))
	return append([]byte(nil), data[s:e]...)
}

// dropBytes removes the [start, start+length) slice of data.
func dropBytes(data []byte, start, length int) []byte {
	start, length = adjustRange(len(data), start, length)
	s := clampInt(start, 0, len(data))
	e := clampInt(start+length, s, len(data))
	out := append([]byte(nil), data[:s]...)
	return append(out, data[e:]...)
}

// perLine applies fn to each LF-separated line and rejoins with LF.
func perLine(data []byte, fn func([]byte) []byte) []byte {
	lines := strings.Split(string(data), "\n")
	for i, l := range lines {
		lines[i] = string(fn([]byte(l)))
	}
	return []byte(strings.Join(lines, "\n"))
}

// DropBytes removes a range of bytes from the input.
type DropBytes struct{}

// Meta returns the operation metadata.
func (DropBytes) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Drop bytes",
		Module:      "Default",
		Description: "Deletes a range of bytes from the input, optionally per line.",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeArrayBuffer,
	}
}

// Args returns the argument definitions.
func (DropBytes) Args() []core.ArgDef { return byteRangeArgs() }

// Run drops the byte range. Ported from CyberChef DropBytes.mjs.
func (DropBytes) Run(in *core.Dish, args []any) (*core.Dish, error) {
	start, length := int(args[0].(float64)), int(args[1].(float64))
	fn := func(b []byte) []byte { return dropBytes(b, start, length) }
	var out []byte
	if args[2].(bool) {
		out = perLine(in.Bytes(), fn)
	} else {
		out = fn(in.Bytes())
	}
	return core.NewDish(out, core.TypeArrayBuffer), nil
}

// TakeBytes keeps only a range of bytes from the input.
type TakeBytes struct{}

// Meta returns the operation metadata.
func (TakeBytes) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Take bytes",
		Module:      "Default",
		Description: "Keeps only a range of bytes from the input, optionally per line.",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeArrayBuffer,
	}
}

// Args returns the argument definitions.
func (TakeBytes) Args() []core.ArgDef { return byteRangeArgs() }

// Run takes the byte range. Ported from CyberChef TakeBytes.mjs.
func (TakeBytes) Run(in *core.Dish, args []any) (*core.Dish, error) {
	start, length := int(args[0].(float64)), int(args[1].(float64))
	fn := func(b []byte) []byte { return takeBytes(b, start, length) }
	var out []byte
	if args[2].(bool) {
		out = perLine(in.Bytes(), fn)
	} else {
		out = fn(in.Bytes())
	}
	return core.NewDish(out, core.TypeArrayBuffer), nil
}

// nthArgs are the shared Every/Starting-at/Apply-to-each-line definitions.
func nthArgs(everyName string) []core.ArgDef {
	return []core.ArgDef{
		{Name: everyName, Type: core.ArgNumber, Value: 4},
		{Name: "Starting at", Type: core.ArgNumber, Integer: true, Value: 0},
		{Name: "Apply to each line", Type: core.ArgBoolean, Value: false},
	}
}

// DropNthBytes drops every nth byte.
type DropNthBytes struct{}

// Meta returns the operation metadata.
func (DropNthBytes) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Drop nth bytes",
		Module:      "Default",
		Description: "Drops every nth byte, starting at a given offset, optionally resetting per line.",
		InputType:   core.TypeByteArray,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (DropNthBytes) Args() []core.ArgDef { return nthArgs("Drop every") }

// Run drops every nth byte. Ported from CyberChef DropNthBytes.mjs.
func (DropNthBytes) Run(in *core.Dish, args []any) (*core.Dish, error) {
	n, start := int(args[0].(float64)), int(args[1].(float64))
	if n <= 0 {
		return nil, fmt.Errorf("'Drop every' must be a positive integer")
	}
	if start < 0 {
		return nil, fmt.Errorf("'Starting at' must be a positive or zero integer")
	}
	eachLine := args[2].(bool)

	data := in.Bytes()
	var out []byte
	offset := 0
	for i := range data {
		switch {
		case eachLine && data[i] == 0x0a:
			out = append(out, 0x0a)
			offset = i + 1
		case i-offset < start || (i-(start+offset))%n != 0:
			out = append(out, data[i])
		}
	}
	return core.NewDish(out, core.TypeByteArray), nil
}

// TakeNthBytes keeps every nth byte.
type TakeNthBytes struct{}

// Meta returns the operation metadata.
func (TakeNthBytes) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Take nth bytes",
		Module:      "Default",
		Description: "Keeps every nth byte, starting at a given offset, optionally resetting per line.",
		InputType:   core.TypeByteArray,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (TakeNthBytes) Args() []core.ArgDef { return nthArgs("Take every") }

// Run keeps every nth byte. Ported from CyberChef TakeNthBytes.mjs.
func (TakeNthBytes) Run(in *core.Dish, args []any) (*core.Dish, error) {
	n, start := int(args[0].(float64)), int(args[1].(float64))
	if n <= 0 {
		return nil, fmt.Errorf("'Take every' must be a positive integer")
	}
	if start < 0 {
		return nil, fmt.Errorf("'Starting at' must be a positive or zero integer")
	}
	eachLine := args[2].(bool)

	data := in.Bytes()
	var out []byte
	offset := 0
	for i := range data {
		switch {
		case eachLine && data[i] == 0x0a:
			out = append(out, 0x0a)
			offset = i + 1
		case i-offset >= start && (i-(start+offset))%n == 0:
			out = append(out, data[i])
		}
	}
	return core.NewDish(out, core.TypeByteArray), nil
}
