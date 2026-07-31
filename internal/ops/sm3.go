package ops

import (
	"encoding/binary"
	"encoding/hex"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(SM3{})
}

// SM3 cryptographic hash (GM/T 0004-2012), a 256-bit Merkle–Damgård hash. It is
// used internally by the SM2 public-key operations (for the C3 tag and the KDF)
// and exposed as the "SM3" operation. This is a from-scratch port of the
// crypto-api SM3 module CyberChef wraps, including its configurable output
// Length and Rounds (and their unusual out-of-bounds behaviour).

// sm3IV is the SM3 initial hash value.
var sm3IV = [8]uint32{
	0x7380166f, 0x4914b2b9, 0x172442d7, 0xda8a0600,
	0xa96f30bc, 0x163138aa, 0xe38dee4d, 0xb0fb0e4e,
}

// sm3rotl rotates x left by n bits (n < 32).
func sm3rotl(x uint32, n uint) uint32 { return (x << n) | (x >> (32 - n)) }

// sm3p0 and sm3p1 are the SM3 permutation functions.
func sm3p0(x uint32) uint32 { return x ^ sm3rotl(x, 9) ^ sm3rotl(x, 17) }
func sm3p1(x uint32) uint32 { return x ^ sm3rotl(x, 15) ^ sm3rotl(x, 23) }

// sm3 block/word/digest sizes in bytes.
const (
	sm3BlockSize = 64
	sm3DigestLen = 32
)

// sm3DefaultRounds and sm3DefaultLength are the standard SM3 parameters (crypto-
// api's defaults) that the internal digest and the operation's default use.
const (
	sm3DefaultRounds = 64
	sm3DefaultLength = 256
	// sm3Words is the size of crypto-api's message-schedule array; reads past it
	// yield zero (the behaviour reproduced for rounds > 64).
	sm3Words = 132
)

// sm3Sum returns the standard 32-byte SM3 digest of msg (64 rounds), as used by
// the SM2 operations.
func sm3Sum(msg []byte) []byte {
	v := sm3State(msg, sm3DefaultRounds)
	out := make([]byte, sm3DigestLen)
	for i := range v {
		binary.BigEndian.PutUint32(out[i*4:], v[i])
	}
	return out
}

// sm3State runs the padded message through the compression function for the
// given number of rounds and returns the final chaining state.
func sm3State(msg []byte, rounds int) [8]uint32 {
	padded := sm3Pad(msg)
	v := sm3IV
	for off := 0; off < len(padded); off += sm3BlockSize {
		sm3Compress(&v, padded[off:off+sm3BlockSize], rounds)
	}
	return v
}

// sm3Hash returns the SM3 digest of msg with crypto-api's configurable output
// length (in bits) and round count. The output is floor(length/32) 32-bit words
// (a zero word count means the full digest, and words past the 8-word state are
// zero-filled), matching the operation's behaviour.
func sm3Hash(msg []byte, length, rounds int) []byte {
	v := sm3State(msg, rounds)
	words := length / 32
	if words == 0 {
		words = len(v)
	}
	if words < 0 {
		return []byte{}
	}
	out := make([]byte, words*4)
	for i := range words {
		var w uint32
		if i < len(v) {
			w = v[i]
		}
		binary.BigEndian.PutUint32(out[i*4:], w)
	}
	return out
}

// sm3Pad appends the SM3 padding: a 0x80 byte, zero bytes to a 56 mod 64
// boundary, then the 64-bit big-endian bit length.
func sm3Pad(msg []byte) []byte {
	bitLen := uint64(len(msg)) * 8
	padded := append([]byte{}, msg...)
	padded = append(padded, 0x80)
	for len(padded)%sm3BlockSize != sm3BlockSize-8 {
		padded = append(padded, 0)
	}
	var lb [8]byte
	binary.BigEndian.PutUint64(lb[:], bitLen)
	return append(padded, lb[:]...)
}

// sm3Compress applies the SM3 compression function to one 64-byte block for the
// given number of rounds, updating the running state v. The message schedule w
// holds 132 words (w[0..67] the expansion, w[68..131] the W′ terms); rounds past
// what those cover read zero, reproducing crypto-api's out-of-bounds behaviour.
func sm3Compress(v *[8]uint32, block []byte, rounds int) {
	var w [sm3Words]uint32
	for i := range 16 {
		w[i] = binary.BigEndian.Uint32(block[i*4:])
	}
	for j := 16; j < 68; j++ {
		w[j] = sm3p1(w[j-16]^w[j-9]^sm3rotl(w[j-3], 15)) ^ sm3rotl(w[j-13], 7) ^ w[j-6]
	}
	for j := 68; j < sm3Words; j++ {
		w[j] = w[j-68] ^ w[j-64]
	}

	a, b, c, d, e, f, g, h := v[0], v[1], v[2], v[3], v[4], v[5], v[6], v[7]
	for j := range rounds {
		tj := uint32(0x79cc4519)
		if j >= 16 {
			tj = 0x7a879d8a
		}
		ss1 := sm3rotl(sm3rotl(a, 12)+e+sm3rotl(tj, uint(j)%32), 7)
		ss2 := ss1 ^ sm3rotl(a, 12)
		var ffj, ggj uint32
		if j < 16 {
			ffj = a ^ b ^ c
			ggj = e ^ f ^ g
		} else {
			ffj = (a & b) | (a & c) | (b & c)
			ggj = (e & f) | (^e & g)
		}
		// crypto-api reads a fixed 132-word schedule; an out-of-bounds index is
		// `undefined`, turning the whole (… + undefined) sum into NaN, which its
		// `| 0` coerces to 0. So an OOB W term zeroes the entire tt value.
		var tt1, tt2 uint32
		if j+68 < sm3Words {
			tt1 = ffj + d + ss2 + w[j+68] // #nosec G602 -- guarded by j+68 < sm3Words == len(w)
		}
		if j < sm3Words {
			tt2 = ggj + h + ss1 + w[j] // #nosec G602 -- guarded by j < sm3Words == len(w)
		}
		d, c, b, a = c, sm3rotl(b, 9), a, tt1
		h, g, f, e = g, sm3rotl(f, 19), e, sm3p0(tt2)
	}
	v[0] ^= a
	v[1] ^= b
	v[2] ^= c
	v[3] ^= d
	v[4] ^= e
	v[5] ^= f
	v[6] ^= g
	v[7] ^= h
}

// SM3 computes the SM3 cryptographic hash.
type SM3 struct{}

// Meta returns the operation metadata.
func (SM3) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "SM3",
		Module:      "Crypto",
		Description: "SM3 is a cryptographic hash function used in the Chinese National Standard. SM3 is mainly used in digital signatures, message authentication codes, and pseudorandom number generators. The message digest algorithm consists, by default, of 64 rounds and length of 256.",
		InfoURL:     "https://wikipedia.org/wiki/SM3_(hash_function)",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeString,
	}
}

// sm3MinRounds is the minimum round count crypto-api accepts.
var sm3MinRounds = 16.0

// Args returns the argument definitions.
func (SM3) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Length", Type: core.ArgNumber, Integer: true, Value: sm3DefaultLength},
		{Name: "Rounds", Type: core.ArgNumber, Integer: true, Value: sm3DefaultRounds, Min: &sm3MinRounds},
	}
}

// Run hashes the input with the given output length (bits) and round count.
func (SM3) Run(in *core.Dish, args []any) (*core.Dish, error) {
	length := int(args[0].(float64))
	rounds := int(args[1].(float64))
	return core.NewDish([]byte(hex.EncodeToString(sm3Hash(in.Bytes(), length, rounds))), core.TypeString), nil
}
