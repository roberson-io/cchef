package ops

import (
	"encoding/binary"
	"hash"
	"math/bits"
)

// SHA0 (the withdrawn 1993 SHA) is SHA-1 without the one-bit rotation in the
// message schedule. Ported from crypto-api's sha0 hasher.

type sha0 struct {
	md64
	h      [5]uint32
	rounds int
}

func newSHA0() hash.Hash { return newSHA0Rounds(80) }

// newSHA0Rounds builds SHA0 with a configurable round count (the standalone SHA0
// operation exposes this; HKDF uses the default 80).
func newSHA0Rounds(rounds int) hash.Hash {
	d := &sha0{rounds: rounds}
	d.Reset()
	return d
}

func (d *sha0) Reset() {
	d.md64 = md64{}
	d.h = [5]uint32{0x67452301, 0xefcdab89, 0x98badcfe, 0x10325476, 0xc3d2e1f0}
}

func (d *sha0) Size() int      { return 20 }
func (d *sha0) BlockSize() int { return 64 }

func (d *sha0) Write(p []byte) (int, error) {
	d.write(p, d.block)
	return len(p), nil
}

func (d *sha0) Sum(in []byte) []byte {
	e := *d
	e.pad(e.block, false)
	var out [20]byte
	for i, s := range e.h {
		binary.BigEndian.PutUint32(out[i*4:], s)
	}
	return append(in, out[:]...)
}

func (d *sha0) block(p []byte) {
	w := make([]uint32, d.rounds)
	for i := 0; i < 16 && i < d.rounds; i++ {
		w[i] = binary.BigEndian.Uint32(p[i*4:])
	}
	for i := 16; i < d.rounds; i++ {
		// SHA-0 omits SHA-1's rotateLeft(..., 1) here.
		w[i] = w[i-3] ^ w[i-8] ^ w[i-14] ^ w[i-16]
	}

	a, b, c, dd, e := d.h[0], d.h[1], d.h[2], d.h[3], d.h[4]
	for i := 0; i < d.rounds; i++ {
		var f, k uint32
		switch {
		case i < 20:
			f, k = (b&c)|(^b&dd), 0x5a827999
		case i < 40:
			f, k = b^c^dd, 0x6ed9eba1
		case i < 60:
			f, k = (b&c)|(b&dd)|(c&dd), 0x8f1bbcdc
		default:
			f, k = b^c^dd, 0xca62c1d6
		}
		t := bits.RotateLeft32(a, 5) + e + w[i] + k + f
		e, dd, c, b, a = dd, c, bits.RotateLeft32(b, 30), a, t
	}

	d.h[0] += a
	d.h[1] += b
	d.h[2] += c
	d.h[3] += dd
	d.h[4] += e
}
