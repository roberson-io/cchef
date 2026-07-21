package ops

// Pure-Go BLAKE3, reimplemented from the reference design
// (https://github.com/BLAKE3-team/BLAKE3/blob/master/reference_impl). It exists
// because BLAKE3 is not in golang.org/x/crypto and the CyberChef operation only
// wraps @noble/hashes. It supports the two modes the operation uses — the plain
// and keyed hashes — with extendable (XOF) output; derive-key mode is not needed.

import (
	"encoding/binary"
	"math/bits"
)

const (
	blake3BlockLen = 64   // compression input block length in bytes
	blake3ChunkLen = 1024 // bytes per chunk (16 blocks)

	// Domain-separation flags.
	blake3ChunkStart = 1 << 0
	blake3ChunkEnd   = 1 << 1
	blake3Parent     = 1 << 2
	blake3Root       = 1 << 3
	blake3KeyedHash  = 1 << 4
)

// blake3IV is the initial chaining value (the SHA-256 IV).
var blake3IV = [8]uint32{
	0x6A09E667, 0xBB67AE85, 0x3C6EF372, 0xA54FF53A,
	0x510E527F, 0x9B05688C, 0x1F83D9AB, 0x5BE0CD19,
}

// blake3MsgPermute is the message-word permutation applied between rounds.
var blake3MsgPermute = [16]int{2, 6, 3, 10, 7, 0, 4, 13, 1, 11, 12, 5, 9, 14, 15, 8}

// blake3G is the BLAKE3 mixing function applied to four state words.
func blake3G(s *[16]uint32, a, b, c, d int, mx, my uint32) {
	s[a] = s[a] + s[b] + mx
	s[d] = bits.RotateLeft32(s[d]^s[a], -16)
	s[c] += s[d]
	s[b] = bits.RotateLeft32(s[b]^s[c], -12)
	s[a] = s[a] + s[b] + my
	s[d] = bits.RotateLeft32(s[d]^s[a], -8)
	s[c] += s[d]
	s[b] = bits.RotateLeft32(s[b]^s[c], -7)
}

// blake3Round mixes the message block into the state (one of seven rounds).
func blake3Round(s *[16]uint32, m *[16]uint32) {
	blake3G(s, 0, 4, 8, 12, m[0], m[1])
	blake3G(s, 1, 5, 9, 13, m[2], m[3])
	blake3G(s, 2, 6, 10, 14, m[4], m[5])
	blake3G(s, 3, 7, 11, 15, m[6], m[7])
	blake3G(s, 0, 5, 10, 15, m[8], m[9])
	blake3G(s, 1, 6, 11, 12, m[10], m[11])
	blake3G(s, 2, 7, 8, 13, m[12], m[13])
	blake3G(s, 3, 4, 9, 14, m[14], m[15])
}

// blake3Compress runs the compression function, returning the full 16-word
// state (the first 8 words are the chaining value; all 16 feed the XOF).
func blake3Compress(cv *[8]uint32, block *[16]uint32, counter uint64, blockLen, flags uint32) [16]uint32 {
	s := [16]uint32{
		cv[0], cv[1], cv[2], cv[3], cv[4], cv[5], cv[6], cv[7],
		blake3IV[0], blake3IV[1], blake3IV[2], blake3IV[3],
		uint32(counter), uint32(counter >> 32), blockLen, flags, // #nosec G115 -- BLAKE3 splits the 64-bit counter into two 32-bit words
	}
	m := *block
	for r := range 7 {
		blake3Round(&s, &m)
		if r < 6 {
			var p [16]uint32
			for i := range p {
				p[i] = m[blake3MsgPermute[i]]
			}
			m = p
		}
	}
	for i := range 8 {
		s[i] ^= s[i+8]
		s[i+8] ^= cv[i]
	}
	return s
}

// blake3WordsFromLE reads a 64-byte block as 16 little-endian words.
func blake3WordsFromLE(block []byte) [16]uint32 {
	var w [16]uint32
	for i := range w {
		w[i] = binary.LittleEndian.Uint32(block[i*4:])
	}
	return w
}

// blake3Output is a chunk's final block or a parent node, from which either a
// chaining value or the extendable root output can be produced.
type blake3Output struct {
	inputCV  [8]uint32
	block    [16]uint32
	counter  uint64
	blockLen uint32
	flags    uint32
}

// chainingValue compresses this output to its 8-word chaining value.
func (o blake3Output) chainingValue() [8]uint32 {
	full := blake3Compress(&o.inputCV, &o.block, o.counter, o.blockLen, o.flags)
	var cv [8]uint32
	copy(cv[:], full[:8])
	return cv
}

// rootBytes produces outLen bytes of extendable root output.
func (o blake3Output) rootBytes(outLen int) []byte {
	out := make([]byte, 0, outLen)
	for counter := uint64(0); len(out) < outLen; counter++ {
		words := blake3Compress(&o.inputCV, &o.block, counter, o.blockLen, o.flags|blake3Root)
		for _, w := range words {
			var b [4]byte
			binary.LittleEndian.PutUint32(b[:], w)
			out = append(out, b[:]...)
		}
	}
	return out[:outLen]
}

// blake3ChunkState accumulates the blocks of one chunk.
type blake3ChunkState struct {
	cv               [8]uint32
	chunkCounter     uint64
	block            [blake3BlockLen]byte
	blockLen         int
	blocksCompressed int
	flags            uint32
}

func newBlake3ChunkState(key [8]uint32, chunkCounter uint64, flags uint32) *blake3ChunkState {
	return &blake3ChunkState{cv: key, chunkCounter: chunkCounter, flags: flags}
}

func (c *blake3ChunkState) length() int {
	return blake3BlockLen*c.blocksCompressed + c.blockLen
}

func (c *blake3ChunkState) startFlag() uint32 {
	if c.blocksCompressed == 0 {
		return blake3ChunkStart
	}
	return 0
}

// update absorbs input bytes, compressing full blocks as they complete.
func (c *blake3ChunkState) update(input []byte) {
	for len(input) > 0 {
		if c.blockLen == blake3BlockLen {
			words := blake3WordsFromLE(c.block[:])
			full := blake3Compress(&c.cv, &words, c.chunkCounter, blake3BlockLen, c.flags|c.startFlag())
			copy(c.cv[:], full[:8])
			c.blocksCompressed++
			c.block = [blake3BlockLen]byte{}
			c.blockLen = 0
		}
		take := min(blake3BlockLen-c.blockLen, len(input))
		copy(c.block[c.blockLen:], input[:take])
		c.blockLen += take
		input = input[take:]
	}
}

// output returns the chunk's final Output (the last block, with CHUNK_END).
func (c *blake3ChunkState) output() blake3Output {
	words := blake3WordsFromLE(c.block[:])
	return blake3Output{
		inputCV:  c.cv,
		block:    words,
		counter:  c.chunkCounter,
		blockLen: uint32(c.blockLen), // #nosec G115 -- block length is 0..64
		flags:    c.flags | c.startFlag() | blake3ChunkEnd,
	}
}

// blake3ParentOutput combines two child chaining values into a parent node.
func blake3ParentOutput(left, right, key [8]uint32, flags uint32) blake3Output {
	var block [16]uint32
	copy(block[:8], left[:])
	copy(block[8:], right[:])
	return blake3Output{inputCV: key, block: block, counter: 0, blockLen: blake3BlockLen, flags: blake3Parent | flags}
}

// blake3ParentCV is the chaining value of a parent node.
func blake3ParentCV(left, right, key [8]uint32, flags uint32) [8]uint32 {
	out := blake3ParentOutput(left, right, key, flags)
	return out.chainingValue()
}

// blake3Hasher is an incremental BLAKE3 hasher with the subtree chaining stack.
type blake3Hasher struct {
	chunk    *blake3ChunkState
	key      [8]uint32
	cvStack  [54][8]uint32
	stackLen int
	flags    uint32
}

func newBlake3Hasher(key [8]uint32, flags uint32) *blake3Hasher {
	return &blake3Hasher{chunk: newBlake3ChunkState(key, 0, flags), key: key, flags: flags}
}

// blake3New returns a plain-mode hasher.
func blake3New() *blake3Hasher { return newBlake3Hasher(blake3IV, 0) }

// blake3NewKeyed returns a keyed-mode hasher for a 32-byte key.
func blake3NewKeyed(key []byte) *blake3Hasher {
	var kw [8]uint32
	for i := range kw {
		kw[i] = binary.LittleEndian.Uint32(key[i*4:])
	}
	return newBlake3Hasher(kw, blake3KeyedHash)
}

// addChunkCV merges a completed chunk's chaining value into the subtree stack,
// collapsing pairs of equal-size subtrees as indicated by the chunk count.
func (h *blake3Hasher) addChunkCV(cv [8]uint32, totalChunks uint64) {
	for totalChunks&1 == 0 {
		h.stackLen--
		cv = blake3ParentCV(h.cvStack[h.stackLen], cv, h.key, h.flags)
		totalChunks >>= 1
	}
	h.cvStack[h.stackLen] = cv
	h.stackLen++
}

// update absorbs input into the hasher.
func (h *blake3Hasher) update(input []byte) {
	for len(input) > 0 {
		if h.chunk.length() == blake3ChunkLen {
			cv := h.chunk.output().chainingValue()
			totalChunks := h.chunk.chunkCounter + 1
			h.addChunkCV(cv, totalChunks)
			h.chunk = newBlake3ChunkState(h.key, totalChunks, h.flags)
		}
		take := min(blake3ChunkLen-h.chunk.length(), len(input))
		h.chunk.update(input[:take])
		input = input[take:]
	}
}

// finalize collapses the subtree stack and produces outLen bytes of output.
func (h *blake3Hasher) finalize(outLen int) []byte {
	output := h.chunk.output()
	for i := h.stackLen - 1; i >= 0; i-- {
		output = blake3ParentOutput(h.cvStack[i], output.chainingValue(), h.key, h.flags)
	}
	return output.rootBytes(outLen)
}
