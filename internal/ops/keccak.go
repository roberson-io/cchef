package ops

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"hash"
	"math/bits"

	"golang.org/x/crypto/sha3"

	"github.com/roberson-io/cchef/internal/core"
)

// Domain-separation bytes appended during padding. Keccak (legacy, e.g.
// Ethereum) uses 0x01; SHA-3 (FIPS 202) uses 0x06. This single byte is the only
// difference between the two over an otherwise identical sponge.
const (
	domainKeccak byte = 0x01
	domainSHA3   byte = 0x06
)

func init() {
	core.Register(Keccak{})
}

// keccakRC are the 24 round constants for Keccak-f[1600].
var keccakRC = [24]uint64{
	0x0000000000000001, 0x0000000000008082, 0x800000000000808A, 0x8000000080008000,
	0x000000000000808B, 0x0000000080000001, 0x8000000080008081, 0x8000000000008009,
	0x000000000000008A, 0x0000000000000088, 0x0000000080008009, 0x000000008000000A,
	0x000000008000808B, 0x800000000000008B, 0x8000000000008089, 0x8000000000008003,
	0x8000000000008002, 0x8000000000000080, 0x000000000000800A, 0x800000008000000A,
	0x8000000080008081, 0x8000000000008080, 0x0000000080000001, 0x8000000080008008,
}

// keccakRot are the rotation offsets, and keccakPi the lane permutation indices,
// for the rho and pi steps (compact reference form).
var (
	keccakRot = [24]uint{1, 3, 6, 10, 15, 21, 28, 36, 45, 55, 2, 14, 27, 41, 56, 8, 25, 43, 62, 18, 39, 61, 20, 44}
	keccakPi  = [24]int{10, 7, 11, 17, 18, 3, 5, 16, 8, 21, 24, 4, 15, 23, 19, 13, 12, 2, 20, 14, 22, 9, 6, 1}
)

// keccakF applies the Keccak-f[1600] permutation to the 25-lane state in place.
func keccakF(st *[25]uint64) {
	var bc [5]uint64
	for round := 0; round < 24; round++ {
		// Theta
		for i := 0; i < 5; i++ {
			bc[i] = st[i] ^ st[i+5] ^ st[i+10] ^ st[i+15] ^ st[i+20]
		}
		for i := 0; i < 5; i++ {
			t := bc[(i+4)%5] ^ bits.RotateLeft64(bc[(i+1)%5], 1)
			for j := 0; j < 25; j += 5 {
				st[j+i] ^= t
			}
		}
		// Rho and Pi
		t := st[1]
		for i := 0; i < 24; i++ {
			j := keccakPi[i]
			tmp := st[j]
			st[j] = bits.RotateLeft64(t, int(keccakRot[i]))
			t = tmp
		}
		// Chi
		for j := 0; j < 25; j += 5 {
			for i := 0; i < 5; i++ {
				bc[i] = st[j+i]
			}
			for i := 0; i < 5; i++ {
				st[j+i] ^= (^bc[(i+1)%5]) & bc[(i+2)%5]
			}
		}
		// Iota
		st[0] ^= keccakRC[round]
	}
}

// keccakSum computes a fixed-size Keccak/SHA-3 digest. sizeBits is the output
// size (224/256/384/512); domain selects Keccak (0x01) vs SHA-3 (0x06).
func keccakSum(data []byte, sizeBits int, domain byte) []byte {
	outLen := sizeBits / 8
	rate := 200 - 2*outLen // bytes; a multiple of 8 for all supported sizes
	rateLanes := rate / 8

	var st [25]uint64

	// Absorb full rate-sized blocks.
	for len(data) >= rate {
		for i := 0; i < rateLanes; i++ {
			st[i] ^= binary.LittleEndian.Uint64(data[i*8:])
		}
		keccakF(&st)
		data = data[rate:]
	}

	// Final block: remaining bytes + pad10*1 with the domain byte.
	block := make([]byte, rate)
	copy(block, data)
	block[len(data)] ^= domain
	block[rate-1] ^= 0x80
	for i := 0; i < rateLanes; i++ {
		st[i] ^= binary.LittleEndian.Uint64(block[i*8:])
	}
	keccakF(&st)

	// Squeeze. For all supported sizes outLen <= rate, so one block suffices.
	out := make([]byte, rate)
	for i := 0; i < rateLanes; i++ {
		binary.LittleEndian.PutUint64(out[i*8:], st[i])
	}
	return out[:outLen]
}

// Keccak computes a legacy Keccak digest (as used by e.g. Ethereum), which
// differs from SHA-3 in its padding.
type Keccak struct{}

// Meta returns the operation metadata.
func (Keccak) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Keccak",
		Module:      "Crypto",
		Description: "Computes the legacy Keccak hash digest at the selected size. Keccak predates and differs from the standardised SHA-3 (different padding); Keccak-256 is the hash used by Ethereum.",
		InfoURL:     "https://wikipedia.org/wiki/SHA-3",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (Keccak) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Size", Type: core.ArgOption, Value: []string{"512", "384", "256", "224"}},
	}
}

// Run computes the digest. The widely-used 256- and 512-bit sizes use the
// vetted golang.org/x/crypto/sha3 legacy-Keccak constructors; 224 and 384 (not
// provided by that package) use the local Keccak-f sponge, which is
// cross-validated against the library for 256/512 in the tests.
func (Keccak) Run(in *core.Dish, args []any) (*core.Dish, error) {
	var digest []byte
	switch args[0].(string) {
	case "256":
		digest = legacyKeccak(sha3.NewLegacyKeccak256(), in.Bytes())
	case "512":
		digest = legacyKeccak(sha3.NewLegacyKeccak512(), in.Bytes())
	case "224":
		digest = keccakSum(in.Bytes(), 224, domainKeccak)
	case "384":
		digest = keccakSum(in.Bytes(), 384, domainKeccak)
	default:
		return nil, fmt.Errorf("invalid size %q", args[0])
	}
	return core.NewDish([]byte(hex.EncodeToString(digest)), core.TypeString), nil
}

// legacyKeccak runs data through an x/crypto legacy-Keccak hasher.
func legacyKeccak(h hash.Hash, data []byte) []byte {
	h.Write(data)
	return h.Sum(nil)
}
