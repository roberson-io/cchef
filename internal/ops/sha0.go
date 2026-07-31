package ops

import (
	"encoding/binary"
	"hash"
	"math/bits"
)

// sha01 is the SHA-0/SHA-1 construction. The two differ in one place: SHA-1
// rotates each derived message-schedule word left by one bit, which is the
// change that fixed SHA-0's withdrawn design.

type sha01 struct {
	md64
	h      [5]uint32
	rounds int
	rotate bool // set for SHA-1
}

func newSHA0() hash.Hash { return newSHA0Rounds(shaLegacyRounds) }

// shaLegacyRounds is the standard round count for both SHA-0 and SHA-1.
const shaLegacyRounds = 80

// newSHA0Rounds builds SHA0 with a configurable round count (the standalone SHA0
// operation exposes this; HKDF uses the default 80).
func newSHA0Rounds(rounds int) hash.Hash {
	d := &sha01{rounds: rounds}
	d.Reset()
	return d
}

// newSHA1Rounds builds SHA1 with a configurable round count.
func newSHA1Rounds(rounds int) hash.Hash {
	d := &sha01{rounds: rounds, rotate: true}
	d.Reset()
	return d
}

func (d *sha01) Reset() {
	d.md64 = md64{}
	d.h = [5]uint32{0x67452301, 0xefcdab89, 0x98badcfe, 0x10325476, 0xc3d2e1f0}
}

func (d *sha01) Size() int      { return 20 }
func (d *sha01) BlockSize() int { return 64 }

func (d *sha01) Write(p []byte) (int, error) {
	d.write(p, d.block)
	return len(p), nil
}

func (d *sha01) Sum(in []byte) []byte {
	e := *d
	e.pad(e.block, false)
	var out [20]byte
	for i, s := range e.h {
		binary.BigEndian.PutUint32(out[i*4:], s)
	}
	return append(in, out[:]...)
}

func (d *sha01) block(p []byte) {
	w := make([]uint32, d.rounds)
	for i := 0; i < 16 && i < d.rounds; i++ {
		w[i] = binary.BigEndian.Uint32(p[i*4:])
	}
	for i := 16; i < d.rounds; i++ {
		w[i] = w[i-3] ^ w[i-8] ^ w[i-14] ^ w[i-16]
		if d.rotate {
			w[i] = bits.RotateLeft32(w[i], 1)
		}
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
