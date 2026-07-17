package ops

import (
	"encoding/binary"
	"hash"
	"math/bits"
)

// Snefru — crypto-api's default snefru128/8 (128-bit output, 8 rounds). Ported
// verbatim, including the deterministic S-box generation from the RAND table.

var snefruPow10 = [6]int{1, 10, 100, 1000, 10000, 100000}

// snefruRand is the deterministic digit stream over snefruRandTable used to
// build the S-box (matching crypto-api's getRandomDigit/getRandomNumber).
type snefruRand struct {
	count5 int
	index  int
}

func (r *snefruRand) digit() int {
	if r.count5 < 0 {
		r.count5 = 4
		r.index++
	}
	d := (snefruRandTable[r.index] % snefruPow10[r.count5+1]) / snefruPow10[r.count5]
	r.count5--
	return d
}

func (r *snefruRand) number(low, high int) int {
	rng := high - low + 1
	var rnd, mx int
	for {
		rnd, mx = 0, 1
		for mx < rng {
			rnd = rnd*10 + r.digit()
			mx *= 10
		}
		if rnd < (mx/rng)*rng {
			break
		}
	}
	return low + rnd%rng
}

// snefruSBox is the 16×256 S-box generated once at init.
var snefruSBox = generateSnefruSBox()

func generateSnefruSBox() *[16][256]uint32 {
	sbox := &[16][256]uint32{}
	r := &snefruRand{count5: 4, index: 0}
	for i := range 16 {
		for row := range 256 {
			v := uint32(row)
			sbox[i][row] = v | v<<8 | v<<16 | v<<24
		}
		for col := 3; col >= 0; col-- {
			mask := uint32(0xff) << (col << 3)
			for row := range 255 {
				row2 := r.number(row, 255)
				temp := sbox[i][row]
				sbox[i][row] = (sbox[i][row] & ^mask) | (sbox[i][row2] & mask)
				sbox[i][row2] = (sbox[i][row2] & ^mask) | (temp & mask)
			}
		}
	}
	return sbox
}

const (
	snefruRounds     = 8
	snefruWords      = 4                      // 128-bit output
	snefruBlockBytes = (16 - snefruWords) * 4 // 48
)

var snefruShift = [4]int{16, 8, 16, 24}

type snefru struct {
	x   [snefruBlockBytes]byte
	nx  int
	len uint64
	h   [snefruWords]uint32
}

func newSnefru() hash.Hash { return &snefru{} }

func (d *snefru) Reset()         { *d = snefru{} }
func (d *snefru) Size() int      { return snefruWords * 4 }
func (d *snefru) BlockSize() int { return snefruBlockBytes }

func (d *snefru) writeRaw(p []byte) {
	for len(p) > 0 {
		c := copy(d.x[d.nx:], p)
		d.nx += c
		p = p[c:]
		if d.nx == snefruBlockBytes {
			d.block(d.x[:])
			d.nx = 0
		}
	}
}

func (d *snefru) Write(p []byte) (int, error) {
	d.len += uint64(len(p))
	d.writeRaw(p)
	return len(p), nil
}

func (d *snefru) Sum(in []byte) []byte {
	e := *d
	bitLen := e.len << 3
	// Zero-pad the partial block (if any) to a full block, then a final block of
	// zeros plus the 64-bit big-endian bit length.
	if e.nx > 0 {
		e.writeRaw(make([]byte, snefruBlockBytes-e.nx))
	}
	e.writeRaw(make([]byte, snefruBlockBytes-8))
	var lenb [8]byte
	binary.BigEndian.PutUint64(lenb[:], bitLen)
	e.writeRaw(lenb[:])

	var out [snefruWords * 4]byte
	for i := range snefruWords {
		binary.BigEndian.PutUint32(out[i*4:], e.h[i])
	}
	return append(in, out[:]...)
}

func (d *snefru) block(p []byte) {
	var w [16]uint32
	for i := range snefruWords {
		w[i] = d.h[i]
	}
	for i := snefruWords; i < 16; i++ {
		w[i] = binary.BigEndian.Uint32(p[(i-snefruWords)*4:])
	}
	for i := 0; i < snefruRounds<<1; i += 2 {
		for byteInWord := range 4 {
			for n := range 16 {
				sbe := snefruSBox[i+(n/2)%2][w[n]&0xff]
				w[(n-1)&0xf] ^= sbe
				w[(n+1)&0xf] ^= sbe
			}
			for n := range 16 {
				w[n] = bits.RotateLeft32(w[n], -snefruShift[byteInWord])
			}
		}
	}
	for i := range snefruWords {
		d.h[i] ^= w[15-i]
	}
}
