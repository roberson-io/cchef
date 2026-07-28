package ops

import (
	"bytes"
	"compress/zlib"
	"errors"
	"hash/adler32"
	"io"

	"github.com/roberson-io/cchef/internal/core"
)

// The zlib container: a raw DEFLATE stream behind two header bytes and an
// Adler-32 checksum of what went in.

// zlibLevelBits records, per block encoding, the guess at how hard the writer
// tried that goes into the header. It tells a reader nothing it needs, and
// nothing checks it.
var zlibLevelBits = map[string]byte{
	"None (Store)":           0,
	"Fixed Huffman Coding":   1,
	"Dynamic Huffman Coding": 2,
}

// zlibHeader returns the two bytes opening a zlib stream: the compression
// method and window size, then the level guess and a check value chosen to make
// the pair a multiple of 31. Only an encoding the writer has already accepted
// reaches here.
func zlibHeader(compressionType string) []byte {
	// Deflate, with the 32K window the writer always uses.
	const cmf = 0x78
	flg := zlibLevelBits[compressionType] << 6
	flg |= byte(31 - (int(cmf)*256+int(flg))%31)
	return []byte{cmf, flg}
}

// zlibEncode wraps a raw DEFLATE stream as a zlib one.
func zlibEncode(data []byte, compressionType string) ([]byte, error) {
	body, err := deflateEncode(data, compressionType)
	if err != nil {
		return nil, err
	}
	header := zlibHeader(compressionType)

	sum := adler32.Checksum(data)
	out := make([]byte, 0, len(header)+len(body)+4)
	out = append(out, header...)
	out = append(out, body...)
	// #nosec G115 -- the checksum is written a byte at a time, most significant first
	return append(out, byte(sum>>24), byte(sum>>16), byte(sum>>8), byte(sum)), nil
}

// zlibDecode reads a zlib stream, starting at the given byte.
func zlibDecode(data []byte, startIndex int) ([]byte, error) {
	if startIndex < 0 || startIndex > len(data) {
		return nil, errors.New("start index is outside the input")
	}
	if len(data) == 0 {
		//nolint:staticcheck,revive // CyberChef's verbatim error text
		return nil, errors.New("Please provide an input.")
	}
	r, err := zlib.NewReader(bytes.NewReader(data[startIndex:]))
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()
	return io.ReadAll(r)
}

// ZlibDeflate compresses the input into a zlib stream.
type ZlibDeflate struct{}

// Meta returns the operation metadata.
func (ZlibDeflate) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Zlib Deflate",
		Module:      "Compression",
		Description: "Compresses data using the zlib deflate algorithm.",
		InfoURL:     "https://wikipedia.org/wiki/Zlib",
		InputType:   core.TypeByteArray,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (ZlibDeflate) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Compression type", Type: core.ArgOption, Value: deflateCompressionTypes},
	}
}

// Run compresses the input.
func (ZlibDeflate) Run(in *core.Dish, args []any) (*core.Dish, error) {
	compressionType, _ := args[0].(string)
	out, err := zlibEncode(in.Bytes(), compressionType)
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeByteArray), nil
}

func init() { core.Register(ZlibDeflate{}) }

// ZlibInflate reads a zlib stream back into the bytes it was made from.
type ZlibInflate struct{}

// Meta returns the operation metadata.
func (ZlibInflate) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Zlib Inflate",
		Module:      "Compression",
		Description: "Decompresses data which has been compressed using the zlib deflate algorithm.",
		InfoURL:     "https://wikipedia.org/wiki/Zlib",
		InputType:   core.TypeByteArray,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (ZlibInflate) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Start index", Type: core.ArgNumber, Value: float64(0)},
		{Name: "Initial output buffer size", Type: core.ArgNumber, Value: float64(0)},
		{Name: "Buffer expansion type", Type: core.ArgOption, Value: deflateBufferTypes},
		{Name: "Resize buffer after decompression", Type: core.ArgBoolean, Value: false},
		{Name: "Verify result", Type: core.ArgBoolean, Value: false},
	}
}

// Run decompresses the input. Only the start index has any bearing on the
// result; the other three size and grow the reader's working buffer.
func (ZlibInflate) Run(in *core.Dish, args []any) (*core.Dish, error) {
	startIndex, _ := args[0].(float64)
	out, err := zlibDecode(in.Bytes(), int(startIndex))
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeByteArray), nil
}

func init() { core.Register(ZlibInflate{}) }
