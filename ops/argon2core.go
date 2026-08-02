package ops

import (
	"encoding/binary"
	"math/bits"

	"golang.org/x/crypto/blake2b"
)

// An Argon2d implementation (RFC 9106). It exists because
// golang.org/x/crypto/argon2 only provides Argon2i and Argon2id; the Argon2 and
// Argon2 compare operations use x/crypto for those two and this for Argon2d.
// Argon2d uses purely data-dependent addressing, so the data-independent
// address-block machinery Argon2i/Argon2id need is not implemented here.

const (
	argon2BlockWords = 128 // 1024-byte block as uint64 words
	argon2SyncPoints = 4   // slices per lane
	argon2Version    = 0x13
)

// argon2Block is one 1024-byte Argon2 memory block.
type argon2Block [argon2BlockWords]uint64

// argon2dRaw computes the raw Argon2d tag for the given password, salt and
// parameters (time, memory in KiB, parallelism/lanes, and tag length in bytes).
func argon2dRaw(password, salt []byte, time, memory, parallelism, tagLen uint32) []byte {
	h0 := argon2InitialHash(password, salt, time, memory, parallelism, tagLen)

	segmentLength := memory / (argon2SyncPoints * parallelism)
	laneLength := segmentLength * argon2SyncPoints
	memoryBlocks := laneLength * parallelism
	mem := make([]argon2Block, memoryBlocks)

	// First two blocks of each lane come from the variable-length hash of H0.
	seed := make([]byte, 72)
	copy(seed, h0)
	for lane := range parallelism {
		for col := range uint32(2) {
			binary.LittleEndian.PutUint32(seed[64:], col)
			binary.LittleEndian.PutUint32(seed[68:], lane)
			bytesToArgon2Block(&mem[lane*laneLength+col], argon2Hprime(seed, argon2BlockWords*8))
		}
	}

	for pass := range time {
		for slice := range uint32(argon2SyncPoints) {
			for lane := range parallelism {
				argon2FillSegment(mem, pass, slice, lane, segmentLength, laneLength, parallelism)
			}
		}
	}

	// XOR the last block of every lane, then run it through the final hash.
	final := mem[laneLength-1]
	for lane := uint32(1); lane < parallelism; lane++ {
		last := &mem[lane*laneLength+laneLength-1]
		for i := range final {
			final[i] ^= last[i]
		}
	}
	return argon2Hprime(argon2BlockToBytes(&final), int(tagLen))
}

// argon2InitialHash computes H0, the 64-byte prehash of the parameters and
// inputs (with empty secret and associated data, as CyberChef uses).
func argon2InitialHash(password, salt []byte, time, memory, parallelism, tagLen uint32) []byte {
	h, _ := blake2b.New(64, nil)
	var buf [4]byte
	put := func(v uint32) {
		binary.LittleEndian.PutUint32(buf[:], v)
		h.Write(buf[:])
	}
	put(parallelism)
	put(tagLen)
	put(memory)
	put(time)
	put(argon2Version)
	put(0)                     // type: Argon2d = 0
	put(uint32(len(password))) // #nosec G115 -- password length fits a uint32
	h.Write(password)
	put(uint32(len(salt))) // #nosec G115 -- salt length fits a uint32
	h.Write(salt)
	put(0) // secret length
	put(0) // associated data length
	return h.Sum(nil)
}

// argon2FillSegment fills one segment (slice of a lane) using data-dependent
// addressing.
func argon2FillSegment(mem []argon2Block, pass, slice, lane, segmentLength, laneLength, parallelism uint32) {
	startIdx := uint32(0)
	if pass == 0 && slice == 0 {
		startIdx = 2
	}
	currOffset := lane*laneLength + slice*segmentLength + startIdx
	prevOffset := currOffset - 1
	if currOffset%laneLength == 0 {
		prevOffset = currOffset + laneLength - 1
	}

	for i := startIdx; i < segmentLength; i++ {
		if currOffset%laneLength == 1 {
			prevOffset = currOffset - 1
		}
		pseudoRand := mem[prevOffset][0]

		refLane := lane
		if pass != 0 || slice != 0 {
			refLane = uint32(pseudoRand>>32) % parallelism
		}
		refIndex := argon2IndexAlpha(pass, slice, i, uint32(pseudoRand), refLane == lane, segmentLength, laneLength) // #nosec G115 -- low 32 bits of the pseudo-random value

		ref := &mem[refLane*laneLength+refIndex]
		curr := &mem[currOffset]
		argon2FillBlock(&mem[prevOffset], ref, curr, pass != 0)

		currOffset++
		prevOffset++
	}
}

// argon2IndexAlpha maps the pseudo-random value to a reference block index within
// the reference lane (RFC 9106 §3.4.1.2).
func argon2IndexAlpha(pass, slice, index, rand uint32, sameLane bool, segmentLength, laneLength uint32) uint32 {
	var refAreaSize uint32
	switch {
	case pass == 0 && slice == 0:
		refAreaSize = index - 1
	case pass == 0:
		refAreaSize = slice*segmentLength + index - 1
		if !sameLane {
			refAreaSize = slice * segmentLength
			if index == 0 {
				refAreaSize--
			}
		}
	default:
		refAreaSize = laneLength - segmentLength + index - 1
		if !sameLane {
			refAreaSize = laneLength - segmentLength
			if index == 0 {
				refAreaSize--
			}
		}
	}

	rel := uint64(rand)
	rel = (rel * rel) >> 32
	rel = uint64(refAreaSize) - 1 - ((uint64(refAreaSize) * rel) >> 32)

	startPos := uint32(0)
	if pass != 0 && slice != argon2SyncPoints-1 {
		startPos = (slice + 1) * segmentLength
	}
	return (startPos + uint32(rel)) % laneLength // #nosec G115 -- relative position is bounded by the reference area
}

// argon2FillBlock is Argon2's compression function G: next = R ^ P(R) where
// R = prev ^ ref (xored with the existing next block on passes after the first).
func argon2FillBlock(prev, ref, next *argon2Block, withXor bool) {
	var r, tmp argon2Block
	for i := range r {
		r[i] = prev[i] ^ ref[i]
	}
	tmp = r
	if withXor {
		for i := range tmp {
			tmp[i] ^= next[i]
		}
	}

	for i := range 8 { // rows: 16 contiguous words (pair stride 2)
		argon2RoundGather(&r, 16*i, 2)
	}
	for i := range 8 { // columns: pairs strided by 16
		argon2RoundGather(&r, 2*i, 16)
	}
	for i := range next {
		next[i] = r[i] ^ tmp[i]
	}
}

// argon2RoundGather applies the BLAKE2 round to the 16 block words starting at
// base with the given stride (contiguous for rows, strided for columns).
func argon2RoundGather(block *argon2Block, base, stride int) {
	var v [16]uint64
	idx := [16]int{}
	for j := range 16 {
		idx[j] = base + (j/2)*stride + (j % 2)
		v[j] = block[idx[j]]
	}
	argon2Round(&v)
	for j := range 16 {
		block[idx[j]] = v[j]
	}
}

// argon2Round is the BLAKE2b round used by Argon2's permutation (no message).
func argon2Round(v *[16]uint64) {
	argon2GB(v, 0, 4, 8, 12)
	argon2GB(v, 1, 5, 9, 13)
	argon2GB(v, 2, 6, 10, 14)
	argon2GB(v, 3, 7, 11, 15)
	argon2GB(v, 0, 5, 10, 15)
	argon2GB(v, 1, 6, 11, 12)
	argon2GB(v, 2, 7, 8, 13)
	argon2GB(v, 3, 4, 9, 14)
}

// argon2GB is Argon2's mixing function, the BLAKE2b GB with the extra 64-bit
// multiplication of the low 32-bit halves.
func argon2GB(v *[16]uint64, a, b, c, d int) {
	v[a] = v[a] + v[b] + 2*(v[a]&0xffffffff)*(v[b]&0xffffffff)
	v[d] = bits.RotateLeft64(v[d]^v[a], -32)
	v[c] = v[c] + v[d] + 2*(v[c]&0xffffffff)*(v[d]&0xffffffff)
	v[b] = bits.RotateLeft64(v[b]^v[c], -24)
	v[a] = v[a] + v[b] + 2*(v[a]&0xffffffff)*(v[b]&0xffffffff)
	v[d] = bits.RotateLeft64(v[d]^v[a], -16)
	v[c] = v[c] + v[d] + 2*(v[c]&0xffffffff)*(v[d]&0xffffffff)
	v[b] = bits.RotateLeft64(v[b]^v[c], -63)
}

// argon2Hprime is the variable-length hash H' (RFC 9106 §3.3): a BLAKE2b of
// (LE32(outLen) ‖ input) for outLen <= 64, or a chained construction otherwise.
func argon2Hprime(input []byte, outLen int) []byte {
	var lenPrefix [4]byte
	binary.LittleEndian.PutUint32(lenPrefix[:], uint32(outLen)) // #nosec G115 -- outLen is a bounded hash length

	if outLen <= 64 {
		h, _ := blake2b.New(outLen, nil)
		h.Write(lenPrefix[:])
		h.Write(input)
		return h.Sum(nil)
	}

	h, _ := blake2b.New(64, nil)
	h.Write(lenPrefix[:])
	h.Write(input)
	v := h.Sum(nil)
	out := append([]byte{}, v[:32]...)
	remaining := outLen - 32
	for remaining > 64 {
		hn, _ := blake2b.New(64, nil)
		hn.Write(v)
		v = hn.Sum(nil)
		out = append(out, v[:32]...)
		remaining -= 32
	}
	hf, _ := blake2b.New(remaining, nil)
	hf.Write(v)
	return append(out, hf.Sum(nil)...)
}

// bytesToArgon2Block loads 1024 little-endian bytes into a block.
func bytesToArgon2Block(b *argon2Block, data []byte) {
	for i := range b {
		b[i] = binary.LittleEndian.Uint64(data[i*8:])
	}
}

// argon2BlockToBytes serialises a block to 1024 little-endian bytes.
func argon2BlockToBytes(b *argon2Block) []byte {
	out := make([]byte, argon2BlockWords*8)
	for i, w := range b {
		binary.LittleEndian.PutUint64(out[i*8:], w)
	}
	return out
}
