package ops

import (
	"encoding/binary"
	"hash"
	"math/bits"
)

// Whirlpool and its two earlier variants (Whirlpool-0, Whirlpool-T), ported from
// crypto-api's whirlpool hasher. crypto-api uses a 64-bit length field (not the
// standard 256-bit), so these match crypto-api/CyberChef, not the ISO reference
// vectors. State is kept as 16 uint32 (eight 64-bit words as hi/lo pairs) exactly
// as the source.

var (
	whirlEBox   = [16]uint32{0x1, 0xb, 0x9, 0xc, 0xd, 0x6, 0xf, 0x3, 0xe, 0x8, 0x7, 0x4, 0xa, 0x2, 0x5, 0x0}
	whirlRBox   = [16]uint32{0x7, 0xc, 0xb, 0xd, 0xe, 0x4, 0x9, 0xf, 0x6, 0x3, 0x8, 0xa, 0x2, 0x5, 0x1, 0x0}
	whirlTheta  = [8]int{1, 1, 4, 1, 8, 5, 2, 9}
	whirlTheta0 = [8]int{1, 1, 3, 1, 5, 8, 9, 5}
	whirlSBox0  = [256]uint32{
		0x68, 0xd0, 0xeb, 0x2b, 0x48, 0x9d, 0x6a, 0xe4, 0xe3, 0xa3, 0x56, 0x81,
		0x7d, 0xf1, 0x85, 0x9e, 0x2c, 0x8e, 0x78, 0xca, 0x17, 0xa9, 0x61, 0xd5,
		0x5d, 0x0b, 0x8c, 0x3c, 0x77, 0x51, 0x22, 0x42, 0x3f, 0x54, 0x41, 0x80,
		0xcc, 0x86, 0xb3, 0x18, 0x2e, 0x57, 0x06, 0x62, 0xf4, 0x36, 0xd1, 0x6b,
		0x1b, 0x65, 0x75, 0x10, 0xda, 0x49, 0x26, 0xf9, 0xcb, 0x66, 0xe7, 0xba,
		0xae, 0x50, 0x52, 0xab, 0x05, 0xf0, 0x0d, 0x73, 0x3b, 0x04, 0x20, 0xfe,
		0xdd, 0xf5, 0xb4, 0x5f, 0x0a, 0xb5, 0xc0, 0xa0, 0x71, 0xa5, 0x2d, 0x60,
		0x72, 0x93, 0x39, 0x08, 0x83, 0x21, 0x5c, 0x87, 0xb1, 0xe0, 0x00, 0xc3,
		0x12, 0x91, 0x8a, 0x02, 0x1c, 0xe6, 0x45, 0xc2, 0xc4, 0xfd, 0xbf, 0x44,
		0xa1, 0x4c, 0x33, 0xc5, 0x84, 0x23, 0x7c, 0xb0, 0x25, 0x15, 0x35, 0x69,
		0xff, 0x94, 0x4d, 0x70, 0xa2, 0xaf, 0xcd, 0xd6, 0x6c, 0xb7, 0xf8, 0x09,
		0xf3, 0x67, 0xa4, 0xea, 0xec, 0xb6, 0xd4, 0xd2, 0x14, 0x1e, 0xe1, 0x24,
		0x38, 0xc6, 0xdb, 0x4b, 0x7a, 0x3a, 0xde, 0x5e, 0xdf, 0x95, 0xfc, 0xaa,
		0xd7, 0xce, 0x07, 0x0f, 0x3d, 0x58, 0x9a, 0x98, 0x9c, 0xf2, 0xa7, 0x11,
		0x7e, 0x8b, 0x43, 0x03, 0xe2, 0xdc, 0xe5, 0xb2, 0x4e, 0xc7, 0x6d, 0xe9,
		0x27, 0x40, 0xd8, 0x37, 0x92, 0x8f, 0x01, 0x1d, 0x53, 0x3e, 0x59, 0xc1,
		0x4f, 0x32, 0x16, 0xfa, 0x74, 0xfb, 0x63, 0x9f, 0x34, 0x1a, 0x2a, 0x5a,
		0x8d, 0xc9, 0xcf, 0xf6, 0x90, 0x28, 0x88, 0x9b, 0x31, 0x0e, 0xbd, 0x4a,
		0xe8, 0x96, 0xa6, 0x0c, 0xc8, 0x79, 0xbc, 0xbe, 0xef, 0x6e, 0x46, 0x97,
		0x5b, 0xed, 0x19, 0xd9, 0xac, 0x99, 0xa8, 0x29, 0x64, 0x1f, 0xad, 0x55,
		0x13, 0xbb, 0xf7, 0x6f, 0xb9, 0x47, 0x2f, 0xee, 0xb8, 0x7b, 0x89, 0x30,
		0xd3, 0x7f, 0x76, 0x82,
	}
)

// whirlTables holds one variant's circulant tables and round constants.
type whirlTables struct {
	C  [8][512]uint32
	RC [22]uint32
}

// whirlpoolComputedSBox is the structured S-box derived from the mini-boxes,
// used by Whirlpool and Whirlpool-T.
var whirlpoolComputedSBox = computeWhirlSBox()

func computeWhirlSBox() [256]uint32 {
	var iBox [16]uint32
	for i := range 16 {
		iBox[whirlEBox[i]] = uint32(i)
	}
	var sbox [256]uint32
	for i := range 256 {
		left := whirlEBox[i>>4]
		right := iBox[i&0xf]
		tmp := whirlRBox[left^right]
		sbox[i] = (whirlEBox[left^tmp] << 4) | iBox[right^tmp]
	}
	return sbox
}

// rr64hi/rr64lo rotate the 64-bit value (hi:lo) right by n bits, returning the
// hi/lo 32-bit half (matching crypto-api's rotateRight64hi/lo).
func rr64hi(hi, lo uint32, n uint) uint32 {
	return uint32(bits.RotateLeft64(uint64(hi)<<32|uint64(lo), -int(n)) >> 32)
}

func rr64lo(hi, lo uint32, n uint) uint32 {
	return uint32(bits.RotateLeft64(uint64(hi)<<32|uint64(lo), -int(n))) // #nosec G115 -- deliberately extracting the low 32 bits
}

// computeWhirlTables builds the C and RC tables for a variant.
func computeWhirlTables(sbox [256]uint32, theta [8]int) *whirlTables {
	t := &whirlTables{}
	for i := range 256 {
		var v [10]uint32
		v[1] = sbox[i]
		v[2] = v[1] << 1
		if v[2] >= 0x100 {
			v[2] ^= 0x11d
		}
		v[3] = v[2] ^ v[1]
		v[4] = v[2] << 1
		if v[4] >= 0x100 {
			v[4] ^= 0x11d
		}
		v[5] = v[4] ^ v[1]
		v[8] = v[4] << 1
		if v[8] >= 0x100 {
			v[8] ^= 0x11d
		}
		v[9] = v[8] ^ v[1]

		t.C[0][i*2] = v[theta[0]]<<24 | v[theta[1]]<<16 | v[theta[2]]<<8 | v[theta[3]]
		t.C[0][i*2+1] = v[theta[4]]<<24 | v[theta[5]]<<16 | v[theta[6]]<<8 | v[theta[7]]
		for r := 1; r < 8; r++ {
			t.C[r][i*2] = rr64lo(t.C[0][i*2+1], t.C[0][i*2], uint(r<<3))
			t.C[r][i*2+1] = rr64hi(t.C[0][i*2+1], t.C[0][i*2], uint(r<<3))
		}
	}
	for i := 1; i <= 10; i++ {
		t.RC[i*2] = (t.C[0][16*i-16] & 0xff000000) ^ (t.C[1][16*i-14] & 0x00ff0000) ^
			(t.C[2][16*i-12] & 0x0000ff00) ^ (t.C[3][16*i-10] & 0x000000ff)
		t.RC[i*2+1] = (t.C[4][16*i-7] & 0xff000000) ^ (t.C[5][16*i-5] & 0x00ff0000) ^
			(t.C[6][16*i-3] & 0x0000ff00) ^ (t.C[7][16*i-1] & 0x000000ff)
	}
	return t
}

var (
	whirlpoolTables  = computeWhirlTables(whirlpoolComputedSBox, whirlTheta)
	whirlpool0Tables = computeWhirlTables(whirlSBox0, whirlTheta0)
	whirlpoolTTables = computeWhirlTables(whirlpoolComputedSBox, whirlTheta0)
)

const whirlpoolRounds = 10

type whirlpool struct {
	md64
	h      [16]uint32
	tables *whirlTables
	rounds int
}

func newWhirlpool() hash.Hash { return &whirlpool{tables: whirlpoolTables, rounds: whirlpoolRounds} }

func newWhirlpool0() hash.Hash { return &whirlpool{tables: whirlpool0Tables, rounds: whirlpoolRounds} }

func newWhirlpoolT() hash.Hash { return &whirlpool{tables: whirlpoolTTables, rounds: whirlpoolRounds} }

// newWhirlpoolVariant builds the given variant ("Whirlpool"/"Whirlpool-T"/
// "Whirlpool-0") with a configurable round count (1..10).
func newWhirlpoolVariant(variant string, rounds int) hash.Hash {
	tables := whirlpoolTables
	switch variant {
	case "Whirlpool-0":
		tables = whirlpool0Tables
	case "Whirlpool-T":
		tables = whirlpoolTTables
	}
	return &whirlpool{tables: tables, rounds: rounds}
}

func (d *whirlpool) Reset()         { d.md64 = md64{}; d.h = [16]uint32{} }
func (d *whirlpool) Size() int      { return 64 }
func (d *whirlpool) BlockSize() int { return 64 }

func (d *whirlpool) Write(p []byte) (int, error) {
	d.write(p, d.block)
	return len(p), nil
}

func (d *whirlpool) Sum(in []byte) []byte {
	e := *d
	// crypto-api pads to 56 mod 64 when the remainder is < 32, else adds a whole
	// extra block, then appends a 64-bit big-endian bit-length.
	ln := e.nx
	padLen := 56 - ln
	if ln >= 32 {
		padLen = 120 - ln
	}
	buf := make([]byte, padLen+8)
	buf[0] = 0x80
	binary.BigEndian.PutUint64(buf[padLen:], e.len<<3)
	e.write(buf, e.block)

	var out [64]byte
	for i := range 16 {
		binary.BigEndian.PutUint32(out[i*4:], e.h[i])
	}
	return append(in, out[:]...)
}

func (d *whirlpool) block(p []byte) {
	var blk [16]uint32
	for i := range 16 {
		blk[i] = binary.BigEndian.Uint32(p[i*4:])
	}

	var k, state, l [16]uint32
	for i := range 16 {
		k[i] = d.h[i]
		state[i] = blk[i] ^ k[i]
	}
	for r := 1; r <= d.rounds; r++ {
		// Compute K^r from K^{r-1}.
		for i := range 8 {
			l[i*2], l[i*2+1] = 0, 0
			whirlMix(&l, &k, i, d.tables)
		}
		k = l
		k[0] ^= d.tables.RC[r*2]
		k[1] ^= d.tables.RC[r*2+1]
		// Apply the r-th round transformation to the state.
		for i := range 8 {
			l[i*2], l[i*2+1] = k[i*2], k[i*2+1]
			whirlMix(&l, &state, i, d.tables)
		}
		state = l
	}
	// Miyaguchi–Preneel.
	for i := range 16 {
		d.h[i] ^= state[i] ^ blk[i]
	}
}

// whirlMix XORs the eight circulant-table lookups for column i of src into l.
func whirlMix(l *[16]uint32, src *[16]uint32, i int, t *whirlTables) {
	for col := range 8 {
		s := 56 - 8*col
		j := 0
		if s < 32 {
			j = 1
		}
		b := (src[((i-col)&7)*2+j] >> uint(s%32)) & 0xff
		l[i*2] ^= t.C[col][b*2]
		l[i*2+1] ^= t.C[col][b*2+1]
	}
}
