package ops

// Pure-Go MD6, reimplemented from the self-contained node-md6 package CyberChef's
// MD6 operation wraps (a standard MD6 implementation). MD6 is not in Go's
// x/crypto. The reference uses two 32-bit halves to emulate 64-bit words; Go's
// uint64 replaces that directly, since every operation is a standard 64-bit
// logical shift / XOR / AND.

import (
	"encoding/binary"
	"math"
	"unicode/utf16"
)

const (
	md6B = 512 // block size in bytes
	md6C = 128 // chaining size in bytes
	md6N = 89  // number of 64-bit words the compression function consumes
)

// md6S0 / md6Sm are the round-constant recurrence seeds.
const (
	md6S0 = 0x0123456789abcdef
	md6Sm = 0x7311c2812425cfa0
)

// md6Q is the 960-bit constant Q (15 words).
var md6Q = [15]uint64{
	0x7311c2812425cfa0, 0x6432286434aac8e7, 0xb60450e9ef68b7c1,
	0xe8fb23908d9f06f1, 0xdd2e76cba691e5bf, 0x0cd0d63b2c30bc41,
	0x1f8ccf6823058f8a, 0x54e5ed5b88e3775d, 0x4ad12aae0a6d6031,
	0x3e7f16bb88222e0d, 0x8af8671d3fb50c2c, 0x995ad1178bd25c31,
	0xc878c1dd04c4b633, 0x3b72066c7a1552ac, 0x0d6f3522631effcb,
}

// md6T are the tap positions, md6RS/md6LS the per-step shift amounts.
var (
	md6T  = [6]int{17, 18, 21, 31, 67, 89}
	md6RS = [16]uint{10, 5, 13, 10, 11, 12, 2, 7, 14, 15, 7, 13, 11, 7, 6, 12}
	md6LS = [16]uint{11, 24, 9, 16, 15, 9, 27, 15, 6, 2, 29, 8, 15, 5, 31, 9}
)

// md6State carries the per-hash parameters shared by the compression phases.
type md6State struct {
	d      int     // digest size in bits
	k      int     // key length in bytes (0..64)
	rounds float64 // compression rounds (may be fractional, as in the reference)
	levels int     // maximum tree height before switching to sequential mode
	ell    int     // current tree level
	key    [8]uint64
}

// md6Bytes reproduces node-md6's charCodeAt-based UTF-8 encoding: it iterates
// UTF-16 code units (so non-BMP input is encoded per surrogate, like the package).
func md6Bytes(s string) []byte {
	var out []byte
	for _, ch := range utf16.Encode([]rune(s)) {
		switch {
		case ch <= 0x7f:
			out = append(out, byte(ch))
		case ch <= 0x7ff:
			out = append(out, byte(ch>>6)|0xc0, byte(ch&0x3f)|0x80)
		default:
			out = append(out, byte(ch>>12)|0xe0, byte((ch>>6)&0x3f)|0x80, byte(ch&0x3f)|0x80)
		}
	}
	return out
}

// md6ToWords reads bytes as big-endian 64-bit words (len must be a multiple of 8).
func md6ToWords(b []byte) []uint64 {
	out := make([]uint64, len(b)/8)
	for i := range out {
		out[i] = binary.BigEndian.Uint64(b[i*8:])
	}
	return out
}

// md6FromWords serialises words back to big-endian bytes.
func md6FromWords(w []uint64) []byte {
	out := make([]byte, len(w)*8)
	for i, v := range w {
		binary.BigEndian.PutUint64(out[i*8:], v)
	}
	return out
}

// f is the MD6 compression function over an md6N-word input.
func (s *md6State) f(n []uint64) []uint64 {
	rounds := int(math.Ceil(s.rounds))
	a := make([]uint64, md6N+16*rounds)
	copy(a, n)
	rc := uint64(md6S0)
	for j, i := 0, md6N; j < rounds; j, i = j+1, i+16 {
		for step := range 16 {
			x := rc
			x ^= a[i+step-md6T[5]]
			x ^= a[i+step-md6T[0]]
			x ^= a[i+step-md6T[1]] & a[i+step-md6T[2]]
			x ^= a[i+step-md6T[3]] & a[i+step-md6T[4]]
			x ^= x >> md6RS[step]
			a[i+step] = x ^ (x << md6LS[step])
		}
		rc = (rc << 1) ^ (rc >> 63) ^ (rc & md6Sm)
	}
	return a[len(a)-16:]
}

// mid builds the two control words U and V and compresses one node.
func (s *md6State) mid(b, c []uint64, i, p, z int) []uint64 {
	u := (uint64(s.ell&0xff)<<24|(uint64(i)>>32)&0xffffff)<<32 | uint64(i)&0xffffffff // #nosec G115 -- masked fields of the U control word

	r := int(s.rounds)
	vHi := uint64((r&0xfff)<<16 | (s.levels&0xff)<<8 | (z&0xf)<<4 | (p&0xf000)>>12) // #nosec G115 -- masked fields of the V control word
	vLo := uint64((p&0xfff)<<20 | (s.k&0xff)<<12 | (s.d & 0xfff))                   // #nosec G115 -- masked fields of the V control word
	v := vHi<<32 | vLo

	n := make([]uint64, 0, md6N)
	n = append(n, md6Q[:]...)
	n = append(n, s.key[:]...)
	n = append(n, u, v)
	n = append(n, c...)
	n = append(n, b...)
	return s.f(n)
}

// pad appends 0x00 bytes until the length is a positive multiple of blockBytes,
// returning the padded data and the number of padding bits added.
func md6Pad(m []byte, blockBytes int) ([]byte, int) {
	p := 0
	for len(m) < 1 || len(m)%blockBytes > 0 {
		m = append(m, 0)
		p += 8
	}
	return m, p
}

// par is one parallel compression pass over the message bytes.
func (s *md6State) par(m []byte) []byte {
	z := 0
	if len(m) <= md6B {
		z = 1
	}
	m, p := md6Pad(m, md6B)
	words := md6ToWords(m)

	const blockWords = md6B / 8
	var c []uint64
	for i := 0; i*blockWords < len(words); i++ {
		block := words[i*blockWords : (i+1)*blockWords]
		pad := 0
		if (i+1)*blockWords >= len(words) {
			pad = p
		}
		c = append(c, s.mid(block, nil, i, pad, z)...)
	}
	return md6FromWords(c)
}

// seq is one sequential compression pass, chaining through the c-word state.
func (s *md6State) seq(m []byte) []byte {
	m, p := md6Pad(m, md6B-md6C)
	words := md6ToWords(m)

	const blockWords = (md6B - md6C) / 8
	c := make([]uint64, md6C/8)
	blocks := len(words) / blockWords
	for i := range blocks {
		block := words[i*blockWords : (i+1)*blockWords]
		pad, z := 0, 0
		if i == blocks-1 {
			pad, z = p, 1
		}
		c = s.mid(block, c, i, pad, z)
	}
	return md6FromWords(c)
}

// md6Crop keeps the rightmost size bits of the hash, masking the final byte.
func md6Crop(size int, hash []byte) []byte {
	length := (size + 7) / 8
	out := append([]byte{}, hash[len(hash)-length:]...)
	if remain := size % 8; remain > 0 {
		out[length-1] &= byte(0xff<<(8-remain)) & 0xff
	}
	return out
}

// md6Hash computes the MD6 digest of data with the given parameters.
func md6Hash(size int, data, key []byte, levels int) []byte {
	if size <= 0 {
		size = 1
	}
	k := min(len(key), 64)
	keyBytes := make([]byte, 64)
	copy(keyBytes, key[:k])

	s := &md6State{
		d:      size,
		k:      k,
		levels: levels,
	}
	kb := 0.0
	if k > 0 {
		kb = 80
	}
	s.rounds = max(kb, 40+float64(size)/4)
	kw := md6ToWords(keyBytes)
	copy(s.key[:], kw)

	m := data
	for {
		s.ell++
		if s.ell > levels {
			m = s.seq(m)
		} else {
			m = s.par(m)
		}
		if len(m) == md6C {
			break
		}
	}
	return md6Crop(size, m)
}
