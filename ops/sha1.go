package ops

import (
	"hash"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(SHA1{})
}

// SHA1 computes the SHA-1 hash over a configurable number of rounds.
type SHA1 struct{}

// Meta returns the operation metadata.
func (SHA1) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "SHA1",
		Module:      "Crypto",
		Description: "The SHA (Secure Hash Algorithm) hash functions were designed by the NSA. SHA-1 is the most established of the existing SHA hash functions and it is used in a variety of security applications and protocols.<br><br>However, SHA-1's collision resistance has been weakening as new attacks are discovered or improved. The message digest algorithm consists, by default, of 80 rounds.",
		InfoURL:     "https://wikipedia.org/wiki/SHA-1",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeString,
	}
}

// sha1RoundsMin is the fewest rounds CyberChef accepts; below it the schedule
// is shorter than the 16 words a block supplies.
var sha1RoundsMin float64 = 16

// sha1RoundsMax caps the count at the standard 80. CyberChef leaves it open,
// but every round past 80 extends the message schedule, so an unbounded count
// is an unbounded allocation and the digest it produces means nothing.
var sha1RoundsMax float64 = shaLegacyRounds

// Args returns the argument definitions.
func (SHA1) Args() []core.ArgDef {
	return []core.ArgDef{
		{
			Name: "Rounds", Type: core.ArgNumber, Integer: true, Value: float64(shaLegacyRounds),
			Min: &sha1RoundsMin, Max: &sha1RoundsMax,
		},
	}
}

// Run computes the digest.
func (SHA1) Run(in *core.Dish, args []any) (*core.Dish, error) {
	rounds := int(args[0].(float64))
	return runHashOp(func() hash.Hash { return newSHA1Rounds(rounds) }, in), nil
}
