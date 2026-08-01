package ops

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/bits"

	"github.com/roberson-io/cchef/core"
)

// LZ4 compression and decompression.
//
// A frame opens with a short descriptor and then carries blocks, each a run of
// literal bytes followed by a reference back to something already written. What
// an encoder chooses to reference is left open, so two encoders that both write
// valid LZ4 rarely write the same bytes; the search here is the one CyberChef
// uses, so the frames match byte for byte.
//
// The reader is more forgiving than the writer needs. It takes the frame
// options CyberChef never writes — a stated content size, checksums over each
// block and over the whole content, blocks of any permitted size, a dictionary
// identifier — and it reads a run of frames rather than stopping at the first.
// It also checks the checksums, which CyberChef ignores.

// Sizes and limits the compressed form is built around.
const (
	// lz4MinMatch is the shortest repeat a back reference may stand for, and so
	// the amount a stated match length is short by.
	lz4MinMatch = 4
	// lz4MinLength is the shortest block worth searching for repeats at all.
	lz4MinLength = 13
	// lz4SearchLimit is how many bytes at the end of a block the format keeps as
	// literals, so that a decoder may copy a match without watching the edge.
	lz4SearchLimit = 5
	// lz4SkipTrigger sets how quickly the search gives up on data that is not
	// repeating: after each miss it steps a little further ahead.
	lz4SkipTrigger = 6
	// lz4HashSize is how many positions the match table remembers.
	lz4HashSize = 1 << 16
	// lz4MaxOffset is the furthest back a match may reach.
	lz4MaxOffset = 1<<16 - 1

	// A sequence opens with one byte holding two counts: literals in the high
	// nibble, match length in the low one. Either at its maximum means the real
	// count follows the literals as a run of 255s and a remainder.
	lz4MatchBits = 4
	lz4MatchMask = 1<<lz4MatchBits - 1
	lz4RunBits   = 4
	lz4RunMask   = 1<<lz4RunBits - 1
)

// The frame around those blocks.
const (
	lz4Magic = 0x184D2204
	// lz4HeaderSize is the shortest header there is: the magic number, the two
	// descriptor bytes and their checksum.
	lz4HeaderSize = 7

	lz4FlagDictID          = 0x01
	lz4FlagContentChecksum = 0x04
	lz4FlagContentSize     = 0x08
	lz4FlagBlockChecksum   = 0x10
	lz4Version             = 0x40
	lz4VersionMask         = 0xC0

	// lz4StoredBlock is the bit in a block's length saying it was left as it
	// was, because compressing it would not have made it smaller.
	lz4StoredBlock = 0x80000000

	lz4BlockSizeShift = 4
	lz4BlockSizeMask  = 7
	// Only four of the eight block sizes are given a meaning.
	lz4MinBlockIndex = 4
	lz4MaxBlockIndex = 7
	// lz4DefaultBlockIndex is the four megabytes CyberChef always asks for.
	lz4DefaultBlockIndex = 7
)

// lz4BlockSizeFor turns a descriptor's block-size index into the largest block
// it allows: index 4 is 64 KB, and each step up multiplies that by four.
func lz4BlockSizeFor(index int) int { return 1 << (2*index + 8) }

// LZ4Compress compresses the input into an LZ4 frame.
type LZ4Compress struct{}

// Meta returns the operation metadata.
func (LZ4Compress) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "LZ4 Compress",
		Module:      "Compression",
		Description: "LZ4 is a lossless data compression algorithm that is focused on compression and decompression speed. It belongs to the LZ77 family of byte-oriented compression schemes.",
		InfoURL:     "https://wikipedia.org/wiki/LZ4_(compression_algorithm)",
		InputType:   core.TypeByteArray,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (LZ4Compress) Args() []core.ArgDef { return nil }

// Run compresses the input.
func (LZ4Compress) Run(in *core.Dish, _ []any) (*core.Dish, error) {
	return core.NewDish(lz4CompressFrame(in.Bytes()), core.TypeByteArray), nil
}

// lz4CompressBound is the largest frame an input of n bytes can turn into: every
// block stored as it was, each behind its length, inside the header and the end
// mark.
func lz4CompressBound(n int) int {
	blocks := n/lz4BlockSizeFor(lz4DefaultBlockIndex) + 1
	return lz4HeaderSize + n + 4*blocks + 4
}

// lz4CompressFrame writes one frame holding the whole input.
func lz4CompressFrame(src []byte) []byte {
	out := make([]byte, 0, lz4CompressBound(len(src)))
	out = binary.LittleEndian.AppendUint32(out, lz4Magic)
	out = append(out, lz4Version, lz4DefaultBlockIndex<<lz4BlockSizeShift)
	out = append(out, byte(lz4XXH32(0, out[4:])>>8)) // #nosec G115 -- the header keeps eight bits of the hash by design

	maxBlock := lz4BlockSizeFor(lz4DefaultBlockIndex)
	table := make([]uint32, lz4HashSize)
	var block []byte
	for pos := 0; pos < len(src); {
		n := min(len(src)-pos, maxBlock)
		block = lz4CompressBlock(src, pos, n, table, block[:0])
		if len(block) > n {
			// #nosec G115 -- a block is never larger than the four megabytes asked for
			out = binary.LittleEndian.AppendUint32(out, lz4StoredBlock|uint32(n))
			out = append(out, src[pos:pos+n]...)
		} else {
			// #nosec G115 -- a compressed block is smaller still
			out = binary.LittleEndian.AppendUint32(out, uint32(len(block)))
			out = append(out, block...)
		}
		pos += n
	}
	return binary.LittleEndian.AppendUint32(out, 0)
}

// lz4CompressBlock encodes length bytes of src from start, appending to dst.
//
// The match table is not cleared between blocks, so a match may reach back into
// the block before this one. That is why the descriptor says the blocks are
// linked rather than standing alone.
func lz4CompressBlock(src []byte, start, length int, table []uint32, dst []byte) []byte {
	end := start + length
	anchor := start
	if length >= lz4MinLength {
		anchor, dst = lz4FindMatches(src, start, end, table, dst)
	}
	return lz4AppendLiterals(dst, src[anchor:end], 0)
}

// lz4FindMatches walks the block writing a sequence for each repeat it finds,
// and returns where the literals it has not written yet begin.
//
// The table holds one position per hash and is never chained, so a hash that
// collides simply loses the older position. Any position it does keep is
// checked in full before it is used.
func lz4FindMatches(src []byte, start, end int, table []uint32, dst []byte) (int, []byte) {
	anchor := start
	// How far to step after a miss, in sixty-fourths: a run of misses widens
	// the stride, and a hit puts it back.
	stride := 1<<lz4SkipTrigger + 3
	for i := start; i+lz4MinMatch < end-lz4SearchLimit; {
		seq := binary.LittleEndian.Uint32(src[i:])
		hash := lz4HashU32(seq)
		hash = (hash>>16 ^ hash) & (lz4HashSize - 1)

		match := int(table[hash]) - 1
		table[hash] = uint32(i + 1) // #nosec G115 -- a position within the input, which is far short of four gigabytes

		if match < 0 || i-match > lz4MaxOffset || binary.LittleEndian.Uint32(src[match:]) != seq {
			i += stride >> lz4SkipTrigger
			stride++
			continue
		}
		stride = 1<<lz4SkipTrigger + 3

		literals := src[anchor:i]
		offset := i - match
		// The first four bytes are known to be equal already.
		i += lz4MinMatch
		match += lz4MinMatch
		from := i
		for i < end-lz4SearchLimit && src[i] == src[match] {
			i++
			match++
		}
		// The length written is what the match runs to beyond those four.
		length := i - from

		dst = lz4AppendLiterals(dst, literals, min(length, lz4MatchMask))
		// #nosec G115 -- a distance is checked against lz4MaxOffset before it is used
		dst = append(dst, byte(offset), byte(offset>>8))
		if length >= lz4MatchMask {
			dst = lz4AppendLength(dst, length-lz4MatchMask)
		}
		anchor = i
	}
	return anchor, dst
}

// lz4AppendLiterals writes a sequence token and the literals it counts, leaving
// the match length in the token's low nibble for the caller to follow.
func lz4AppendLiterals(dst, literals []byte, matchToken int) []byte {
	n := len(literals)
	if n >= lz4RunMask {
		dst = append(dst, byte(lz4RunMask<<lz4MatchBits|matchToken)) // #nosec G115 -- both halves are nibbles
		dst = lz4AppendLength(dst, n-lz4RunMask)
	} else {
		dst = append(dst, byte(n<<lz4MatchBits|matchToken)) // #nosec G115 -- both halves are nibbles
	}
	return append(dst, literals...)
}

// lz4AppendLength writes a count too large for its nibble, as a run of 255s and
// then what is left over.
func lz4AppendLength(dst []byte, n int) []byte {
	for ; n >= 0xff; n -= 0xff {
		dst = append(dst, 0xff)
	}
	return append(dst, byte(n)) // #nosec G115 -- the loop above leaves n below 255
}

// lz4HashU32 spreads four bytes of input across the match table. It is Thomas
// Wang's 32-bit integer mix, chosen for using no multiplication.
func lz4HashU32(a uint32) uint32 {
	a = a + 0x7ed55d16 + a<<12
	a = (a ^ 0xc761c23c) ^ a>>19
	a = a + 0x165667b1 + a<<5
	a = (a + 0xd3a2646c) ^ a<<9
	a = a + 0xfd7046c5 + a<<3
	return (a ^ 0xb55a4f09) ^ a>>16
}

// LZ4Decompress reads an LZ4 frame back into the bytes it was made from.
type LZ4Decompress struct{}

// Meta returns the operation metadata.
func (LZ4Decompress) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "LZ4 Decompress",
		Module:      "Compression",
		Description: "LZ4 is a lossless data compression algorithm that is focused on compression and decompression speed. It belongs to the LZ77 family of byte-oriented compression schemes.",
		InfoURL:     "https://wikipedia.org/wiki/LZ4_(compression_algorithm)",
		InputType:   core.TypeByteArray,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (LZ4Decompress) Args() []core.ArgDef { return nil }

// Run decompresses the input.
func (LZ4Decompress) Run(in *core.Dish, _ []any) (*core.Dish, error) {
	out, err := lz4DecompressFrames(in.Bytes())
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeByteArray), nil
}

// errLZ4Header is what a header that stops partway through gets.
var errLZ4Header = errors.New("the LZ4 frame header is cut short")

// lz4DecompressFrames reads every frame in the input, one after another, into a
// single run of bytes. Frames written one after the other are a normal way to
// hold LZ4 data and are read here as one.
func lz4DecompressFrames(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return nil, errors.New("there is no LZ4 frame to read")
	}
	r := &lz4Reader{src: src}
	var out []byte
	var err error
	for r.left() > 0 {
		if out, err = r.frame(out); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// lz4Reader walks a stream of frames.
type lz4Reader struct {
	src []byte
	pos int
}

// left is how much of the input has not been read.
func (r *lz4Reader) left() int { return len(r.src) - r.pos }

// take reads the next n bytes, reporting whether there were that many.
func (r *lz4Reader) take(n int) ([]byte, bool) {
	if n > r.left() {
		return nil, false
	}
	b := r.src[r.pos : r.pos+n]
	r.pos += n
	return b, true
}

// u32 reads a little-endian four-byte number.
func (r *lz4Reader) u32() (uint32, bool) {
	b, ok := r.take(4)
	if !ok {
		return 0, false
	}
	return binary.LittleEndian.Uint32(b), true
}

// lz4FrameHeader is what a frame's descriptor says about the blocks to come.
type lz4FrameHeader struct {
	blockChecksum   bool
	contentChecksum bool
	hasContentSize  bool
	contentSize     uint64
	maxBlockSize    int
}

// frame reads one whole frame, appending what it holds to out.
func (r *lz4Reader) frame(out []byte) ([]byte, error) {
	at := r.pos
	magic, ok := r.u32()
	if !ok || magic != lz4Magic {
		return nil, fmt.Errorf("there is no LZ4 frame at byte %d", at)
	}
	h, err := r.header()
	if err != nil {
		return nil, err
	}

	start := len(out)
	for {
		size, ok := r.u32()
		if !ok {
			return nil, errors.New("the LZ4 frame stops before its last block")
		}
		if size == 0 {
			break
		}
		if out, err = r.block(h, size, out, start); err != nil {
			return nil, err
		}
	}
	return r.finish(h, out, start)
}

// header reads the frame descriptor and checks it against its own checksum.
func (r *lz4Reader) header() (lz4FrameHeader, error) {
	var h lz4FrameHeader
	start := r.pos
	fields, ok := r.take(2)
	if !ok {
		return h, errLZ4Header
	}
	flags, blockDesc := fields[0], fields[1]

	if flags&lz4VersionMask != lz4Version {
		return h, fmt.Errorf("LZ4 frame version %d is not supported", flags&lz4VersionMask>>6)
	}
	index := int(blockDesc>>lz4BlockSizeShift) & lz4BlockSizeMask
	if index < lz4MinBlockIndex || index > lz4MaxBlockIndex {
		return h, fmt.Errorf("the LZ4 frame asks for block size %d, which has no meaning", index)
	}
	h.maxBlockSize = lz4BlockSizeFor(index)

	if flags&lz4FlagContentSize != 0 {
		size, ok := r.take(8)
		if !ok {
			return h, errLZ4Header
		}
		h.hasContentSize, h.contentSize = true, binary.LittleEndian.Uint64(size)
	}
	if flags&lz4FlagDictID != 0 {
		if _, ok := r.take(4); !ok {
			return h, errLZ4Header
		}
	}

	check, ok := r.take(1)
	if !ok {
		return h, errLZ4Header
	}
	// #nosec G115 -- the header keeps eight bits of the hash by design
	if want := byte(lz4XXH32(0, r.src[start:r.pos-1]) >> 8); check[0] != want {
		return h, errors.New("the LZ4 frame header does not match its own checksum")
	}

	h.blockChecksum = flags&lz4FlagBlockChecksum != 0
	h.contentChecksum = flags&lz4FlagContentChecksum != 0
	return h, nil
}

// block reads one block of the given length, appending what it holds to out.
// Matches may reach back into earlier blocks of the same frame but no further,
// which is what floor marks.
func (r *lz4Reader) block(h lz4FrameHeader, size uint32, out []byte, floor int) ([]byte, error) {
	stored := size&lz4StoredBlock != 0
	size &^= lz4StoredBlock
	// #nosec G115 -- a block size is one of four constants, the largest four megabytes
	if size > uint32(h.maxBlockSize) {
		return nil, fmt.Errorf("an LZ4 block claims %d bytes, more than the %d the frame allows",
			size, h.maxBlockSize)
	}
	body, ok := r.take(int(size))
	if !ok {
		return nil, errors.New("an LZ4 block runs past the end of the input")
	}
	if h.blockChecksum {
		if err := r.checksum(body, "block"); err != nil {
			return nil, err
		}
	}
	if stored {
		return append(out, body...), nil
	}
	return lz4DecodeBlock(body, out, floor)
}

// checksum reads a stored hash and checks the data against it.
func (r *lz4Reader) checksum(data []byte, what string) error {
	want, ok := r.u32()
	if !ok {
		return fmt.Errorf("the LZ4 %s checksum is missing", what)
	}
	if lz4XXH32(0, data) != want {
		return fmt.Errorf("the LZ4 %s checksum does not match the data", what)
	}
	return nil
}

// finish reads whatever follows a frame's last block and checks the frame
// against what its header promised.
func (r *lz4Reader) finish(h lz4FrameHeader, out []byte, start int) ([]byte, error) {
	content := out[start:]
	if h.contentChecksum {
		if err := r.checksum(content, "content"); err != nil {
			return nil, err
		}
	}
	if h.hasContentSize && h.contentSize != uint64(len(content)) {
		return nil, fmt.Errorf("the LZ4 frame says it holds %d bytes but gave %d",
			h.contentSize, len(content))
	}
	return out, nil
}

// lz4DecodeBlock reads a compressed block: literals to copy across, then a
// distance and a length saying what to repeat from what has been written.
func lz4DecodeBlock(src, out []byte, floor int) ([]byte, error) {
	var err error
	for i := 0; i < len(src); {
		token := src[i]
		i++

		var literals []byte
		if literals, i, err = lz4Literals(src, i, int(token>>lz4MatchBits)); err != nil {
			return nil, err
		}
		out = append(out, literals...)

		// A block ends on its literals, with no match to follow.
		if i >= len(src) {
			break
		}
		if out, i, err = lz4Match(src, i, token, out, floor); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// lz4Literals reads the literals a token counts, extending the count first if
// the nibble was full.
func lz4Literals(src []byte, i, count int) ([]byte, int, error) {
	if count == lz4RunMask {
		extra, next, err := lz4ReadLength(src, i)
		if err != nil {
			return nil, 0, err
		}
		count, i = count+extra, next
	}
	if count > len(src)-i {
		return nil, 0, errors.New("an LZ4 block promises more literals than it holds")
	}
	return src[i : i+count], i + count, nil
}

// lz4Match reads a back reference and copies what it points at. The copy runs a
// byte at a time because a match is allowed to overlap what it is writing,
// which is how a short repeat stands for a long run.
func lz4Match(src []byte, i int, token byte, out []byte, floor int) ([]byte, int, error) {
	if len(src)-i < 2 {
		return nil, 0, errors.New("an LZ4 block ends before a match distance")
	}
	offset := int(src[i]) | int(src[i+1])<<8
	i += 2

	length := int(token & lz4MatchMask)
	if length == lz4MatchMask {
		extra, next, err := lz4ReadLength(src, i)
		if err != nil {
			return nil, 0, err
		}
		length, i = length+extra, next
	}
	length += lz4MinMatch

	from := len(out) - offset
	if offset == 0 || from < floor {
		return nil, 0, fmt.Errorf("an LZ4 match reaches %d bytes back, past the start of the data", offset)
	}
	for ; length > 0; length-- {
		out = append(out, out[from])
		from++
	}
	return out, i, nil
}

// lz4ReadLength reads a count that did not fit in its nibble: 255s until a
// byte that is something else, all added together.
func lz4ReadLength(src []byte, i int) (int, int, error) {
	n := 0
	for {
		if i >= len(src) {
			return 0, 0, errors.New("a length in an LZ4 block runs past the end of it")
		}
		b := src[i]
		i++
		n += int(b)
		if b != 0xff {
			return n, i, nil
		}
	}
}

// The multipliers xxHash32 mixes with.
const (
	lz4Prime1 = 0x9e3779b1
	lz4Prime2 = 0x85ebca77
	lz4Prime3 = 0xc2b2ae3d
	lz4Prime4 = 0x27d4eb2f
	lz4Prime5 = 0x165667b1
)

// lz4XXH32 is xxHash32, the hash the frame format checks itself with: the
// header carries eight bits of it, and each block and the whole content may
// carry all thirty-two.
func lz4XXH32(seed uint32, data []byte) uint32 {
	n, i := len(data), 0
	var h uint32
	if n >= 16 {
		// Four running values, each fed every fourth word, so that the rounds do
		// not have to wait on one another.
		v := [4]uint32{seed + lz4Prime1 + lz4Prime2, seed + lz4Prime2, seed, seed - lz4Prime1}
		for ; n-i >= 16; i += 16 {
			for lane := range v {
				word := binary.LittleEndian.Uint32(data[i+4*lane:])
				v[lane] = bits.RotateLeft32(v[lane]+word*lz4Prime2, 13) * lz4Prime1
			}
		}
		h = bits.RotateLeft32(v[0], 1) + bits.RotateLeft32(v[1], 7) +
			bits.RotateLeft32(v[2], 12) + bits.RotateLeft32(v[3], 18) + uint32(n) // #nosec G115 -- a length, and xxHash32 mixes in only its low 32 bits anyway
	} else {
		h = seed + lz4Prime5 + uint32(n) // #nosec G115 -- as above
	}

	for ; n-i >= 4; i += 4 {
		h = bits.RotateLeft32(h+binary.LittleEndian.Uint32(data[i:])*lz4Prime3, 17) * lz4Prime4
	}
	for ; i < n; i++ {
		h = bits.RotateLeft32(h+uint32(data[i])*lz4Prime5, 11) * lz4Prime1
	}

	h ^= h >> 15
	h *= lz4Prime2
	h ^= h >> 13
	h *= lz4Prime3
	return h ^ h>>16
}

func init() {
	core.Register(LZ4Compress{})
	core.Register(LZ4Decompress{})
}
