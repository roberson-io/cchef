package ops

import (
	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(CaretMDecode{})
}

// CaretMDecode decodes caret-notation and M-notation escapes (as produced by
// tools such as `cat -v`).
type CaretMDecode struct{}

// Meta returns the operation metadata.
func (CaretMDecode) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Caret/M-decode",
		Module:      "Default",
		Description: "Decodes caret or M-encoded strings, i.e. ^M turns into a newline, M-^] turns into 0x9d. Sources such as `cat -v`.\n\nPlease be aware that when using `cat -v` ^_ (caret-underscore) will not be encoded, but represents a valid encoding (namely that of 0x1f).",
		InfoURL:     "https://en.wikipedia.org/wiki/Caret_notation",
		InputType:   core.TypeString,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (CaretMDecode) Args() []core.ArgDef {
	return nil
}

// Caret/M-decode parser states, tracking the escape prefix seen so far.
const (
	cmNone = iota
	cmM    // "M"
	cmMDsh // "M-"
	cmMCar // "M-^"
	cmCar  // "^"
)

// caretOffset is the control-code offset in caret notation (^X = X ^ 0x40);
// metaBit is the high bit set by the M- meta prefix.
const (
	caretOffset = 64
	metaBit     = 128
)

// Run decodes the input. Ported from CaretMdecode.mjs. Each input byte is
// treated as a character code (0-255), matching CyberChef's charCodeAt over its
// Latin-1 string view of the bytes.
func (CaretMDecode) Run(in *core.Dish, args []any) (*core.Dish, error) {
	data := in.Bytes()
	d := &cmDecoder{out: make([]byte, 0, len(data))}
	for _, c := range data {
		d.feed(c)
	}
	return core.NewDish(d.out, core.TypeByteArray), nil
}

// cmDecoder is the caret/M-notation state machine: it accumulates decoded bytes
// while tracking the escape prefix (prev) seen so far.
type cmDecoder struct {
	out  []byte
	prev int
}

// feed processes one input byte, dispatching on the current escape state.
func (d *cmDecoder) feed(c byte) {
	switch d.prev {
	case cmMCar:
		d.afterMCaret(c)
	case cmMDsh:
		d.afterMDash(c)
	case cmM:
		d.afterM(c)
	case cmCar:
		d.afterCaret(c)
	default: // cmNone
		d.atStart(c)
	}
}

// afterMCaret handles the byte after an "M-^" prefix.
func (d *cmDecoder) afterMCaret(c byte) {
	switch {
	case c > '?' && c <= '_':
		d.out = append(d.out, c+caretOffset)
	case c == '?':
		d.out = append(d.out, 255)
	default:
		d.out = append(d.out, 'M', '-', '^', c)
	}
	d.prev = cmNone
}

// afterMDash handles the byte after an "M-" prefix.
func (d *cmDecoder) afterMDash(c byte) {
	switch {
	case c == '^':
		d.prev = cmMCar
	case c >= ' ' && c <= '~':
		d.out = append(d.out, c+metaBit)
		d.prev = cmNone
	default:
		d.out = append(d.out, 'M', '-', c)
		d.prev = cmNone
	}
}

// afterM handles the byte after an "M" prefix.
func (d *cmDecoder) afterM(c byte) {
	if c == '-' {
		d.prev = cmMDsh
	} else {
		d.out = append(d.out, 'M', c)
		d.prev = cmNone
	}
}

// afterCaret handles the byte after a "^" prefix.
func (d *cmDecoder) afterCaret(c byte) {
	switch {
	case c > '?' && c <= '~':
		d.out = append(d.out, c-caretOffset)
	case c == '?':
		d.out = append(d.out, 127)
	default:
		d.out = append(d.out, '^', c)
	}
	d.prev = cmNone
}

// atStart handles a byte in the initial state, looking for an escape prefix.
func (d *cmDecoder) atStart(c byte) {
	switch c {
	case 'M':
		d.prev = cmM
	case '^':
		d.prev = cmCar
	default:
		d.out = append(d.out, c)
	}
}
