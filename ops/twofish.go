package ops

import (
	"encoding/hex"
	"fmt"
	"math/bits"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(TwofishEncrypt{})
	core.Register(TwofishDecrypt{})
}

// twofishDescription is shared by both operations.
const twofishDescription = "Twofish is a symmetric key block cipher designed by Bruce Schneier. It was one of the five AES finalists. Twofish operates on 128-bit blocks and supports key sizes of 128, 192, or 256 bits with 16 rounds of a Feistel network. When using CBC or ECB mode, the PKCS#7 padding scheme is used."

// twofishRounds is the number of Feistel rounds.
const twofishRounds = 16

// twofishBlockSize is the Twofish block size in bytes (128 bits).
const twofishBlockSize = 16

// twofishQ0 is the Q0 permutation table.
var twofishQ0 = [256]byte{
	0xa9, 0x67, 0xb3, 0xe8, 0x04, 0xfd, 0xa3, 0x76, 0x9a, 0x92, 0x80, 0x78, 0xe4, 0xdd, 0xd1, 0x38,
	0x0d, 0xc6, 0x35, 0x98, 0x18, 0xf7, 0xec, 0x6c, 0x43, 0x75, 0x37, 0x26, 0xfa, 0x13, 0x94, 0x48,
	0xf2, 0xd0, 0x8b, 0x30, 0x84, 0x54, 0xdf, 0x23, 0x19, 0x5b, 0x3d, 0x59, 0xf3, 0xae, 0xa2, 0x82,
	0x63, 0x01, 0x83, 0x2e, 0xd9, 0x51, 0x9b, 0x7c, 0xa6, 0xeb, 0xa5, 0xbe, 0x16, 0x0c, 0xe3, 0x61,
	0xc0, 0x8c, 0x3a, 0xf5, 0x73, 0x2c, 0x25, 0x0b, 0xbb, 0x4e, 0x89, 0x6b, 0x53, 0x6a, 0xb4, 0xf1,
	0xe1, 0xe6, 0xbd, 0x45, 0xe2, 0xf4, 0xb6, 0x66, 0xcc, 0x95, 0x03, 0x56, 0xd4, 0x1c, 0x1e, 0xd7,
	0xfb, 0xc3, 0x8e, 0xb5, 0xe9, 0xcf, 0xbf, 0xba, 0xea, 0x77, 0x39, 0xaf, 0x33, 0xc9, 0x62, 0x71,
	0x81, 0x79, 0x09, 0xad, 0x24, 0xcd, 0xf9, 0xd8, 0xe5, 0xc5, 0xb9, 0x4d, 0x44, 0x08, 0x86, 0xe7,
	0xa1, 0x1d, 0xaa, 0xed, 0x06, 0x70, 0xb2, 0xd2, 0x41, 0x7b, 0xa0, 0x11, 0x31, 0xc2, 0x27, 0x90,
	0x20, 0xf6, 0x60, 0xff, 0x96, 0x5c, 0xb1, 0xab, 0x9e, 0x9c, 0x52, 0x1b, 0x5f, 0x93, 0x0a, 0xef,
	0x91, 0x85, 0x49, 0xee, 0x2d, 0x4f, 0x8f, 0x3b, 0x47, 0x87, 0x6d, 0x46, 0xd6, 0x3e, 0x69, 0x64,
	0x2a, 0xce, 0xcb, 0x2f, 0xfc, 0x97, 0x05, 0x7a, 0xac, 0x7f, 0xd5, 0x1a, 0x4b, 0x0e, 0xa7, 0x5a,
	0x28, 0x14, 0x3f, 0x29, 0x88, 0x3c, 0x4c, 0x02, 0xb8, 0xda, 0xb0, 0x17, 0x55, 0x1f, 0x8a, 0x7d,
	0x57, 0xc7, 0x8d, 0x74, 0xb7, 0xc4, 0x9f, 0x72, 0x7e, 0x15, 0x22, 0x12, 0x58, 0x07, 0x99, 0x34,
	0x6e, 0x50, 0xde, 0x68, 0x65, 0xbc, 0xdb, 0xf8, 0xc8, 0xa8, 0x2b, 0x40, 0xdc, 0xfe, 0x32, 0xa4,
	0xca, 0x10, 0x21, 0xf0, 0xd3, 0x5d, 0x0f, 0x00, 0x6f, 0x9d, 0x36, 0x42, 0x4a, 0x5e, 0xc1, 0xe0,
}

// twofishQ1 is the Q1 permutation table.
var twofishQ1 = [256]byte{
	0x75, 0xf3, 0xc6, 0xf4, 0xdb, 0x7b, 0xfb, 0xc8, 0x4a, 0xd3, 0xe6, 0x6b, 0x45, 0x7d, 0xe8, 0x4b,
	0xd6, 0x32, 0xd8, 0xfd, 0x37, 0x71, 0xf1, 0xe1, 0x30, 0x0f, 0xf8, 0x1b, 0x87, 0xfa, 0x06, 0x3f,
	0x5e, 0xba, 0xae, 0x5b, 0x8a, 0x00, 0xbc, 0x9d, 0x6d, 0xc1, 0xb1, 0x0e, 0x80, 0x5d, 0xd2, 0xd5,
	0xa0, 0x84, 0x07, 0x14, 0xb5, 0x90, 0x2c, 0xa3, 0xb2, 0x73, 0x4c, 0x54, 0x92, 0x74, 0x36, 0x51,
	0x38, 0xb0, 0xbd, 0x5a, 0xfc, 0x60, 0x62, 0x96, 0x6c, 0x42, 0xf7, 0x10, 0x7c, 0x28, 0x27, 0x8c,
	0x13, 0x95, 0x9c, 0xc7, 0x24, 0x46, 0x3b, 0x70, 0xca, 0xe3, 0x85, 0xcb, 0x11, 0xd0, 0x93, 0xb8,
	0xa6, 0x83, 0x20, 0xff, 0x9f, 0x77, 0xc3, 0xcc, 0x03, 0x6f, 0x08, 0xbf, 0x40, 0xe7, 0x2b, 0xe2,
	0x79, 0x0c, 0xaa, 0x82, 0x41, 0x3a, 0xea, 0xb9, 0xe4, 0x9a, 0xa4, 0x97, 0x7e, 0xda, 0x7a, 0x17,
	0x66, 0x94, 0xa1, 0x1d, 0x3d, 0xf0, 0xde, 0xb3, 0x0b, 0x72, 0xa7, 0x1c, 0xef, 0xd1, 0x53, 0x3e,
	0x8f, 0x33, 0x26, 0x5f, 0xec, 0x76, 0x2a, 0x49, 0x81, 0x88, 0xee, 0x21, 0xc4, 0x1a, 0xeb, 0xd9,
	0xc5, 0x39, 0x99, 0xcd, 0xad, 0x31, 0x8b, 0x01, 0x18, 0x23, 0xdd, 0x1f, 0x4e, 0x2d, 0xf9, 0x48,
	0x4f, 0xf2, 0x65, 0x8e, 0x78, 0x5c, 0x58, 0x19, 0x8d, 0xe5, 0x98, 0x57, 0x67, 0x7f, 0x05, 0x64,
	0xaf, 0x63, 0xb6, 0xfe, 0xf5, 0xb7, 0x3c, 0xa5, 0xce, 0xe9, 0x68, 0x44, 0xe0, 0x4d, 0x43, 0x69,
	0x29, 0x2e, 0xac, 0x15, 0x59, 0xa8, 0x0a, 0x9e, 0x6e, 0x47, 0xdf, 0x34, 0x35, 0x6a, 0xcf, 0xdc,
	0x22, 0xc9, 0xc0, 0x9b, 0x89, 0xd4, 0xed, 0xab, 0x12, 0xa2, 0x0d, 0x52, 0xbb, 0x02, 0x2f, 0xa9,
	0xd7, 0x61, 0x1e, 0xb4, 0x50, 0x04, 0xf6, 0xc2, 0x16, 0x25, 0x86, 0x56, 0x55, 0x09, 0xbe, 0x91,
}

// twofishRS is the Reed-Solomon matrix used in the key schedule.
var twofishRS = [4][8]byte{
	{0x01, 0xa4, 0x55, 0x87, 0x5a, 0x58, 0xdb, 0x9e},
	{0xa4, 0x56, 0x82, 0xf3, 0x1e, 0xc6, 0x68, 0xe5},
	{0x02, 0xa1, 0xfc, 0xc1, 0x47, 0xae, 0x3d, 0x19},
	{0xa4, 0x55, 0x87, 0x5a, 0x58, 0xdb, 0x9e, 0x03},
}

// twofishMDSPoly is the GF(2^8) reduction polynomial for the MDS matrix.
const twofishMDSPoly = 0x169

// twofishRSPoly is the GF(2^8) reduction polynomial for the Reed-Solomon step.
const twofishRSPoly = 0x14d

// twofishKeyData holds the derived key schedule: 40 round subkeys, the k S-box
// key words, and k = keyLen/8 (2, 3, or 4).
type twofishKeyData struct {
	subkeys [40]uint32
	s       []uint32
	k       int
}

// twofishGFMult multiplies a and b in GF(2^8) modulo poly.
func twofishGFMult(a, b, poly uint32) uint32 {
	var result uint32
	for b != 0 {
		if b&1 != 0 {
			result ^= a
		}
		a <<= 1
		if a&0x100 != 0 {
			a ^= poly
		}
		b >>= 1
	}
	return result & 0xff
}

// twofishMDSMultiply applies the MDS matrix to the four bytes of x.
func twofishMDSMultiply(x uint32) uint32 {
	b0, b1, b2, b3 := x&0xff, (x>>8)&0xff, (x>>16)&0xff, (x>>24)&0xff
	m := func(v, c uint32) uint32 { return twofishGFMult(v, c, twofishMDSPoly) }
	r0 := m(b0, 0x01) ^ m(b1, 0xef) ^ m(b2, 0x5b) ^ m(b3, 0x5b)
	r1 := m(b0, 0x5b) ^ m(b1, 0xef) ^ m(b2, 0xef) ^ m(b3, 0x01)
	r2 := m(b0, 0xef) ^ m(b1, 0x5b) ^ m(b2, 0x01) ^ m(b3, 0xef)
	r3 := m(b0, 0xef) ^ m(b1, 0x01) ^ m(b2, 0xef) ^ m(b3, 0x5b)
	return r3<<24 | r2<<16 | r1<<8 | r0
}

// twofishRSMultiply reduces the eight key bytes to one word via the RS matrix.
func twofishRSMultiply(key8 []byte) uint32 {
	var result uint32
	for i := range 4 {
		var x uint32
		for j := range 8 {
			x ^= twofishGFMult(uint32(twofishRS[i][j]), uint32(key8[j]), twofishRSPoly)
		}
		result |= x << (i * 8)
	}
	return result
}

// twofishH is the keyed permutation h; l is either the key words (Me/Mo) during
// the key schedule or the S-box key words during encryption, with k of them.
func twofishH(x uint32, l []uint32, k int) uint32 {
	y0, y1, y2, y3 := x&0xff, (x>>8)&0xff, (x>>16)&0xff, (x>>24)&0xff
	q0, q1 := &twofishQ0, &twofishQ1
	if k == 4 {
		y0 = uint32(q1[y0]) ^ (l[3] & 0xff)
		y1 = uint32(q0[y1]) ^ ((l[3] >> 8) & 0xff)
		y2 = uint32(q0[y2]) ^ ((l[3] >> 16) & 0xff)
		y3 = uint32(q1[y3]) ^ ((l[3] >> 24) & 0xff)
	}
	if k >= 3 {
		y0 = uint32(q1[y0]) ^ (l[2] & 0xff)
		y1 = uint32(q1[y1]) ^ ((l[2] >> 8) & 0xff)
		y2 = uint32(q0[y2]) ^ ((l[2] >> 16) & 0xff)
		y3 = uint32(q0[y3]) ^ ((l[2] >> 24) & 0xff)
	}
	y0 = uint32(q0[uint32(q0[y0])^(l[1]&0xff)]) ^ (l[0] & 0xff)
	y1 = uint32(q0[uint32(q1[y1])^((l[1]>>8)&0xff)]) ^ ((l[0] >> 8) & 0xff)
	y2 = uint32(q1[uint32(q0[y2])^((l[1]>>16)&0xff)]) ^ ((l[0] >> 16) & 0xff)
	y3 = uint32(q1[uint32(q1[y3])^((l[1]>>24)&0xff)]) ^ ((l[0] >> 24) & 0xff)
	y0 = uint32(q1[y0])
	y1 = uint32(q0[y1])
	y2 = uint32(q1[y2])
	y3 = uint32(q0[y3])
	return twofishMDSMultiply(y3<<24 | y2<<16 | y1<<8 | y0)
}

// twofishWord reads four little-endian bytes of b at offset as a word.
func twofishWord(b []byte, offset int) uint32 {
	return uint32(b[offset]) | uint32(b[offset+1])<<8 | uint32(b[offset+2])<<16 | uint32(b[offset+3])<<24
}

// twofishGenerateSubkeys derives the key schedule from a 16/24/32-byte key.
func twofishGenerateSubkeys(key []byte) twofishKeyData {
	k := len(key) / 8
	me := make([]uint32, k)
	mo := make([]uint32, k)
	for i := range k {
		me[i] = twofishWord(key, i*8)
		mo[i] = twofishWord(key, i*8+4)
	}
	s := make([]uint32, k)
	for i := range k {
		offset := (k - 1 - i) * 8
		s[i] = twofishRSMultiply(key[offset : offset+8])
	}
	const rho = 0x01010101
	var subkeys [40]uint32
	for i := range 20 {
		a := twofishH(uint32(2*i)*rho, me, k)
		b := bits.RotateLeft32(twofishH(uint32(2*i+1)*rho, mo, k), 8)
		subkeys[2*i] = a + b
		subkeys[2*i+1] = bits.RotateLeft32(a+2*b, 9)
	}
	return twofishKeyData{subkeys: subkeys, s: s, k: k}
}

// twofishBlockToBytes converts the four output words to little-endian bytes,
// applying the Twofish final swap (R2, R3, R0, R1).
func twofishBlockToBytes(r2, r3, r0, r1 uint32) []byte {
	out := make([]byte, 0, twofishBlockSize)
	for _, w := range [4]uint32{r2, r3, r0, r1} {
		// #nosec G115 -- deliberate little-endian byte extraction from a 32-bit word
		out = append(out, byte(w), byte(w>>8), byte(w>>16), byte(w>>24))
	}
	return out
}

// twofishEncryptBlock enciphers one 16-byte block.
func twofishEncryptBlock(block []byte, kd twofishKeyData) []byte {
	sk := &kd.subkeys
	r0 := twofishWord(block, 0) ^ sk[0]
	r1 := twofishWord(block, 4) ^ sk[1]
	r2 := twofishWord(block, 8) ^ sk[2]
	r3 := twofishWord(block, 12) ^ sk[3]
	for r := 0; r < twofishRounds; r += 2 {
		t0 := twofishH(r0, kd.s, kd.k)
		t1 := twofishH(bits.RotateLeft32(r1, 8), kd.s, kd.k)
		r2 = bits.RotateLeft32(r2^(t0+t1+sk[8+2*r]), -1)
		r3 = bits.RotateLeft32(r3, 1) ^ (t0 + 2*t1 + sk[9+2*r])
		t0 = twofishH(r2, kd.s, kd.k)
		t1 = twofishH(bits.RotateLeft32(r3, 8), kd.s, kd.k)
		r0 = bits.RotateLeft32(r0^(t0+t1+sk[8+2*r+2]), -1)
		r1 = bits.RotateLeft32(r1, 1) ^ (t0 + 2*t1 + sk[9+2*r+2])
	}
	return twofishBlockToBytes(r2^sk[4], r3^sk[5], r0^sk[6], r1^sk[7])
}

// twofishDecryptBlock deciphers one 16-byte block.
func twofishDecryptBlock(block []byte, kd twofishKeyData) []byte {
	sk := &kd.subkeys
	r0 := twofishWord(block, 0) ^ sk[4]
	r1 := twofishWord(block, 4) ^ sk[5]
	r2 := twofishWord(block, 8) ^ sk[6]
	r3 := twofishWord(block, 12) ^ sk[7]
	for r := twofishRounds - 2; r >= 0; r -= 2 {
		t0 := twofishH(r0, kd.s, kd.k)
		t1 := twofishH(bits.RotateLeft32(r1, 8), kd.s, kd.k)
		r2 = bits.RotateLeft32(r2, 1) ^ (t0 + t1 + sk[8+2*r+2])
		r3 = bits.RotateLeft32(r3^(t0+2*t1+sk[9+2*r+2]), -1)
		t0 = twofishH(r2, kd.s, kd.k)
		t1 = twofishH(bits.RotateLeft32(r3, 8), kd.s, kd.k)
		r0 = bits.RotateLeft32(r0, 1) ^ (t0 + t1 + sk[8+2*r])
		r1 = bits.RotateLeft32(r1^(t0+2*t1+sk[9+2*r]), -1)
	}
	return twofishBlockToBytes(r2^sk[0], r3^sk[1], r0^sk[2], r1^sk[3])
}

// twofishXorBlocks returns the byte-wise XOR of two 16-byte blocks.
func twofishXorBlocks(a, b []byte) []byte {
	out := make([]byte, twofishBlockSize)
	for i := range out {
		out[i] = a[i] ^ b[i]
	}
	return out
}

// twofishIncrementCounter increments the little-endian counter block by one.
func twofishIncrementCounter(counter []byte) []byte {
	out := append([]byte{}, counter...)
	for i := range out {
		out[i]++
		if out[i] != 0 {
			break
		}
	}
	return out
}

// twofishECBCrypt applies the block permutation to every block; enc selects
// encryption (forward) or decryption (inverse).
func twofishECBCrypt(data []byte, kd twofishKeyData, enc bool) []byte {
	out := make([]byte, 0, len(data))
	for i := 0; i+twofishBlockSize <= len(data); i += twofishBlockSize {
		block := data[i : i+twofishBlockSize]
		if enc {
			out = append(out, twofishEncryptBlock(block, kd)...)
		} else {
			out = append(out, twofishDecryptBlock(block, kd)...)
		}
	}
	return out
}

// twofishCBCEncrypt encrypts in CBC mode.
func twofishCBCEncrypt(data []byte, kd twofishKeyData, iv []byte) []byte {
	out := make([]byte, 0, len(data))
	prev := iv
	for i := 0; i+twofishBlockSize <= len(data); i += twofishBlockSize {
		prev = twofishEncryptBlock(twofishXorBlocks(data[i:i+twofishBlockSize], prev), kd)
		out = append(out, prev...)
	}
	return out
}

// twofishCBCDecrypt decrypts in CBC mode.
func twofishCBCDecrypt(data []byte, kd twofishKeyData, iv []byte) []byte {
	out := make([]byte, 0, len(data))
	prev := iv
	for i := 0; i+twofishBlockSize <= len(data); i += twofishBlockSize {
		block := data[i : i+twofishBlockSize]
		out = append(out, twofishXorBlocks(twofishDecryptBlock(block, kd), prev)...)
		prev = block
	}
	return out
}

// twofishCFBEncrypt encrypts in CFB mode.
func twofishCFBEncrypt(data []byte, kd twofishKeyData, iv []byte) []byte {
	out := make([]byte, 0, len(data))
	prev := iv
	for i := 0; i+twofishBlockSize <= len(data); i += twofishBlockSize {
		prev = twofishXorBlocks(twofishEncryptBlock(prev, kd), data[i:i+twofishBlockSize])
		out = append(out, prev...)
	}
	return out
}

// twofishCFBDecrypt decrypts in CFB mode (the keystream uses the forward cipher).
func twofishCFBDecrypt(data []byte, kd twofishKeyData, iv []byte) []byte {
	out := make([]byte, 0, len(data))
	prev := iv
	for i := 0; i+twofishBlockSize <= len(data); i += twofishBlockSize {
		block := data[i : i+twofishBlockSize]
		out = append(out, twofishXorBlocks(twofishEncryptBlock(prev, kd), block)...)
		prev = block
	}
	return out
}

// twofishOFBCrypt applies OFB mode (identical for encryption and decryption).
func twofishOFBCrypt(data []byte, kd twofishKeyData, iv []byte) []byte {
	out := make([]byte, 0, len(data))
	feedback := iv
	for i := 0; i+twofishBlockSize <= len(data); i += twofishBlockSize {
		feedback = twofishEncryptBlock(feedback, kd)
		out = append(out, twofishXorBlocks(feedback, data[i:i+twofishBlockSize])...)
	}
	return out
}

// twofishCTRCrypt applies CTR mode (identical for encryption and decryption).
func twofishCTRCrypt(data []byte, kd twofishKeyData, iv []byte) []byte {
	out := make([]byte, 0, len(data))
	counter := iv
	for i := 0; i+twofishBlockSize <= len(data); i += twofishBlockSize {
		out = append(out, twofishXorBlocks(twofishEncryptBlock(counter, kd), data[i:i+twofishBlockSize])...)
		counter = twofishIncrementCounter(counter)
	}
	return out
}

// twofishPadStream zero-pads data up to a whole number of blocks (for the stream
// modes, which then slice their output back to the original length).
func twofishPadStream(data []byte) []byte {
	out := append([]byte{}, data...)
	for len(out)%twofishBlockSize != 0 {
		out = append(out, 0)
	}
	return out
}

// twofishEncrypt enciphers message with the given key, IV, mode, and padding.
func twofishEncrypt(message, key, iv []byte, mode, padding string) ([]byte, error) {
	if len(message) == 0 {
		return []byte{}, nil
	}
	kd := twofishGenerateSubkeys(key)
	msgLen := len(message)

	padded := message
	if mode == "ECB" || mode == "CBC" {
		var err error
		if padded, err = blockApplyPadding(message, padding, twofishBlockSize); err != nil {
			return nil, err
		}
	}

	var cipherText []byte
	switch mode {
	case "ECB":
		cipherText = twofishECBCrypt(padded, kd, true)
	case "CBC":
		cipherText = twofishCBCEncrypt(padded, kd, iv)
	case "CFB":
		cipherText = twofishCFBEncrypt(twofishPadStream(padded), kd, iv)
	case "OFB":
		cipherText = twofishOFBCrypt(twofishPadStream(padded), kd, iv)
	case "CTR":
		cipherText = twofishCTRCrypt(twofishPadStream(padded), kd, iv)
	}
	if mode == "ECB" || mode == "CBC" {
		return cipherText, nil
	}
	return cipherText[:msgLen], nil
}

// twofishDecrypt deciphers cipherText with the given key, IV, mode, and padding.
func twofishDecrypt(cipherText, key, iv []byte, mode, padding string) ([]byte, error) {
	originalLength := len(cipherText)
	if originalLength == 0 {
		return []byte{}, nil
	}
	kd := twofishGenerateSubkeys(key)

	data := cipherText
	if mode == "ECB" || mode == "CBC" {
		if originalLength%twofishBlockSize != 0 {
			//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
			return nil, fmt.Errorf("Invalid ciphertext length: %d bytes. Must be a multiple of 16.", originalLength)
		}
	} else {
		data = twofishPadStream(cipherText)
	}

	var plainText []byte
	switch mode {
	case "ECB":
		plainText = twofishECBCrypt(data, kd, false)
	case "CBC":
		plainText = twofishCBCDecrypt(data, kd, iv)
	case "CFB":
		plainText = twofishCFBDecrypt(data, kd, iv)
	case "OFB":
		plainText = twofishOFBCrypt(data, kd, iv)
	case "CTR":
		plainText = twofishCTRCrypt(data, kd, iv)
	}
	if mode == "ECB" || mode == "CBC" {
		return blockRemovePadding(plainText, padding, twofishBlockSize)
	}
	return plainText[:originalLength], nil
}

// twofishArgs builds the shared argument list, differing only in the option
// defaults for Input/Output (encrypt takes Raw/Hex, decrypt takes Hex/Raw).
func twofishArgs(inputVals, outputVals []string) []core.ArgDef {
	return []core.ArgDef{
		{Name: "Key", Type: core.ArgToggleString, Value: "", ToggleValues: aesToggleValues},
		{Name: "IV", Type: core.ArgToggleString, Value: "", ToggleValues: aesToggleValues},
		{Name: "Mode", Type: core.ArgOption, Value: []string{"CBC", "CFB", "OFB", "CTR", "ECB"}},
		{Name: "Input", Type: core.ArgOption, Value: inputVals},
		{Name: "Output", Type: core.ArgOption, Value: outputVals},
		{Name: "Padding", Type: core.ArgOption, Value: []string{"PKCS5", "NO", "ZERO", "RANDOM", "BIT"}},
	}
}

// twofishInputs parses and validates the shared key/IV/input arguments.
func twofishInputs(in *core.Dish, args []any) (key, iv, input []byte, mode string, err error) {
	ks := args[0].(core.ToggleString)
	ivs := args[1].(core.ToggleString)
	mode = args[2].(string)
	if key, err = convertToByteArray(ks.Value, ks.Option); err != nil {
		return
	}
	if iv, err = convertToByteArray(ivs.Value, ivs.Option); err != nil {
		return
	}
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		err = fmt.Errorf("Invalid key length: %d bytes\n\nTwofish uses a key length of 16 bytes (128 bits), 24 bytes (192 bits), or 32 bytes (256 bits).", len(key))
		return
	}
	if len(iv) != 16 && mode != "ECB" {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		err = fmt.Errorf("Invalid IV length: %d bytes\n\nTwofish uses an IV length of 16 bytes (128 bits).\nMake sure you have specified the type correctly (e.g. Hex vs UTF8).", len(iv))
		return
	}
	input = decodeAESInput(in, args[3].(string))
	return
}

// twofishOutput formats the result as continuous hex or a raw string.
func twofishOutput(out []byte, outType string) *core.Dish {
	if outType == "Hex" {
		return core.NewDish([]byte(hex.EncodeToString(out)), core.TypeString)
	}
	return core.NewDish([]byte(byteArrayToUtf8(out)), core.TypeString)
}

// TwofishEncrypt encrypts with the Twofish block cipher.
type TwofishEncrypt struct{}

// Meta returns the operation metadata.
func (TwofishEncrypt) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Twofish Encrypt",
		Module:      "Ciphers",
		Description: twofishDescription,
		InfoURL:     "https://wikipedia.org/wiki/Twofish",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (TwofishEncrypt) Args() []core.ArgDef {
	return twofishArgs([]string{"Raw", "Hex"}, []string{"Hex", "Raw"})
}

// Run encrypts with Twofish. Ported from CyberChef TwofishEncrypt.mjs.
func (TwofishEncrypt) Run(in *core.Dish, args []any) (*core.Dish, error) {
	key, iv, input, mode, err := twofishInputs(in, args)
	if err != nil {
		return nil, err
	}
	out, err := twofishEncrypt(input, key, iv, mode, args[5].(string))
	if err != nil {
		return nil, err
	}
	return twofishOutput(out, args[4].(string)), nil
}

// TwofishDecrypt decrypts with the Twofish block cipher.
type TwofishDecrypt struct{}

// Meta returns the operation metadata.
func (TwofishDecrypt) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Twofish Decrypt",
		Module:      "Ciphers",
		Description: twofishDescription,
		InfoURL:     "https://wikipedia.org/wiki/Twofish",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (TwofishDecrypt) Args() []core.ArgDef {
	return twofishArgs([]string{"Hex", "Raw"}, []string{"Raw", "Hex"})
}

// Run decrypts with Twofish. Ported from CyberChef TwofishDecrypt.mjs.
func (TwofishDecrypt) Run(in *core.Dish, args []any) (*core.Dish, error) {
	key, iv, input, mode, err := twofishInputs(in, args)
	if err != nil {
		return nil, err
	}
	out, err := twofishDecrypt(input, key, iv, mode, args[5].(string))
	if err != nil {
		return nil, err
	}
	return twofishOutput(out, args[4].(string)), nil
}
