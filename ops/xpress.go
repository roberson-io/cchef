package ops

import (
	"encoding/binary"
	"errors"

	"github.com/roberson-io/cchef/core"
)

// XPRESS decompression, in the two forms Windows uses: the plain LZ77 one
// behind RtlDecompressBuffer's COMPRESSION_FORMAT_XPRESS, and the LZ77+Huffman
// one WIM images and Windows Overlay Filter files are stored in. Both are
// specified in MS-XCA sections 2.1 and 2.2. Only decompression is implemented,
// which is all CyberChef offers.

const (
	// xpressMaxDecompressed bounds what one call will produce. Windows sizes
	// an XPRESS block at up to 32 MiB for a WIM chunk, so nothing legitimate
	// reaches it, and a corrupt length cannot ask for an unbounded run.
	xpressMaxDecompressed = 32 << 20

	// xpressMinMatch is the shortest run a match may stand for, and the amount
	// every escaped raw length is counted from.
	xpressMinMatch = 3
	// xpressRawEscape is the raw-length byte that defers to an LE16, which in
	// turn defers to an LE32 when it is zero.
	xpressRawEscape = 0xff
	// xpressMinRawLength is the smallest value the plain form's escaped raw
	// length may carry: a shorter run has a shorter spelling, so anything
	// below this means the stream is not what it claims to be.
	xpressMinRawLength = 22
	// xpressPlainRawBase and xpressHuffmanRawBase are what an unescaped raw
	// length is counted from in each form, the shorter spellings having
	// already covered the lengths below.
	xpressPlainRawBase   = 25
	xpressHuffmanRawBase = 18
	// xpressNibbleBase is what a shared length nibble is counted from, and
	// xpressNibbleEscape is the nibble that defers to a raw length instead.
	xpressNibbleBase   = 10
	xpressNibbleEscape = 15
)

// The messages are CyberChef's, verbatim.
//
//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
var (
	errXPRESSFlags       = errors.New("XPRESS: truncated flag group")
	errXPRESSLiteral     = errors.New("XPRESS: truncated literal")
	errXPRESSMatch       = errors.New("XPRESS: truncated match")
	errXPRESSNibble      = errors.New("XPRESS: truncated shared nibble")
	errXPRESSRawLength   = errors.New("XPRESS: truncated raw length")
	errXPRESSMatchLength = errors.New("XPRESS: invalid match length")
	errXPRESSOffset      = errors.New("XPRESS: match offset out of range")
	errXPRESSRatio       = errors.New("XPRESS: decompression ratio too large")
	errXPRESSSize        = errors.New("XPRESS: invalid decompressed size")
	errXPRESSTable       = errors.New("XPRESS: truncated Huffman table")
	errXPRESSCodeLengths = errors.New("XPRESS: invalid Huffman code lengths")
	errXPRESSBitStream   = errors.New("XPRESS: truncated bit stream")
	errXPRESSTooLong     = errors.New("XPRESS: output exceeds declared size")
	errXPRESSEndOfData   = errors.New("XPRESS: corrupt end-of-data marker")
)

// XPRESSDecompress reads a plain LZ77 stream back into the bytes it was made
// from.
type XPRESSDecompress struct{}

// Meta returns the operation metadata.
func (XPRESSDecompress) Meta() core.OpMeta {
	return core.OpMeta{
		Name:   "XPRESS Decompress",
		Module: "Compression",
		Description: "Decompresses data using the XPRESS plain LZ77 algorithm (MS-XCA section 2.1)." +
			"\n\nSimilar to the Windows API RtlDecompressBuffer with COMPRESSION_FORMAT_XPRESS.",
		InfoURL: "https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-xca/" +
			"5655f4a3-6ba4-489b-959f-e1f407c52f15",
		InputType:  core.TypeByteArray,
		OutputType: core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (XPRESSDecompress) Args() []core.ArgDef { return nil }

// Run decompresses the input.
func (XPRESSDecompress) Run(in *core.Dish, _ []any) (*core.Dish, error) {
	out, err := xpressDecode(in.Bytes())
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeByteArray), nil
}

// XPRESSHuffmanDecompress reads an LZ77+Huffman stream back into the bytes it
// was made from.
type XPRESSHuffmanDecompress struct{}

// Meta returns the operation metadata.
func (XPRESSHuffmanDecompress) Meta() core.OpMeta {
	return core.OpMeta{
		Name:   "XPRESS LZ77+Huffman Decompress",
		Module: "Compression",
		Description: "Decompresses data using the XPRESS LZ77+Huffman algorithm (MS-XCA section 2.2)." +
			"\n\nThe uncompressed size must be known in advance, as it is from the WOF chunk table " +
			"or WIM header, so it is taken as an argument.",
		InfoURL: "https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-xca/" +
			"5655f4a3-6ba4-489b-959f-e1f407c52f15",
		InputType:  core.TypeByteArray,
		OutputType: core.TypeByteArray,
	}
}

// Args returns the argument definitions. The size counts bytes, so a
// fractional value is a mistake rather than something to truncate.
func (XPRESSHuffmanDecompress) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Decompressed size", Type: core.ArgNumber, Value: 4096, Integer: true},
	}
}

// Run decompresses the input into the size given.
func (XPRESSHuffmanDecompress) Run(in *core.Dish, args []any) (*core.Dish, error) {
	size, _ := args[0].(float64)
	out, err := xpressDecodeHuffman(in.Bytes(), int(size))
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeByteArray), nil
}

// xpressDecode reads a plain LZ77 stream (MS-XCA section 2.1).
//
// The stream is a run of 32-bit flag groups, each read from bit 31 down. A
// clear bit is a literal byte; a set bit is a match described by an LE16 word
// holding the offset less one in its top 13 bits and the length less three in
// its low 3 bits. The last flag group is padded with set bits, so a match flag
// with no input behind it is where the data ends.
func xpressDecode(src []byte) ([]byte, error) {
	d := &xpressPlainDecoder{src: src, pending: -1}
	var out []byte
	var flags uint32
	flagsLeft := 0

	for {
		if flagsLeft == 0 {
			var err error
			if flags, err = d.flagGroup(); err != nil {
				return nil, err
			}
			flagsLeft = 32
		}
		flagsLeft--

		if flags>>flagsLeft&1 == 0 {
			b, err := d.literal()
			if err != nil {
				return nil, err
			}
			out = append(out, b)
			continue
		}

		// A match flag with no input left is the end of the data.
		if d.i >= len(d.src) {
			return out, nil
		}
		moff, mlen, err := d.match()
		if err != nil {
			return nil, err
		}
		if err = xpressCheckMatch(len(out), moff, mlen); err != nil {
			return nil, err
		}
		// #nosec G115 -- the cap check above bounds mlen at 32 MiB
		out = xpressCopyMatch(out, moff, int(mlen))
	}
}

// xpressPlainDecoder walks the byte stream of a plain LZ77 form.
type xpressPlainDecoder struct {
	src []byte
	i   int
	// pending is where the byte holding a half-used length nibble sits, or -1
	// when there is none. The two halves serve two separate matches, which
	// need not be adjacent: anything between them, literals included, leaves
	// the byte waiting.
	pending int
}

func (d *xpressPlainDecoder) flagGroup() (uint32, error) {
	if len(d.src)-d.i < 4 {
		return 0, errXPRESSFlags
	}
	flags := binary.LittleEndian.Uint32(d.src[d.i:])
	d.i += 4
	return flags, nil
}

func (d *xpressPlainDecoder) literal() (byte, error) {
	if d.i >= len(d.src) {
		return 0, errXPRESSLiteral
	}
	b := d.src[d.i]
	d.i++
	return b, nil
}

// match reads the word describing a match and returns how far back it reaches
// and how much it stands for.
func (d *xpressPlainDecoder) match() (moff int, mlen uint64, err error) {
	if len(d.src)-d.i < 2 {
		return 0, 0, errXPRESSMatch
	}
	mb := binary.LittleEndian.Uint16(d.src[d.i:])
	d.i += 2

	// The offset takes 13 bits, so it cannot reach past the 8192-byte window;
	// only the bound against what has been written needs testing.
	moff = int(mb>>3) + 1
	if short := mb & 7; short != 7 {
		return moff, uint64(short) + xpressMinMatch, nil
	}
	if mlen, err = d.longLength(); err != nil {
		return 0, 0, err
	}
	return moff, mlen, nil
}

// longLength reads the length of a match whose low three bits are all set. It
// comes from a nibble of a byte shared with one other such match, and a nibble
// of 15 defers in turn to a raw length.
func (d *xpressPlainDecoder) longLength() (uint64, error) {
	var nib byte
	if d.pending == -1 {
		if d.i >= len(d.src) {
			return 0, errXPRESSNibble
		}
		nib = d.src[d.i] & 0x0f
		d.pending = d.i
		d.i++
	} else {
		nib = d.src[d.pending] >> 4
		d.pending = -1
	}
	if nib != xpressNibbleEscape {
		return uint64(nib) + xpressNibbleBase, nil
	}

	v, escaped, err := xpressRawLength(d.src, &d.i)
	switch {
	case err != nil:
		return 0, err
	case !escaped:
		return v + xpressPlainRawBase, nil
	case v < xpressMinRawLength:
		return 0, errXPRESSMatchLength
	}
	return v + xpressMinMatch, nil
}

// xpressRawLength reads a length written as plain bytes rather than in the
// coded stream: one byte, an LE16 if that byte is 255, and an LE32 if that
// LE16 is zero. escaped reports whether the byte deferred, which decides what
// the value is counted from.
func xpressRawLength(src []byte, i *int) (v uint64, escaped bool, err error) {
	if *i >= len(src) {
		return 0, false, errXPRESSRawLength
	}
	b := src[*i]
	*i++
	if b != xpressRawEscape {
		return uint64(b), false, nil
	}

	if len(src)-*i < 2 {
		return 0, false, errXPRESSRawLength
	}
	word := binary.LittleEndian.Uint16(src[*i:])
	*i += 2
	if word != 0 {
		return uint64(word), true, nil
	}

	if len(src)-*i < 4 {
		return 0, false, errXPRESSRawLength
	}
	long := binary.LittleEndian.Uint32(src[*i:])
	*i += 4
	return uint64(long), true, nil
}

// xpressCheckMatch rejects a match reaching back further than the output
// holds, or one long enough to take the output past the cap.
func xpressCheckMatch(written, moff int, mlen uint64) error {
	if moff > written {
		return errXPRESSOffset
	}
	// #nosec G115 -- written is a slice length, so never negative
	if uint64(written)+mlen > xpressMaxDecompressed {
		return errXPRESSRatio
	}
	return nil
}

// xpressCopyMatch repeats mlen bytes from moff back. The copy runs a byte at a
// time because a match may overlap what it is writing, which is how a short
// offset stands for a long run.
func xpressCopyMatch(out []byte, moff, mlen int) []byte {
	for from := len(out) - moff; mlen > 0; mlen-- {
		out = append(out, out[from])
		from++
	}
	return out
}

const (
	// xpressSymbols is how many symbols the Huffman alphabet holds: 256
	// literals, one end-of-data marker, and 255 match descriptions.
	xpressSymbols = 512
	// xpressTableBytes is the size of the code-length table the stream opens
	// with, two 4-bit lengths to the byte.
	xpressTableBytes = xpressSymbols / 2
	// xpressEndOfData is the symbol that closes the stream, and it is also
	// what a match symbol is counted from.
	xpressEndOfData = 256

	// xpressDecodeBits is how many bits the decode table is indexed by, which
	// is the longest code the format allows, and xpressDecodeSize is how many
	// entries that comes to.
	xpressDecodeBits = 15
	xpressDecodeSize = 1 << xpressDecodeBits
	// xpressWordBits is the width of one input word, and xpressRegisterBits
	// the width of the register the words are read through.
	xpressWordBits     = 16
	xpressRegisterBits = 32
)

// xpressDecodeHuffman reads an LZ77+Huffman stream (MS-XCA section 2.2) into
// exactly size bytes. Unlike the plain form this one does not say where it
// ends, so the size has to be known already — from a WIM header or a Windows
// Overlay Filter chunk table.
//
// The stream opens with 256 bytes holding 512 four-bit code lengths, the even
// symbol of each pair in the low nibble. Codes are canonical, assigned in
// order of length and then symbol, and read most significant bit first from a
// stream of LE16 words. Symbols below 256 are literal bytes and 256 closes the
// stream; the rest describe a match, ((s-256)>>4) giving how many offset bits
// follow and ((s-256)&15) the length.
func xpressDecodeHuffman(src []byte, size int) ([]byte, error) {
	if size <= 0 || size > xpressMaxDecompressed {
		return nil, errXPRESSSize
	}
	d := &xpressHuffmanDecoder{src: src}
	if err := d.readTable(); err != nil {
		return nil, err
	}
	if err := d.refill(xpressRegisterBits); err != nil {
		return nil, err
	}

	out := make([]byte, 0, size)
	for {
		sym, err := d.symbol()
		if err != nil {
			return nil, err
		}

		switch {
		case sym < xpressEndOfData:
			if len(out)+1 > size {
				return nil, errXPRESSTooLong
			}
			// #nosec G115 -- the case above bounds sym below 256
			out = append(out, byte(sym))
			continue
		case sym == xpressEndOfData:
			// Only at the declared size does this symbol end the stream.
			// Anywhere else it is an ordinary match of three bytes at an
			// offset of one, which is how a run of one repeated byte is
			// spelled.
			if len(out) == size {
				return out, nil
			}
			if len(out) == 0 || size-len(out) < xpressMinMatch {
				return nil, errXPRESSEndOfData
			}
			out = xpressCopyMatch(out, 1, xpressMinMatch)
			continue
		}

		moff, mlen, err := d.match(sym)
		if err != nil {
			return nil, err
		}
		if err = xpressCheckMatch(len(out), moff, mlen); err != nil {
			return nil, err
		}
		if uint64(len(out))+mlen > uint64(size) {
			return nil, errXPRESSTooLong
		}
		// #nosec G115 -- the checks above bound mlen at the declared size
		out = xpressCopyMatch(out, moff, int(mlen))
	}
}

// xpressHuffmanDecoder reads symbols from the coded stream. Bits are held at
// the top of a 32-bit register that words are shifted into as room appears;
// nbits says how many of them are still unread.
type xpressHuffmanDecoder struct {
	src   []byte
	i     int
	bits  uint32
	nbits int
	lens  [xpressSymbols]byte
	table []uint16
}

// readTable reads the code lengths and expands them into a table mapping the
// next 15 bits straight to a symbol. Codes are canonical, so assigning them in
// order of length and then symbol reproduces the encoder's assignment, and
// each code claims a block of entries proportional to how short it is. The
// blocks have to fill the table exactly; anything else means the lengths do
// not describe a code.
func (d *xpressHuffmanDecoder) readTable() error {
	if len(d.src) < xpressTableBytes {
		return errXPRESSTable
	}
	for l := range xpressTableBytes {
		d.lens[l*2] = d.src[l] & 0x0f
		d.lens[l*2+1] = d.src[l] >> 4
	}
	d.i = xpressTableBytes

	d.table = make([]uint16, xpressDecodeSize)
	e := 0
	for l := byte(1); l <= xpressDecodeBits; l++ {
		for s := range xpressSymbols {
			if d.lens[s] != l {
				continue
			}
			n := 1 << (xpressDecodeBits - l)
			// CyberChef writes past the end of its table, which a JavaScript
			// array simply grows to fit, and catches the overflow on the count
			// below. This table is a fixed size, so the same lengths have to be
			// refused as they are read.
			if e+n > xpressDecodeSize {
				return errXPRESSCodeLengths
			}
			for ; n > 0; n-- {
				d.table[e] = uint16(s)
				e++
			}
		}
	}
	if e != xpressDecodeSize {
		return errXPRESSCodeLengths
	}
	return nil
}

// refill shifts words in until at least need bits are unread. A word arrives
// at the top of what is still empty, so the bits stay in the order they were
// written.
func (d *xpressHuffmanDecoder) refill(need int) error {
	for d.nbits < need {
		if len(d.src)-d.i < 2 {
			return errXPRESSBitStream
		}
		word := binary.LittleEndian.Uint16(d.src[d.i:])
		d.i += 2
		d.bits |= uint32(word) << (xpressWordBits - d.nbits)
		d.nbits += xpressWordBits
	}
	return nil
}

// symbol reads the next code. The table is indexed by the longest code the
// format allows, so a shorter code simply appears many times over and only its
// own length is consumed.
func (d *xpressHuffmanDecoder) symbol() (int, error) {
	if err := d.refill(xpressDecodeBits); err != nil {
		return 0, err
	}
	sym := int(d.table[d.bits>>(xpressRegisterBits-xpressDecodeBits)])
	clen := d.lens[sym]
	d.bits <<= clen
	d.nbits -= int(clen)
	return sym, nil
}

// match reads what a match symbol stands for: how far back it reaches, from
// the offset bits that follow it, and how much it repeats.
func (d *xpressHuffmanDecoder) match(sym int) (moff int, mlen uint64, err error) {
	bits := (sym - xpressEndOfData) >> 4
	nib := (sym - xpressEndOfData) & 0x0f

	if nib != xpressNibbleEscape {
		mlen = uint64(nib) + xpressMinMatch
	} else {
		v, escaped, err := xpressRawLength(d.src, &d.i)
		if err != nil {
			return 0, 0, err
		}
		if escaped {
			mlen = v + xpressMinMatch
		} else {
			mlen = v + xpressHuffmanRawBase
		}
	}

	if err = d.refill(bits); err != nil {
		return 0, 0, err
	}
	if bits > 0 {
		moff = int(d.bits >> (xpressRegisterBits - bits))
		d.bits <<= bits
		d.nbits -= bits
	}
	return moff + 1<<bits, mlen, nil
}

func init() {
	core.Register(XPRESSDecompress{})
	core.Register(XPRESSHuffmanDecompress{})
}
