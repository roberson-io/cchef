package ops

import (
	"encoding/hex"
	"hash"
	"strconv"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/snefru"
)

func init() {
	core.Register(MD2{})
	core.Register(MD4{})
	core.Register(SHA0{})
	core.Register(HAS160{})
	core.Register(RIPEMD{})
	core.Register(Snefru{})
	core.Register(Whirlpool{})
}

// hashOpMeta builds the common metadata for a Crypto-module hash operation.
func hashOpMeta(name, desc, url string) core.OpMeta {
	return core.OpMeta{
		Name: name, Module: "Crypto", Description: desc, InfoURL: url,
		InputType: core.TypeArrayBuffer, OutputType: core.TypeString,
	}
}

// runHashOp hashes the input bytes and returns the lowercase hex digest.
func runHashOp(newHash func() hash.Hash, in *core.Dish) *core.Dish {
	h := newHash()
	h.Write(in.Bytes())
	return core.NewDish([]byte(hex.EncodeToString(h.Sum(nil))), core.TypeString)
}

var (
	sha0RoundsMin   float64 = 16
	has160RoundsMin float64 = 1
	has160RoundsMax float64 = 80
	md2RoundsMin    float64
	whirlRoundsMin  float64 = 1
	whirlRoundsMax  float64 = 10
	snefruSizeMin   float64 = 32
	snefruSizeMax   float64 = 480
)

// MD2 computes the RFC 1319 MD2 hash.
type MD2 struct{}

// Meta returns the operation metadata.
func (MD2) Meta() core.OpMeta {
	return hashOpMeta("MD2", "The MD2 (Message-Digest 2) algorithm is a cryptographic hash function developed by Ronald Rivest in 1989. The algorithm is optimized for 8-bit computers.<br><br>Although MD2 is no longer considered secure, even as of 2014, it remains in use in public key infrastructures as part of certificates generated with MD2 and RSA.", "https://wikipedia.org/wiki/MD2_(cryptography)")
}

// Args returns the argument definitions.
func (MD2) Args() []core.ArgDef {
	return []core.ArgDef{{Name: "Rounds", Type: core.ArgNumber, Integer: true, Value: float64(18), Min: &md2RoundsMin}}
}

// Run computes the MD2 hash.
func (MD2) Run(in *core.Dish, args []any) (*core.Dish, error) {
	rounds := int(args[0].(float64))
	if rounds == 0 { // crypto-api's `rounds || 18`: a falsy 0 becomes the default.
		rounds = 18
	}
	return runHashOp(func() hash.Hash { return newMD2Rounds(rounds) }, in), nil
}

// MD4 computes the RFC 1320 MD4 hash.
type MD4 struct{}

// Meta returns the operation metadata.
func (MD4) Meta() core.OpMeta {
	return hashOpMeta("MD4", "The MD4 (Message-Digest 4) algorithm is a cryptographic hash function developed by Ronald Rivest in 1990. The digest length is 128 bits. The algorithm has influenced later designs, such as the MD5, SHA-1 and RIPEMD algorithms.<br><br>The security of MD4 has been severely compromised.", "https://wikipedia.org/wiki/MD4")
}

// Args returns the argument definitions.
func (MD4) Args() []core.ArgDef { return nil }

// Run computes the MD4 hash.
func (MD4) Run(in *core.Dish, args []any) (*core.Dish, error) {
	return runHashOp(newMD4, in), nil
}

// SHA0 computes the (withdrawn 1993) SHA-0 hash.
type SHA0 struct{}

// Meta returns the operation metadata.
func (SHA0) Meta() core.OpMeta {
	return hashOpMeta("SHA0", "SHA-0 is a retronym applied to the original version of the 160-bit hash function published in 1993 under the name 'SHA'. It was withdrawn shortly after publication due to an undisclosed 'significant flaw' and replaced by the slightly revised version SHA-1. The message is broken into 512-bit chunks, and the padding of the message adds a 64-bit integer to the end of the final chunk, until it can be evenly divided by 512 bits.", "https://wikipedia.org/wiki/SHA-1#SHA-0")
}

// Args returns the argument definitions.
func (SHA0) Args() []core.ArgDef {
	return []core.ArgDef{{Name: "Rounds", Type: core.ArgNumber, Integer: true, Value: float64(80), Min: &sha0RoundsMin}}
}

// Run computes the SHA-0 hash.
func (SHA0) Run(in *core.Dish, args []any) (*core.Dish, error) {
	rounds := int(args[0].(float64))
	return runHashOp(func() hash.Hash { return newSHA0Rounds(rounds) }, in), nil
}

// HAS160 computes the Korean HAS-160 hash.
type HAS160 struct{}

// Meta returns the operation metadata.
func (HAS160) Meta() core.OpMeta {
	return hashOpMeta("HAS-160", "HAS-160 is a cryptographic hash function designed for use with the Korean KCDSA digital signature algorithm. It is derived from SHA-1, with assorted changes intended to increase its security. It produces a 160-bit output.<br><br>HAS-160 is used in the same way as SHA-1. First it divides input in blocks of 512 bits each and pads the final block. A digest function updates the intermediate hash value by processing the input blocks in turn.", "https://wikipedia.org/wiki/HAS-160")
}

// Args returns the argument definitions.
func (HAS160) Args() []core.ArgDef {
	return []core.ArgDef{{Name: "Rounds", Type: core.ArgNumber, Integer: true, Value: float64(80), Min: &has160RoundsMin, Max: &has160RoundsMax}}
}

// Run computes the HAS-160 hash.
func (HAS160) Run(in *core.Dish, args []any) (*core.Dish, error) {
	rounds := int(args[0].(float64))
	return runHashOp(func() hash.Hash { return newHAS160Rounds(rounds) }, in), nil
}

// RIPEMD computes the RIPEMD hash at one of the four output sizes.
type RIPEMD struct{}

// Meta returns the operation metadata.
func (RIPEMD) Meta() core.OpMeta {
	return hashOpMeta("RIPEMD", "RIPEMD (RACE Integrity Primitives Evaluation Message Digest) is a family of cryptographic hash functions developed in Leuven, Belgium, by Hans Dobbertin, Antoon Bosselaers and Bart Preneel at the COSIC research group at the Katholieke Universiteit Leuven, and first published in 1996.<br><br>RIPEMD was based upon the design principles used in MD4, and is similar in performance to the more popular SHA-1.", "https://wikipedia.org/wiki/RIPEMD")
}

var ripemdSizes = []string{"320", "256", "160", "128"}

// Args returns the argument definitions.
func (RIPEMD) Args() []core.ArgDef {
	return []core.ArgDef{{Name: "Size", Type: core.ArgOption, Value: ripemdSizes}}
}

// Run computes the RIPEMD hash at the selected size.
func (RIPEMD) Run(in *core.Dish, args []any) (*core.Dish, error) {
	var newHash func() hash.Hash
	switch args[0].(string) {
	case "128":
		newHash = newRIPEMD128
	case "160":
		newHash = newRIPEMD160
	case "256":
		newHash = newRIPEMD256
	default: // 320
		newHash = newRIPEMD320
	}
	return runHashOp(newHash, in), nil
}

// Snefru computes the Snefru hash at the given output length and round count.
type Snefru struct{}

// Meta returns the operation metadata.
func (Snefru) Meta() core.OpMeta {
	return hashOpMeta("Snefru", "Snefru is a cryptographic hash function invented by Ralph Merkle in 1990 while working at Xerox PARC. The function supports 128-bit and 256-bit output. It was named after the Egyptian Pharaoh Sneferu, continuing the tradition of the Khufu and Khafre block ciphers.<br><br>The original design of Snefru was shown to be insecure by Eli Biham and Adi Shamir who were able to use differential cryptanalysis to find hash collisions. The design was then modified by increasing the number of iterations of the main pass of the algorithm from two to eight.", "https://wikipedia.org/wiki/Snefru")
}

var snefruRoundsOpts = []string{"8", "4", "2"}

// Args returns the argument definitions.
func (Snefru) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Size", Type: core.ArgNumber, Integer: true, Value: float64(128), Min: &snefruSizeMin, Max: &snefruSizeMax},
		{Name: "Rounds", Type: core.ArgOption, Value: snefruRoundsOpts},
	}
}

// Run computes the Snefru hash.
func (Snefru) Run(in *core.Dish, args []any) (*core.Dish, error) {
	size := int(args[0].(float64))
	rounds, _ := strconv.Atoi(args[1].(string))
	return runHashOp(func() hash.Hash { return snefru.NewWithParams(size, rounds) }, in), nil
}

// Whirlpool computes the Whirlpool hash (or its -0/-T variants).
type Whirlpool struct{}

// Meta returns the operation metadata.
func (Whirlpool) Meta() core.OpMeta {
	return hashOpMeta("Whirlpool", "Whirlpool is a cryptographic hash function designed by Vincent Rijmen (co-creator of the Advanced Encryption Standard) and Paulo S. L. M. Barreto, who first described it in 2000.<br><br>Several variants exist:<ul><li>Whirlpool-0 is the original version released in 2000.</li><li>Whirlpool-T is the first revision, released in 2001, improving the generation of the s-box.</li><li>Whirlpool is the latest revision, released in 2003, fixing a flaw in the diffusion matrix.</li></ul>", "https://wikipedia.org/wiki/Whirlpool_(hash_function)")
}

var whirlpoolVariants = []string{"Whirlpool", "Whirlpool-T", "Whirlpool-0"}

// Args returns the argument definitions.
func (Whirlpool) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Variant", Type: core.ArgOption, Value: whirlpoolVariants},
		{Name: "Rounds", Type: core.ArgNumber, Integer: true, Value: float64(10), Min: &whirlRoundsMin, Max: &whirlRoundsMax},
	}
}

// Run computes the Whirlpool hash.
func (Whirlpool) Run(in *core.Dish, args []any) (*core.Dish, error) {
	variant := args[0].(string)
	rounds := int(args[1].(float64))
	return runHashOp(func() hash.Hash { return newWhirlpoolVariant(variant, rounds) }, in), nil
}
