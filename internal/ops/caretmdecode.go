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

// Run decodes the input. Ported from CaretMdecode.mjs. Each input byte is
// treated as a character code (0-255), matching CyberChef's charCodeAt over its
// Latin-1 string view of the bytes.
func (CaretMDecode) Run(in *core.Dish, args []any) (*core.Dish, error) {
	data := in.Bytes()
	out := make([]byte, 0, len(data))
	prev := cmNone

	for _, c := range data {
		switch prev {
		case cmMCar:
			switch {
			case c > 63 && c <= 95:
				out = append(out, c+64)
			case c == 63:
				out = append(out, 255)
			default:
				out = append(out, 77, 45, 94, c)
			}
			prev = cmNone
		case cmMDsh:
			switch {
			case c == '^':
				prev = cmMCar
			case c >= 32 && c <= 126:
				out = append(out, c+128)
				prev = cmNone
			default:
				out = append(out, 77, 45, c)
				prev = cmNone
			}
		case cmM:
			if c == '-' {
				prev = cmMDsh
			} else {
				out = append(out, 77, c)
				prev = cmNone
			}
		case cmCar:
			switch {
			case c > 63 && c <= 126:
				out = append(out, c-64)
			case c == 63:
				out = append(out, 127)
			default:
				out = append(out, 94, c)
			}
			prev = cmNone
		default: // cmNone
			switch c {
			case 'M':
				prev = cmM
			case '^':
				prev = cmCar
			default:
				out = append(out, c)
			}
		}
	}

	return core.NewDish(out, core.TypeByteArray), nil
}
