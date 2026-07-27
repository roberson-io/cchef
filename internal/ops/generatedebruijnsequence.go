package ops

import (
	"errors"
	"math"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

// The sizes a sequence may be built for. The alphabet is written with the
// digits, so it stops at nine, and a key of one character would not be a
// sequence at all.
const (
	deBruijnMinAlphabet = 2
	deBruijnMaxAlphabet = 9
	deBruijnMinKey      = 2
)

// deBruijnMaxKeys caps how much work is done at once. The sequence is as long
// as the number of keys it covers, which is the alphabet size raised to the key
// length.
const deBruijnMaxKeys = 50000

// The complaints about sizes no sequence will be built for.
//
//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
var (
	errDeBruijnAlphabetRange = errors.New(
		"Invalid alphabet size, required to be between 2 and 9 (inclusive).")
	errDeBruijnAlphabetWhole = errors.New("Invalid alphabet size, required to be integer.")
	errDeBruijnKeyWhole      = errors.New("Invalid key length, required to be integer.")
	errDeBruijnKeyTooShort   = errors.New("Invalid key length, required to be at least 2.")
	errDeBruijnTooManyKeys   = errors.New(
		"Too many permutations, please reduce k^n to under 50,000.")
)

// GenerateDeBruijnSequence builds a De Bruijn sequence.
type GenerateDeBruijnSequence struct{}

// Meta returns the operation metadata.
func (GenerateDeBruijnSequence) Meta() core.OpMeta {
	return core.OpMeta{
		Name:   "Generate De Bruijn Sequence",
		Module: "Default",
		Description: "Generates rolling keycode combinations given a certain " +
			"alphabet size and key length.<br><br>Read as a loop, the sequence " +
			"holds every key of that length over that alphabet exactly once, so it " +
			"is the shortest string that tries them all.",
		InfoURL:    "https://wikipedia.org/wiki/De_Bruijn_sequence",
		InputType:  core.TypeString,
		OutputType: core.TypeString,
	}
}

// Args returns the argument definitions.
func (GenerateDeBruijnSequence) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Alphabet size (k)", Type: core.ArgNumber, Value: float64(deBruijnMinAlphabet)},
		{Name: "Key length (n)", Type: core.ArgNumber, Value: 3.0},
	}
}

// Run builds the sequence. The input plays no part.
func (GenerateDeBruijnSequence) Run(in *core.Dish, args []any) (*core.Dish, error) {
	alphabet, _ := args[0].(float64)
	keyLength, _ := args[1].(float64)

	if err := checkDeBruijnSizes(alphabet, keyLength); err != nil {
		return nil, err
	}
	sequence := deBruijnSequence(int(alphabet), int(keyLength))
	return core.NewDish([]byte(sequence), core.TypeString), nil
}

// checkDeBruijnSizes reports whether a sequence will be built for those sizes.
// The order the complaints come in is CyberChef's: the range of the alphabet is
// looked at before whether it is a whole number.
func checkDeBruijnSizes(alphabet, keyLength float64) error {
	switch {
	case alphabet < deBruijnMinAlphabet || alphabet > deBruijnMaxAlphabet:
		return errDeBruijnAlphabetRange
	case alphabet != math.Trunc(alphabet):
		return errDeBruijnAlphabetWhole
	case keyLength != math.Trunc(keyLength):
		return errDeBruijnKeyWhole
	case keyLength < deBruijnMinKey:
		return errDeBruijnKeyTooShort
	case math.Pow(alphabet, keyLength) > deBruijnMaxKeys:
		return errDeBruijnTooManyKeys
	}
	return nil
}

// deBruijnSequence builds the sequence by the usual method: walk the necklaces
// of the alphabet in order, and take the ones whose length divides the key
// length. Their digits, run together, are the sequence.
func deBruijnSequence(alphabet, keyLength int) string {
	var out strings.Builder
	held := make([]int, alphabet*keyLength)

	// walk carries two positions: how far along the necklace being built it has
	// got, and the length of the run it is repeating.
	var walk func(at, period int)
	walk = func(at, period int) {
		if at > keyLength {
			if keyLength%period == 0 {
				for i := 1; i <= period; i++ {
					// #nosec G115 -- the alphabet stops at nine, so every digit
					// held is one character
					out.WriteByte(byte('0' + held[i]))
				}
			}
			return
		}

		held[at] = held[at-period]
		walk(at+1, period)
		for digit := held[at-period] + 1; digit < alphabet; digit++ {
			held[at] = digit
			walk(at+1, at)
		}
	}
	walk(1, 1)

	return out.String()
}

func init() { core.Register(GenerateDeBruijnSequence{}) }
