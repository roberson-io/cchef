package ops

import (
	"encoding/binary"
	"hash"
	"math/bits"
)

// MD4 (RFC 1320). Ported from crypto-api's md4 hasher (little-endian, standard
// Merkle–Damgård padding).

func md4FF(x, y, z uint32) uint32 { return (x & y) | (^x & z) }
func md4GG(x, y, z uint32) uint32 { return (x & y) | (x & z) | (y & z) }
func md4HH(x, y, z uint32) uint32 { return x ^ y ^ z }

func md4CC(f func(a, b, c uint32) uint32, k, a, x, y, z, m uint32, s int) uint32 {
	return bits.RotateLeft32(a+f(x, y, z)+m+k, s)
}

type md4hash struct {
	md64
	h [4]uint32
}

func newMD4() hash.Hash {
	d := &md4hash{}
	d.Reset()
	return d
}

func (d *md4hash) Reset() {
	d.md64 = md64{}
	d.h = [4]uint32{0x67452301, 0xefcdab89, 0x98badcfe, 0x10325476}
}

func (d *md4hash) Size() int      { return 16 }
func (d *md4hash) BlockSize() int { return 64 }

func (d *md4hash) Write(p []byte) (int, error) {
	d.write(p, d.block)
	return len(p), nil
}

func (d *md4hash) Sum(in []byte) []byte {
	e := *d
	e.pad(e.block, true)
	var out [16]byte
	for i, s := range e.h {
		binary.LittleEndian.PutUint32(out[i*4:], s)
	}
	return append(in, out[:]...)
}

func (d *md4hash) block(p []byte) {
	var x [16]uint32
	for i := range 16 {
		x[i] = binary.LittleEndian.Uint32(p[i*4:])
	}
	a, b, c, e := d.h[0], d.h[1], d.h[2], d.h[3]

	// Round 1: F, k=0, shifts 3/7/11/19.
	const kf = 0x00000000
	for i := range 4 {
		a = md4CC(md4FF, kf, a, b, c, e, x[i*4], 3)
		e = md4CC(md4FF, kf, e, a, b, c, x[i*4+1], 7)
		c = md4CC(md4FF, kf, c, e, a, b, x[i*4+2], 11)
		b = md4CC(md4FF, kf, b, c, e, a, x[i*4+3], 19)
	}
	// Round 2: G, k=0x5a827999, column order, shifts 3/5/9/13.
	const kg = 0x5a827999
	for i := range 4 {
		a = md4CC(md4GG, kg, a, b, c, e, x[i], 3)
		e = md4CC(md4GG, kg, e, a, b, c, x[i+4], 5)
		c = md4CC(md4GG, kg, c, e, a, b, x[i+8], 9)
		b = md4CC(md4GG, kg, b, c, e, a, x[i+12], 13)
	}
	// Round 3: H, k=0x6ed9eba1, bit-reversed order, shifts 3/9/11/15.
	const kh = 0x6ed9eba1
	order3 := [4]int{0, 2, 1, 3}
	for _, i := range order3 {
		a = md4CC(md4HH, kh, a, b, c, e, x[i], 3)
		e = md4CC(md4HH, kh, e, a, b, c, x[i+8], 9)
		c = md4CC(md4HH, kh, c, e, a, b, x[i+4], 11)
		b = md4CC(md4HH, kh, b, c, e, a, x[i+12], 15)
	}

	d.h[0] += a
	d.h[1] += b
	d.h[2] += c
	d.h[3] += e
}
