package ops

import (
	"bytes"
	"errors"
	"io"
	"strconv"

	"github.com/ulikunitz/xz/lzma"

	"github.com/roberson-io/cchef/core"
)

// LZMA compression and decompression.
//
// Decompression has one right answer, so it is exact. Compression is not: the
// stream depends on how the encoder looks for repeats and how it weighs one
// encoding against another, and cchef uses a different encoder from CyberChef.
// Both write valid LZMA, and each reads the other's; the bytes differ.

// lzmaHeaderSize is how many bytes open an LZMA stream: one of coding
// properties, four of window size, and eight of length.
const lzmaHeaderSize = 13

// lzmaProperties is the packed lc/lp/pb the format always uses here: three
// literal context bits, no literal position bits, two position bits.
const lzmaProperties = 0x5d

// lzmaModes are the compression modes the operation offers.
var lzmaModes = []string{"1", "2", "3", "4", "5", "6", "7", "8", "9"}

// lzmaWindowBits gives, per mode, how many bits of window the encoder gets —
// the one setting that reaches the stream, where it is recorded so a reader
// knows how much to keep. These are the sizes CyberChef's own modes ask for.
var lzmaWindowBits = [9]uint{16, 20, 19, 20, 21, 22, 23, 24, 25}

// LZMACompress compresses the input into an LZMA stream.
type LZMACompress struct{}

// Meta returns the operation metadata.
func (LZMACompress) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "LZMA Compress",
		Module:      "Compression",
		Description: "Compresses data using the Lempel–Ziv–Markov chain Algorithm.",
		InfoURL:     "https://wikipedia.org/wiki/Lempel-Ziv-Markov_chain_algorithm",
		InputType:   core.TypeByteArray,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (LZMACompress) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Compression Mode", Type: core.ArgOption, Value: lzmaModes, DefaultIndex: 6},
	}
}

// Run compresses the input.
func (LZMACompress) Run(in *core.Dish, args []any) (*core.Dish, error) {
	mode, _ := args[0].(string)
	out, err := lzmaEncode(in.Bytes(), mode)
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeByteArray), nil
}

// lzmaEncode compresses data, giving the encoder the window the mode asks for.
//
// The length is deliberately left unknown in the header, with an end marker
// closing the stream instead. That is the form the lzma command writes and
// every reader takes, and it steers clear of a known fault in the encoder
// underneath: asked to state a length of zero, it writes a header saying the
// length is unknown and then omits the end marker, leaving a stream nothing can
// read (ulikunitz/xz issue 71).
func lzmaEncode(data []byte, mode string) ([]byte, error) {
	windowBits, err := lzmaWindow(mode)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	cfg := lzma.WriterConfig{DictCap: 1 << windowBits}
	// The window comes from the table above, so the settings are always ones the
	// writer accepts; the error is returned rather than checked because there is
	// no second thing to do about it.
	err = lzmaEncodeTo(&buf, data, cfg)
	return buf.Bytes(), err
}

// lzmaEncodeTo writes one compressed stream, taking its destination and
// settings from the caller so that both can be driven directly by a test.
func lzmaEncodeTo(dst io.Writer, data []byte, cfg lzma.WriterConfig) error {
	w, err := cfg.NewWriter(dst)
	if err != nil {
		return err
	}
	if _, err := w.Write(data); err != nil {
		return err
	}
	return w.Close()
}

// lzmaWindow reads a mode as the window size it stands for.
func lzmaWindow(mode string) (uint, error) {
	n, err := strconv.Atoi(mode)
	if err != nil || n < 1 || n > len(lzmaWindowBits) {
		return 0, errors.New("compression mode must be 1 to 9")
	}
	return lzmaWindowBits[n-1], nil
}

// LZMADecompress reads an LZMA stream back into the bytes it was made from.
type LZMADecompress struct{}

// Meta returns the operation metadata.
func (LZMADecompress) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "LZMA Decompress",
		Module:      "Compression",
		Description: "Decompresses data using the Lempel–Ziv–Markov chain Algorithm.",
		InfoURL:     "https://wikipedia.org/wiki/Lempel-Ziv-Markov_chain_algorithm",
		InputType:   core.TypeByteArray,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (LZMADecompress) Args() []core.ArgDef { return nil }

// Run decompresses the input.
func (LZMADecompress) Run(in *core.Dish, args []any) (*core.Dish, error) {
	out, err := lzmaDecode(in.Bytes())
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeByteArray), nil
}

// lzmaDecode reads a stream, whether its header states the length outright or
// leaves it unknown and closes with an end marker.
func lzmaDecode(data []byte) ([]byte, error) {
	if len(data) == 0 {
		//nolint:staticcheck,revive // CyberChef's verbatim error text
		return nil, errors.New("Please provide an input.")
	}
	r, err := lzma.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return io.ReadAll(r)
}

func init() {
	core.Register(LZMACompress{})
	core.Register(LZMADecompress{})
}
