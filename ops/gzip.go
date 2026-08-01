package ops

import (
	"bytes"
	"compress/gzip"
	"errors"
	"hash/crc32"
	"io"
	"time"
	"unicode/utf16"

	"github.com/roberson-io/cchef/core"
)

// The gzip container: a raw DEFLATE stream behind a header that can carry a
// filename and a comment, and a trailer holding a checksum and the original
// length.

// The header flags saying which optional fields are present.
const (
	gzFlagFHCRC    = 1 << 1
	gzFlagFNAME    = 1 << 3
	gzFlagFCOMMENT = 1 << 4
)

// gzUnknownOS is the value written where the header records which system wrote
// the file. gzip stamps a real one; this writer says nothing, as CyberChef does.
const gzUnknownOS = 0xff

// gzNow is the clock the header timestamp comes from, replaced in tests.
var gzNow = time.Now

// gzipEncode wraps a raw DEFLATE stream as a gzip one. The header records the
// time it was written, so the same input does not give the same bytes twice.
func gzipEncode(data []byte, compressionType, filename, comment string, checksum bool) ([]byte, error) {
	var flags byte
	if filename != "" {
		flags |= gzFlagFNAME
	}
	if comment != "" {
		flags |= gzFlagFCOMMENT
	}
	if checksum {
		flags |= gzFlagFHCRC
	}

	mtime := uint32(gzNow().Unix()) // #nosec G115 -- a wrapping clock is what gzip records
	out := []byte{
		0x1f, 0x8b, 0x08, flags,
		// #nosec G115 -- the timestamp is written a byte at a time, least significant first
		byte(mtime), byte(mtime >> 8), byte(mtime >> 16), byte(mtime >> 24),
		0x00, gzUnknownOS,
	}
	if filename != "" {
		out = append(append(out, gzText(filename)...), 0)
	}
	if comment != "" {
		out = append(append(out, gzText(comment)...), 0)
	}
	if checksum {
		// Only the low half of the header's checksum is kept.
		sum := uint16(crc32.ChecksumIEEE(out)) // #nosec G115 -- the narrowing is the format's
		// #nosec G115 -- the checksum is written a byte at a time, least significant first
		out = append(out, byte(sum), byte(sum>>8))
	}

	body, err := deflateEncode(data, compressionType)
	if err != nil {
		return nil, err
	}
	out = append(out, body...)

	sum := crc32.ChecksumIEEE(data)
	size := uint32(len(data)) // #nosec G115 -- the length is recorded modulo 2^32
	return append(out,
		// #nosec G115 -- the checksum is written a byte at a time, least significant first
		byte(sum), byte(sum>>8), byte(sum>>16), byte(sum>>24),
		// #nosec G115 -- the length is written a byte at a time, least significant first
		byte(size), byte(size>>8), byte(size>>16), byte(size>>24)), nil
}

// gzText renders a filename or comment for the header. The field holds bytes,
// and a character needing more than one is written most significant byte first
// — which is what CyberChef does by walking the text a UTF-16 unit at a time.
func gzText(s string) []byte {
	var out []byte
	for _, unit := range utf16.Encode([]rune(s)) {
		if unit > 0xff {
			out = append(out, byte(unit>>8))
		}
		// #nosec G115 -- the low byte of the character is what the field holds
		out = append(out, byte(unit))
	}
	return out
}

// gzipDecode reads a gzip stream, including one holding several members
// written one after another.
func gzipDecode(data []byte) ([]byte, error) {
	if len(data) == 0 {
		//nolint:staticcheck,revive // CyberChef's verbatim error text
		return nil, errors.New("Please provide an input.")
	}
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()
	return io.ReadAll(r)
}

// Gzip compresses the input into a gzip stream.
type Gzip struct{}

// Meta returns the operation metadata.
func (Gzip) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Gzip",
		Module:      "Compression",
		Description: "Compresses data using the deflate algorithm with gzip headers.",
		InfoURL:     "https://wikipedia.org/wiki/Gzip",
		InputType:   core.TypeByteArray,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (Gzip) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Compression type", Type: core.ArgOption, Value: deflateCompressionTypes},
		{Name: "Filename (optional)", Type: core.ArgString, Value: ""},
		{Name: "Comment (optional)", Type: core.ArgString, Value: ""},
		{Name: "Include file checksum", Type: core.ArgBoolean, Value: false},
	}
}

// Run compresses the input.
func (Gzip) Run(in *core.Dish, args []any) (*core.Dish, error) {
	compressionType, _ := args[0].(string)
	filename, _ := args[1].(string)
	comment, _ := args[2].(string)
	checksum, _ := args[3].(bool)

	out, err := gzipEncode(in.Bytes(), compressionType, filename, comment, checksum)
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeByteArray), nil
}

func init() { core.Register(Gzip{}) }

// Gunzip reads a gzip stream back into the bytes it was made from.
type Gunzip struct{}

// Meta returns the operation metadata.
func (Gunzip) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Gunzip",
		Module:      "Compression",
		Description: "Decompresses data which has been compressed using the deflate algorithm with gzip headers.",
		InfoURL:     "https://wikipedia.org/wiki/Gzip",
		InputType:   core.TypeByteArray,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (Gunzip) Args() []core.ArgDef {
	return []core.ArgDef{}
}

// Run decompresses the input.
func (Gunzip) Run(in *core.Dish, args []any) (*core.Dish, error) {
	out, err := gzipDecode(in.Bytes())
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeByteArray), nil
}

func init() { core.Register(Gunzip{}) }
