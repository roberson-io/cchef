package ops

import (
	"bytes"
	"errors"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(ToCOBS{})
	core.Register(FromCOBS{})
}

// cobsMaxRun is the longest stretch a single group may cover. A group's first
// byte counts the bytes to the next zero, so 0xFF is reserved to mean "254
// bytes and no zero after them".
const cobsMaxRun = 0xFF

// toCOBS removes every zero byte from the data by replacing it with a count of
// how far away the next one is. Ported from CyberChef lib/COBS.mjs.
func toCOBS(data []byte) []byte {
	if len(data) == 0 {
		return nil
	}

	var out []byte
	// A leading zero is prepended so that every group begins the same way; it
	// is never emitted, only counted from.
	work := append([]byte{0}, data...)

	for len(work) > 0 {
		end := bytes.IndexByte(work[1:], 0)
		if end >= 0 {
			end++ // the index within work, not within the tail
		}
		switch {
		case (end < 0 || end > 254) && len(work) > 254:
			out = append(out, cobsMaxRun)
			out = append(out, work[1:255]...)
			work = work[255:]
			if len(work) != 0 {
				work = append([]byte{0}, work...)
			}
		case end < 0:
			// The first case above takes every stretch longer than 254, so what
			// is left here always fits in the length byte.
			out = append(out, byte(len(work))) // #nosec G115 -- bounded to 254 by the case above
			out = append(out, work[1:]...)
			work = nil
		default:
			// Likewise, a zero further than 254 away is handled above.
			out = append(out, byte(end)) // #nosec G115 -- bounded to 254 by the first case
			out = append(out, work[1:end]...)
			work = work[end:]
		}
	}
	return out
}

// fromCOBS puts the zero bytes back. Ported from CyberChef lib/COBS.mjs.
func fromCOBS(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, nil
	}
	if bytes.IndexByte(data, 0) >= 0 {
		return nil, errors.New("Could not decode from COBS: payload must not contain a 0x00 byte") //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}

	var out []byte
	for len(data) > 0 {
		if data[0] == cobsMaxRun {
			out = append(out, cobsGroup(data, 255)...)
			data = cobsDrop(data, 255)
			continue
		}
		next := int(data[0])
		out = append(out, cobsGroup(data, next)...)
		data = cobsDrop(data, next)

		for len(data) > 0 {
			group := int(data[0])
			out = append(out, 0)
			out = append(out, cobsGroup(data, group)...)
			data = cobsDrop(data, group)
			if group == cobsMaxRun {
				break
			}
		}
	}
	return out, nil
}

// cobsGroup is the body of the group at the front of data — everything after
// the length byte, up to the length it gives. A length pointing past the end
// shortens the group rather than failing. Both callers hold at least the length
// byte, and a length of zero is refused before decoding starts, so the slice is
// always well formed.
func cobsGroup(data []byte, to int) []byte {
	if to > len(data) {
		to = len(data)
	}
	return data[1:to]
}

// cobsDrop is data[n:], or nothing at all when n runs past the end.
func cobsDrop(data []byte, n int) []byte {
	if n >= len(data) {
		return nil
	}
	return data[n:]
}

// ToCOBS encodes bytes so that none of them is zero. Ported from CyberChef
// ToCOBS.mjs.
type ToCOBS struct{}

// Meta returns the operation metadata.
func (ToCOBS) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "To COBS",
		Module:      "Default",
		Description: "Encodes bytes in COBS format",
		InfoURL:     "https://wikipedia.org/wiki/Consistent_Overhead_Byte_Stuffing",
		InputType:   core.TypeByteArray,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns no arguments; the operation takes none.
func (ToCOBS) Args() []core.ArgDef { return nil }

// Run encodes the input.
func (ToCOBS) Run(in *core.Dish, args []any) (*core.Dish, error) {
	return core.NewDish(toCOBS(in.Bytes()), core.TypeByteArray), nil
}

// FromCOBS puts back the zero bytes COBS removed. Ported from CyberChef
// FromCOBS.mjs.
type FromCOBS struct{}

// Meta returns the operation metadata.
func (FromCOBS) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "From COBS",
		Module:      "Default",
		Description: "Decodes COBS encoded bytes",
		InfoURL:     "https://wikipedia.org/wiki/Consistent_Overhead_Byte_Stuffing",
		InputType:   core.TypeByteArray,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns no arguments; the operation takes none.
func (FromCOBS) Args() []core.ArgDef { return nil }

// Run decodes the input.
func (FromCOBS) Run(in *core.Dish, args []any) (*core.Dish, error) {
	out, err := fromCOBS(in.Bytes())
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeByteArray), nil
}
