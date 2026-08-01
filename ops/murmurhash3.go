package ops

import (
	"math/bits"
	"strconv"
	"unicode/utf16"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(MurmurHash3{})
}

// MurmurHash3 computes the 32-bit MurmurHash v3 (x86_32 variant) of the input,
// with an optional seed. Ported from CyberChef's MurmurHash3 (based on Gary
// Court's murmurhash-js): each input character contributes its low byte, and the
// result is returned as an unsigned (or optionally signed) 32-bit integer.
type MurmurHash3 struct{}

// Meta returns the operation metadata.
func (MurmurHash3) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "MurmurHash3",
		Module:      "Hashing",
		Description: "Generates a MurmurHash v3 for a string input and an optional seed input",
		InfoURL:     "https://wikipedia.org/wiki/MurmurHash",
		InputType:   core.TypeString,
		OutputType:  core.TypeNumber,
	}
}

// Args returns the argument definitions.
func (MurmurHash3) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Seed", Type: core.ArgNumber, Integer: true, Value: 0},
		{Name: "Convert to Signed", Type: core.ArgBoolean, Value: false},
	}
}

// Run computes the hash and formats it as a signed or unsigned integer.
func (MurmurHash3) Run(in *core.Dish, args []any) (*core.Dish, error) {
	// #nosec G115 -- the seed is a 32-bit accumulator; wrapping matches CyberChef
	seed := uint32(int64(args[0].(float64)))
	hash := mmh3(in.String(), seed)

	var out string
	if args[1].(bool) {
		out = strconv.FormatInt(murmurToSigned(hash), 10)
	} else {
		out = strconv.FormatUint(uint64(hash), 10)
	}
	return core.NewDish([]byte(out), core.TypeNumber), nil
}

// mmh3 computes MurmurHash3 x86_32 over the input's UTF-16 code units, each
// reduced to its low byte (matching CyberChef's charCodeAt & 0xff).
func mmh3(input string, seed uint32) uint32 {
	units := utf16.Encode([]rune(input))
	data := make([]byte, len(units))
	for i, u := range units {
		data[i] = byte(u) // #nosec G115 -- low byte of the UTF-16 code unit (charCodeAt & 0xff)
	}

	const c1, c2 = 0xcc9e2d51, 0x1b873593
	h1 := seed
	nblocks := len(data) / 4
	for i := range nblocks {
		k1 := uint32(data[i*4]) | uint32(data[i*4+1])<<8 | uint32(data[i*4+2])<<16 | uint32(data[i*4+3])<<24
		k1 *= c1
		k1 = bits.RotateLeft32(k1, 15)
		k1 *= c2
		h1 ^= k1
		h1 = bits.RotateLeft32(h1, 13)
		h1 = h1*5 + 0xe6546b64
	}

	var k1 uint32
	tail := data[nblocks*4:]
	switch len(data) & 3 {
	case 3:
		k1 ^= uint32(tail[2]) << 16
		fallthrough
	case 2:
		k1 ^= uint32(tail[1]) << 8
		fallthrough
	case 1:
		k1 ^= uint32(tail[0])
		k1 *= c1
		k1 = bits.RotateLeft32(k1, 15)
		k1 *= c2
		h1 ^= k1
	}

	h1 ^= uint32(len(data)) // #nosec G115 -- length is xored into the 32-bit hash; wrapping is intended
	h1 ^= h1 >> 16
	h1 *= 0x85ebca6b
	h1 ^= h1 >> 13
	h1 *= 0xc2b2ae35
	h1 ^= h1 >> 16
	return h1
}

// murmurToSigned reinterprets a 32-bit hash as a signed integer, matching
// CyberChef's unsignedToSigned (value − 2³² when the high bit is set).
func murmurToSigned(h uint32) int64 {
	if h&0x80000000 != 0 {
		return int64(h) - 0x100000000
	}
	return int64(h)
}
