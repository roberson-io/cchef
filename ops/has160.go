package ops

import (
	"encoding/binary"
	"hash"
	"math/bits"
)

// HAS-160 is a Korean 160-bit hash (TTAS.KO-12.0011/R1).

var (
	has160K    = [4]uint32{0x00000000, 0x5a827999, 0x6ed9eba1, 0x8f1bbcdc}
	has160Rot  = [20]uint{5, 11, 7, 15, 6, 13, 8, 14, 7, 12, 9, 11, 8, 15, 6, 12, 9, 14, 5, 13}
	has160Rot2 = [4]uint{10, 17, 25, 30}
	has160Ind  = [80]int{
		18, 0, 1, 2, 3, 19, 4, 5, 6, 7, 16, 8, 9, 10, 11, 17, 12, 13, 14, 15,
		22, 3, 6, 9, 12, 23, 15, 2, 5, 8, 20, 11, 14, 1, 4, 21, 7, 10, 13, 0,
		26, 12, 5, 14, 7, 27, 0, 9, 2, 11, 24, 4, 13, 6, 15, 25, 8, 1, 10, 3,
		30, 7, 2, 13, 8, 31, 3, 14, 9, 4, 28, 15, 10, 5, 0, 29, 11, 6, 1, 12,
	}
)

type has160 struct {
	md64
	h      [5]uint32
	rounds int
}

func newHAS160() hash.Hash { return newHAS160Rounds(80) }

// newHAS160Rounds builds HAS-160 with a configurable round count (1..80).
func newHAS160Rounds(rounds int) hash.Hash {
	d := &has160{rounds: rounds}
	d.Reset()
	return d
}

func (d *has160) Reset() {
	d.md64 = md64{}
	d.h = [5]uint32{0x67452301, 0xefcdab89, 0x98badcfe, 0x10325476, 0xc3d2e1f0}
}

func (d *has160) Size() int      { return 20 }
func (d *has160) BlockSize() int { return 64 }

func (d *has160) Write(p []byte) (int, error) {
	d.write(p, d.block)
	return len(p), nil
}

func (d *has160) Sum(in []byte) []byte {
	e := *d
	e.pad(e.block, true)
	var out [20]byte
	for i, s := range e.h {
		binary.LittleEndian.PutUint32(out[i*4:], s)
	}
	return append(in, out[:]...)
}

func (d *has160) block(p []byte) {
	var w [32]uint32
	for i := range 16 {
		w[i] = binary.LittleEndian.Uint32(p[i*4:])
	}
	w[16] = w[0] ^ w[1] ^ w[2] ^ w[3]
	w[17] = w[4] ^ w[5] ^ w[6] ^ w[7]
	w[18] = w[8] ^ w[9] ^ w[10] ^ w[11]
	w[19] = w[12] ^ w[13] ^ w[14] ^ w[15]
	w[20] = w[3] ^ w[6] ^ w[9] ^ w[12]
	w[21] = w[2] ^ w[5] ^ w[8] ^ w[15]
	w[22] = w[1] ^ w[4] ^ w[11] ^ w[14]
	w[23] = w[0] ^ w[7] ^ w[10] ^ w[13]
	w[24] = w[5] ^ w[7] ^ w[12] ^ w[14]
	w[25] = w[0] ^ w[2] ^ w[9] ^ w[11]
	w[26] = w[4] ^ w[6] ^ w[13] ^ w[15]
	w[27] = w[1] ^ w[3] ^ w[8] ^ w[10]
	w[28] = w[2] ^ w[7] ^ w[8] ^ w[13]
	w[29] = w[3] ^ w[4] ^ w[9] ^ w[14]
	w[30] = w[0] ^ w[5] ^ w[10] ^ w[15]
	w[31] = w[1] ^ w[6] ^ w[11] ^ w[12]

	a, b, c, dd, e := d.h[0], d.h[1], d.h[2], d.h[3], d.h[4]
	for i := 0; i < d.rounds; i++ {
		var f uint32
		switch {
		case i < 20:
			f = (b & c) | (^b & dd)
		case i < 40:
			f = b ^ c ^ dd
		case i < 60:
			f = c ^ (b | ^dd)
		default:
			f = b ^ c ^ dd
		}
		t := bits.RotateLeft32(a, int(has160Rot[i%20])) + e + w[has160Ind[i]] + has160K[i/20] + f
		e, dd, c, b, a = dd, c, bits.RotateLeft32(b, int(has160Rot2[i/20])), a, t
	}

	d.h[0] += a
	d.h[1] += b
	d.h[2] += c
	d.h[3] += dd
	d.h[4] += e
}
