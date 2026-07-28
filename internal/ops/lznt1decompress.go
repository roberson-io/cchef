package ops

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/roberson-io/cchef/internal/core"
)

// LZNT1 decompression, the form NTFS compresses a file with and the Windows
// call RtlDecompressBuffer reads.
//
// A stream is a run of chunks, each holding up to four kilobytes of data.
// A chunk either keeps its bytes as they are or encodes them as flag groups:
// one byte of flags and then eight items, each either a literal byte or a
// reference back over what the chunk has produced so far. How a reference
// splits its sixteen bits between distance and length changes as the chunk
// fills, so that reaching further back costs a shorter run.

const (
	// lznt1CompressedChunk is the bit in a chunk header saying the chunk is
	// encoded rather than stored as it stands.
	lznt1CompressedChunk = 1 << 15
	// lznt1SizeMask is where a chunk header keeps its length, less one, and
	// also the widest a reference's length field can be.
	lznt1SizeMask = 1<<12 - 1
	// lznt1MinMatch is the shortest run a reference may stand for.
	lznt1MinMatch = 3
	// lznt1ItemsPerFlagByte is how many items one byte of flags accounts for.
	lznt1ItemsPerFlagByte = 8
	// lznt1PointerBits is how many bits a reference has to divide between the
	// distance and the length.
	lznt1PointerBits = 12
	// lznt1WindowStep is the point at which the distance claims another bit.
	lznt1WindowStep = 0x10
)

// LZNT1Decompress reads an LZNT1 stream back into the bytes it was made from.
type LZNT1Decompress struct{}

// Meta returns the operation metadata.
func (LZNT1Decompress) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "LZNT1 Decompress",
		Module:      "Compression",
		Description: "Decompresses data using the LZNT1 algorithm.\n\nSimilar to the Windows API RtlDecompressBuffer.",
		InfoURL:     "https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-xca/5655f4a3-6ba4-489b-959f-e1f407c52f15",
		InputType:   core.TypeByteArray,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (LZNT1Decompress) Args() []core.ArgDef { return nil }

// Run decompresses the input.
func (LZNT1Decompress) Run(in *core.Dish, _ []any) (*core.Dish, error) {
	out, err := lznt1Decode(in.Bytes())
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeByteArray), nil
}

// lznt1Decode reads every chunk in the stream.
func lznt1Decode(src []byte) ([]byte, error) {
	var out []byte
	var err error
	for pos := 0; pos+2 <= len(src); {
		header := binary.LittleEndian.Uint16(src[pos:])
		pos += 2
		// A header of nothing at all ends the stream. The size alone is not the
		// test: a chunk holding a single byte records a size of zero.
		if header == 0 {
			break
		}

		size := int(header&lznt1SizeMask) + 1
		if size > len(src)-pos {
			return nil, fmt.Errorf("the LZNT1 chunk claims %d bytes but %d are left",
				size, len(src)-pos)
		}
		body := src[pos : pos+size]
		pos += size

		if header&lznt1CompressedChunk == 0 {
			out = append(out, body...)
			continue
		}
		if out, err = lznt1DecodeChunk(body, out); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// lznt1DecodeChunk reads one encoded chunk, appending what it holds to out.
//
// A flag byte accounts for eight items, but a chunk may end partway through a
// group, which is why the count is bounded by the chunk as well.
func lznt1DecodeChunk(body, out []byte) ([]byte, error) {
	start := len(out)
	var err error
	for i := 0; i < len(body); {
		flags := body[i]
		i++
		for n := 0; n < lznt1ItemsPerFlagByte && i < len(body); n++ {
			if flags&1 == 0 {
				out = append(out, body[i])
				i++
			} else {
				if len(body)-i < 2 {
					return nil, errors.New("an LZNT1 reference is cut in half")
				}
				pointer := binary.LittleEndian.Uint16(body[i:])
				i += 2
				if out, err = lznt1Repeat(out, start, pointer); err != nil {
					return nil, err
				}
			}
			flags >>= 1
		}
	}
	return out, nil
}

// lznt1Repeat copies back over what the chunk has already written. The copy runs
// a byte at a time because a reference may overlap what it is writing, which is
// how a short repeat stands for a long run.
func lznt1Repeat(out []byte, start int, pointer uint16) ([]byte, error) {
	shift := lznt1Displacement(len(out) - start - 1)
	distance := int(pointer>>(lznt1PointerBits-shift)) + 1
	length := int(pointer&uint16(lznt1SizeMask>>shift)) + lznt1MinMatch

	from := len(out) - distance
	if from < start {
		return nil, fmt.Errorf("an LZNT1 reference reaches %d bytes back, past the start of its chunk",
			distance)
	}
	for ; length > 0; length-- {
		out = append(out, out[from])
		from++
	}
	return out, nil
}

// lznt1Displacement is how many bits of a reference go to the distance rather
// than the length, given how far into its chunk the reference sits. Every
// doubling of that position past sixteen moves one more bit across.
func lznt1Displacement(offset int) uint {
	shift := uint(0)
	for offset >= lznt1WindowStep {
		offset >>= 1
		shift++
	}
	return shift
}

func init() {
	core.Register(LZNT1Decompress{})
}
