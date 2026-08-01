package ops

import (
	"encoding/binary"
	"hash"
	"math/bits"
)

// RIPEMD-128/256/320, ported from crypto-api's ripemd hasher. RIPEMD-160 is
// provided by golang.org/x/crypto/ripemd160; these are the other three lengths.

var (
	ripZL = [80]int{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15,
		7, 4, 13, 1, 10, 6, 15, 3, 12, 0, 9, 5, 2, 14, 11, 8,
		3, 10, 14, 4, 9, 15, 8, 1, 2, 7, 0, 6, 13, 11, 5, 12,
		1, 9, 11, 10, 0, 8, 12, 4, 13, 3, 7, 15, 14, 5, 6, 2,
		4, 0, 5, 9, 7, 12, 2, 10, 14, 1, 3, 8, 11, 6, 15, 13,
	}
	ripZR = [80]int{
		5, 14, 7, 0, 9, 2, 11, 4, 13, 6, 15, 8, 1, 10, 3, 12,
		6, 11, 3, 7, 0, 13, 5, 10, 14, 15, 8, 12, 4, 9, 1, 2,
		15, 5, 1, 3, 7, 14, 6, 9, 11, 8, 12, 2, 10, 0, 4, 13,
		8, 6, 4, 1, 3, 11, 15, 0, 5, 12, 2, 13, 9, 7, 10, 14,
		12, 15, 10, 4, 1, 5, 8, 7, 6, 2, 13, 14, 0, 3, 9, 11,
	}
	ripSL = [80]int{
		11, 14, 15, 12, 5, 8, 7, 9, 11, 13, 14, 15, 6, 7, 9, 8,
		7, 6, 8, 13, 11, 9, 7, 15, 7, 12, 15, 9, 11, 7, 13, 12,
		11, 13, 6, 7, 14, 9, 13, 15, 14, 8, 13, 6, 5, 12, 7, 5,
		11, 12, 14, 15, 14, 15, 9, 8, 9, 14, 5, 6, 8, 6, 5, 12,
		9, 15, 5, 11, 6, 8, 13, 12, 5, 12, 13, 14, 11, 8, 5, 6,
	}
	ripSR = [80]int{
		8, 9, 9, 11, 13, 15, 15, 5, 7, 7, 8, 11, 14, 14, 12, 6,
		9, 13, 15, 7, 12, 8, 9, 11, 7, 7, 12, 7, 6, 15, 13, 11,
		9, 7, 15, 11, 8, 6, 6, 14, 12, 13, 5, 14, 13, 13, 7, 5,
		15, 5, 8, 11, 14, 14, 6, 14, 6, 9, 12, 9, 12, 5, 15, 8,
		8, 5, 12, 9, 12, 5, 14, 6, 8, 13, 6, 5, 15, 13, 11, 11,
	}
)

func ripF(x, y, z uint32) uint32 { return x ^ y ^ z }
func ripG(x, y, z uint32) uint32 { return (x & y) | (^x & z) }
func ripH(x, y, z uint32) uint32 { return (x | ^y) ^ z }
func ripI(x, y, z uint32) uint32 { return (x & z) | (y & ^z) }
func ripJ(x, y, z uint32) uint32 { return x ^ (y | ^z) }

// ripT is the left-line round function (used by all lengths).
func ripT(i int, b, c, d uint32) uint32 {
	switch {
	case i < 16:
		return ripF(b, c, d)
	case i < 32:
		return ripG(b, c, d) + 0x5a827999
	case i < 48:
		return ripH(b, c, d) + 0x6ed9eba1
	case i < 64:
		return ripI(b, c, d) + 0x8f1bbcdc
	default:
		return ripJ(b, c, d) + 0xa953fd4e
	}
}

// ripT64 is the right-line round function for the 64-round lengths (128, 256).
func ripT64(i int, b, c, d uint32) uint32 {
	switch {
	case i < 16:
		return ripI(b, c, d) + 0x50a28be6
	case i < 32:
		return ripH(b, c, d) + 0x5c4dd124
	case i < 48:
		return ripG(b, c, d) + 0x6d703ef3
	default:
		return ripF(b, c, d)
	}
}

// ripT80 is the right-line round function for the 80-round lengths (160, 320).
func ripT80(i int, b, c, d uint32) uint32 {
	switch {
	case i < 16:
		return ripJ(b, c, d) + 0x50a28be6
	case i < 32:
		return ripI(b, c, d) + 0x5c4dd124
	case i < 48:
		return ripH(b, c, d) + 0x6d703ef3
	case i < 64:
		return ripG(b, c, d) + 0x7a6d76e9
	default:
		return ripF(b, c, d)
	}
}

// ripemd holds the state for one of the 128/256/320-bit variants. words is the
// number of 32-bit output words (4, 8 or 10).
type ripemd struct {
	md64
	h     []uint32
	words int
}

func newRIPEMD128() hash.Hash { return &ripemd{words: 4, h: ripInit(4)} }
func newRIPEMD160() hash.Hash { return &ripemd{words: 5, h: ripInit(5)} }
func newRIPEMD256() hash.Hash { return &ripemd{words: 8, h: ripInit(8)} }
func newRIPEMD320() hash.Hash { return &ripemd{words: 10, h: ripInit(10)} }

func ripInit(words int) []uint32 {
	switch words {
	case 4:
		return []uint32{0x67452301, 0xefcdab89, 0x98badcfe, 0x10325476}
	case 5:
		return []uint32{0x67452301, 0xefcdab89, 0x98badcfe, 0x10325476, 0xc3d2e1f0}
	case 8:
		return []uint32{
			0x67452301, 0xefcdab89, 0x98badcfe, 0x10325476,
			0x76543210, 0xfedcba98, 0x89abcdef, 0x01234567,
		}
	default: // 10
		return []uint32{
			0x67452301, 0xefcdab89, 0x98badcfe, 0x10325476, 0xc3d2e1f0,
			0x76543210, 0xfedcba98, 0x89abcdef, 0x01234567, 0x3c2d1e0f,
		}
	}
}

func (d *ripemd) Reset()         { d.md64 = md64{}; d.h = ripInit(d.words) }
func (d *ripemd) Size() int      { return d.words * 4 }
func (d *ripemd) BlockSize() int { return 64 }

func (d *ripemd) Write(p []byte) (int, error) {
	d.write(p, d.block)
	return len(p), nil
}

func (d *ripemd) Sum(in []byte) []byte {
	e := *d
	e.h = append([]uint32(nil), d.h...)
	e.pad(e.block, true)
	out := make([]byte, e.words*4)
	for i := 0; i < e.words; i++ {
		binary.LittleEndian.PutUint32(out[i*4:], e.h[i])
	}
	return append(in, out...)
}

func (d *ripemd) block(p []byte) {
	var m [16]uint32
	for i := range 16 {
		m[i] = binary.LittleEndian.Uint32(p[i*4:])
	}
	switch d.words {
	case 4:
		d.block128(&m)
	case 5:
		d.block160(&m)
	case 8:
		d.block256(&m)
	default:
		d.block320(&m)
	}
}

func (d *ripemd) block160(m *[16]uint32) {
	al, bl, cl, dl, el := d.h[0], d.h[1], d.h[2], d.h[3], d.h[4]
	ar, br, cr, dr, er := al, bl, cl, dl, el
	for i := range 80 {
		t := bits.RotateLeft32(al+m[ripZL[i]]+ripT(i, bl, cl, dl), ripSL[i]) + el
		al, el, dl, cl, bl = el, dl, bits.RotateLeft32(cl, 10), bl, t
		t = bits.RotateLeft32(ar+m[ripZR[i]]+ripT80(i, br, cr, dr), ripSR[i]) + er
		ar, er, dr, cr, br = er, dr, bits.RotateLeft32(cr, 10), br, t
	}
	t := d.h[1] + cl + dr
	d.h[1] = d.h[2] + dl + er
	d.h[2] = d.h[3] + el + ar
	d.h[3] = d.h[4] + al + br
	d.h[4] = d.h[0] + bl + cr
	d.h[0] = t
}

func (d *ripemd) block128(m *[16]uint32) {
	al, bl, cl, dl := d.h[0], d.h[1], d.h[2], d.h[3]
	ar, br, cr, dr := al, bl, cl, dl
	for i := range 64 {
		t := bits.RotateLeft32(al+m[ripZL[i]]+ripT(i, bl, cl, dl), ripSL[i])
		al, dl, cl, bl = dl, cl, bl, t
		t = bits.RotateLeft32(ar+m[ripZR[i]]+ripT64(i, br, cr, dr), ripSR[i])
		ar, dr, cr, br = dr, cr, br, t
	}
	t := d.h[1] + cl + dr
	d.h[1] = d.h[2] + dl + ar
	d.h[2] = d.h[3] + al + br
	d.h[3] = d.h[0] + bl + cr
	d.h[0] = t
}

func (d *ripemd) block256(m *[16]uint32) {
	al, bl, cl, dl := d.h[0], d.h[1], d.h[2], d.h[3]
	ar, br, cr, dr := d.h[4], d.h[5], d.h[6], d.h[7]
	for i := range 64 {
		t := bits.RotateLeft32(al+m[ripZL[i]]+ripT(i, bl, cl, dl), ripSL[i])
		al, dl, cl, bl = dl, cl, bl, t
		t = bits.RotateLeft32(ar+m[ripZR[i]]+ripT64(i, br, cr, dr), ripSR[i])
		ar, dr, cr, br = dr, cr, br, t
		switch i {
		case 15:
			al, ar = ar, al
		case 31:
			bl, br = br, bl
		case 47:
			cl, cr = cr, cl
		case 63:
			dl, dr = dr, dl
		}
	}
	d.h[0] += al
	d.h[1] += bl
	d.h[2] += cl
	d.h[3] += dl
	d.h[4] += ar
	d.h[5] += br
	d.h[6] += cr
	d.h[7] += dr
}

func (d *ripemd) block320(m *[16]uint32) {
	al, bl, cl, dl, el := d.h[0], d.h[1], d.h[2], d.h[3], d.h[4]
	ar, br, cr, dr, er := d.h[5], d.h[6], d.h[7], d.h[8], d.h[9]
	for i := range 80 {
		t := bits.RotateLeft32(al+m[ripZL[i]]+ripT(i, bl, cl, dl), ripSL[i]) + el
		al, el, dl, cl, bl = el, dl, bits.RotateLeft32(cl, 10), bl, t
		t = bits.RotateLeft32(ar+m[ripZR[i]]+ripT80(i, br, cr, dr), ripSR[i]) + er
		ar, er, dr, cr, br = er, dr, bits.RotateLeft32(cr, 10), br, t
		switch i {
		case 15:
			bl, br = br, bl
		case 31:
			dl, dr = dr, dl
		case 47:
			al, ar = ar, al
		case 63:
			cl, cr = cr, cl
		case 79:
			el, er = er, el
		}
	}
	d.h[0] += al
	d.h[1] += bl
	d.h[2] += cl
	d.h[3] += dl
	d.h[4] += el
	d.h[5] += ar
	d.h[6] += br
	d.h[7] += cr
	d.h[8] += dr
	d.h[9] += er
}
